# PLAN Axym: Identity-Governed Action Proof for Software Delivery

Date: 2026-03-19
Source of truth: user-provided 2026-03-19 recommendations, `product/axym.md`, `product/dev_guides.md`, `product/architecture_guides.md`, `AGENTS.md`
Scope: Axym OSS CLI only. Plan the minimum backlog required to shift Axym from action-first proof language to portable proof of identity-governed action in software delivery, without weakening determinism, offline-first defaults, fail-closed behavior, schema stability, or exit-code stability.

---

## Global Decisions (Locked)

- Keep the wedge inside software delivery. This backlog does not widen Axym into a general-purpose IAM, PAM, or IGA product.
- Reframe Axym as proof of which non-human identity acted, through which delegated chain, against which target, under which policy and approval, not merely proof of what an AI system did.
- Preserve Axym's non-negotiables: deterministic collect/map/gaps/bundle/verify behavior, offline-first defaults, local evidence handling, stable `--json` envelopes, and stable exit codes.
- Model identity-governance additively on top of existing `Clyra-AI/proof` semantics. Prefer canonical field placement inside existing `event`, `metadata`, and `relationship` extension points over inventing a second Axym-only record format.
- The public contract must still expose named normalized fields such as `actor_identity`, `downstream_identity`, `delegation_chain`, `policy_digest`, `approval_token_ref`, and `owner_identity` or an equivalent owner pointer.
- Treat architecture boundaries as enforcement points: source adapters and collection, normalization and record construction, proof emission, sibling ingestion and translation, compliance matching, gap/review/regression evaluation, and bundle/export/verification remain separate.
- Missing identity linkage is not "just another event." When Axym cannot tie an action to a known non-human identity, owner, policy digest, or approval chain, the evidence must downgrade deterministically to `partial` or `gap`.
- Do not add speculative sibling-product scope. There is no `agnt` integration surface in this repo today; Agnt alignment in this backlog must land through generic proof-format, governance-event, or manual-record compatibility rather than a new Axym-owned product integration.
- Bundle outputs must make the boundary explicit: Axym proves portable action-governance evidence around IAM/PAM/IGA systems, while those systems remain authoritative for identity lifecycle, credential issuance, entitlement management, and interactive access control.
- Public docs and examples must never position Axym as an IAM replacement or imply identity-governance coverage beyond the shipped software-delivery seam.

---

## Current Baseline (Observed)

- `product/axym.md` already contains identity-adjacent primitives deeper in the document: relationship-envelope preservation, Gait approval/delegation token translation, Wrkr privilege-drift mapping, deterministic auditability grades, and `boundary-contract.md` in the bundle.
- The top-level framing is still action-first. The Executive Summary and JTBD describe structured proof of AI system behavior more strongly than proof of identity-governed action.
- The evidence-surface tables mention permissions and human approvals, but they do not yet elevate acting identity, downstream execution identity, owner/approver, delegation chain, policy digest binding, approval-token binding, and identity/privilege drift as first-class capture targets.
- The proof-record example does not explicitly show `actor_identity`, `downstream_identity`, `delegation_chain`, `policy_digest`, `approval_token_ref`, or `owner_identity`, even though later sections discuss related lineage concepts.
- The bundle section already includes `auditability-grade.yaml` and `boundary-contract.md`, but it does not foreground identity-governance artifacts such as an identity-chain summary, owner/approver register, privilege-drift report, or delegated-chain exceptions view.
- Runtime already has nearby implementation surfaces for this seam: `core/normalize`, `core/record`, `core/collect/governanceevent`, `core/ingest/wrkr`, `core/ingest/gait/translate`, `core/compliance/match`, `core/gaps`, `core/review/grade`, `core/bundle`, and `core/verify`.
- Wrkr ingest already computes deterministic privilege-drift gaps in `core/review/privilegedrift`, and Gait ingest already preserves the `relationship` envelope during translation.
- `cmd/axym/ingest` currently supports `wrkr` and `gait` only. Generic producer inputs currently enter Axym through `collect --governance-event-file` and `record add`, which is the correct in-scope landing point for Agnt-compatible inputs in this backlog.
- Current grade derivation in `core/review/grade/grade.go` is driven by global `covered` / `partial` / `gap` counts. It does not yet appear to apply an explicit identity-linkage penalty model.
- The repo already has relevant fixtures and acceptance surfaces to extend: `fixtures/governance/context_engineering.jsonl`, `fixtures/ingest/wrkr/proof_records.jsonl`, `fixtures/ingest/gait/native_records.jsonl`, `fixtures/records/decision.json`, `scenarios/axym/**`, and `testinfra/acceptance`.

