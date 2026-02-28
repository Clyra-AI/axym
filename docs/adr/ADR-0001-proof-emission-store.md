# ADR-0001: Deterministic Proof Emission and Append-Only Local Store

## Status
Accepted (2026-02-28)

## Context
Epic 1.2 requires deterministic proof emission using `Clyra-AI/proof` primitives, idempotent re-ingest behavior, and fail-closed handling for sink failures in compliance mode.

## Decision
- Use a dedicated proof-emission layer (`core/proofemit`) that is authoritative for record creation + append behavior.
- Persist an append-only local chain in `core/store` with atomic write+rename and directory `fsync` in compliance mode.
- Use a TTL-bounded dedupe index keyed by `source_product + record_type + event_hash`.
- Sign each emitted record before append using a persistent local signing key.
- Route sink failures through explicit policy decisions (`fail_closed`, `advisory_only`, `shadow`) via `core/policy/sink`.

## Alternatives Considered
- Append directly from collectors: rejected because it collapses architecture boundaries and weakens testability.
- In-memory dedupe only: rejected because restart/re-ingest idempotency would be unreliable.
- Fail-open default on sink failures: rejected because compliance mode requires evidence-loss budget 0.

## Tradeoffs
- Local state management adds complexity and more hardening tests.
- Compliance mode can return failures in cases where advisory mode would continue.
- Chain replay checks for dedupe fallback increase append cost but preserve idempotency if index persistence fails.

## Rollback Plan
- Revert proof emission/store packages and restore previous command behavior.
- Keep schema and contract tests as guardrails while iterating on an alternative persistence strategy.

## Validation Plan
- `go test ./core/proofemit/... -count=1`
- `make test-hardening`
- `make test-chaos`
- `make prepush-full`
