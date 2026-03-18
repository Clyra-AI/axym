#!/usr/bin/env bash
set -euo pipefail

docs=(
  "README.md"
  "docs/commands/axym.md"
  "docs-site/public/llm/axym.md"
)

for doc in "${docs[@]}"; do
  test -f "$doc"
done
test -f docs-site/public/llms.txt
test -f CONTRIBUTING.md
test -f SECURITY.md

required_commands=(
  "axym init"
  "axym collect"
  "axym map"
  "axym gaps"
  "axym regress"
  "axym review"
  "axym override create"
  "axym replay"
  "axym ingest"
  "axym bundle"
  "axym verify"
  "axym record add"
)

for doc in "${docs[@]}"; do
  for command in "${required_commands[@]}"; do
    grep -Fq "$command" "$doc"
  done
done

required_doc_snippets=(
  "make lint-go"
  "make test-security"
  "make test-docs-links"
  "make prepush-full"
  "./scripts/release_go_nogo.sh --dist-dir dist --binary-name axym"
  "./axym collect --json --governance-event-file ./fixtures/governance/context_engineering.jsonl"
)

for doc in "${docs[@]}"; do
  for snippet in "${required_doc_snippets[@]}"; do
    grep -Fq "$snippet" "$doc"
  done
done

for doc in "${docs[@]}"; do
  for code in 0 1 2 3 4 5 6 7 8; do
    grep -Fq "\`$code\`" "$doc"
  done
done

grep -Fq "CONTRIBUTING.md" README.md
grep -Fq "SECURITY.md" README.md
grep -Fq "CONTRIBUTING.md" docs-site/public/llms.txt
grep -Fq "SECURITY.md" docs-site/public/llms.txt
grep -Fq "README.md" docs-site/public/llms.txt
grep -Fq "docs/commands/axym.md" docs-site/public/llms.txt
grep -Fq "docs-site/public/llm/axym.md" docs-site/public/llms.txt
