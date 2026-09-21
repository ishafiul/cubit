package application_test

import (
	"context"
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
				true,
				nil,
				nil,
			)

			autoDeployFalse := false
			cDate := "2024-09-23"
			flags := []string{"nodejs_compat"}
			mem := 256
			dur := 100
			updated, err := svc.Update(context.Background(), app.ID, "staging", "", &autoDeployFalse, nil, nil, &cDate, &flags, &mem, &dur)

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
		})
	})
}

