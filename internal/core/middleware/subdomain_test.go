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
