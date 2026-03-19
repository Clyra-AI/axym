# PLAN Launch-Ready OSS: Truthful First Value and Public Trust Baseline


Date: 2026-03-19
Source of truth: user-provided 2026-03-19 audit findings, `product/axym.md`, `product/dev_guides.md`, `product/architecture_guides.md`, `README.md`, `docs/commands/axym.md`, `docs-site/public/llms.txt`, `AGENTS.md`
Scope: Axym OSS CLI only. Plan the minimum backlog required to convert the current no-go public-launch audit into a truthful, launch-ready OSS release without weakening determinism, offline-first defaults, fail-closed behavior, schema stability, or exit-code stability.


---


## Global Decisions (Locked)


- Choose a truth-aligned public launch over widening built-in collector scope in this backlog. This plan does not add new first-party approval, guardrail, permission-check, incident, or risk-register collectors just to satisfy launch messaging.
- Preserve Axym's contract surfaces: deterministic outputs, offline-first defaults, local evidence handling, stable `--json` envelopes, and stable exit codes.
- Treat the first-value path as a release contract. The installed-binary operator path must produce non-empty evidence locally without requiring repo-only fixtures or a hosted service.
- Split public user journeys into three explicit modes: `smoke test`, `sample proof path`, and `real integration path`.
- Keep architecture boundaries intact: onboarding/sample assets may assist adoption, but they must not collapse collection, normalization, proof emission, mapping, or verification boundaries.
- Keep CLI orchestration thin. Any onboarding/sample-pack behavior must live behind explicit names and side effects rather than implicit downloads or hidden network fetches.
- Public docs must distinguish built-in Axym collection, plugin collection, manual record append, and sibling ingest. Do not describe those as a single default behavior.
- `README.md`, `docs/commands/axym.md`, `docs-site/public/llms.txt`, and `docs-site/public/llm/axym.md` are one launch-facing docs contract and must be updated together.
- OSS trust baseline is part of launch scope: `LICENSE` is release-blocking, and `CHANGELOG`, `CODE_OF_CONDUCT`, issue templates, PR template, and maintainer/support expectations must exist before broad external adoption.
- No dashboard-first scope, hosted demo path, or remote sample fetch flow is allowed in this backlog.


---


## Current Baseline (Observed)


- `go build ./cmd/axym` passed locally during the 2026-03-19 audit.
- `make prepush` passed locally during the 2026-03-19 audit, including package, E2E, integration, acceptance, and contract test lanes.
- The clean quickstart path (`init -> collect --dry-run -> collect -> map -> gaps -> bundle -> verify`) succeeds mechanically on a fresh directory but returns `captured: 0`, `coverage: 0`, grade `F`, and an empty verified chain.
- The built-in collector registry currently exposes `mcp`, `llmapi`, `webhook`, `githubactions`, `gitmeta`, `dbt`, `snowflake`, `governanceevent`, and optional plugins. That shipped surface is materially narrower than the broader product narrative.
- A fixture-backed flow can append evidence and verify successfully, but the audited run still produced only `1` covered control out of `6`, grade `E`, and `incomplete_controls: 5`.
- `README.md`, `docs/commands/axym.md`, and `docs-site/public/llm/axym.md` currently present the empty clean-room path as "First value" / "First 15 minutes".
- `CONTRIBUTING.md` and `SECURITY.md` exist in the repo today.
- `LICENSE`, `CHANGELOG*`, `CODE_OF_CONDUCT*`, `.github/ISSUE_TEMPLATE/**`, and `.github/pull_request_template*` are currently absent.
- `docs-site/public/llms.txt` already exists and must remain in sync with the public docs surface.


---


## Exit Criteria


