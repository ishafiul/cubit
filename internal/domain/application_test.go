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

	t.Run("Given inline application parameters without git repo", func(t *testing.T) {
		id := "app-inline-1"
		name := "hello-world-worker"

		t.Run("When creating with blank code then it initializes with default Hello World template", func(t *testing.T) {
			app, err := domain.NewApplicationWithSource(id, name, domain.SourceTypeInline, "", "", "", nil, nil)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if app.SourceType != domain.SourceTypeInline {
				t.Errorf("expected SourceTypeInline, got %s", app.SourceType)
			}
			if app.InlineCode != domain.DefaultHelloWorldWorker {
				t.Errorf("expected DefaultHelloWorldWorker, got %s", app.InlineCode)
			}
			if app.GitRepo != "" {
				t.Errorf("expected empty git repo, got %s", app.GitRepo)
			}
		})

		t.Run("When creating with custom worker code then it stores the custom code", func(t *testing.T) {
			customCode := `export default { fetch: () => new Response("custom") };`
			app, err := domain.NewApplicationWithSource(id, name, domain.SourceTypeInline, "", "", customCode, nil, nil)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if app.InlineCode != customCode {
				t.Errorf("expected custom code, got %s", app.InlineCode)
			}
		})
	})

	t.Run("Given git application parameters without git repo", func(t *testing.T) {
		t.Run("When creating a git application then it returns validation error", func(t *testing.T) {
			_, err := domain.NewApplicationWithSource("app-2", "git-worker", domain.SourceTypeGit, "", "main", "", nil, nil)

			if err == nil {
				t.Fatal("expected validation error for missing git repo, got nil")
			}
		})
	})

	t.Run("Given an existing inline application", func(t *testing.T) {
		app, err := domain.NewApplicationWithSource("app-3", "editable-worker", domain.SourceTypeInline, "", "", "", nil, nil)
		if err != nil {
			t.Fatalf("setup failed: %v", err)
		}

		t.Run("When updating inline code then it updates successfully", func(t *testing.T) {
			newCode := `export default { fetch: () => new Response("updated") };`
			err := app.UpdateInlineCode(newCode)

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if app.InlineCode != newCode {
				t.Errorf("expected updated code, got %s", app.InlineCode)
			}
		})

		t.Run("When updating with empty code then it returns validation error", func(t *testing.T) {
			err := app.UpdateInlineCode("   ")

			if err == nil {
				t.Fatal("expected validation error for empty inline code, got nil")
			}
		})
	})
}
