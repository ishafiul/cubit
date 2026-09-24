# Cubit — Agentic Engineering Guide

This document is the authoritative operating manual for AI agents and human contributors working in the **Cubit** repository.

---

## 1. Project Overview & Architecture

**Cubit** is an open-source, bare-metal PaaS designed to execute Cloudflare Workers and Durable Objects on self-hosted infrastructure.

- **Runtime Engine**: [`celld`](https://github.com/denoland/celld) by Deno Land (embedded V8 isolates, private SQLite database cells, LTX replication).
- **Control Plane**: `cubitd` (Go 1.24+, Gin REST API, SQLite database in WAL mode).
- **Ingress & Proxy**: Traefik v3 (dynamic file provider, dynamic route sync, SSL/TLS, edge header injection).
- **Object Storage**: Garage S3 (lightweight distributed object storage for worker bundles, LTX logs, and R2 buckets).
- **Frontend**: React 19, Vite, Tailwind CSS, Lucide icons (served statically by `cubitd` or standalone).

```
[ Inbound HTTP Traffic ]
           │
           ▼
[ Traefik v3 Ingress ] ──(Routes by Domain/Host)──► [ celld Daemon (:8080) ]
           │                                                │
(Dynamic config via YAML)                          (LTX replication & CAS)
           │                                                │
[ Cubit Control Plane (cubitd :8000) ]                      ▼
           │                                       [ Garage S3 (:3900) ]
           ▼
   [ SQLite (cubit.db) ]
```

---

## 2. Issue-Driven Workflow (RPG Wayfinder)

All non-trivial engineering work in this repository is strictly **issue-driven** via GitHub Issues (`ishafiul/cubit`).

### Label Vocabulary
- **Type**:
  - `type:map` — Root initiative map tracking destination, decisions, and fog of war.
  - `type:prd` — Authoritative Master PRD specification.
  - `type:task` — Parent implementation capability issue.
  - `type:subtask` — Atomic feature implementation issue (executable by `/tdd`).
  - `type:decision` — Focused architectural question explored via grilling.
- **Status**:
  - `status:ready-for-agent` — Unblocked, fully specified, ready for immediate execution.
  - `status:in-progress` — Currently being implemented.
  - `status:blocked` — Blocked by an upstream subtask or dependency.
  - `status:completed` — Verified with passing tests, merged, and closed.

### Lifecycle Procedure
1. **Select Next Task**: Pick the highest-priority issue marked `status:ready-for-agent`.
2. **Assign & Start**: Mark issue `status:in-progress`.
3. **Execute TDD Loop (`/tdd`)**:
   - Write failing unit/integration tests matching the agreed seam.
   - Write minimal production code to pass.
   - Refactor for cleanliness and performance.
4. **Code Review (`/code-review`)**: Run two-axis review (Standards + Spec) on the diff.
5. **Commit & Close**: Commit using conventional commit format (`feat: ...`, `fix: ...`) via `/git-committer` and close the GitHub issue with a summary comment referencing the commit SHA.

---

## 3. Hybrid Skills Architecture

Skills live under [`.agents/skills/`](.agents/skills/) and are cataloged in [`.agents/SKILLS-CATALOG.md`](.agents/SKILLS-CATALOG.md).

### Tier 1: Core Daily Drivers (Always in Context)
Available immediately for standard coding workflows without subagent calls:
- `tdd`: Red-green-refactor testing loop.
- `code-review`: Standards + Spec diff review.
- `git-committer`: Conventional commit message generation.
- `gwt-tester`: Given/When/Then test structure.
- `diagnosing-bugs`: Hypothesis-driven debugging loop.
- `codebase-design`: Deep module vocabulary & testable seams.

### Tier 2: Specialized Skills (Loaded On-Demand)
Loaded via slash command or discovered dynamically via the `skill_router` subagent:
- Planning: `rpg-wayfinder`, `wayfinder`, `to-spec`, `to-tickets`.
- Grilling: `grill-me`, `grilling`, `grill-with-docs`.
- Prototyping & Architecture: `prototype`, `improve-codebase-architecture`, `domain-modeling`.
- Maintenance: `resolving-merge-conflicts`, `wizard`, `writing-for-agents`, `triage`.

When importing or editing a skill, run `bash .agents/scripts/sync-skills.sh` to update the catalog and index.

---

## 4. Engineering Conventions & Standards

- **Language**: Go 1.24+ (standard toolchain).
- **Module Depth**: Follow *Philosophy of Software Design* (John Ousterhout). Design deep modules with small, clean interfaces that hide significant internal complexity.
- **Error Handling**: Always wrap errors with context: `fmt.Errorf("action description: %w", err)`. Never swallow errors.
- **Testing**:
  - Use Go's built-in `testing` package with table-driven tests.
  - Follow Given/When/Then structure (`t.Run("Given... When... Then...")`).
  - Unit tests live alongside code (`*_test.go`).
  - Integration tests live in `deploy/` or `internal/adapters/`.
- **Concurrency**: Guard mutable state with `sync.Mutex` or `sync.RWMutex`. Always test concurrent code with `go test -race ./...`.

---

## 5. Key Commands

```bash
# Run unit and integration tests
go test -v ./...

# Run tests with race detector
go test -race ./...

# Build control plane binary
go build -o bin/cubitd cmd/cubitd/main.go

# Start local Docker dependencies (Garage S3, Traefik, celld)
docker compose -f deploy/docker-compose.yml up -d

# Provision GitHub issue labels
bash .agents/skills/rpg-wayfinder/scripts/provision-labels.sh

# Re-index skills
bash .agents/scripts/sync-skills.sh
```
