# PLAN Next: Launch Gate Closure and Context Engineering Evidence

Date: 2026-03-18
Source of truth: user-provided recommended items from 2026-03-18, `product/axym.md`, `product/dev_guides.md`, `product/architecture_guides.md`, `AGENTS.md`
Scope: Axym OSS CLI only. Plan the remaining work needed to close CI/release contract gaps, migrate GitHub Actions runtime assumptions ahead of the June 2, 2026 Node.js 24 cutover, add governance-relevant proof capture for context engineering events, and complete OSS trust-baseline docs.

---

## Global Decisions (Locked)

- Preserve Axym determinism, offline-first defaults, fail-closed behavior, schema stability, and exit-code stability.
- Treat GitHub Actions workflow behavior as a user-visible delivery contract. Required checks, release integrity gates, and install verification remain part of the product surface.
- Prefer official/pinned GitHub Actions and scanners over floating or ad hoc CI wiring.
- Context engineering evidence capture must stay digest-first and local-first: no raw prompt, instruction, or knowledge-body exfiltration by default.
- Context engineering events must enter Axym through existing architecture boundaries: schema validation -> collector promotion -> normalization -> proof emission -> chain/bundle/reporting.
- Existing governance-event producers must remain backward-compatible unless explicitly versioned.
- Public docs are source-of-truth surfaces for install, verify, collect, and OSS support expectations; docs drift is a contract failure, not a polish issue.
- Contract/runtime waves complete before OSS trust-baseline and docs/onboarding waves.

---

## Current Baseline (Observed)

- The repository already contains a buildable Go CLI, broad runtime coverage, acceptance scenarios, release scripts, and signed local release smoke paths.
- `go test ./... -count=1` passed locally on 2026-03-18.
- `make prepush-full` passed locally on 2026-03-18, including local CodeQL, scenarios, docs checks, release-local, checksum/signature/SBOM/provenance, and release go/no-go validation.
- GitHub-hosted workflows exist in `.github/workflows/pr.yml`, `.github/workflows/main.yml`, `.github/workflows/nightly.yml`, and `.github/workflows/release.yml`.
- Current workflows still pin `actions/checkout@v4` and `actions/setup-go@v5`, which the recommendation set flags as part of the Node.js 20 deprecation window ahead of June 2, 2026.
- GitHub-hosted workflow coverage is still narrower than the v1.0 plan target: no GitHub Actions execution path currently enforces `golangci-lint`, `gosec`, docs-link checks, or CodeQL through workflow YAML.
- Local `make codeql` exists, but CodeQL is not yet wired into GitHub Actions as an authoritative hosted lane.
- `CONTRIBUTING.md` and `SECURITY.md` are absent.
- Repo hygiene tests currently enforce the presence of `product/PLAN_v1.0.md` and prohibit a small set of tracked secret artifacts, but they do not yet enforce the OSS trust-baseline docs.
- Governance-event collection already exists behind `axym collect --json --governance-event-file ...`, with schema validation and collector promotion for generic governance events.
- Current governance-event coverage is generic. It does not yet define a typed context-engineering taxonomy for agent instruction rewrites, context resets, or knowledge imports, and current tests only cover broader governance-event promotion/rejection paths.

---

## Exit Criteria

1. All tracked GitHub Actions workflows use Node.js 24-compatible pinned JavaScript actions or an explicit verified Node 24 execution contract, with no dependency on insecure fallback runner settings.
2. GitHub-hosted CI enforces `golangci-lint`, `gosec`, CodeQL, and docs-link checks in the correct PR/main/nightly/release lanes, with contract tests covering their presence.
3. Required PR checks remain emitted by PR-triggered workflows only, and branch-protection/status-name contracts remain deterministic.
4. Governance-event schema and collector flow accept digest-first context-engineering events for instruction rewrite, context reset, and knowledge import, and reject malformed or overexposing payloads deterministically.
5. `axym collect --json --governance-event-file ...` can ingest context-engineering event JSONL into proof records without changing offline/local-only defaults.
6. Context-engineering records survive chain append, bundle assembly, and verify flows with deterministic, byte-stable artifacts where applicable.
7. `CONTRIBUTING.md` and `SECURITY.md` exist, are linked from the README, and are enforced by repo hygiene/docs parity checks.
8. README, command guide, and docs-site sources stay synchronized for CI expectations, release verification, and context-engineering event capture.
9. `go test ./... -count=1` and `make prepush-full` remain green after the work lands.

