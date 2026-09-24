---
name: rpg-wayfinder
description: Guide large initiatives from requirements and GitHub grilling issues through an approved RPG PRD, implementation branches and PRs, and issue closure.
disable-model-invocation: true
---

# RPG Wayfinder

Use for work with enough uncertainty or scope to span multiple sessions. It combines Wayfinder's issue map with a GitHub-first PRD workflow. GitHub issues are the planning source of truth; the checked-in PRD is linked from its issue.

For a small, already clear change, use the repo's simpler spec or ticket workflow.

## Lifecycle

`Requirements → map + grilling issues → approved PRD → tasks + subtasks → branches + PRs → merged and closed issue tree`

Every phase ends with a visible GitHub artifact. Continue across phases only when the stated gate is met.

## Start: collect requirements and check GitHub

For a new initiative, fill this form from the user's request and repository context. Accept prose or partial answers; mark unresolved fields `TBD`. Ask the user only for missing information needed to name the destination or establish basic scope.

### Base requirements form

- **Initiative / destination**: What outcome should this effort reach?
- **Problem and impact**: What is painful today, and for whom?
- **Users / actors**: Who or what uses the capability?
- **Desired outcome and success measures**: What changes, and how will success be recognized?
- **In scope**: What must this effort cover?
- **Out of scope**: What is explicitly deferred?
- **Acceptance examples**: What user-visible or system-visible behavior should hold?
- **Constraints and dependencies**: Technical, product, operational, or timing limits.
- **Relevant repository areas**: Known modules, services, APIs, or data.
- **Known decisions**: Choices already settled.
- **Open questions and assumptions**: Anything that may change the requirements.

Before creating issues, read the repository's issue-tracker instructions, confirm `gh auth status` and `gh repo view`, and check that required labels exist. If GitHub access is unavailable, stop at the blocker and ask the user to restore access; report no issue as created unless `gh` returned its URL.

For a continuing effort, load the named map and its open child issues first. Reuse an existing active map for the same initiative rather than creating a duplicate.

## 1. Create or update the map

Create one `type:map,status:in-progress` issue for the initiative. Its body holds the completed requirements form and these sections:

- **Destination**
- **Base requirements**
- **Decisions so far** — one-line gist and link per resolved grilling issue
- **Not yet specified** — in-scope uncertainty that cannot yet be phrased as a decision
- **Out of scope**

The map is an index. Detailed answers live on their grilling issues. Link child issues as native sub-issues when available; otherwise put `Part of #<map>` in each issue and maintain a checklist on the map.

Completion: map issue exists, contains the intake, and its URL is recorded.

## 2. Create the grilling backlog before interviewing

Derive grilling issues from the form's open questions, assumptions, scope boundaries, acceptance examples, and decisions that affect the PRD. Resolve factual questions by inspecting the repository or using relevant research tools. A grilling issue asks the user to choose or define something; factual questions the agent can answer do not become grilling issues.

Create one `type:decision,status:grilling` issue for each independent decision that can be stated now. Combine questions only when one answer resolves them together. Use specific titles such as `[Grilling] Choose worker bundle retention policy`. Each issue body records:

- context and source requirement(s)
- the exact decision needed
- known options and trade-offs, if available
- `Part of #<map>`

Create every currently specifiable grilling issue before asking the first grilling question. Add each issue to the map's child list and retain its URL. For dependent questions, create the known issues and wire their blockers; put questions that cannot yet be stated precisely in **Not yet specified**. Create those later when the frontier makes them clear.

If there are no unresolved decisions, confirm the intake is complete and proceed to the PRD gate.

Completion: every currently specifiable user decision has an open, linked grilling issue with the correct labels. Show the user the issue links before starting an interview.

## 3. Resolve grilling issues

Work one human-in-the-loop grilling issue per session, following the repo's `grilling` skill. Select a user-named issue or the first unblocked, unclaimed issue. Assign it to `@me` before investigating or interviewing.

Ask the issue's decision questions in rounds and wait for the user's answers. After each round, comment on that GitHub issue with the question, answer, findings, and trade-offs. When the decision is settled:

1. Post an authoritative resolution comment with the chosen answer, constraints, and rejected alternatives that matter.
2. Add `status:approved`, remove `status:grilling`, and close the issue.
3. Add a one-line linked gist to the map's **Decisions so far**.
4. If the answer reveals another user decision, create and link its grilling issue before interviewing on it. Record still-foggy areas under **Not yet specified**.

Grilling is complete only when every in-scope decision issue is closed with `status:approved`, and the map has no in-scope uncertainty that could materially change the PRD. Resolve or explicitly scope out each such item first. If issues remain, stop after the current issue and report the next frontier.

## 4. Synthesize and publish the PRD

After the grilling completion gate, read the intake, map, and final resolution comments from every grilling issue. Create `docs/prd/<feature-slug>.md` from [the RPG PRD template](templates/prd-rpg.md). Preserve traceability with links to the map and source decisions. Separate functional **WHAT** from structural **HOW**, include explicit dependencies, acceptance criteria, test strategy, risks, and roadmap, and mark any remaining assumptions clearly.

Create a `type:prd,status:needs-review` issue linking to the file. Include a concise summary and review checklist, then link the PRD issue from the map. The issue is the review and approval record.

