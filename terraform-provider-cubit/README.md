# Terraform Provider for Cubit

The official HashiCorp Terraform Provider for **Cubit** – an open-source, self-hosted bare-metal PaaS for running Cloudflare Workers, Durable Objects, and stateful edge workloads.

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.24 (for building the provider)

## Resources

- `cubit_node`: Register, monitor, and manage bare-metal cluster servers.
- `cubit_application`: Provision worker applications, Git source repositories, and resource bindings.
- `cubit_domain`: Manage custom hostnames, Traefik dynamic routing paths, and Let's Encrypt TLS certificates.

## Example Usage

```hcl
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

resource "cubit_node" "edge_primary" {
  name       = "edge-node-01"
  ip_address = "192.168.1.100"
  cpu_cores  = 8
  memory_mb  = 16384
}

resource "cubit_application" "auth_service" {
  name     = "auth-service"
  git_repo = "https://github.com/org/auth-worker.git"
}

resource "cubit_domain" "auth_api" {
  domain_name    = "auth.example.com"
  application_id = cubit_application.auth_service.id
}
```
