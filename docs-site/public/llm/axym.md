# Axym

## Commands

- `axym collect --dry-run --json`: validates deterministic would-capture output with no writes.
- `axym collect --json`: runs built-in collectors and appends signed proof records.
- `axym collect --json --plugin "<cmd>"`: runs third-party collector protocol (`stdin` config, `stdout` JSONL).
- `axym collect --json --governance-event-file <file.jsonl>`: ingests governance events and promotes valid events to proof records.
- `axym map --frameworks eu-ai-act,soc2 --json`: deterministically maps chain evidence to framework controls and emits explainable match rationale.
- `axym gaps --frameworks eu-ai-act,soc2 --json`: computes deterministic `covered`/`partial`/`gap` ranking, remediation guidance, and auditability grade.
- `axym map --json` and `axym gaps --json`: default to frameworks `eu-ai-act,soc2` when `--frameworks` is omitted.
- `axym map --frameworks eu-ai-act --min-coverage 0.80 --json`: enforces threshold policy and exits non-zero when coverage is below threshold.
- `axym verify --chain --json`: verifies local append-only chain integrity.
- `axym verify --bundle <path> --json`: delegates cryptographic bundle checks to `Clyra-AI/proof`.

## Exit codes

- `0` success
- `1` runtime failure
- `2` verification failure
- `3` policy/schema violation
- `5` regression drift (including threshold failures in `map`/`gaps`)
- `6` invalid input
- `8` unsafe operation blocked
