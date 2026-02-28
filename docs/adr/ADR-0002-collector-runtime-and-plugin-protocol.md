# ADR-0002: Deterministic Collector Runtime, Data-Pipeline Policies, and Plugin Promotion

## Status
Accepted (2026-02-28)

## Context
Epic 2 requires deterministic collector acquisition across built-in surfaces, data-pipeline policy semantics (SoD/freeze/enrichment), and an external plugin protocol with strict schema rejection.

## Decision
- Introduce a dedicated collector boundary (`core/collector`) and deterministic registry ordering.
- Implement collect orchestration in `core/collect` so collectors remain adapters and proof emission remains in `core/proofemit`.
- Add built-in collectors for MCP, LLM middleware, webhook, GitHub Actions, git metadata, dbt, and Snowflake.
- Keep data-pipeline policy evaluation in policy layer packages (`core/policy/sod`, `core/policy/freeze`) and surface deterministic `decision.pass` + reason codes.
- Enforce digest-first semantics for SQL/query capture and attach replay inputs in metadata.
- Add plugin runtime (`stdin` config, `stdout` JSONL, timeout/isolation, deterministic error classification).
- Add governance-event promotion path with strict JSON Schema validation before proof-record construction.

## Alternatives Considered
- Let collectors append directly to chain: rejected because it collapses architecture boundaries and weakens fail-closed reasoning.
- Accept plugin output as already-valid proof records: rejected because it could bypass normalization/schema gates.
- Fail whole collect command on any collector failure: rejected because non-blocking collection mode is required.

## Tradeoffs
- More package surface area and test matrix complexity.
- Plugin protocol remains intentionally strict; malformed streams are rejected entirely.
- Non-blocking collector failures require richer summary/reporting contracts.

## Rollback Plan
- Revert collector runtime packages and restore prior collect stub behavior.
- Keep schema and contract tests in place to safely iterate on replacement designs.

## Validation Plan
- `go test ./core/collect/... -count=1`
- `go test ./internal/integration/collect -count=1`
- `go test ./internal/integration/datapipeline -count=1`
- `go test ./internal/e2e/plugin -count=1`
- `make test-hardening`
- `make test-chaos`
- `make prepush-full`