---

## Exit Criteria

1. Axym's source-of-truth product framing explicitly states that Axym proves which non-human identity acted, through which delegated chain, against which target, under which policy and approval.
2. Axym has one additive normalized identity-governance view across native collection, Wrkr ingest, Gait translation, and Agnt-compatible governance-event/manual/proof-record inputs.
3. The normalized view covers: who initiated, which identity executed, which target was touched, which owner/approver was responsible, which delegation chain applied, which policy digest applied, and which approval token bound the action when applicable.
4. Missing identity linkage deterministically downgrades evidence quality, control coverage, and auditability grade with stable reason codes and stable `--json` behavior.
5. Audit bundles include dedicated identity-governance artifacts: identity-chain summary, ownership/approver register, privilege-drift report, delegated-chain exceptions, and a clear Axym-vs-IAM/PAM/IGA boundary note.
6. `map`, `gaps`, `review`, `regress`, `bundle`, and `verify` surfaces remain deterministic and explain weak identity linkage as weak evidence rather than silently counting it as full coverage.
7. Public docs, operator docs, sample assets, and docs-site content reflect the identity-governed action seam truthfully without widening Axym into an identity product.
8. Cross-product and scenario tests cover Wrkr + Gait + Axym interoperability and an Agnt-compatible input path without adding out-of-scope sibling-product logic.
9. CI wiring enforces the new contract in Fast, Core CI, Acceptance, Cross-platform, and Risk lanes where applicable.

---

## Recommendation Traceability

| Recommendation | Why | Strategic direction | Expected moat/benefit | Planned stories |
|---|---|---|---|---|
| Reframe the Executive Summary and JTBD around which non-human identity acted, through which delegated chain, under which policy and approval | Current top-level framing is still behavior-first and undersells the identity-governance seam | Make Axym the proof layer for governed action in software delivery | Sharper market wedge; clearer differentiation from telemetry-only tools | `W1-S1`, `W3-S1` |
| Add identity-bearing signals as first-class evidence targets: acting identity, downstream identity, owner/approver, delegated chain, policy digest and approval binding, identity/privilege drift | Permissions and approvals are present today, but the full accountability chain is not foregrounded strongly enough | Normalize one explainable accountability view across runtime and sibling ingest | Stronger compliance narrative and more defensible audit evidence | `W1-S1`, `W1-S2`, `W2-S1` |
| Show explicit identity-chain fields in the proof-record example | The current example is still tool-event-centric | Turn identity chain from implied concept into concrete contract | Makes the record format memorable, implementable, and auditable | `W1-S1`, `W1-S2`, `W3-S1` |
| Add dedicated identity-governance outputs to the audit bundle | Current bundle structure is strong but does not foreground identity lineage | Make bundles auditor-ready for action governance, not only activity evidence | More compelling bundle deliverable and clearer audit handoff | `W2-S2`, `W3-S1`, `W3-S2` |
| Add explicit identity-chain normalization across Wrkr, Gait, and Agnt-compatible inputs | Gait delegation ingestion exists, but the unified "who initiated / who executed / what was touched / what policy applied" view is not explicit enough | Use Axym as the normalizer, not an identity-system replacement | Higher interoperability moat across the Clyra loop and adjacent agent producers | `W1-S2`, `W3-S2` |
| Missing identity linkage must lower auditability grade and weaken coverage | Unlinked actions should not count like fully governed evidence | Make evidence strength deterministic and explainable | Harder-to-game coverage metrics; better auditor trust | `W2-S1`, `W2-S2` |
| Add a small non-goal that Axym does not replace IAM / PAM / IGA | Prevent category drift and avoid overclaiming | Keep Axym in the proof/evidence lane | Protects positioning and reduces expectation mismatch | `W1-S1`, `W3-S1` |
| Do not widen the wedge beyond software delivery or reposition Axym as an identity product | The recommendation is a seam refinement, not a new category | Tighten focus while expanding proof depth | Higher credibility and lower implementation risk | `W1-S1`, `W3-S1` |

---

## Test Matrix Wiring

