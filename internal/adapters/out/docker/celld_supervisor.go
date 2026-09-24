package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/ishaf/cubit/internal/domain"
)

const (
	// DefaultImageName is the official celld image repository.
	DefaultImageName = "ghcr.io/denoland/celld"
	// DefaultStorageEndpoint is the default local Garage S3 listener.
	DefaultStorageEndpoint = "http://garage:3900"
	// DefaultStorageRegion is the default local Garage S3 region.
	DefaultStorageRegion = "cubit-local"
	// DefaultStopTimeoutSec is the default seconds to wait for SIGTERM LTX flush.
	DefaultStopTimeoutSec = 10
)

var containerNameSanitizer = regexp.MustCompile(`[^a-zA-Z0-9_.-]+`)

// CommandRunner abstracts system command execution for docker CLI commands.
type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) ([]byte, error)
}

type execCommandRunner struct{}

func (e *execCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

// CelldContainerOptions defines the parameters required to run a celld container instance.
type CelldContainerOptions struct {
	ContainerName   string
	Image           string
	WorkerPort      int
	InternalPort    int
	AdvertiseIP     string
	BucketURL       string
	StorageEndpoint string
	StorageRegion   string
	AccessKeyID     string
	SecretAccessKey string
	Network         string
}

// BuildRunArgs generates the CLI argument slice for 'docker run' to launch a celld daemon.
func BuildRunArgs(opts CelldContainerOptions) []string {
	args := []string{
		"run", "-d",
		"--name", opts.ContainerName,
		"--restart", "unless-stopped",
		"-p", fmt.Sprintf("%d:%d", opts.WorkerPort, opts.WorkerPort),
		"-p", fmt.Sprintf("%d:%d", opts.InternalPort, opts.InternalPort),
	}

	if opts.AccessKeyID != "" {
		args = append(args, "-e", fmt.Sprintf("AWS_ACCESS_KEY_ID=%s", opts.AccessKeyID))
	}
	if opts.SecretAccessKey != "" {
		args = append(args, "-e", fmt.Sprintf("AWS_SECRET_ACCESS_KEY=%s", opts.SecretAccessKey))
	}
	if opts.Network != "" {
		args = append(args, "--network", opts.Network)
	}

	args = append(args,
		opts.Image,
		"--bucket", opts.BucketURL,
		"--endpoint", opts.StorageEndpoint,
		"--region", opts.StorageRegion,
		"--listen", fmt.Sprintf("0.0.0.0:%d", opts.WorkerPort),
		"--internal-listen", fmt.Sprintf("0.0.0.0:%d", opts.InternalPort),
		"--advertise", fmt.Sprintf("%s:%d", opts.AdvertiseIP, opts.InternalPort),
	)

	return args
}

// BuildStopArgs generates the CLI argument slice for 'docker stop' with timeout.
func BuildStopArgs(containerName string, timeoutSeconds int) []string {
	if timeoutSeconds <= 0 {
		timeoutSeconds = DefaultStopTimeoutSec
	}
	return []string{"stop", "--time", strconv.Itoa(timeoutSeconds), containerName}
}

// CelldSupervisorConfig provides configuration options for CelldSupervisor.
type CelldSupervisorConfig struct {
	ImageName       string
	StorageEndpoint string
	StorageRegion   string
	AccessKeyID     string
	SecretAccessKey string
	Network         string
	StopTimeoutSec  int
	Runner          CommandRunner
}

// CelldSupervisor coordinates live celld container instances across fleet nodes.
type CelldSupervisor struct {
	cfg             CelldSupervisorConfig
	supervisedNodes map[string]string // nodeID -> containerName
	mu              sync.RWMutex
}

// NewCelldSupervisor creates a new CelldSupervisor with default environment-based configuration.
func NewCelldSupervisor(imageName string) *CelldSupervisor {
	return NewCelldSupervisorWithConfig(CelldSupervisorConfig{
		ImageName: imageName,
	})
}

// NewCelldSupervisorWithConfig creates a new CelldSupervisor with customized configuration.
func NewCelldSupervisorWithConfig(cfg CelldSupervisorConfig) *CelldSupervisor {
	if cfg.ImageName == "" {
		cfg.ImageName = DefaultImageName
	}
	if cfg.StorageEndpoint == "" {
		if env := os.Getenv("CELLD_STORAGE_ENDPOINT"); env != "" {
			cfg.StorageEndpoint = env
		} else {
			cfg.StorageEndpoint = DefaultStorageEndpoint
		}
	}
	if cfg.StorageRegion == "" {
		if env := os.Getenv("CELLD_STORAGE_REGION"); env != "" {
			cfg.StorageRegion = env
		} else {
			cfg.StorageRegion = DefaultStorageRegion
		}
	}
	if cfg.AccessKeyID == "" {
		cfg.AccessKeyID = os.Getenv("AWS_ACCESS_KEY_ID")
	}
	if cfg.SecretAccessKey == "" {
		cfg.SecretAccessKey = os.Getenv("AWS_SECRET_ACCESS_KEY")
	}
	if cfg.Network == "" {
		cfg.Network = os.Getenv("CELLD_DOCKER_NETWORK")
	}
	if cfg.StopTimeoutSec <= 0 {
		cfg.StopTimeoutSec = DefaultStopTimeoutSec
	}
	if cfg.Runner == nil {
		cfg.Runner = &execCommandRunner{}
	}

	return &CelldSupervisor{
		cfg:             cfg,
		supervisedNodes: make(map[string]string),
	}
}

func sanitizeContainerName(name string) string {
	cleaned := containerNameSanitizer.ReplaceAllString(name, "-")
	return strings.Trim(cleaned, "-")
}

func (s *CelldSupervisor) containerNameForNode(node *domain.Node) string {
	return sanitizeContainerName(fmt.Sprintf("celld-%s-%s", node.Name, node.ID))
}

func (s *CelldSupervisor) resolveImage(version string) string {
	base := s.cfg.ImageName
	if strings.Contains(base, ":") {
		return base
	}
	if version != "" {
		return fmt.Sprintf("%s:%s", base, version)
	}
	return fmt.Sprintf("%s:latest", base)
}

// IsContainerRunning inspects whether a Docker container is active and running.
func (s *CelldSupervisor) IsContainerRunning(ctx context.Context, containerName string) (bool, error) {
	out, err := s.cfg.Runner.Run(ctx, "docker", "inspect", "-f", "{{.State.Running}}", containerName)
	if err != nil {
		return false, fmt.Errorf("failed to inspect container '%s': %w", containerName, err)
	}
	return strings.TrimSpace(string(out)) == "true", nil
}

func (s *CelldSupervisor) runContainer(ctx context.Context, opts CelldContainerOptions) error {
	args := BuildRunArgs(opts)
	out, err := s.cfg.Runner.Run(ctx, "docker", args...)
	if err != nil {
		return fmt.Errorf("failed to run celld container '%s': %w (output: %s)", opts.ContainerName, err, strings.TrimSpace(string(out)))
	}

	running, err := s.IsContainerRunning(ctx, opts.ContainerName)
	if err != nil {
		return fmt.Errorf("failed to verify celld container '%s' status: %w", opts.ContainerName, err)
	}
	if !running {
		return fmt.Errorf("celld container '%s' started but is not running", opts.ContainerName)
	}

	return nil
}

func (s *CelldSupervisor) stopAndRemoveContainer(ctx context.Context, containerName string) error {
	stopArgs := BuildStopArgs(containerName, s.cfg.StopTimeoutSec)
	out, err := s.cfg.Runner.Run(ctx, "docker", stopArgs...)
	if err != nil {
		return fmt.Errorf("failed to stop celld container '%s': %w (output: %s)", containerName, err, strings.TrimSpace(string(out)))
	}

	rmArgs := []string{"rm", "-f", containerName}
	out, err = s.cfg.Runner.Run(ctx, "docker", rmArgs...)
	if err != nil {
		return fmt.Errorf("failed to remove celld container '%s': %w (output: %s)", containerName, err, strings.TrimSpace(string(out)))
	}

	return nil
}

// StartCelld launches the docker container for a fleet node.
func (s *CelldSupervisor) StartCelld(ctx context.Context, node *domain.Node, version string, bucketURL string) error {
	if node == nil {
		return fmt.Errorf("node cannot be nil")
	}

	containerName := s.containerNameForNode(node)
	image := s.resolveImage(version)

	opts := CelldContainerOptions{
		ContainerName:   containerName,
		Image:           image,
		WorkerPort:      node.WorkerPort,
		InternalPort:    node.InternalPort,
		AdvertiseIP:     node.IPAddress,
		BucketURL:       bucketURL,
		StorageEndpoint: s.cfg.StorageEndpoint,
		StorageRegion:   s.cfg.StorageRegion,
		AccessKeyID:     s.cfg.AccessKeyID,
		SecretAccessKey: s.cfg.SecretAccessKey,
		Network:         s.cfg.Network,
	}

	if err := s.runContainer(ctx, opts); err != nil {
		return err
	}

	s.mu.Lock()
	s.supervisedNodes[node.ID] = containerName
	s.mu.Unlock()

	return nil
}

// StopCelld gracefully terminates a fleet node's celld container via SIGTERM.
func (s *CelldSupervisor) StopCelld(ctx context.Context, node *domain.Node) error {
	if node == nil {
		return fmt.Errorf("node cannot be nil")
	}

	s.mu.Lock()
	containerName, exists := s.supervisedNodes[node.ID]
	if !exists {
		containerName = s.containerNameForNode(node)
	}
	delete(s.supervisedNodes, node.ID)
	s.mu.Unlock()

	return s.stopAndRemoveContainer(ctx, containerName)
}

// GracefulRestartCelld stops existing celld container and launches an updated version without TOCTOU race.
func (s *CelldSupervisor) GracefulRestartCelld(ctx context.Context, node *domain.Node, newVersion string, bucketURL string) error {
	if node == nil {
		return fmt.Errorf("node cannot be nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	oldContainer, exists := s.supervisedNodes[node.ID]
	if !exists {
		oldContainer = s.containerNameForNode(node)
	}

	if err := s.stopAndRemoveContainer(ctx, oldContainer); err != nil {
		return fmt.Errorf("failed to stop existing container '%s' during restart: %w", oldContainer, err)
	}

	containerName := s.containerNameForNode(node)
	image := s.resolveImage(newVersion)

	opts := CelldContainerOptions{
		ContainerName:   containerName,
		Image:           image,
		WorkerPort:      node.WorkerPort,
		InternalPort:    node.InternalPort,
		AdvertiseIP:     node.IPAddress,
		BucketURL:       bucketURL,
		StorageEndpoint: s.cfg.StorageEndpoint,
		StorageRegion:   s.cfg.StorageRegion,
		AccessKeyID:     s.cfg.AccessKeyID,
		SecretAccessKey: s.cfg.SecretAccessKey,
		Network:         s.cfg.Network,
	}

	if err := s.runContainer(ctx, opts); err != nil {
		return err
	}

	s.supervisedNodes[node.ID] = containerName
	return nil
}

// StartCelldContainer runs a celld container instance, satisfying Capability 3 export.
func (s *CelldSupervisor) StartCelldContainer(ctx context.Context, opts CelldContainerOptions) (string, error) {
	if opts.ContainerName == "" {
		return "", fmt.Errorf("container name cannot be empty")
	}
	if opts.Image == "" {
		opts.Image = s.resolveImage("")
	}
	if opts.StorageEndpoint == "" {
		opts.StorageEndpoint = s.cfg.StorageEndpoint
	}
	if opts.StorageRegion == "" {
		opts.StorageRegion = s.cfg.StorageRegion
	}
	if opts.AdvertiseIP == "" {
		opts.AdvertiseIP = "127.0.0.1"
	}
	if opts.AccessKeyID == "" {
		opts.AccessKeyID = s.cfg.AccessKeyID
	}
	if opts.SecretAccessKey == "" {
		opts.SecretAccessKey = s.cfg.SecretAccessKey
	}
	if opts.Network == "" {
		opts.Network = s.cfg.Network
	}

	if err := s.runContainer(ctx, opts); err != nil {
		return "", err
	}

	return opts.ContainerName, nil
}

// StopCelldContainer stops and removes a standalone celld container.
func (s *CelldSupervisor) StopCelldContainer(ctx context.Context, containerID string) error {
	return s.stopAndRemoveContainer(ctx, containerID)
}

// GetSupervisedContainer returns the container name for a supervised node ID.
func (s *CelldSupervisor) GetSupervisedContainer(nodeID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	containerName, exists := s.supervisedNodes[nodeID]
	return containerName, exists
}
