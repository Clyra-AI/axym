# Changelog

All notable changes to Axym will be documented in this file.

The format is based on Keep a Changelog, and Axym follows semver-style tagged releases for user-visible CLI and contract changes.

## Unreleased

### Added

- [semver:minor] Added exact producer-native Action Contract retention and governed register, signed evidence packet, reducer timeline, graph, replay, redaction, completeness, and bundle drift verification paths.
- Added quarantined Axym v0.2.0 producer fixture coverage joined to exact Wrkr/Gait proposal, activation, lifecycle, gate, schema, and manifest digests.

### Fixed

- Hardened action-contract envelopes, digest-bound causal joins, signature key-ID checks, telemetry verification exit semantics, staged bundle output, and offline fixture regeneration/checks.
- Updated the pinned Proof dependency and current documentation/fixture provenance to `v0.7.0`, while retaining legacy Gait and quarantined producer fixture versions as compatibility fixtures.

## [0.1.4] - 2026-08-25

### Fixed

- Pinned hosted Cosign to v2.6.5 so release checksum signing continues to emit the established separate `.sig` and `.pem` artifacts.

## [0.1.3] - 2026-08-25

### Fixed

- Replaced temporary/action-upload SBOM generation with deterministic pinned Syft output at `dist/axym.spdx.json`, including a non-empty output assertion for hosted releases.

## [0.1.2] - 2026-08-25

### Fixed

- Corrected hosted release checksum verification to run from the `dist` directory, matching GoReleaser's basename entries in `checksums.txt`.
- Hardened downstream release integrity checks to resolve basename checksum manifests from `dist` and require explicit keyless Cosign certificate identity and issuer verification.

## [0.1.1] - 2026-08-25

### Fixed

- Installed pinned Syft before GoReleaser in the hosted release workflow so archive-level SBOM generation succeeds reliably; retained the later aggregate SBOM normalization and verification steps.

## [0.1.0] - 2026-08-25

### Added

- [semver:minor] Added fail-closed consumption of the exact Gait v1.5.0 Action Contract lifecycle evidence set: strict typed parsing, canonical digest and Ed25519 verification, digest-bound lineage/order/replay checks, deterministic evidence-set projection, and fixture-key quarantine. Lifecycle packs require caller-supplied trusted bindings and are never downgraded into ordinary Proof records.
- Bounded Action Contract interoperability: exact Wrkr v3 proposal ingestion, Gait activation binding verification, deterministic conformance classifications, pinned producer fixtures, and the `WRKR_AXYM_ACTION_CONTRACT_CONSUMER` receipt entrypoint.
- Action Contract verification hardening: Go 1.26.6 pin, explicit current-selection/evaluation-time requirements, development-signing quarantine, stable reason codes, descriptor-bound artifact reads, and value-based conformance classification.
- Added the exact released Gait v1.4.0 activation fixture pack at tag commit `a44fdcf`, traceable to PR #170 fixture-generation commit `4509fa7`: six byte-pinned development-signed goldens, public fixture key, and three manifest rejection dispositions.
- `init --sample-pack <dir>` for a supported offline first-value path that produces non-empty local evidence and compliance results.
- Launch-facing docs that separate `smoke test`, `sample proof path`, and `real integration path`.
- Root `LICENSE`, governance assets, issue templates, and PR template for the public OSS baseline.
- [semver:minor] First-class Gait authorization bundle and `--gait-pack` ingest support, plus runtime-control bundle artifacts: `authorization-register.json`, `insurance-evidence-profile.json`, `credential-posture-register.json`, `freeze-window-coverage.json`, `kill-switch-coverage.json`, `enforcement-explain-register.json`, `sandbox-coverage.json`, and `control-maturity.json`.

### Changed

- Pinned `Clyra-AI/proof` to `v0.6.1` and retained deterministic framework adaptation for both legacy controls and scoped evidence-set alternatives without flattening multi-record requirements into false coverage.
- Updated the public first-value contract for Proof `v0.5.0`'s expanded evidence-set framework catalog: `4` governance-event captures, `6` total sample records, `6/10` covered controls, four partial controls, grade `C`, and truthful `complete=false` / `weak_record_count=1` messaging.
- Corrected public install guidance so Homebrew users verify with `axym version --json` while source builds and unpacked release binaries use `./axym version --json`.
- Clarified contributor full-gate and release-local prerequisites, plus the explicit private and fallback public security reporting paths.
- Added an authoritative public `record add` contract under `schemas/v1/record/`, normalized compatibility-only `record_version: "1.0"` inputs to canonical `v1`, and locked `schema_violation` JSON behavior for payload contract failures.
- Reconciled release docs and contracts with the hosted workflow: GoReleaser `v2.14.1`, local `dist/local-cosign.pub` verification for maintainer gates, and GitHub OIDC plus `dist/checksums.txt.pem` for hosted tag releases.
- [semver:minor] Extended `verify --bundle` to recompute declared insurer-facing runtime-control artifacts and fail closed on drift, schema violations, or missing required Gait/Wrkr/Proof evidence classes.