1. An installed Axym binary can produce a supported local sample proof path with no network dependency and no repo fixture dependency.
2. The documented first-value journey ends in non-empty evidence and a non-empty compliance result, with explicit expected outcomes stated in docs and acceptance tests.
3. Public docs no longer imply that clean `collect --json` on a fresh environment will capture meaningful default evidence without inputs.
4. Public docs clearly distinguish `smoke test`, `sample proof path`, and `real integration path`.
5. Public docs accurately distinguish built-in collectors from plugin/manual/ingest paths and do not market unshipped built-in coverage.
6. `LICENSE` exists at repo root and is discoverable from public-facing docs.
7. `CHANGELOG.md`, `CODE_OF_CONDUCT.md`, issue templates, PR template, and maintainer/support expectations exist and are linked from the public docs surface.
8. Docs source-of-truth checks fail when README, command docs, docs-site summaries, or operator docs drift on install, first value, supported surfaces, or verification behavior.
9. All touched stories preserve stable CLI help, `--json`, install/version discoverability, and exit-code contracts.
10. The repo can credibly move from "no-go" to "go" for public OSS launch without claiming unimplemented coverage breadth.


---


## Recommendation Traceability


| Recommendation | Why | Strategic direction | Expected moat/benefit | Planned stories |
|---|---|---|---|---|
| Repair the first-run path so clean-room users get real value | Current quickstart yields `captured: 0`, `coverage: 0`, and grade `F` in a clean environment | Add a supported offline sample proof path and test it as a public contract | Converts install curiosity into a believable product aha without hosted dependencies | `W1-S1`, `W1-S2` |
| Align public claims with shipped collector surfaces and observed demo coverage | Launch-facing messaging currently outruns built-in collection breadth | Narrow public claims to shipped behavior and distinguish built-in vs plugin/manual/ingest paths | Preserves trust and prevents reputational damage from expectation mismatch | `W1-S2` |
| Split operator quickstart from contributor setup and document a non-empty sample path | Install friction is low, but operator first value is hidden behind repo fixtures and implicit context | Publish separate smoke-test and sample-proof journeys | Improves adoption without conflating contributor and operator concerns | `W1-S1`, `W1-S2`, `W2-S2` |
| Add a top-level license file | Repo is described as open-source but currently has no `LICENSE` | Add legal distribution baseline before public launch | Removes a hard launch blocker and clarifies reuse terms | `W1-S3` |
| Add CHANGELOG, CODE_OF_CONDUCT, issue templates, and PR template | Public project governance baseline is incomplete | Add OSS trust assets and enforce their presence | Makes Axym safer to adopt and contribute to publicly | `W2-S1` |
| Add maintainer/support expectations, operator docs links, and an integration-boundary diagram | Users currently have weak guidance on ownership, support path, and runtime placement | Publish explicit operator docs and governance expectations | Sharpens the wedge and reduces support ambiguity | `W2-S1`, `W2-S2` |


---


## Test Matrix Wiring


Lane definitions:


- Fast lane: targeted unit, contract, docs-consistency, and repo-hygiene checks that must pass on every PR.
- Core CI lane: primary Linux CI coverage for touched CLI, contract, docs, and workflow tests.
- Acceptance lane: end-to-end operator-path checks for documented first-value and launch-facing workflows.
- Cross-platform lane: macOS and Windows validation for any story that changes public command surfaces, path behavior, or install-facing output.
- Risk lane: release-blocking checks for runtime/onboarding contract changes, deterministic sample-pack behavior, docs source-of-truth drift, and launch-surface regressions.


Story-to-lane map:


| Story | Fast | Core CI | Acceptance | Cross-platform | Risk |
|---|---|---|---|---|---|
| W1-S1 | Yes | Yes | Yes | Yes | Yes |
| W1-S2 | Yes | Yes | Yes | No | Yes |
| W1-S3 | Yes | Yes | No | No | No |
| W2-S1 | Yes | Yes | No | No | No |
| W2-S2 | Yes | Yes | Yes | No | No |


Merge/release gating rule:


- PR merge blocks on every touched story's Fast and Core CI lanes.
- Any story marked `Acceptance: Yes` is incomplete until the documented operator path passes end-to-end in CI.
- Any story marked `Cross-platform: Yes` is incomplete until Linux, macOS, and Windows public command-surface checks all pass.
- Any story marked `Risk: Yes` is release-blocking until its docs/runtime contract checks are green.
- Public launch is still blocked until all Wave 1 stories and all OSS-baseline Wave 2 stories are complete.


