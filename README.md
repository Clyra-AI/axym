# Axym

Axym is an open-source Go CLI for deterministic AI governance evidence collection, proof record emission, compliance mapping, and audit-ready bundle generation.

## Local bootstrap

```bash
go build ./cmd/axym
go test ./...
```

## Epic 1 command surface

```bash
./axym collect --dry-run --json
./axym verify --chain --json
./axym verify --bundle ./fixtures/bundles/good --json
```

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