Read the PRD issue comments for review feedback. If the user requests a change, add `status:changes-requested` and remove `status:needs-review`; update the PRD file and issue description, then comment with what changed and links to the updated file and relevant sections. Return the issue to `status:needs-review` and wait for another review.

Accept approval only when the PRD issue records it: a user/maintainer sets `status:approved` or leaves an explicit approval comment. If the comment approves but the label is missing, add the label and record the approval. Chat-only approval does not release task creation. Recheck the latest issue comments and ensure the file and issue description match before proceeding.

Completion: the PRD file and issue agree, there are no outstanding change requests, and the PRD issue has `status:approved` based on issue-level approval. No task or subtask is created before this gate.

## 5. Create tasks and subtasks from the approved PRD

Create one `type:task` issue per PRD capability and one `type:subtask` issue per feature. Each issue links its source PRD section, map, relevant files or decisions, and parent/child issues. Each subtask includes description, inputs, outputs, behavior, acceptance criteria, and verification approach.

Create all issue records first, then wire dependencies from the PRD graph using GitHub's native issue dependencies when available. Otherwise record `Blocked by: #<issue>` in the issue body. Apply exactly one readiness status to each actionable issue: `status:ready-for-agent` when it has no open blockers, or `status:blocked` while any blocker remains. Keep parent issues as capability rollups with child checklists.

Completion: every PRD capability and feature is represented, links and blockers are recorded, and actionable issues have correct readiness labels. Report the created issue links and the ready frontier.

## 6. Start or resume from an issue or PR link

An issue number or URL is enough to start or resume its type-specific workflow. Read the issue body, comments, labels, parent links, blockers, and source PRD before changing state. If the user gives a PR link, inspect its current state and related issues.

- **Grilling / decision**: continue the interview in the issue. Comment each Q&A round and final decision; update the issue description with the current question and settled outcome while preserving the comment history. Link relevant requirements, source files, and related issues.
- **PRD**: read new issue comments, update the PRD file and issue description for requested changes, comment with a summary and file links, and create tasks only after issue-level approval is recorded.
- **Map**: resume its frontier and keep its issue links and progress index current.
- **Task / subtask**: an open, ready issue ID or URL means start or resume implementation. First confirm the PRD is approved and all blockers are closed. If blocked, keep `status:blocked`, comment with blocker links, and stop. Otherwise assign the issue to `@me`, add `status:in-progress`, remove `status:ready-for-agent`, and comment with the PRD and relevant child/parent links.

For a task or subtask, inspect existing branches and PRs for the issue. Resume its existing branch/PR when present. Otherwise create a branch from the repository's default branch using its naming convention; when none exists, use `issue/<number>-<slug>`. Preserve unrelated work in the working tree.

Keep the issue description aligned with approved scope and acceptance criteria. If implementation reveals an accepted clarification or a new blocker, update the description and add a comment linking its source issue, PRD section, or repository file.

Implement against the approved issue and PRD using the repo's `/tdd` and `/code-review` workflows. Commit and push the issue branch, then create a PR. Format the PR title using Conventional Commits referencing the issue (e.g. `<type>(<scope>): <Summary> (#<issue-number>)`). The PR description must describe what was actually implemented and include verification results plus links to the issue, parent task, PRD, map, and relevant files or decisions. Use `Closes #<issue>` for a leaf subtask or task with no children. For a parent task with open subtasks, use `Part of #<task>` so its PR cannot close the rollup early.

After opening a PR, change the implementation issue to `status:in-review` (remove `status:in-progress`) and comment with the PR link. Update the PR description and issue comment when implementation scope or verification changes. Use comments for meaningful progress, decisions, blockers, and handoffs; keep each comment useful to the next person.

## 7. Reconcile merged PRs and close the issue tree

Whenever resuming from an issue or PR link, refresh GitHub state. If a leaf issue's PR is merged, add `status:completed`, remove stale readiness/progress/review labels, and close the issue if GitHub has not already closed it. Comment with the merged PR link. Never mark work completed while its PR is open or unmerged. If a parent task's PR is merged while subtasks remain open, remove `status:in-review`, keep the task open as `status:in-progress`, and comment that child work remains. If all subtasks are already closed, use the checklist and parent closure gate below before closing the parent.

After a subtask PR merges, or before closing its parent, inspect the parent task's issue-body checklist. GitHub normally checks task-list items that directly reference closed issues. Verify every closed child's item is checked; for each unchecked item whose issue is closed, update only that marker to `[x]`, preserve the rest of the issue body, and comment with the child and merged PR links. Never check an item before its issue is closed.

Then inspect all subtasks linked to the parent. When every subtask is closed and checked in the parent and the parent has no unmerged PR, mark the parent `status:completed`, comment with the child links, and close it. If the parent has an open PR, keep it `status:in-review` until that PR merges. If any child remains open, keep the parent open and report the remaining children.

After all tasks and subtasks linked to a PRD are closed, mark the PRD `status:completed`, comment with the completed task/PR links, and close it. Then, if the map has no other in-scope open work, mark the map `status:completed`, comment with the PRD link, and close it.

The skill reconciles GitHub web merges the next time it is invoked or resumed with the issue/PR link; it does not run a background GitHub watcher.

## References

- [Label taxonomy](references/label-taxonomy.md)
- [GitHub issue commands](references/github-issue-operations.md)
- [RPG PRD template](templates/prd-rpg.md)
- [Provision labels](scripts/provision-labels.sh)
