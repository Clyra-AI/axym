# ADR-0003: Deterministic Sibling Ingest, Translation, and Session Gap Stitching

## Status
Accepted (2026-02-28)

## Context
Epic 3 requires deterministic Wrkr and Gait ingestion, translation contracts for Gait native pack types, idempotent re-ingest behavior, and explicit session-gap signaling without mutating source records.

## Decision
- Add `axym ingest` CLI surface with deterministic source routing (`wrkr`, `gait`) and stable JSON output.
- Implement Wrkr ingest in `core/ingest/wrkr` with:
  - proof-record passthrough for supported types,
  - idempotent append via existing chain dedupe,
  - persisted drift baseline state in `.axym/wrkr-last-ingest.json`,
  - lock-guarded state updates (`.axym/wrkr-last-ingest.lock`),
  - deterministic privilege-drift gap generation in `core/review/privilegedrift`.
- Implement Gait ingest in `core/ingest/gait` with:
  - PackSpec readers for zip, extracted directory, and explicit file paths,
  - deterministic native-type translation (`trace`, `approval_token`, `delegation_token`) in `core/ingest/gait/translate`,
  - passthrough ingest for `proof_records.jsonl` when present,
  - relationship envelope preservation through translation.
- Add session-boundary stitching in `core/ingest/stitch`:
  - detect discontinuity windows with exact start/end timestamps,
  - emit `CHAIN_SESSION_GAP` signals deterministically,
  - grade review-facing signals in `core/review/sessiongap`.

## Alternatives Considered
- Fold sibling ingest into collector runtime: rejected because ingestion and collection boundaries differ and testability/regression isolation would degrade.
- Treat Gait native payloads as opaque passthrough only: rejected because required deterministic type mapping would be missing.
- Infer session continuity heuristically with non-deterministic tolerances: rejected because audit-grade reproducibility requires exact window computation.

## Tradeoffs
- Additional package and test-matrix surface area.
- More local state management for Wrkr ingest lock/state lifecycle.
- Session-gap signaling is conservative and may surface more explicit operational gaps that require downstream triage.

## Rollback Plan
- Remove `ingest` command and new ingest packages.
- Retain existing collect/verify surfaces and chain contracts.
- Keep newly added tests as guardrails for iterative reintroduction.

## Validation Plan
- `go test ./core/ingest/wrkr/... -count=1`
- `go test ./core/ingest/gait/... -count=1`
- `go test ./core/ingest/stitch/... -count=1`
- `go test ./internal/integration/ingest/... -count=1`
- `make test-hardening`
- `make test-contracts`
- `make prepush-full`
