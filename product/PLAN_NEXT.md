# PLAN Axym: Security Signal, Self-Serve Record Contract, and Launch Docs Enforcement

Date: 2026-03-20
Source of truth: user-provided 2026-03-20 app-audit findings and remediation guidance, `product/axym.md`, `product/dev_guides.md`, `product/architecture_guides.md`, `AGENTS.md`
Scope: Axym OSS CLI only. Convert the current launch-readiness audit findings into an execution-ready backlog covering hosted security-signal truth, manual `record add` contract clarity, deterministic collect diagnostics, PR docs-enforcement wiring, and launch-surface/release-doc alignment. No new product-scope expansion beyond Axym's shipped CLI and `Clyra-AI/proof` interoperability.

---

## Global Decisions (Locked)

- This plan supersedes the prior `PLAN_NEXT.md` focus. The current priority is launch re-open through truthful security posture, self-serve public contracts, and enforceable launch docs.
- Broad public OSS launch remains no-go until both Wave `W1` and Wave `W2` are complete. A clearly labeled technical preview can be reconsidered after Wave `W1`.
- The current first-10-minute structure stays in place:
  - `smoke test`
  - `sample proof path`
  - `real integration path`
- The first-value promise remains "non-empty evidence, ranked gaps, and intact local verification," not "full compliance completeness on first run."
- The public sample path remains truthful to the currently shipped runtime:
  - `4` governance-event captures
  - `6` total records after manual append
  - `5` covered controls out of `6`
  - grade `C`
  - `complete=false`
  - `weak_record_count=1`
- GitHub-hosted CodeQL remains part of Axym's public OSS trust story for this plan. Therefore the hosted workflow must upload real analysis results on `pull_request` and `main`; removing the public claim is not the preferred path for this backlog.
- `v1` is the canonical public `record_version` token for Axym-authored manual proof-record examples and fixtures. If legacy `1.0` compatibility is retained for previously documented inputs, it must normalize deterministically and be explicitly tested in the same change.
- Public docs must distinguish stable user-facing command/integration surfaces from internal implementation details. Any deprecated surface must include migration guidance and versioning expectations.
- Contract/runtime and architecture-boundary work lands in `W1` before later docs, OSS hygiene, onboarding, and distribution UX work in `W2`.
- Preserve Axym non-negotiables throughout:
  - determinism
  - offline-first defaults
  - fail-closed policy enforcement
  - schema stability
  - exit-code stability
- No dashboard-first, SaaS-first, or enterprise-fork work belongs in this plan.

---

## Current Baseline (Observed)

- The end-to-end product model is coherent. `README.md`, `docs/operator/quickstart.md`, `docs/operator/integration-model.md`, and `docs-site/public/llm/axym.md` already separate `smoke`, `sample proof path`, and `real integration path`.
- The runtime is healthy. The audited repository passed:
  - `go build ./cmd/axym`
  - `make prepush`
  - `make test-security`
  - `make release-local`
  - `make release-go-nogo-local`
- The shipped sample path is already truthful and deterministic when executed locally:
  - `4` governance-event captures
  - `6` total records
  - `5/6` covered controls
  - grade `C`
  - intact `6`-record chain
  - verified bundle with `complete=false`
- Filesystem mutation boundaries are already strong and out of scope for redesign in this plan:
  - bundle output rejects non-managed non-empty dirs
  - managed-marker symlink/directory cases fail closed
  - `verify --bundle --temp-dir ...` does not create `.axym-managed`
- `.github/workflows/codeql.yml` still uses `upload: never`, while public surfaces claim GitHub-hosted CodeQL analysis and `product/dev_guides.md` states CodeQL is part of the PR/protected-branch security posture.
- `schemas/v1/record/` only exposes `normalized-input.schema.json`. The published `record add --input <record.json> --json` integration path does not yet have a self-serve in-repo schema/example contract that a user can find directly from launch docs.
- Manual proof-record version notation drifts across shipped surfaces:
  - `v1` in runtime fixtures and examples
  - `1.0` in `product/axym.md`
