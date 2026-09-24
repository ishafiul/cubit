package application_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/modules/application"
)

type mockAppRepo struct {
	apps map[string]*domain.Application
}

func newMockAppRepo() *mockAppRepo {
	return &mockAppRepo{apps: make(map[string]*domain.Application)}
}

func (m *mockAppRepo) Save(ctx context.Context, app *domain.Application) error {
	m.apps[app.ID] = app
	return nil
}

func (m *mockAppRepo) Update(ctx context.Context, app *domain.Application) error {
	if _, ok := m.apps[app.ID]; !ok {
		return domain.NewNotFoundError("application not found: " + app.ID)
	}
	m.apps[app.ID] = app
	return nil
}

func (m *mockAppRepo) GetByID(ctx context.Context, id string) (*domain.Application, error) {
	if app, ok := m.apps[id]; ok {
		return app, nil
	}
	return nil, domain.NewNotFoundError("application not found: " + id)
}

func (m *mockAppRepo) GetBySubdomain(ctx context.Context, subdomain string) (*domain.Application, error) {
	for _, a := range m.apps {
		if a.Subdomain == subdomain || a.Name == subdomain {
			return a, nil
		}
	}
	return nil, domain.NewNotFoundError("application not found: " + subdomain)
}

func (m *mockAppRepo) List(ctx context.Context) ([]*domain.Application, error) {
	var list []*domain.Application
	for _, a := range m.apps {
		list = append(list, a)
	}
	return list, nil
}

func (m *mockAppRepo) Delete(ctx context.Context, id string) error {
	if _, ok := m.apps[id]; !ok {
		return domain.NewNotFoundError("application not found: " + id)
	}
	delete(m.apps, id)
	return nil
}

func (m *mockAppRepo) ListByGitRepoAndBranch(ctx context.Context, repo, branch string) ([]*domain.Application, error) {
	var list []*domain.Application
	for _, a := range m.apps {
		if a.GitRepo == repo && a.Branch == branch {
			list = append(list, a)
		}
	}
	return list, nil
}

