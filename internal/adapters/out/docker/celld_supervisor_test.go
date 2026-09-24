package docker_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/ishaf/cubit/internal/adapters/out/docker"
	"github.com/ishaf/cubit/internal/domain"
)

// mockCommandRunner records all executed commands and arguments.
type mockCommandRunner struct {
	mu            sync.Mutex
	calls         [][]string
	errOnCmd      map[string]error
	inspectOutput string
}

func newMockCommandRunner() *mockCommandRunner {
	return &mockCommandRunner{
		errOnCmd:      make(map[string]error),
		inspectOutput: "true\n",
	}
}

func (m *mockCommandRunner) Run(ctx context.Context, name string, args ...string) ([]byte, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	call := append([]string{name}, args...)
	m.calls = append(m.calls, call)

	if len(args) > 0 {
		subCmd := args[0]
		if err, ok := m.errOnCmd[subCmd]; ok {
			return nil, err
		}
		if subCmd == "inspect" {
			return []byte(m.inspectOutput), nil
		}
	}

	return []byte("container-id-12345\n"), nil
}

func (m *mockCommandRunner) Calls() [][]string {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := make([][]string, len(m.calls))
	copy(copied, m.calls)
	return copied
}

func TestBuildRunArgs(t *testing.T) {
	tests := []struct {
		name         string
		opts         docker.CelldContainerOptions
		expectedArgs []string
	}{
		{
			name: "Given complete container options with credentials and network, When BuildRunArgs is invoked, Then returns expected arguments",
			opts: docker.CelldContainerOptions{
				ContainerName:   "celld-worker-1",
				Image:           "ghcr.io/denoland/celld:0.5.1",
				WorkerPort:      8080,
				InternalPort:    8081,
				AdvertiseIP:     "192.168.1.100",
				BucketURL:       "cubit-fleet",
				StorageEndpoint: "http://garage:3900",
				StorageRegion:   "cubit-local",
				AccessKeyID:     "test-key",
				SecretAccessKey: "test-secret",
				Network:         "cubit_net",
			},
			expectedArgs: []string{
				"run", "-d",
				"--name", "celld-worker-1",
				"--restart", "unless-stopped",
				"-p", "8080:8080",
				"-p", "8081:8081",
				"-e", "AWS_ACCESS_KEY_ID=test-key",
				"-e", "AWS_SECRET_ACCESS_KEY=test-secret",
				"--network", "cubit_net",
				"ghcr.io/denoland/celld:0.5.1",
				"--bucket", "cubit-fleet",
				"--endpoint", "http://garage:3900",
				"--region", "cubit-local",
				"--listen", "0.0.0.0:8080",
				"--internal-listen", "0.0.0.0:8081",
				"--advertise", "192.168.1.100:8081",
			},
		},
		{
			name: "Given minimal container options without credentials or network, When BuildRunArgs is invoked, Then excludes credentials and network flags",
			opts: docker.CelldContainerOptions{
				ContainerName:   "celld-node-2",
				Image:           "ghcr.io/denoland/celld:latest",
				WorkerPort:      9000,
				InternalPort:    9001,
				AdvertiseIP:     "127.0.0.1",
				BucketURL:       "s3://my-bucket",
				StorageEndpoint: "http://127.0.0.1:3900",
				StorageRegion:   "garage",
			},
			expectedArgs: []string{
				"run", "-d",
				"--name", "celld-node-2",
				"--restart", "unless-stopped",
				"-p", "9000:9000",
				"-p", "9001:9001",
				"ghcr.io/denoland/celld:latest",
				"--bucket", "s3://my-bucket",
				"--endpoint", "http://127.0.0.1:3900",
				"--region", "garage",
				"--listen", "0.0.0.0:9000",
				"--internal-listen", "0.0.0.0:9001",
				"--advertise", "127.0.0.1:9001",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := docker.BuildRunArgs(tc.opts)
			if !slices.Equal(args, tc.expectedArgs) {
				t.Errorf("BuildRunArgs() mismatch:\nExpected: %v\nGot:      %v", tc.expectedArgs, args)
			}
		})
	}
}

