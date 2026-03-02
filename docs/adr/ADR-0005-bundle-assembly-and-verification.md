# ADR-0005: Deterministic Bundle Assembly and Compliance-Aware Bundle Verification

## Status
Accepted (2026-03-02)

## Context
Epic 5 requires deterministic, signed bundle assembly with safe output-path controls and a `verify --bundle` path that combines proof-level cryptographic checks with Axym compliance-completeness interpretation.

## Decision
- Add bundle assembly boundary `core/bundle` with deterministic artifact generation:
  - `chain.json`
  - `raw-records.jsonl`
  - `chain-verification.yaml`
  - `auditability-grade.yaml`
  - `executive-summary.json`
  - `boundary-contract.md`
  - `retention-matrix.json`
  - `oscal-v1.1/component-definition.json`
- Add `core/export/safety` managed output-path contract:
  - `non-empty + non-managed => fail`,
  - managed marker must be a regular file,
  - unsafe operations fail closed with exit `8`.
- Add deterministic manifest generation boundary `core/export/manifest` and sign bundles via `Clyra-AI/proof` manifest primitives.
- Add OSCAL export boundary `core/export/oscal` with schema-validated output using `schemas/v1/bundle`.
- Add compliance-aware bundle verification boundary `core/verify/bundle`:
  - keep crypto verification delegated to `proof.VerifyBundle`,
  - recompute deterministic compliance completeness and grade from bundle chain evidence,
  - validate OSCAL and executive-summary schema contracts.
- Add CLI surface `axym bundle` and extend `axym verify --bundle` to emit both crypto and compliance verification in one deterministic envelope.

## Alternatives Considered
- Keep bundle verification crypto-only: rejected because auditors require deterministic compliance completeness evidence in the same verification envelope.
- Recompute compliance by trusting only embedded summary fields: rejected because verify must recompute and compare, not trust reported values.
- Allow unmanaged non-empty output directories: rejected due unsafe overwrite and evidence integrity risk.

## Tradeoffs
- Bundle artifact surface area and contract tests increase maintenance cost.
- Compliance-aware verify introduces additional strictness and can fail on malformed/contract-drift artifacts even when crypto hashes pass.
- Deterministic fixed timestamp strategy avoids time-based drift but removes "generation time" as a runtime signal from default artifacts.

## Rollback Plan
- Remove `bundle` command and revert `verify --bundle` to crypto-only behavior.
- Remove Epic 5 export packages and bundle schemas.
- Preserve chain verify and existing map/gaps contracts.

## Validation Plan
- `go test ./core/export/safety/... -count=1`
- `go test ./core/export/manifest/... -count=1`
- `go test ./core/bundle/... -count=1`
- `go test ./core/verify/bundle/... -count=1`
- `go test ./cmd/axym -count=1`
- `go test ./internal/integration/bundle -count=1`
- `go test ./internal/e2e/bundleverify -count=1`
- `go test ./testinfra/contracts/... -count=1`
- `make prepush-full`