---

## Recommendation Traceability

| Recommendation | Why | Strategic direction | Expected moat/benefit | Planned stories |
|---|---|---|---|---|
| Close the Story 0.3 CI gap by adding GitHub-hosted `golangci-lint`, `gosec`, CodeQL, and docs-link checks | Merge/release gates are not fully enforced by hosted CI today | Make delivery and release integrity part of the product contract | Stronger launch credibility and less hidden local-only validation | `W1-S1`, `W1-S2` |
| Remove dependency on Node.js 20-era workflow assumptions before the June 2, 2026 GitHub Actions cutover | Current workflow action pins may age into breakage or warning-driven drift | Future-proof delivery infrastructure with pinned, tested runtime assumptions | Lower release interruption risk and cleaner branch-protection behavior | `W1-S1` |
| Capture context engineering events as governance-relevant proof records | Self-modifying agent behavior is material governance evidence not fully modeled today | Expand Axym evidence breadth to modern agent operation boundaries | Differentiated governance coverage beyond tool-call-only evidence | `W2-S1`, `W2-S2` |
| Close missing OSS trust-baseline docs (`CONTRIBUTING.md`, `SECURITY.md`) and enforce them | OSS launch/support expectations are not fully expressed or tested | Raise adopter and contributor trust with explicit support/disclosure contracts | Better OSS readiness and reduced adoption friction | `W3-S1`, `W3-S2` |

---

## Test Matrix Wiring

Tier model used in this plan:

- Tier 1 Unit: isolated schema, parser, helper-script, and contract-unit coverage.
- Tier 2 Integration: deterministic cross-package behavior for collector promotion, normalization, and artifact generation.
- Tier 3 E2E CLI: command invocation, `--json`, help, and exit-code assertions.
- Tier 4 Acceptance: operator workflows and scenario/golden validation.
- Tier 5 Hardening: workflow contract, timeout/cancellation, and lifecycle edge conditions where applicable.
- Tier 9 Contract: workflow/status/schema/docs/output compatibility checks.
- Tier 10 UAT: release/install/release-binary smoke and publish-gate verification.
- Tier 11 Scenario: black-box scenario fixtures and golden outputs.

Lane definitions:

- Fast lane: focused unit, contract, docs, and helper-script checks suitable for rapid PR feedback.
- Core CI lane: mandatory PR/main coverage for touched codepaths, CLI behavior, and contract tests.
- Acceptance lane: scenario and end-to-end operator flows proving the new behavior from user entrypoint to artifacts.
- Cross-platform lane: Linux/macOS/Windows coverage for public CLI/docs/install surfaces when touched.
- Risk lane: security scanners, CodeQL, release-integrity gates, bundle determinism, and workflow runtime/state risks.

Story-to-lane map:

| Story | Fast | Core CI | Acceptance | Cross-platform | Risk |
|---|---|---|---|---|---|
| W1-S1 | Yes | Yes | No | Yes | Yes |
| W1-S2 | Yes | Yes | No | No | Yes |
| W2-S1 | Yes | Yes | No | Yes | No |
| W2-S2 | Yes | Yes | Yes | Yes | Yes |
| W3-S1 | Yes | Yes | No | No | No |
| W3-S2 | Yes | Yes | Yes | Yes | No |

Merge/release gating rule:

- A story is not complete until every lane marked `Yes` above is green for that story’s changed surfaces.
- PR merge blocks on required Fast and Core CI failures, plus any story-specific required Cross-platform lane.
- Release blocks on Risk-lane failures, release/install smoke failures, or docs/source-of-truth drift for public install and verification surfaces.

---

## Epic W1: CI Contract Closure and Node24 Readiness

Objective: make hosted CI/release behavior match Axym’s plan-level contract, remove Node20-era workflow assumptions, and preserve required-check determinism.