func TestBuildStopArgs(t *testing.T) {
	tests := []struct {
		name          string
		containerName string
		timeout       int
		expectedArgs  []string
	}{
		{
			name:          "Given container name and positive timeout, When BuildStopArgs is invoked, Then returns stop command with specified timeout",
			containerName: "celld-worker-1",
			timeout:       15,
			expectedArgs:  []string{"stop", "--time", "15", "celld-worker-1"},
		},
		{
			name:          "Given container name and zero timeout, When BuildStopArgs is invoked, Then falls back to default timeout",
			containerName: "celld-worker-default",
			timeout:       0,
			expectedArgs:  []string{"stop", "--time", "10", "celld-worker-default"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := docker.BuildStopArgs(tc.containerName, tc.timeout)
			if !slices.Equal(args, tc.expectedArgs) {
				t.Errorf("expected %v, got %v", tc.expectedArgs, args)
			}
		})
	}
}

func TestCelldSupervisor_NodeLifecycle(t *testing.T) {
	ctx := context.Background()
	runner := newMockCommandRunner()
	cfg := docker.CelldSupervisorConfig{
		ImageName:       "ghcr.io/denoland/celld",
		StorageEndpoint: "http://garage:3900",
		StorageRegion:   "cubit-local",
		AccessKeyID:     "access123",
		SecretAccessKey: "secret123",
		Network:         "cubit_net",
		StopTimeoutSec:  10,
		Runner:          runner,
	}
	supervisor := docker.NewCelldSupervisorWithConfig(cfg)
	node, err := domain.NewNode("n-1", "node-1", "10.0.0.1", 8081, 8080, "0.5.1")
	if err != nil {
		t.Fatalf("failed to create node: %v", err)
	}

	t.Run("Given a configured supervisor and a node", func(t *testing.T) {
		t.Run("When starting celld on the node", func(t *testing.T) {
			err := supervisor.StartCelld(ctx, node, "0.5.1", "cubit-fleet")

			t.Run("Then it executes docker run, verifies health, and marks the container supervised", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected start to succeed, got: %v", err)
				}

				calls := runner.Calls()
				if len(calls) < 2 {
					t.Fatalf("expected at least 2 calls (run, inspect), got %d", len(calls))
				}
				runCall := calls[0]
				if runCall[0] != "docker" || runCall[1] != "run" {
					t.Errorf("expected docker run call, got %v", runCall)
				}
				inspectCall := calls[1]
				if inspectCall[0] != "docker" || inspectCall[1] != "inspect" {
					t.Errorf("expected docker inspect call, got %v", inspectCall)
				}

				containerID, ok := supervisor.GetSupervisedContainer(node.ID)
				if !ok || containerID == "" {
					t.Errorf("expected node %s to have supervised container, got %s (ok=%v)", node.ID, containerID, ok)
				}
			})
		})

		t.Run("When performing a graceful restart of celld", func(t *testing.T) {
			err := supervisor.GracefulRestartCelld(ctx, node, "0.5.2", "cubit-fleet")

			t.Run("Then it sends SIGTERM stop to old container and starts new container version", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected graceful restart to succeed, got: %v", err)
				}

				calls := runner.Calls()
				var hasStop, hasRm, hasNewRun bool
				for _, c := range calls {
					cmdStr := strings.Join(c, " ")
					if strings.Contains(cmdStr, "docker stop --time 10") {
						hasStop = true
					}
					if strings.Contains(cmdStr, "docker rm -f") {
						hasRm = true
					}
					if strings.Contains(cmdStr, "docker run") && strings.Contains(cmdStr, "ghcr.io/denoland/celld:0.5.2") {
						hasNewRun = true
					}
				}

				if !hasStop {
					t.Errorf("expected docker stop call in restart, got calls: %v", calls)
				}
				if !hasRm {
					t.Errorf("expected docker rm call in restart, got calls: %v", calls)
				}
				if !hasNewRun {
					t.Errorf("expected docker run with version 0.5.2 in restart, got calls: %v", calls)
				}
			})
		})

		t.Run("When stopping celld on the node", func(t *testing.T) {
			err := supervisor.StopCelld(ctx, node)

			t.Run("Then it issues graceful stop, removes container, and unregisters supervision", func(t *testing.T) {
				if err != nil {
					t.Fatalf("expected stop to succeed, got: %v", err)
				}

				containerID, ok := supervisor.GetSupervisedContainer(node.ID)
				if ok || containerID != "" {
					t.Errorf("expected node to no longer be supervised, got %s", containerID)
				}
			})
		})
	})
}

