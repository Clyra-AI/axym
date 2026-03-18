#!/usr/bin/env bash
set -euo pipefail

docs=(
  "README.md"
  "docs/commands/axym.md"
)

for doc in "${docs[@]}"; do
  test -f "$doc"
  grep -qi "platform" "$doc"
  grep -qi "grc" "$doc"
  grep -Fq "brew install Clyra-AI/tap/axym" "$doc"
  grep -Fq "go build ./cmd/axym" "$doc"
  grep -Fq "./axym version --json" "$doc"
  grep -Fq "make lint-go" "$doc"
  grep -Fq "make test-security" "$doc"
  grep -Fq "make test-docs-links" "$doc"
  grep -Fq "./axym collect --json --governance-event-file ./fixtures/governance/context_engineering.jsonl" "$doc"
done

line_number() {
  local needle="$1"
  local file="$2"
  if command -v rg >/dev/null 2>&1; then
    rg -n -F "$needle" "$file" | head -n 1 | cut -d: -f1
    return
  fi
  grep -n -F "$needle" "$file" | head -n 1 | cut -d: -f1
}

for doc in "${docs[@]}"; do
  install_line="$(line_number "brew install Clyra-AI/tap/axym" "$doc")"
  init_line="$(line_number "./axym init --json" "$doc")"
  dry_run_line="$(line_number "./axym collect --dry-run --json" "$doc")"
  collect_line="$(line_number "./axym collect --json" "$doc")"
  map_line="$(line_number "./axym map --frameworks eu-ai-act,soc2 --json" "$doc")"
  bundle_line="$(line_number "./axym bundle --audit Q3-2026 --frameworks eu-ai-act,soc2 --json" "$doc")"

  test -n "$install_line"
  test -n "$init_line"
  test -n "$dry_run_line"
  test -n "$collect_line"
  test -n "$map_line"
  test -n "$bundle_line"

  if (( install_line >= init_line || init_line >= dry_run_line || dry_run_line >= collect_line || collect_line >= map_line || map_line >= bundle_line )); then
    echo "storyline order mismatch in $doc" >&2
    exit 1
  fi
done
