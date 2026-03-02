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
./axym verify --chain --json
./axym verify --bundle ./fixtures/bundles/good --json
```

`collect` emits deterministic per-source summaries (`sources[]`) with `reason_codes`, supports non-blocking collector failures, and keeps malformed plugin/governance payloads out of the proof chain.

`ingest` supports deterministic sibling ingest from Wrkr and Gait. Wrkr ingest persists drift baseline state in `.axym/wrkr-last-ingest.json`; Gait ingest supports zip/extracted/explicit-path packs and translates `trace`, `approval_token`, and `delegation_token` native records to proof records while preserving relationship envelopes.

`map` deterministically matches chain evidence to framework controls and emits per-control rationale for `covered`/`partial`/`gap` outcomes.

`map`/`gaps` default to `eu-ai-act,soc2` when `--frameworks` is omitted.

`gaps` ranks `partial`/`gap` controls with deterministic remediation and auditability grade output; `--min-coverage` or `--policy-config` can enforce fail-closed coverage thresholds.

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
