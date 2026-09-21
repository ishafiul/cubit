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

func TestNodeService(t *testing.T) {
	t.Run("Given a fresh NodeService", func(t *testing.T) {
		repo := newMockNodeRepo()
		svc := node.NewService(repo, nil, "s3://cubit-fleet")

		t.Run("When registering a new fleet node", func(t *testing.T) {
			n, err := svc.RegisterNode(context.Background(), "baremetal-01", "192.168.1.10", 9090, 8080, "v0.5.1")

			t.Run("Then registration succeeds with online status", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if n.Name != "baremetal-01" {
					t.Fatalf("expected name baremetal-01, got %s", n.Name)
				}
				if n.Status != domain.NodeStatusActive {
					t.Fatalf("expected active status, got %s", n.Status)
				}
			})
		})

		t.Run("When draining a fleet node", func(t *testing.T) {
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
		})
	})
}