Lane definitions:

- Fast lane: `make lint-fast`, targeted package/unit tests, schema/contract tests in `testinfra/contracts`, and docs consistency checks when story scope touches docs.
- Core CI lane: `make test-fast`, targeted `internal/integration/...` suites, affected CLI contract tests, and docs-storyline checks for any user-visible workflow changes.
- Acceptance lane: `go test ./testinfra/acceptance -count=1` plus relevant `scenarios/axym/**` suites for runnable end-to-end flows such as collect -> map -> gaps -> bundle -> verify and mixed-source ingest -> bundle -> verify.
- Cross-platform lane: GitHub Actions matrix validation on Linux, macOS, and Windows for stories that change CLI output contracts, path behavior, sample-pack behavior, or bundle artifact layout.
- Risk lane: `make prepush-full`, `make codeql`, targeted hardening/determinism checks, and nightly-risk workflow coverage for stories that change schema semantics, bundle completeness, grading, or required evidence rules.

Story-to-lane map:

| Story | Fast | Core CI | Acceptance | Cross-platform | Risk |
|---|---|---|---|---|---|
| W1-S1 | Yes | Yes | No | No | No |
| W1-S2 | Yes | Yes | Yes | No | Yes |
| W2-S1 | Yes | Yes | Yes | No | Yes |
| W2-S2 | Yes | Yes | Yes | Yes | Yes |
| W3-S1 | Yes | Yes | Yes | No | No |
| W3-S2 | Yes | Yes | Yes | Yes | Yes |

Merge/release gating rule:

- Any story touching runtime contract, schema, grading, or bundle semantics must pass its Fast and Core CI lanes before merge.
- Any story marked `Acceptance: Yes` is incomplete until the documented identity-governed action flow passes end-to-end in `testinfra/acceptance` and relevant `scenarios/axym/**` suites.
- Any story marked `Cross-platform: Yes` is incomplete until Linux, macOS, and Windows confirm the changed CLI/bundle/path surface.
- Any story marked `Risk: Yes` is release-blocking until `make prepush-full`, CodeQL, and the targeted identity-governance determinism checks are green.
- Broad doc repositioning must not ship ahead of runtime truth. At minimum, `W1-S2`, `W2-S1`, and `W2-S2` must be complete before the full public messaging change is treated as done.

---

## Epic W1: Identity-Governed Action Contract and Normalization

Objective: lock the additive identity-governance contract first, then normalize one deterministic identity-chain view across Axym-native collection and sibling ingestion before changing scoring or bundle outputs.

### Story W1-S1: Rewrite the source-of-truth product contract around identity-governed action
Priority: P0
Tasks:
- Update `product/axym.md` Executive Summary, JTBD, one-liner, evidence-surface tables, proof-record example, audit-bundle description, functional requirements, goals, and non-goals so they explicitly state that Axym proves which non-human identity acted, through which delegated chain, under which policy and approval.
- Add first-class evidence targets for acting identity, downstream execution identity, owner/approver, delegation chain, policy digest, approval-token binding, and identity/privilege drift between runs.
- Add an explicit boundary note and non-goal that Axym does not replace IAM / PAM / IGA and remains scoped to software-delivery governance proof.
- Add a PRD-local contract check so the new identity-governed action terminology and non-goals cannot silently regress while later public-doc alignment work is still pending.
- Call out where the current OSS surface is narrower than long-horizon PRD ambition so the doc stays truthful while the backlog is in flight.
Repo paths:
- `product/axym.md`
- `testinfra/contracts/product_identity_contract_test.go`
Run commands:
- `go test ./testinfra/contracts -run 'ProductIdentity' -count=1`
Test requirements:
- Docs/examples changes: PRD-local contract checks for identity-governed action language, non-goals, and boundary wording.
- Source-of-truth contract checks only; launch-facing doc sync is deferred to `W3-S1`.
Matrix wiring:
- Lanes: Fast, Core CI.
Acceptance criteria:
- `product/axym.md` explicitly describes Axym as proof of identity-governed action in software delivery, not only proof of AI behavior.
- The PRD includes the requested identity-bearing evidence signals, proof-record fields, bundle outputs, and IAM/PAM/IGA non-goal.
- The PRD remains truthful about current shipped scope and does not imply an IAM replacement or a widened wedge.
- Automated checks fail if the PRD regresses on identity-governed action terminology, boundary wording, or non-goals.
Stable/internal boundary notes:
- Public: this positioning becomes part of Axym's product contract.
- Internal: identity-governance scope remains limited to portable software-delivery evidence around upstream identity systems.
Migration expectations:
- Narrative update only; no command removals or exit-code changes are allowed in this story.
Integration hooks:
- `W3-S1` will sync launch-facing docs and operator guidance to this contract.
Dependencies:
- None.
Risks:
- PRD-only rewriting without later contract tests will recreate the same mismatch under new language.

