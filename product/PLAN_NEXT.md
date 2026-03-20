# PLAN Axym: Launch Contract Truth and OSS Trust Baseline

Date: 2026-03-20
Source of truth: user-provided 2026-03-20 app-audit findings and fix-wave guidance, `product/axym.md`, `product/dev_guides.md`, `product/architecture_guides.md`, `AGENTS.md`
Scope: Axym OSS CLI only. Convert the current launch-readiness audit findings into an execution-ready backlog covering public onboarding contract truth, first-value sample contract codification, docs/source-of-truth validation, and OSS trust-baseline updates. No new product-scope expansion beyond Axym's existing CLI and `Clyra-AI/proof` interoperability.

---

## Global Decisions (Locked)

- This plan supersedes the prior `PLAN_NEXT.md` theme. The current top priority is launch-contract truth and OSS trust, not proof-integrity remediation.
- Truth-over-reshape is locked for this plan. Public docs and tests will align to the shipped first-value path rather than changing the sample pack to preserve an older story.
- The canonical first-value sample contract for this plan is:
  - `collect --json --governance-event-file ...` captures `4` governance events from the sample pack
  - the end-to-end sample journey yields `6` total chain records after manual approval and risk-assessment append
  - `map --frameworks eu-ai-act,soc2 --json` reports `5` covered controls out of `6`
  - `gaps --frameworks eu-ai-act,soc2 --json` reports grade `C`
  - `bundle --audit sample --frameworks eu-ai-act,soc2 --json` emits the required identity-governance artifacts
  - bundle compliance remains incomplete (`complete=false`) with `weak_record_count=1`
- Homebrew-installed binaries are invoked as `axym`; source builds and unpacked release binaries are invoked as `./axym`.
- The first-value promise is "non-empty evidence, ranked gaps, and intact local verification," not "full audit completeness on first run."
- Launch-truth checks must be executable and PR-blocking. String-presence-only docs tests are insufficient.
- Contract/runtime validation work must land before docs/onboarding/distribution copy changes.
- Public source-of-truth surfaces for this plan are:
  - `README.md`
  - `docs/commands/axym.md`
  - `docs/operator/quickstart.md`
  - `docs/operator/integration-model.md`
  - `docs-site/public/llm/axym.md`
  - `docs-site/public/llms.txt`
  - `CONTRIBUTING.md`
  - `SECURITY.md`
  - `CHANGELOG.md`
- Preserve Axym non-negotiables throughout:
  - determinism
  - offline-first defaults
  - fail-closed policy behavior
  - schema stability
  - exit-code stability
- No dashboard-first, SaaS-first, or enterprise-fork work belongs in this plan.

---

## Current Baseline (Observed)

- The core runtime is healthy. The audited repo passed `go build ./cmd/axym`, `make lint-fast`, `make test-fast`, `make test-contracts`, `make test-acceptance`, `make test-scenarios`, `make test-docs-consistency`, `make test-docs-storyline`, `make test-docs-links`, `make test-security`, `make lint-go`, `make codeql`, `make release-local`, and `make release-go-nogo-local`.
- Filesystem and bundle-verify safety are already fail-closed for unmanaged output directories and marker trust. This plan does not reopen those boundaries.
- The current first-value acceptance contract is too loose. `scenarios/axym/first_value_sample/contract.json` only enforces a minimum covered-count and forbids grade `F`, so it does not protect the exact public launch claims.
- `testinfra/contracts/command_docs_parity_contract_test.go`, `scripts/check_docs_consistency.sh`, and `scripts/check_docs_storyline.sh` check string presence/order only. They do not validate sample-path outcome truth.
- Existing install-doc parity checks normalize the wrong invocation mode across install paths by treating `./axym version --json` as the universal install snippet, including Homebrew-facing surfaces.
- Public launch docs currently drift from shipped behavior in two release-blocking ways:
  - Homebrew onboarding is inconsistent across launch surfaces
  - sample-path counts and claims do not match the shipped sample pack and actual `--json` results
- OSS trust docs are directionally good but still under-document full local gate prerequisites and keep the private security reporting path less explicit than ideal for a public launch.
- The launch verdict remains no-go until Wave 1 and Wave 2 are complete.

