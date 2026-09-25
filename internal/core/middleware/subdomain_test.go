package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/infrastructure/db"
	appModule "github.com/ishaf/cubit/internal/modules/application"
	"github.com/ishaf/cubit/internal/adapters/out/storage"
)

type mockInvoker struct {
	app *domain.Application
	err error
}

func (m *mockInvoker) GetApplicationBySubdomain(ctx context.Context, subdomain string) (*domain.Application, error) {
	return m.app, m.err
}

func (m *mockInvoker) InvokeApplication(ctx context.Context, appID, method, uri string, headers map[string]string, body []byte) (int, map[string]string, []byte, error) {
	return 200, map[string]string{"Content-Type": "text/html"}, []byte("<h1>Hello from Worker</h1>"), nil
}

func TestSubdomainRouter(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	inv := &mockInvoker{
		app: &domain.Application{
			ID:                 "app-1",
			Name:               "cubit-react",
			Subdomain:          "cubit-react",
			ActiveDeploymentID: "dep-1",
		},
	}
	r.Use(SubdomainRouter(inv))
	r.NoRoute(func(c *gin.Context) {
		c.String(200, "SPA Dashboard")
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Host = "cubit-react.localhost:8000"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Body.String() != "<h1>Hello from Worker</h1>" {
		t.Fatalf("expected worker response, got: %s", w.Body.String())
	}
}

func TestSubdomainRouter_EdgeHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var capturedHeaders map[string]string

	inv := &mockHeaderCaptureInvoker{
		app: &domain.Application{
			ID:                 "app-edge",
			Name:               "edge-worker",
			Subdomain:          "edge-worker",
			ActiveDeploymentID: "dep-1",
		},
		onInvoke: func(headers map[string]string) {
			capturedHeaders = headers
		},
	}

	r := gin.New()
	r.Use(SubdomainRouter(inv))

	req := httptest.NewRequest("GET", "/api/data", nil)
	req.Host = "edge-worker.localhost:8000"
	req.RemoteAddr = "198.51.100.77:5432"

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if capturedHeaders == nil {
		t.Fatal("expected InvokeApplication to be called and headers captured")
	}

	t.Run("Then CF-Connecting-IP matches client IP", func(t *testing.T) {
		if capturedHeaders["CF-Connecting-IP"] != "198.51.100.77" {
			t.Errorf("expected CF-Connecting-IP 198.51.100.77, got %q", capturedHeaders["CF-Connecting-IP"])
		}
	})

	t.Run("Then CF-IPCountry is present", func(t *testing.T) {
		if capturedHeaders["CF-IPCountry"] == "" {
			t.Error("expected non-empty CF-IPCountry")
		}
	})

	t.Run("Then CF-IPCity is present", func(t *testing.T) {
		if capturedHeaders["CF-IPCity"] == "" {
			t.Error("expected non-empty CF-IPCity")
		}
	})

	t.Run("Then CF-Ray is populated", func(t *testing.T) {
		if capturedHeaders["CF-Ray"] == "" {
			t.Error("expected non-empty CF-Ray")
		}
	})

	t.Run("Then CF-Visitor is populated with scheme", func(t *testing.T) {
		if capturedHeaders["CF-Visitor"] == "" {
			t.Error("expected non-empty CF-Visitor")
		}
	})
}

type mockHeaderCaptureInvoker struct {
	app      *domain.Application
	onInvoke func(headers map[string]string)
}

func (m *mockHeaderCaptureInvoker) GetApplicationBySubdomain(ctx context.Context, subdomain string) (*domain.Application, error) {
	return m.app, nil
}

func (m *mockHeaderCaptureInvoker) InvokeApplication(ctx context.Context, appID, method, uri string, headers map[string]string, body []byte) (int, map[string]string, []byte, error) {
	if m.onInvoke != nil {
		m.onInvoke(headers)
	}
	return 200, map[string]string{"Content-Type": "application/json"}, []byte(`{"ok":true}`), nil
}

func TestRealAppServiceSubdomain(t *testing.T) {
	database, err := db.OpenSQLite("file:../../../.data/cubit.db?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		t.Fatalf("db open failed: %v", err)
	}
	defer database.Close()

	appRepo := appModule.NewRepository(database)
	storageAdapter, _ := storage.NewLocalStorageAdapter("../../../.data/storage", string(storage.DriverGarageLocal))
	appService := appModule.NewService(appRepo, storageAdapter, nil, "cubit-fleet")

	app, err := appService.GetApplicationBySubdomain(context.Background(), "cubit-react")
	if err != nil {
		t.Fatalf("failed getting app by subdomain: %v", err)
	}
	t.Logf("found app: %s (id: %s, activeDep: %s)", app.Name, app.ID, app.ActiveDeploymentID)

	status, _, body, err := appService.InvokeApplication(context.Background(), app.ID, "GET", "/", map[string]string{"Host": "cubit-react.localhost:8000"}, nil)
	t.Logf("InvokeApplication status: %d, body:\n%s", status, string(body))
	if err != nil {
		t.Fatalf("Invoke failed: %v", err)
	}
}
