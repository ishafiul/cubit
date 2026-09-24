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
```

GitHub issues hold the planning record. The map stores the requirements and links to decisions; grilling comments preserve user answers; the PRD issue gates task creation. The agent creates grilling issues before interviewing and creates tasks only after PRD approval.

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