- `core/collect/governanceevent/collector.go` returns an empty result without source-level `reason_codes` when no governance-event files are supplied, even though launch docs describe deterministic per-source `reason_codes`.
- `.github/workflows/pr.yml` currently runs `docs-links` only. `make test-docs-consistency` and `make test-docs-storyline` are merge-blocking by documentation and on `main`/release lanes, but not yet on the PR workflow itself.
- Release-tooling docs still have truth gaps versus the actual workflow and scripts:
  - `product/dev_guides.md` lists GoReleaser `v2.13.3`
  - `.github/workflows/release.yml` uses `goreleaser/goreleaser-action@v7` with `version: v2.14.1`
  - local `release-go-nogo` validation is intentionally different from the hosted OIDC signing path and needs clearer wording

---

## Exit Criteria

1. GitHub-hosted CodeQL uploads real analysis results on `pull_request` and `main`, and contract tests fail if that hosted-security promise regresses.
2. Public docs, contributor docs, and CI contract tests all tell the same truth about local `make codeql` vs GitHub-hosted CodeQL responsibilities.
3. `record add` has an authoritative in-repo public contract with:
   - a discoverable schema/example path
   - stable error-envelope expectations
   - locked `record_version` behavior
   - docs links from launch-facing surfaces
4. All public Axym-authored manual record examples use canonical `record_version: "v1"`, and any retained compatibility for legacy documented `1.0` inputs is deterministic and tested.
5. `governanceevent` returns deterministic source-level `reason_codes` for the no-input path in `collect --dry-run --json` and matching contract tests.
6. PR workflow includes merge-blocking `docs-consistency` and `docs-storyline` jobs, and branch-protection contract tests require them.
7. Release/install/contributor docs align with the actual workflow and scripts, including hosted vs local release verification differences.
8. Launch docs and docs-site:
   - classify public surfaces as stable/internal/deprecated where relevant
   - link the authoritative manual proof-record contract
   - preserve the expectation-safe first-value story
9. Wave `W1` is green before technical preview reconsideration.
10. Wave `W2` is green before broad public OSS launch is reconsidered.

---

## Recommendation Traceability

| Recommendation | Why | Strategic direction | Expected moat/benefit | Planned stories |
|---|---|---|---|---|
| Enable real hosted CodeQL visibility and stop overclaiming security posture | Public trust breaks if the repo claims hosted CodeQL but uploads nothing | Verifiable OSS security-signal baseline | Lower reputational risk and stronger launch trust | `W1-S1`, `W2-S2` |
| Make CodeQL/branch-protection behavior explicit in contributor and CI contracts | Security checks must be visible and reviewable in PR workflows, not implied | Truthful CI governance | Fewer surprise failures and clearer maintainer posture | `W1-S1`, `W2-S1` |
| Publish a self-serve public contract for `record add` | The promoted manual-append path is too opaque today | Stable extension surface for operator and ecosystem adoption | Lower integration friction and clearer ownership boundary | `W1-S2`, `W2-S3` |
| Unify `record_version` notation around one canonical token | Mixed `v1` / `1.0` examples create contract ambiguity | Versioned public payload contract | Cleaner docs, safer integrations, fewer support loops | `W1-S2` |
| Return `NO_INPUT` for empty governance-event collection | Per-source diagnostics should be explainable and deterministic | Better runtime/operator diagnostics | Faster setup/debug loop with less guesswork | `W1-S3` |
| Move docs-consistency/storyline checks into PR gating | Merge-blocking docs claims should be enforced where merges happen | Docs source-of-truth enforcement | Prevents launch-doc regressions before merge | `W2-S1` |
| Align release-tooling docs with the actual workflow | Tool/version drift weakens contributor and release trust | Release-truth alignment | Better contributor confidence and fewer release surprises | `W2-S2` |
| Add stable/internal/deprecated notes and keep first-value messaging expectation-safe | Users need to know what is truly supported and what first value actually means | Public-surface clarity without wedge creep | Higher onboarding quality and lower expectation mismatch | `W2-S3` |

---

## Test Matrix Wiring

Lane definitions:

- Fast lane:
  - `make lint-fast`
  - targeted `go test ./testinfra/contracts ... -count=1`
  - targeted package tests for `cmd/axym`, `core/collect/governanceevent`, or `schemas/v1/record` when those surfaces change
  - `make test-docs-consistency`
  - `make test-docs-storyline`