### Story W1-S1: Migrate GitHub Actions runtime assumptions to pinned Node24-compatible actions
Priority: P0
Tasks:
- Audit every JavaScript-based action referenced from `pr`, `main`, `nightly`, and `release` workflows.
- Upgrade or replace `actions/checkout@v4`, `actions/setup-go@v5`, and any other affected JavaScript actions with Node24-compatible pinned versions after official compatibility verification.
- Add explicit workflow/runtime contract coverage so Node24 is exercised before the June 2, 2026 default cutover and insecure fallback env vars are forbidden.
- Preserve current required-check semantics or update branch-protection/status-name contract tests atomically if names must change.
- Keep workflow concurrency and cancellation guarantees intact where already required.
Repo paths:
- `.github/workflows/pr.yml`
- `.github/workflows/main.yml`
- `.github/workflows/nightly.yml`
- `.github/workflows/release.yml`
- `testinfra/contracts/ci_required_checks_test.go`
- `testinfra/contracts/ci_contract_test.go`
- `testinfra/contracts/release_gate_contract_test.go`
- `README.md`
- `docs/commands/axym.md`
Run commands:
- `go test ./testinfra/contracts/... -count=1`
- `make test-contracts`
- `make prepush-full`
Test requirements:
- Tier 9: workflow pin/runtime contract tests for Node24-compatible action usage and forbidden insecure fallback env vars.
- Tier 5: workflow lifecycle contract checks preserving PR-emitted required checks and concurrency behavior.
- Tier 10: release smoke verifies source build and release-binary smoke still succeed after workflow pin updates.
Matrix wiring:
- Lanes: Fast, Core CI, Cross-platform, Risk.
- Pipeline placement: PR (workflow contract subset), Main (full workflow contract matrix), Release (release smoke and integrity contracts).
Acceptance criteria:
- No tracked workflow depends on Node20-only JavaScript action majors.
- Required PR checks remain emitted by PR-triggered workflows only.
- Workflow/runtime contract tests fail deterministically if deprecated action majors or insecure fallback env vars reappear.
Stable/internal boundary notes:
- Public: required-check names, install verification path, and release smoke behavior are contract surfaces.
- Internal: exact action versions and job topology may change if required-check/status contracts remain stable.
Migration expectations:
- Existing contributor entrypoints remain `make ...` and `go test ...`; no new dashboard or hosted prerequisite is introduced.
- If required-check names must change, branch-protection contract tests and contributor docs update in the same change.
Integration hooks:
- Contributors and release automation consume the same pinned action/runtime contract across PR, main, nightly, and tag pipelines.
Dependencies:
- None.
Risks:
- Branch-protection drift if workflow job names change without contract-test updates.

### Story W1-S2: Add missing GitHub-hosted scanner and docs-link gates
Priority: P0
Tasks:
- Add GitHub-hosted `golangci-lint`, `gosec`, and docs-link validation with pinned versions/actions.
- Add GitHub-hosted CodeQL analysis through workflow YAML so it is no longer local-only.
- Add or refine local helper targets/scripts so contributors can reproduce the non-hosted portions of these gates locally.
- Update PR/main/nightly/release workflow wiring so scanner and docs-link coverage lands in the correct pipelines without unnecessary duplication.
- Extend CI/release contract tests to enforce the presence and placement of these gates.
Repo paths:
- `Makefile`
- `.github/workflows/pr.yml`
- `.github/workflows/main.yml`
- `.github/workflows/nightly.yml`
- `.github/workflows/release.yml`
- `.github/workflows/codeql.yml`
- `scripts/check_docs_links.sh`
- `testinfra/contracts/ci_contract_test.go`
- `testinfra/contracts/ci_required_checks_test.go`
- `testinfra/contracts/release_gate_contract_test.go`
Run commands:
- `make lint-fast`
- `make test-contracts`
- `make test-docs-consistency`
- `make prepush-full`
Test requirements:
- Tier 1: helper-script smoke tests for docs-link and local command-lane wiring.
- Tier 9: workflow contract tests for scanner/docs-link presence, pinned action usage, and release-lane enforcement.
- Tier 10: release gate contract verifies security and integrity checks remain publish-blocking.
- Tier 5: workflow lifecycle checks keep heavy scanners in the intended pipelines and preserve timeout/cancellation expectations.
Matrix wiring:
- Lanes: Fast, Core CI, Risk.
- Pipeline placement: PR (fast scanner/docs contract subset), Main (full core security/docs gates), Nightly (expanded security depth), Release (publish-blocking integrity/security gates).
Acceptance criteria:
- GitHub Actions enforces `golangci-lint`, `gosec`, CodeQL, and docs-link checks in the intended hosted lanes.
- Local contributor commands clearly distinguish reproducible local gates from CI-authoritative hosted analysis.
- Release gating fails closed when required security or docs-link steps are absent from workflow YAML.
Stable/internal boundary notes:
- Public: contributor/release verification commands and required check set are part of the repo contract.
- Internal: workflow split across files/jobs may change if merge/release gates remain enforced and tested.
Migration expectations:
- Docs clearly state which checks are locally reproducible and which are CI-authoritative.
Integration hooks:
- PR, main, nightly, and release workflows all exercise the same declared security/docs gate model with lane-appropriate depth.
Dependencies:
- `W1-S1`
Risks:
- PR latency regression if heavy scanners are placed in the wrong lane or duplicated unnecessarily.

