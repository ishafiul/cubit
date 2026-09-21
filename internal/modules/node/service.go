package node

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/ishaf/cubit/internal/domain"
)

// ContainerSupervisor controls celld runtime processes for nodes.
type ContainerSupervisor interface {
	StartCelld(ctx context.Context, node *domain.Node, version string, bucketURL string) error
	StopCelld(ctx context.Context, node *domain.Node) error
	GracefulRestartCelld(ctx context.Context, node *domain.Node, newVersion string, bucketURL string) error
}

// Service defines fleet node business operations.
type Service interface {
	RegisterNode(ctx context.Context, name, ipAddress string, internalPort, workerPort int, celldVersion string) (*domain.Node, error)
	GetNode(ctx context.Context, id string) (*domain.Node, error)
	ListNodes(ctx context.Context) ([]*domain.Node, error)
	DrainNode(ctx context.Context, id string) (*domain.Node, error)
	DeleteNode(ctx context.Context, id string) error
}

// NodeService coordinates fleet nodes.
type NodeService struct {
	repo       Repository
	supervisor ContainerSupervisor
	bucketURL  string
}

// NewService creates a new NodeService.
func NewService(repo Repository, supervisor ContainerSupervisor, bucketURL string) *NodeService {
	return &NodeService{
		repo:       repo,
		supervisor: supervisor,
		bucketURL:  bucketURL,
	}
}

func generateID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// RegisterNode registers a new node and launches celld.
func (s *NodeService) RegisterNode(ctx context.Context, name, ipAddress string, internalPort, workerPort int, celldVersion string) (*domain.Node, error) {
	nodeID := generateID()
	node, err := domain.NewNode(nodeID, name, ipAddress, internalPort, workerPort, celldVersion)
	if err != nil {
		return nil, err
	}

	if err := s.repo.Save(ctx, node); err != nil {
		return nil, err
	}

	if s.supervisor != nil {
		if err := s.supervisor.StartCelld(ctx, node, celldVersion, s.bucketURL); err != nil {
			node.MarkOffline()
			_ = s.repo.Update(ctx, node)
			return nil, fmt.Errorf("failed to start celld on node: %w", err)
		}
	}

	return node, nil
}

// GetNode retrieves a node by ID.
func (s *NodeService) GetNode(ctx context.Context, id string) (*domain.Node, error) {
	return s.repo.GetByID(ctx, id)
}

// ListNodes lists all registered nodes.
func (s *NodeService) ListNodes(ctx context.Context) ([]*domain.Node, error) {
	return s.repo.List(ctx)
}

// DrainNode marks a node as draining to facilitate graceful lease handover.
func (s *NodeService) DrainNode(ctx context.Context, id string) (*domain.Node, error) {
	node, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := node.Drain(); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, node); err != nil {
		return nil, err
	}

	return node, nil
}

// DeleteNode removes a node and stops its supervised celld container.
func (s *NodeService) DeleteNode(ctx context.Context, id string) error {
	node, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if s.supervisor != nil {
		_ = s.supervisor.StopCelld(ctx, node)
	}

	return s.repo.Delete(ctx, id)
}
