package deploy_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

type composeConfig struct {
	Services map[string]struct {
		Image         string            `yaml:"image"`
		ContainerName string            `yaml:"container_name"`
		Ports         []string          `yaml:"ports"`
		Environment   []string          `yaml:"environment"`
		Volumes       []string          `yaml:"volumes"`
		DependsOn     []string          `yaml:"depends_on"`
		Networks      []string          `yaml:"networks"`
	} `yaml:"services"`
	Networks map[string]interface{} `yaml:"networks"`
	Volumes  map[string]interface{} `yaml:"volumes"`
}

func TestDeploymentPackaging(t *testing.T) {
	t.Run("Given the production deployment artifacts", func(t *testing.T) {
		repoRoot := ".."

		t.Run("When validating the docker-compose configuration", func(t *testing.T) {
			composePath := filepath.Join(repoRoot, "deploy", "docker-compose.yml")
			data, err := os.ReadFile(composePath)
			if err != nil {
				t.Fatalf("failed to read docker-compose.yml: %v", err)
			}

			var cfg composeConfig
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				t.Fatalf("failed to parse docker-compose.yml: %v", err)
			}

			t.Run("Then all required services are defined", func(t *testing.T) {
				required := []string{"cubitd", "traefik", "garage", "celld"}
				for _, svc := range required {
					if _, ok := cfg.Services[svc]; !ok {
						t.Errorf("expected service %q to be present in compose file", svc)
					}
				}
			})

			t.Run("Then required persistent storage volumes are declared", func(t *testing.T) {
				requiredVols := []string{"cubit_data", "traefik_dynamic", "garage_data"}
				for _, vol := range requiredVols {
					if _, ok := cfg.Volumes[vol]; !ok {
						t.Errorf("expected volume %q in compose file", vol)
					}
				}
			})

			t.Run("Then cubitd exposes port 8000 and mounts docker socket", func(t *testing.T) {
				cubitd := cfg.Services["cubitd"]
				hasPort8000 := false
				for _, p := range cubitd.Ports {
					if strings.Contains(p, "8000") {
						hasPort8000 = true
						break
					}
				}
				if !hasPort8000 {
					t.Errorf("expected cubitd to expose port 8000, got %v", cubitd.Ports)
				}

				hasDockerSock := false
				for _, v := range cubitd.Volumes {
					if strings.Contains(v, "docker.sock") {
						hasDockerSock = true
						break
					}
				}
				if !hasDockerSock {
					t.Errorf("expected cubitd to mount /var/run/docker.sock, got %v", cubitd.Volumes)
				}
			})
		})

		t.Run("When inspecting the Traefik configuration", func(t *testing.T) {
			traefikPath := filepath.Join(repoRoot, "deploy", "traefik.yaml")
			content, err := os.ReadFile(traefikPath)
			if err != nil {
				t.Fatalf("failed to read traefik.yaml: %v", err)
			}
			raw := string(content)

			t.Run("Then dynamic file provider is enabled", func(t *testing.T) {
				if !strings.Contains(raw, "/etc/traefik/dynamic") {
					t.Errorf("expected traefik.yaml to watch /etc/traefik/dynamic")
				}
			})

			t.Run("Then web and websecure entrypoints are configured", func(t *testing.T) {
				if !strings.Contains(raw, ":80") || !strings.Contains(raw, ":443") {
					t.Errorf("expected entrypoints for port 80 and 443")
				}
			})
		})

		t.Run("When inspecting the Garage S3 configuration", func(t *testing.T) {
			garagePath := filepath.Join(repoRoot, "deploy", "garage.toml")
			content, err := os.ReadFile(garagePath)
			if err != nil {
				t.Fatalf("failed to read garage.toml: %v", err)
			}
			raw := string(content)

			t.Run("Then S3 API is bound on port 3900", func(t *testing.T) {
				if !strings.Contains(raw, "3900") {
					t.Errorf("expected garage.toml to configure S3 API port 3900")
				}
			})

			t.Run("Then RPC is bound on port 3901", func(t *testing.T) {
				if !strings.Contains(raw, "3901") {
					t.Errorf("expected garage.toml to configure RPC port 3901")
				}
			})
		})

		t.Run("When inspecting the installation bootstrapping script", func(t *testing.T) {
			scriptPath := filepath.Join(repoRoot, "scripts", "install.sh")
			content, err := os.ReadFile(scriptPath)
			if err != nil {
				t.Fatalf("failed to read install.sh: %v", err)
			}
			raw := string(content)

			t.Run("Then it sets bash strict mode and checks root execution", func(t *testing.T) {
				if !strings.Contains(raw, "set -euo pipefail") {
					t.Errorf("expected script to set -euo pipefail")
				}
				if !strings.Contains(raw, "EUID") {
					t.Errorf("expected script to check EUID")
				}
			})

			t.Run("Then it manages the /opt/cubit directory and docker compose", func(t *testing.T) {
				if !strings.Contains(raw, "/opt/cubit") {
					t.Errorf("expected script to target /opt/cubit")
				}
				if !strings.Contains(raw, "docker compose") {
					t.Errorf("expected script to invoke docker compose")
				}
			})
		})
	})
}