func TestApplicationService(t *testing.T) {
	t.Run("Given a fresh ApplicationService", func(t *testing.T) {
		repo := newMockAppRepo()
		svc := application.NewService(repo, nil, nil, "cubit-fleet")

		t.Run("When creating an application with valid details", func(t *testing.T) {
			app, err := svc.Create(
				context.Background(),
				"test-worker",
				domain.SourceTypeGit,
				"https://github.com/example/worker",
				"main",
				"",
				"",
				true,
				nil,
				nil,
			)

			t.Run("Then creation succeeds without error", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if app == nil {
					t.Fatal("expected created application, got nil")
				}
				if app.Name != "test-worker" {
					t.Fatalf("expected name test-worker, got %s", app.Name)
				}
				if !app.AutoDeploy {
					t.Fatal("expected autoDeploy to be true")
				}
			})
		})

		t.Run("When updating application configuration and runtime settings", func(t *testing.T) {
			app, _ := svc.Create(
				context.Background(),
				"update-worker",
				domain.SourceTypeGit,
				"https://github.com/example/worker",
				"main",
				"",
				"",
				true,
				nil,
				nil,
			)

			autoDeployFalse := false
			cDate := "2024-09-23"
			flags := []string{"nodejs_compat"}
			mem := 256
			dur := 100
			updated, err := svc.Update(context.Background(), app.ID, "staging", "", "", &autoDeployFalse, nil, nil, &cDate, &flags, &mem, &dur)

			t.Run("Then configuration and runtime settings are persisted", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if updated.Branch != "staging" {
					t.Fatalf("expected branch staging, got %s", updated.Branch)
				}
				if updated.AutoDeploy != false {
					t.Fatal("expected autoDeploy to be false")
				}
				if updated.CompatibilityDate != "2024-09-23" {
					t.Fatalf("expected compat date 2024-09-23, got %s", updated.CompatibilityDate)
				}
				if len(updated.CompatibilityFlags) != 1 || updated.CompatibilityFlags[0] != "nodejs_compat" {
					t.Fatalf("expected nodejs_compat flag, got %v", updated.CompatibilityFlags)
				}
				if updated.MemoryLimitMB != 256 {
					t.Fatalf("expected memory limit 256, got %d", updated.MemoryLimitMB)
				}
				if updated.MaxDurationMs != 100 {
					t.Fatalf("expected max duration 100, got %d", updated.MaxDurationMs)
				}
			})
		})

		t.Run("When tracking request execution metrics", func(t *testing.T) {
			app, _ := svc.Create(
				context.Background(),
				"metrics-worker",
				domain.SourceTypeInline,
				"",
				"main",
				"",
				"export default {}",
				true,
				nil,
				nil,
			)

			svc.RecordExecution(app.ID, "GET", "/api", 200, 10.0, "127.0.0.1", "ok")
			svc.RecordExecution(app.ID, "POST", "/submit", 500, 20.0, "127.0.0.1", "error")

			metrics, err := svc.GetMetrics(context.Background(), app.ID)

			t.Run("Then aggregated request metrics are calculated accurately", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if metrics.TotalRequests != 2 {
					t.Fatalf("expected 2 total requests, got %d", metrics.TotalRequests)
				}
				if metrics.Status2xx != 1 {
					t.Fatalf("expected 1 2xx status, got %d", metrics.Status2xx)
				}
				if metrics.Status5xx != 1 {
					t.Fatalf("expected 1 5xx status, got %d", metrics.Status5xx)
				}
				if metrics.AvgDurationMs != 15.0 {
					t.Fatalf("expected avg duration 15.0, got %f", metrics.AvgDurationMs)
				}
			})
		})

		t.Run("When subscribing to live request logs", func(t *testing.T) {
			app, _ := svc.Create(
				context.Background(),
				"logs-worker",
				domain.SourceTypeInline,
				"",
				"main",
				"",
				"export default {}",
				true,
				nil,
				nil,
			)

			logsChan, unsubscribe := svc.SubscribeLiveLogs(app.ID)
			defer unsubscribe()

			svc.RecordExecutionEvent(app.ID, domain.RequestLogEvent{
				ID:              "ray_test123",
				Timestamp:       time.Now().UTC(),
				Method:          "POST",
				Path:            "/api/test",
				URL:             "http://logs-worker.localhost:8000/api/test",
				StatusCode:      200,
				DurationMs:      8.5,
				ClientIP:        "192.168.1.50",
				Message:         "Worker executed",
				Outcome:         "ok",
				RequestHeaders:  map[string]string{"User-Agent": "CubitClient/1.0", "Host": "logs-worker.localhost:8000"},
				RequestBody:     `{"hello":"world"}`,
				ResponseHeaders: map[string]string{"Content-Type": "application/json"},
				ResponseBody:    `{"status":"ok"}`,
				Logs: []domain.ConsoleLogEntry{
					{Level: "log", Message: "handled request", Timestamp: time.Now().UnixMilli()},
				},
			})

			t.Run("Then subscriber receives rich live log event", func(t *testing.T) {
				select {
				case event := <-logsChan:
					if event.ID != "ray_test123" {
						t.Errorf("expected ray_test123, got %s", event.ID)
					}
					if event.Method != "POST" || event.Path != "/api/test" || event.StatusCode != 200 {
						t.Fatalf("unexpected event: %+v", event)
					}
					if event.RequestHeaders["User-Agent"] != "CubitClient/1.0" {
						t.Errorf("expected User-Agent CubitClient/1.0, got %s", event.RequestHeaders["User-Agent"])
					}
					if event.ResponseHeaders["Content-Type"] != "application/json" {
						t.Errorf("expected Content-Type application/json, got %s", event.ResponseHeaders["Content-Type"])
					}
					if len(event.Logs) != 1 || event.Logs[0].Message != "handled request" {
						t.Errorf("expected 1 console log, got %+v", event.Logs)
					}
				default:
					t.Fatal("expected to receive live log event, but channel was empty")
				}
			})
		})

		t.Run("When getting the bundle for an inline application", func(t *testing.T) {
			inlineCode := "export default { fetch: () => new Response('bundle test') };"
			app, _ := svc.Create(
				context.Background(),
				"bundle-worker",
				domain.SourceTypeInline,
				"",
				"",
				"",
				inlineCode,
				true,
				nil,
				nil,
			)
			_ = svc.SetActiveDeployment(context.Background(), app.ID, "dep-123")

			bundle, err := svc.GetBundle(context.Background(), app.ID, "")

			t.Run("Then bundle content matches inline code", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if string(bundle) != inlineCode {
					t.Fatalf("expected %q, got %q", inlineCode, string(bundle))
				}
			})
		})

		t.Run("When invoking a worker bundle with injected environment variables", func(t *testing.T) {
			workerScript := []byte(`
export default {
    async fetch(req, env) {
        return new Response(JSON.stringify({ key: env.SECRET_KEY, mode: env.APP_MODE }), {
            headers: { "Content-Type": "application/json" }
        });
    }
};
`)
			res, err := application.RunWorkerBundleWithEnv(
				context.Background(),
				workerScript,
				"GET",
				"/test",
				nil,
				nil,
				map[string]string{
					"SECRET_KEY": "supersecret123",
					"APP_MODE":   "production",
				},
			)

			t.Run("Then isolate receives and returns env variables", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if res.Status != 200 {
					t.Fatalf("expected status 200, got %d", res.Status)
				}
				body := string(res.Body)
				if !strings.Contains(body, "supersecret123") {
					t.Errorf("expected body to contain supersecret123, got %s", body)
				}
				if !strings.Contains(body, "production") {
					t.Errorf("expected body to contain production, got %s", body)
				}
			})
		})

		t.Run("When importing wrangler configuration with existing secrets", func(t *testing.T) {
			app, _ := svc.Create(
				context.Background(),
				"wrangler-app",
				domain.SourceTypeInline,
				"",
				"main",
				"",
				"export default {}",
				false,
				[]domain.EnvironmentVariable{
					{Key: "DB_PASSWORD", Value: "existing-secret-pass", IsSecret: true},
					{Key: "DEBUG", Value: "true", IsSecret: false},
				},
				nil,
			)

			wranglerJSON := `{
				"compatibility_date": "2024-10-01",
				"compatibility_flags": ["nodejs_compat"],
				"vars": {
					"DEBUG": "false",
					"DB_PASSWORD": "dummy-wrangler-value",
					"NEW_CONFIG_KEY": "hello-world"
				},
				"d1_databases": [
					{ "binding": "DB", "database_name": "app-d1" }
				]
			}`

			updatedApp, summary, err := svc.ImportWrangler(context.Background(), app.ID, wranglerJSON, "auto", "")

			t.Run("Then configuration is imported and secrets are preserved", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error importing wrangler config, got: %v", err)
				}
				if summary.ImportedVarsCount != 3 {
					t.Errorf("expected 3 imported vars, got %d", summary.ImportedVarsCount)
				}
				if summary.PreservedSecretsCount != 1 {
					t.Errorf("expected 1 preserved secret, got %d", summary.PreservedSecretsCount)
				}
				if updatedApp.CompatibilityDate != "2024-10-01" {
					t.Errorf("expected compat date 2024-10-01, got %s", updatedApp.CompatibilityDate)
				}

				vMap := make(map[string]domain.EnvironmentVariable)
				for _, v := range updatedApp.EnvVars {
					vMap[v.Key] = v
				}

				if !vMap["DB_PASSWORD"].IsSecret {
					t.Errorf("DB_PASSWORD must remain secret, got: %+v", vMap["DB_PASSWORD"])
				}
				if vMap["DEBUG"].Value != "false" {
					t.Errorf("DEBUG should be false, got: %s", vMap["DEBUG"].Value)
				}
				if vMap["NEW_CONFIG_KEY"].Value != "hello-world" {
					t.Errorf("NEW_CONFIG_KEY should be hello-world, got: %s", vMap["NEW_CONFIG_KEY"].Value)
				}
				if len(updatedApp.Bindings) != 1 || updatedApp.Bindings[0].Name != "DB" {
					t.Errorf("expected DB binding, got: %v", updatedApp.Bindings)
				}
			})

			t.Run("When importing wrangler config with routes and domain registrar configured", func(t *testing.T) {
				reg := &mockAppDomainRegistrar{}
				svc.SetDomainRegistrar(reg)

				wranglerWithRoutes := `{
					"name": "worker-with-routes",
					"routes": [
						"api.example.com/*",
						"*sub.domain.com/path*"
					]
				}`

				_, summary, err := svc.ImportWrangler(context.Background(), app.ID, wranglerWithRoutes, "json", "")

				t.Run("Then extracted routes are registered and summarized", func(t *testing.T) {
					if err != nil {
						t.Fatalf("expected no error importing wrangler config, got: %v", err)
					}
					if summary.ImportedRoutesCount != 3 {
						t.Errorf("expected 3 registered domains, got %d", summary.ImportedRoutesCount)
					}
					if len(reg.registeredRoutes) != 2 {
						t.Errorf("expected 2 extracted route patterns passed to registrar, got %d", len(reg.registeredRoutes))
					}
				})
			})

			t.Run("When importing wrangler config with Durable Objects and migrations", func(t *testing.T) {
				wranglerWithDO := `{
					"name": "worker-with-do",
					"durable_objects": {
						"bindings": [
							{ "name": "STATE_DO", "class_name": "AppStateDO" }
						]
					},
					"migrations": [
						{ "tag": "v1", "new_classes": ["AppStateDO"] }
					]
				}`

				_, summary, err := svc.ImportWrangler(context.Background(), app.ID, wranglerWithDO, "json", "")

				t.Run("Then Durable Objects and migrations are persisted to SQLite repository", func(t *testing.T) {
					if err != nil {
						t.Fatalf("expected no error importing wrangler config, got: %v", err)
					}
					if summary.ImportedDurableObjectsCount != 1 {
						t.Errorf("expected 1 imported DO, got %d", summary.ImportedDurableObjectsCount)
					}
					if summary.ImportedMigrationsCount != 1 {
						t.Errorf("expected 1 imported migration, got %d", summary.ImportedMigrationsCount)
					}

					refetched, err := svc.GetByID(context.Background(), app.ID)
					if err != nil {
						t.Fatalf("expected no error getting app by id: %v", err)
					}

					var foundDO bool
					for _, b := range refetched.Bindings {
						if b.Type == domain.BindingTypeDurableObject && b.Name == "STATE_DO" && b.ClassName == "AppStateDO" {
							foundDO = true
							break
						}
					}
					if !foundDO {
						t.Errorf("expected STATE_DO in app bindings, got: %+v", refetched.Bindings)
					}

					if len(refetched.Migrations) != 1 || refetched.Migrations[0].Tag != "v1" || len(refetched.Migrations[0].NewClasses) != 1 || refetched.Migrations[0].NewClasses[0] != "AppStateDO" {
						t.Errorf("expected v1 migration persisted in repo, got: %+v", refetched.Migrations)
					}
				})
			})

			t.Run("When importing wrangler config with workflows and containers", func(t *testing.T) {
				wranglerWithWfCt := `{
					"name": "worker-with-wf-ct",
					"workflows": [
						{ "name": "order-pipeline", "binding": "ORDER_WF", "class_name": "OrderWorkflow" }
					],
					"containers": [
						{ "name": "cache-sidecar", "image": "redis:7-alpine", "port": 6379 }
					]
				}`

				_, summary, err := svc.ImportWrangler(context.Background(), app.ID, wranglerWithWfCt, "json", "")

				t.Run("Then workflows and containers are persisted to SQLite repository", func(t *testing.T) {
					if err != nil {
						t.Fatalf("expected no error importing wrangler config, got: %v", err)
					}
					if summary.ImportedWorkflowsCount != 1 {
						t.Errorf("expected 1 imported workflow, got %d", summary.ImportedWorkflowsCount)
					}
					if summary.ImportedContainersCount != 1 {
						t.Errorf("expected 1 imported container, got %d", summary.ImportedContainersCount)
					}

					refetched, err := svc.GetByID(context.Background(), app.ID)
					if err != nil {
						t.Fatalf("expected no error getting app by id: %v", err)
					}

					var foundWF, foundCT bool
					for _, b := range refetched.Bindings {
						if b.Type == domain.BindingTypeWorkflow && b.Name == "ORDER_WF" && b.ClassName == "OrderWorkflow" && b.WorkflowName == "order-pipeline" {
							foundWF = true
						}
						if b.Type == domain.BindingTypeContainer && b.Name == "cache-sidecar" && b.Image == "redis:7-alpine" && b.Port == 6379 {
							foundCT = true
						}
					}
					if !foundWF {
						t.Errorf("expected ORDER_WF in app bindings, got: %+v", refetched.Bindings)
					}
					if !foundCT {
						t.Errorf("expected cache-sidecar in app bindings, got: %+v", refetched.Bindings)
					}
				})
			})
		})

		t.Run("When accessing Cloudflare edge context request.cf", func(t *testing.T) {
			workerScript := []byte(`
export default {
    async fetch(req, env) {
        return new Response(JSON.stringify({
            country: req.cf?.country,
            city: req.cf?.city,
            asn: req.cf?.asn,
            connectingIp: req.headers.get("cf-connecting-ip")
        }), { headers: { "Content-Type": "application/json" } });
    }
};
`)
			headers := map[string]string{
				"CF-IPCountry":     "GB",
				"CF-Connecting-IP": "82.165.197.1",
			}
			res, err := application.RunWorkerBundleWithEnvAndBindings(
				context.Background(),
				workerScript,
				"GET",
				"/cf-test",
				headers,
				nil,
				nil,
				nil,
				"http://localhost:8000",
				"fetch",
			)

			t.Run("Then isolate populates request.cf and Cloudflare headers", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				var cfData struct {
					Country      string `json:"country"`
					City         string `json:"city"`
					ASN          int    `json:"asn"`
					ConnectingIP string `json:"connectingIp"`
				}
				if err := json.Unmarshal(res.Body, &cfData); err != nil {
					t.Fatalf("failed unmarshaling body %s: %v", string(res.Body), err)
				}
				if cfData.Country != "GB" {
					t.Errorf("expected country GB, got %s", cfData.Country)
				}
				if cfData.ConnectingIP != "82.165.197.1" {
					t.Errorf("expected connecting IP 82.165.197.1, got %s", cfData.ConnectingIP)
				}
				if cfData.ASN != 13335 {
					t.Errorf("expected ASN 13335, got %d", cfData.ASN)
				}
			})
		})

		t.Run("When worker uses ctx.waitUntil for background execution", func(t *testing.T) {
			workerScript := []byte(`
export default {
    async fetch(req, env, ctx) {
        ctx.waitUntil(new Promise((resolve) => {
            setTimeout(() => {
                console.log("Background promise settled successfully");
                resolve();
            }, 30);
        }));
        return new Response("OK");
    }
};
`)
			res, err := application.RunWorkerBundleWithEnvAndBindings(
				context.Background(),
				workerScript,
				"GET",
				"/wait-until",
				nil,
				nil,
				nil,
				nil,
				"http://localhost:8000",
				"fetch",
			)

			t.Run("Then isolate awaits background promise and captures logs", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				var foundLog bool
				for _, log := range res.Logs {
					if strings.Contains(log.Message, "Background promise settled successfully") {
						foundLog = true
						break
					}
				}
				if !foundLog {
					t.Errorf("expected background log from ctx.waitUntil, logs: %+v", res.Logs)
				}
			})
		})

		t.Run("When worker uses global Cache API caches.default", func(t *testing.T) {
			workerScript := []byte(`
export default {
    async fetch(req, env) {
        const cache = caches.default;
        const cacheKey = "http://localhost/cached-asset";
        let match = await cache.match(cacheKey);
        if (!match) {
            await cache.put(cacheKey, new Response("freshly-cached-data", {
                headers: { "x-cubit-cache": "MISS" }
            }));
            return new Response("miss-stored");
        }
        const text = await match.text();
        return new Response("hit:" + text);
    }
};
`)
			res, err := application.RunWorkerBundleWithEnvAndBindings(
				context.Background(),
				workerScript,
				"GET",
				"/cache",
				nil,
				nil,
				nil,
				nil,
				"http://localhost:8000",
				"fetch",
			)

			t.Run("Then Cache API stores response without throwing errors", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if string(res.Body) != "miss-stored" {
					t.Errorf("expected 'miss-stored', got %s", string(res.Body))
				}
			})
		})

		t.Run("When worker constructs WebSocketPair", func(t *testing.T) {
			workerScript := []byte(`
export default {
    async fetch(req) {
        const pair = new WebSocketPair();
        const [client, server] = Object.values(pair);
        server.accept();
        return new Response(null, { status: 101, webSocket: client });
    }
};
`)
			res, err := application.RunWorkerBundleWithEnvAndBindings(
				context.Background(),
				workerScript,
				"GET",
				"/ws",
				nil,
				nil,
				nil,
				nil,
				"http://localhost:8000",
				"fetch",
			)

			t.Run("Then WebSocketPair instantiates and returns 101 Switching Protocols", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if res.Status != 101 {
					t.Fatalf("expected status 101, got %d (body: %s)", res.Status, string(res.Body))
				}
			})
		})

		t.Run("When invoking scheduled lifecycle event", func(t *testing.T) {
			workerScript := []byte(`
export default {
    async scheduled(event, env, ctx) {
        console.log("Cron executed with schedule: " + event.cron);
    }
};
`)
			headers := map[string]string{
				"X-Cubit-Event": "scheduled",
				"X-Cubit-Cron":  "*/10 * * * *",
			}
			res, err := application.RunWorkerBundleWithEnvAndBindings(
				context.Background(),
				workerScript,
				"GET",
				"/scheduled",
				headers,
				nil,
				nil,
				nil,
				"http://localhost:8000",
				"scheduled",
			)

			t.Run("Then scheduled handler executes successfully", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if res.Status != 200 {
					t.Fatalf("expected status 200, got %d", res.Status)
				}
				var foundLog bool
				for _, log := range res.Logs {
					if strings.Contains(log.Message, "Cron executed with schedule: */10 * * * *") {
						foundLog = true
						break
					}
				}
				if !foundLog {
					t.Errorf("expected scheduled cron log, got: %+v", res.Logs)
				}
			})
		})

		t.Run("When invoking queue consumer lifecycle event", func(t *testing.T) {
			workerScript := []byte(`
export default {
    async queue(batch, env, ctx) {
        for (const msg of batch.messages) {
            console.log("Queue message received: " + msg.body.task);
            msg.ack();
        }
    }
};
`)
			headers := map[string]string{
				"X-Cubit-Event": "queue",
				"X-Cubit-Queue": "email-tasks",
			}
			batchBody := []byte(`[{"id":"msg_1","body":{"task":"send_welcome_email"}}]`)
			res, err := application.RunWorkerBundleWithEnvAndBindings(
				context.Background(),
				workerScript,
				"POST",
				"/queue",
				headers,
				batchBody,
				nil,
				nil,
				"http://localhost:8000",
				"queue",
			)

			t.Run("Then queue handler processes message batch", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if res.Status != 200 {
					t.Fatalf("expected status 200, got %d", res.Status)
				}
				var foundLog bool
				for _, log := range res.Logs {
					if strings.Contains(log.Message, "Queue message received: send_welcome_email") {
						foundLog = true
						break
					}
				}
				if !foundLog {
					t.Errorf("expected queue message log, got: %+v", res.Logs)
				}
			})
		})

		t.Run("When invoking worker with in-isolate KV, D1, and R2 bindings", func(t *testing.T) {
			// Setup a mock control plane server mimicking Cubit REST API
			mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.URL.Path == "/api/v1/kv/namespaces/kv-ns-1/values/user_config" && r.Method == http.MethodGet:
					_ = json.NewEncoder(w).Encode(map[string]any{"value": `{"theme":"dark"}`})
				case r.URL.Path == "/api/v1/d1/databases/d1-users/query" && r.Method == http.MethodPost:
					_ = json.NewEncoder(w).Encode(map[string]any{
						"columns":      []string{"id", "username"},
						"rows":         []map[string]any{{"id": "u1", "username": "alice"}},
						"rowsAffected": 0,
						"durationMs":   1.2,
					})
				case r.URL.Path == "/api/v1/r2/buckets/assets-bkt/upload" && r.Method == http.MethodPost:
					_ = json.NewEncoder(w).Encode(map[string]string{"status": "uploaded"})
				case r.URL.Path == "/api/v1/r2/buckets/assets-bkt/objects/avatar.png" && r.Method == http.MethodGet:
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("fake-png-binary-data"))
				default:
					http.NotFound(w, r)
				}
			}))
			defer mockServer.Close()

			workerScript := []byte(`
export default {
    async fetch(req, env) {
        // Test KV
        const config = await env.CONFIG_KV.get("user_config", "json");

        // Test D1
        const user = await env.USERS_DB.prepare("SELECT * FROM users WHERE id = ?").bind("u1").first();

        // Test R2 put and get
        await env.ASSETS.put("avatar.png", "fake-png-binary-data");
        const obj = await env.ASSETS.get("avatar.png");
        const objText = await obj.text();

        return new Response(JSON.stringify({
            theme: config?.theme,
            user: user?.username,
            avatar: objText
        }), { headers: { "Content-Type": "application/json" } });
    }
};
`)
			bindings := []domain.ResourceBinding{
				{Type: domain.BindingTypeKV, Name: "CONFIG_KV", ResourceID: "kv-ns-1"},
				{Type: domain.BindingTypeD1, Name: "USERS_DB", ResourceID: "d1-users"},
				{Type: domain.BindingTypeR2, Name: "ASSETS", ResourceID: "assets-bkt"},
			}

			res, err := application.RunWorkerBundleWithEnvAndBindings(
				context.Background(),
				workerScript,
				"GET",
				"/bindings-test",
				nil,
				nil,
				nil,
				bindings,
				mockServer.URL,
				"fetch",
			)

			t.Run("Then worker isolate interacts with KV, D1, and R2 natively", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v (body: %s)", err, string(res.Body))
				}
				if res.Status != 200 {
					t.Fatalf("expected 200, got %d (body: %s)", res.Status, string(res.Body))
				}
				var out struct {
					Theme  string `json:"theme"`
					User   string `json:"user"`
					Avatar string `json:"avatar"`
				}
				if err := json.Unmarshal(res.Body, &out); err != nil {
					t.Fatalf("failed parsing body %s: %v", string(res.Body), err)
				}
				if out.Theme != "dark" {
					t.Errorf("expected theme dark, got %s", out.Theme)
				}
				if out.User != "alice" {
					t.Errorf("expected user alice, got %s", out.User)
				}
				if out.Avatar != "fake-png-binary-data" {
					t.Errorf("expected avatar fake-png-binary-data, got %s", out.Avatar)
				}
			})
		})

		t.Run("When recording execution events with edge context and fetching metrics", func(t *testing.T) {
			app, _ := svc.Create(
				context.Background(),
				"metrics-telemetry-worker",
				domain.SourceTypeInline,
				"",
				"main",
				"",
				"export default { async fetch() { return new Response('ok'); } };",
				false,
				nil,
				nil,
			)

			// Record 2xx event from US / SFO
			svc.RecordExecutionEvent(app.ID, domain.RequestLogEvent{
				ID:         "ray_test_1",
				Timestamp:  time.Now().UTC(),
				Method:     "GET",
				Path:       "/api/hello",
				URL:        "http://test.localhost/api/hello",
				StatusCode: 200,
				DurationMs: 15.5,
				ClientIP:   "1.2.3.4",
				CF: map[string]interface{}{
					"country": "US",
					"colo":    "SFO",
					"city":    "San Francisco",
				},
			})

			// Record 500 event from DE / FRA
			svc.RecordExecutionEvent(app.ID, domain.RequestLogEvent{
				ID:         "ray_test_2",
				Timestamp:  time.Now().UTC(),
				Method:     "POST",
				Path:       "/api/fail",
				URL:        "http://test.localhost/api/fail",
				StatusCode: 500,
				DurationMs: 42.0,
				ClientIP:   "5.6.7.8",
				CF: map[string]interface{}{
					"country": "DE",
					"colo":    "FRA",
					"city":    "Frankfurt",
				},
			})

			metrics, err := svc.GetMetrics(context.Background(), app.ID)

			t.Run("Then metrics snapshot contains edge telemetry and recent event stream", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error getting metrics, got %v", err)
				}
				if metrics.TotalRequests != 2 {
					t.Errorf("expected 2 total requests, got %d", metrics.TotalRequests)
				}
				if metrics.Status2xx != 1 || metrics.Status5xx != 1 {
					t.Errorf("expected 1 2xx and 1 5xx, got %d and %d", metrics.Status2xx, metrics.Status5xx)
				}
				if metrics.SuccessRate != 50.0 {
					t.Errorf("expected 50%% success rate, got %f", metrics.SuccessRate)
				}
				if metrics.ErrorRate != 50.0 {
					t.Errorf("expected 50%% error rate, got %f", metrics.ErrorRate)
				}
				if metrics.RequestsByCountry["US"] != 1 || metrics.RequestsByCountry["DE"] != 1 {
					t.Errorf("expected 1 US and 1 DE, got %v", metrics.RequestsByCountry)
				}
				if metrics.RequestsByColo["SFO"] != 1 || metrics.RequestsByColo["FRA"] != 1 {
					t.Errorf("expected 1 SFO and 1 FRA, got %v", metrics.RequestsByColo)
				}
				if len(metrics.RecentEvents) != 2 {
					t.Fatalf("expected 2 recent events, got %d", len(metrics.RecentEvents))
				}
				if metrics.RecentEvents[0].CF["country"] != "DE" {
					t.Errorf("expected newest event first with country DE, got %v", metrics.RecentEvents[0].CF)
				}
				if metrics.RecentEvents[1].CF["country"] != "US" {
					t.Errorf("expected oldest event second with country US, got %v", metrics.RecentEvents[1].CF)
				}
			})
		})
	})
}

type mockAppDomainRegistrar struct {
	registeredRoutes []string
}

func (m *mockAppDomainRegistrar) RegisterRoutes(ctx context.Context, appID string, routes []string) ([]*domain.Domain, error) {
	m.registeredRoutes = append(m.registeredRoutes, routes...)
	var res []*domain.Domain
	for i := range routes {
		d, _ := domain.NewDomain(fmt.Sprintf("dom-%d", i), appID, "example.com", "/")
		res = append(res, d)
	}
	dExtra, _ := domain.NewDomain("dom-extra", appID, "*.example.com", "/")
	res = append(res, dExtra)
	return res, nil
}


