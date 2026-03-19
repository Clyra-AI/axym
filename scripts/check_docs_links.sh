#!/usr/bin/env bash
set -euo pipefail

python3 - <<'PY'
from pathlib import Path
import re
import sys

repo_root = Path.cwd().resolve()
docs = [
    repo_root / "README.md",
    repo_root / "docs/commands/axym.md",
    repo_root / "docs-site/public/llm/axym.md",
    repo_root / "docs-site/public/llms.txt",
    repo_root / "docs/operator/quickstart.md",
    repo_root / "docs/operator/integration-model.md",
]

for doc in docs:
    if not doc.is_file():
        print(f"missing docs file: {doc.relative_to(repo_root)}", file=sys.stderr)
        sys.exit(1)

link_pattern = re.compile(r"\[([^\]]+)\]\(([^)]+)\)")
failures = []

for doc in docs:
    text = doc.read_text(encoding="utf-8")
    for _label, target in link_pattern.findall(text):
        if target.startswith(("http://", "https://", "mailto:")):
            continue
        if target.startswith("#"):
            continue

        path_part = target.split("#", 1)[0].strip()
        if not path_part:
            continue
        candidate = (doc.parent / path_part).resolve()
        if repo_root not in candidate.parents and candidate != repo_root:
            failures.append(f"{doc.relative_to(repo_root)} -> {target}: escapes repo root")
            continue
        if not candidate.exists():
            failures.append(f"{doc.relative_to(repo_root)} -> {target}: missing target")

llms_index = (repo_root / "docs-site/public/llms.txt").read_text(encoding="utf-8")
for required in [
    "README.md",
    "docs/commands/axym.md",
    "docs-site/public/llm/axym.md",
    "docs/operator/quickstart.md",
    "docs/operator/integration-model.md",
    "docs/operator/integration-boundary.mmd",
]:
    if required not in llms_index:
        failures.append(f"docs-site/public/llms.txt missing {required}")

if failures:
    for failure in failures:
        print(failure, file=sys.stderr)
    sys.exit(1)
PY
