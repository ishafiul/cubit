# RPG Wayfinder Label Taxonomy

This taxonomy standardizes issue types and status states across any GitHub repository using RPG Wayfinder.

---

## Type Labels (`type:*`)

| Label | Color | Role |
| :--- | :--- | :--- |
| `type:map` | `#0E8A16` (Green) | The root initiative map tracking destination, decisions, and fog |
| `type:decision` | `#D93F0B` (Orange) | Focused decision issue explored through interactive grilling |
| `type:research` | `#1D76DB` (Blue) | Background research issue for external documentation or APIs |
| `type:prd` | `#5319E7` (Purple) | Authoritative Master PRD specification issue |
| `type:task` | `#006B75` (Teal) | Implementation task issue (maps to an RPG Capability) |
| `type:subtask` | `#FBCA04` (Yellow) | Granular child issue (maps to an RPG Feature) |

---

## Status Labels (`status:*`)

| Status Label | Color | Lifecycle Role |
| :--- | :--- | :--- |
| `status:needs-triage` | `#EDEDED` (Gray) | Newly filed, pending initial scope/classification |
| `status:grilling` | `#E99695` (Light Red) | Currently undergoing interactive Q&A / interview |
| `status:needs-review` | `#FEF2C0` (Light Yellow) | Draft ready, awaiting human review or PRD sign-off |
| `status:changes-requested` | `#F9D0C4` (Peach) | Feedback provided; revisions needed before sign-off |
| `status:approved` | `#C2E0C6` (Light Green) | Specification or architecture decision signed off |
| `status:ready-for-agent` | `#0E8A16` (Dark Green) | Fully specified, unblocked, ready for autonomous execution |
| `status:in-progress` | `#BFD4F2` (Light Blue) | Currently being implemented by agent or human |
| `status:blocked` | `#B60205` (Dark Red) | Blocked by a dependency, external access, or open question |
| `status:in-review` | `#D4C5F9` (Lavender) | Implementation done, undergoing `/code-review` / PR |
| `status:completed` | `#0E8A16` (Green) | Work verified, tests passing, closed |
| `status:out-of-scope` | `#FFFFFF` (White) | Explicitly ruled out or deferred |
