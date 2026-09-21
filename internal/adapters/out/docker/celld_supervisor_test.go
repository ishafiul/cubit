package docker_test

import (
	"context"
	"testing"

	"github.com/ishaf/cubit/internal/adapters/out/docker"
	"github.com/ishaf/cubit/internal/domain"
)

func TestCelldSupervisor(t *testing.T) {
	ctx := context.Background()
	supervisor := docker.NewCelldSupervisor("ghcr.io/denoland/celld")
	node, _ := domain.NewNode("n-1", "node-1", "10.0.0.1", 8081, 8080, "0.2.0")

	t.Run("Given a node and a supervisor", func(t *testing.T) {
		t.Run("When starting celld then container is registered", func(t *testing.T) {
			if err := supervisor.StartCelld(ctx, node, "0.2.0", "s3://fleet-bucket"); err != nil {
				t.Fatalf("expected start to succeed: %v", err)
			}
		})

		t.Run("When issuing graceful restart then version is updated cleanly", func(t *testing.T) {
			if err := supervisor.GracefulRestartCelld(ctx, node, "0.3.0", "s3://fleet-bucket"); err != nil {
				t.Fatalf("expected restart to succeed: %v", err)
			}
		})

		t.Run("When stopping celld then container is removed", func(t *testing.T) {
			if err := supervisor.StopCelld(ctx, node); err != nil {
				t.Fatalf("expected stop to succeed: %v", err)
			}
		})
	})
}
