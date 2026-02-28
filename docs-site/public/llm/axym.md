# Axym

## Commands

- `axym collect --dry-run --json`: validates local collection readiness.
- `axym verify --chain --json`: verifies local append-only chain integrity.
- `axym verify --bundle <path> --json`: delegates cryptographic bundle checks to `Clyra-AI/proof`.

## Exit codes

- `0` success
- `2` verification failure
- `6` invalid input
- `8` unsafe operation blocked
