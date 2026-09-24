# Issue Tracker: GitHub

Repository: `ishafiul/cubit`  
Tooling: GitHub CLI (`gh`)

## Overview
All planning, grilling findings, master PRDs, and implementation tasks live directly on GitHub Issues.

## Label Taxonomy

### Type Labels (`type:*`)
- `type:map` (or `wayfinder:map`) — High-level initiative anchor tracking destination and closed decisions.
- `type:decision` (or `wayfinder:grilling`) — HITL decision ticket explored via interactive interview.
- `type:research` (or `wayfinder:research`) — AFK background research ticket for docs or APIs.
- `type:prd` — Master Product Requirements Document following RPG format.
- `type:task` — Workable implementation task issue (maps to an RPG Capability).
- `type:subtask` — Granular child issue (maps to an RPG Feature).

### Status Labels (`status:*`)
- `status:needs-triage` — Newly filed issue.
- `status:grilling` — Currently undergoing interactive Q&A.
- `status:needs-review` — Draft/decision ready for human review.
- `status:changes-requested` — Feedback given; revisions needed.
- `status:approved` — PRD or decision signed off.
- `status:ready-for-agent` — Fully specified, unblocked, ready to be implemented.
- `status:in-progress` — Currently being implemented.
- `status:blocked` — Blocked by an unresolved dependency.
- `status:in-review` — Implementation complete, undergoing `/code-review`.
- `status:completed` — Verified, tests pass, closed.
- `status:out-of-scope` — Explicitly ruled out or deferred.

## Key GitHub CLI (`gh`) Commands

### Map Operations
```bash
# Create Wayfinder Map
gh issue create --title "[Map] <Feature Name>" --label "type:map,status:in-progress" --body "## Destination\n...\n\n## Notes\n...\n\n## Decisions so far\n\n## Not yet specified\n...\n\n## Out of scope\n..."

# Update Map Decisions So Far
gh issue comment <map_id> --body "Recorded decision: [#<dec_id> <Title>](<url>) - <gist>"
```

### Decision / Grilling Operations
```bash
# Create Decision Ticket
gh issue create --title "[Decision] <Topic>" --label "type:decision,status:grilling" --body "## Question\n<specific question to resolve>\n\nPart of #<map_id>"

# Post Grilling Findings & Trade-offs
gh issue comment <dec_id> --body "### Findings & Q&A\n- **Q**: ...\n- **A**: ...\n- **Trade-offs**: ..."

# Resolve and Close Decision Ticket
gh issue edit <dec_id> --add-label "status:approved" --remove-label "status:grilling"
gh issue close <dec_id> --comment "### Decision\n<Final resolution summary>"
```

### PRD Operations
```bash
# Create Master PRD Issue
gh issue create --title "[PRD] <Feature Name>" --label "type:prd,status:needs-review" --body "..."

# Approve PRD
gh issue edit <prd_id> --add-label "status:approved" --remove-label "status:needs-review"
```

### Task & Sub-Task Operations
```bash
# Create Task (Capability)
gh issue create --title "[Task] <Capability>" --label "type:task,status:ready-for-agent" --body "Part of #<prd_id>\n\n### Sub-tasks\n- [ ] #<subtask_1> ...\n- [ ] #<subtask_2> ..."

# Create Sub-task (Feature)
gh issue create --title "[Subtask] <Feature>" --label "type:subtask,status:ready-for-agent" --body "Parent Task: #<task_id>\nPart of #<prd_id>\n\nBlocked by: #<dep_id>"
```