---

## Exit Criteria

1. All public install surfaces use the correct invocation mode:
   - Homebrew: `axym version --json`
   - source build: `./axym version --json`
   - release binary: `./axym version --json`
2. One machine-readable sample contract defines and acceptance-tests the shipped first-value journey with exact deterministic expectations for:
   - governance-event capture count
   - final chain record count
   - covered-control count
   - grade
   - required bundle artifacts
   - compliance completeness state
   - identity-governance weakness count
   - chain integrity
3. PR-blocking contract tests fail if any launch-facing doc drifts from the locked sample/install contract.
4. `README.md`, `docs/commands/axym.md`, `docs/operator/quickstart.md`, `docs-site/public/llm/axym.md`, and `docs-site/public/llms.txt` tell the same truthful smoke/sample/real-integration story.
5. First-value docs explicitly set expectations around "evidence + ranked gaps + intact chain" and truthfully retain the remaining sample gap/completeness state.
6. Contributor-facing docs distinguish fast-path setup from full local gate/release-local prerequisites and list the required external tools for those workflows.
7. `SECURITY.md` and linked public surfaces identify one explicit private reporting path and one consistent fallback/public path.
8. Wave `W1` and `W2` stories are green in their declared lanes before launch is reconsidered.

---

## Recommendation Traceability

| Recommendation | Why | Strategic direction | Expected moat/benefit | Planned stories |
|---|---|---|---|---|
| Fix Homebrew onboarding command drift | Wrong copy breaks the first install-verification step for some users | Install-surface truth and discoverability | Higher onboarding trust and lower day-one confusion | `W1-S2`, `W2-S1` |
| Fix sample-proof false counts/outcomes | Public launch claims currently disagree with shipped runtime behavior | Truthful first-value contract | Auditor/operator trust and lower reputational risk | `W1-S1`, `W2-S1` |
| Add semantic docs validation | Current string-only docs gates are too shallow to stop drift | PR-blocking executable launch contract | Prevents recurrence of launch-doc regressions | `W1-S1`, `W1-S2` |
| Clarify the first-value aha | The real win is evidence + ranked gaps, not full completeness | Expectation management without shrinking the wedge | Better activation quality and less launch disappointment | `W2-S1` |
| Document full-gate tooling prerequisites | Contributors currently discover missing tools late in the workflow | OSS trust-baseline clarity | Lower support burden and better contributor success | `W3-S1` |
| Make security/support expectations explicit | Ambiguous private-path language is risky for public launch | Governance clarity | Safer disclosure flow and clearer maintainer expectations | `W3-S2` |

---

## Test Matrix Wiring

Lane definitions:

- Fast lane:
  - `make lint-fast`
  - targeted `go test ./testinfra/contracts ... -count=1`
  - `make test-docs-consistency`
  - `make test-docs-storyline`
- Core CI lane:
  - `make test-fast`
  - `make test-contracts`
- Acceptance lane:
  - `go test ./testinfra/acceptance -run TestInstalledBinaryFirstValueSamplePath -count=1`
  - `make test-scenarios` when scenario/storyline fixtures change
- Cross-platform lane:
  - existing CLI/docs/contract coverage through `.github/workflows/main.yml` across Linux, macOS, and Windows for stories that affect CLI invocation examples, sample-path copy, or bundle/verification expectations
- Risk lane:
  - `make prepush-full`
  - `make release-local`
  - `make release-go-nogo-local`
  - targeted release/install verification when launch-facing install or release guidance changes

Story-to-lane map:

| Story | Fast | Core CI | Acceptance | Cross-platform | Risk |
|---|---|---|---|---|---|
| `W1-S1` | Yes | Yes | Yes | Yes | No |
| `W1-S2` | Yes | Yes | No | Yes | No |
| `W2-S1` | Yes | Yes | Yes | Yes | Yes |
| `W3-S1` | Yes | Yes | No | No | Yes |
| `W3-S2` | Yes | Yes | No | No | No |

Merge/release gating rule:

