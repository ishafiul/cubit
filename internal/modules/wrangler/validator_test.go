package wrangler_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ishaf/cubit/internal/modules/wrangler"
)

func TestValidateCompatibility(t *testing.T) {
	t.Run("Given a fully compatible Cloudflare project with KV, D1, DO, and assets", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "compat-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		srcCode := `
export default {
    async fetch(req, env, ctx) {
        const country = req.cf?.country;
        const val = await env.CACHE.get("key");
        const rows = await env.DB.prepare("SELECT * FROM users").all();
        ctx.waitUntil(Promise.resolve());
        return new Response("OK " + country);
    }
};
`
		if err := os.WriteFile(filepath.Join(tmpDir, "index.ts"), []byte(srcCode), 0644); err != nil {
			t.Fatalf("failed to write source: %v", err)
		}

		cfg := &wrangler.WranglerConfig{
			Name: "my-worker",
			KVNamespaces: []wrangler.KVNamespaceBinding{
				{Binding: "CACHE", ID: "kv-123"},
			},
			D1Databases: []wrangler.D1DatabaseBinding{
				{Binding: "DB", DatabaseID: "d1-123"},
			},
			DurableObjects: &wrangler.DurableObjectsConfig{
				Bindings: []wrangler.DurableObjectBinding{
					{Name: "CHAT", ClassName: "ChatRoom"},
				},
			},
			Assets: &wrangler.AssetsConfig{
				Directory: "./public",
				Binding:   "ASSETS",
			},
		}

		t.Run("When validating compatibility", func(t *testing.T) {
			report, err := wrangler.ValidateCompatibility(cfg, tmpDir)

			t.Run("Then report indicates compatible and can deploy", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				if report.Level != wrangler.LevelCompatible {
					t.Errorf("expected level compatible, got %s", report.Level)
				}
				if !report.CanDeploy {
					t.Errorf("expected CanDeploy true")
				}
				if report.UnsupportedCount != 0 {
					t.Errorf("expected 0 unsupported features, got %d", report.UnsupportedCount)
				}
			})

			t.Run("Then supported features are recognized", func(t *testing.T) {
				foundKV := false
				foundD1 := false
				foundDO := false
				foundAssets := false
				foundCF := false

				for _, feat := range report.SupportedFeatures {
					switch feat.Name {
					case "KV Namespace (CACHE)":
						foundKV = true
					case "D1 Database (DB)":
						foundD1 = true
					case "Durable Object (CHAT)":
						foundDO = true
					case "Static Assets":
						foundAssets = true
					case "request.cf Edge Context":
						foundCF = true
					}
				}

				if !foundKV {
					t.Errorf("expected KV Namespace in supported features")
				}
				if !foundD1 {
					t.Errorf("expected D1 Database in supported features")
				}
				if !foundDO {
					t.Errorf("expected Durable Object in supported features")
				}
				if !foundAssets {
					t.Errorf("expected Static Assets in supported features")
				}
				if !foundCF {
					t.Errorf("expected request.cf in supported features")
				}
			})
		})
	})

	t.Run("Given a project using unsupported AI and Vectorize APIs", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "compat-unsupported-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		srcCode := `
import { Ai } from "@cloudflare/ai";

export default {
    async fetch(req, env) {
        const response = await env.AI.run("@cf/meta/llama-3-8b-instruct", { prompt: "hi" });
        const vectors = await env.VECTORIZE.query([0.1, 0.2]);
        return new Response(JSON.stringify(response));
    }
};
`
		if err := os.WriteFile(filepath.Join(tmpDir, "worker.js"), []byte(srcCode), 0644); err != nil {
			t.Fatalf("failed to write worker.js: %v", err)
		}

		cfg := &wrangler.WranglerConfig{
			Name: "ai-worker",
		}

		t.Run("When validating compatibility", func(t *testing.T) {
			report, err := wrangler.ValidateCompatibility(cfg, tmpDir)

			t.Run("Then report indicates incompatible and cannot deploy", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				if report.Level != wrangler.LevelIncompatible {
					t.Errorf("expected level incompatible, got %s", report.Level)
				}
				if report.CanDeploy {
					t.Errorf("expected CanDeploy false for unsupported APIs")
				}
				if report.UnsupportedCount < 2 {
					t.Errorf("expected at least 2 unsupported features, got %d", report.UnsupportedCount)
				}
			})

			t.Run("Then remediation steps are provided for AI and Vectorize", func(t *testing.T) {
				foundAIRemediation := false
				foundVectorizeRemediation := false

				for _, feat := range report.UnsupportedFeatures {
					if feat.Name == "Cloudflare Workers AI (env.AI)" && feat.Remediation != "" {
						foundAIRemediation = true
					}
					if feat.Name == "Cloudflare Vectorize (env.VECTORIZE)" && feat.Remediation != "" {
						foundVectorizeRemediation = true
					}
				}

				if !foundAIRemediation {
					t.Errorf("missing remediation advice for AI")
				}
				if !foundVectorizeRemediation {
					t.Errorf("missing remediation advice for Vectorize")
				}
			})
		})
	})

	t.Run("Given a project using Hyperdrive and Puppeteer", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "compat-puppeteer-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		srcCode := `
import puppeteer from "@cloudflare/puppeteer";

export default {
    async fetch(req, env) {
        const browser = await puppeteer.launch(env.BROWSER);
        const client = await env.HYPERDRIVE.connect();
        return new Response("done");
    }
};
`
		if err := os.WriteFile(filepath.Join(tmpDir, "index.mjs"), []byte(srcCode), 0644); err != nil {
			t.Fatalf("failed to write index.mjs: %v", err)
		}

		cfg := &wrangler.WranglerConfig{
			Name: "browser-worker",
		}

		t.Run("When validating compatibility", func(t *testing.T) {
			report, err := wrangler.ValidateCompatibility(cfg, tmpDir)

			t.Run("Then detects both puppeteer and hyperdrive with remediation", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				var foundPuppeteer, foundHyperdrive bool
				for _, feat := range report.UnsupportedFeatures {
					if feat.Name == "Cloudflare Browser Rendering (@cloudflare/puppeteer)" {
						foundPuppeteer = true
					}
					if feat.Name == "Cloudflare Hyperdrive (env.HYPERDRIVE)" {
						foundHyperdrive = true
					}
				}
				if !foundPuppeteer {
					t.Errorf("expected Puppeteer detected")
				}
				if !foundHyperdrive {
					t.Errorf("expected Hyperdrive detected")
				}
			})
		})
	})

	t.Run("Given a project directory scanned via ScanProject", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "scan-project-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		wranglerContent := `{
  "name": "scanned-worker",
  "compatibility_date": "2024-09-23",
  "d1_databases": [
    { "binding": "DB", "database_id": "db-abc" }
  ]
}`
		if err := os.WriteFile(filepath.Join(tmpDir, "wrangler.json"), []byte(wranglerContent), 0644); err != nil {
			t.Fatalf("failed to write wrangler.json: %v", err)
		}

		workerCode := `export default { fetch(req, env) { return new Response("hello"); } };`
		if err := os.WriteFile(filepath.Join(tmpDir, "index.js"), []byte(workerCode), 0644); err != nil {
			t.Fatalf("failed to write index.js: %v", err)
		}

		t.Run("When scanning project directory", func(t *testing.T) {
			report, err := wrangler.ScanProject(tmpDir)
			if err != nil {
				t.Fatalf("unexpected error scanning project: %v", err)
			}
			if report.Level != wrangler.LevelCompatible {
				t.Errorf("expected level compatible, got %s", report.Level)
			}
			if !report.CanDeploy {
				t.Errorf("expected CanDeploy true")
			}
		})
	})
}