- Core CI lane:
  - `make test-fast`
  - `make test-contracts`
- Acceptance lane:
  - `make test-acceptance`
  - targeted first-value acceptance coverage when launch-facing sample/integration docs or payload contracts change
- Cross-platform lane:
  - existing `.github/workflows/main.yml` Linux/macOS/Windows matrix for stories that change public CLI invocation examples, install/release guidance, or public integration-surface docs
- Risk lane:
  - `make test-security`
  - `make lint-go`
  - `make codeql`
  - `make release-local`
  - `make release-go-nogo-local`
  - nightly/release workflow parity checks when CI or release gates change

Story-to-lane map:

| Story | Fast | Core CI | Acceptance | Cross-platform | Risk |
|---|---|---|---|---|---|
| `W1-S1` | Yes | Yes | No | No | Yes |
| `W1-S2` | Yes | Yes | Yes | Yes | No |
| `W1-S3` | Yes | Yes | No | No | No |
| `W2-S1` | Yes | Yes | No | No | No |
| `W2-S2` | Yes | Yes | No | No | Yes |
| `W2-S3` | Yes | Yes | Yes | Yes | No |

Merge/release gating rule:

- `W1` removes the launch no-go blockers and must land before any broad docs/distribution polish is considered complete.
- `W2` cannot start closing out launch-facing copy until the `W1` public contracts are locked and testable.
- Any story that changes a public launch surface must pass its declared docs-contract lanes in the same PR.
- Any story that changes CI or release behavior must update the corresponding contract tests in the same PR.
- Technical preview reconsideration gate:
  - `W1-S1`
  - `W1-S2`
  - `W1-S3`
- Broad public OSS launch reconsideration gate:
  - all `W1` stories
  - all `W2` stories

---

## Epic W1: Launch Blocker Contracts and Runtime Truth

Objective: remove the current launch blockers in hosted security signal and manual append contract clarity, then close the remaining runtime diagnostic gap while preserving Axym's existing architecture boundaries and deterministic behavior.

### Story W1-S1: Restore truthful hosted CodeQL signal and CI security contracts
Priority: P0
Tasks:
- Remove the effective "no hosted results" behavior from `.github/workflows/codeql.yml` by enabling real SARIF upload for the GitHub-hosted CodeQL workflow on `pull_request` and `main`.
- Extend CI contract tests so they assert active hosted CodeQL behavior, not just workflow-file presence and trigger snippets.
- Keep local `make codeql` as the contributor-facing CLI path, but explicitly separate it from the hosted GitHub analysis path in docs and contracts.
- Update launch-facing and contributor-facing wording so "GitHub-hosted CodeQL analysis" is only claimed where the workflow behavior actually supports it.
- Document the intended relationship between hosted CodeQL, local `make codeql`, and PR security visibility.
Repo paths:
- `.github/workflows/codeql.yml`
- `testinfra/contracts/ci_contract_test.go`
- `testinfra/contracts/ci_required_checks_test.go`
- `README.md`
- `CONTRIBUTING.md`
- `product/dev_guides.md`
- `docs-site/public/llm/axym.md`
Run commands:
- `go test ./testinfra/contracts -run 'CI|RequiredChecks' -count=1`
- `make test-contracts`
- `make codeql`
- `make test-docs-consistency`
Test requirements:
- SDK/adapter boundary changes:
  - CI/workflow contract tests that assert hosted-security behavior is mapped correctly
- Docs/examples changes:
  - README and contributor-doc consistency checks when hosted-security wording changes
- Determinism/hash/sign/packaging changes:
  - no packaging change, but CI contract tests must pin the intended hosted-analysis behavior
Matrix wiring:
- Lanes: Fast, Core CI, Risk
- Required gates:
  - targeted CI contract tests
  - `make test-contracts`
  - `make codeql`
Acceptance criteria:
- `.github/workflows/codeql.yml` no longer leaves hosted CodeQL disabled while public docs still claim hosted analysis.
- Contract tests fail if hosted CodeQL regressions reintroduce "upload nothing" behavior.
- Docs clearly separate local `make codeql` from GitHub-hosted CodeQL results.
- Axym's public OSS trust story no longer overclaims its hosted security signal.
Stable/internal boundary notes:
- Public:
  - GitHub-hosted CodeQL availability
  - local `make codeql` discoverability
  - PR-visible security analysis posture
