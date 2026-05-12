# Axym Bundle Schemas

Axym-owned bundle artifact schemas live in this directory.

Runtime-control artifact contracts introduced by the Gait source-of-truth ingest flow:

- `authorization-register-v1.schema.json`
- `insurance-evidence-profile-v1.schema.json`
- `credential-posture-register-v1.schema.json`
- `freeze-window-coverage-v1.schema.json`
- `kill-switch-coverage-v1.schema.json`
- `enforcement-explain-register-v1.schema.json`
- `sandbox-coverage-v1.schema.json`
- `control-maturity-v1.schema.json`

Ownership boundary:

- Gait enforces runtime controls and emits the authoritative authorization/control artifacts.
- Wrkr inventories posture, ownership, and system-discovery evidence.
- Proof verifies signatures, chains, and portable bundle integrity.
- Axym correlates those inputs into deterministic bundle artifacts and verifies their recomputation.

These schemas are additive `v1` contracts. Legacy bundles without these artifacts remain verifiable, while bundles that declare them must pass recomputation during `axym verify --bundle`.
