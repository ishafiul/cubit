# Repository Planning Graph (RPG) PRD: Celld v0.6.1 Upgrade & Compatibility Integration

**Feature Slug**: `celld-v061-upgrade`  
**Status**: `Ready for Review`  
**Target Release**: `v1.2.1`  
**Tracking Issue**: [Map #82](https://github.com/ishafiul/cubit/issues/82)  

---

<overview>
## Problem Statement
Deno Land released `celld` v0.6.1 featuring:
1. **Python Workers**: Native support for Python workers with `async def fetch(request, env)`.
2. **Bucket Usage & LTX Epoch Retention (`CELLD_LTX_RETENTION_SECS`)**: Prevents S3 bucket storage bloat by automatically deleting superseded epochs that no restore reads, accompanied by `celld cell gc --dry-run`.
3. **Configurable Asset & Dynamic Worker Limits**: `CELLD_MAX_ASSET_FILE_BYTES` (default 25 MiB) and `CELLD_MAX_DYNAMIC_WORKER_CODE_BYTES` (default 64 MiB).
4. **OpenTelemetry Log Severity in Console Records**: Console logs now carry standard OpenTelemetry severity (`DEBUG`, `INFO`, `WARN`, `ERROR`) in OTLP exports and structured data.
5. **Durable Object Facets Startup Properties & Named IDs**: `ctx.id` keeps the name of named IDs, and `ctx.exports.App({ props })` passes startup properties.

Without upgrading, Cubit users cannot run Python workers, bucket usage grows with every activation without epoch GC guidance, and log levels lack OpenTelemetry standard severity mappings.

## Target Users & Personas
- **Edge Developers**: Developers authoring Python or JavaScript/TypeScript Workers and stateful Durable Objects.
- **Bare-Metal Fleet Operators**: DevOps engineers running self-hosted Cubit clusters managing node upgrades, storage retention, and memory/code limits.

## Success Metrics
- **Runtime Default**: Newly registered fleet nodes and Docker supervisors default to `ghcr.io/denoland/celld:0.6.1`.
- **Python Worker Parity**: Developers can create, preview, edit, and invoke Python workers with the standard signature.
- **OTel Severity Accuracy**: 100% of isolate console log records carry structured OTel severity tags (`DEBUG`, `INFO`, `WARN`, `ERROR`) visible in Live Tail.
- **Operator Guidance**: UpgradeModal and deployment configs provide clear guidance on `CELLD_LTX_RETENTION_SECS` and epoch GC.
- **Zero Regressions**: 100% passing rate across all Go backend tests and Vitest frontend suites.

## Source Requirements & Decisions
- **Requirements Map**: [#82](https://github.com/ishafiul/cubit/issues/82)
- **Decisions**:
  - Rolling updates remain fully supported between v0.6.0 and v0.6.1.
  - Python worker template provides standard `async def fetch(request, env)` with `Response`.
  - Console methods map to standard OTel severities: `debug` -> `DEBUG`, `log`/`info` -> `INFO`, `warn` -> `WARN`, `error` -> `ERROR`.
</overview>

---

<functional-decomposition>
## Capability Tree (The WHAT)

```
Celld v0.6.1 Upgrade
├── Capability 1: Runtime Engine Core Upgrade to 0.6.1
│   ├── Feature 1.1: Default Celld Version Bump to 0.6.1
│   └── Feature 1.2: Docker Celld Supervisor Image & Compose Config
├── Capability 2: Python Worker Support
│   ├── Feature 2.1: Python Worker Template in NewAppModal & DynamicWorkersView
│   └── Feature 2.2: Celld Manifest & Runner Python Execution Handling
├── Capability 3: OTel Log Severity & Configurable Limits
│   ├── Feature 3.1: OpenTelemetry Log Severity in Console Output & LiveTailView
│   └── Feature 3.2: 25 MiB Asset & 64 MiB Dynamic Worker Code Limits
└── Capability 4: Frontend UI, Durable Objects & Upgrade Experience
    ├── Feature 4.1: UI Version Badges & Snippet Updates to 0.6.1
    ├── Feature 4.2: Durable Object Facet Startup Properties & Named IDs
    └── Feature 4.3: Epoch Retention (CELLD_LTX_RETENTION_SECS) & GC Guidance
```

### Capability 1: Runtime Engine Core Upgrade to 0.6.1
Updates core domain constants, supervisor defaults, and deployment configurations to target `celld` v0.6.1.

#### Feature 1.1: Default Celld Version Bump to 0.6.1
- **Inputs**: Node entity initialization.
- **Outputs**: `node.CelldVersion = "0.6.1"`, `domain.DefaultCelldVersion = "0.6.1"`.
- **Verification**: `internal/domain/node_test.go` and runtime handler tests verify default version `0.6.1`.

#### Feature 1.2: Docker Celld Supervisor Image & Compose Config
- **Inputs**: `deploy/docker-compose.yml`, supervisor image tagging.
- **Outputs**: Documented environment variables for `CELLD_LTX_RETENTION_SECS`, `CELLD_MAX_ASSET_FILE_BYTES`, and `CELLD_MAX_DYNAMIC_WORKER_CODE_BYTES`.

### Capability 2: Python Worker Support
Introduces native Python worker templates, code editing, and execution handling.

#### Feature 2.1: Python Worker Templates
- **Inputs**: User creates a new application or selects template.
- **Outputs**: Python template:
  ```python
  from js import Response

  async def fetch(request, env):
      return Response.new("Hello from Python Worker on Cubit celld v0.6.1!")
  ```
- **Verification**: `NewAppModal.tsx` template selection populates Python worker entrypoint.

#### Feature 2.2: Celld Manifest & Runner Python Handling
- **Inputs**: Worker code written in Python (`sourceType === 'inline'`, `.py` entrypoint).
- **Outputs**: Runtime execution recognized as Python worker isolate.

### Capability 3: OTel Log Severity & Configurable Limits
Propagates OpenTelemetry log severity levels in console records and respects new size limits.

#### Feature 3.1: OpenTelemetry Log Severity
- **Inputs**: Console output from isolate executions (`console.debug`, `console.info`, `console.warn`, `console.error`).
- **Outputs**: Log entries tagged with `DEBUG`, `INFO`, `WARN`, `ERROR` OTel severity pills in Live Tail and edge inspector.
- **Verification**: `LiveTailView` displays color-coded severity badges matching OTel specifications.

#### Feature 3.2: Configurable Limits
- **Inputs**: Static assets uploads up to 25 MiB, dynamic worker code up to 64 MiB.
- **Outputs**: StaticAssetsView and DynamicWorkersView display 25 MiB asset and 64 MiB module limits.

### Capability 4: Frontend UI, Durable Objects & Upgrade Experience
Reflects v0.6.1 across developer and operator interfaces.

#### Feature 4.1: UI Version Badges to 0.6.1
- **Inputs**: Header runtime badge, D1 query template header, Dynamic Worker runtime badges.
- **Outputs**: Displays `celld v0.6.1`.

#### Feature 4.2: Durable Object Facet Startup Properties & Named IDs
- **Inputs**: DO instantiation request with optional `props` startup parameters.
- **Outputs**: Facets pass startup properties and preserve named IDs.

#### Feature 4.3: Epoch Retention Guidance
- **Inputs**: UpgradeModal and fleet storage views.
- **Outputs**: Guidance explaining `CELLD_LTX_RETENTION_SECS` epoch GC to reclaim bucket storage without breaking restore capabilities.
</functional-decomposition>

---

<structural-decomposition>
## Physical Architecture (The HOW)

```
cubit/
├── internal/
│   ├── domain/
│   │   ├── node.go                  # DefaultCelldVersion = "0.6.1"
│   │   └── node_test.go             # Verifies 0.6.1
│   ├── modules/
│   │   ├── runtime/                 # Runtime tests updated to 0.6.1
│   │   └── deployment/              # Manifest and edge log OTel severity mapping
├── deploy/
│   └── docker-compose.yml           # celld 0.6.1 env variables documented
└── web/
    └── src/
        ├── App.tsx                  # Header celld v0.6.1 badge
        ├── components/
        │   ├── modals/
        │   │   ├── NewAppModal.tsx  # Python worker template
        │   │   └── UpgradeModal.tsx # 0.6.1 upgrade & epoch GC guidance
        │   └── services/
        │       ├── D1View.tsx       # D1 query header 0.6.1
        │       └── DynamicWorkersView.tsx # celld 0.6.1 runtime snippet
        └── features/
            └── applications/
                └── components/
                    └── tabs/
                        └── LiveTailView.tsx # OTel severity badges
```
</structural-decomposition>
