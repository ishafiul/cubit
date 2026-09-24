---
name: skill-routing
description: Guidelines for utilizing the Hybrid Skill Architecture (Tier 1 Always-On + Tier 2 Router Subagent)
always_on: true
---

# Hybrid Skill Architecture

This repository uses a two-tier hybrid architecture to keep the primary agent prompt lean while retaining instant access to 25+ specialized workflows.

## Tiers Overview

1. **Tier 1: Core Daily Drivers (Direct in Prompt)**
   - `tdd`: Red-green-refactor loop.
   - `code-review`: Standards + Spec diff review.
   - `git-committer`: Conventional commit generation.
   - `gwt-tester`: Given/When/Then test structure.
   - `diagnosing-bugs`: Hypothesis-driven diagnosis loop.
   - `codebase-design`: Deep module vocabulary & testable seams.
   *These are available with zero latency and no subagent call.*

2. **Tier 2: Specialized Skills (Loaded On-Demand via Router)**
   - Includes planning (`rpg-wayfinder` for GitHub grilling issues → approved PRD → tasks/subtasks; `wayfinder` for a decision map and handoff; `to-spec`; `to-tickets`), interviews (`grill-me`, `grill-with-docs`), prototyping (`prototype`), conflict resolution (`resolving-merge-conflicts`), automation (`wizard`), and knowledge management (`domain-modeling`, `teach`, `writing-for-agents`).
   - Marked with `disable-model-invocation: true` to prevent prompt clutter.
   - Indexed in `.agents/skills-index.json`.

## Routing Procedure for Complex Tasks

When a task requires a specialized or multi-step workflow beyond routine daily coding:
1. Invoke the `skill_router` subagent:
   ```json
   {
     "TypeName": "skill_router",
     "Role": "Skill Router",
     "Model": "flash_lite",
     "Prompt": "Classify task: <summary of user request>"
   }
   ```
2. The subagent inspects `.agents/skills-index.json` in ~300ms and returns matching skills.
3. Read the selected `SKILL.md` using `view_file` and execute the instructions.

## Adding Future Skills

When creating or importing a new skill:
1. Create `.agents/skills/<skill-name>/SKILL.md`.
2. Run `bash .agents/scripts/sync-skills.sh`.
3. The script automatically classifies the skill, updates frontmatter, and adds it to `.agents/skills-index.json` and `.agents/SKILLS-CATALOG.md`.
