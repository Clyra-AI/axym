# Axym

## Commands

- `axym collect --dry-run --json`: validates deterministic would-capture output with no writes.
- `axym collect --json`: runs built-in collectors and appends signed proof records.
- `axym collect --json --plugin "<cmd>"`: runs third-party collector protocol (`stdin` config, `stdout` JSONL).
- `axym collect --json --governance-event-file <file.jsonl>`: ingests governance events and promotes valid events to proof records.
- `axym verify --chain --json`: verifies local append-only chain integrity.
- `axym verify --bundle <path> --json`: delegates cryptographic bundle checks to `Clyra-AI/proof`.

## Exit codes

- `0` success
- `1` runtime failure
- `3` policy/schema violation
- `2` verification failure
- `6` invalid input
- `8` unsafe operation blocked
