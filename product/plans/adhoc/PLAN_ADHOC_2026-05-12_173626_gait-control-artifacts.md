# PLAN ADHOC: Gait Control Artifact Source-of-Truth Ingest

Date: 2026-05-12
Slug: gait-control-artifacts
Source of truth: user-provided recommendations, `AGENTS.md`, `product/axym.md`, `product/dev_guides.md`, `product/architecture_guides.md`, `factory/profiles/axym.yaml`
Scope: Axym OSS CLI only. This plan changes Axym's runtime-control direction from reconstructing enforcement semantics to consuming Gait first-class control artifacts as authoritative evidence.

Repository path note: all Axym paths below are relative to `/Users/tr/Clyra/axym`. User-supplied upstream paths under Gait, Wrkr, and Proof are treated as cross-product contract references, not implementation paths in this repository.

---

## Global Decisions (Locked)

- Gait owns enforcement semantics for authorization, scoped authority, JIT credentials, freeze windows, kill switches, sandbox policy, and structured policy explain output.
- Wrkr owns posture, inventory, ownership, and identity/inventory evidence where those fields come from system discovery.
- Proof owns record format, canonicalization, signature verification, chain integrity, and portable bundle verification primitives.
- Axym owns deterministic correlation, compliance mapping, completeness scoring, gap detection, and auditor/insurer packaging.
- Axym must not invent runtime-control decisions when a Gait authorization bundle or Gait control artifact is available. It may report missing, stale, unverifiable, or incomplete source artifacts with explicit reason codes.
- Default collect/map/gaps/bundle/verify behavior remains deterministic, local, and no-egress. No LLM calls are allowed.
- New insurer-facing artifacts are derived artifacts. They must be recomputable from Proof records, Gait authorization bundles/control artifacts, and Wrkr inventory evidence.
- New bundle artifacts are additive `v1` contracts. Legacy proof-only bundles remain cryptographically verifiable, while bundles declaring insurer evidence profiles must pass the stricter recomputation checks.
- Every new artifact must preserve source product, source artifact ID, proof record refs, verification status, provenance, and deterministic gap reason codes.
- Structured parsing is required for Gait/Wrkr/Proof JSON artifacts. Regex-only extraction is forbidden for structured payloads.
- New schemas live under `schemas/v1/` for Axym-owned outputs. Upstream Gait/Wrkr/Proof schemas are consumed as external contracts and covered by fixtures.
- New public CLI/bundle JSON surfaces require docs, schema fixtures, contract tests, changelog entries, and semver markers.

---

## Current Baseline (Observed)

Observed in this checkout on 2026-05-12:

- Axym has a buildable Go CLI scaffold under `cmd/axym/` and core packages under `core/`.
- Gait ingest exists under `core/ingest/gait/`, including `pack/read.go` and `translate/translate.go`.
- Current Gait pack reading supports `proof_records.jsonl` and `native_records.jsonl` from directories, zip files, or explicit JSONL paths.
- Current native Gait translation supports native types `trace`, `approval_token`, and `delegation_token`, mapping them to Proof record types `tool_invocation`, `approval`, and `policy_enforcement`.
- Current bundle assembly emits `chain.json`, `raw-records.jsonl`, `chain-verification.yaml`, `record-signing-key.json`, `auditability-grade.yaml`, identity governance artifacts, `executive-summary.json`, `executive-summary.pdf`, `retention-matrix.json`, `boundary-contract.md`, optional overrides, and OSCAL.
- Current bundle verification recomputes executive summary compliance, identity governance artifacts, grade, OSCAL schema validity, Proof chain integrity, bundle signature, and record signatures.
- Current identity governance artifacts summarize actor/downstream/owner/policy/approval/delegation fields but do not emit insurer-specific runtime-control registers.
- Current review logic has a freeze exception class from Axym policy reason codes but does not consume Gait freeze-window state as the source of truth.
- There is no first-class Axym schema or derived artifact for Gait authorization bundles, insurance evidence profile, credential posture, freeze-window coverage, kill-switch coverage, enforcement explain, sandbox coverage, or trust graduation/control maturity.
- `CHANGELOG.md` has an `Unreleased` section with `Added` and `Changed` headings available for the follow-up implementation PR.

---

## Exit Criteria

This plan is complete when:

1. Axym ingests Gait `gate_authorization_bundle` or PackSpec authorization profile artifacts as authoritative runtime-control evidence.
2. Axym emits `authorization-register.json` from Gait authorization bundles with verification metadata, linked proof refs, source product, provenance, and deterministic status.
3. Axym emits `insurance-evidence-profile.json` as a derived insurer checklist over Gait authorization/control artifacts, Wrkr inventory evidence, and Proof verification results.
4. Axym emits `credential-posture-register.json` for standing credential blocked, JIT required, broker source, issuer, TTL, scope, and binding proof.
5. Axym emits `freeze-window-coverage.json` and `kill-switch-coverage.json` from Gait policy/control artifacts without implementing freeze or stop semantics.
6. Axym emits `enforcement-explain-register.json` from structured Gait explain output and links it deterministically to policy enforcement records and authorization bundles.
7. Axym emits `sandbox-coverage.json` for high-risk execution paths such as `proc.exec` and generated-code execution.
8. Axym emits `control-maturity.json` for trust stages across paths using Gait trust metadata and Wrkr posture state.
9. `axym verify --bundle <artifact-path> --json` recomputes every declared insurer-facing artifact and fails closed on drift, invalid schemas, signature failures, or required evidence classes missing from declared profiles.
10. New artifacts are byte-stable for the same inputs except explicit timestamp/version fields.
11. New CLI JSON, schemas, docs, changelog entries, and test fixtures are aligned.

