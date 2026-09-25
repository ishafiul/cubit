package deployment_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/modules/deployment"
)

// mockAppPortStore serves as both AppPortLister and AppPortUpdater for testing.
type mockAppPortStore struct {
	apps map[string]*domain.Application
}

func newMockAppPortStore() *mockAppPortStore {
	return &mockAppPortStore{apps: make(map[string]*domain.Application)}
}

func (m *mockAppPortStore) List(ctx context.Context) ([]*domain.Application, error) {
	var result []*domain.Application
	for _, a := range m.apps {
		result = append(result, a)
	}
	return result, nil
}

func (m *mockAppPortStore) GetByID(ctx context.Context, id string) (*domain.Application, error) {
	a, ok := m.apps[id]
	if !ok {
		return nil, domain.NewNotFoundError("not found: " + id)
	}
	return a, nil
}

func (m *mockAppPortStore) Update(ctx context.Context, app *domain.Application) error {
	if _, ok := m.apps[app.ID]; !ok {
		return domain.NewNotFoundError("not found: " + app.ID)
	}
	m.apps[app.ID] = app
	return nil
}

func (m *mockAppPortStore) addApp(id, name string, port int) {
	m.apps[id] = &domain.Application{
		ID:         id,
		Name:       name,
		WorkerPort: port,
		Status:     domain.AppStatusRunning,
	}
}

