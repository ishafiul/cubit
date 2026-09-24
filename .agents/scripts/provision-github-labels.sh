#!/usr/bin/env bash
set -euo pipefail

# Provision GitHub Labels for RPG Wayfinder Workflow
REPO="ishafiul/cubit"

echo "Provisioning labels on $REPO..."

create_label() {
  local name="$1"
  local color="$2"
  local desc="$3"
  
  # gh label create with --force updates existing label or creates new one
  gh label create "$name" --repo "$REPO" --color "$color" --description "$desc" --force
  echo "✓ Created/updated label: $name"
}

# Type Labels
create_label "type:map" "0e8a16" "Umbrella feature map tracking destination & decisions"
create_label "wayfinder:map" "0e8a16" "Alias for type:map"
create_label "type:decision" "d93f0b" "HITL decision/interview topic under grilling"
create_label "wayfinder:grilling" "d93f0b" "Alias for type:decision"
create_label "type:research" "1d76db" "AFK background research topic"
create_label "type:prd" "5319e7" "Authoritative Master PRD specification"
create_label "type:task" "006b75" "Workable implementation task (RPG Capability)"
create_label "type:subtask" "fbca04" "Granular child task (RPG Feature)"

# Status Labels
create_label "status:needs-triage" "ededed" "Newly filed, pending initial scope/classification"
create_label "status:grilling" "e99695" "Currently undergoing interactive Q&A / interview"
create_label "status:needs-review" "fef2c0" "Draft ready, awaiting human review or sign-off"
create_label "status:changes-requested" "f9d0c4" "Feedback provided; revisions needed before sign-off"
create_label "status:approved" "c2e0c6" "Specification or architecture decision signed off"
create_label "status:ready-for-agent" "0e8a16" "Fully specified, unblocked, ready for autonomous execution"
create_label "status:in-progress" "bfd4f2" "Currently being implemented by agent or human"
create_label "status:blocked" "b60205" "Blocked by dependency, external access, or question"
create_label "status:in-review" "d4c5f9" "Implementation done, undergoing code review"
create_label "status:completed" "0e8a16" "Work verified, tests passing, closed"
create_label "status:out-of-scope" "ffffff" "Explicitly ruled out or deferred"

echo "All labels successfully provisioned on $REPO!"
