# Axym

Axym is a deterministic AI governance CLI for platform, security, and GRC engineers who need local evidence collection, compliance mapping, and audit-ready bundles.

## Install

- Homebrew: `brew install Clyra-AI/tap/axym` then `axym version --json`
- Source: `go build ./cmd/axym` then `./axym version --json`
- Release binary: `./axym version --json`

## First value

- `axym init --json`
- `axym collect --dry-run --json`
- `axym collect --json`
- `axym map --frameworks eu-ai-act,soc2 --json`
- `axym gaps --frameworks eu-ai-act,soc2 --json`
- `axym bundle --audit Q3-2026 --frameworks eu-ai-act,soc2 --json`
- `axym verify --chain --json`

## Commands

- `axym collect --dry-run --json`: validates deterministic would-capture output with no writes.
- `axym init --json`: initializes local store scaffolding and writes `axym-policy.yaml` defaults.
- `axym collect --json`: runs built-in collectors and appends signed proof records.
- `axym record add --input <record.json> --json`: appends a JSON proof record payload to the local chain with deterministic dedupe behavior.
- `axym collect --json --plugin "<cmd>"`: runs third-party collector protocol (`stdin` config, `stdout` JSONL).
- `axym collect --json --governance-event-file <file.jsonl>`: ingests governance events and promotes valid events to proof records.
- `axym map --frameworks eu-ai-act,soc2 --json`: deterministically maps chain evidence to framework controls and emits explainable match rationale.
- `axym gaps --frameworks eu-ai-act,soc2 --json`: computes deterministic `covered`/`partial`/`gap` ranking, remediation guidance, and auditability grade.
- `axym map --policy-config ./axym-policy.yaml --json`: applies schema-validated policy defaults and threshold constraints; invalid policy input exits `6`.
- `axym regress init --baseline ./tmp/regress-baseline.json --frameworks eu-ai-act,soc2 --json`: captures deterministic baseline coverage snapshots for regression checks.
- `axym regress run --baseline ./tmp/regress-baseline.json --frameworks eu-ai-act,soc2 --json`: reports deterministic coverage drift and exits `5` when controls regress.
- `axym review --date 2026-09-15 --json`: emits deterministic daily review pack output with exception classes and grade distribution summaries.
- `axym override create --bundle Q3-2026 --reason "<reason>" --signer ops-key --json`: appends signed override artifacts and proof evidence.
- `axym ingest --source wrkr --input <path> --json`: ingests sibling evidence from Wrkr/Gait payloads with deterministic append/dedupe/reject counters.
- `axym replay --model payments-agent --tier A --json`: emits deterministic replay-certification evidence for review workflows.
- `axym bundle --audit <name> --frameworks eu-ai-act,soc2 --json`: assembles deterministic signed audit bundles with executive summary (`.json` + `.pdf`), chain verification, and OSCAL export.
- `axym map --json` and `axym gaps --json`: default to frameworks `eu-ai-act,soc2` when `--frameworks` is omitted.
- `axym map --frameworks eu-ai-act --min-coverage 0.80 --json`: enforces threshold policy and exits non-zero when coverage is below threshold.
- `axym verify --chain --json`: verifies local append-only chain integrity.
- `axym verify --bundle <path> --json`: combines proof cryptographic verification with deterministic bundle compliance-completeness checks.

## Exit codes

- `0` success
- `1` runtime failure
- `2` verification failure
- `3` policy/schema violation
- `4` approval required
- `5` regression drift (including threshold failures in `map`/`gaps`)
- `6` invalid input
- `7` dependency missing
- `8` unsafe operation blocked