---


## Epic W1: Launch-Blocking First Value and Public Contract Repair


Objective: remove the current launch blockers by making Axym's first-value path work after install, making the public surface truthful about what is shipped today, and adding the legal minimum for OSS distribution.


### Story W1-S1: Add a supported offline sample proof path for installed-binary first value
Priority: P0
Tasks:
- Add an explicit onboarding surface that materializes a deterministic local sample input pack after install, without requiring repo fixtures or a hosted fetch. Preferred direction: an additive `init` option such as `--sample-pack <dir>` or an equivalently explicit setup command.
- Ensure the sample pack contains only local deterministic assets needed to drive a credible first-value flow, for example governance-event input plus manual proof-record payloads for high-signal coverage types such as approval and risk-assessment evidence.
- Keep side effects explicit in command naming and JSON output. The generated sample-pack path, created files, and next-step commands must be machine-readable when `--json` is used.
- Add acceptance coverage proving the documented installed-binary path yields non-empty evidence, meaningful map/gaps output, a successful bundle, and successful verification.
- Preserve offline-first defaults, stable exit codes, and stable help behavior when the new onboarding option is omitted.
Repo paths:
- `cmd/axym/init.go`
- `cmd/axym/init_test.go`
- `core/samplepack/pack.go`
- `core/samplepack/pack_test.go`
- `internal/e2e/cli/command_surface_contract_test.go`
- `testinfra/acceptance/scenario_contract_test.go`
- `scenarios/axym/first_value_sample/`
Run commands:
- `go test ./cmd/axym ./core/samplepack -count=1`
- `go test ./internal/e2e/cli ./testinfra/acceptance -count=1`
- `./axym init --sample-pack ./axym-sample --json`
- `./axym collect --json --governance-event-file ./axym-sample/governance/context_engineering.jsonl`
- `./axym record add --input ./axym-sample/records/approval.json --json`
- `./axym record add --input ./axym-sample/records/risk_assessment.json --json`
- `./axym map --frameworks eu-ai-act,soc2 --json`
- `./axym gaps --frameworks eu-ai-act,soc2 --json`
- `./axym bundle --audit sample --frameworks eu-ai-act,soc2 --json`
- `./axym verify --chain --json`
Test requirements:
- Tier 1: unit tests for sample-pack generation, path validation, and deterministic asset contents.
- Tier 3: CLI help/usage, `--json`, and exit-code tests for the new onboarding surface.
- Tier 4: acceptance tests for the documented installed-binary first-value flow.
- Tier 5: explicit side-effect and atomic-write tests if sample-pack creation adds new filesystem write paths.
- Tier 9: command contract tests ensuring additive output stays stable and deterministic.
- Tier 11: scenario fixtures for the published first-value sample journey.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
Acceptance criteria:
- A fresh install can materialize a local sample pack without repo fixture paths or network access.
- The documented first-value path yields non-empty local evidence and `map` output with `covered_count >= 2` across the documented flow.
- `gaps` no longer returns grade `F` for the published sample-proof journey.
- Existing `init --json` behavior remains backward-compatible when the sample-pack option is not used.
Stable/internal boundary notes:
- Public: the onboarding surface and its `--json` output become part of the supported install/first-value contract.
- Internal: sample assets may be template-backed or embedded, but the user-visible command semantics and deterministic outputs must stay stable.
Migration expectations:
- This is additive only. Existing `init` users do not need to change commands unless they want the published sample path.
- If a new flag or subcommand is introduced, install docs and version-discoverability docs must be updated in the same story.
Integration hooks:
- This sample path is for local evaluation only and must be documented separately from real production integration.
Dependencies:
- None.
Risks:
- A weak sample pack that still ends in trivial coverage would preserve the audit failure in a new wrapper. Gate the result with acceptance thresholds, not prose alone.


