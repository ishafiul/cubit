package domain_test

import (
	"testing"

	"github.com/ishaf/cubit/internal/domain"
)

func TestApplication(t *testing.T) {
	t.Run("Given valid application parameters", func(t *testing.T) {
		id := "app-1"
		name := "my-worker-app"
		repo := "https://github.com/myorg/worker"
		branch := "main"
		envVars := []domain.EnvironmentVariable{
			{Key: "API_KEY", Value: "secret-value", IsSecret: true},
		}

		t.Run("When creating an application then it initializes with created status", func(t *testing.T) {
			app, err := domain.NewApplication(id, name, repo, branch, envVars, nil)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if app.Status != domain.AppStatusCreated {
				t.Errorf("expected status created, got %s", app.Status)
			}
			if app.Branch != "main" {
				t.Errorf("expected branch main, got %s", app.Branch)
			}
		})
	})

	t.Run("Given an application with invalid name format", func(t *testing.T) {
		t.Run("When creating an application with uppercase letters then it returns validation error", func(t *testing.T) {
			_, err := domain.NewApplication("app-1", "Invalid_Name!", "https://github.com/org/repo", "main", nil, nil)

			if err == nil {
				t.Fatal("expected validation error for invalid name, got nil")
			}
		})
	})

	t.Run("Given an existing application", func(t *testing.T) {
		app, err := domain.NewApplication("app-1", "my-app", "https://github.com/org/repo", "main", nil, nil)
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		t.Run("When setting active deployment then it updates status to running", func(t *testing.T) {
			app.SetActiveDeployment("dep-123")

			if app.ActiveDeploymentID != "dep-123" {
				t.Errorf("expected deployment ID dep-123, got %s", app.ActiveDeploymentID)
			}
			if app.Status != domain.AppStatusRunning {
				t.Errorf("expected status running, got %s", app.Status)
			}
		})
	})
}