### Story W1-S2: Introduce a normalized identity-chain view across native collection and sibling ingest
Priority: P0
Tasks:
- Define Axym's additive normalized identity-governance fields and canonical field placement, including `actor_identity`, `downstream_identity`, `delegation_chain`, `policy_digest`, `approval_token_ref`, and `owner_identity` or an equivalent owner reference.
- Extend `core/normalize` and `core/record` so Axym-native records can carry the normalized identity view without breaking existing `proof.Record` compatibility.
- Extend governance-event promotion so `collect --governance-event-file` becomes the in-scope Agnt-compatible path for identity-bearing action evidence.
- Extend Wrkr ingest and Gait translation to preserve or synthesize the normalized identity view from incoming records, including target touched, delegation lineage, policy binding, and approval binding when present.
- Preserve backward compatibility: existing records remain valid, but records missing identity linkage will later grade as weaker evidence rather than being rejected outright.
- Add fixtures, schema validation, and mixed-source integration tests covering Wrkr + Gait + Axym-native evidence in one deterministic chain.
Repo paths:
- `core/normalize/normalize.go`
- `core/record/builder.go`
- `schemas/v1/record/normalized-input.schema.json`
- `schemas/v1/record/schema.go`
- `core/collect/governanceevent/collector.go`
- `core/collect/governanceevent/collector_test.go`
- `core/ingest/gait/translate/translate.go`
- `core/ingest/gait/translate/verdict_mapping_test.go`
- `core/ingest/wrkr/ingest.go`
- `core/ingest/wrkr/ingest_test.go`
- `internal/integration/ingest/mixed_source_chain_test.go`
- `internal/integration/ingest/gait/relationship_preservation_test.go`
- `fixtures/governance/context_engineering.jsonl`
- `fixtures/ingest/wrkr/proof_records.jsonl`
- `fixtures/ingest/gait/native_records.jsonl`
- `fixtures/records/decision.json`
Run commands:
- `go build -o ./axym ./cmd/axym`
- `go test ./core/normalize ./core/record ./core/collect/governanceevent ./core/ingest/... -count=1`
- `go test ./internal/integration/ingest/... -count=1`
- `go test ./testinfra/contracts -run 'Collect|GovernanceEvent|RecordSchema' -count=1`
- `./axym collect --governance-event-file fixtures/governance/context_engineering.jsonl --json`
- `./axym ingest --source wrkr --input fixtures/ingest/wrkr/proof_records.jsonl --json`
- `./axym ingest --source gait --input fixtures/ingest/gait/native_records.jsonl --json`
- `./axym record add --input fixtures/records/decision.json --json`
Test requirements:
- Schema/artifact changes: schema validation tests, fixture/golden updates, compatibility tests for additive fields.
- SDK/adapter boundary changes: governance-event, Wrkr, and Gait adapter parity/conformance tests.
- Scenario/context changes: mixed-source chain fixtures validating the unified identity view.
- Determinism checks: repeat-run normalization and digest stability for synthesized identity fields.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Risk.
Acceptance criteria:
- Axym exposes one normalized identity-governance view across Axym-native, Wrkr, Gait, and Agnt-compatible governance-event/manual inputs.
- The normalized view can answer: who initiated, which identity executed, which target was touched, which owner/approver was responsible, which delegation chain applied, and which policy/approval bound the action when available.
- Gait translation and Wrkr ingest preserve or synthesize the needed identity lineage without breaking proof-chain integrity.
- Existing records without the new fields remain ingestible and verifiable.
Stable/internal boundary notes:
- Public: the normalized identity field names and semantics become part of the supported evidence contract.
- Internal: whether the implementation stores them in `event`, `metadata`, or `relationship` helpers is an internal detail as long as the documented normalized view stays stable.
Migration expectations:
- Additive only. Existing record producers keep working, but incomplete identity linkage will no longer count as full-strength evidence once `W2-S1` lands.
Integration hooks:
- Agnt compatibility lands via `collect --governance-event-file`, `record add`, and proof-format/manual inputs, not a speculative `ingest --source agnt` flag.
Dependencies:
- `W1-S1`
Risks:
- If field placement is left ambiguous, later mapping, grading, and bundle work will fork the model and create drift.

