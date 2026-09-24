#!/usr/bin/env bash
set -euo pipefail

# provision-labels.sh
# Sets up RPG Wayfinder type:* and status:* labels in the current GitHub repository.

if ! command -v gh >/dev/null 2>&1; then
  echo "Error: gh CLI is not installed or not in PATH." >&2
  exit 1
fi

if ! gh auth status >/dev/null 2>&1; then
  echo "Error: gh CLI is not authenticated. Please run 'gh auth login' first." >&2
  exit 1
fi

REPO=$(gh repo view --json nameWithOwner -q .nameWithOwner 2>/dev/null || true)
if [ -z "$REPO" ]; then
  echo "Error: Could not determine GitHub repository. Ensure you are inside a git repo with a GitHub remote." >&2
  exit 1
fi

echo "==> Provisioning RPG Wayfinder labels for: $REPO"

# Array of labels: "NAME|COLOR|DESCRIPTION"
LABELS=(
  # Type Labels
  "type:map|0E8A16|The root initiative map tracking destination, decisions, and fog"
  "type:decision|D93F0B|Focused decision issue explored through interactive grilling"
  "type:research|1D76DB|Background research issue for external documentation or APIs"
  "type:prd|5319E7|Authoritative Master PRD specification issue"
  "type:task|006B75|Implementation task issue (maps to an RPG Capability)"
  "type:subtask|FBCA04|Granular child issue (maps to an RPG Feature)"

  # Status Labels
  "status:needs-triage|EDEDED|Newly filed, pending initial scope/classification"
  "status:grilling|E99695|Currently undergoing interactive Q&A / interview"
  "status:needs-review|FEF2C0|Draft ready, awaiting human review or PRD sign-off"
  "status:changes-requested|F9D0C4|Feedback provided; revisions needed before sign-off"
  "status:approved|C2E0C6|Specification or architecture decision signed off"
  "status:ready-for-agent|0E8A16|Fully specified, unblocked, ready for autonomous execution"
  "status:in-progress|BFD4F2|Currently being implemented by agent or human"
  "status:blocked|B60205|Blocked by a dependency, external access, or open question"
  "status:in-review|D4C5F9|Implementation done, undergoing code review or PR"
  "status:completed|0E8A16|Work verified, tests passing, closed"
  "status:out-of-scope|FFFFFF|Explicitly ruled out or deferred"
)

for item in "${LABELS[@]}"; do
  IFS="|" read -r name color desc <<< "$item"
  echo "  Creating/updating label: $name ($color)..."
  gh label create "$name" --color "$color" --description "$desc" --force >/dev/null
done

echo "==> All RPG Wayfinder labels provisioned successfully on $REPO!"