---

## Epic W2: Context Engineering Governance Evidence

Objective: expand Axym evidence coverage so instruction rewrites, context resets, and knowledge imports become first-class, deterministic proof records without weakening privacy or architecture boundaries.

### Story W2-S1: Extend governance-event schema and promotion for context engineering events
Priority: P0
Tasks:
- Extend the governance-event schema with a typed context-engineering taxonomy covering `instruction_rewrite`, `context_reset`, and `knowledge_import`.
- Require digest-first/provenance-first metadata for context-engineering events, including prior/current hashes or artifact digests rather than raw instructions or knowledge bodies.
- Update governance-event promotion and normalization so context-engineering events map to stable proof record fields and deterministic reason surfaces.
- Add redaction and validation rules so accidental raw content exposure is rejected or hashed before record creation.
- Keep backward compatibility for existing non-context governance events.
Repo paths:
- `schemas/v1/governance_event/governance-event.schema.json`
- `schemas/v1/governance_event/schema.go`
- `core/collect/governanceevent/collector.go`
- `core/collect/governanceevent/collector_test.go`
- `core/normalize/normalize.go`
- `core/redact/redact.go`
- `testinfra/contracts/governance_event_schema_contract_test.go`
- `fixtures/governance/context_engineering.jsonl`
Run commands:
- `axym collect --json --governance-event-file ./fixtures/governance/context_engineering.jsonl`
- `go test ./core/collect/governanceevent/... -count=1`
- `go test ./testinfra/contracts/... -count=1`
Test requirements:
- Tier 1: schema validator and promoter unit tests for each context-engineering event class.
- Tier 2: normalize/promote integration fixtures proving digest-only capture and deterministic field mapping.
- Tier 3: collect command JSON-envelope tests for governance-event-file inputs carrying context-engineering events.
- Tier 9: schema compatibility tests, valid/invalid fixture coverage, and reason-code stability checks.
Matrix wiring:
- Lanes: Fast, Core CI, Cross-platform.
- Pipeline placement: PR (schema/collector subset), Main (full schema + collect contract coverage), Cross-platform (public collect surface across OS runners).
Acceptance criteria:
- Instruction rewrites, context resets, and knowledge imports can be represented as valid governance-event JSONL and promoted to proof records.
- Raw instruction text, prompts, or knowledge bodies are not required and are not persisted by default.
- Existing governance-event payloads that do not use the new context-engineering taxonomy remain valid unless explicitly malformed.
Stable/internal boundary notes:
- Public: governance-event schema fields and `axym collect --json --governance-event-file ...` behavior are versioned contracts.
- Internal: exact proof record type selection may evolve if schema stability and downstream proof semantics remain intact.
Migration expectations:
- Existing governance-event producers continue to work unchanged.
- Producers that want richer context-engineering coverage can add the new typed fields without adopting any hosted service.
Integration hooks:
- Agent runtimes write local JSONL during instruction rewrite, context clear, or knowledge-import phases and hand that file to Axym via the existing collect flag.
Dependencies:
- None.
Risks:
- Overfitting the schema or allowing metadata fields that accidentally reintroduce evidence exfiltration.