---

## Public API and Contract Map

Inputs consumed:

- Gait authorization source: `gate_authorization_bundle` or PackSpec authorization profile.
- Gait proof source: `proof_records.jsonl` and native translated records.
- Gait control sources: credential evidence, freeze-window decisions, kill-switch state, policy explain output, sandbox policy metadata, trust-graduation metadata.
- Wrkr source: inventory and ownership/posture evidence from Wrkr proof records or inventory artifacts.
- Proof source: record validation, chain verification, bundle manifest verification, signature verification, canonical hash/digest semantics.

Axym-owned outputs:

- `authorization-register.json`: bundle-level authorization correlation and verification register.
- `insurance-evidence-profile.json`: insurer checklist with status, source product, evidence refs, and gaps.
- `credential-posture-register.json`: high-risk credential posture and JIT coverage.
- `freeze-window-coverage.json`: Gait freeze evaluation/enforcement/bypass/missing summary.
- `kill-switch-coverage.json`: generalized kill-switch state and enforcement summary.
- `enforcement-explain-register.json`: structured allow/block/approval reasons and proof refs.
- `sandbox-coverage.json`: execution isolation posture for high-risk execution paths.
- `control-maturity.json`: trust-stage rollout and drift summary.

CLI and behavior contracts:

- `axym ingest --gait-pack <path> --json` must report discovered authorization bundle counts and unsupported artifact reason codes.
- `axym bundle --audit <name> --frameworks eu-ai-act,soc2 --json` must include the new artifacts when source evidence exists.
- `axym verify --bundle <artifact-path> --json` must return exit `2` for drift or failed signature/chain verification, exit `3` for schema violations, exit `6` for invalid inputs, and exit `8` for unsafe output operations.
- Existing `axym collect --dry-run --json`, `axym map --frameworks eu-ai-act,soc2 --json`, `axym gaps --frameworks eu-ai-act,soc2 --json`, and `axym verify --chain --json` examples remain valid.

Schema/versioning contracts:

- Axym-derived output schemas are additive `schemas/v1/...` contracts.
- Upstream schema compatibility is fixture-backed and tracked through cross-product integration tests.
- If a Gait/Wrkr/Proof upstream contract changes incompatibly, Axym must reject or classify with explicit reason codes rather than silently reinterpret.

---

## Docs and OSS Readiness Baseline

- Public docs currently describe Axym as the Prove product in See -> Prove -> Control and correctly state that Axym does not replace IAM, PAM, IGA, or enforcement systems.
- `README.md`, `docs/commands/axym.md`, and `docs/operator/integration-model.md` need surgical updates once these artifacts ship.
- `schemas/v1/` already holds Axym-owned schema contracts and should receive new artifact schemas and fixtures.
- `CHANGELOG.md` is ready for implementation entries under `Unreleased`.
- No marketing or enforcement claims should say Axym blocks, freezes, brokers, or sandboxes execution. Public wording must say Axym consumes and verifies Gait/Wrkr/Proof evidence and reports completeness.

---

## Recommendation Traceability

| Rec | Normalized recommendation | Why | Strategic direction | Expected benefit | Planned story |
|---|---|---|---|---|---|
| 1 | Ingest Gait authorization bundles as first-class source evidence. | Gait emits the authoritative runtime-control unit. | Stop reconstructing enforcement from loose records. | Smaller, cleaner, more credible control evidence. | 1.1, 1.2 |
| 2 | Keep insurance evidence profile, but derive it from Gait, Wrkr, and Proof. | Insurer checklist should not become a parallel Axym control model. | Axym packages evidence and gaps. | Commercially useful insurer artifact with source refs. | 2.1 |
| 3 | Emit JIT credential evidence register. | Gait makes replacing standing privilege first-class. | Surface credential posture directly. | Better insurer proof for least privilege and brokered access. | 2.2 |
| 4 | Add freeze-window coverage as consumed Gait policy decision class. | Gait owns deterministic freeze evaluation. | Axym summarizes evaluated/enforced/bypassed/missing state. | Clear change-window control evidence. | 2.3 |
| 5 | Consume generalized kill-switch state and outcomes. | Gait owns stop semantics. | Axym proves coverage and results. | Clear emergency stop evidence without Axym enforcement logic. | 2.3 |
| 6 | Preserve structured enforcement explain output. | Reviewers need machine-readable reasons. | Link explain output to enforcement records and bundles. | Better audit and insurer rationale. | 2.4 |
| 7 | Add sandbox control evidence. | Gait can enforce sandbox metadata for high-risk execution. | Axym reports isolation sufficiency. | Proves execution control coverage for risky paths. | 2.4 |
| 8 | Add trust graduation/control maturity reporting. | Rollout maturity reduces approval fatigue and risk. | Consume Gait trust metadata plus Wrkr posture. | Measurable control maturity by path. | 3.1 |
| 9 | Extend bundle verification for cross-product derived artifacts. | Summary artifacts become commercial claims. | Recompute and fail on drift. | Reproducible insurer and auditor evidence. | 3.2 |

