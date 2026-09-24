---
name: rpg-wayfinder
description: Guide a large initiative from a user requirements intake through GitHub grilling issues, an approved RPG PRD, and dependent task and subtask issues.
disable-model-invocation: true
---

# RPG Wayfinder

Use for work with enough uncertainty or scope to span multiple sessions. It combines Wayfinder's issue map with a GitHub-first PRD workflow. GitHub issues are the planning source of truth; the checked-in PRD is linked from its issue.

For a small, already clear change, use the repo's simpler spec or ticket workflow.

## Lifecycle

`Requirements form → map + grilling issues → resolved grilling → PRD review → approved PRD → tasks + subtasks`

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

If the user requests changes, update the file and issue and keep the PRD in review. Treat explicit user approval in chat as authorization to add `status:approved` to the PRD issue and record the approval comment.

Completion: the PRD file and issue agree, and the PRD issue has `status:approved`. No task or subtask is created before this gate.

## 5. Create tasks and subtasks from the approved PRD

Create one `type:task` parent issue per PRD capability and one `type:subtask` issue per feature. Each subtask includes its source PRD section, description, inputs, outputs, behavior, acceptance criteria, and verification approach. Link every issue to its PRD and map; link subtasks to their parent task.

Create all issue records first, then wire dependencies from the PRD graph using GitHub's native issue dependencies when available. Otherwise record `Blocked by: #<issue>` in the issue body. Apply exactly one readiness status to each actionable issue: `status:ready-for-agent` when it has no open blockers, or `status:blocked` while any blocker remains. Keep parent issues as capability rollups with child checklists.

Completion: every PRD capability and feature is represented, links and blockers are recorded, and actionable issues have correct readiness labels. Report the created issue links and the ready frontier.

## Implementation

When the user asks to implement a ready issue, use the repo's `/tdd` and `/code-review` workflows. Keep implementation scope tied to that issue and its approved PRD.

## References

- [Label taxonomy](references/label-taxonomy.md)
- [GitHub issue commands](references/github-issue-operations.md)
- [RPG PRD template](templates/prd-rpg.md)
- [Provision labels](scripts/provision-labels.sh)