### Story W1-S2: Reframe and machine-check the public first-value and supported-surface contract
Priority: P0
Tasks:
- Rewrite launch-facing docs around an explicit sequence: problem -> who it is for -> where Axym sits in the runtime path -> built-in supported surfaces today -> smoke test -> sample proof path -> real integration path.
- Remove or clearly qualify claims that imply built-in default capture of approvals, guardrails, permission checks, incidents, risk assessments, or broader coverage than currently shipped.
- Distinguish customer code, Axym-owned logic, and upstream tool/provider outputs so operators can see what they must connect themselves.
- Split contributor setup from operator onboarding; contributor commands stay in contributor sections, while first-value/operator commands live in launch-facing sections.
- Extend docs consistency/storyline checks so README, command docs, docs-site index, and docs-site LLM page cannot drift on install steps, first-value commands, supported surfaces, or verify behavior.
- Add a launch-facing annotation in product/docs source-of-truth where needed so current OSS baseline is distinguishable from longer-horizon PRD ambition.
Repo paths:
- `README.md`
- `docs/commands/axym.md`
- `docs-site/public/llms.txt`
- `docs-site/public/llm/axym.md`
- `product/axym.md`
- `scripts/check_docs_consistency.sh`
- `scripts/check_docs_storyline.sh`
- `testinfra/contracts/command_docs_parity_contract_test.go`
- `testinfra/contracts/repo_hygiene_test.go`
Run commands:
- `make test-docs-consistency`
- `make test-docs-storyline`
- `make test-docs-links`
- `go test ./testinfra/contracts -count=1`
- `./axym collect --dry-run --json`
- `./axym map --frameworks eu-ai-act,soc2 --json`
- `./axym verify --bundle ./axym-evidence --json`
Test requirements:
- Tier 3: public command help/usage contract checks for any onboarding surface updated in docs.
- Tier 4: storyline/smoke checks proving docs map to a runnable operator flow.
- Tier 8 docs/examples: docs consistency, storyline checks, README/quickstart/integration coverage checks, and docs-site source-of-truth sync tests.
- Tier 9: command/docs parity checks for launch-facing command examples and exit semantics.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Risk.
Acceptance criteria:
- Public docs no longer present clean `collect --json` on a fresh environment as a non-empty first-value path.
- Public docs explicitly distinguish built-in, plugin, manual, and sibling-ingest evidence paths.
- README, command docs, and docs-site content stay in sync under automated checks.
- Launch-facing docs describe what the user integrates, what Axym does, and what remains upstream/provider-owned.
Stable/internal boundary notes:
- Public: launch-facing docs are part of the OSS contract and must not outpace shipped behavior.
- Internal: the PRD may remain broader, but shipped-surface annotations must make current OSS scope explicit.
Migration expectations:
- No command removals are allowed in this story.
- Any new onboarding flag or command introduced by `W1-S1` must be reflected additively and consistently across all docs surfaces.
Integration hooks:
- Operators must be able to see where Axym attaches in CI/runtime and which steps still belong to their pipeline, source tool, or provider.
Dependencies:
- `W1-S1`
Risks:
- Messaging-only updates without automated drift tests will regress quickly as command surfaces evolve.


### Story W1-S3: Add the release-blocking OSS license baseline
Priority: P0
Tasks:
- Add a top-level `LICENSE` file using the maintainer-approved OSS license for Axym.
- Link the license from launch-facing docs so install/evaluation users can discover terms without hunting through the repo.
- Add or extend repo-hygiene contract tests so future removals or renames fail CI.
- Confirm release/distribution docs do not describe Axym as open-source without an actual root license file.
Repo paths:
- `LICENSE`
- `README.md`
- `CONTRIBUTING.md`
- `docs-site/public/llms.txt`
- `testinfra/contracts/repo_hygiene_test.go`
Run commands:
- `go test ./testinfra/contracts -count=1`
- `make test-docs-links`
- `make test-docs-consistency`
Test requirements:
- Tier 8 docs/examples: OSS trust-baseline checks for public launch assets.
- Tier 9: repo hygiene tests for required root assets and launch-facing links.
Matrix wiring:
- Lanes: Fast, Core CI.
Acceptance criteria:
- `LICENSE` exists at repo root and is linked from launch-facing docs.
- CI fails if the license file is removed or renamed without contract updates.
- The repo no longer presents itself as open-source while missing a root license.
Stable/internal boundary notes:
- Public: the chosen license becomes part of the distribution contract.
- Internal: any future license change requires explicit governance review and changelog coverage.
Migration expectations:
- No CLI/runtime behavior changes.
Integration hooks:
- External users and downstream packagers can evaluate reuse terms directly from the repo root.
Dependencies:
- None.
Risks:
- Leaving license choice implicit during implementation creates a false sense of launch readiness; pick and lock it in this wave.


