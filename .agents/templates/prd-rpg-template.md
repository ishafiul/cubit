# Repository Planning Graph (RPG) Method - PRD Template

This template structures PRDs using the Repository Planning Graph (RPG) methodology (Microsoft Research). It strictly separates **WHAT** (functional capabilities) from **HOW** (structural code modules), and connects them with explicit, topological dependencies for AI-driven task execution.

---

## Core Principles
1. **Dual-Semantics**: Separate functional capabilities from code structure before mapping them.
2. **Explicit Dependencies**: Always state what depends on what (`Depends on: [X, Y]`).
3. **Topological Order**: Build foundation (Phase 0) first, then layer upwards.
4. **Atomic Features**: Define inputs, outputs, and behavior for every feature.

---

## Template Structure

### 1. Overview (`<overview>`)
- **Problem Statement**: What concrete pain point exists? Who experiences it?
- **Target Users**: Personas, workflows, and objectives.
- **Success Metrics**: Quantifiable outcomes.

### 2. Functional Decomposition (`<functional-decomposition>`)
- **Capability Tree**:
  - `### Capability: [Name]`
    - Description of capability domain
    - `#### Feature: [Name]`
      - **Description**: [One sentence]
      - **Inputs**: [Data / context needed]
      - **Outputs**: [Result / output produced]
      - **Behavior**: [Key logic / transformations]

### 3. Structural Decomposition (`<structural-decomposition>`)
- **Repository Structure**:
  - Directory and file layout mapping capabilities to modules.
- **Module Definitions**:
  - `### Module: [Name]`
    - **Maps to capability**: [Capability name]
    - **Responsibility**: [Single clear purpose]
    - **File structure**: Layout of feature files and `index`
    - **Exports**: Public interfaces, functions, and classes

### 4. Dependency Graph (`<dependency-graph>`)
- **Foundation Layer (Phase 0)**: No dependencies (built first).
- **Layer 1 (Phase 1)**: Depends on Phase 0 modules.
- **Layer 2 (Phase 2)**: Depends on Phase 1 & Foundation modules.

### 5. Implementation Roadmap (`<implementation-roadmap>`)
- **Phased Development**:
  - `### Phase X: [Name]`
    - **Goal**: Foundational or capability milestone
    - **Entry Criteria**: What must exist before starting
    - **Tasks**: List of atomic tasks with acceptance criteria
    - **Exit Criteria**: Observable outcome proving completion
    - **Delivers**: User or developer capability unlocked

### 6. Test Strategy (`<test-strategy>`)
- **Test Pyramid**: Unit vs Integration vs E2E target ratios
- **Coverage Requirements**: Minimum line/branch/function thresholds
- **Critical Test Scenarios**: Happy path, edge cases, error cases, integration seams
- **TDD Guidelines**: Directives for red-green test generation

### 7. Architecture & Decisions (`<architecture>`)
- **System Components**: Major subsystems and communication protocols
- **Data Models**: Schemas, persistence, migrations
- **Technology Stack & ADRs**: Rationale, trade-offs, and alternatives considered

### 8. Risks & Mitigations (`<risks>`)
- **Technical Risks**: Impact, likelihood, mitigation, fallback
- **Dependency & Scope Risks**: Blockers and boundaries
