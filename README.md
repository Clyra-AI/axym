# Axym

Axym is an open-source Go CLI for deterministic AI governance evidence collection, proof record emission, compliance mapping, and audit-ready bundle generation.

## Local bootstrap

```bash
go build ./cmd/axym
go test ./...
```

## Collect command surface

```bash
./axym collect --dry-run --json
./axym collect --json
./axym collect --json --plugin "./my-collector"
./axym collect --json --governance-event-file ./events.jsonl
./axym ingest --source wrkr --json --input ./fixtures/ingest/wrkr/proof_records.jsonl
./axym ingest --source gait --json --input ./fixtures/ingest/gait
./axym map --frameworks eu-ai-act,soc2 --json
./axym gaps --frameworks eu-ai-act,soc2 --json
./axym review --date 2026-09-15 --json
./axym review --date 2026-09-15 --format csv
./axym override create --bundle Q3-2026 --reason "fixture" --signer ops-key --json
./axym replay --model payments-agent --tier A --json
./axym bundle --audit Q3-2026 --frameworks eu-ai-act,soc2 --json
./axym verify --chain --json
./axym verify --bundle ./axym-evidence --json
```

`collect` emits deterministic per-source summaries (`sources[]`) with `reason_codes`, supports non-blocking collector failures, and keeps malformed plugin/governance payloads out of the proof chain.

`ingest` supports deterministic sibling ingest from Wrkr and Gait. Wrkr ingest persists drift baseline state in `.axym/wrkr-last-ingest.json`; Gait ingest supports zip/extracted/explicit-path packs and translates `trace`, `approval_token`, and `delegation_token` native records to proof records while preserving relationship envelopes.

`map` deterministically matches chain evidence to framework controls and emits per-control rationale for `covered`/`partial`/`gap` outcomes.

`map`/`gaps` default to `eu-ai-act,soc2` when `--frameworks` is omitted.

`gaps` ranks `partial`/`gap` controls with deterministic remediation and auditability grade output; `--min-coverage` or `--policy-config` can enforce fail-closed coverage thresholds.

`review` emits deterministic daily exception packs with fixed exception classes (`sod`, `approvals`, `enrichment`, `attach`, `replay`, `freeze`, `chain-session-gap`), per-record auditability, replay tier distributions, and attach SLA/status envelopes.

`override create` appends signed override evidence records and append-only override artifacts under `.axym/overrides/`.

`replay` emits `replay_certification` proof records with deterministic tier classification and blast-radius summary fields.

`bundle` assembles deterministic artifact sets (`manifest.json`, `chain-verification.yaml`, `auditability-grade.yaml`, `executive-summary.json`, `executive-summary.pdf`, OSCAL export, and retention/boundary contracts), signs the manifest with local proof keys, and enforces managed output path safety.

`verify --bundle` reports cryptographic integrity plus deterministic Axym compliance-completeness checks (required record classes, field-coverage state, grade recomputation, and OSCAL schema validation).

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