func TestCelldSupervisor_DirectContainerManagement(t *testing.T) {
	ctx := context.Background()
	runner := newMockCommandRunner()
	cfg := docker.CelldSupervisorConfig{
		ImageName:      "ghcr.io/denoland/celld",
		StopTimeoutSec: 10,
		Runner:         runner,
	}
	supervisor := docker.NewCelldSupervisorWithConfig(cfg)

	t.Run("Given direct container execution request with options", func(t *testing.T) {
		opts := docker.CelldContainerOptions{
			ContainerName:   "celld-app-123",
			WorkerPort:      8080,
			InternalPort:    8081,
			BucketURL:       "cubit-fleet",
			StorageEndpoint: "http://garage:3900",
			StorageRegion:   "cubit-local",
		}

		t.Run("When StartCelldContainer is invoked", func(t *testing.T) {
			containerName, err := supervisor.StartCelldContainer(ctx, opts)

			t.Run("Then it starts the container and returns valid name", func(t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error starting container: %v", err)
				}
				if containerName != "celld-app-123" {
					t.Fatalf("expected container name 'celld-app-123', got %s", containerName)
				}
			})
		})

		t.Run("When StopCelldContainer is invoked", func(t *testing.T) {
			err := supervisor.StopCelldContainer(ctx, "celld-app-123")

			t.Run("Then it issues stop and remove commands", func(t *testing.T) {
				if err != nil {
					t.Fatalf("unexpected error stopping container: %v", err)
				}

				calls := runner.Calls()
				var foundStop, foundRm bool
				for _, c := range calls {
					cmdStr := strings.Join(c, " ")
					if strings.Contains(cmdStr, "docker stop --time 10 celld-app-123") {
						foundStop = true
					}
					if strings.Contains(cmdStr, "docker rm -f celld-app-123") {
						foundRm = true
					}
				}
				if !foundStop || !foundRm {
					t.Errorf("expected both stop and rm calls for container, got: %v", calls)
				}
			})
		})
	})
}

func TestCelldSupervisor_HealthAndErrorHandling(t *testing.T) {
	ctx := context.Background()

	t.Run("Given container that fails health verification", func(t *testing.T) {
		runner := newMockCommandRunner()
		runner.inspectOutput = "false\n"
		supervisor := docker.NewCelldSupervisorWithConfig(docker.CelldSupervisorConfig{
			Runner: runner,
		})
		node, _ := domain.NewNode("n-dead", "node-dead", "127.0.0.1", 9091, 9090, "0.5.1")

		t.Run("When StartCelld is called", func(t *testing.T) {
			err := supervisor.StartCelld(ctx, node, "0.5.1", "cubit-fleet")

			t.Run("Then it returns error indicating container is not running", func(t *testing.T) {
				if err == nil {
					t.Fatal("expected health check error, got nil")
				}
				if !strings.Contains(err.Error(), "started but is not running") {
					t.Errorf("expected health error, got: %v", err)
				}
			})
		})
	})

	t.Run("Given docker run failure", func(t *testing.T) {
		runner := newMockCommandRunner()
		runner.errOnCmd["run"] = errors.New("docker daemon connection refused")
		supervisor := docker.NewCelldSupervisorWithConfig(docker.CelldSupervisorConfig{
			Runner: runner,
		})
		node, _ := domain.NewNode("n-err", "node-err", "127.0.0.1", 9091, 9090, "0.5.1")

		t.Run("When StartCelld is called", func(t *testing.T) {
			err := supervisor.StartCelld(ctx, node, "0.5.1", "cubit-fleet")

			t.Run("Then it returns wrapped error and node is not marked supervised", func(t *testing.T) {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				if !strings.Contains(err.Error(), "docker daemon connection refused") {
					t.Errorf("expected error to wrap runner failure, got: %v", err)
				}
				if _, ok := supervisor.GetSupervisedContainer(node.ID); ok {
					t.Error("expected node to not be supervised after failure")
				}
			})
		})
	})

	t.Run("Given stop failure during graceful restart", func(t *testing.T) {
		runner := newMockCommandRunner()
		runner.errOnCmd["stop"] = errors.New("cannot kill frozen container")
		supervisor := docker.NewCelldSupervisorWithConfig(docker.CelldSupervisorConfig{
			Runner: runner,
		})
		node, _ := domain.NewNode("n-restart-err", "node-restart-err", "127.0.0.1", 9091, 9090, "0.5.1")

		t.Run("When GracefulRestartCelld is called", func(t *testing.T) {
			err := supervisor.GracefulRestartCelld(ctx, node, "0.5.2", "cubit-fleet")

			t.Run("Then it does not swallow the error and wraps it properly", func(t *testing.T) {
				if err == nil {
					t.Fatal("expected error during restart stop failure, got nil")
				}
				if !strings.Contains(err.Error(), "failed to stop existing container") {
					t.Errorf("expected wrapped stop failure error, got: %v", err)
				}
			})
		})
	})
}
