package domain_test

import (
	"testing"

	"github.com/ishaf/cubit/internal/domain"
)

func TestDeployment(t *testing.T) {
	t.Run("Given valid deployment parameters", func(t *testing.T) {
		id := "dep-1"
		appID := "app-1"
		commitHash := "abcdef1"
		commitMsg := "feat: Initial worker"

		t.Run("When creating a deployment then it initializes in pending status", func(t *testing.T) {
			dep, err := domain.NewDeployment(id, appID, commitHash, commitMsg)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if dep.Status != domain.DeploymentStatusPending {
				t.Errorf("expected status pending, got %s", dep.Status)
			}
		})
	})

	t.Run("Given a pending deployment", func(t *testing.T) {
		dep, err := domain.NewDeployment("dep-1", "app-1", "abc", "msg")
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		t.Run("When progressing through successful build and deploy lifecycle", func(t *testing.T) {
			t.Run("Then building state succeeds", func(t *testing.T) {
				if err := dep.StartBuilding(); err != nil {
					t.Fatalf("unexpected error starting build: %v", err)
				}
				if dep.Status != domain.DeploymentStatusBuilding {
					t.Errorf("expected building, got %s", dep.Status)
				}
			})

			t.Run("Then deploying state succeeds with bundle size", func(t *testing.T) {
				if err := dep.StartDeploying(102400); err != nil {
					t.Fatalf("unexpected error starting deploy: %v", err)
				}
				if dep.Status != domain.DeploymentStatusDeploying {
					t.Errorf("expected deploying, got %s", dep.Status)
				}
				if dep.BundleSize != 102400 {
					t.Errorf("expected bundle size 102400, got %d", dep.BundleSize)
				}
			})

			t.Run("Then active state succeeds and records finished timestamp", func(t *testing.T) {
				if err := dep.MarkActive(); err != nil {
					t.Fatalf("unexpected error marking active: %v", err)
				}
				if dep.Status != domain.DeploymentStatusActive {
					t.Errorf("expected active, got %s", dep.Status)
				}
				if dep.FinishedAt == nil {
					t.Error("expected finishedAt timestamp to be set")
				}
			})
		})
	})

	t.Run("Given a pending deployment", func(t *testing.T) {
		dep, _ := domain.NewDeployment("dep-1", "app-1", "abc", "msg")

		t.Run("When build fails then deployment records error message and failed status", func(t *testing.T) {
			dep.MarkFailed("esbuild syntax error")

			if dep.Status != domain.DeploymentStatusFailed {
				t.Errorf("expected failed status, got %s", dep.Status)
			}
			if dep.ErrorMessage != "esbuild syntax error" {
				t.Errorf("expected error message preserved, got %s", dep.ErrorMessage)
			}
		})
	})
}