---

## Test Matrix Wiring

Fast lane:

- `make lint-fast`
- `make test-fast`
- Focused package tests for changed parser, translator, artifact builder, and verifier packages.

Core CI lane:

- `make test-contracts`
- `go test ./... -count=1`
- CLI JSON/exit-code tests for ingest, bundle, and verify behavior.

Acceptance lane:

- `make test-acceptance`
- `make test-scenarios`
- End-to-end fixture workflow: Gait authorization bundle + Wrkr inventory + Proof records -> Axym bundle -> Axym verify.

Cross-platform lane:

- `make test-fast`
- `make test-contracts`
- Run on Linux, macOS, and Windows for file ordering, zip ordering, path handling, and JSON byte stability.

Risk lane:

- `make test-hardening`
- `make test-chaos`
- `make test-perf`
- Include malformed packs, partial upstream artifacts, duplicate ingest, stale inventory, signature failures, missing required evidence classes, and bundle artifact drift.

Release/UAT lane:

- `make prepush-full`
- `make test-acceptance`
- `make test-scenarios`
- `axym verify --bundle <artifact-path> --json`
- Release smoke must verify the generated bundle and insurer-facing artifacts before publish.

Gating rule:

- Any story touching schemas, CLI JSON, bundle layout, verification, or source artifact semantics is incomplete until Fast, Core CI, Acceptance, Cross-platform, and relevant Risk lanes are green.
- `axym verify --bundle` drift, schema, chain, signature, or required-evidence failures must be merge-blocking for new insurer-profile bundles.
- Full repo validation for this generated plan is not required; implementation PRs must run the story-specific lanes above.

---

## Minimum-Now Sequence

1. Wave 1: define the Axym input model for Gait authorization bundles, parse PackSpec authorization profiles, and emit `authorization-register.json`.
2. Wave 2: derive insurer-facing coverage artifacts from authoritative sources: insurance profile, credential posture, freeze/kill-switch coverage, enforcement explain, and sandbox coverage.
3. Wave 3: add control maturity, cross-product recomputation in `verify --bundle`, and complete docs/changelog/schema fixtures.

The minimum commercially useful slice is Stories 1.1, 1.2, 2.1, and 3.2: authorization bundles in, insurer profile out, verify recomputes it.

---

## Explicit Non-Goals

- Do not implement Gait enforcement, freeze-window evaluation, kill-switch dispatch blocking, credential brokering, sandbox policy evaluation, or trust-graduation decision logic in Axym.
- Do not implement Wrkr inventory or posture scanning in Axym.
- Do not redefine Proof record format, signing, canonicalization, or verification semantics.
- Do not add default network calls, cloud enrichment, or evidence exfiltration.
- Do not use LLM calls in collect/map/gaps/bundle/verify paths.
- Do not overwrite `product/PLAN_NEXT.md`.
- Do not weaken legacy bundle cryptographic verification to make new optional artifacts pass.

---

## Wave 1: Authoritative Gait Authorization Ingest

### Story 1.1: Detect and parse Gait authorization bundles and PackSpec authorization profiles

Priority: P0

Tasks:

- Add typed Gait authorization input structs that preserve bundle ID, schema/profile version, decision trace refs, approval audit refs, credential evidence refs, action outcome refs, verification metadata, and source file provenance.
- Extend `core/ingest/gait/pack/read.go` to discover authorization bundle entries in directories and zip packs using deterministic sorted ordering.
- Support both explicit `gate_authorization_bundle` JSON artifacts and PackSpec authorization profile manifests without interpreting enforcement semantics beyond typed fields and validation status.
- Add explicit unsupported/invalid reason codes for unknown authorization profile versions, malformed JSON, missing required IDs, and unverifiable linked refs.
- Add fixtures for directory, zip, mixed proof/native/auth bundles, duplicate bundle IDs, and malformed authorization bundles.

Repo paths:

- `core/ingest/gait/pack/read.go`
- `core/ingest/gait/pack/read_test.go`
- `core/ingest/gait/translate/translate.go`
- `core/ingest/gait/ingest.go`
- `schemas/v1/bundle/authorization-register-v1.schema.json`
- `testinfra/contracts/fixtures/gait/`

Run commands:

- `go test ./core/ingest/gait/... -count=1`
- `make test-contracts`
- `make test-scenarios`

Test requirements:

