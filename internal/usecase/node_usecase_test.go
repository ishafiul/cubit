package usecase_test

import (
	"context"
	"testing"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/usecase"
)

// In-memory mock NodeRepository
type mockNodeRepo struct {
	nodes map[string]*domain.Node
}

func newMockNodeRepo() *mockNodeRepo {
	return &mockNodeRepo{nodes: make(map[string]*domain.Node)}
}

func (m *mockNodeRepo) Save(ctx context.Context, node *domain.Node) error {
	m.nodes[node.ID] = node
	return nil
}

func (m *mockNodeRepo) GetByID(ctx context.Context, id string) (*domain.Node, error) {
	node, exists := m.nodes[id]
	if !exists {
		return nil, domain.NewNotFoundError("node not found")
	}
	return node, nil
}

func (m *mockNodeRepo) List(ctx context.Context) ([]*domain.Node, error) {
	var list []*domain.Node
	for _, n := range m.nodes {
		list = append(list, n)
	}
	return list, nil
}

func (m *mockNodeRepo) Delete(ctx context.Context, id string) error {
	delete(m.nodes, id)
	return nil
}

func (m *mockNodeRepo) Update(ctx context.Context, node *domain.Node) error {
	m.nodes[node.ID] = node
	return nil
}

// Mock ContainerSupervisor
type mockSupervisor struct {
	startedNodes map[string]string
}

func newMockSupervisor() *mockSupervisor {
	return &mockSupervisor{startedNodes: make(map[string]string)}
}

func (m *mockSupervisor) StartCelld(ctx context.Context, node *domain.Node, version string, bucketURL string) error {
	m.startedNodes[node.ID] = version
	return nil
}

func (m *mockSupervisor) StopCelld(ctx context.Context, node *domain.Node) error {
	delete(m.startedNodes, node.ID)
	return nil
}

func (m *mockSupervisor) GracefulRestartCelld(ctx context.Context, node *domain.Node, newVersion string, bucketURL string) error {
	m.startedNodes[node.ID] = newVersion
	return nil
}

func TestNodeUsecase(t *testing.T) {
	ctx := context.Background()

	t.Run("Given an empty node repository and supervisor", func(t *testing.T) {
		repo := newMockNodeRepo()
		supervisor := newMockSupervisor()
		uc := usecase.NewNodeUsecase(repo, supervisor, "s3://fleet-bucket")

		t.Run("When registering a new node then it saves and starts celld container", func(t *testing.T) {
			node, err := uc.RegisterNode(ctx, "node-1", "10.0.0.1", 8081, 8080, "0.2.0")

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if node.Name != "node-1" {
				t.Errorf("expected node name node-1, got %s", node.Name)
			}
			if supervisor.startedNodes[node.ID] != "0.2.0" {
				t.Errorf("expected celld started with version 0.2.0, got %s", supervisor.startedNodes[node.ID])
			}
		})
	})

	t.Run("Given an active registered node", func(t *testing.T) {
		repo := newMockNodeRepo()
		supervisor := newMockSupervisor()
		uc := usecase.NewNodeUsecase(repo, supervisor, "s3://fleet-bucket")
		node, err := uc.RegisterNode(ctx, "node-1", "10.0.0.1", 8081, 8080, "0.2.0")
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		t.Run("When draining the node then status is updated in repository", func(t *testing.T) {
			drained, drainErr := uc.DrainNode(ctx, node.ID)

			if drainErr != nil {
				t.Fatalf("expected no error, got %v", drainErr)
			}
			if drained.Status != domain.NodeStatusDraining {
				t.Errorf("expected status draining, got %s", drained.Status)
			}
		})

		t.Run("When deleting the node then it is removed from repo and supervisor", func(t *testing.T) {
			delErr := uc.DeleteNode(ctx, node.ID)

			if delErr != nil {
				t.Fatalf("expected no error, got %v", delErr)
			}
			_, getErr := repo.GetByID(ctx, node.ID)
			if getErr == nil {
				t.Error("expected node to be deleted from repository")
			}
			if _, exists := supervisor.startedNodes[node.ID]; exists {
				t.Error("expected celld container to be stopped in supervisor")
			}
		})
	})
}