---

## Epic W2: Identity-Aware Compliance Evaluation and Bundle Outputs

Objective: once the identity-chain view exists, make Axym score it deterministically and surface it directly in bundles, gaps, reviews, and verification artifacts.

### Story W2-S1: Downgrade weak identity linkage in mapping, gaps, review, and regression
Priority: P0
Tasks:
- Extend compliance matching so controls that rely on governed action evidence require identity linkage, ownership, delegation, and policy/approval binding where applicable.
- Add deterministic reason codes for weak identity evidence, for example missing actor linkage, missing owner/approver linkage, missing approval binding, incomplete delegation chain, and unapproved privilege drift.
- Update `gaps`, auditability-grade derivation, Daily Review, and regression baselines so identity-linkage weakness lowers evidence strength predictably instead of being counted as a normal covered event.
- Ensure privilege drift without linked approval evidence surfaces as actionable governance weakness in `map`, `gaps`, `review`, and `regress`.
- Keep behavior additive and deterministic: same inputs still produce same outputs, but incomplete identity linkage must now score weaker.
Repo paths:
- `core/compliance/match/matcher.go`
- `core/compliance/match/matcher_test.go`
- `core/compliance/coverage/coverage.go`
- `core/gaps/gaps.go`
- `core/gaps/gaps_test.go`
- `core/review/grade/grade.go`
- `core/review/grade/weakest_link_test.go`
- `core/review/review.go`
- `core/review/privilegedrift/analyzer.go`
- `internal/integration/gaps/gaps_workflow_test.go`
- `internal/integration/regress/exit5_on_drift_test.go`
- `internal/e2e/review/empty_day_contract_test.go`
- `testinfra/contracts/map_gaps_contract_test.go`
- `testinfra/contracts/compliance_exit_contract_test.go`
- `testinfra/contracts/invalid_evidence_not_counted_test.go`
- `testinfra/contracts/review_override_replay_contract_test.go`
- `fixtures/frameworks/regress-minimal.yaml`
Run commands:
- `go build -o ./axym ./cmd/axym`
- `go test ./core/compliance/... ./core/gaps ./core/review/... -count=1`
- `go test ./internal/integration/gaps ./internal/integration/regress ./internal/e2e/review -count=1`
- `go test ./testinfra/contracts -run 'Map|Gaps|Compliance|Review|Regress' -count=1`
- `./axym map --frameworks eu-ai-act,soc2 --json`
- `./axym gaps --frameworks eu-ai-act,soc2 --json`
- `./axym review --date 2026-02-28 --json`
- `./axym regress init --baseline ./.axym/identity-baseline.json --frameworks eu-ai-act,soc2 --json`
- `./axym regress run --baseline ./.axym/identity-baseline.json --frameworks eu-ai-act,soc2 --json`
Test requirements:
- Gate/policy/fail-closed changes: deterministic `covered` / `partial` / `gap` fixtures, fail-closed undecidable-path checks, and reason-code stability tests.
- Determinism/hash changes: golden coverage outputs and repeat-run grading checks.
- Scenario/context changes: scenarios proving that missing identity linkage is weak evidence, not invisible evidence.
- CLI behavior changes: `--json` stability and exit-code contract tests where regress or threshold outcomes change.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Risk.
Acceptance criteria:
- Controls that require governed identity evidence cannot remain `covered` when the action cannot be tied to the required non-human identity, owner, delegation chain, or approval binding.
- `map`, `gaps`, `review`, and `regress` surface the weakness with stable reason codes and deterministic outputs.
- Privilege drift without linked approval evidence is surfaced as an actionable governance weakness rather than only a raw diff.
- Repeat runs with the same inputs produce identical grades, reason codes, and ranking.
Stable/internal boundary notes:
- Public: `covered` / `partial` / `gap`, auditability-grade reasoning, and JSON reason codes are contract surfaces.
- Internal: scoring weights may evolve only through explicit deterministic fixture updates and docs changes.
Migration expectations:
- Existing regression baselines and golden outputs will need intentional refresh because weak identity linkage is now scored differently.
Integration hooks:
- Bundle, verify, and docs work in later stories must consume these reason codes instead of inventing their own identity-health model.
Dependencies:
- `W1-S2`
Risks:
- If downgrade rules are fuzzy or framework-specific without contract tests, teams will fight nondeterministic coverage churn.

