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
./axym verify --chain --json
./axym verify --bundle ./fixtures/bundles/good --json
```

`collect` emits deterministic per-source summaries (`sources[]`) with `reason_codes`, supports non-blocking collector failures, and keeps malformed plugin/governance payloads out of the proof chain.

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