- Internal:
  - workflow-step implementation details beyond the documented contract
Migration expectations:
- No CLI migration.
- Contributors may see clearer or newly effective hosted CodeQL results on PRs.
Integration hooks:
- GitHub `pull_request` security workflow
- protected-branch security review
- contributor local verification with `make codeql`
Dependencies:
- None
Risks:
- A contract test that only checks workflow-file existence would preserve the current false-positive state and is insufficient.

### Story W1-S2: Publish a canonical manual proof-record contract and normalize `record_version`
Priority: P0
Tasks:
- Add an authoritative public manual proof-record contract under `schemas/v1/record/` or an equivalent in-repo location that is directly linkable from launch docs.
- Add or update deterministic example fixtures for the public `record add` path and keep them separate from internal normalized-input-only schema fixtures.
- Lock Axym-authored public examples to canonical `record_version: "v1"`.
- Retain deterministic compatibility for legacy documented `record_version: "1.0"` manual inputs by normalizing them at the manual-append boundary, or fail them with a typed contract error if that compatibility decision proves impossible. The chosen behavior must be encoded in tests and docs in the same PR.
- Extend contract tests to cover:
  - valid manual proof-record append
  - invalid required-field cases
  - unknown `record_type`
  - version-token behavior
  - machine-readable `schema_violation` envelopes
- Update public docs to explain what Axym validates locally versus what `Clyra-AI/proof` owns as the shared primitive contract.
Repo paths:
- `cmd/axym/record.go`
- `cmd/axym/record_test.go`
- `core/samplepack/pack.go`
- `fixtures/records/`
- `schemas/v1/record/`
- `testinfra/contracts/record_schema_contract_test.go`
- `README.md`
- `docs/commands/axym.md`
- `docs/operator/quickstart.md`
- `docs/operator/integration-model.md`
- `docs-site/public/llm/axym.md`
- `product/axym.md`
- `CHANGELOG.md`
Run commands:
- `go test ./cmd/axym ./testinfra/contracts -run 'Record|Schema' -count=1`
- `make test-contracts`
- `axym record add --input ./fixtures/records/decision.json --json`
Test requirements:
- Schema/artifact changes:
  - schema validation tests
  - valid/invalid fixture updates
  - compatibility tests for legacy documented version tokens if supported
- CLI behavior changes:
  - `--json` stability tests
  - exit-code contract tests
  - machine-readable `schema_violation` envelope checks
- Docs/examples changes:
  - README/quickstart/command-guide/docs-site parity
  - docs source-of-truth sync for the new contract link
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform
- Required gates:
  - targeted record/schema contract tests
  - `make test-contracts`
  - `make test-docs-consistency`
Acceptance criteria:
- A user can find one authoritative in-repo contract for the public `record add` payload without reading source code.
- All Axym-authored public examples use `record_version: "v1"`.
- The runtime behavior for any retained legacy documented `1.0` inputs is deterministic and tested.
- Manual-append contract tests fail if docs drift from the locked schema/example behavior.
- Public docs explain the ownership boundary between Axym validation/signing and `Clyra-AI/proof` primitives.
Stable/internal boundary notes:
- Public:
  - manual proof-record JSON payload shape
  - `record_version` semantics
  - `schema_violation` error behavior
  - discoverable docs link for manual append
- Internal:
  - normalized-input schema used before proof-record construction
  - helper placement and fixture organization
Migration expectations:
- New public examples migrate to `v1`.
- If legacy `1.0` remains accepted, docs must call it compatibility-only and new examples must not use it.
Integration hooks:
- operator/manual append workflow
- automation that emits proof records for `axym record add`
- docs-site/public LLM-friendly integration guidance
Dependencies:
- None
Risks:
- Updating docs without adding a linkable public contract will keep the self-serve integration gap open.

