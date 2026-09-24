#!/usr/bin/env python3
"""
Sync Skills Index
Scans .agents/skills/ directory, configures Tier 1 (always-on) vs Tier 2 (on-demand),
and generates .agents/skills-index.json and .agents/SKILLS-CATALOG.md.
"""

import os
import re
import json

REPO_ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
SKILLS_DIR = os.path.join(REPO_ROOT, ".agents", "skills")
OUTPUT_JSON = os.path.join(REPO_ROOT, ".agents", "skills-index.json")
OUTPUT_MD = os.path.join(REPO_ROOT, ".agents", "SKILLS-CATALOG.md")

TIER_1_SKILLS = {
    "tdd",
    "code-review",
    "git-committer",
    "gwt-tester",
    "diagnosing-bugs",
    "codebase-design",
}

def parse_frontmatter(content):
    match = re.match(r"^---\s*\n(.*?)\n---\s*\n(.*)$", content, re.DOTALL)
    if not match:
        return {}, content
    yaml_text = match.group(1)
    body = match.group(2)
    meta = {}
    current_key = None
    multiline_buf = []

    for line in yaml_text.splitlines():
        if line.strip().startswith("#"):
            continue
        key_match = re.match(r"^([a-zA-Z0-9_-]+):\s*(.*)$", line)
        if key_match:
            if current_key and multiline_buf:
                meta[current_key] = " ".join(multiline_buf).strip().strip('"').strip("'")
                multiline_buf = []
            key = key_match.group(1)
            val = key_match.group(2).strip()
            current_key = key
            if val in (">", ">-", "|", "|-"):
                multiline_buf = []
            elif val.lower() == "true":
                meta[key] = True
                current_key = None
            elif val.lower() == "false":
                meta[key] = False
                current_key = None
            elif val != "":
                meta[key] = val.strip('"').strip("'")
                current_key = None
        elif current_key and line.startswith("  "):
            multiline_buf.append(line.strip())

    if current_key and multiline_buf:
        meta[current_key] = " ".join(multiline_buf).strip().strip('"').strip("'")

    return meta, body

def update_skill_tier(skill_path, is_tier_1):
    with open(skill_path, "r", encoding="utf-8") as f:
        content = f.read()

    meta, body = parse_frontmatter(content)
    needs_update = False

    if is_tier_1:
        # Tier 1 should NOT have disable-model-invocation: true
        if meta.get("disable-model-invocation") is True:
            # remove line
            lines = content.splitlines(keepends=True)
            new_lines = []
            in_fm = False
            fm_count = 0
            for line in lines:
                if line.strip() == "---":
                    fm_count += 1
                    in_fm = (fm_count == 1)
                elif in_fm and re.match(r"^disable-model-invocation:\s*true", line.strip()):
                    needs_update = True
                    continue
                new_lines.append(line)
            content = "".join(new_lines)
    else:
        # Tier 2 MUST have disable-model-invocation: true
        if meta.get("disable-model-invocation") is not True:
            # insert into frontmatter
            lines = content.splitlines(keepends=True)
            new_lines = []
            fm_count = 0
            inserted = False
            for line in lines:
                if line.strip() == "---":
                    fm_count += 1
                    if fm_count == 2 and not inserted:
                        new_lines.append("disable-model-invocation: true\n")
                        inserted = True
                        needs_update = True
                new_lines.append(line)
            content = "".join(new_lines)

    if needs_update:
        with open(skill_path, "w", encoding="utf-8") as f:
            f.write(content)

    return parse_frontmatter(content)[0]

def main():
    if not os.path.exists(SKILLS_DIR):
        print(f"Error: {SKILLS_DIR} not found")
        return

    skills = []
    for entry in sorted(os.listdir(SKILLS_DIR)):
        skill_dir = os.path.join(SKILLS_DIR, entry)
        skill_file = os.path.join(skill_dir, "SKILL.md")
        if not os.path.isdir(skill_dir) or not os.path.isfile(skill_file):
            continue

        is_tier_1 = entry in TIER_1_SKILLS
        meta = update_skill_tier(skill_file, is_tier_1)
        rel_path = os.path.relpath(skill_file, REPO_ROOT)

        name = meta.get("name", entry)
        desc = meta.get("description", "").strip()
        disabled = meta.get("disable-model-invocation", False)

        skills.append({
            "name": name,
            "tier": 1 if is_tier_1 else 2,
            "path": rel_path,
            "description": desc,
            "disable_model_invocation": bool(disabled)
        })

    # Sort: Tier 1 first, then Tier 2 alphabetically
    skills.sort(key=lambda s: (s["tier"], s["name"]))

    with open(OUTPUT_JSON, "w", encoding="utf-8") as f:
        json.dump(skills, f, indent=2)

    # Write Markdown catalog
    md_lines = [
        "# Antigravity Skills Catalog",
        "",
        "This catalog indexes all skills available in `.agents/skills/`.",
        "It supports the **Hybrid Architecture**: Tier 1 core skills are always present in the prompt, while Tier 2 specialized skills are dynamically discovered by the `skill_router` subagent (`Model: 'flash_lite'`).",
        "",
        "## Tier 1: Core Daily Drivers (Always in prompt)",
        "Zero latency, immediately available for common development workflows.",
        "",
        "| Skill | Description | Path |",
        "| :--- | :--- | :--- |",
    ]

    for s in skills:
        if s["tier"] == 1:
            md_lines.append(f"| **`{s['name']}`** | {s['description']} | [`{s['path']}`](file://{os.path.join(REPO_ROOT, s['path'])}) |")

    md_lines.extend([
        "",
        "## Tier 2: Specialized On-Demand Skills (Loaded via Router Subagent)",
        "Hidden from the main prompt (`disable-model-invocation: true`) to preserve context tokens. Loaded dynamically via `skill_router` subagent or invoked manually by user slash commands.",
        "",
        "| Skill | Description | Path |",
        "| :--- | :--- | :--- |",
    ])

    for s in skills:
        if s["tier"] == 2:
            md_lines.append(f"| **`{s['name']}`** | {s['description']} | [`{s['path']}`](file://{os.path.join(REPO_ROOT, s['path'])}) |")

    with open(OUTPUT_MD, "w", encoding="utf-8") as f:
        f.write("\n".join(md_lines) + "\n")

    print(f"Indexed {len(skills)} skills ({len([s for s in skills if s['tier'] == 1])} Tier 1, {len([s for s in skills if s['tier'] == 2])} Tier 2).")
    print(f"Generated {OUTPUT_JSON} and {OUTPUT_MD}")

if __name__ == "__main__":
    main()
