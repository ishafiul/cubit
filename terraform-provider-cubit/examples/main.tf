terraform {
  required_providers {
    cubit = {
      source  = "ishaf/cubit"
      version = "~> 1.0.0"
    }
  }
}

provider "cubit" {
  endpoint = "http://localhost:8000"
}

# Register a bare-metal edge server
resource "cubit_node" "edge_primary" {
  name       = "edge-node-01"
  ip_address = "192.168.1.100"
  cpu_cores  = 8
  memory_mb  = 16384
}

# Provision a Cloudflare Worker application
resource "cubit_application" "my_worker" {
  name     = "auth-service"
  git_repo = "https://github.com/myorg/auth-worker.git"
}

# Bind a custom domain with automated Let's Encrypt TLS
resource "cubit_domain" "auth_api" {
  domain_name    = "auth.example.com"
  application_id = cubit_application.my_worker.id
  path_prefix    = "/"
}