---


## Epic W2: OSS Governance and Operator Docs Baseline


Objective: complete the public-project trust baseline and publish operator-facing docs that explain support expectations, contribution paths, and Axym's runtime boundary model without requiring internal context.


### Story W2-S1: Add OSS governance assets and maintainer-support expectations
Priority: P1
Tasks:
- Add `CHANGELOG.md`, `CODE_OF_CONDUCT.md`, `.github/ISSUE_TEMPLATE/bug_report.yml`, `.github/ISSUE_TEMPLATE/feature_request.yml`, and `.github/pull_request_template.md`.
- Expand `CONTRIBUTING.md` with maintainer/support expectations, supported reporting paths, response-boundary notes, and explicit distinctions between bug reports, feature requests, and security issues.
- Link governance assets from `README.md` and docs-site index surfaces so external users can discover them from the first screen.
- Add repo-hygiene coverage so required OSS launch assets remain enforced in CI.
Repo paths:
- `CHANGELOG.md`
- `CODE_OF_CONDUCT.md`
- `.github/ISSUE_TEMPLATE/bug_report.yml`
- `.github/ISSUE_TEMPLATE/feature_request.yml`
- `.github/pull_request_template.md`
- `CONTRIBUTING.md`
- `README.md`
- `docs-site/public/llms.txt`
- `testinfra/contracts/repo_hygiene_test.go`
Run commands:
- `go test ./testinfra/contracts -count=1`
- `make test-docs-links`
- `make test-docs-consistency`
Test requirements:
- Tier 8 docs/examples: OSS trust-baseline and public-link coverage checks.
- Tier 9: repo hygiene tests for required governance and contribution assets.
Matrix wiring:
- Lanes: Fast, Core CI.
Acceptance criteria:
- OSS governance baseline files exist, are linked, and are enforced by tests.
- `CONTRIBUTING.md` clearly states maintainer/support expectations and how users should route bugs, features, and security issues.
- Public launch docs do not require users to infer support norms from internal context.
Stable/internal boundary notes:
- Public: support and contribution expectations are part of the project contract.
- Internal: maintainers may evolve process details, but discovery paths and support boundaries must remain explicit.
Migration expectations:
- No CLI/runtime changes.
Integration hooks:
- External contributors have a clear path for issues, PRs, and responsible disclosure.
Dependencies:
- `W1-S3`
Risks:
- Governance assets without links or CI enforcement will exist on paper but remain invisible to users.