### Story W2-S2: Add identity-governance artifacts and completeness checks to the audit bundle
Priority: P0
Tasks:
- Add deterministic bundle artifacts for identity governance, including an identity-chain summary, ownership/approver register, privilege-drift report, delegated-chain exceptions report, and an updated boundary contract clarifying Axym versus IAM/PAM/IGA responsibility.
- Extend bundle verification so missing or inconsistent identity-governance artifacts fail completeness checks in a typed, machine-readable way.
- Keep the bundle additive: existing artifacts remain present, new identity artifacts are layered in without breaking current bundle consumers.
- Update executive-summary and bundle schemas so the identity-governance outputs are validated, documented, and stable.
- Ensure bundle generation remains byte-stable for identical inputs and does not introduce hidden nondeterminism.
Repo paths:
- `core/bundle/bundle.go`
- `core/bundle/bundle_test.go`
- `core/verify/bundle/verify.go`
- `core/verify/verify.go`
- `schemas/v1/bundle/executive-summary-v1.schema.json`
- `schemas/v1/bundle/schema.go`
- `internal/integration/bundle/context_engineering_bundle_test.go`
- `internal/e2e/bundleverify/oscal_schema_validation_test.go`
- `cmd/axym/bundle_test.go`
- `cmd/axym/verify_test.go`
- `testinfra/contracts/bundle_contract_test.go`
- `testinfra/contracts/oscal_schema_contract_test.go`
- `testinfra/contracts/verify_contract_test.go`
- `fixtures/bundles/good/manifest.json`
- `fixtures/bundles/good/evidence.json`
Run commands:
- `go build -o ./axym ./cmd/axym`
- `go test ./core/bundle ./core/verify/... -count=1`
- `go test ./internal/integration/bundle ./internal/e2e/bundleverify -count=1`
- `go test ./testinfra/contracts -run 'Bundle|OSCAL|Verify' -count=1`
- `./axym bundle --audit identity-governance --frameworks eu-ai-act,soc2 --json`
- `./axym verify --bundle ./axym-evidence --frameworks eu-ai-act,soc2 --json`
- `./axym verify --chain --json`
Test requirements:
- Schema/artifact changes: schema validation tests, fixture/golden updates, compatibility checks for additive bundle files.
- CLI behavior changes: `--json` stability tests and exit-code contract checks for `bundle` and `verify`.
- Determinism/hash/sign changes: byte-stability repeat-run tests, canonicalization checks, verify determinism tests, and `make test-contracts` coverage.
- Scenario/context changes: bundle fixtures proving that identity lineage is present, explainable, and portable.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
Acceptance criteria:
- The bundle includes dedicated identity-governance artifacts and preserves existing bundle artifacts.
- `verify --bundle --json` surfaces identity-governance completeness in a deterministic, typed way and fails closed when required artifacts are missing or inconsistent.
- The bundle itself explains what Axym proves versus what IAM/PAM/IGA still owns.
- Repeat bundle builds with the same inputs produce byte-stable identity-governance artifacts.
Stable/internal boundary notes:
- Public: artifact names, locations, and schema shapes for identity-governance outputs become supported bundle contract.
- Internal: implementation helpers may change, but bundle layout and verification semantics must remain stable once published.
Migration expectations:
- Additive only. Existing bundle consumers can ignore the new files, but removal or rename of those files after publication requires explicit contract versioning.
Integration hooks:
- Docs and sample flows in `W3-S1` must show these artifacts directly so operators understand the bundle deliverable.
Dependencies:
- `W2-S1`
Risks:
- Adding identity artifacts without verify-time completeness checks would make the bundle look stronger without actually tightening the contract.

---

## Epic W3: Public Docs, Examples, and CI Enforcement

Objective: once runtime truth exists, update the public/operator narrative and wire durable acceptance checks so the repo cannot drift back to action-first positioning or incomplete identity evidence.