- Wave `W1` and Wave `W2` are launch-blocking and must complete in order.
- Any story that changes first-value sample claims must pass both the acceptance contract and the docs-contract lane in the same PR.
- Any story that changes install/release guidance must pass the relevant docs-contract lane and release-local verification in the same merge window.
- Wave `W3` is not required to lift the no-go verdict, but it is required for a stronger public OSS trust baseline.

---

## Epic W1: Launch Contract Codification

Objective: convert the current first-value and install story into executable, PR-blocking contract checks before changing public copy.

### Story W1-S1: Lock the exact first-value sample contract
Priority: P0
Tasks:
- Expand `scenarios/axym/first_value_sample/contract.json` from threshold-style checks to exact deterministic expectations for the shipped sample path.
- Extend `testinfra/acceptance/scenario_contract_test.go` to assert:
  - governance-event capture count
  - final chain record count
  - covered-control count
  - grade
  - required identity-governance artifacts
  - `complete=false`
  - `weak_record_count=1`
  - `verify --chain --json` intact state
- Add PR-blocking contract coverage that cross-checks launch-facing docs against the locked sample contract instead of relying on free-text snippets alone.
- Tighten `scripts/check_docs_consistency.sh` and `scripts/check_docs_storyline.sh` so they reflect the same exact sample-path contract or call a shared contract helper.
- Preserve fully offline, deterministic execution for the sample-path validation harness.
Repo paths:
- `scenarios/axym/first_value_sample/contract.json`
- `testinfra/acceptance/scenario_contract_test.go`
- `testinfra/contracts/command_docs_parity_contract_test.go`
- `testinfra/contracts/repo_hygiene_test.go`
- `scripts/check_docs_consistency.sh`
- `scripts/check_docs_storyline.sh`
Run commands:
- `go test ./testinfra/acceptance -run TestInstalledBinaryFirstValueSamplePath -count=1`
- `go test ./testinfra/contracts -run 'Launch|Command|RepoHygiene' -count=1`
- `make test-docs-consistency`
- `make test-docs-storyline`
- `axym init --sample-pack ./axym-sample --json`
- `axym collect --json --governance-event-file ./axym-sample/governance/context_engineering.jsonl`
- `axym record add --input ./axym-sample/records/approval.json --json`
- `axym record add --input ./axym-sample/records/risk_assessment.json --json`
- `axym bundle --audit sample --frameworks eu-ai-act,soc2 --json`
- `axym verify --chain --json`
Test requirements:
- Scenario/context changes:
  - exact acceptance checks for the first-value sample journey
- Docs/examples changes:
  - docs consistency checks
  - storyline/smoke checks when the user flow contract changes
  - launch-surface docs coverage checks across README, quickstart, docs-site, and command guide
- CLI behavior changes:
  - `--json` assertions for the sample-path commands exercised by the acceptance suite
- Determinism/hash/sign/packaging:
  - repeat-run stability for the sample-path contract
  - `make test-contracts`
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform
- Required gates:
  - `go test ./testinfra/acceptance -run TestInstalledBinaryFirstValueSamplePath -count=1`
  - `make test-contracts`
  - `make test-docs-consistency`
  - `make test-docs-storyline`
Acceptance criteria:
- One machine-readable contract defines the shipped first-value sample journey exactly.
- PR-blocking tests fail if launch-facing docs claim `3` governance captures or a `5`-record chain.
- The sample-path validation harness stays offline and deterministic.
- Any future intentional change to the sample journey must update the contract, docs, and tests in the same PR.
Stable/internal boundary notes:
- Public: first-value sample counts, remaining-gap messaging, and required artifact claims are launch-contract surfaces.
- Internal: helper/test harness placement is implementation detail.
Migration expectations:
- This is a contract-tightening change only. No runtime migration is introduced.
Integration hooks:
- Operator first-10-minute onboarding
- docs-site/public launch content
- search/LLM-facing summaries that quote the sample path
Dependencies:
- None
Risks:
- Lower-bound-only assertions will recreate the current drift and are insufficient.

### Story W1-S2: Split install-surface parity by invocation mode
Priority: P0
Tasks:
- Replace universal install-doc checks with path-specific checks:
  - Homebrew => `axym version --json`
  - source build => `./axym version --json`
  - release binary => `./axym version --json`