- Unit parser tests for directory and zip discovery with stable ordering.
- Contract fixture tests for valid and invalid authorization bundle shapes.
- Scenario tests proving legacy `proof_records.jsonl` and `native_records.jsonl` packs still ingest.
- Failure tests for malformed JSON, unsupported profile version, missing IDs, duplicate IDs, and unresolved linked refs.

Matrix wiring:

- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Required commands: `go test ./core/ingest/gait/... -count=1`, `make test-contracts`, `make test-scenarios`.

Acceptance criteria:

- Axym reports authorization bundle counts in `axym ingest --gait-pack <path> --json`.
- Pack reader preserves source path and source product metadata for every authorization bundle.
- Mixed Gait packs parse deterministically across OSes.
- Invalid authorization artifacts fail with stable reason codes and exit behavior through the CLI.

Changelog impact: required
Changelog section: Added
Draft changelog entry: Added first-class Gait authorization bundle and PackSpec authorization profile ingest as authoritative runtime-control evidence inputs.
Semver marker override: [semver:minor]
Contract/API impact: Adds new ingest JSON fields for discovered authorization bundles and new Axym-derived authorization register schema.
Versioning/migration impact: Additive v1 schema; legacy Gait proof/native packs remain supported.
Architecture constraints: Stay inside Sibling ingestion and translation; do not leak authorization parsing into compliance matching or bundle verification.
ADR required: yes
TDD first failing test(s): `core/ingest/gait/pack/read_test.go::TestReadDirectoryDetectsAuthorizationBundle` and `TestReadZipOrdersAuthorizationBundlesDeterministically`.
Cost/perf impact: low
Chaos/failure hypothesis: A pack with one malformed authorization artifact and valid proof records must fail closed for the invalid pack rather than silently omitting the bad authorization evidence.

### Story 1.2: Translate linked authorization evidence into Axym chain refs and emit `authorization-register.json`

Priority: P0

Tasks:

- Build a deterministic authorization correlation layer that links Gait authorization bundles to Proof records by bundle ID, trace ID, intent digest, policy digest, approval ref, credential evidence ref, action outcome ref, and record ID.
- Preserve Gait verification metadata without recalculating Gait enforcement results.
- Emit `authorization-register.json` during bundle assembly with one entry per authorization bundle and stable status values such as `verified`, `partial`, `missing_link`, `signature_failed`, and `schema_invalid`.
- Include source product, source artifact path, source schema/profile version, linked proof refs, linked record hashes, and gap reason codes.
- Add schema validation and byte-stability tests for `authorization-register.json`.

Repo paths:

- `core/ingest/gait/translate/translate.go`
- `core/bundle/bundle.go`
- `core/verify/bundle/verify.go`
- `schemas/v1/bundle/authorization-register-v1.schema.json`
- `testinfra/contracts/bundle_contract_test.go`
- `testinfra/contracts/verify_contract_test.go`

Run commands:

- `go test ./core/ingest/gait/... ./core/bundle ./core/verify/bundle -count=1`
- `make test-contracts`
- `make test-acceptance`

Test requirements:

- Unit tests for correlation precedence and stable sorting.
- Contract tests for schema-valid register output.
- Golden tests proving same input yields byte-identical register output.
- Verification tests proving tampered linked refs or register drift fail `verify --bundle` with exit `2`.

Matrix wiring:

- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Required commands: `go test ./core/ingest/gait/... ./core/bundle ./core/verify/bundle -count=1`, `make test-contracts`, `make test-acceptance`.

Acceptance criteria:

- Bundles containing Gait authorization evidence include `authorization-register.json`.
- Every register entry is traceable to source artifact refs and Proof record refs.
- Register output is deterministic and schema-valid.
- `verify --bundle` recomputes the register and fails on drift.

Changelog impact: required
Changelog section: Added
Draft changelog entry: Added `authorization-register.json` to Axym bundles for deterministic Gait authorization bundle correlation and verification.
Semver marker override: [semver:minor]
Contract/API impact: Adds a public bundle artifact and verify recomputation contract.
Versioning/migration impact: Additive artifact; verification remains backward-compatible for legacy bundles that do not declare the authorization register.
Architecture constraints: Bundle assembly may consume translated ingest metadata but must not reinterpret Gait allow/block/approval decisions.
ADR required: yes
TDD first failing test(s): `core/bundle/bundle_test.go::TestBuildEmitsAuthorizationRegisterFromGaitBundles` and `core/verify/bundle/verify_test.go::TestVerifyFailsAuthorizationRegisterDrift`.
Cost/perf impact: medium
Chaos/failure hypothesis: Missing linked records in a declared authorization bundle must produce deterministic partial/gap status and fail verifier only when the declared profile marks the evidence class required.

---

## Wave 2: Insurer-Facing Derived Evidence

### Story 2.1: Derive `insurance-evidence-profile.json` from Gait, Wrkr, and Proof

Priority: P0

Tasks:

