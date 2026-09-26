# Repository Planning Graph (RPG) PRD: Celld v0.6.0 Upgrade & Compatibility Integration

**Feature Slug**: `celld-v060-upgrade`  
**Status**: `Draft / Ready for Review`  
**Target Release**: `v1.2.0`  
**Tracking Issue**: [Map #65](https://github.com/ishaf/cubit/issues/65)  

---

<overview>
## Problem Statement
Deno Land released `celld` v0.6.0 featuring stricter Cloudflare `workerd` parity, expanded cryptographic primitive support, isolated per-facet SQLite replication streams for Durable Objects, and specific upgrade constraints between durability modes.
Currently, Cubit's backend defaults, Docker container supervisor, manifest generation, pre-flight compatibility linter, and web frontend interfaces are pegged to `celld` v0.5.1. Without upgrading:
1. Newly deployed workers lack guarantee of `compatibilityDate`, which is now strictly enforced by celld v0.6.0 `WorkerCode`.
2. Developers utilizing modern Web Crypto (`Ed25519`, `X25519`) are improperly flagged as unsupported or unverified by Cubit's linters.
3. Operators upgrading fleet daemons need explicit reassurance regarding rolling update safety under Garage S3 bucket durability vs fleet durability.

## Target Users & Personas
- **Bare-Metal Fleet Operators**: DevOps engineers running self-hosted Cubit clusters who need zero-downtime rolling upgrades to celld v0.6.0 across worker nodes.
- **Cloudflare Edge Developers**: Migrators writing Workers and Durable Objects requiring modern Web Crypto (Ed25519) and strict workerd execution parity.

## Success Metrics
- **Runtime Default**: 100% of newly registered fleet nodes and Docker supervisors default to `ghcr.io/denoland/celld:0.6.0`.
- **Manifest Parity**: 100% of generated celld manifests guarantee a valid `compatibility_date` (defaulting to `2024-09-23` if omitted).
- **Linter Accuracy**: Dual linter engines (Go backend `validator.go` and TypeScript frontend `compatibility-linter.ts`) report `crypto.subtle` Ed25519/X25519 as fully supported and warn on unpinned compatibility dates.
- **Zero-Regression Verification**: 100% passing rate across all Go backend tests and Vitest frontend suites.

## Source Requirements & Decisions
- **Requirements Map**: [#65](https://github.com/ishaf/cubit/issues/65)
- **Resolved Grilling Issues**:
  - [#66](https://github.com/ishaf/cubit/issues/66): Missing `compatibility_date` auto-defaults to `2024-09-23` in manifests with a linter warning.
  - [#67](https://github.com/ishaf/cubit/issues/67): Fleet upgrade defaults to rolling zero-downtime update using Garage S3 bucket durability, surfacing clear status in the UI.
- **Constraints Carried Forward**: Backwards compatibility with existing applications, SQLite state schemas, and Garage S3 distributed bucket storage.
</overview>

---

<functional-decomposition>
## Capability Tree (The WHAT)

```
Celld v0.6.0 Upgrade
├── Capability 1: Runtime Engine Core Upgrade
│   ├── Feature 1.1: Default Celld Version Bump to 0.6.0
│   └── Feature 1.2: Docker Celld Supervisor Image Tagging
├── Capability 2: Manifest & Workerd Parity Enforcement
│   ├── Feature 2.1: Mandatory Compatibility Date Generation
│   └── Feature 2.2: Root Route RunWorkerFirst Assets Support
├── Capability 3: Compatibility Linter & Diagnostics
│   ├── Feature 3.1: Ed25519 & X25519 Web Crypto Detection
│   └── Feature 3.2: Compatibility Date Warning Diagnostics
└── Capability 4: Frontend UI & Operator Experience
    ├── Feature 4.1: UI Version Label & Dynamic Worker Template Updates
    └── Feature 4.2: Bucket Durability Rolling Upgrade Guidance
```

### Capability 1: Runtime Engine Core Upgrade
Updates the core domain defaults, Docker supervisor image tags, and test assertions to target `celld` v0.6.0.

#### Feature 1.1: Default Celld Version Bump to 0.6.0
- **Description**: Update `DefaultCelldVersion` constant in domain model and ensure node registration defaults to v0.6.0.
- **Inputs**: Node registration payload or control plane startup configuration.
- **Outputs**: Node instances initialized with `CelldVersion = "0.6.0"`.
- **Behavior**: Replace `0.5.1` with `0.6.0` in `internal/domain/node.go` and mock registrations.

#### Feature 1.2: Docker Celld Supervisor Image Tagging
- **Description**: Ensure container supervisor provisions `ghcr.io/denoland/celld:0.6.0` containers.
- **Inputs**: Node launch or rolling restart commands.
- **Outputs**: Running container instances with tag `0.6.0`.
- **Behavior**: Verify image formatting and restart arguments pass the new version tag.

### Capability 2: Manifest & Workerd Parity Enforcement
Ensures deployment manifests adhere to celld v0.6.0 workerd parity requirements.

#### Feature 2.1: Mandatory Compatibility Date Generation
- **Description**: Guarantee that `CelldManifest` contains a valid `compatibility_date` fallback (`2024-09-23`) if not explicitly set.
- **Inputs**: Application entity and deployment specifications.
- **Outputs**: `CelldManifest` JSON payload with populated `compatibility_date`.
- **Behavior**: In `BuildCelldManifest`, ensure `rawMetadata["compatibility_date"]` is non-empty.

#### Feature 2.2: Root Route RunWorkerFirst Assets Support
- **Description**: Allow static asset deployments to declare `run_worker_first: true` covering root `/` routes.
- **Inputs**: Wrangler asset configurations with root route handling.
- **Outputs**: Valid `CelldAssetConfig` with `RunWorkerFirst: true`.
- **Behavior**: Propagate `run_worker_first` flag cleanly to celld manifest.

### Capability 3: Compatibility Linter & Diagnostics
Updates dual-engine linters (Go and TypeScript) to reflect celld v0.6.0 capabilities and warnings.

#### Feature 3.1: Ed25519 & X25519 Web Crypto Detection
- **Description**: Recognize `Ed25519` and `X25519` subtle crypto usages as natively supported by celld v0.6.0.
- **Inputs**: Source code files containing `crypto.subtle` algorithms.
- **Outputs**: Feature findings with `status: supported` and details on celld v0.6.0 support.
- **Behavior**: Scan for `Ed25519`, `NODE-ED25519`, and `X25519` key generation / import / export.

#### Feature 3.2: Compatibility Date Warning Diagnostics
- **Description**: Emit a non-blocking warning when `compatibility_date` is omitted from `wrangler.jsonc` or `wrangler.toml`.
- **Inputs**: Configuration files lacking `compatibility_date`.
- **Outputs**: Feature finding with `status: warning` recommending explicit date pinning.
- **Behavior**: Check `cfg.CompatibilityDate` and append warning diagnostic.

### Capability 4: Frontend UI & Operator Experience
Reflects celld v0.6.0 across developer and operator interfaces.

#### Feature 4.1: UI Version Label & Dynamic Worker Template Updates
- **Description**: Update header versions, D1 query templates, and Dynamic Worker execution snippets to celld v0.6.0.
- **Inputs**: Frontend view rendering.
- **Outputs**: Display labels showing `v0.6.0`.
- **Behavior**: Update `App.tsx`, `D1View.tsx`, and `DynamicWorkersView.tsx`.

#### Feature 4.2: Bucket Durability Rolling Upgrade Guidance
- **Description**: Update `UpgradeModal.tsx` copy confirming Garage S3 bucket durability enables safe rolling upgrades.
- **Inputs**: Operator opening the daemon rolling upgrade modal.
- **Outputs**: Informational badge and guidance describing zero-downtime rolling upgrade guarantees.
- **Behavior**: Clarify that bucket durability enables rolling node restarts without stopping the whole fleet.
</functional-decomposition>

---

<structural-decomposition>
## Repository Structure & Modules (The HOW)

```text
cubit/
├── internal/
│   ├── domain/
│   │   └── node.go                  # DefaultCelldVersion = "0.6.0"
│   ├── modules/
│   │   ├── deployment/
│   │   │   └── celld_manifest.go    # Compatibility date fallback & manifest metadata
│   │   ├── runtime/
│   │   │   └── service.go           # Runtime fleet upgrade orchestrator
│   │   └── wrangler/
│   │       └── validator.go         # Go compatibility linter rules (crypto, compat date)
│   └── adapters/out/docker/
│       └── celld_supervisor.go      # Docker container lifecycle
└── web/
    └── src/
        ├── App.tsx                  # Version badge fallback
        ├── components/
        │   ├── modals/
        │   │   └── UpgradeModal.tsx # Bucket durability guidance
        │   └── services/
        │       ├── D1View.tsx       # SQL template celld 0.6.0
        │       └── DynamicWorkersView.tsx # Runtime snippet celld 0.6.0
        └── features/applications/utils/
            └── compatibility-linter.ts # TypeScript compatibility linter rules
```
</structural-decomposition>

---

<dependency-graph>
## Explicit Topological Dependency Chain

### Foundation Layer (Phase 0)
- **Domain Constants**: `internal/domain/node.go` (`DefaultCelldVersion = "0.6.0"`). No dependencies.

### Core Implementation Layer (Phase 1)
- **Manifest Engine**: `internal/modules/deployment/celld_manifest.go`. Depends on Domain Constants.
- **Backend Linter**: `internal/modules/wrangler/validator.go`. Depends on Domain Constants.
- **Docker Supervisor**: `internal/adapters/out/docker/celld_supervisor.go`. Depends on Domain Constants.

### Frontend Presentation Layer (Phase 2)
- **Frontend Linter**: `web/src/features/applications/utils/compatibility-linter.ts`. Independent TypeScript engine.
- **UI Views & Modals**: `web/src/App.tsx`, `UpgradeModal.tsx`, `DynamicWorkersView.tsx`, `D1View.tsx`.
</dependency-graph>

---

<verification-and-testing>
## Test Strategy & Quality Gates
- **Go Tests**:
  - `internal/domain/node_test.go`: Verify default version is 0.6.0.
  - `internal/modules/deployment/celld_manifest_test.go`: Verify compatibility_date fallback in manifests.
  - `internal/modules/wrangler/validator_test.go`: Test crypto detection (Ed25519) and missing date warning.
  - `internal/adapters/out/docker/celld_supervisor_test.go`: Verify 0.6.0 image tag handling.
- **Vitest Suites**:
  - `web/src/features/applications/utils/compatibility-linter.test.ts`: Verify Ed25519/X25519 detection and warning on missing date.
  - `web/src/components/services/services.test.ts`: Updated suite names and expectations.
- **Race Detector**:
  - `go test -race ./...`
</verification-and-testing>