### Story W3-S1: Sync public docs and sample assets to the identity-governed action seam
Priority: P1
Tasks:
- Update launch-facing and operator docs so they describe Axym as portable proof of identity-governed action in software delivery, with explicit examples of acting identity, downstream identity, owner/approver, delegation chain, policy digest, and approval binding.
- Update sample-pack and fixture-backed examples so the published first-value flow demonstrates identity-bearing evidence instead of only generic activity events.
- Keep docs honest to shipped command surfaces and repo scope: no speculative `agnt` CLI source, no IAM replacement language, and no wedge expansion beyond software delivery.
- Show the Axym-vs-IAM/PAM/IGA boundary in operator docs, docs-site summaries, and sample-bundle walkthroughs.
- Keep launch-facing docs aligned with the source-of-truth language from `product/axym.md` and the actual bundle outputs from `W2-S2`.
Repo paths:
- `README.md`
- `docs/commands/axym.md`
- `docs/operator/quickstart.md`
- `docs/operator/integration-model.md`
- `docs/operator/integration-boundary.mmd`
- `docs-site/public/llms.txt`
- `docs-site/public/llm/axym.md`
- `core/samplepack/pack.go`
- `fixtures/governance/context_engineering.jsonl`
- `fixtures/records/decision.json`
- `scenarios/axym/first_value_sample/contract.json`
- `product/axym.md`
Run commands:
- `go build -o ./axym ./cmd/axym`
- `make test-docs-consistency`
- `make test-docs-storyline`
- `make test-docs-links`
- `go test ./testinfra/contracts -run 'CommandDocs|Collect|Bundle|Verify' -count=1`
- `go test ./scenarios/axym/... -count=1`
- `./axym init --sample-pack ./axym-sample --json`
- `./axym collect --governance-event-file ./axym-sample/governance/context_engineering.jsonl --json`
- `./axym bundle --audit docs-identity-sample --frameworks eu-ai-act,soc2 --json`
Test requirements:
- Docs/examples changes: docs consistency, storyline/smoke checks, README/quickstart/integration coverage checks, and docs-site source-of-truth sync checks.
- Scenario/context changes: sample-pack and scenario fixtures that assert identity-governance outputs.
- CLI behavior changes: command-doc parity tests for any updated sample commands.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance.
Acceptance criteria:
- README, operator docs, command docs, and docs-site content all describe the identity-governed action seam consistently.
- Published examples and sample assets include explicit identity-chain evidence and show the resulting bundle outputs.
- No public doc positions Axym as an IAM/PAM/IGA replacement or widens the wedge beyond software delivery.
- Docs remain truthful to the actual command surface and runtime outputs.
Stable/internal boundary notes:
- Public: launch-facing positioning and example flows are part of the OSS contract.
- Internal: fixture internals may evolve, but the published story and example semantics must stay aligned with shipped behavior.
Migration expectations:
- Existing docs examples should be updated in place; do not keep parallel, conflicting "behavior-only" examples alive.
Integration hooks:
- Operators should be able to see exactly where Axym fits relative to Gait, Wrkr, and upstream identity systems.
Dependencies:
- `W1-S1`
- `W2-S2`
Risks:
- Public docs that move before sample assets and bundle outputs are real will recreate an expectation gap immediately.