- Extend install-surface parity coverage to include `docs/operator/quickstart.md` and `docs-site/public/llms.txt`.
- Keep release/install contract coverage aligned with the new invocation rules without weakening install discoverability checks.
- Ensure PR-blocking contract tests catch mixed invocation-mode regressions before merge.
Repo paths:
- `testinfra/contracts/command_docs_parity_contract_test.go`
- `testinfra/contracts/release_gate_contract_test.go`
- `scripts/check_docs_consistency.sh`
- `scripts/check_docs_storyline.sh`
- `docs-site/public/llms.txt`
Run commands:
- `go test ./testinfra/contracts -run 'CommandInstallSurfaceDocsParity|LaunchStory|ReleaseGoNoGo' -count=1`
- `make test-docs-consistency`
- `make test-docs-storyline`
Test requirements:
- Docs/examples changes:
  - install-surface coverage across README, command guide, quickstart, docs-site/public/llm, and docs-site/public/llms.txt
- CLI behavior changes:
  - install/version discoverability contract stays truthful where docs reference `version --json`
- Contract checks:
  - path-specific install parity becomes merge-blocking
Matrix wiring:
- Lanes: Fast, Core CI, Cross-platform
- Required gates:
  - targeted contract tests
  - `make test-docs-consistency`
  - `make test-docs-storyline`
Acceptance criteria:
- Contract tests distinguish Homebrew vs source/release binary invocation correctly.
- Any doc that reintroduces `./axym version --json` under a Homebrew heading fails in PR.
- Existing source and release-binary guidance remains covered and truthful.
Stable/internal boundary notes:
- Public: install snippets are part of install/version discoverability contract.
- Internal: choice of helper implementation is not part of the public contract.
Migration expectations:
- Additive test tightening only. No CLI runtime behavior changes.
Integration hooks:
- Homebrew install verification
- source-build verification
- release download smoke verification
Dependencies:
- None
Risks:
- If quickstart and docs-site index surfaces are omitted, the same bug can recur outside the most obvious docs pages.

---

## Epic W2: Public Launch Surface Truth Alignment

Objective: rewrite launch-facing docs to match the locked sample/install contract and set truthful first-value expectations.

### Story W2-S1: Align public launch docs to the shipped sample and install contract
Priority: P0
Tasks:
- Update `README.md`, `docs/commands/axym.md`, `docs/operator/quickstart.md`, `docs-site/public/llm/axym.md`, and `docs-site/public/llms.txt` to use the correct Homebrew invocation.
- Update sample-path wording across all launch surfaces to reflect the locked contract:
  - `4` governance-event captures
  - `6` total records after manual append
  - `5/6` covered controls
  - grade `C`
  - intact chain verification
  - remaining sample gap/completeness state
- Reframe the first-value promise so it explicitly sells evidence + ranked gaps + intact chain rather than full audit completeness.
- Keep smoke vs sample vs real-integration path separation intact and preserve current runtime-boundary language.
- Update `CHANGELOG.md` with the user-visible launch-contract docs correction.
- Refresh `docs/operator/integration-model.md` only where first-value path language or boundary wording must match the launch surfaces exactly.
Repo paths:
- `README.md`
- `docs/commands/axym.md`
- `docs/operator/quickstart.md`
- `docs/operator/integration-model.md`
- `docs-site/public/llm/axym.md`
- `docs-site/public/llms.txt`
- `CHANGELOG.md`
Run commands:
- `make test-docs-consistency`
- `make test-docs-storyline`
- `make test-docs-links`
- `go test ./testinfra/contracts -run 'Command|Launch|RepoHygiene' -count=1`
- `go test ./testinfra/acceptance -run TestInstalledBinaryFirstValueSamplePath -count=1`
- `axym init --sample-pack ./axym-sample --json`
- `axym map --frameworks eu-ai-act,soc2 --json`
- `axym gaps --frameworks eu-ai-act,soc2 --json`
- `axym bundle --audit sample --frameworks eu-ai-act,soc2 --json`
Test requirements:
- Docs/examples changes:
  - docs consistency checks
  - storyline/smoke checks
  - README/quickstart/integration coverage checks
  - docs source-of-truth sync for README, `docs/`, `docs-site/public/llms.txt`, and `docs-site/public/llm/*.md`