- Define Axym-owned insurer checklist output with required entries: `inventory`, `ownership`, `scoped_authority`, `approval_gates`, `kill_switch`, `drift_tracking`, `incident_reconstruction`, `jit_credentials`, `freeze_windows`, `sandboxing`, and `proof_of_enforcement`.
- For each checklist entry emit status, source product, source artifact refs, proof record refs, confidence/completeness score, reason codes, and actionable gaps.
- Prefer source precedence: Gait for runtime controls, Wrkr for inventory/ownership/posture, Proof for verification/chain/signature claims, Axym for derived mapping/gap scoring only.
- Reuse existing identity governance artifacts where possible instead of duplicating ownership/delegation logic.
- Add schema, golden, and verification recomputation tests.

Repo paths:

- `core/bundle/bundle.go`
- `core/verify/bundle/verify.go`
- `core/identitygovernance/artifacts.go`
- `core/compliance/match/matcher.go`
- `schemas/v1/bundle/insurance-evidence-profile-v1.schema.json`
- `docs/operator/integration-model.md`

Run commands:

- `go test ./core/bundle ./core/verify/bundle ./core/identitygovernance ./core/compliance/... -count=1`
- `make test-contracts`
- `make test-scenarios`
- `make test-acceptance`

Test requirements:

- Unit tests for each checklist entry and source-product precedence.
- Contract tests for schema validity and stable reason codes.
- Scenario test with Gait authorization bundle + Wrkr inventory + Proof chain.
- Gap tests for missing inventory, missing JIT evidence, missing kill-switch state, missing sandbox metadata, and failed Proof verification.

Matrix wiring:

- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Required commands: `make test-contracts`, `make test-scenarios`, `make test-acceptance`.

Acceptance criteria:

- `insurance-evidence-profile.json` is emitted for bundles with insurer profile inputs.
- Every checklist item has status, source product, evidence refs, and gaps.
- No checklist item claims coverage without a source artifact or verified proof ref.
- Verification recomputes the profile and fails on drift.

Changelog impact: required
Changelog section: Added
Draft changelog entry: Added an insurer evidence profile derived from Gait authorization/control evidence, Wrkr inventory evidence, and Proof verification results.
Semver marker override: [semver:minor]
Contract/API impact: Adds public bundle artifact and insurer checklist schema.
Versioning/migration impact: Additive v1 artifact; missing source artifacts become explicit gaps, not inferred coverage.
Architecture constraints: Keep insurer scoring in Bundle assembly and verification; do not move checklist semantics into Gait/Wrkr ingest.
ADR required: yes
TDD first failing test(s): `core/bundle/bundle_test.go::TestBuildInsuranceEvidenceProfileFromAuthoritativeSources`.
Cost/perf impact: medium
Chaos/failure hypothesis: If Wrkr inventory is stale or Proof verification fails, the profile must downgrade affected entries and emit reason codes rather than reporting covered controls.

### Story 2.2: Emit `credential-posture-register.json` for standing privilege and JIT controls

Priority: P1

Tasks:

- Extract credential posture facts from Gait broker credential evidence and authorization bundle links.
- Emit entries for standing credential blocked, JIT credential required, broker source, issuer, TTL, scope, binding proof, action refs, and proof refs.
- Add deterministic status values for `jit_verified`, `standing_blocked`, `missing_binding`, `ttl_missing`, `scope_missing`, `broker_missing`, and `issuer_missing`.
- Feed JIT coverage into the `jit_credentials` insurance profile entry.
- Reject or redact raw secret material; store only metadata, digests, refs, TTLs, and scope descriptors.

Repo paths:

- `core/ingest/gait/translate/translate.go`
- `core/bundle/bundle.go`
- `core/verify/bundle/verify.go`
- `schemas/v1/bundle/credential-posture-register-v1.schema.json`
- `testinfra/contracts/bundle_contract_test.go`

Run commands:

- `go test ./core/ingest/gait/... ./core/bundle ./core/verify/bundle -count=1`
- `make test-contracts`
- `make test-hardening`

Test requirements:

- Unit tests for credential evidence extraction and redaction.
- Contract tests for register schema.
- Hardening tests proving secret-like values are not emitted.
- Verification drift tests for TTL/scope/binding proof changes.

Matrix wiring:

- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Required commands: `make test-contracts`, `make test-hardening`.

Acceptance criteria:

- Bundles with Gait credential evidence include `credential-posture-register.json`.
- JIT credential coverage appears in `insurance-evidence-profile.json`.
- Raw credential secrets are never emitted.
- Register drift fails verification.

Changelog impact: required
Changelog section: Added
Draft changelog entry: Added `credential-posture-register.json` for insurer-facing JIT credential and standing privilege evidence.
Semver marker override: [semver:minor]
Contract/API impact: Adds public credential posture artifact and insurance profile source input.
Versioning/migration impact: Additive v1 artifact; missing credential fields become explicit gaps.
Architecture constraints: Axym consumes broker evidence only; Gait remains authoritative for credential issuance and enforcement.
ADR required: no
TDD first failing test(s): `core/bundle/bundle_test.go::TestBuildCredentialPostureRegisterRedactsSecrets`.
Cost/perf impact: low
Chaos/failure hypothesis: Credential evidence containing secret-like fields must produce redacted/digest-only output and fail hardening tests if raw values appear.

