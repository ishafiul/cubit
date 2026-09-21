package node_test

import (
	"context"
	"testing"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/modules/node"
)

type mockNodeRepo struct {
	nodes map[string]*domain.Node
}

func newMockNodeRepo() *mockNodeRepo {
	return &mockNodeRepo{nodes: make(map[string]*domain.Node)}
}

func (m *mockNodeRepo) Save(ctx context.Context, n *domain.Node) error {
	m.nodes[n.ID] = n
	return nil
}

func (m *mockNodeRepo) GetByID(ctx context.Context, id string) (*domain.Node, error) {
	if n, ok := m.nodes[id]; ok {
		return n, nil
	}
	return nil, domain.NewNotFoundError("node not found: " + id)
}

func (m *mockNodeRepo) List(ctx context.Context) ([]*domain.Node, error) {
	var list []*domain.Node
	for _, n := range m.nodes {
		list = append(list, n)
	}
	return list, nil
}

func (m *mockNodeRepo) Update(ctx context.Context, n *domain.Node) error {
	if _, ok := m.nodes[n.ID]; !ok {
		return domain.NewNotFoundError("node not found: " + n.ID)
	}
	m.nodes[n.ID] = n
	return nil
}

func (m *mockNodeRepo) Delete(ctx context.Context, id string) error {
	if _, ok := m.nodes[id]; !ok {
		return domain.NewNotFoundError("node not found: " + id)
	}
	delete(m.nodes, id)
	return nil
}

type mockRouteSyncer struct {
	syncCount int
}

func (m *mockRouteSyncer) SyncRoutes(ctx context.Context) error {
	m.syncCount++
	return nil
}

func TestNodeService(t *testing.T) {
	t.Run("Given a fresh NodeService", func(t *testing.T) {
		repo := newMockNodeRepo()
		syncer := &mockRouteSyncer{}
		svc := node.NewService(repo, nil, syncer, "s3://cubit-fleet")

		t.Run("When registering a new fleet node", func(t *testing.T) {
			n, err := svc.RegisterNode(context.Background(), "baremetal-01", "192.168.1.10", 9090, 8080, "v0.5.1")

			t.Run("Then registration succeeds with active status and routes synced", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if n.Name != "baremetal-01" {
					t.Fatalf("expected name baremetal-01, got %s", n.Name)
				}
				if n.Status != domain.NodeStatusActive {
					t.Fatalf("expected active status, got %s", n.Status)
				}
				if syncer.syncCount == 0 {
					t.Errorf("expected route sync to be called")
				}
			})
		})

		t.Run("When draining and reactivating a fleet node", func(t *testing.T) {
			n, _ := svc.RegisterNode(context.Background(), "baremetal-02", "192.168.1.11", 9090, 8080, "v0.5.1")
			drained, err := svc.DrainNode(context.Background(), n.ID)

			t.Run("Then node status transitions to draining", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if drained.Status != domain.NodeStatusDraining {
					t.Fatalf("expected status draining, got %s", drained.Status)
				}
			})

			t.Run("Then activating restores status to active", func(t *testing.T) {
				activated, err := svc.ActivateNode(context.Background(), n.ID)
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if activated.Status != domain.NodeStatusActive {
					t.Fatalf("expected status active, got %s", activated.Status)
				}
			})
		})

		t.Run("When attempting to drain or delete the protected main node", func(t *testing.T) {
			mainNode, _ := svc.RegisterNode(context.Background(), "worker-node-01", "127.0.0.1", 9091, 8081, "v0.5.1")
			mainNode.IsProtected = true
			_ = repo.Update(context.Background(), mainNode)

			_, drainErr := svc.DrainNode(context.Background(), mainNode.ID)
			deleteErr := svc.DeleteNode(context.Background(), mainNode.ID)

			t.Run("Then drain is forbidden", func(t *testing.T) {
				if drainErr == nil {
					t.Fatalf("expected drain error on protected main node, got nil")
				}
			})

			t.Run("Then delete is forbidden", func(t *testing.T) {
				if deleteErr == nil {
					t.Fatalf("expected delete error on protected main node, got nil")
				}
			})
		})
	})
}