### Story W2-S2: Publish operator docs and the Axym integration-boundary model
Priority: P1
Tasks:
- Add dedicated operator docs that explain where Axym sits in the runtime path, what belongs to customer code vs Axym vs tool/provider, and how sync vs async evidence paths behave.
- Add an integration-boundary diagram or Mermaid source showing the `smoke test`, `sample proof path`, and `real integration path` relationships.
- Link the operator docs and diagram from README, command docs, and docs-site surfaces.
- Extend docs source-of-truth checks so operator docs participate in consistency, storyline, and link validation.
- Ensure the operator docs call out failure handling and expected outputs for the published first-value path.
Repo paths:
- `docs/operator/quickstart.md`
- `docs/operator/integration-model.md`
- `docs/operator/integration-boundary.mmd`
- `README.md`
- `docs/commands/axym.md`
- `docs-site/public/llms.txt`
- `docs-site/public/llm/axym.md`
- `scripts/check_docs_consistency.sh`
- `scripts/check_docs_storyline.sh`
- `scripts/check_docs_links.sh`
Run commands:
- `make test-docs-consistency`
- `make test-docs-storyline`
- `make test-docs-links`
- `go test ./testinfra/contracts -count=1`
- `./axym init --sample-pack ./axym-sample --json`
- `./axym collect --json --governance-event-file ./axym-sample/governance/context_engineering.jsonl`
- `./axym verify --chain --json`
Test requirements:
- Tier 4: operator-path storyline checks against the published quickstart/operator docs.
- Tier 8 docs/examples: README/quickstart/integration coverage checks and docs-source-of-truth sync checks across `README.md`, `docs/`, and `docs-site/public/*`.
- Tier 9: docs parity checks for newly linked operator-doc surfaces.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance.
Acceptance criteria:
- External users can identify where Axym integrates, what it owns, what upstream systems must provide, and which path to use first.
- Operator docs explicitly distinguish smoke testing from the sample proof path and from real production integration.
- The integration-boundary diagram is linked from launch-facing docs and stays validated by docs checks.
Stable/internal boundary notes:
- Public: operator docs become the launch-facing source of truth for integration shape and first-run expectations.
- Internal: ADRs remain engineering decision records, not a substitute for operator onboarding.
Migration expectations:
- No command removals; this story is additive on top of Wave 1 command/docs changes.
Integration hooks:
- Operators must be able to map Axym to CI pipelines, runtime evidence sources, and sibling-ingest workflows without needing internal product context.
Dependencies:
- `W1-S1`
- `W1-S2`
- `W2-S1`
Risks:
- If the operator docs duplicate README prose without shared checks, the launch contract will drift again.


---


## Minimum-Now Sequence


Wave order rationale:


- Wave 1 lands first because it fixes the launch-blocking contract: first value, shipped-surface truthfulness, and legal OSS distribution baseline.
- Wave 2 lands after Wave 1 because OSS governance assets and richer operator docs only help once the launch story is truthful and runnable.


Recommended implementation order:


1. `W1-S1` because every later quickstart/operator doc must describe a real installed-binary first-value path.
2. `W1-S3` in parallel or immediately after `W1-S1`; it is independent technically but must be complete before any public launch announcement or release tag.
3. `W1-S2` after `W1-S1`, so docs and contract tests can target the actual supported onboarding surface and shipped coverage floor.
4. `W2-S1` once the public launch surface is stable enough to link governance assets and support expectations.
5. `W2-S2` last, so operator docs and the integration-boundary diagram reflect the finalized Wave 1 contract and the finalized governance/support links.


Minimum-now release gate:


- Do not announce or tag a public OSS launch until every Wave 1 story is complete.
- Treat Wave 2 as mandatory before broad external adoption or contributor outreach, even if a narrow soft launch follows Wave 1 internally.


---


## Explicit Non-Goals


- No new first-party production collectors for approvals, guardrails, permission checks, incidents, or enterprise risk systems in this backlog.
- No hosted demo service, telemetry-backed onboarding, or remote fixture download flow.
- No dashboard/UI work.
- No changes to proof-chain integrity, bundle verification semantics, or exit-code contracts beyond additive onboarding/public-doc behavior required by this plan.
- No broad rewrite of the PRD into current-state-only language; only launch-facing scope calibration required to keep public claims truthful.


---


## Definition of Done


- Every recommendation in this plan maps to at least one completed story with green required lanes.
- Wave 1 yields a truthful, tested, installable first-value path and removes the legal launch blocker.
- Wave 2 yields a credible OSS trust baseline and operator documentation set with automated source-of-truth coverage.
- Public-facing docs, docs-site summaries, and command docs stay aligned under automated checks.
- Public/internal boundaries, side effects, and integration ownership are explicit wherever user-facing behavior is touched.
- The resulting repo can credibly be re-audited as `go` for public OSS launch without inventing coverage that Axym does not yet ship.