### Story 2.3: Emit freeze-window and kill-switch coverage from Gait control artifacts

Priority: P1

Tasks:

- Consume Gait freeze-window explain/trace fields and summarize evaluated, enforced, bypassed-by-approval, missing, and stale states into `freeze-window-coverage.json`.
- Consume Gait generalized kill-switch state refs, blocked dispatch proof, expiry, actor, target, and environment into `kill-switch-coverage.json`.
- Include both artifacts in `insurance-evidence-profile.json` under `freeze_windows` and `kill_switch`.
- Preserve Gait source refs and reason/explain fields without implementing freeze or kill-switch semantics in Axym.
- Add deterministic summary ordering by environment, target kind, target ID, policy digest, and bundle ID.

Repo paths:

- `core/ingest/gait/pack/read.go`
- `core/ingest/gait/translate/translate.go`
- `core/bundle/bundle.go`
- `core/review/review.go`
- `core/verify/bundle/verify.go`
- `schemas/v1/bundle/freeze-window-coverage-v1.schema.json`
- `schemas/v1/bundle/kill-switch-coverage-v1.schema.json`

Run commands:

- `go test ./core/ingest/gait/... ./core/bundle ./core/review ./core/verify/bundle -count=1`
- `make test-contracts`
- `make test-scenarios`
- `make test-hardening`

Test requirements:

- Parser/translation tests for freeze and kill-switch source refs.
- Review tests proving Gait freeze reason classes are surfaced without Axym policy reinterpretation.
- Contract tests for both schemas.
- Scenario tests for enforced, bypassed, expired, missing, and stale source states.

Matrix wiring:

- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Required commands: `make test-contracts`, `make test-scenarios`, `make test-hardening`.

Acceptance criteria:

- Bundles with Gait freeze/kill-switch evidence emit both coverage artifacts.
- Insurance profile entries derive from these artifacts.
- Gait explain/trace fields are preserved as source context.
- Missing or expired kill-switch state is an explicit gap, not inferred as covered.

Changelog impact: required
Changelog section: Added
Draft changelog entry: Added freeze-window and kill-switch coverage artifacts sourced from Gait control evidence.
Semver marker override: [semver:minor]
Contract/API impact: Adds two public bundle artifacts and profile inputs.
Versioning/migration impact: Additive v1 artifacts; existing Axym freeze exception reporting remains compatible.
Architecture constraints: Keep Gait control consumption in Sibling ingestion/translation and Bundle assembly; do not add Axym enforcement decisions.
ADR required: no
TDD first failing test(s): `core/bundle/bundle_test.go::TestBuildFreezeAndKillSwitchCoverageFromGaitArtifacts`.
Cost/perf impact: low
Chaos/failure hypothesis: Expired kill-switch state with blocked dispatch proof must be represented as stale/partial rather than covered.

### Story 2.4: Emit enforcement explain and sandbox coverage registers

Priority: P1

Tasks:

- Preserve structured Gait `gate eval --explain --json` output during ingest.
- Link explain entries to policy enforcement records by trace ID, intent digest, policy digest, authorization bundle ID, and proof record ID.
- Emit `enforcement-explain-register.json` with allow/block/approval decision, missing fields, credential posture, freeze state, kill-switch state, sandbox state, and proof refs.
- Extract sandbox metadata for high-risk execution paths such as `proc.exec` and generated-code execution.
- Emit `sandbox-coverage.json` with network mode, writable paths, env exposure, timeout, filesystem isolation, policy result, and gaps.

Repo paths:

- `core/ingest/gait/translate/translate.go`
- `core/compliance/match/matcher.go`
- `core/gaps/gaps.go`
- `core/bundle/bundle.go`
- `core/verify/bundle/verify.go`
- `schemas/v1/bundle/enforcement-explain-register-v1.schema.json`
- `schemas/v1/bundle/sandbox-coverage-v1.schema.json`

Run commands:

- `go test ./core/ingest/gait/... ./core/compliance/... ./core/gaps ./core/bundle ./core/verify/bundle -count=1`
- `make test-contracts`
- `make test-scenarios`
- `make test-hardening`

Test requirements:

- Unit tests for explain linkage precedence and reason-code stability.
- Contract tests for explain and sandbox schemas.
- Gap ranking tests proving sandbox gaps affect insurer profile and control gaps deterministically.
- Hardening tests for malformed explain payloads and unsafe path fields.

Matrix wiring:

- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Required commands: `make test-contracts`, `make test-scenarios`, `make test-hardening`.

Acceptance criteria:

- `enforcement-explain-register.json` preserves structured reasons from Gait and links them to Proof refs.
- `sandbox-coverage.json` summarizes high-risk execution isolation without Axym evaluating sandbox policy.
- Missing sandbox isolation fields produce explicit gaps in the insurance profile.
- Verification recomputes both artifacts and fails on drift.

