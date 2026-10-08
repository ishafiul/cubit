# PRD: Worker Execution Engine & Runtime Seam Deepening

This Product Requirements Document structures the architectural deepening of Cubit's worker isolate execution engine using the **Repository Planning Graph (RPG)** methodology. It strictly separates **WHAT** (functional execution capabilities) from **HOW** (structural code modules and seams).

---

<overview>
## Problem Statement
- **What concrete pain point exists?**
  `internal/modules/application/service.go` is 1,522 lines and carries over 650 lines of low-level Node subprocess spawning, Cache API in-memory polyfills, WebSocketPair shims, Ray ID generation, and stdout JSON decoding. This leaks child-process mechanics and isolate runtime details into high-level application CRUD and domain logic.
- **Who experiences it?**
  Backend engineers extending application capabilities, and automated test runners. Testing `ApplicationService` currently spawns external Node processes, which fails in sandboxed environments, causes flaky tests, and couples domain tests to the host system's Node installation.
- **Why existing solutions don't work?**
  Embedding execution inside `ApplicationService` violates module depth: callers must learn a massive 20+ method interface, and the module cannot be tested without full database fixtures and external processes.

## Target Users & Personas
- **Platform Maintainer**: Needs clean, deep modules with high locality so bugs in V8 isolate polyfills or process lifecycles concentrate in one place rather than spreading into application management.
- **CI / Automated Test Runner**: Needs fast, deterministic, in-memory execution of application tests without requiring external Node binaries.
- **Fleet Operator**: Needs a clear, unified runtime module that handles both daemon management and worker execution across local nodes and celld isolates.

## Success Metrics
- **Metric 1**: `internal/modules/application/service.go` shrinks by ~650 lines, eliminating all `exec.CommandContext("node", ...)` calls from the application domain package.
- **Metric 2**: 100% test pass rate across backend Go tests (`go test -race ./...`) and frontend test suites (`npm test`).
- **Metric 3**: Zero regressions in application HTTP contracts (`/api/v1/applications/:id/invoke`), CF edge header injection, or live SSE log streaming.

