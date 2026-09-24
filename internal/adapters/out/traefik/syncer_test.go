package traefik_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ishaf/cubit/internal/adapters/out/traefik"
	"github.com/ishaf/cubit/internal/domain"
)

type mockAppLister struct {
	apps []*domain.Application
}

func (m *mockAppLister) List(ctx context.Context) ([]*domain.Application, error) {
	return m.apps, nil
}

type mockDomainLister struct {
	domainsByApp map[string][]*domain.Domain
}

func (m *mockDomainLister) ListByAppID(ctx context.Context, appID string) ([]*domain.Domain, error) {
	return m.domainsByApp[appID], nil
}

type mockNodeLister struct {
	nodes []*domain.Node
}

func (m *mockNodeLister) List(ctx context.Context) ([]*domain.Node, error) {
	return m.nodes, nil
}

func TestRouteSyncer_PerAppPortRouting(t *testing.T) {
	ctx := context.Background()

	t.Run("Given two apps with different worker ports When syncing routes Then each app routes to its own port", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "syncer-test-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		outFile := filepath.Join(tmpDir, "cubit.yaml")
		provider := traefik.NewFileProvider(outFile, "letsencrypt")

		nodes := &mockNodeLister{nodes: []*domain.Node{
			{ID: "node-1", Name: "n1", IPAddress: "10.0.0.1", WorkerPort: 8080, InternalPort: 8081, Status: domain.NodeStatusActive},
			{ID: "node-2", Name: "n2", IPAddress: "10.0.0.2", WorkerPort: 8080, InternalPort: 8081, Status: domain.NodeStatusActive},
		}}

		apps := &mockAppLister{apps: []*domain.Application{
			{ID: "app-1", Name: "api-service", Subdomain: "api-service", Status: domain.AppStatusRunning, WorkerPort: 9001},
			{ID: "app-2", Name: "web-frontend", Subdomain: "web-frontend", Status: domain.AppStatusRunning, WorkerPort: 9002},
		}}

		domains := &mockDomainLister{domainsByApp: make(map[string][]*domain.Domain)}

		syncer := traefik.NewRouteSyncer(provider, apps, domains, nodes)
		if err := syncer.SyncRoutes(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("expected config file to exist: %v", err)
		}
		yamlStr := string(content)

		t.Run("Then api-service routes to port 9001 on both nodes", func(t *testing.T) {
			if !strings.Contains(yamlStr, "http://10.0.0.1:9001") {
				t.Errorf("expected api-service target on node-1 port 9001, got:\n%s", yamlStr)
			}
			if !strings.Contains(yamlStr, "http://10.0.0.2:9001") {
				t.Errorf("expected api-service target on node-2 port 9001, got:\n%s", yamlStr)
			}
		})

		t.Run("Then web-frontend routes to port 9002 on both nodes", func(t *testing.T) {
			if !strings.Contains(yamlStr, "http://10.0.0.1:9002") {
				t.Errorf("expected web-frontend target on node-1 port 9002, got:\n%s", yamlStr)
			}
			if !strings.Contains(yamlStr, "http://10.0.0.2:9002") {
				t.Errorf("expected web-frontend target on node-2 port 9002, got:\n%s", yamlStr)
			}
		})

		t.Run("Then no routes use the shared node port 8080", func(t *testing.T) {
			if strings.Contains(yamlStr, ":8080") {
				t.Errorf("expected no shared port 8080 when per-app ports are assigned, got:\n%s", yamlStr)
			}
		})
	})

	t.Run("Given an app without assigned port When syncing routes Then falls back to node worker port", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "syncer-fallback-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		outFile := filepath.Join(tmpDir, "cubit.yaml")
		provider := traefik.NewFileProvider(outFile, "letsencrypt")

		nodes := &mockNodeLister{nodes: []*domain.Node{
			{ID: "node-1", Name: "n1", IPAddress: "10.0.0.1", WorkerPort: 8080, InternalPort: 8081, Status: domain.NodeStatusActive},
		}}

		apps := &mockAppLister{apps: []*domain.Application{
			{ID: "app-1", Name: "legacy-worker", Subdomain: "legacy-worker", Status: domain.AppStatusRunning, WorkerPort: 0},
		}}

		domains := &mockDomainLister{domainsByApp: make(map[string][]*domain.Domain)}

		syncer := traefik.NewRouteSyncer(provider, apps, domains, nodes)
		if err := syncer.SyncRoutes(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("expected config file to exist: %v", err)
		}
		yamlStr := string(content)

		t.Run("Then routes use the node's shared worker port 8080", func(t *testing.T) {
			if !strings.Contains(yamlStr, "http://10.0.0.1:8080") {
				t.Errorf("expected fallback to node port 8080, got:\n%s", yamlStr)
			}
		})
	})

	t.Run("Given mixed apps with and without ports When syncing Then each uses correct port", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "syncer-mixed-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		outFile := filepath.Join(tmpDir, "cubit.yaml")
		provider := traefik.NewFileProvider(outFile, "letsencrypt")

		nodes := &mockNodeLister{nodes: []*domain.Node{
			{ID: "node-1", Name: "n1", IPAddress: "10.0.0.1", WorkerPort: 8080, InternalPort: 8081, Status: domain.NodeStatusActive},
		}}

		apps := &mockAppLister{apps: []*domain.Application{
			{ID: "app-1", Name: "with-port", Subdomain: "with-port", Status: domain.AppStatusRunning, WorkerPort: 9500},
			{ID: "app-2", Name: "without-port", Subdomain: "without-port", Status: domain.AppStatusRunning, WorkerPort: 0},
		}}

		domains := &mockDomainLister{domainsByApp: make(map[string][]*domain.Domain)}

		syncer := traefik.NewRouteSyncer(provider, apps, domains, nodes)
		if err := syncer.SyncRoutes(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("expected config file to exist: %v", err)
		}
		yamlStr := string(content)

		t.Run("Then app with assigned port uses 9500", func(t *testing.T) {
			if !strings.Contains(yamlStr, "http://10.0.0.1:9500") {
				t.Errorf("expected port 9500 for with-port app, got:\n%s", yamlStr)
			}
		})

		t.Run("Then app without port falls back to 8080", func(t *testing.T) {
			if !strings.Contains(yamlStr, "http://10.0.0.1:8080") {
				t.Errorf("expected fallback port 8080 for without-port app, got:\n%s", yamlStr)
			}
		})
	})

	t.Run("Given only draining nodes When syncing Then writes empty config", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "syncer-drain-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		outFile := filepath.Join(tmpDir, "cubit.yaml")
		provider := traefik.NewFileProvider(outFile, "letsencrypt")

		nodes := &mockNodeLister{nodes: []*domain.Node{
			{ID: "node-1", Name: "n1", IPAddress: "10.0.0.1", WorkerPort: 8080, InternalPort: 8081, Status: domain.NodeStatusDraining},
		}}

		apps := &mockAppLister{apps: []*domain.Application{
			{ID: "app-1", Name: "my-worker", Subdomain: "my-worker", Status: domain.AppStatusRunning, WorkerPort: 9001},
		}}

		domains := &mockDomainLister{domainsByApp: make(map[string][]*domain.Domain)}

		syncer := traefik.NewRouteSyncer(provider, apps, domains, nodes)
		if err := syncer.SyncRoutes(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("expected config file to exist: %v", err)
		}
		yamlStr := string(content)

		t.Run("Then no routes contain worker port targets", func(t *testing.T) {
			if strings.Contains(yamlStr, "9001") {
				t.Errorf("expected no routes for draining nodes, got:\n%s", yamlStr)
			}
		})
	})

	t.Run("Given app with custom domain When syncing Then custom domain routes use app port", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "syncer-custom-*")
		if err != nil {
			t.Fatalf("failed to create temp dir: %v", err)
		}
		defer os.RemoveAll(tmpDir)

		outFile := filepath.Join(tmpDir, "cubit.yaml")
		provider := traefik.NewFileProvider(outFile, "letsencrypt")

		nodes := &mockNodeLister{nodes: []*domain.Node{
			{ID: "node-1", Name: "n1", IPAddress: "10.0.0.1", WorkerPort: 8080, InternalPort: 8081, Status: domain.NodeStatusActive},
		}}

		apps := &mockAppLister{apps: []*domain.Application{
			{ID: "app-1", Name: "my-api", Subdomain: "my-api", Status: domain.AppStatusRunning, WorkerPort: 9100},
		}}

		domains := &mockDomainLister{domainsByApp: map[string][]*domain.Domain{
			"app-1": {
				{ID: "dom-1", ApplicationID: "app-1", Hostname: "api.example.com", PathPrefix: "/v1", SSLActive: true},
			},
		}}

		syncer := traefik.NewRouteSyncer(provider, apps, domains, nodes)
		if err := syncer.SyncRoutes(ctx); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(outFile)
		if err != nil {
			t.Fatalf("expected config file to exist: %v", err)
		}
		yamlStr := string(content)

		t.Run("Then custom domain route uses app port 9100", func(t *testing.T) {
			if !strings.Contains(yamlStr, "http://10.0.0.1:9100") {
				t.Errorf("expected custom domain target at port 9100, got:\n%s", yamlStr)
			}
		})

		t.Run("Then custom domain host rule is present", func(t *testing.T) {
			if !strings.Contains(yamlStr, "Host(`api.example.com`)") {
				t.Errorf("expected custom domain Host rule, got:\n%s", yamlStr)
			}
		})

		t.Run("Then TLS is configured for custom domain", func(t *testing.T) {
			if !strings.Contains(yamlStr, "certResolver: letsencrypt") {
				t.Errorf("expected TLS cert resolver for custom domain, got:\n%s", yamlStr)
			}
		})
	})
}