Changelog impact: required
Changelog section: Added
Draft changelog entry: Added enforcement explain and sandbox coverage bundle artifacts sourced from structured Gait policy evidence.
Semver marker override: [semver:minor]
Contract/API impact: Adds two public bundle artifacts and expands deterministic gap inputs.
Versioning/migration impact: Additive v1 artifacts; no change to legacy map/gaps behavior unless these source artifacts are present.
Architecture constraints: Compliance matching may consume derived context but must not parse raw Gait payloads directly outside the ingest/translation boundary.
ADR required: no
TDD first failing test(s): `core/ingest/gait/translate/verdict_mapping_test.go::TestExplainOutputLinksToPolicyEnforcementRecord`.
Cost/perf impact: medium
Chaos/failure hypothesis: Ambiguous explain linkage must produce `partial` with reason codes, not attach the explain output to the wrong enforcement record.

---

## Wave 3: Maturity, Verification, and Public Readiness

### Story 3.1: Emit `control-maturity.json` for trust graduation and rollout drift

Priority: P1

Tasks:

- Consume Gait trust-graduation metadata and Wrkr posture state to report per-path trust stages: `observe`, `dry_run`, `read_only_allow`, `approval_gated_write`, `brokered_write`, and `blocked_destructive`.
- Emit current stage, prior stage, drift, source product, source artifact refs, proof refs, remaining gaps, and approval fatigue indicators where source data exists.
- Add deterministic stage ordering and stable drift reason codes.
- Feed maturity gaps into `insurance-evidence-profile.json` where they affect scoped authority, approval gates, JIT credentials, sandboxing, or proof of enforcement.
- Integrate with regression reporting so maturity regressions are visible through `axym regress run`.

Repo paths:

- `core/compliance/context/context.go`
- `core/regress/regress.go`
- `core/bundle/bundle.go`
- `core/verify/bundle/verify.go`
- `schemas/v1/bundle/control-maturity-v1.schema.json`
- `schemas/v1/regress/regress-baseline-v1.schema.json`

Run commands:

- `go test ./core/compliance/context ./core/regress ./core/bundle ./core/verify/bundle -count=1`
- `make test-contracts`
- `make test-scenarios`
- `make test-perf`

Test requirements:

- Unit tests for stage ordering, drift classification, and source precedence.
- Contract tests for control maturity schema and regression baseline compatibility.
- Scenario tests for stage advancement, regression, missing posture state, and remaining gaps.
- Performance test to ensure maturity scoring does not materially slow bundle generation on large chains.

Matrix wiring:

- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Required commands: `make test-contracts`, `make test-scenarios`, `make test-perf`.

Acceptance criteria:

- Bundles with trust/posture evidence include `control-maturity.json`.
- `axym regress run` can detect maturity regressions deterministically when baselines include maturity state.
- Missing source data is represented as a gap and never synthesized as mature control coverage.

Changelog impact: required
Changelog section: Added
Draft changelog entry: Added control maturity reporting for trust-stage rollout and drift using Gait trust metadata and Wrkr posture state.
Semver marker override: [semver:minor]
Contract/API impact: Adds public bundle artifact and extends regression baseline semantics for maturity drift.
Versioning/migration impact: Additive v1 artifact; regression baseline changes must remain backward-compatible for baselines without maturity entries.
Architecture constraints: Keep maturity derivation in Context enrichment, Review/regression evaluation, and Bundle verification boundaries.
ADR required: yes
TDD first failing test(s): `core/regress/regress_test.go::TestRunDetectsControlMaturityRegression`.
Cost/perf impact: medium
Chaos/failure hypothesis: A path that regresses from `brokered_write` to `approval_gated_write` or loses Wrkr posture evidence must produce deterministic drift output and exit `5` in regression runs.

### Story 3.2: Extend `axym verify --bundle` to recompute every declared insurer-facing artifact

Priority: P0

Tasks:

- Add a verifier registry for declared derived artifacts so verification recomputes `authorization-register.json`, `insurance-evidence-profile.json`, `credential-posture-register.json`, `freeze-window-coverage.json`, `kill-switch-coverage.json`, `enforcement-explain-register.json`, `sandbox-coverage.json`, and `control-maturity.json`.
- Compare recomputed artifacts byte-for-byte after canonical deterministic marshaling.
- Fail with exit `2` for drift, failed signatures, chain failures, required evidence classes missing from declared profiles, or unverifiable source refs.
- Fail with exit `3` for schema violations and exit `6` for unreadable/invalid bundle inputs.
- Keep legacy proof-only bundle verification behavior for bundles without declared insurer artifacts.

Repo paths:

- `core/verify/bundle/verify.go`
- `cmd/axym/verify.go`
- `core/bundle/bundle.go`
- `core/verifysupport/support.go`
- `testinfra/contracts/verify_contract_test.go`
- `docs/commands/axym.md`

Run commands:

- `go test ./core/verify/... ./cmd/axym -count=1`
- `make test-contracts`
- `make test-acceptance`
- `make test-hardening`
- `make prepush-full`

Test requirements:

- Contract tests for exit codes and JSON error envelopes.
- Bundle tamper tests for every new derived artifact.
- Signature/chain failure tests using Proof verification.
- Acceptance workflow proving source records plus Gait/Wrkr artifacts recompute to the emitted bundle files.
- Legacy compatibility test for proof-only bundles.

