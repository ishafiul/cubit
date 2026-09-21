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

// RouteSyncer triggers Traefik routing rules refresh across the fleet.
type RouteSyncer interface {
	SyncRoutes(ctx context.Context) error
}

// Service defines fleet node business operations.
type Service interface {
	RegisterNode(ctx context.Context, name, ipAddress string, internalPort, workerPort int, celldVersion string) (*domain.Node, error)
	GetNode(ctx context.Context, id string) (*domain.Node, error)
	ListNodes(ctx context.Context) ([]*domain.Node, error)
	DrainNode(ctx context.Context, id string) (*domain.Node, error)
	ActivateNode(ctx context.Context, id string) (*domain.Node, error)
	DeleteNode(ctx context.Context, id string) error
}

// NodeService coordinates fleet nodes.
type NodeService struct {
	repo        Repository
	supervisor  ContainerSupervisor
	routeSyncer RouteSyncer
	bucketURL   string
}

// NewService creates a new NodeService.
func NewService(repo Repository, supervisor ContainerSupervisor, routeSyncer RouteSyncer, bucketURL string) *NodeService {
	return &NodeService{
		repo:        repo,
		supervisor:  supervisor,
		routeSyncer: routeSyncer,
		bucketURL:   bucketURL,
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

	if s.routeSyncer != nil {
		_ = s.routeSyncer.SyncRoutes(ctx)
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

	if node.IsProtected || node.Name == "worker-node-01" {
		return nil, domain.NewForbiddenError("main node is protected and cannot be drained")
	}

	if err := node.Drain(); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, node); err != nil {
		return nil, err
	}

	if s.routeSyncer != nil {
		_ = s.routeSyncer.SyncRoutes(ctx)
	}

	return node, nil
}

// ActivateNode restores a draining or offline node back to active status.
func (s *NodeService) ActivateNode(ctx context.Context, id string) (*domain.Node, error) {
	node, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := node.MarkActive(); err != nil {
		return nil, err
	}

	if err := s.repo.Update(ctx, node); err != nil {
		return nil, err
	}

	if s.routeSyncer != nil {
		_ = s.routeSyncer.SyncRoutes(ctx)
	}

	return node, nil
}

// DeleteNode removes a node and stops its supervised celld container.
func (s *NodeService) DeleteNode(ctx context.Context, id string) error {
	node, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if node.IsProtected || node.Name == "worker-node-01" {
		return domain.NewForbiddenError("main node is protected and cannot be deleted")
	}

	if s.supervisor != nil {
		_ = s.supervisor.StopCelld(ctx, node)
	}

	err = s.repo.Delete(ctx, id)
	if err == nil && s.routeSyncer != nil {
		_ = s.routeSyncer.SyncRoutes(ctx)
	}
	return err
}