- CLI behavior changes:
  - launch docs must remain aligned with existing `--json` command surfaces
- Scenario/context changes:
  - installed-binary first-value acceptance suite remains green with exact expectations
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk
- Required gates:
  - docs scripts
  - targeted contract tests
  - installed-binary acceptance test
  - `make prepush-full`
  - `make release-local`
  - `make release-go-nogo-local`
Acceptance criteria:
- All public launch docs tell the same truthful install and sample story.
- The docs no longer claim `3` governance captures or a `5`-record chain.
- Homebrew users are told to run `axym version --json`.
- The first-value journey is explicitly framed as evidence + ranked gaps + intact chain with remaining sample incompleteness called out truthfully.
Stable/internal boundary notes:
- Public: README, operator docs, command guide, docs-site/public LLM docs, and changelog are public contract surfaces.
- Internal: the sample pack remains unchanged in this plan as long as docs match the shipped behavior.
Migration expectations:
- No runtime migration. Existing user commands continue to work.
- External references may still cite old counts; the changelog should make the correction discoverable.
Integration hooks:
- Website/docs-site ingestion
- search/LLM docs consumers
- Homebrew onboarding
- operator onboarding and evaluation workflows
Dependencies:
- `W1-S1`
- `W1-S2`
Risks:
- Partial docs updates will keep stale claims alive through search, snippets, and LLM ingestion even if one source-of-truth file is corrected.

---

## Epic W3: OSS Trust Baseline and Support/Distribution Clarity

Objective: close the non-blocking but meaningful OSS readiness gaps after launch-contract truth is restored.

### Story W3-S1: Document full local gate and release-tool prerequisites
Priority: P1
Tasks:
- Update `CONTRIBUTING.md` to separate the Go-only fast path from the full local gate and release-local workflows.
- Explicitly list the required external tools for:
  - `make prepush-full`
  - `make release-local`
  - `make release-go-nogo-local`
- Cover at least:
  - `golangci-lint`
  - `gosec`
  - `codeql`
  - `syft`
  - `cosign`
- Clarify which workflows are expected for normal contributors versus maintainers/release managers.
- Extend repo-hygiene/docs checks so prerequisite mentions are required where full-gate commands are documented publicly.
Repo paths:
- `CONTRIBUTING.md`
- `README.md`
- `docs-site/public/llms.txt`
- `testinfra/contracts/repo_hygiene_test.go`
- `scripts/check_docs_consistency.sh`
Run commands:
- `go test ./testinfra/contracts -run 'RepoHygiene|Command' -count=1`
- `make test-docs-consistency`
- `make test-docs-links`
Test requirements:
- Docs/examples changes:
  - OSS trust-baseline updates
  - docs source-of-truth sync
- Contract checks:
  - prerequisite mentions become deterministic where full-gate commands are documented
Matrix wiring:
- Lanes: Fast, Core CI, Risk
- Required gates:
  - targeted contract tests
  - `make test-docs-consistency`
  - `make test-docs-links`
Acceptance criteria:
- Contributor docs clearly distinguish fast-path setup from full local gate/release-local prerequisites.
- Required tools are listed explicitly anywhere the full gate is recommended.
- The docs do not imply that normal operator usage requires maintainer-only release tooling.
Stable/internal boundary notes:
- Public: contributor/setup expectations are OSS trust surfaces.
- Internal: exact phrasing may evolve if the tool list and command scopes stay explicit.
Migration expectations:
- No runtime changes. Contributor expectation-setting only.
Integration hooks:
- external contributors
- maintainers
- release operators
Dependencies:
- `W2-S1`
Risks:
- If prerequisites remain implicit, contributors will keep discovering failures late and perceive the project as fragile.

