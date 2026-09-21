package provider_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	cubitprovider "github.com/ishaf/terraform-provider-cubit/internal/provider"
)

func TestProviderSchema(t *testing.T) {
	t.Run("Given the Cubit Terraform Provider", func(t *testing.T) {
		p := cubitprovider.New("1.0.0")()

		t.Run("When provider metadata is queried", func(t *testing.T) {
			var resp provider.MetadataResponse
			p.Metadata(context.Background(), provider.MetadataRequest{}, &resp)

			t.Run("Then type name is cubit", func(t *testing.T) {
				if resp.TypeName != "cubit" {
					t.Errorf("expected typeName cubit, got %q", resp.TypeName)
				}
			})

			t.Run("Then version is 1.0.0", func(t *testing.T) {
				if resp.Version != "1.0.0" {
					t.Errorf("expected version 1.0.0, got %q", resp.Version)
				}
			})
		})

		t.Run("When provider schema is requested", func(t *testing.T) {
			var resp provider.SchemaResponse
			p.Schema(context.Background(), provider.SchemaRequest{}, &resp)

			t.Run("Then endpoint attribute is present and optional", func(t *testing.T) {
				attr, ok := resp.Schema.Attributes["endpoint"]
				if !ok {
					t.Fatalf("missing endpoint attribute")
				}
				strAttr, ok := attr.(schema.StringAttribute)
				if !ok || !strAttr.Optional {
					t.Errorf("expected endpoint to be optional string attribute")
				}
			})

			t.Run("Then token attribute is present and marked sensitive", func(t *testing.T) {
				attr, ok := resp.Schema.Attributes["token"]
				if !ok {
					t.Fatalf("missing token attribute")
				}
				strAttr, ok := attr.(schema.StringAttribute)
				if !ok || !strAttr.Sensitive {
					t.Errorf("expected token to be sensitive string attribute")
				}
			})
		})
	})
}

func TestClientOperations(t *testing.T) {
	t.Run("Given a Cubit API mock server", func(t *testing.T) {
		mux := http.NewServeMux()

		mux.HandleFunc("/api/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				var req map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&req)
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"id":        "node-test-1",
					"name":      req["name"],
					"ipAddress": req["ipAddress"],
					"status":    "online",
					"cpuCores":  8,
					"memoryMb":  16384,
				})
			}
		})

		mux.HandleFunc("/api/v1/nodes/node-test-1", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"id":        "node-test-1",
					"name":      "edge-sg-1",
					"ipAddress": "192.168.1.50",
					"status":    "online",
					"cpuCores":  8,
					"memoryMb":  16384,
				})
			} else if r.Method == http.MethodDelete {
				w.WriteHeader(http.StatusNoContent)
			}
		})

		mux.HandleFunc("/api/v1/applications", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				var req map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&req)
				w.WriteHeader(http.StatusCreated)
				resp := map[string]interface{}{
					"id":         "app-test-1",
					"name":       req["name"],
					"sourceType": req["sourceType"],
					"gitRepo":    req["gitRepo"],
					"inlineCode": req["inlineCode"],
					"status":     "created",
				}
				if resp["sourceType"] == nil || resp["sourceType"] == "" {
					resp["sourceType"] = "git"
				}
				_ = json.NewEncoder(w).Encode(resp)
			}
		})

		mux.HandleFunc("/api/v1/domains", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost {
				var req map[string]interface{}
				_ = json.NewDecoder(r.Body).Decode(&req)
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"id":            "dom-test-1",
					"hostname":      req["hostname"],
					"applicationId": req["applicationId"],
					"tlsStatus":     "ready",
				})
			} else if r.Method == http.MethodGet {
				_ = json.NewEncoder(w).Encode([]map[string]interface{}{
					{
						"id":            "dom-test-1",
						"hostname":      "api.cubit.local",
						"applicationId": "app-test-1",
						"tlsStatus":     "ready",
					},
				})
			}
		})

		srv := httptest.NewServer(mux)
		defer srv.Close()

		client := cubitprovider.NewCubitClient(srv.URL, "secret-token")

		t.Run("When creating a new server node", func(t *testing.T) {
			node, err := client.CreateNode(context.Background(), &cubitprovider.ClientNode{
				Name:      "edge-sg-1",
				IPAddress: "192.168.1.50",
				CPUCores:  8,
				MemoryMB:  16384,
			})
			if err != nil {
				t.Fatalf("failed to create node: %v", err)
			}

			t.Run("Then node ID is populated", func(t *testing.T) {
				if node.ID != "node-test-1" {
					t.Errorf("expected id node-test-1, got %q", node.ID)
				}
			})

			t.Run("Then node status is online", func(t *testing.T) {
				if node.Status != "online" {
					t.Errorf("expected online status, got %q", node.Status)
				}
			})
		})

		t.Run("When creating a worker application", func(t *testing.T) {
			app, err := client.CreateApplication(context.Background(), &cubitprovider.ClientApplication{
				Name:    "worker-auth",
				GitRepo: "https://github.com/org/worker-auth",
			})
			if err != nil {
				t.Fatalf("failed to create app: %v", err)
			}

			t.Run("Then application ID and status are set", func(t *testing.T) {
				if app.ID != "app-test-1" {
					t.Errorf("expected id app-test-1, got %q", app.ID)
				}
				if app.Status != "created" {
					t.Errorf("expected status created, got %q", app.Status)
				}
			})
		})

		t.Run("When creating an inline worker application", func(t *testing.T) {
			app, err := client.CreateApplication(context.Background(), &cubitprovider.ClientApplication{
				Name:       "hello-worker",
				SourceType: "inline",
				InlineCode: `export default { fetch: () => new Response("hello") };`,
			})
			if err != nil {
				t.Fatalf("failed to create inline app: %v", err)
			}

			t.Run("Then application is created with sourceType inline", func(t *testing.T) {
				if app.ID != "app-test-1" {
					t.Errorf("expected id app-test-1, got %q", app.ID)
				}
				if app.SourceType != "inline" {
					t.Errorf("expected sourceType inline, got %q", app.SourceType)
				}
			})
		})

		t.Run("When binding a domain to an application", func(t *testing.T) {
			dom, err := client.CreateDomain(context.Background(), &cubitprovider.ClientDomain{
				DomainName:    "api.cubit.local",
				ApplicationID: "app-test-1",
			})
			if err != nil {
				t.Fatalf("failed to create domain: %v", err)
			}

			t.Run("Then domain route is created with TLS status ready", func(t *testing.T) {
				if dom.ID != "dom-test-1" {
					t.Errorf("expected id dom-test-1, got %q", dom.ID)
				}
				if dom.TLSStatus != "ready" {
					t.Errorf("expected TLS status ready, got %q", dom.TLSStatus)
				}
			})
		})

		t.Run("When requesting a non-existent resource", func(t *testing.T) {
			_, err := client.GetNode(context.Background(), "unknown-node")

			t.Run("Then a not found error is returned", func(t *testing.T) {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
			})
		})
	})
}
