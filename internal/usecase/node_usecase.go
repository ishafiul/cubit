package usecase

import (
	"context"
	"crypto/rand"
	"fmt"

	"github.com/ishaf/cubit/internal/domain"
)

// NodeUsecase manages the fleet nodes lifecycle.
type NodeUsecase struct {
	nodeRepo   NodeRepository
	supervisor ContainerSupervisor
	bucketURL  string
}

// NewNodeUsecase creates an instance of NodeUsecase.
func NewNodeUsecase(nodeRepo NodeRepository, supervisor ContainerSupervisor, bucketURL string) *NodeUsecase {
	return &NodeUsecase{
		nodeRepo:   nodeRepo,
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
func (u *NodeUsecase) RegisterNode(ctx context.Context, name, ipAddress string, internalPort, workerPort int, celldVersion string) (*domain.Node, error) {
	nodeID := generateID()
	node, err := domain.NewNode(nodeID, name, ipAddress, internalPort, workerPort, celldVersion)
	if err != nil {
		return nil, err
	}

	if err := u.nodeRepo.Save(ctx, node); err != nil {
		return nil, err
	}

	if u.supervisor != nil {
		if err := u.supervisor.StartCelld(ctx, node, celldVersion, u.bucketURL); err != nil {
			node.MarkOffline()
			_ = u.nodeRepo.Update(ctx, node)
			return nil, fmt.Errorf("failed to start celld on node: %w", err)
		}
	}

	return node, nil
}

// GetNode retrieves a node by ID.
func (u *NodeUsecase) GetNode(ctx context.Context, id string) (*domain.Node, error) {
	return u.nodeRepo.GetByID(ctx, id)
}

// ListNodes lists all registered nodes.
func (u *NodeUsecase) ListNodes(ctx context.Context) ([]*domain.Node, error) {
	return u.nodeRepo.List(ctx)
}

// DrainNode marks a node as draining to facilitate graceful lease handover.
func (u *NodeUsecase) DrainNode(ctx context.Context, id string) (*domain.Node, error) {
	node, err := u.nodeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if err := node.Drain(); err != nil {
		return nil, err
	}

	if err := u.nodeRepo.Update(ctx, node); err != nil {
		return nil, err
	}

	return node, nil
}

// DeleteNode removes a node and stops its supervised celld container.
func (u *NodeUsecase) DeleteNode(ctx context.Context, id string) error {
	node, err := u.nodeRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if u.supervisor != nil {
		_ = u.supervisor.StopCelld(ctx, node)
	}

	return u.nodeRepo.Delete(ctx, id)
}
