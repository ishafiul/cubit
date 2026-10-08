# Repository Planning Graph (RPG) PRD: Celld v0.6.2 Upgrade & Compatibility Integration

**Feature Slug**: `celld-v062-upgrade`  
**Status**: `Ready for Review`  
**Target Release**: `v1.2.2`  
**Tracking Issue**: [Map #123](https://github.com/ishafiul/cubit/issues/123)  

---

<overview>
## Problem Statement
Deno Land released `celld` v0.6.2 with critical engine resilience fixes, developer capabilities, and protocol enhancements:
1. **Durable Object stub.fetch() Error Propagation**: Forwarded `stub.fetch()` calls now reject cleanly when the handler fails on the owner (mirroring local invocation), without retrying or swallowing errors.
2. **WebSocket Upgrade Enforcement**: Workers returning WebSocket responses without `Upgrade: websocket` headers now fail with a `TypeError` and HTTP 500, enforcing Cloudflare standards.
3. **DO Plain Classes & WebSockets**: `getDurableObjectClass()` accepts plain classes without requiring extension of a runtime base class, and DO facets support inbound and outbound WebSockets.
4. **Isolate Globals & AbortSignal**: `self` is defined globally across Worker, DO, and Loader isolates; loaded workers honor caller `AbortSignal` (including `AbortSignal.timeout()`).
5. **LTX Crash Recovery & Epoch GC**: Torn WAL headers are recovered on database open; `CELLD_LTX_RETENTION_SECS` epoch GC now actively runs on nodes configured with bucket durability.
6. **Fleet Follower Recruitment & Disk Bounding**: Single-follower fleets recruit secondary followers automatically when peers become available, and follower disk use stays bounded after snapshot evictions.

Without upgrading, Cubit defaults to `celld` v0.6.1, lacks plain class DO and facet WebSocket documentation, misses crash recovery on torn WALs, and operator guidance doesn't reflect v0.6.2 rolling updates.

## Target Users & Personas
- **Edge Developers**: Developers building Workers and Durable Objects utilizing plain classes, WebSockets, `self` globals, and `AbortSignal` timeouts.
- **Fleet Operators**: DevOps engineers operating self-hosted Cubit clusters managing node upgrades, storage retention, and multi-node replication.

## Success Metrics
- **Runtime Default**: Newly created fleet nodes and Docker supervisors default to `ghcr.io/denoland/celld:0.6.2`.
- **Zero-Downtime Rolling Update**: Operators can perform rolling upgrades from v0.6.1 to v0.6.2 without dropped traffic.
- **Operator Guidance**: `UpgradeModal` defaults target version to `0.6.2` and explains bucket durability epoch GC and rolling updates.
- **Developer Workbench Parity**: `DynamicWorkersView`, `DurableObjectsView`, `NewAppModal`, `D1View`, and header badges reflect v0.6.2 capabilities.
- **Zero Regressions**: 100% passing tests across Go backend suites (`go test ./...`) and frontend test suites (`npm test`).

## Source Requirements & Decisions
- **Requirements Map**: [#123](https://github.com/ishafiul/cubit/issues/123)
- **Decisions**:
  - Rolling updates remain fully supported between v0.6.1 and v0.6.2.
  - Plain DO classes without runtime base inheritance are officially supported.
  - DO facets support inbound and outbound WebSockets.
  - `UpgradeModal` defaults to `0.6.2` with bucket durability epoch GC guidance.
</overview>

---

<functional-decomposition>
## Capability Tree (The WHAT)

```
Celld v0.6.2 Upgrade
├── Capability 1: Runtime Engine Core Upgrade to 0.6.2
│   ├── Feature 1.1: Default Celld Version Bump to 0.6.2
│   └── Feature 1.2: Docker Celld Supervisor Image & Compose Config
├── Capability 2: Durable Object & Worker Isolate Parity
│   ├── Feature 2.1: Plain Class Durable Objects Guidance & Snippets
│   ├── Feature 2.2: DO Facet WebSockets & Forwarded Error Rejection Guidance
│   └── Feature 2.3: Worker Isolate Globals (self) & AbortSignal Timeout Support
├── Capability 3: Fleet Operator & Upgrade Experience
│   ├── Feature 3.1: UpgradeModal v0.6.2 Rolling Upgrade from v0.6.1
│   └── Feature 3.2: Bucket Durability Epoch GC Guidance (CELLD_LTX_RETENTION_SECS)
└── Capability 4: Frontend Workbench & Compatibility Linters
    ├── Feature 4.1: UI Version Badges & Service Snippets to 0.6.2
    └── Feature 4.2: Compatibility Linter & Worker Template Updates
```

### Capability 1: Runtime Engine Core Upgrade to 0.6.2
Updates core domain constants, supervisor defaults, and deployment configurations to target `celld` v0.6.2.

#### Feature 1.1: Default Celld Version Bump to 0.6.2
- **Description**: Sets the default runtime version across node domain models and runtime handlers to 0.6.2.
- **Inputs**: Node initialization and runtime service creation.
- **Outputs**: `node.CelldVersion = "0.6.2"`, `domain.DefaultCelldVersion = "0.6.2"`.
- **Behavior**: Any new node registered in the cluster or runtime status queried defaults to `0.6.2`.

#### Feature 1.2: Docker Celld Supervisor Image & Compose Config
- **Description**: Updates docker compose and supervisor container image tags to `ghcr.io/denoland/celld:0.6.2`.
- **Inputs**: `deploy/docker-compose.yml`, supervisor image tagging.
- **Outputs**: Docker containers boot `celld:0.6.2`.
- **Behavior**: Local development stack runs `celld` v0.6.2.

### Capability 2: Durable Object & Worker Isolate Parity
Reflects newly added isolate and actor capabilities across documentation, code snippets, and guidance.

#### Feature 2.1: Plain Class Durable Objects Guidance & Snippets
- **Description**: Highlights that `getDurableObjectClass()` supports plain ES6 classes without extending a base class.
- **Inputs**: `DurableObjectsView` component render.
- **Outputs**: Developer guidance panel and class definitions note plain class support.
- **Behavior**: Informs users they no longer need `extends DurableObject`.

#### Feature 2.2: DO Facet WebSockets & Forwarded Error Rejection Guidance
- **Description**: Documents DO facet WebSocket support (including outbound sockets) and remote error propagation.
- **Inputs**: `DurableObjectsView` RPC facets tab.
- **Outputs**: Updated guidance panel noting WebSockets in facets and forwarded `stub.fetch()` error rejection.
- **Behavior**: Users understand that forwarded actor calls fail fast and reject on callers.

#### Feature 2.3: Worker Isolate Globals & AbortSignal Support
- **Description**: Documents `self` global availability and caller `AbortSignal` timeout propagation in dynamic workers.
- **Inputs**: `DynamicWorkersView` runtime info and templates.
- **Outputs**: Developer guidance showing `self.crypto` / `self.location` and `AbortSignal.timeout(ms)` compatibility.
- **Behavior**: Users leverage standard Web APIs in worker code.

### Capability 3: Fleet Operator & Upgrade Experience
Enhances operator upgrade interfaces with v0.6.2 rolling update guidance.

#### Feature 3.1: UpgradeModal v0.6.2 Rolling Upgrade from v0.6.1
- **Description**: Sets `UpgradeModal` default target version to `0.6.2` and informs operators of rolling update support.
- **Inputs**: Operator opens UpgradeModal in Web UI.
- **Outputs**: Target version field defaults to `0.6.2`.
- **Behavior**: Submits `{ targetVersion: "0.6.2" }` to runtime upgrade endpoint.

#### Feature 3.2: Bucket Durability Epoch GC Guidance
- **Description**: Informs operators that `CELLD_LTX_RETENTION_SECS` actively runs on nodes configured with bucket durability.
- **Inputs**: `UpgradeModal` informational cards.
- **Outputs**: Guidance card detailing automatic epoch GC with S3/Garage storage.
- **Behavior**: Operators are assured that storage bloat is controlled automatically in v0.6.2.

### Capability 4: Frontend Workbench & Compatibility Linters
Synchronizes version strings, templates, and compatibility linters across the frontend.

#### Feature 4.1: UI Version Badges & Service Snippets to 0.6.2
- **Description**: Updates header version badge, D1 query header comment, Dynamic Workers runtime badge, and NewAppModal snippets to `0.6.2`.
- **Inputs**: User interface renders.
- **Outputs**: Displays `celld v0.6.2` across badges and templates.
- **Behavior**: Consistent version branding throughout the app.

#### Feature 4.2: Compatibility Linter & Worker Template Updates
- **Description**: Updates compatibility linter and Python worker templates to reference `celld v0.6.2`.
- **Inputs**: Code linting in Application Workbench.
- **Outputs**: Linter findings indicate support for celld v0.6.2.
- **Behavior**: Code analysis correctly identifies supported features in celld v0.6.2.
</functional-decomposition>

---

<structural-decomposition>
## Repository Structure & Modules (The HOW)

```
cubit/
├── internal/
│   ├── domain/
│   │   ├── node.go                  # DefaultCelldVersion = "0.6.2"
│   │   ├── node_test.go             # Verifies 0.6.2
│   │   └── application.go           # Template comments & constants updated to 0.6.2
│   └── modules/
│       └── runtime/
│           ├── service_test.go      # Runtime service tests updated to 0.6.2
│           └── handler_test.go      # Runtime handler tests updated to 0.6.2
├── deploy/
│   └── docker-compose.yml           # celld 0.6.2 image and env configs
└── web/
    └── src/
        ├── App.tsx                  # Header celld v0.6.2 badge
        ├── components/
        │   ├── modals/
        │   │   ├── NewAppModal.tsx  # Python worker template v0.6.2 & badges
        │   │   ├── NewAppModal.test.tsx # Verifies v0.6.2 template
        │   │   ├── UpgradeModal.tsx # Target version 0.6.2, rolling update & epoch GC notes
        │   │   └── UpgradeModal.test.tsx # Verifies 0.6.2 target and guidance
        │   └── services/
        │       ├── D1View.tsx       # D1 query header 0.6.2
        │       ├── DurableObjectsView.tsx # DO plain classes, WebSockets, forwarded errors
        │       └── DynamicWorkersView.tsx # celld 0.6.2 runtime badge & AbortSignal/self snippets
        └── features/
            └── applications/
                └── utils/
                    ├── compatibility-linter.ts      # Python worker & celld v0.6.2 references
                    └── compatibility-linter.test.ts # Tests for v0.6.2 linter rules
```
</structural-decomposition>

---

<dependency-graph>
## Explicit Topological Dependency Chain

### Foundation Layer (Phase 0)
- **internal/domain/node.go**: Update `DefaultCelldVersion = "0.6.2"`. No dependencies.
- **deploy/docker-compose.yml**: Update celld docker image to `0.6.2`. No dependencies.

### Core Domain & Tests Layer (Phase 1)
- **internal/domain/node_test.go**: Depends on Phase 0.
- **internal/modules/runtime/*_test.go**: Depends on Phase 0.

### Presentation & Frontend Layer (Phase 2)
- **web/src/components/modals/UpgradeModal.tsx**: Depends on Phase 0.
- **web/src/components/services/DurableObjectsView.tsx**: Depends on Phase 0.
- **web/src/components/services/DynamicWorkersView.tsx**: Depends on Phase 0.
- **web/src/components/modals/NewAppModal.tsx**: Depends on Phase 0.
- **web/src/features/applications/utils/compatibility-linter.ts**: Depends on Phase 0.
- **web/src/App.tsx**: Depends on Phase 0.
- **Frontend test suites**: Verifies Phase 2 components.
</dependency-graph>

---

<implementation-roadmap>
## Phased Development Roadmap

### Phase 0: Runtime Engine Core Upgrade (Task 1)
- **Goal**: Upgrade default runtime engine version constants and Docker compose to `0.6.2`.
- **Entry Criteria**: Clean working tree on main.
- **Tasks**:
  - [ ] Update `internal/domain/node.go` (`DefaultCelldVersion = "0.6.2"`).
  - [ ] Update `deploy/docker-compose.yml` (`image: ghcr.io/denoland/celld:0.6.2`).
  - [ ] Update Go domain and runtime test assertions to expect `0.6.2`.
- **Exit Criteria**: `go test ./...` passes with 100% green tests.

### Phase 1: Frontend Modals & Upgrade Experience (Task 2)
- **Goal**: Update `UpgradeModal` and `NewAppModal` to default to `0.6.2` with rolling upgrade and epoch GC guidance.
- **Entry Criteria**: Phase 0 complete.
- **Tasks**:
  - [ ] Update `UpgradeModal.tsx` target version to `0.6.2`, add rolling update notes from v0.6.1, and detail epoch GC on bucket durability.
  - [ ] Update `UpgradeModal.test.tsx` to assert `0.6.2` target version and guidance.
  - [ ] Update `NewAppModal.tsx` and `NewAppModal.test.tsx` to reflect `v0.6.2`.
- **Exit Criteria**: Modal Vitest tests pass.

### Phase 2: Frontend Services Views & Compatibility Linters (Task 3)
- **Goal**: Update `DynamicWorkersView`, `DurableObjectsView`, `D1View`, `App.tsx`, and compatibility linters to reflect `0.6.2` capabilities.
- **Entry Criteria**: Phase 1 complete.
- **Tasks**:
  - [ ] Update `DurableObjectsView.tsx` with plain class DO support and facet WebSocket guidance.
  - [ ] Update `DynamicWorkersView.tsx` with runtime badge `0.6.2`, `self` global, and `AbortSignal.timeout()` snippet.
  - [ ] Update `D1View.tsx` and `App.tsx` badges.
  - [ ] Update `compatibility-linter.ts` and `compatibility-linter.test.ts` for celld `v0.6.2`.
- **Exit Criteria**: All frontend tests pass (`npm test`).
</implementation-roadmap>

---

<test-strategy>
## Test Strategy & Critical Scenarios

### Test Pyramid
- **Unit Tests (Go)**: Verify `node.CelldVersion` defaults to `0.6.2` and runtime service reports `0.6.2`.
- **Component Tests (Vitest)**: Verify `UpgradeModal`, `NewAppModal`, `compatibility-linter`, and views render `0.6.2` with correct guidance.
- **End-to-End Simulation**: Rolling upgrade mutation sends `{ targetVersion: "0.6.2" }`.
</test-strategy>

---

<architecture>
## System Architecture & ADRs
- **ADR 1: Rolling Update from v0.6.1 to v0.6.2**:
  - Upstream celld explicitly verifies that upgrading from v0.6.1 to v0.6.2 can use rolling updates.
  - Cubit preserves zero dropped requests across cluster nodes during upgrade.
- **ADR 2: Plain Class Durable Objects**:
  - In celld v0.6.2, classes bound to Durable Objects no longer require extending a runtime base class.
  - Cubit guides users to write idiomatic, unconstrained JavaScript/TypeScript actor classes.
</architecture>

---

<risks>
## Risk Analysis & Mitigations
- **Risk**: Hardcoded `0.6.1` strings causing test failures in Vitest.
  - **Mitigation**: Grep and update all version test assertions synchronously.
- **Risk**: Incompatible node state during rolling updates.
  - **Mitigation**: Upstream celld v0.6.2 explicitly maintains wire and storage compatibility with v0.6.1 nodes.
</risks>