Matrix wiring:

- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk, Release/UAT.
- Required commands: `make test-contracts`, `make test-acceptance`, `make test-hardening`, `make prepush-full`.

Acceptance criteria:

- `axym verify --bundle <artifact-path> --json` verifies all declared insurer-facing artifacts.
- Tampering any declared derived artifact fails verification with exit `2`.
- Schema invalid artifacts fail with exit `3`.
- Legacy bundles without declared insurer artifacts remain cryptographically verifiable.

Changelog impact: required
Changelog section: Changed
Draft changelog entry: Extended `axym verify --bundle` to recompute declared insurer-facing artifacts and fail closed on drift or missing required evidence classes.
Semver marker override: [semver:minor]
Contract/API impact: Changes verification semantics for new bundles that declare insurer-facing artifacts and adds JSON result fields for derived-artifact verification.
Versioning/migration impact: Additive verifier registry with legacy compatibility; no breaking change for old bundles.
Architecture constraints: Verification must consume only bundle artifacts and Proof/Wrkr/Gait source evidence inside the bundle, never live external systems.
ADR required: yes
TDD first failing test(s): `core/verify/bundle/verify_test.go::TestVerifyFailsDerivedArtifactDrift` and `cmd/axym/verify_test.go::TestVerifyBundleReportsDerivedArtifactJSON`.
Cost/perf impact: medium
Chaos/failure hypothesis: If a bundle manifest is valid but a derived artifact no longer matches source records, verification must fail closed with precise artifact and reason code.

### Story 3.3: Align docs, schemas, changelog, and cross-product fixtures

Priority: P1

Tasks:

- Document the new source-of-truth boundary: Gait enforces, Wrkr inventories, Proof verifies, Axym correlates/maps/packages.
- Add command docs for ingest, bundle, and verify JSON changes.
- Add schema README entries and valid/invalid fixtures for every new Axym-owned artifact.
- Add cross-product fixture provenance notes for upstream Gait/Wrkr/Proof contract samples.
- Update `CHANGELOG.md` with story-level entries and semver markers.
- Add docs parity tests for new artifact names, exit codes, and command output examples.

Repo paths:

- `README.md`
- `docs/commands/axym.md`
- `docs/operator/integration-model.md`
- `schemas/v1/`
- `CHANGELOG.md`
- `testinfra/contracts/command_docs_parity_contract_test.go`
- `testinfra/contracts/product_identity_contract_test.go`

Run commands:

- `make lint-fast`
- `make test-contracts`
- `make test-scenarios`
- `make prepush-full`

Test requirements:

- Docs parity tests for command flags, JSON fields, exit codes, and artifact names.
- Schema fixture validation tests for valid and invalid examples.
- Product identity tests ensuring public docs do not claim Axym implements Gait/Wrkr/Proof responsibilities.
- Scenario docs smoke test for the insurer evidence workflow.

Matrix wiring:

- Lanes: Fast, Core CI, Acceptance, Cross-platform, Release/UAT.
- Required commands: `make lint-fast`, `make test-contracts`, `make test-scenarios`, `make prepush-full`.

Acceptance criteria:

- Public docs accurately describe Axym as a consumer and verifier of authoritative Gait/Wrkr/Proof artifacts.
- New artifact schemas and fixtures are discoverable under `schemas/v1/`.
- Changelog entries exist for all user-visible artifact and verify changes.
- Docs parity checks pass.

Changelog impact: required
Changelog section: Changed
Draft changelog entry: Documented the Gait/Wrkr/Proof source-of-truth boundary and the new insurer-facing Axym bundle artifacts.
Semver marker override: [semver:minor]
Contract/API impact: Documents public contract additions and updated verify behavior.
Versioning/migration impact: Docs must state additive v1 artifacts and legacy bundle compatibility.
Architecture constraints: Public docs must preserve Axym scope and avoid enforcement ownership claims.
ADR required: no
TDD first failing test(s): `testinfra/contracts/command_docs_parity_contract_test.go::TestDocsMentionDerivedArtifactVerification`.
Cost/perf impact: low
Chaos/failure hypothesis: Public docs claiming Axym enforces runtime controls should fail product identity contract tests.

---

## Definition of Done

- All stories preserve Axym scope, deterministic behavior, zero default exfiltration, Proof contracts, and exit-code contracts.
- Every new artifact has schema, valid fixtures, invalid fixtures, byte-stability tests, bundle emission tests, and bundle verification drift tests.
- Every new source artifact is parsed structurally with stable reason codes for invalid, missing, stale, unsupported, or unverifiable inputs.
- All public docs and `CHANGELOG.md` are updated in the same implementation PRs as user-visible behavior.
- `make lint-fast`, `make test-fast`, `make test-contracts`, `make test-scenarios`, and story-specific risk lanes pass.
- `make prepush-full` passes before landing the final cross-product verification story.
- Implementation handoff command: `Use $plan-implement with plan_path: product/plans/adhoc/PLAN_ADHOC_2026-05-12_173626_gait-control-artifacts.md`
