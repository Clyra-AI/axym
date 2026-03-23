# Axym Manual Record Contract

This directory is the authoritative public contract for `axym record add --input <record.json> --json`.

Axym validates this input envelope locally, normalizes compatibility-only `record_version: "1.0"` inputs to canonical `record_version: "v1"`, then signs and appends the record to the local chain. Shared proof-record semantics, record-type-specific validation, hashing, and signature primitives remain owned by `Clyra-AI/proof`.

## Contract files

- Schema: [`manual-input.schema.json`](./manual-input.schema.json)
- Canonical examples:
  - [`examples/decision.v1.json`](./examples/decision.v1.json)
  - [`examples/approval.v1.json`](./examples/approval.v1.json)
- Internal normalized collector contract: [`normalized-input.schema.json`](./normalized-input.schema.json)

## Version behavior

- Canonical Axym-authored examples use `record_version: "v1"`.
- Legacy documented `record_version: "1.0"` inputs remain compatibility-only and normalize to `v1` before append.
- If `record_version` is omitted, Axym defaults it to `v1` before validating the manual-input contract.

## Error contract

- Contract violations return exit code `3`.
- `--json` failures use `error.reason: "schema_violation"`.
- File-read or malformed-JSON input problems remain `invalid_input` with exit code `6`.
