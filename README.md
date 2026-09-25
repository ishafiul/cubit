# Cubit ⚡

**Cubit** is an open-source, bare-metal PaaS designed to execute **Cloudflare Workers** and **Durable Objects** on self-hosted infrastructure.

Powered by [`celld`](https://github.com/denoland/celld) (by Deno Land), Cubit gives you sub-millisecond V8 isolate execution, private SQLite database cells per Durable Object, dynamic Traefik ingress with edge header synthesis, and Garage S3 distributed storage—all without vendor lock-in.

---

## 🏛️ Architecture Overview

```
[ Inbound HTTP / WebSocket Traffic ]
                 │
                 ▼
      [ Traefik v3 Ingress ]
      (Dynamic file provider & route sync, SSL/TLS, edge headers)
                 │
                 ├──(Routes by Domain/Host)──► [ celld Daemon (:8080) ]
                 │                                      │
                 │                             (LTX replication & CAS)
                 ▼                                      ▼
[ Cubit Control Plane (cubitd :8000) ]        [ Garage S3 (:3900) ]
  (Go 1.24+, REST API, SQLite WAL)              (Bundles, LTX journals, R2)
                 │
                 ▼
        [ SQLite (cubit.db) ]
```

### Core Components

| Component | Technology | Description |
| :--- | :--- | :--- |
| **Runtime Engine** | [`celld`](https://github.com/denoland/celld) (Deno Land) | High-performance embedded V8 isolates, private SQLite cells per Durable Object instance, LTX transaction logs, and live code hot-reloading. |
| **Control Plane** | `cubitd` (Go 1.24+, Gin) | Central REST API managing worker bundles, deployments, storage, SQLite state in WAL mode, and ingress synchronization. |
| **Ingress & Proxy** | Traefik v3 | Dynamic file provider, Let's Encrypt / ACME automatic TLS, path routing, and edge metadata injection (`CF-*`). |
| **Object Storage** | Garage S3 (Rust) | Lightweight, distributed S3-compatible engine storing worker bundles, LTX transaction journals, and user R2 buckets. |
| **Web Dashboard** | React 19, Vite, Tailwind CSS | Modern workbench interface featuring real-time deployment logs, visual configuration, compatibility diagnostics, and state management. |

---

## ✨ Features Provided with `celld`

### 1. Cloudflare Workers & Durable Objects on Self-Hosted Hardware
- **Sub-Millisecond Cold Starts**: Lightweight V8 isolates created and scheduled in `<5ms`.
- **Private SQLite Cells**: Each Durable Object class and instance receives a private, isolated SQLite database cell backed by LTX replication and WAL consistency.
- **Worker-to-Worker Invocations**: Native `service` bindings enable direct micro-service communication between workers.
- **Durable Workflows & Docker Containers**: Orchestrate durable workflows and attach Docker container sidecars for compute-heavy workloads.

### 2. Full `wrangler` Configuration Compatibility
- **Multi-Format Parsing**: Directly parses `wrangler.json`, `wrangler.jsonc`, and `wrangler.toml` files.
- **Automatic Sanitization**: Prohibited Cloudflare-managed properties (such as `routes` and `account_id`) are stripped safely, while routes are converted into Traefik ingress rules and registered domains.
- **Dynamic Compatibility Date & Flags**: Full support for `compatibility_date` and `compatibility_flags` (e.g. `nodejs_compat`, `streams_enable_constructors`).

### 3. Pre-Flight Compatibility Linter & Diagnostics
- **Automated Scanning**: Scans worker code and `wrangler` configurations before deployment.
- **Supported Feature Detection**: Automatically identifies supported APIs (`request.cf`, `caches.default`, `WebSocketPair`, `ctx.waitUntil`, KV, D1, R2, Durable Objects, containers, assets, crons).
- **Unsupported Feature Remediation**: Detects proprietary Cloudflare APIs (`env.AI`, `env.VECTORIZE`, `env.HYPERDRIVE`, `@cloudflare/puppeteer`, `env.ANALYTICS`) and provides actionable remediation guidance in the UI and CLI.

### 4. Cloudflare State Data Migration Tooling
- **KV Bulk Importer**: Ingest `wrangler kv bulk get` JSON export dumps via drag-and-drop or pasting into `POST /api/v1/kv/namespaces/:id/import`.
- **D1 SQL Dump Importer**: Execute `wrangler d1 export` SQL dumps directly against private SQLite databases via `POST /api/v1/d1/databases/:id/import`.
- **R2 Object Importer**: Ingest object migration manifests to replicate S3 buckets into Garage storage via `POST /api/v1/r2/buckets/:name/import`.

### 5. Edge Context Forwarding & Synthesis
- **Traefik Edge Header Injection**: Ingress injects `CF-Connecting-IP`, `CF-IPCountry`, `CF-IPCity`, `CF-Ray`, and `CF-Visitor` headers.
- **Synthesized `request.cf`**: Isolate bootstrap synthesizes standard Cloudflare edge metadata into `request.cf` inside `worker.fetch(request, env, ctx)`.

### 6. Extended Resource Binding Types
Cubit supports comprehensive resource bindings across control plane and UI:
- `kv_namespace` — Distributed key-value stores.
- `d1_database` — Private SQLite database instances.
- `r2_bucket` — S3-compatible object storage buckets.
- `durable_object` — Stateful isolate classes with embedded SQLite.
- `container` — Dedicated Docker container workloads.
- `assets` — Static asset directories served directly from S3 / cache.
- `workflow` — Durable workflow execution engines.
- `service` — Direct worker-to-worker invocation bindings.
- `queue` — Message queue dispatching.

---

## 🛠️ Prerequisites

Ensure the following tools are installed on your machine:

- **Docker & Docker Compose** (v24+)
- **Go** (v1.24+)
- **Node.js** (v20+) & **npm**
- **Git**

---

## ⚡ Quick Start: One-Line Server Installer

Just like **Dokploy** or **Coolify**, you can install and bootstrap the complete Cubit platform on any bare-metal Linux server or cloud VPS (Ubuntu, Debian, Fedora, Arch, etc.) with a single command:

```bash
curl -fsSL https://raw.githubusercontent.com/ishafiul/cubit/main/scripts/install.sh | sudo bash
```

### What the Installer Does Automatically:
1. **Detects Architecture & OS** (x86_64, arm64).
2. **Provisions Docker Engine & Compose** automatically if missing.
3. **Creates System Layout** at `/opt/cubit` (data, Traefik dynamic configs, TLS certs, Garage S3 storage).
4. **Writes Production Configs** (`traefik.yaml`, `garage.toml`, `docker-compose.yml`).
5. **Starts the Full Fleet**: Launches `cubitd`, `celld`, `traefik`, and `garage`.
6. **Detects Public IP**: Outputs your web workbench URL (`http://<SERVER_IP>:8000`), Traefik dashboard (`:8081`), and S3 storage (`:3900`).

---

## 💻 Local PC / Development Setup

Running Cubit locally on your development machine (macOS / Linux / Windows WSL) takes 3 simple steps:

### Step 1: Start Infrastructure Services (Docker)

Spin up Garage S3, Traefik v3, and the `celld` runtime daemon in the background:

```bash
docker compose -f deploy/docker-compose.yml up -d garage traefik celld
```

Verify that the services are healthy:
```bash
docker compose -f deploy/docker-compose.yml ps
```

* **Garage S3 API**: `http://localhost:3900`
* **celld Daemon**: `http://localhost:8080`
* **Traefik Ingress**: `http://localhost:80` (Dashboard: `http://localhost:8081`)

---

### Step 2: Start the Cubit Control Plane (`cubitd`)

Run the backend daemon locally with SQLite in WAL mode:

```bash
# From repository root
go run cmd/cubitd/main.go
```

The control plane starts on `http://localhost:8000`.

To build a binary instead:
```bash
go build -o bin/cubitd cmd/cubitd/main.go
./bin/cubitd
```

---

### Step 3: Start the Web Dashboard

In a separate terminal, start the React 19 frontend development server:

```bash
cd web
npm install
npm run dev
```

The web dashboard is available at:
👉 **`http://localhost:5173`**

The Vite dev server automatically proxies API requests (`/api/*`) to `cubitd` on port `8000`.

---

## 🧪 Running Tests

### Backend Unit & Integration Tests (Go)

```bash
# Run all Go tests
go test -v ./...

# Run tests with race condition detector
go test -race ./...
```

### Frontend Tests & Typecheck (Vitest & TypeScript)

```bash
cd web

# Run Vitest test suites
npm test

# Run TypeScript typechecker and production build
npm run build
```

---

## 🐳 Running the Complete Production Stack

To run the entire platform (including the compiled control plane container) using Docker Compose:

```bash
docker compose -f deploy/docker-compose.yml up -d
```

To stop all services:
```bash
docker compose -f deploy/docker-compose.yml down
```

---

## ⚙️ Environment Configuration

`cubitd` is configured via environment variables or flags:

| Variable | Default | Description |
| :--- | :--- | :--- |
| `CUBIT_PORT` | `8000` | Port for the control plane REST API. |
| `CUBIT_DB_PATH` | `./cubit.db` | Local SQLite database file path (WAL mode). |
| `CUBIT_TRAEFIK_DYNAMIC_PATH` | `/etc/traefik/dynamic/cubit.yaml` | Output path for dynamic Traefik ingress configuration. |
| `CUBIT_STORAGE_BACKEND` | `garage_s3` | Storage backend (`local` or `garage_s3`). |
| `CUBIT_STORAGE_DIR` | `./storage` | Directory for local file storage. |
| `CUBIT_CELLD_IMAGE` | `ghcr.io/denoland/celld:latest` | Docker image for `celld` worker instances. |
| `GARAGE_S3_ENDPOINT` | `http://localhost:3900` | Garage S3 API endpoint. |
| `CELLD_ENDPOINT` | `http://localhost:8080` | celld daemon API endpoint. |

---

## 📖 API Reference & Workflows

### Deploying a Worker Bundle Directly
Deploy pre-bundled JavaScript or TypeScript workers (from CI/CD or `wrangler deploy`):

```bash
curl -X POST http://localhost:8000/api/v1/applications/deploy \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-worker",
    "bundle": "export default { fetch(req) { return new Response(\"Hello from Cubit!\"); } };",
    "wranglerConfig": "{\"name\":\"my-worker\",\"compatibility_date\":\"2024-09-23\"}"
  }'
```

### Synchronizing Wrangler Configuration
Import and synchronize existing Cloudflare `wrangler.json` or `wrangler.toml`:

```bash
curl -X POST http://localhost:8000/api/v1/applications/<APP_ID>/wrangler/import \
  -H "Content-Type: application/json" \
  -d '{
    "rawConfig": "name = \"my-worker\"\ncompatibility_date = \"2024-09-23\"",
    "format": "auto"
  }'
```

---

## 🤝 Contributing

We welcome contributions! Please see [`AGENTS.md`](./AGENTS.md) for our issue-driven development workflow, hybrid skills architecture, and coding conventions.

---

## 📄 License

Licensed under the [Apache License 2.0](./LICENSE).