### Story W1-S3: Make governance-event no-input diagnostics explicit and stable
Priority: P1
Tasks:
- Return explicit source-level `NO_INPUT` reason codes from `governanceevent` collection when no governance-event files are supplied.
- Add collector-level tests for the no-input case so `governanceevent` behaves like the other built-in collectors.
- Extend `collect` contract coverage to assert that clean-room `collect --dry-run --json` includes deterministic `reason_codes` for `governanceevent`, not just for the other built-in sources.
- Update launch docs only where they need to clarify deterministic empty-source diagnostics.
Repo paths:
- `core/collect/governanceevent/collector.go`
- `core/collect/governanceevent/`
- `testinfra/contracts/collect_contract_test.go`
- `README.md`
- `docs/commands/axym.md`
Run commands:
- `go test ./core/collect/governanceevent ./testinfra/contracts -run 'Collect' -count=1`
- `make test-contracts`
- `axym collect --dry-run --json`
Test requirements:
- CLI behavior changes:
  - `--json` stability checks for per-source `reason_codes`
  - reason-code contract checks
- Docs/examples changes:
  - docs parity updates only if wording changes
- Scenario/context changes:
  - not required unless a scenario fixture intentionally encodes the empty-source path
Matrix wiring:
- Lanes: Fast, Core CI
- Required gates:
  - targeted collect contract tests
  - `make test-contracts`
Acceptance criteria:
- `governanceevent` reports source-level `reason_codes: ["NO_INPUT"]` on clean-room dry runs.
- The command remains side-effect free and successful in the no-input case.
- Contract tests fail if `governanceevent` falls back to empty-without-explanation again.
Stable/internal boundary notes:
- Public:
  - deterministic `collect --json` source summaries
  - stable reason-code semantics for empty sources
- Internal:
  - collector helper implementation and ordering
Migration expectations:
- Additive diagnostics improvement only.
- No CLI migration or exit-code change.
Integration hooks:
- operator smoke path
- real integration readiness checks before a governance-event file exists
Dependencies:
- None
Risks:
- Fixing only aggregate `reason_codes` without fixing the per-source summary will not satisfy the launch-doc contract.

---

## Epic W2: PR Enforcement, Release Truth, and Public Surface Clarity

Objective: once the blocking contracts are locked, make launch-facing docs enforceable at PR time and align release/install/public-surface wording with the actual runtime, workflow, and support posture.

### Story W2-S1: Promote docs-consistency and docs-storyline into PR-blocking workflow
Priority: P1
Tasks:
- Add dedicated `docs-consistency` and `docs-storyline` jobs to `.github/workflows/pr.yml`.
- Keep those jobs emitted by the `pull_request` workflow so they can be used as branch-protection required checks.
- Update `scripts/check_branch_protection_contract.sh` and CI contract tests to require the new PR jobs and their stable job names.
- Update contributor-facing docs so the documented merge-blocking lanes match the actual PR workflow.
Repo paths:
- `.github/workflows/pr.yml`
- `scripts/check_branch_protection_contract.sh`
- `testinfra/contracts/ci_contract_test.go`
- `testinfra/contracts/ci_required_checks_test.go`
- `CONTRIBUTING.md`
- `product/dev_guides.md`
Run commands:
- `go test ./testinfra/contracts -run 'CI|RequiredChecks' -count=1`
- `make test-docs-consistency`
- `make test-docs-storyline`
- `make prepush`
Test requirements:
- SDK/adapter boundary changes:
  - CI/workflow contract tests for PR job presence and trigger type
- Docs/examples changes:
  - contributor-doc alignment checks when PR gate descriptions change
- Job runtime/state/concurrency changes:
  - workflow contract tests to ensure the new jobs remain `pull_request` emitted and non-aliased
Matrix wiring:
- Lanes: Fast, Core CI
- Required gates:
  - targeted CI contract tests
  - `make test-docs-consistency`
  - `make test-docs-storyline`
Acceptance criteria:
- PR workflow contains dedicated merge-blocking `docs-consistency` and `docs-storyline` jobs.
- Contract tests fail if those jobs are removed, renamed, or moved off the `pull_request` workflow.
- Contributor docs no longer overstate docs gating relative to the actual PR workflow.
Stable/internal boundary notes:
- Public:
  - merge-blocking docs parity promise
  - contributor pre-merge workflow expectations
