package deployment_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"testing"

	"github.com/ishaf/cubit/internal/modules/deployment"
)

func runWrappedWorkerInNode(t *testing.T, wrappedScript string, headers map[string]string) (string, map[string]interface{}) {
	t.Helper()

	headersJSON, err := json.Marshal(headers)
	if err != nil {
		t.Fatalf("failed to marshal headers: %v", err)
	}

	testRunner := fmt.Sprintf(`
const rawModule = %q;
const headers = %s;

const req = new Request("http://localhost/test", { headers });
const encoded = Buffer.from(rawModule).toString("base64");
const mod = await import("data:text/javascript;base64," + encoded);
const worker = mod.default || mod;

let resp;
if (typeof worker.fetch === "function") {
    resp = await worker.fetch(req, {}, {});
} else if (typeof worker === "function") {
    resp = await worker(req, {}, {});
} else {
    throw new Error("Worker is not a function or object with fetch");
}

const bodyText = await resp.text();
process.stdout.write(JSON.stringify({
    body: bodyText,
    cf: req.cf || null
}));
`, wrappedScript, string(headersJSON))

	cmd := exec.CommandContext(context.Background(), "node", "--input-type=module", "-e", testRunner)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		t.Fatalf("node execution failed: %v (stderr: %s)", err, stderr.String())
	}

	var output struct {
		Body string                 `json:"body"`
		CF   map[string]interface{} `json:"cf"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &output); err != nil {
		t.Fatalf("failed to unmarshal node output %q: %v", stdout.String(), err)
	}

	return output.Body, output.CF
}

func TestWrapBundleWithEdgeContext(t *testing.T) {
	t.Run("Given a standard export default worker", func(t *testing.T) {
		bundle := []byte(`export default {
    async fetch(req, env, ctx) {
        return new Response("Hello " + req.cf.country + " from " + req.cf.city + " IP:" + req.cf.clientIP);
    }
};`)

		t.Run("When wrapping with edge context", func(t *testing.T) {
			wrapped := deployment.WrapBundleWithEdgeContext(bundle)

			t.Run("Then wrapper code is attached", func(t *testing.T) {
				if !strings.Contains(string(wrapped), "__cubit_synthesize_cf") {
					t.Errorf("expected wrapped bundle to contain __cubit_synthesize_cf")
				}
			})

			t.Run("Then node execution synthesizes request.cf from incoming headers", func(t *testing.T) {
				headers := map[string]string{
					"cf-connecting-ip": "198.51.100.22",
					"cf-ipcountry":     "JP",
					"cf-ipcity":        "Tokyo",
					"cf-colo":          "NRT",
					"cf-ray":           "testray12345-NRT",
				}
				body, cf := runWrappedWorkerInNode(t, string(wrapped), headers)

				expectedBody := "Hello JP from Tokyo IP:198.51.100.22"
				if body != expectedBody {
					t.Errorf("expected body %q, got %q", expectedBody, body)
				}

				if cf == nil {
					t.Fatal("expected req.cf to be populated")
				}
				if cf["country"] != "JP" {
					t.Errorf("expected country JP, got %v", cf["country"])
				}
				if cf["city"] != "Tokyo" {
					t.Errorf("expected city Tokyo, got %v", cf["city"])
				}
				if cf["clientIP"] != "198.51.100.22" {
					t.Errorf("expected clientIP 198.51.100.22, got %v", cf["clientIP"])
				}
				if cf["colo"] != "NRT" {
					t.Errorf("expected colo NRT, got %v", cf["colo"])
				}
				if cf["rayID"] != "testray12345-NRT" {
					t.Errorf("expected rayID testray12345-NRT, got %v", cf["rayID"])
				}
			})
		})
	})

	t.Run("Given an esbuild style named export", func(t *testing.T) {
		bundle := []byte(`
var worker_default = {
    async fetch(req) {
        return new Response("City is " + req.cf.city);
    }
};
export {
    worker_default as default
};`)

		t.Run("When wrapping and executing", func(t *testing.T) {
			wrapped := deployment.WrapBundleWithEdgeContext(bundle)
			headers := map[string]string{
				"cf-ipcountry": "GB",
				"cf-ipcity":    "London",
			}
			body, cf := runWrappedWorkerInNode(t, string(wrapped), headers)

			if body != "City is London" {
				t.Errorf("expected 'City is London', got %q", body)
			}
			if cf["isEUCountry"] != "1" {
				t.Errorf("expected isEUCountry '1' for GB, got %v", cf["isEUCountry"])
			}
			if cf["continent"] != "EU" {
				t.Errorf("expected continent 'EU' for GB, got %v", cf["continent"])
			}
		})
	})

	t.Run("Given a function default export worker", func(t *testing.T) {
		bundle := []byte(`export default function(req, env, ctx) {
    return new Response("Function IP:" + req.cf.clientIP);
};`)

		t.Run("When wrapping and executing", func(t *testing.T) {
			wrapped := deployment.WrapBundleWithEdgeContext(bundle)
			headers := map[string]string{
				"cf-connecting-ip": "203.0.113.50",
			}
			body, cf := runWrappedWorkerInNode(t, string(wrapped), headers)

			if body != "Function IP:203.0.113.50" {
				t.Errorf("expected 'Function IP:203.0.113.50', got %q", body)
			}
			if cf["clientIP"] != "203.0.113.50" {
				t.Errorf("expected clientIP 203.0.113.50, got %v", cf["clientIP"])
			}
		})
	})

	t.Run("Given an already wrapped bundle", func(t *testing.T) {
		bundle := []byte(`// Some code with __cubit_synthesize_cf already present
function __cubit_synthesize_cf(req) {}
export default {};`)

		t.Run("When wrapping again Then returns identical slice without duplicating", func(t *testing.T) {
			wrapped := deployment.WrapBundleWithEdgeContext(bundle)
			if !bytes.Equal(wrapped, bundle) {
				t.Errorf("expected idempotent wrap, bytes changed")
			}
		})
	})

	t.Run("Given comprehensive Cloudflare headers", func(t *testing.T) {
		bundle := []byte(`export default {
    async fetch(req) {
        return new Response(JSON.stringify(req.cf));
    }
};`)

		wrapped := deployment.WrapBundleWithEdgeContext(bundle)
		headers := map[string]string{
			"cf-connecting-ip":   "198.51.100.99",
			"cf-ipcountry":       "DE",
			"cf-ipcity":          "Frankfurt",
			"cf-colo":            "FRA",
			"cf-ray":             "abcdef1234567890-FRA",
			"cf-asn":             "12345",
			"cf-asorganization": "Sample Telecom",
			"cf-iplatitude":      "50.1109",
			"cf-iplongitude":     "8.6821",
			"cf-postalcode":      "60311",
			"cf-metrocode":       "100",
			"cf-region":          "Hesse",
			"cf-regioncode":      "HE",
			"cf-timezone":        "Europe/Berlin",
			"cf-http-protocol":   "HTTP/3",
			"cf-tls-version":     "TLSv1.3",
			"cf-tls-cipher":      "TLS_AES_128_GCM_SHA256",
		}

		body, cf := runWrappedWorkerInNode(t, string(wrapped), headers)
		if len(body) == 0 || cf == nil {
			t.Fatalf("expected response body and cf data")
		}

		if cf["country"] != "DE" {
			t.Errorf("expected country DE, got %v", cf["country"])
		}
		if cf["city"] != "Frankfurt" {
			t.Errorf("expected city Frankfurt, got %v", cf["city"])
		}
		if cf["colo"] != "FRA" {
			t.Errorf("expected colo FRA, got %v", cf["colo"])
		}
		if cf["rayID"] != "abcdef1234567890-FRA" {
			t.Errorf("expected rayID abcdef1234567890-FRA, got %v", cf["rayID"])
		}
		if cf["asn"] != float64(12345) {
			t.Errorf("expected asn 12345, got %v", cf["asn"])
		}
		if cf["asOrganization"] != "Sample Telecom" {
			t.Errorf("expected asOrganization Sample Telecom, got %v", cf["asOrganization"])
		}
		if cf["latitude"] != "50.1109" {
			t.Errorf("expected latitude 50.1109, got %v", cf["latitude"])
		}
		if cf["longitude"] != "8.6821" {
			t.Errorf("expected longitude 8.6821, got %v", cf["longitude"])
		}
		if cf["postalCode"] != "60311" {
			t.Errorf("expected postalCode 60311, got %v", cf["postalCode"])
		}
		if cf["region"] != "Hesse" {
			t.Errorf("expected region Hesse, got %v", cf["region"])
		}
		if cf["regionCode"] != "HE" {
			t.Errorf("expected regionCode HE, got %v", cf["regionCode"])
		}
		if cf["timezone"] != "Europe/Berlin" {
			t.Errorf("expected timezone Europe/Berlin, got %v", cf["timezone"])
		}
		if cf["httpProtocol"] != "HTTP/3" {
			t.Errorf("expected httpProtocol HTTP/3, got %v", cf["httpProtocol"])
		}
		if cf["isEUCountry"] != "1" {
			t.Errorf("expected isEUCountry 1, got %v", cf["isEUCountry"])
		}
		if cf["continent"] != "EU" {
			t.Errorf("expected continent EU, got %v", cf["continent"])
		}
	})
}
