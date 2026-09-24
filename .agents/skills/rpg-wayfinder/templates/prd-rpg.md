# Repository Planning Graph (RPG) Method - PRD Template

This template structures Product Requirements Documents (PRDs) using the **Repository Planning Graph (RPG) methodology** (Microsoft Research). It strictly separates **WHAT** (functional capabilities) from **HOW** (structural code modules), connecting them with explicit, topological dependencies for AI and human execution.

---

<overview>
## Problem Statement
- **What concrete pain point exists?** [Describe problem in detail]
- **Who experiences it?** [Target personas and user segments]
- **Why existing solutions don't work?** [Current limitations and friction]

## Target Users & Personas
- **Persona 1**: [Description, workflow, and goals]
- **Persona 2**: [Description, workflow, and goals]

## Success Metrics
- [Metric 1: Quantifiable outcome]
- [Metric 2: Quantifiable outcome]

## Source Requirements & Decisions
- **Requirements map**: [Map issue link]
- **Resolved grilling issues**: [Links to approved decision issues]
- **Constraints carried forward**: [Material constraints and accepted trade-offs]
</overview>

---

<functional-decomposition>
## Capability Tree (The WHAT)
Enumerate system capabilities independent of code organization.

### Capability: [Capability Name]
[Brief description of capability domain]

#### Feature: [Feature Name]
- **Description**: [Single sentence defining what this feature does]
- **Inputs**: [Data, parameters, context, or triggers required]
- **Outputs**: [Resulting data, payloads, or state mutations]
- **Behavior**: [Core transformation, logic rules, and validation]

#### Feature: [Feature Name]
- **Description**: [Single sentence]
- **Inputs**: [Inputs]
- **Outputs**: [Outputs]
- **Behavior**: [Behavior]
</functional-decomposition>

---

<structural-decomposition>
## Repository Structure & Modules (The HOW)
Map functional capabilities directly to codebase architecture and clear module seams.

### Repository Structure
```text
src/ (or internal/)
├── [module-name]/       # Maps to: [Capability Name]
│   ├── [feature-1].ts   # Maps to: [Feature 1 Name]
│   ├── [feature-2].ts   # Maps to: [Feature 2 Name]
│   └── index.ts         # Public exports (the seam)
└── [module-name]/
```

### Module Definitions

#### Module: [Module Name]
- **Maps to capability**: [Capability from Functional Decomposition]
- **Responsibility**: [Single, clear responsibility]
- **File Structure**:
  ```text
  [module-name]/
  ├── ...
  └── index.ts
  ```
- **Exports (Public Seams)**:
  - `functionName(param: Type): ReturnType` — [What it does]
  - `ClassName` — [Public class interface]
</structural-decomposition>

---

<dependency-graph>
## Explicit Topological Dependency Chain
Define explicit dependencies between modules. Foundation modules (Phase 0) have NO dependencies and are built first.

### Foundation Layer (Phase 0)
No dependencies — foundational contracts, data primitives, error types.
- **[Module A]**: No dependencies. [Provides: base types, error handling]
- **[Module B]**: No dependencies. [Provides: configuration, logging]

### Core Domain Layer (Phase 1)
- **[Module C]**: Depends on [[Module A], [Module B]]
- **[Module D]**: Depends on [[Module A]]

### Application / Presentation Layer (Phase 2)
- **[Module E]**: Depends on [[Module C], [Module D]]
</dependency-graph>

---

<implementation-roadmap>
## Phased Development Roadmap
Topological order following the Dependency Graph.

### Phase 0: Foundation
- **Goal**: [Establish foundational primitives]
- **Entry Criteria**: [Clean repository state]
- **Tasks**:
  - [ ] [Task 1 Name] (depends on: none)
    - Acceptance criteria: [How we verify it]
    - Test strategy: [Unit test at seam]
  - [ ] [Task 2 Name] (depends on: none)
- **Exit Criteria**: [Foundation compiles, passes tests, and exports cleanly]
- **Delivers**: [Unblocks Phase 1 development]

### Phase 1: Core Domain
- **Goal**: [Build core capabilities]
- **Entry Criteria**: Phase 0 complete and verified
- **Tasks**:
  - [ ] [Task 3 Name] (depends on: [Task 1, Task 2])
- **Exit Criteria**: [End-to-end core workflows operational]
- **Delivers**: [Runnable core business logic]
</implementation-roadmap>

---

<test-strategy>
## Test Strategy & Critical Scenarios
Defines test verification for `/tdd` red-green execution.

### Test Pyramid
```text
        /\
       /E2E\        ← 10% (End-to-end integration flows)
      /------\
     /Integration\  ← 30% (Module interactions at seams)
    /------------\
   /  Unit Tests  \ ← 60% (Fast, deterministic component tests)
  /----------------\
```

### Critical Test Scenarios
#### Module: [Module Name]
- **Happy Path**: [Valid payload/state produces expected output]
- **Edge Cases**: [Boundary conditions, empty lists, nulls]
- **Error Cases**: [Invalid inputs handle failures gracefully with typed errors]
- **Integration Seams**: [Inter-module boundaries tested via mock or real adapter]
</test-strategy>

---

<architecture>
## System Architecture & ADRs
- **Architecture Overview**: [System components and sequence flows]
- **Data Models**: [Database schemas, migrations, persistence contracts]
- **Decisions & Trade-offs (ADRs)**:
  - **Decision**: [Technology, pattern, or constraint]
  - **Rationale**: [Why chosen]
  - **Trade-offs**: [Trade-offs accepted]
  - **Alternatives Considered**: [Rejected options and why]
</architecture>

---

<risks>
## Risk Analysis & Mitigations
- **Technical Risks**: [Complexity, performance bottlenecks, unknowns]
- **Dependency Risks**: [External APIs, third-party libraries]
- **Scope Risks**: [Feature creep boundaries]
</risks>