### Story W3-S2: Wire identity-governed action acceptance and CI enforcement
Priority: P1
Tasks:
- Add or formalize an acceptance target for identity-governed action workflows, including sample-pack -> collect -> map -> gaps -> bundle -> verify and mixed-source Wrkr + Gait + Axym-native flows.
- Extend scenario contracts and goldens so they assert identity-chain summaries, delegated-chain exceptions, privilege-drift reporting, and weak-evidence downgrade behavior.
- Wire PR, main, and nightly workflows so the new acceptance and risk checks are enforced without weakening existing branch-protection contracts.
- Update CI contract tests and required-check fixtures whenever workflow names, triggers, or required jobs change.
- Ensure cross-platform validation covers any CLI/path/bundle contract changes introduced by the identity-governance work.
Repo paths:
- `Makefile`
- `.github/workflows/pr.yml`
- `.github/workflows/main.yml`
- `.github/workflows/nightly.yml`
- `testinfra/acceptance/scenario_contract_test.go`
- `testinfra/contracts/ci_contract_test.go`
- `testinfra/contracts/ci_required_checks_test.go`
- `testinfra/contracts/release_gate_contract_test.go`
- `scenarios/axym/context_engineering_scenario_test.go`
- `scenarios/axym/first_value_sample/contract.json`
- `scenarios/axym/golden/results.json`
- `internal/scenarios/scenario_suite_test.go`
- `scripts/check_branch_protection_contract.sh`
Run commands:
- `go build -o ./axym ./cmd/axym`
- `make test-acceptance`
- `make test-scenarios`
- `go test ./testinfra/contracts -run 'CI|ReleaseGate' -count=1`
- `./scripts/check_branch_protection_contract.sh`
- `make prepush-full`
Test requirements:
- Job runtime/state changes: workflow lifecycle tests for the full collect/map/gaps/bundle/verify flow when acceptance wiring changes.
- Scenario/context changes: acceptance and scenario suites proving identity-governed action outcomes end to end.
- Docs/examples changes: storyline checks remain wired in CI.
- Cross-platform and release-gate contract tests for workflow/required-check correctness.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
Acceptance criteria:
- CI has explicit acceptance coverage for the identity-governed action seam and required-check contracts know about it.
- Scenario and acceptance suites fail if identity-chain artifacts, weak-evidence downgrades, or bundle boundary notes regress.
- Cross-platform validation covers any changed CLI or bundle path semantics.
- Branch-protection contract checks stay truthful after workflow updates.
Stable/internal boundary notes:
- Public: required checks and release gates become part of the trust baseline for the new seam.
- Internal: workflow internals may change, but named required checks and enforced contract coverage must stay in sync.
Migration expectations:
- Any workflow rename or trigger change must update CI contract tests in the same PR.
Integration hooks:
- Acceptance coverage must include Wrkr + Gait interoperability and the generic Agnt-compatible input path, not a new sibling-specific code path.
Dependencies:
- `W1-S2`
- `W2-S2`
- `W3-S1`
Risks:
- Leaving acceptance wiring as an informal local-only step will let the repo slide back into behavior-first evidence without visible breakage.

---

## Minimum-Now Sequence

1. `W1-S1` and `W1-S2`
Rationale: lock the source-of-truth language and the normalized identity model before scoring or bundle work hardens the wrong semantics.
2. `W2-S1`
Rationale: once the normalized fields exist, weak identity linkage must become weak evidence everywhere Axym calculates compliance.
3. `W2-S2`
Rationale: bundle outputs should be built on top of the final grading and normalization model so filenames, schemas, and verify semantics only settle once.
4. `W3-S1`
Rationale: public/operator docs should move only after runtime truth and bundle deliverables are real.
5. `W3-S2`
Rationale: wire final acceptance and CI gates after the contract is stable so required checks protect the final seam rather than an intermediate state.

If only the minimum runtime slice can be landed before a broader docs push, stop after `W2-S2`. That is the first point where Axym actually earns the stronger identity-governed action framing.

---

## Explicit Non-Goals

- No repositioning of Axym as a standalone identity product.
- No replacement of IAM, PAM, or IGA systems.
- No widening beyond software-delivery governance evidence.
- No speculative `ingest --source agnt` product integration in this repo.
- No LLM-based inference in default collect/map/gaps/verify paths.
- No breaking top-level proof-record format, exit codes, or bundle contracts without explicit versioning.
- No dashboard-first or hosted-control-plane scope in this backlog.

---

## Definition of Done

- Every recommendation in this plan maps to merged code/docs/tests or an explicitly accepted follow-up issue.
- `product/axym.md` and launch-facing docs consistently describe Axym as proof of identity-governed action in software delivery.
- Axym exposes one additive normalized identity-governance view across native collection, Wrkr ingest, Gait translation, and Agnt-compatible generic inputs.
- Missing identity linkage lowers evidence quality, control coverage, and auditability grade deterministically with stable reason codes.
- Bundles and bundle verification include and enforce the new identity-governance artifacts.
- Existing record producers remain compatible; any scoring or baseline changes are documented with intentional fixture refreshes.
- Tests are added or updated at the right layers: schema, unit, integration, E2E CLI, acceptance, scenario, contract, cross-product, and CI-contract coverage where touched.
- CI matrix wiring is updated and green for all required lanes in this plan.
- Public docs and examples remain truthful and do not position Axym as an IAM/PAM/IGA replacement.
- `make prepush-full`, targeted acceptance coverage, and CodeQL pass for the completed implementation set before release.