### Story W2-S2: Add end-to-end contract and scenario coverage for context engineering record flows
Priority: P1
Tasks:
- Add fixtures and scenarios that collect context-engineering governance events, append them to the local chain, and carry them into bundle artifacts.
- Extend collect/bundle/verify contract tests to assert deterministic presence of context-engineering record digests and provenance fields.
- Add golden outputs for operator workflows showing local JSONL -> collect -> chain -> bundle behavior.
- Keep compliance mapping, gap grading, and regression semantics unchanged in this wave unless a later plan explicitly introduces policy interpretation for these new records.
Repo paths:
- `cmd/axym/collect_test.go`
- `internal/integration/record/normalize_validate_test.go`
- `internal/integration/bundle/byte_stability_test.go`
- `scenarios/axym/fixtures.yaml`
- `scenarios/axym/golden/results.json`
- `internal/scenarios/helpers_test.go`
- `testinfra/acceptance/scenario_contract_test.go`
- `fixtures/governance/context_engineering.jsonl`
Run commands:
- `axym collect --json --governance-event-file ./fixtures/governance/context_engineering.jsonl`
- `axym bundle --audit Q3-2026 --frameworks eu-ai-act,soc2 --json`
- `make test-scenarios`
- `go test ./... -count=1`
Test requirements:
- Tier 3: CLI collect/bundle JSON stability tests with context-engineering inputs.
- Tier 4: acceptance flows proving local JSONL -> chain -> bundle behavior.
- Tier 9: raw-record and bundle contract tests for deterministic artifact presence.
- Tier 11: scenario/golden coverage for the new operator workflow.
- Determinism/hash/signing: byte-stability repeat-run checks where bundle raw-record content changes.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (contract subset), Main (full CLI + scenario coverage), Nightly (bundle determinism/regression depth), Cross-platform (public CLI surface).
Acceptance criteria:
- The same context-engineering input corpus yields deterministic collect output and deterministic bundle artifact presence across repeated runs.
- `axym verify --chain --json` and `axym verify --bundle ... --json` continue to succeed on mixed standard and context-engineering evidence sets.
- Scenario goldens fail deterministically if context-engineering record shape, provenance fields, or artifact presence drift.
Stable/internal boundary notes:
- Public: operator integration pattern and JSON/bundle surfaces are stable.
- Internal: compliance scoring remains unchanged in this wave.
Migration expectations:
- Users who do not emit context-engineering events do not need to change configs, frameworks, or pipelines.
Integration hooks:
- Agent frameworks can adopt this evidence path incrementally by writing JSONL locally and reusing existing Axym commands.
Dependencies:
- `W2-S1`
Risks:
- Scenario flakiness if hashes, timestamps, or bundle ordering are not canonicalized.

---

## Epic W3: OSS Trust Baseline and Public Docs Closure

Objective: close the remaining OSS launch/readiness gaps by making contribution, security, CI, and context-engineering integration guidance explicit and test-enforced.

### Story W3-S1: Add OSS trust-baseline docs and enforce their presence
Priority: P1
Tasks:
- Author `CONTRIBUTING.md` with contribution flow, reproducible bug report expectations, local validation commands, and scope boundaries.
- Author `SECURITY.md` with disclosure path, supported release verification expectations, and security-support boundaries.
- Update the README to link both docs and to surface release-verification and support/disclosure expectations.
- Strengthen repo hygiene contract tests so the trust-baseline docs are required and key generated/secrets artifacts remain prohibited.
Repo paths:
- `CONTRIBUTING.md`
- `SECURITY.md`
- `README.md`
- `testinfra/contracts/repo_hygiene_test.go`
- `testinfra/contracts/command_docs_parity_contract_test.go`
Run commands:
- `make test-contracts`
- `make test-docs-consistency`
- `make test-docs-storyline`
Test requirements:
- Tier 9: repo-hygiene and docs-parity contract checks for trust-baseline doc presence and README references.
- Docs/examples: README/quickstart/support coverage checks for the new trust-baseline surfaces.
Matrix wiring:
- Lanes: Fast, Core CI.
- Pipeline placement: PR (contract/doc subset), Main (full docs parity), Fast lane (repo hygiene and README references).
Acceptance criteria:
- `CONTRIBUTING.md` and `SECURITY.md` exist and are linked from the README.
- Repo hygiene tests fail deterministically when either trust-baseline doc is removed.
- Public OSS contribution and disclosure expectations are versioned repo contracts rather than tribal knowledge.
Stable/internal boundary notes:
- Public: contribution, support, and disclosure expectations are repo-level contracts.
- Internal: exact wording can evolve if required topics and link presence remain enforced.
Migration expectations:
- No CLI behavior changes are introduced by this story.
Integration hooks:
- Contributors have a single documented path for local validation and security disclosure without needing private coordination.
Dependencies:
- None, but scheduled after runtime-contract waves by policy.
Risks:
- Incomplete guidance could still leave release verification or disclosure expectations ambiguous.

