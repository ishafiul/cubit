package usecase

import (
	"context"
	"fmt"
	"sync"

	"github.com/ishaf/cubit/internal/domain"
)

// RuntimeStatus contains fleet-wide runtime state and celld version information.
type RuntimeStatus struct {
	ActiveNodesCount   int
	CurrentCelldVersion string
	TargetCelldVersion  string
	IsUpgrading         bool
	StorageBackend      string
}

// RuntimeUsecase orchestrates fleet health and zero-downtime rolling upgrades of the celld daemon.
type RuntimeUsecase struct {
	nodeRepo       NodeRepository
	supervisor     ContainerSupervisor
	storage        StoragePort
	bucketURL      string
	currentVersion string
	targetVersion  string
	isUpgrading    bool
	mu             sync.RWMutex
}

// NewRuntimeUsecase creates a new RuntimeUsecase instance.
func NewRuntimeUsecase(
	nodeRepo NodeRepository,
	supervisor ContainerSupervisor,
	storage StoragePort,
	bucketURL string,
	currentVersion string,
) *RuntimeUsecase {
	return &RuntimeUsecase{
		nodeRepo:       nodeRepo,
		supervisor:     supervisor,
		storage:        storage,
		bucketURL:      bucketURL,
		currentVersion: currentVersion,
	}
}

// GetStatus returns the current fleet runtime status.
func (u *RuntimeUsecase) GetStatus(ctx context.Context) (*RuntimeStatus, error) {
	u.mu.RLock()
	defer u.mu.RUnlock()

	nodes, err := u.nodeRepo.List(ctx)
	if err != nil {
		return nil, err
	}

	activeCount := 0
	for _, n := range nodes {
		if n.Status == domain.NodeStatusActive {
			activeCount++
		}
	}

	storageBackend := "unknown"
	if u.storage != nil {
		storageBackend = u.storage.DriverName()
	}

	return &RuntimeStatus{
		ActiveNodesCount:   activeCount,
		CurrentCelldVersion: u.currentVersion,
		TargetCelldVersion:  u.targetVersion,
		IsUpgrading:         u.isUpgrading,
		StorageBackend:      storageBackend,
	}, nil
}

// UpgradeCelldDaemon executes a rolling zero-downtime upgrade of the celld daemon across nodes.
func (u *RuntimeUsecase) UpgradeCelldDaemon(ctx context.Context, targetVersion string) error {
	u.mu.Lock()
	if u.isUpgrading {
		u.mu.Unlock()
		return domain.NewConflictError("an upgrade is already in progress")
	}
	u.isUpgrading = true
	u.targetVersion = targetVersion
	u.mu.Unlock()

	defer func() {
		u.mu.Lock()
		u.isUpgrading = false
		u.currentVersion = targetVersion
		u.targetVersion = ""
		u.mu.Unlock()
	}()

	nodes, err := u.nodeRepo.List(ctx)
	if err != nil {
		return err
	}

	// Rolling restart: one node at a time
	for _, node := range nodes {
		if node.Status != domain.NodeStatusActive {
			continue
		}

		if u.supervisor != nil {
			if err := u.supervisor.GracefulRestartCelld(ctx, node, targetVersion, u.bucketURL); err != nil {
				return fmt.Errorf("failed upgrading node %s (%s): %w", node.Name, node.ID, err)
			}
		}

		node.UpdateCelldVersion(targetVersion)
		if err := u.nodeRepo.Update(ctx, node); err != nil {
			return err
		}
	}

	return nil
}
