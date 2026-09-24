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
gh issue edit <decision> --body-file /tmp/decision-body.md
gh issue edit <decision> --add-label "status:approved" --remove-label "status:grilling"
gh issue close <decision>
gh issue comment <map> --body "- [<Decision title>](<issue-url>): <One-line gist>"
```

The updated description keeps the original question and current resolution easy to find; comments remain the full Q&A history. Link the source requirement, relevant files, and related issues where useful.

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

Keep user review on the PRD issue. For requested changes, update the checked-in PRD and issue description, comment with a concise change summary and file/section links, then return the issue to review:

```bash
gh issue edit <prd> --add-label "status:changes-requested" --remove-label "status:needs-review"
gh issue edit <prd> --body-file /tmp/updated-prd-issue.md
gh issue comment <prd> --body "Updated `docs/prd/<feature-slug>.md`: <change summary and relevant section links>. Ready for review."
gh issue edit <prd> --add-label "status:needs-review" --remove-label "status:changes-requested"
```

Create tasks only after the PRD issue shows `status:approved` from explicit user/maintainer approval. An explicit approval comment is sufficient; apply the label and record it if missing. Chat-only approval does not pass this gate.

```bash
gh issue edit <prd> --add-label "status:approved" --remove-label "status:needs-review" --remove-label "status:changes-requested"
gh issue comment <prd> --body "PRD approval recorded on this issue. Creating capability and feature issues."
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

## Implementation branches and PRs

When the user gives a task/subtask issue number or URL, read it and its comments, confirm PRD approval and closed blockers, then claim it:

```bash
gh issue view <issue> --comments
gh issue edit <issue> --add-assignee "@me" --add-label "status:in-progress" \
  --remove-label "status:ready-for-agent" --remove-label "status:blocked"
gh issue comment <issue> --body "Started implementation. Source PRD: #<prd>. Related: #<parent>."
```

Resume an existing issue branch or PR. Inspect `git status` first; isolate work or stop if unrelated changes would be carried into the PR. Otherwise branch from the repository default branch:

```bash
git switch <default-branch>
git pull --ff-only
git switch -c "issue/<issue-number>-<slug>"
git push -u origin HEAD
```

Write the PR body from the work actually completed. Include a summary, relevant behavior, verification results, and related issue/file links. Use `Closes #<subtask>` for a subtask, `Closes #<task>` for a task with no subtasks, and `Part of #<task>` for a parent task with children. Also link the PRD and map with `Part of` references.

```bash
gh pr create --base <default-branch> --title "[#<issue-number>] <Summary>" --body-file /tmp/pr-body.md
gh pr edit <pr> --body-file /tmp/pr-body.md
gh issue edit <issue> --add-label "status:in-review" --remove-label "status:in-progress"
gh issue comment <issue> --body "PR: <pr-url>. Verification: <commands/results>."
```

On every resume with an issue or PR link, refresh merge state:

```bash
gh pr view <pr> --json state,mergedAt,url
```

Only a merged PR completes its leaf issue. GitHub may already have closed an issue referenced by `Closes`; still apply the completed label and record the merge link. Run `gh issue close` only when the issue is still open:

```bash
gh issue edit <issue> --add-label "status:completed" --remove-label "status:in-review" \
  --remove-label "status:in-progress" --remove-label "status:ready-for-agent" \
  --remove-label "status:blocked"
gh issue close <issue> --comment "Completed by merged PR <pr-url>."
```

If GitHub already closed the leaf through `Closes`, apply the label and add a comment with `gh issue comment <issue> --body "Completed by merged PR <pr-url>."` instead of running `gh issue close` again.

If the issue is a parent task with open subtasks, leave it open as `status:in-progress` after its own PR merges. Close it after every child subtask is closed and its own PR, if any, is merged. If children finish while its PR is still open, keep it `status:in-review` until merge. Once all tasks and subtasks for a PRD are closed, mark and close the PRD; when the map has no remaining in-scope work, mark and close the map too. Add a rollup comment with links to the closed children and merged PRs at each closure.

```bash
gh issue edit <prd> --add-label "status:completed" --remove-label "status:approved"
gh issue close <prd> --comment "All tasks and subtasks are complete: <issue/PR links>."
gh issue edit <map> --add-label "status:completed" --remove-label "status:in-progress"
gh issue close <map> --comment "Initiative complete. PRD: #<prd>."
```
