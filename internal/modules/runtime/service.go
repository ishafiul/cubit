package runtime

import (
	"context"
	"fmt"
	"sync"

	"github.com/ishaf/cubit/internal/domain"
)

// NodeRepository defines node access needed by the runtime service.
type NodeRepository interface {
	List(ctx context.Context) ([]*domain.Node, error)
	Update(ctx context.Context, n *domain.Node) error
}

// ContainerSupervisor manages restarting celld on fleet nodes.
type ContainerSupervisor interface {
	GracefulRestartCelld(ctx context.Context, node *domain.Node, newVersion string, bucketURL string) error
}

// StorageDriver provides storage driver name.
type StorageDriver interface {
	DriverName() string
}

// Status contains fleet-wide runtime state and celld version information.
type Status struct {
	ActiveNodesCount    int    `json:"activeNodesCount"`
	CurrentCelldVersion string `json:"currentCelldVersion"`
	TargetCelldVersion  string `json:"targetCelldVersion"`
	IsUpgrading         bool   `json:"isUpgrading"`
	StorageBackend      string `json:"storageBackend"`
}

// Service defines runtime business operations.
type Service interface {
	GetStatus(ctx context.Context) (*Status, error)
	UpgradeCelldDaemon(ctx context.Context, targetVersion string) error
}

// RuntimeService orchestrates fleet health and zero-downtime rolling upgrades of the celld daemon.
type RuntimeService struct {
	nodeRepo       NodeRepository
	supervisor     ContainerSupervisor
	storage        StorageDriver
	bucketURL      string
	currentVersion string
	targetVersion  string
	isUpgrading    bool
	mu             sync.RWMutex
}

// NewService creates a new RuntimeService instance.
func NewService(
	nodeRepo NodeRepository,
	supervisor ContainerSupervisor,
	storage StorageDriver,
	bucketURL string,
	currentVersion string,
) *RuntimeService {
	if currentVersion == "" {
		currentVersion = domain.DefaultCelldVersion
	}
	return &RuntimeService{
		nodeRepo:       nodeRepo,
		supervisor:     supervisor,
		storage:        storage,
		bucketURL:      bucketURL,
		currentVersion: currentVersion,
	}
}

// GetStatus returns the current fleet runtime status.
func (s *RuntimeService) GetStatus(ctx context.Context) (*Status, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	activeCount := 0
	for _, n := range nodes {
		if n.Status == domain.NodeStatusActive {
			activeCount++
		}
	}

	storageBackend := "local"
	if s.storage != nil {
		storageBackend = s.storage.DriverName()
	}

	return &Status{
		ActiveNodesCount:    activeCount,
		CurrentCelldVersion: s.currentVersion,
		TargetCelldVersion:  s.targetVersion,
		IsUpgrading:         s.isUpgrading,
		StorageBackend:      storageBackend,
	}, nil
}

// UpgradeCelldDaemon executes a rolling zero-downtime upgrade of the celld daemon across nodes.
func (s *RuntimeService) UpgradeCelldDaemon(ctx context.Context, targetVersion string) error {
	s.mu.Lock()
	if s.isUpgrading {
		s.mu.Unlock()
		return domain.NewConflictError("an upgrade is already in progress")
	}
	s.isUpgrading = true
	s.targetVersion = targetVersion
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isUpgrading = false
		s.currentVersion = targetVersion
		s.targetVersion = ""
		s.mu.Unlock()
	}()

	nodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return err
	}

	for _, node := range nodes {
		if node.Status != domain.NodeStatusActive {
			continue
		}

		if s.supervisor != nil {
			if err := s.supervisor.GracefulRestartCelld(ctx, node, targetVersion, s.bucketURL); err != nil {
				return fmt.Errorf("failed upgrading node %s (%s): %w", node.Name, node.ID, err)
			}
		}

		node.UpdateCelldVersion(targetVersion)
		if err := s.nodeRepo.Update(ctx, node); err != nil {
			return err
		}
	}

	return nil
}
