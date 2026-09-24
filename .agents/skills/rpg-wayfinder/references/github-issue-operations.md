# GitHub issue operations

Use these patterns from the target repository. Confirm `gh auth status` and `gh repo view` before writes. Save each returned URL and issue number; a planned command is not a created issue.

## Map

Create one map with the user's completed intake. Continue an existing active map when one already tracks the initiative.

```bash
gh issue create \
  --title "[Map] <Initiative>" \
  --label "type:map,status:in-progress" \
  --body "$(cat <<'EOF'
## Destination
<Outcome>

## Base requirements
- Problem and impact:
- Users / actors:
- Desired outcome and success measures:
- In scope:
- Out of scope:
- Acceptance examples:
- Constraints and dependencies:
- Relevant repository areas:
- Known decisions:
- Open questions and assumptions:

## Decisions so far

## Not yet specified

## Out of scope
EOF
)"
```

Put `Part of #<map>` in each child issue. When native sub-issues are available, link the child to the map and keep a short checklist on the map.

After creating the initial grilling backlog, update the map body with links to every created issue while preserving its intake and other sections.

## Grilling issues

Create one issue per independent, currently specifiable decision before interviewing the user.

```bash
gh issue create \
  --title "[Grilling] <Specific decision>" \
  --label "type:decision,status:grilling" \
  --body "$(cat <<'EOF'
Part of #<map>

## Context
<Relevant requirement and repository facts>

## Decision needed
<One question the user must settle>

## Known options and trade-offs
<Options, or TBD>

## Source requirements
<Requirement names or links>
EOF
)"
```

After each answer, preserve the discussion on the issue:

```bash
gh issue comment <decision> --body "$(cat <<'EOF'
### Grilling findings
- **Question**: <Question>
- **Answer**: <User's answer>
- **Findings and trade-offs**: <Relevant facts and alternatives>
EOF
)"
```

Resolve only after the user settles the decision:

```bash
gh issue comment <decision> --body "$(cat <<'EOF'
### Final decision
<Chosen answer, constraints, and rejected alternatives that matter>
EOF
)"
gh issue edit <decision> --add-label "status:approved" --remove-label "status:grilling"
gh issue close <decision>
gh issue comment <map> --body "- [<Decision title>](<issue-url>): <One-line gist>"
```

When an answer reveals another user decision, create its issue before interviewing on it. Add a parent link or map checklist entry for every new issue.

## PRD

Create the local PRD only after all in-scope grilling issues are closed and approved and material fog is resolved or scoped out. Then publish the review issue:

```bash
gh issue create \
  --title "[PRD] <Initiative>" \
  --label "type:prd,status:needs-review" \
  --body "$(cat <<'EOF'
Specification: `docs/prd/<feature-slug>.md`
Part of #<map>

## Summary
<Problem, users, outcome>

## Review checklist
- [ ] Requirements and grilling decisions are reflected
- [ ] Functional capabilities and acceptance criteria are complete
- [ ] Structural modules and public seams are clear
- [ ] Dependencies are topologically ordered
- [ ] Test strategy and risks are documented
EOF
)"
```

On approval, record it on the issue. If the user explicitly approves in chat, that authorizes applying the label:

```bash
gh issue edit <prd> --add-label "status:approved" --remove-label "status:needs-review" --remove-label "status:changes-requested"
gh issue comment <prd> --body "PRD approved. Creating capability and feature issues."
```

## Tasks and subtasks

Create one parent task per capability and one subtask per feature. Include `Part of #<prd>` and the parent link in each subtask. Create all issue records before wiring dependency edges.

```bash
gh issue create \
  --title "[Task] <Capability>" \
  --label "type:task" \
  --body "$(cat <<'EOF'
Part of #<prd>

## Capability
<Outcome and scope>

## Subtasks
<!-- Add issue links after creation -->
EOF
)"

gh issue create \
  --title "[Subtask] <Feature>" \
  --label "type:subtask" \
  --body "$(cat <<'EOF'
Parent task: #<task>
Part of #<prd>

## Feature specification
- Description:
- Inputs:
- Outputs:
- Behavior:
- Acceptance criteria:
- Verification:

## Blocked by
None
EOF
)"
```

Link each created subtask from its parent using a Markdown task list. Link native issue dependencies when available; the blocker value must be the blocker's GitHub database ID:

```bash
gh api repos/<owner>/<repo>/issues/<issue-number>/dependencies/blocked_by \
  --method POST -F issue_id=<blocker-database-id>
```

If native dependencies are unavailable, keep `Blocked by: #<issue>` in the body. After wiring, give each issue exactly one readiness label: `status:ready-for-agent` when no open blockers remain, or `status:blocked` otherwise.

```bash
gh issue edit <issue> --add-label "status:ready-for-agent" --remove-label "status:blocked"
gh issue edit <issue> --add-label "status:blocked" --remove-label "status:ready-for-agent"
```