- Internal:
  - exact job implementation beyond the required check names and triggers
Migration expectations:
- Contributors may see two additional PR checks.
- No CLI/runtime migration.
Integration hooks:
- GitHub PR workflow
- branch protection
- contributor pre-merge checklist
Dependencies:
- `W1-S2` should land first so docs lanes can enforce the new manual contract link as part of launch-surface parity
Risks:
- Updating docs without adding PR jobs preserves the current mismatch between promise and enforcement.

### Story W2-S2: Align release-tooling docs with real workflow and verification behavior
Priority: P1
Tasks:
- Reconcile documented release-tool versions and steps with the actual workflow and scripts:
  - GoReleaser version
  - `syft`
  - `cosign`
  - local vs hosted signing/verification behavior
- Update release-facing docs so they clearly distinguish:
  - local `make release-local`
  - local `make release-go-nogo-local`
  - hosted GitHub release pipeline behavior
- Extend release/CI contract tests where necessary to lock the corrected tooling/version wording or contract expectations.
- Update `CHANGELOG.md` if release-facing contract wording changes.
Repo paths:
- `product/dev_guides.md`
- `README.md`
- `CONTRIBUTING.md`
- `.github/workflows/release.yml`
- `.goreleaser.yaml`
- `scripts/release_go_nogo.sh`
- `testinfra/contracts/release_gate_contract_test.go`
- `testinfra/contracts/ci_contract_test.go`
- `CHANGELOG.md`
Run commands:
- `go test ./testinfra/contracts -run 'Release|CI' -count=1`
- `make test-docs-consistency`
- `make release-local`
- `make release-go-nogo-local`
Test requirements:
- Determinism/hash/sign/packaging changes:
  - release gate contract checks
  - checksum/signature/provenance verification command parity
- Docs/examples changes:
  - contributor/release-doc consistency checks
  - README release/install guidance parity when touched
- SDK/adapter boundary changes:
  - workflow contract tests if release workflow wording or step assumptions are locked
Matrix wiring:
- Lanes: Fast, Core CI, Risk
- Required gates:
  - targeted release contract tests
  - `make release-local`
  - `make release-go-nogo-local`
Acceptance criteria:
- Documented release-tooling/version claims match the actual workflow and scripts.
- Local verification and hosted release verification differences are explicit and truthful.
- Release contract tests fail if the locked release/install truth regresses.
Stable/internal boundary notes:
- Public:
  - release/install verification commands
  - release prerequisites
  - local vs hosted verification expectations
- Internal:
  - workflow step ordering that is not part of the documented contract
Migration expectations:
- No CLI migration.
- Contributor/release-manager instructions update in place.
Integration hooks:
- release managers
- maintainers running local release gates
- Homebrew/distribution validation surfaces
Dependencies:
- `W1-S1` should land first so CodeQL wording is already truthful before release docs are reconciled
Risks:
- Correcting only README copy without reconciling `product/dev_guides.md` and contract tests will reintroduce drift.

### Story W2-S3: Add public surface classification and expectation-safe launch docs
Priority: P2
Tasks:
- Add a short stable/internal/deprecated classification note for the public command and integration surfaces:
  - built-in collection
  - plugin collection
  - manual record append
  - sibling ingest
  - map/gaps/bundle/verify
- Link the authoritative manual proof-record contract from all launch-facing docs and docs-site surfaces.
- Preserve and reinforce the expectation-safe first-value story:
  - evidence
  - ranked gaps
  - intact local verification
  - not full completeness on first run
- Keep the ownership-boundary story explicit:
  - customer code/runtime
  - Axym
  - provider/upstream systems
- Update generated/public docs indexes so repo docs and `docs-site/public/llms.txt` stay aligned.
Repo paths:
- `README.md`
- `docs/commands/axym.md`
- `docs/operator/quickstart.md`
- `docs/operator/integration-model.md`
- `docs-site/public/llm/axym.md`
- `docs-site/public/llms.txt`
- `product/axym.md`
- `testinfra/contracts/command_docs_parity_contract_test.go`
- `testinfra/contracts/repo_hygiene_test.go`
Run commands:
- `go test ./testinfra/contracts -run 'Command|Launch|RepoHygiene' -count=1`
- `make test-docs-consistency`
- `make test-docs-storyline`
- `make test-docs-links`
- `make test-acceptance`
Test requirements:
- Docs/examples changes:
  - docs consistency checks
  - storyline checks
  - README/quickstart/integration coverage checks
  - docs source-of-truth sync for `README.md`, `docs/`, `docs-site/public/llms.txt`, and `docs-site/public/llm/*.md`