### Story W3-S2: Sync public docs and docs-site sources with CI and context-engineering flows
Priority: P1
Tasks:
- Update `README.md`, `docs/commands/axym.md`, `docs-site/public/llm/axym.md`, and `docs-site/public/llms.txt` with Node24-ready workflow expectations, local-vs-hosted security checks, and context-engineering governance-event examples.
- Add or extend docs checks so the new examples and support references are covered by consistency, storyline, and link validation.
- Ensure docs examples for context-engineering JSONL use the same file paths and command contracts as acceptance fixtures where possible.
- Clarify release verification commands and install/version discoverability in public docs after the CI/runtime changes.
Repo paths:
- `README.md`
- `docs/commands/axym.md`
- `docs-site/public/llm/axym.md`
- `docs-site/public/llms.txt`
- `scripts/check_docs_consistency.sh`
- `scripts/check_docs_storyline.sh`
- `scripts/check_docs_links.sh`
- `testinfra/contracts/command_docs_parity_contract_test.go`
- `fixtures/governance/context_engineering.jsonl`
Run commands:
- `make test-docs-consistency`
- `make test-docs-storyline`
- `make prepush-full`
- `axym collect --json --governance-event-file ./fixtures/governance/context_engineering.jsonl`
Test requirements:
- Docs/examples changes: docs consistency checks, storyline/smoke checks, README/quickstart/integration coverage checks, and docs source-of-truth sync tasks.
- Tier 9: docs parity contract tests across README, `docs/`, and docs-site sources.
- Tier 4: acceptance/docs smoke uses the same example paths as the tested operator flow where feasible.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform.
- Pipeline placement: PR (docs contract subset), Main (full docs parity/storyline), Acceptance (example-driven smoke flow), Cross-platform (install/docs public surface).
Acceptance criteria:
- One documented local operator path exists for emitting context-engineering JSONL and collecting it into Axym.
- One documented contributor path exists for reproducing CI/security gates locally and understanding which hosted gates remain CI-authoritative.
- Public docs for install, collect, verify, and release verification stay synchronized across README, command guide, and docs-site files.
Stable/internal boundary notes:
- Public: docs become authoritative for install, collect, CI, and security verification flows.
- Internal: exact example filenames may change if docs and fixture checks stay synchronized.
Migration expectations:
- Existing example commands remain valid or are updated atomically with test fixtures and docs parity checks.
Integration hooks:
- Agent/platform teams can copy one deterministic JSONL example flow directly from public docs into local or CI automation.
Dependencies:
- `W1-S2`
- `W2-S2`
- `W3-S1`
Risks:
- Docs drift if example commands or file paths diverge from acceptance fixtures.

---

## Minimum-Now Sequence

Wave 1: hosted contract and release-gate closure

- Deliver `W1-S1`, then `W1-S2`.
- Rationale: workflow/runtime compatibility and scanner gate closure are the highest-risk blockers because they can silently undermine merge/release integrity even when local validation is green.

Wave 2: context-engineering evidence model and runtime coverage

- Deliver `W2-S1`, then `W2-S2`.
- Rationale: once CI/release contracts are trustworthy, extend Axym’s runtime evidence model to cover modern agent self-modification behavior without destabilizing existing compliance semantics.

Wave 3: OSS trust baseline and public docs synchronization

- Deliver `W3-S1`, then `W3-S2`.
- Rationale: docs and OSS trust-baseline work should follow the runtime and workflow contract changes so public guidance reflects the final shipped behavior rather than interim states.

Recommended cut line for immediate execution:

- Minimum-now is all of Wave 1 plus `W2-S1`.
- Reason: this closes the highest-risk delivery gap and establishes the schema/runtime contract for the new governance evidence without waiting for later docs polish.

---

## Explicit Non-Goals

- No dashboard-first or hosted service scope.
- No LLM-based compliance inference in collect/map/gaps/verify defaults.
- No raw prompt, instruction, or knowledge-body capture by default for context-engineering events.
- No changes to compliance scoring, threshold policy, or regression semantics solely because context-engineering records now exist.
- No rearchitecture of Axym’s collector -> normalize -> proof emit -> bundle pipeline.
- No enterprise-only fork paths for GitHub Actions or evidence capture.

---

## Definition of Done

- Every recommendation above maps to at least one completed story and all mapped stories are green in their required lanes.
- Hosted GitHub Actions coverage matches the declared scanner/runtime/release gate contract and is enforced by tests.
- Node24-ready workflow pins and runtime assumptions are explicit, pinned, and contract-tested.
- Context-engineering governance events are schema-validated, digest-first, locally collectible, and covered by deterministic CLI/integration/scenario tests.
- Public docs and docs-site sources remain synchronized for install, collect, verify, CI, and release verification flows.
- OSS trust-baseline docs are present, linked, and enforced by repo hygiene/tests.
- `go test ./... -count=1` and `make prepush-full` remain green after the work sequence completes.
