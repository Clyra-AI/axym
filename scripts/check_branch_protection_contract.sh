#!/usr/bin/env bash
set -euo pipefail

workflow=".github/workflows/pr.yml"
test -f "$workflow"

required_jobs=(
  "lint-fast:"
  "test-fast:"
  "test-contracts:"
  "test-acceptance:"
  "docs-consistency:"
  "docs-storyline:"
)

for job in "${required_jobs[@]}"; do
  grep -q "$job" "$workflow" || {
    echo "missing required PR job: $job" >&2
    exit 1
  }
done

grep -q "pull_request:" "$workflow" || {
  echo "required checks must map to pull_request workflow outputs" >&2
  exit 1
}