- Scenario/context changes:
  - acceptance coverage when first-value launch messaging or sample/integration contract wording changes
- CLI behavior changes:
  - none expected, but public-surface notes must not overstate internal-only implementation details as public API
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform
- Required gates:
  - launch/docs parity contract tests
  - `make test-docs-consistency`
  - `make test-docs-storyline`
  - `make test-docs-links`
Acceptance criteria:
- All launch-facing repo docs and docs-site surfaces tell the same expectation-safe story.
- Public docs clearly distinguish stable user-facing surfaces from internal-only details.
- The authoritative manual proof-record contract is linked anywhere `record add` is promoted as an integration path.
- No launch-facing doc claims complete compliance on first run.
Stable/internal boundary notes:
- Public:
  - supported CLI and integration surfaces today
  - stable/internal/deprecated classification
  - first-value expectation language
- Internal:
  - package names and implementation details not explicitly documented as supported extension points
Migration expectations:
- Additive docs clarification only.
- If any deprecated term or surface is introduced, it must carry explicit migration guidance in the same PR.
Integration hooks:
- operator onboarding
- docs-site/public LLM-friendly docs
- plugin/manual append/sibling ingest evaluation
Dependencies:
- `W1-S2`
- `W2-S2`
Risks:
- Adding a classification note without docs-site sync will leave search/LLM-facing docs stale and inconsistent.

---

## Minimum-Now Sequence

1. `W1-S1` - restore truthful hosted CodeQL behavior and security contracts.
2. `W1-S2` - publish the canonical manual proof-record contract and lock `record_version`.
3. `W1-S3` - close the governance-event no-input diagnostic gap.
4. `W2-S1` - make docs-consistency and docs-storyline truly PR-blocking.
5. `W2-S2` - reconcile release-tooling docs with real workflow behavior.
6. `W2-S3` - add public surface classification and expectation-safe launch-doc sync.

Sequence rationale:

- `W1-S1` and `W1-S2` are the launch no-go blockers identified by the audit and must be resolved before launch trust can be re-established.
- `W1-S3` belongs in the same early wave because it is a runtime contract/diagnostic issue, not a later docs-only polish item.
- `W2-S1` follows `W1` so the newly locked public contracts can actually be enforced in PRs.
- `W2-S2` and `W2-S3` then align the broader launch/release/public-surface story around the now-locked contracts instead of documenting unstable assumptions.

---

## Explicit Non-Goals

- No new collectors, evidence classes, or product-scope expansion beyond the currently shipped Axym CLI.
- No redesign of the sample pack, first-value counts, or bundle-completeness target.
- No new dashboard, hosted control plane, or enterprise-only branching of the OSS CLI.
- No new cleanup/reset/delete command surface.
- No changes to output-path safety semantics beyond tests/docs adjustments if touched incidentally.
- No re-architecture of proof emission, ingest translation, mapping, gaps, review, replay, or bundle assembly unrelated to the audit findings above.

---

## Definition of Done

- Every audit recommendation in scope maps to at least one explicit story in this plan.
- `W1` and `W2` are completed in order, with contract/runtime work landing before docs/onboarding/distribution work.
- Every story ships with matching tests and declared lane wiring.
- Public claims about hosted security analysis, manual append payloads, per-source collect diagnostics, merge-blocking docs checks, and release verification are all executable and enforced by repo tests or workflows.
- A new contributor or operator can:
  - understand what Axym promises publicly
  - find the authoritative manual `record add` contract
  - see truthful release/install/security guidance
  - distinguish supported stable surfaces from internal implementation details
- Launch reconsideration does not occur until:
  - technical preview gate: all `W1` stories are green
  - broad public OSS gate: all `W1` and `W2` stories are green