func TestPortAllocator(t *testing.T) {
	ctx := context.Background()

	t.Run("Given an empty fleet When allocating a port Then returns range start", func(t *testing.T) {
		store := newMockAppPortStore()
		store.addApp("app-1", "worker-a", 0)

		pa := deployment.NewPortAllocatorWithRange(store, store, 9000, 9010)
		port, err := pa.AllocatePort(ctx, "app-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if port != 9000 {
			t.Errorf("expected port 9000, got %d", port)
		}
		// Verify persistence
		if store.apps["app-1"].WorkerPort != 9000 {
			t.Errorf("expected stored WorkerPort 9000, got %d", store.apps["app-1"].WorkerPort)
		}
	})

	t.Run("Given an app already has a port When allocating Then returns existing port", func(t *testing.T) {
		store := newMockAppPortStore()
		store.addApp("app-1", "worker-a", 9005)

		pa := deployment.NewPortAllocatorWithRange(store, store, 9000, 9010)
		port, err := pa.AllocatePort(ctx, "app-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if port != 9005 {
			t.Errorf("expected existing port 9005, got %d", port)
		}
	})

	t.Run("Given some ports are occupied When allocating Then returns next available", func(t *testing.T) {
		store := newMockAppPortStore()
		store.addApp("app-1", "worker-a", 9000)
		store.addApp("app-2", "worker-b", 9001)
		store.addApp("app-3", "worker-c", 0)

		pa := deployment.NewPortAllocatorWithRange(store, store, 9000, 9010)
		port, err := pa.AllocatePort(ctx, "app-3")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if port != 9002 {
			t.Errorf("expected port 9002, got %d", port)
		}
	})

	t.Run("Given all ports are exhausted When allocating Then returns error", func(t *testing.T) {
		store := newMockAppPortStore()
		// Fill the entire range [9000, 9002] with 3 ports
		store.addApp("app-1", "worker-a", 9000)
		store.addApp("app-2", "worker-b", 9001)
		store.addApp("app-3", "worker-c", 9002)
		store.addApp("app-4", "worker-d", 0)

		pa := deployment.NewPortAllocatorWithRange(store, store, 9000, 9002)
		_, err := pa.AllocatePort(ctx, "app-4")
		if err == nil {
			t.Fatal("expected error when all ports are exhausted")
		}
	})

	t.Run("Given an allocated port When releasing Then clears the port", func(t *testing.T) {
		store := newMockAppPortStore()
		store.addApp("app-1", "worker-a", 9005)

		pa := deployment.NewPortAllocatorWithRange(store, store, 9000, 9010)
		if err := pa.ReleasePort(ctx, "app-1"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if store.apps["app-1"].WorkerPort != 0 {
			t.Errorf("expected WorkerPort 0 after release, got %d", store.apps["app-1"].WorkerPort)
		}
	})

	t.Run("Given released port When allocating new app Then reuses freed port", func(t *testing.T) {
		store := newMockAppPortStore()
		store.addApp("app-1", "worker-a", 9000)
		store.addApp("app-2", "worker-b", 9001)
		store.addApp("app-3", "worker-c", 0)

		pa := deployment.NewPortAllocatorWithRange(store, store, 9000, 9010)

		// Release port 9000
		if err := pa.ReleasePort(ctx, "app-1"); err != nil {
			t.Fatalf("unexpected release error: %v", err)
		}

		// Allocate for app-3, should get 9000 (the freed one, since it's lowest available)
		port, err := pa.AllocatePort(ctx, "app-3")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if port != 9000 {
			t.Errorf("expected reused port 9000, got %d", port)
		}
	})

	t.Run("Given a non-existent app When allocating Then returns error", func(t *testing.T) {
		store := newMockAppPortStore()
		pa := deployment.NewPortAllocatorWithRange(store, store, 9000, 9010)
		_, err := pa.AllocatePort(ctx, "ghost-app")
		if err == nil {
			t.Fatal("expected error for non-existent app")
		}
	})

	t.Run("Given multiple apps When listing allocations Then returns sorted by port", func(t *testing.T) {
		store := newMockAppPortStore()
		store.addApp("app-1", "worker-a", 9005)
		store.addApp("app-2", "worker-b", 9001)
		store.addApp("app-3", "worker-c", 0)

		pa := deployment.NewPortAllocatorWithRange(store, store, 9000, 9010)
		allocations, err := pa.ListAllocations(ctx)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(allocations) != 2 {
			t.Fatalf("expected 2 allocations, got %d", len(allocations))
		}
		if allocations[0].Port != 9001 {
			t.Errorf("expected first allocation port 9001, got %d", allocations[0].Port)
		}
		if allocations[1].Port != 9005 {
			t.Errorf("expected second allocation port 9005, got %d", allocations[1].Port)
		}
	})

	t.Run("Given gaps in allocation When allocating Then fills lowest gap first", func(t *testing.T) {
		store := newMockAppPortStore()
		store.addApp("app-1", "worker-a", 9000)
		store.addApp("app-2", "worker-b", 9002) // gap at 9001
		store.addApp("app-3", "worker-c", 9004) // gap at 9003
		store.addApp("app-new", "worker-new", 0)

		pa := deployment.NewPortAllocatorWithRange(store, store, 9000, 9010)
		port, err := pa.AllocatePort(ctx, "app-new")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if port != 9001 {
			t.Errorf("expected gap-filling port 9001, got %d", port)
		}
	})
}

func TestPortAllocator_DefaultRange(t *testing.T) {
	t.Run("Given default constructor When allocating Then uses range 9000-9999", func(t *testing.T) {
		store := newMockAppPortStore()
		store.addApp("app-1", "worker-a", 0)

		pa := deployment.NewPortAllocator(store, store)
		port, err := pa.AllocatePort(context.Background(), "app-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if port != 9000 {
			t.Errorf("expected default range start 9000, got %d", port)
		}
	})
}

func TestPortAllocator_Sequential(t *testing.T) {
	t.Run("Given sequential allocations When allocating 5 apps Then assigns consecutive ports", func(t *testing.T) {
		store := newMockAppPortStore()
		for i := 0; i < 5; i++ {
			store.addApp(fmt.Sprintf("app-%d", i), fmt.Sprintf("worker-%d", i), 0)
		}

		pa := deployment.NewPortAllocatorWithRange(store, store, 9000, 9010)
		for i := 0; i < 5; i++ {
			port, err := pa.AllocatePort(context.Background(), fmt.Sprintf("app-%d", i))
			if err != nil {
				t.Fatalf("unexpected error allocating app-%d: %v", i, err)
			}
			if port != 9000+i {
				t.Errorf("app-%d: expected port %d, got %d", i, 9000+i, port)
			}
		}
	})
}
