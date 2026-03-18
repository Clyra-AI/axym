# Axym Command Guide

Axym is a deterministic AI governance CLI for platform, security, and GRC teams that need local evidence collection, compliance mapping, and audit-ready bundles.

## Install paths

Homebrew:

```bash
brew install Clyra-AI/tap/axym
axym version --json
```

Source:

```bash
go build ./cmd/axym
./axym version --json
```

Release binary:

```bash
./axym version --json
```

## First 15 minutes

```bash
./axym init --json
./axym collect --dry-run --json
./axym collect --json
./axym map --frameworks eu-ai-act,soc2 --json
./axym gaps --frameworks eu-ai-act,soc2 --json
./axym bundle --audit Q3-2026 --frameworks eu-ai-act,soc2 --json
./axym verify --chain --json
```

## Commands

- `axym init --json`: creates local store scaffolding and policy defaults.
- `axym collect --dry-run --json`: validates fixture and environment readiness without writes.
- `axym collect --json`: runs built-in collectors and appends signed proof records.
- `axym collect --json --plugin "<cmd>"`: runs a third-party collector protocol and rejects malformed JSONL deterministically.
- `axym collect --json --governance-event-file ./events.jsonl`: promotes valid governance events to proof records.
- `axym record add --input ./fixtures/records/decision.json --json`: appends a user-supplied proof record payload.
- `axym ingest --source wrkr --json --input ./fixtures/ingest/wrkr/proof_records.jsonl`: ingests Wrkr evidence with stateful drift tracking.
- `axym ingest --source gait --json --input ./fixtures/ingest/gait`: ingests Gait native/proof pack artifacts with translation.
- `axym map --frameworks eu-ai-act,soc2 --json`: deterministically maps chain evidence to framework controls.
- `axym gaps --frameworks eu-ai-act,soc2 --json`: ranks `covered`, `partial`, and `gap` outcomes with remediation and effort.
- `axym regress init --baseline ./tmp/regress-baseline.json --frameworks eu-ai-act,soc2 --json`: captures deterministic baseline coverage.
- `axym regress run --baseline ./tmp/regress-baseline.json --frameworks eu-ai-act,soc2 --json`: exits `5` on coverage drift with stable control output.
- `axym review --date 2026-09-15 --json`: emits a deterministic Daily Review Pack.
- `axym override create --bundle Q3-2026 --reason "fixture" --signer ops-key --json`: appends signed override evidence and artifacts.
- `axym replay --model payments-agent --tier A --json`: emits replay-certification evidence with deterministic blast-radius summaries.
- `axym bundle --audit Q3-2026 --frameworks eu-ai-act,soc2 --json`: assembles signed audit bundles with executive summary, OSCAL, and portable raw records.
- `axym verify --chain --json`: verifies append-only chain integrity.
- `axym verify --bundle ./axym-evidence --json`: verifies cryptographic bundle integrity and compliance completeness.

## Exit codes

- `0` success
- `1` runtime failure
- `2` verification failure
- `3` policy/schema violation
- `4` approval required
- `5` regression drift
- `6` invalid input
- `7` dependency missing
- `8` unsafe operation blocked
