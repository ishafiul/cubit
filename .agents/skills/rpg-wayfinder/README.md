# RPG Wayfinder

An issue-driven planning skill for taking a large initiative from user requirements through an approved PRD to linked implementation issues.

## Workflow

```
Requirements intake
  → map issue + all currently specifiable grilling issues
  → resolve and close every grilling issue
  → RPG PRD file + review issue
  → explicit PRD approval
  → capability tasks + feature subtasks
  → issue branches + linked PRs
  → merged PRs close subtasks, parent tasks, PRD, and map in order
```

GitHub issues hold the planning and execution record. The map stores requirements and decision links; grilling comments preserve user answers; PRD issue comments carry review feedback and approval. The PRD file and issue are updated together, and task creation waits for issue-level approval. Task/subtask issues track implementation through branch, PR review, and merge. Parent tasks close after all subtasks; the PRD and map close after all work under them closes.

## Use

Invoke `rpg-wayfinder` for a large or uncertain initiative. It starts from the Base requirements form in [SKILL.md](SKILL.md), checks GitHub access, and follows the phase gates there.

The workflow needs GitHub CLI authenticated for the repository. Provision labels from the repository root when needed:

```bash
bash .agents/skills/rpg-wayfinder/scripts/provision-labels.sh
```

## References

- [Label taxonomy](references/label-taxonomy.md)
- [GitHub issue operations](references/github-issue-operations.md)
- [RPG PRD template](templates/prd-rpg.md)
