# ADR-0004: Deterministic Compliance Mapping, Gap Ranking, and Threshold Enforcement

## Status
Accepted (2026-03-02)

## Context
Epic 4 requires deterministic framework loading, evidence-to-control matching with explainability, gap ranking with remediation/grade output, and fail-closed threshold behavior that excludes invalid evidence classes from coverage.

## Decision
- Introduce compliance-only boundaries:
  - `core/compliance/framework` for strict framework loading/flattening,
  - `core/compliance/context` for deterministic context enrichment + weighting,
  - `core/compliance/match` for explainable per-record and per-control mapping,
  - `core/compliance/coverage` for normalized `covered`/`partial`/`gap` summaries,
  - `core/compliance/threshold` for policy threshold evaluation and invalid-evidence boundary enforcement.
- Add `core/gaps` for deterministic ranking/remediation generation and `core/review/grade` for weakest-link auditability grading.
- Add CLI surfaces:
  - `axym map --frameworks ... [--policy-config|--min-coverage]`
  - `axym gaps --frameworks ... [--policy-config|--min-coverage]`
- Keep deterministic and fail-closed behavior:
  - invalid evidence classes (`invalid_record`, `schema_error`, `mapping_error`) are excluded from valid coverage,
  - threshold failures return non-zero exit (`5`) with typed reason code `COVERAGE_THRESHOLD_NOT_MET` and failing-control details in JSON output.

## Alternatives Considered
- Perform matching inside collectors/ingestors: rejected because it collapses architecture boundaries and mixes evidence acquisition with compliance semantics.
- Treat all syntactically valid records as coverage-eligible: rejected because invalid evidence classes must never inflate compliance posture.
- Best-effort threshold warnings only: rejected because policy gates require deterministic non-zero failure behavior.

## Tradeoffs
- Added package and contract surface area increases test matrix scope.
- Context weighting is deterministic but intentionally heuristic; rationale fields are required to keep outcomes explainable.
- Threshold enforcement can fail workflows earlier when coverage is below policy.

## Rollback Plan
- Remove `map`/`gaps` command surfaces and compliance packages introduced in Epic 4.
- Preserve existing collect/ingest/verify behavior and contracts.
- Keep contract tests for thresholds/invalid-evidence to guard future reintroduction.

## Validation Plan
- `go test ./core/compliance/... -count=1`
- `go test ./core/gaps/... -count=1`
- `go test ./core/review/grade/... -count=1`
- `go test ./cmd/axym -count=1`
- `go test ./internal/integration/compliance/match -count=1`
- `go test ./internal/integration/gaps -count=1`
- `go test ./testinfra/contracts/... -count=1`
- `make test-perf`
- `make prepush-full`
