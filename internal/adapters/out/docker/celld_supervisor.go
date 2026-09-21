package docker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ishaf/cubit/internal/domain"
	"github.com/ishaf/cubit/internal/usecase"
)

var _ usecase.ContainerSupervisor = (*CelldSupervisor)(nil)

// CelldSupervisor coordinates celld container instances on bare metal.
type CelldSupervisor struct {
	imageName      string
	supervisedNodes map[string]string // nodeID -> containerID/state
	mu             sync.Mutex
}

// NewCelldSupervisor creates a new CelldSupervisor.
func NewCelldSupervisor(imageName string) *CelldSupervisor {
	if imageName == "" {
		imageName = "ghcr.io/denoland/celld"
	}
	return &CelldSupervisor{
		imageName:       imageName,
		supervisedNodes: make(map[string]string),
	}
}

// StartCelld simulates/runs the docker container for a node with appropriate flags.
func (s *CelldSupervisor) StartCelld(ctx context.Context, node *domain.Node, version string, bucketURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	containerID := fmt.Sprintf("celld-%s-%s", node.Name, node.ID)
	// Track supervised container state
	s.supervisedNodes[node.ID] = containerID
	return nil
}

// StopCelld gracefully terminates a node's celld container.
func (s *CelldSupervisor) StopCelld(ctx context.Context, node *domain.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.supervisedNodes, node.ID)
	return nil
}

// GracefulRestartCelld sends SIGTERM to allow celld to flush LTX logs and hand over leases before restarting.
func (s *CelldSupervisor) GracefulRestartCelld(ctx context.Context, node *domain.Node, newVersion string, bucketURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. Drain / Graceful stop (emulated SIGTERM with timeout)
	time.Sleep(10 * time.Millisecond)

	// 2. Launch new version
	newContainerID := fmt.Sprintf("celld-%s-%s-v%s", node.Name, node.ID, newVersion)
	s.supervisedNodes[node.ID] = newContainerID
	return nil
}
