# ADR-0006: Gait Runtime-Control Artifacts as Source-of-Truth Evidence

## Status
Accepted (2026-05-12)

## Context
Axym previously ingested Gait proof/native records but did not treat Gait authorization bundles and structured runtime-control artifacts as first-class evidence. That left bundle outputs reconstructing runtime-control posture from partial signals instead of consuming the authoritative control source.

The repository plan `product/plans/adhoc/PLAN_ADHOC_2026-05-12_173626_gait-control-artifacts.md` requires Axym to:

- ingest Gait authorization bundles and control artifacts deterministically,
- preserve Gait provenance and verification metadata,
- emit insurer-facing derived artifacts from Gait, Wrkr, and Proof evidence,
- recompute those artifacts during `verify --bundle`,
- keep Gait enforcement, Wrkr inventory, and Proof verification ownership outside Axym.

## Decision
- Add `--gait-pack` as the explicit Gait source-of-truth ingest path.
- Extend Gait pack reading to detect:
  - `gate_authorization_bundle`
  - PackSpec authorization profiles
  - structured control artifacts for credential posture, freeze windows, kill switches, explain output, sandbox coverage, and trust graduation
- Translate those Gait source artifacts into signed Axym proof records while preserving:
  - source product
  - source artifact ID
  - source artifact path
  - schema/profile version
  - Gait verification status and reason codes
  - explicit linked proof-record references
- Build new bundle artifacts from translated Gait records plus Wrkr and Proof evidence:
  - `authorization-register.json`
  - `insurance-evidence-profile.json`
  - `credential-posture-register.json`
  - `freeze-window-coverage.json`
  - `kill-switch-coverage.json`
  - `enforcement-explain-register.json`
  - `sandbox-coverage.json`
  - `control-maturity.json`
- Recompute every declared derived artifact during `verify --bundle` and fail closed on:
  - byte drift
  - schema violations
  - missing declared artifacts
  - unexpected declared artifacts
- Extend regression baselines so `regress run` can detect control-maturity regressions, not only coverage regressions.

## Alternatives Considered
- Keep Gait source artifacts outside the chain and only bundle raw sidecars.
  Rejected because verification and regression would need a second persistence path and would weaken deterministic local recomputation.
- Reconstruct runtime-control posture only from generic proof/native records.
  Rejected because Gait already owns the authoritative control unit and Axym should not infer enforcement semantics when first-class artifacts exist.
- Merge insurer-facing logic directly into compliance mapping.
  Rejected because it would blur the bundle/verification boundary and complicate deterministic artifact recomputation.

## Tradeoffs
- More translated Gait records and more public bundle artifacts increase surface area and test burden.
- Bundle verification now performs deeper recomputation work for bundles that declare runtime-control artifacts.
- Regression baselines become richer, but remain additive and backward-compatible.

## Rollback Plan
- Remove `--gait-pack` and the translated Gait source-artifact path.
- Stop emitting the new runtime-control bundle artifacts.
- Revert `verify --bundle` to legacy proof-only bundle verification for those surfaces.
- Leave Wrkr ingest, baseline bundle integrity checks, and existing identity-governance artifacts intact.

## Validation Plan
- `go test ./core/ingest/gait/... -count=1`
- `go test ./core/bundle ./core/verify/bundle ./core/regress ./cmd/axym -count=1`
- `make test-contracts`
- `make test-scenarios`
- `make test-acceptance`
- `make prepush-full`