## Source Requirements & Decisions
- **Requirements map**: [#140 [Map] Codebase Deepening: Carve out Worker Execution Engine](https://github.com/ishaf/cubit/issues/140)
- **Resolved grilling issues**:
  - [#141 Package placement and naming for WorkerExecutor](https://github.com/ishaf/cubit/issues/141): Settled on `internal/modules/runtime` to deepen the existing runtime package without package sprawl.
  - [#142 Allocation of telemetry and log streaming](https://github.com/ishaf/cubit/issues/142): Settled on a Clean Result Boundary; `WorkerExecutor` returns `WorkerExecutionResult` while `ApplicationService` handles metrics and `SubscribeLiveLogs` subscriber fan-out.
  - [#143 Test double strategy for WorkerExecutor](https://github.com/ishaf/cubit/issues/143): Settled on a first-class reusable `FakeWorkerExecutor` satisfying the two-adapter seam rule.
- **Constraints carried forward**:
  - Zero breaking changes to public REST APIs or JSON schemas.
  - Preserve exact Cloudflare compatibility shims: `globalThis.caches.default`, `globalThis.WebSocketPair`, `cf` geo metadata, and bindings.
</overview>

---

<functional-decomposition>
## Capability Tree (The WHAT)

### Capability: Worker Isolate Execution
Provides isolated execution of bundled JavaScript/TypeScript workers with simulated Cloudflare runtime environments.

#### Feature: Execution Payload Ingestion
- **Description**: Accepts a compiled worker bundle, HTTP invocation context (method, path, headers, body), environment variables, resource bindings (D1, KV, R2, Vectorize), and execution event type.
- **Inputs**: `ExecutionPayload` containing bundle bytes, HTTP request metadata, environment key-values, and binding definitions.
- **Outputs**: Validated invocation context prepared for runtime isolation.
- **Behavior**: Validates default methods and paths, formats environment variables and bindings into JSON strings, and injects default CF-Ray and CF-Connecting-IP headers if missing.

#### Feature: V8 Compatibility Polyfills
- **Description**: Generates isolation harness scripts supplying Cloudflare Workers standard global APIs inside the isolate.
- **Inputs**: Request headers, environment variables, and resource bindings.
- **Outputs**: Embedded ES module harness containing polyfills.
- **Behavior**: Injects `globalThis.caches.default` (match, put, delete), `globalThis.WebSocketPair`, `CloudflareResponse` (status 101 support), and mock binding proxies for D1, KV, R2, and Vectorize.

#### Feature: Process Lifecycle & Timeout Management
- **Description**: Executes the runner harness under a monitored subprocess with context cancellation and execution timeouts.
- **Inputs**: Execution context and generated harness script.
- **Outputs**: Subprocess stdout/stderr streams.
- **Behavior**: Spawns `node --input-type=module -e <script>`, captures stdout JSON payload, captures stderr upon failure, and respects context timeouts.

#### Feature: Output & Exception Parsing
- **Description**: Parses execution stdout into structured status codes, response headers, response bodies, console logs, and exception stacks.
- **Inputs**: Raw process stdout bytes or stderr error strings.
- **Outputs**: `WorkerExecutionResult` struct.
- **Behavior**: Decodes status, response headers, parsed console logs, and exception lists. In case of non-zero exit or syntax errors, returns a formatted 500 error result with stack traces preserved in exceptions and logs.

### Capability: Application Runtime Seam
Provides a decoupled invocation interface between the Application domain and the Worker Execution engine.

#### Feature: ApplicationService Delegation
- **Description**: Invocations via `ApplicationService.Invoke` delegate execution to `WorkerExecutor` rather than running child processes directly.
- **Inputs**: Application ID, HTTP request metadata.
- **Outputs**: HTTP status, response headers, response body, error.
- **Behavior**: Resolves active deployment bundle and application bindings, calls `executor.Execute(ctx, payload)`, records telemetry, and returns the response.

#### Feature: Telemetry & Live Log Dispatch
- **Description**: Records execution events in `ApplicationService`'s metric tracker and fans out console logs to SSE subscribers.
- **Inputs**: Completed `WorkerExecutionResult`.
- **Outputs**: Updated `ApplicationMetrics` and dispatched `domain.RequestLogEvent`.
- **Behavior**: Iterates over `result.Logs` and dispatches individual log events to `SubscribeLiveLogs` listener channels.

### Capability: In-Memory Test Double
Provides a deterministic, zero-dependency test adapter for testing callers of `WorkerExecutor`.

#### Feature: Reusable FakeWorkerExecutor
- **Description**: An in-memory implementation of `WorkerExecutor` for unit and integration tests.
- **Inputs**: Configurable default response, canned errors, or recorded call expectations.
- **Outputs**: Simulated `WorkerExecutionResult`.
- **Behavior**: Thread-safely records all incoming `ExecutionPayload` objects and returns pre-configured responses without invoking any external binaries.
</functional-decomposition>

---

<structural-decomposition>
## Repository Structure & Modules (The HOW)

### Repository Structure
```text
internal/
├── modules/
│   ├── runtime/
│   │   ├── service.go           # Daemon supervision & celld upgrade (existing)
│   │   ├── service_test.go      # Daemon tests (existing)
│   │   ├── executor.go          # WorkerExecutor interface, ExecutionPayload, NodeWorkerExecutor (NEW)
│   │   ├── executor_test.go     # NodeWorkerExecutor unit tests (NEW)
│   │   ├── fake_executor.go     # Reusable FakeWorkerExecutor for tests (NEW)
│   │   └── fake_executor_test.go# FakeWorkerExecutor unit tests (NEW)
│   └── application/
│       ├── service.go           # Refactored: constructor takes WorkerExecutor, runner code removed
│       └── service_test.go      # Refactored: uses FakeWorkerExecutor
cmd/
└── cubitd/
    └── main.go                  # DI: instantiates NodeWorkerExecutor and passes to application.NewService
```

### Module Definitions

#### Module: `internal/modules/runtime.WorkerExecutor`
- **Maps to capability**: Worker Isolate Execution
- **Responsibility**: Encapsulates worker isolate script generation, V8 polyfills, Node child process lifecycles, and stdout parsing.
- **File Structure**:
  ```text
  internal/modules/runtime/
  ├── executor.go
  ├── executor_test.go
  ├── fake_executor.go
  └── fake_executor_test.go
  ```
- **Public Seam (Interface)**:
  ```go
  type WorkerExecutor interface {
      Execute(ctx context.Context, payload ExecutionPayload) (*WorkerExecutionResult, error)
  }
  ```
- **Data Structures**:
  ```go
  type ExecutionPayload struct {
      Bundle      []byte
      Method      string
      Path        string
      Headers     map[string]string
      Body        []byte
      EnvVars     map[string]string
      Bindings    []domain.ResourceBinding
      BaseURL     string
      EventType   string
      TargetAppID string
  }

  type WorkerExecutionResult struct {
      Status      int
      Headers     map[string]string
      Body        []byte
      Logs        []domain.ConsoleLogEntry
      Exceptions  []string
      CF          map[string]interface{}
      DurationMs  float64
  }
  ```
- **Concrete Adapters**:
  - `NewNodeWorkerExecutor() WorkerExecutor`: Production in-process Node runner adapter.
  - `NewFakeWorkerExecutor() *FakeWorkerExecutor`: Reusable test double adapter with `SetResponse`, `SetError`, and `GetInvocations`.

#### Module: `internal/modules/application.ApplicationService`
- **Maps to capability**: Application Runtime Seam
- **Responsibility**: Manages Application lifecycle, CRUD, storage bundles, route syncing, metrics, and live log streaming.
- **Changes**:
  - `NewService` signature updated to accept `WorkerExecutor`:
    ```go
    func NewService(repo Repository, storage StorageDownloader, routeSyncer RouteSyncer, fleetBucket string, executor runtime.WorkerExecutor) *ApplicationService
    ```
  - Removes lines 810–1465 (`RunWorkerBundleWithEnvAndBindings`, `bindingDTO`, script formatting templates).
  - `Invoke` method calls `s.executor.Execute(ctx, payload)` and feeds results into `s.metrics.record` and `s.RecordExecutionEvent`.
</structural-decomposition>

---

<dependency-graph>
## Explicit Topological Dependency Chain

### Foundation Layer (Phase 0)
No dependencies — foundational contracts, data structures, and errors.
- **`ExecutionPayload` & `WorkerExecutionResult`**: Foundational types in `runtime/executor.go`.
- **`WorkerExecutor` Interface**: Foundational contract in `runtime/executor.go`.

### Core Implementation Layer (Phase 1)
- **`NodeWorkerExecutor`**: Implements `WorkerExecutor`. Depends on Phase 0 contracts.
- **`FakeWorkerExecutor`**: Implements `WorkerExecutor`. Depends on Phase 0 contracts.

### Integration & Wiring Layer (Phase 2)
- **`ApplicationService`**: Depends on `runtime.WorkerExecutor` (Phase 0).
- **`cmd/cubitd/main.go`**: Depends on `runtime.NewNodeWorkerExecutor` (Phase 1) and `application.NewService` (Phase 2).
</dependency-graph>

---

<implementation-roadmap>
## Phased Development Roadmap

### Phase 0: Foundation
- **Goal**: Define `WorkerExecutor` interface, `ExecutionPayload`, and `WorkerExecutionResult` types in `internal/modules/runtime/executor.go`.
- **Entry Criteria**: Working tree clean on branch `refactor/codebase-architecture`.
- **Tasks**:
  - [ ] Implement `WorkerExecutor` interface and payload structs in `internal/modules/runtime/executor.go`.
- **Exit Criteria**: `go build ./internal/modules/runtime/...` succeeds.
- **Delivers**: The authoritative execution seam contract.

### Phase 1: Core Domain Adapters
- **Goal**: Implement `NodeWorkerExecutor` and `FakeWorkerExecutor` with dedicated unit tests.
- **Entry Criteria**: Phase 0 complete.
- **Tasks**:
  - [ ] Implement `NodeWorkerExecutor` in `internal/modules/runtime/executor.go` containing V8 polyfills, script generation, subprocess execution, and stdout JSON parsing.
  - [ ] Implement `FakeWorkerExecutor` in `internal/modules/runtime/fake_executor.go`.
  - [ ] Write unit tests in `internal/modules/runtime/executor_test.go` and `internal/modules/runtime/fake_executor_test.go`.
- **Exit Criteria**: `go test -v ./internal/modules/runtime/...` passes 100%.
- **Delivers**: Both production and test execution adapters.

### Phase 2: Application Seam Refactoring
- **Goal**: Refactor `ApplicationService` to consume `WorkerExecutor` via constructor injection and update `cmd/cubitd/main.go`.
- **Entry Criteria**: Phase 1 complete.
- **Tasks**:
  - [ ] Update `application.NewService` to accept `WorkerExecutor` and delegate `Invoke` calls.
  - [ ] Delete ~650 lines of duplicate subprocess script runner code from `internal/modules/application/service.go`.
  - [ ] Update `cmd/cubitd/main.go` to construct `NodeWorkerExecutor` and pass to `application.NewService`.
  - [ ] Refactor `internal/modules/application/service_test.go` to use `FakeWorkerExecutor`.
  - [ ] Run full test suite (`go test -race ./...` and `npm test`).
- **Exit Criteria**: All Go and web tests pass, zero lint or race warnings.
- **Delivers**: Complete architectural deepening with clean locality and high test leverage.
</implementation-roadmap>

---

<test-strategy>
## Test Strategy & Critical Scenarios

### Test Pyramid
```text
        /\
       /E2E\        ← 10% (End-to-end /api/v1/applications/:id/invoke test)
      /------\
     /Integration\  ← 30% (ApplicationService + FakeWorkerExecutor)
    /------------\
   /  Unit Tests  \ ← 60% (NodeWorkerExecutor isolate polyfill & harness tests)
  /----------------\
```

### Critical Test Scenarios
#### Module: `NodeWorkerExecutor`
- **Happy Path**: Executes basic Hello World worker; returns status 200, headers, and body.
- **Console Log Capture**: Worker calling `console.log`, `console.warn`, `console.error` returns populated `Logs` array with timestamps and log levels.
- **Cache API Polyfill**: Worker calling `caches.default.put` and `caches.default.match` returns cached response correctly.
- **WebSocketPair Polyfill**: Worker creating `new WebSocketPair()` and returning status 101 succeeds with CF status headers.
- **Timeout / Exception Handling**: Infinite loop or syntax error terminates gracefully and returns status 500 with captured exception message.

#### Module: `FakeWorkerExecutor`
- **Invocation Recording**: Calls to `Execute` record payload method, path, and body into an inspectable slice.
- **Configured Response**: Returns pre-set status, headers, and body without running any subprocess.
- **Simulated Errors**: Returns configured error when requested.

#### Module: `ApplicationService` (Integration at Seam)
- **Invoke Delegation**: `Invoke` queries app and bundle, constructs `ExecutionPayload`, calls `executor.Execute`, records metric events, and dispatches log events to SSE channel.
</test-strategy>

---

<architecture>
## System Architecture & ADRs

### Architectural Overview
```text
[ HTTP Caller ]
       │
       ▼
[ ApplicationService ] ──(Execute Payload)──► [ runtime.WorkerExecutor (Seam) ]
       │                                                      │
(Records metrics & SSE logs)                     ┌────────────┴────────────┐
                                                 ▼                         ▼
                                       [ NodeWorkerExecutor ]    [ FakeWorkerExecutor ]
                                                 │                         │
                                      (executes node isolate)     (in-memory test double)
```

### Architectural Decisions (ADRs)
- **ADR-0001: Place WorkerExecutor in `internal/modules/runtime` (#141)**
  - *Rationale*: Deepens the existing `runtime` module into a comprehensive isolate execution and daemon management package, avoiding unnecessary package proliferation.
- **ADR-0002: Clean Result Boundary for Telemetry (#142)**
  - *Rationale*: Keeps `WorkerExecutor` completely stateless and focused purely on code execution, while `ApplicationService` retains ownership of metrics history and subscriber fan-out.
- **ADR-0003: First-Class Reusable FakeWorkerExecutor (#143)**
  - *Rationale*: Satisfies the principle that two adapters justify a real seam, making application domain tests fast, robust, and isolated from host process environments.
</architecture>

---

<risks>
## Risk Analysis & Mitigations
- **Risk: Breaking Invocations in Integration Tests**
  - *Mitigation*: Run the existing comprehensive `internal/modules/application/service_test.go` and `deploy/deploy_test.go` suites against both `FakeWorkerExecutor` and `NodeWorkerExecutor`.
- **Risk: Missing Edge Headers or CF Context**
  - *Mitigation*: Ensure `ExecutionPayload` preserves `CF-Ray`, `CF-Connecting-IP`, and `cf` geo metadata defaults exactly as implemented currently.
- **Risk: Concurrency in FakeWorkerExecutor**
  - *Mitigation*: Guard `FakeWorkerExecutor` invocation history and configuration with `sync.RWMutex`.
</risks>