### Story W3-S2: Make private security reporting and support expectations explicit
Priority: P2
Tasks:
- Update `SECURITY.md` to identify one explicit private reporting path and one fallback/public path.
- Default to GitHub Security Advisories as the named private path unless the repo policy changes in the same work window.
- Tighten `CONTRIBUTING.md`, `README.md`, and `docs-site/public/llms.txt` references so security-sensitive reports are redirected consistently to `SECURITY.md`.
- Update `.github/ISSUE_TEMPLATE/bug_report.yml` if needed so public bug intake does not imply that security issues belong in a normal public bug thread.
- Keep best-effort OSS support expectations explicit and avoid implying informal or private maintainer channels that are not part of the public contract.
Repo paths:
- `SECURITY.md`
- `CONTRIBUTING.md`
- `README.md`
- `docs-site/public/llms.txt`
- `.github/ISSUE_TEMPLATE/bug_report.yml`
- `CODE_OF_CONDUCT.md`
Run commands:
- `go test ./testinfra/contracts -run 'RepoHygiene' -count=1`
- `make test-docs-consistency`
- `make test-docs-links`
Test requirements:
- Docs/examples changes:
  - OSS trust-baseline updates
  - source-of-truth sync for linked governance/support surfaces
- Repo hygiene checks:
  - required launch assets and references remain intact
Matrix wiring:
- Lanes: Fast, Core CI
- Required gates:
  - targeted contract tests
  - `make test-docs-consistency`
  - `make test-docs-links`
Acceptance criteria:
- `SECURITY.md` names one explicit private path and one fallback/public path.
- `CONTRIBUTING.md`, `README.md`, and the bug template no longer imply unspecified private support/security channels.
- Launch-facing docs remain consistent about best-effort OSS support and security escalation.
Stable/internal boundary notes:
- Public: security reporting path and support expectations are launch-trust surfaces.
- Internal: future routing changes must update `SECURITY.md` and linked docs together.
Migration expectations:
- No runtime changes. Governance clarity only.
Integration hooks:
- vulnerability disclosure
- public issue filing
- maintainer triage
Dependencies:
- `W2-S1`
Risks:
- Ambiguous disclosure guidance creates avoidable public-reporting risk during launch.

---

## Minimum-Now Sequence

1. `W1-S1` first.
   Reason: the sample-path truth must be machine-checked before any copy is edited, or the plan will just replace one unverified story with another.
2. `W1-S2` second.
   Reason: the Homebrew invocation bug needs path-specific install parity checks before docs are updated, so the same mismatch cannot recur in a follow-up PR.
3. `W2-S1` third.
   Reason: once the contract is locked and enforced, all public launch surfaces can be updated in one coherent, truth-aligned pass.
4. `W3-S1` fourth.
   Reason: contributor prerequisite clarity depends on the launch docs already being stable, but it is not required to lift the current launch no-go.
5. `W3-S2` fifth.
   Reason: security/support governance wording should land after the main public surfaces are synchronized so it can reference the final launch docs cleanly.

Wave order rationale:

- Wave `W1` is contract-locking and drift-prevention work.
- Wave `W2` is the release-blocking public docs alignment that must satisfy Wave `W1`.
- Wave `W3` improves OSS trust baseline without changing product/runtime semantics.

---

## Explicit Non-Goals

- No change to core sample-pack contents or runtime behavior to force the old `3`-capture/`5`-record story back into existence.
- No change to collector coverage, framework coverage, bundle layout, proof semantics, or exit-code vocabulary.
- No UI, dashboard, hosted service, or enterprise-support scope.
- No broad positioning rewrite outside truthful first-value expectation setting and OSS trust-baseline clarity.
- No architectural boundary changes to collect, normalize, proof emit, ingest, map/gaps, review/regress, bundle, or verify packages.

---

## Definition of Done

- Every recommendation in this audit-derived plan maps to at least one story in `W1` through `W3`.
- Wave `W1` and Wave `W2` are green in their required lanes and lift the current public-launch blockers.
- One exact, machine-readable first-value sample contract exists and is enforced by acceptance and contract suites.
- Launch docs are semantically validated against shipped runtime behavior, not just checked for string presence.
- Homebrew/source/release install guidance is correct and consistent across all source-of-truth docs.
- Contributor docs explicitly list full-gate/release-local prerequisites and distinguish them from the fast path.
- `SECURITY.md`, linked docs, and public issue intake surfaces present consistent security/support expectations.
- `product/PLAN_NEXT.md` now reflects the active launch-remediation backlog and no longer carries the replaced proof-integrity plan theme.
