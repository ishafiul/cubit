package runtime_test

import (
	"context"
	"testing"

	"github.com/ishaf/cubit/internal/domain"
	rtModule "github.com/ishaf/cubit/internal/modules/runtime"
)

type mockNodeLister struct {
	nodes []*domain.Node
}

func (m *mockNodeLister) List(ctx context.Context) ([]*domain.Node, error) {
	return m.nodes, nil
}

func (m *mockNodeLister) Update(ctx context.Context, n *domain.Node) error {
	for i, node := range m.nodes {
		if node.ID == n.ID {
			m.nodes[i] = n
			return nil
		}
	}
	return nil
}

func TestRuntimeService(t *testing.T) {
	t.Run("Given a RuntimeService with active fleet nodes", func(t *testing.T) {
		nodes := []*domain.Node{
			{ID: "node-1", Name: "n1", Status: domain.NodeStatusActive, CelldVersion: "0.5.1"},
			{ID: "node-2", Name: "n2", Status: domain.NodeStatusActive, CelldVersion: "0.5.1"},
		}
		repo := &mockNodeLister{nodes: nodes}
		svc := rtModule.NewService(repo, nil, nil, "s3://cubit-fleet", "0.5.1")

		t.Run("When inspecting runtime status", func(t *testing.T) {
			status, err := svc.GetStatus(context.Background())

			t.Run("Then status reflects active node count and version", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				if status.ActiveNodesCount != 2 {
					t.Fatalf("expected 2 active nodes, got %d", status.ActiveNodesCount)
				}
				if status.CurrentCelldVersion != "0.5.1" {
					t.Fatalf("expected celld version 0.5.1, got %s", status.CurrentCelldVersion)
				}
			})
		})

		t.Run("When upgrading celld daemon", func(t *testing.T) {
			err := svc.UpgradeCelldDaemon(context.Background(), "0.5.2")

			t.Run("Then all active nodes are updated to target version", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				st, _ := svc.GetStatus(context.Background())
				if st.CurrentCelldVersion != "0.5.2" {
					t.Fatalf("expected current celld version 0.5.2, got %s", st.CurrentCelldVersion)
				}
			})
		})
	})
}
