---
name: initial-plan
description: Transform the Axym PRD in product/axym.md into a world-class, zero-ambiguity execution plan that mirrors the detail level of product/PLAN_v1.md (or gait/product/PLAN_v1.md when needed), while enforcing product/dev_guides.md and product/architecture_guides.md standards for coding, testing, CI, determinism, architecture governance, and contracts. Use when the user asks for an initial build plan from the PRD (not from ideas/recommendations).
disable-model-invocation: true
---

# PRD to Initial Execution Plan (Axym)

Execute this workflow when asked to create the initial execution plan from the Axym PRD.

## Scope

- Repository root: `.`
- Primary source of truth: `./product/axym.md`
- Standards sources of truth:
  - `./product/dev_guides.md`
  - `./product/architecture_guides.md`
- Style reference (structure and depth): `./product/PLAN_v1.md`
- Default output: `./product/PLAN_v1.0.md` (unless user specifies a different target path)
- Planning only. Do not implement code or docs outside the target plan file.

## Preconditions

`axym.md` must contain enough implementation detail to drive execution planning. At minimum:
- product scope and goals
- functional requirements
- non-functional requirements
- acceptance criteria
- architecture and tech choices
- CLI surfaces and expected behavior

`product/dev_guides.md` and `product/architecture_guides.md` must define enforceable engineering standards. At minimum:
- toolchain versions
- lint/format requirements
- testing tiers and commands
- CI pipeline expectations
- determinism and contract requirements

If these are missing, stop and output a gap note instead of inventing policy.

## Workflow

1. Read `product/axym.md` and extract:
- core product objective and boundaries
- FR/NFR requirements
- acceptance criteria (ACs)
- architecture boundaries and non-goals
- CLI contracts (`--json`, `--explain`, `--quiet`, exit behavior)
- API surface expectations (public/internal/shim/deprecated), versioning/migration rules, and machine-readable error expectations where applicable

2. Read `product/dev_guides.md` and extract locked implementation standards:
- toolchain pins (Go/Python/Node)
- lint and formatting stack
- test tier model (Tier 1-12) and where each tier runs
- CI lane expectations (PR/main/nightly/release)
- coverage gates, determinism, schema and exit code stability
- security scanning and release integrity expectations

3. Read `product/architecture_guides.md` and extract locked architecture execution standards:
- TDD requirements and first-failing-test expectations
- architecture constraints and ADR requirements
- cloud-native execution factors beyond 12-factor expectations
- frugal architecture/cost impact requirements
- chaos/hardening/perf lane triggers and failure-hypothesis expectations

4. Read `./product/PLAN_v1.md` (or `../gait/product/PLAN_v1.md` if local reference is unavailable) to mirror structure depth and story-level specificity.

5. Inspect current repository baseline and convert observed reality into a `Current Baseline (Observed)` section:
- existing directories and key files
- current CI/workflow state
- current command surfaces and gaps versus PRD

6. Build epics by implementation dependency, not by document order:
- Epic 0 foundations/scaffold/contracts (Wave 1: contract/runtime correctness and architecture boundaries)
- core runtime epics (collectors/adapters, ingestion/translation, proof emission, context enrichment, compliance mapping/gaps)
- CLI, regressions, daily review, and remediation flows
- docs/acceptance/release hardening epics (Wave 2: docs, OSS hygiene, distribution UX)

7. Decompose every epic into execution-ready stories with explicit tasks, test wiring, and contract-discipline fields (contract surface, versioning impact, structured-error impact, API symmetry/side-effect invariants, timeout/cancellation invariants for long-running workflows).

8. Add a `Contract Surface Map` section listing stable public APIs, internal-only surfaces, compatibility shims, and active deprecations.

9. Add a plan-level `Test Matrix Wiring` section that maps stories to:
- Fast lane
- Core CI lane
- Acceptance lane
- Cross-platform lane
- Risk lane
- Gating rule

10. Add a dependency-aware `Minimum-Now Sequence` with phased/week execution order and explicit Wave 1 -> Wave 2 progression.

11. Add `Explicit Non-Goals` and `Definition of Done`.

12. Write the plan to the target file, replacing prior contents.

## Handoff Contract (Planning -> Implementation)

- This skill intentionally leaves the generated plan file modified in the working tree.
- Expected follow-up is an implementation skill on a new branch with that plan as input.
- If additional dirty files exist beyond the generated plan file, stop and scope/clean before implementation.

## Non-Negotiables

- Preserve Axym core contracts: deterministic execution, zero data exfiltration by default, fail-closed policy posture, stable schema contracts, stable exit code contracts.
- Keep architecture boundaries testable: collection, ingestion/translation, proof emission, context enrichment, mapping/gaps, bundle/review/regress.
- Go core remains authoritative for enforcement and verification logic; Python remains a thin adoption layer.
- Do not introduce hosted-only dependencies into v1 core.
- Do not produce cosmetic/backlog fluff. Every story must be executable by an engineer without clarification meetings.
- Sequence work in two waves (Wave 1 contracts/runtime first, Wave 2 docs/OSS/distribution UX second).
- No story is complete without same-change tests unless explicitly justified docs-only scope.

## Plan Format Contract

Required top sections:

1. `# PLAN <name>: <theme>`
2. `Date`, `Source of truth`, `Scope`
3. `Global Decisions (Locked)`
4. `Current Baseline (Observed)`
5. `Exit Criteria`
6. `Contract Surface Map`
7. `Test Matrix Wiring`
8. `Epic` sections with objective and story breakdowns
9. `Minimum-Now Sequence` (phased, dependency-aware, Wave 1/Wave 2)
10. `Explicit Non-Goals`
11. `Definition of Done`

Story template (required fields):

- `### Story <ID>: <title>`
- `Priority:`
- `Tasks:`
- `Repo paths:`
- `Run commands:`
- `Test requirements:`
- `Matrix wiring:`
- `Acceptance criteria:`
- `Architecture constraints:`
- `Contract surface: public|internal|shim|deprecated`
- `Versioning/migration impact: none|minor|major + migration expectation`
- `Structured errors impact: none|update_required`
- `API symmetry/side-effect invariants:`
- `ADR required: yes|no`
- `TDD first failing test(s):`
- `Cost/perf impact: low|medium|high`
- `Chaos/failure hypothesis:` (required for risk-bearing stories)
- `Timeout/cancellation invariants:` (required for long-running workflows)
- `Semantic invariants:` (required for stories touching ingestion/dedup/chain/regress behavior)
- Optional when needed:
- `Dependencies:`
- `Risks:`

## Dev Guides Enforcement Contract

For every story, derive required checks from `product/dev_guides.md` by work type. At minimum:

1. Toolchain/runtime changes:
- Lock versions to stated standards (Go 1.26.1, Python 3.13+, Node 22 docs/UI only).
- Include compatibility checks for shared `Clyra-AI/proof` constraints.

2. CLI surface changes (commands/flags/json/exits):
- Add CLI behavior tests and `--json` shape checks.
- Add exit code contract checks.
- Add docs parity checks for command/flag naming.
- Add `axym version` discoverability and minimal dependency install-path checks when install/version surfaces are touched.

3. Schema/artifact/proof changes:
- Add schema validation and compatibility tests.
- Add golden fixtures and deterministic artifact checks.
- Add proof chain verify checks where applicable.

4. Collection/mapping/policy semantics:
- Add deterministic fixture tests for classification, coverage weighting, and fail-closed behavior.
- Add fail-closed tests for undecidable/ambiguous high-risk paths.
- Add reason-code stability and ranking determinism checks.
- Add regression input-boundary tests (`invalid_record`/`schema_error`/`mapping_error` must not be treated as valid evidence).
- Add chain/dedup preservation tests (re-ingest must not duplicate or reorder records; previous-hash linkage must remain stable).
- For stories that clean/reset output paths, add `non-empty + non-managed => fail` tests.
- Add marker trust tests (`marker must be regular file`; reject symlink/directory).

5. Runtime/state/concurrency:
- Add atomic write/checkpoint/lock contention tests.
- Add crash/retry/recovery behavior tests.
- Add end-to-end timeout/cancellation propagation checks for long-running workflows.

6. CI/release/security work:
- Wire required lint/security jobs (golangci-lint, gosec, govulncheck, ruff/mypy/bandit where applicable).
- Add release-integrity checks (SBOM/signing/provenance) when story scope touches release.

7. Docs/examples contract changes:
- Add command-smoke checks for documented flows.
- Update acceptance scripts if operator workflow changed.
- Add README first-screen checks (what it is, for whom, integration path, first value).
- Add docs source-of-truth linkage checks between repo docs and generated/public docs.
- Add integration-before-internals and problem -> solution framing checks for touched docs.

8. API/library contract management:
- Require explicit public/internal/shim/deprecated classification for touched surfaces.
- Require plain-language schema/versioning migration expectations for breaking changes.
- Require structured machine-readable error envelope compatibility for SDK/library consumers.
- Require extension-point strategy for enterprise integration seams to avoid fork-only paths.

## Architecture Guides Enforcement Contract

For every story, enforce `product/architecture_guides.md` requirements:

1. TDD requirements:
- Capture first failing test(s) before implementation tasks.
- Require red-green-refactor intent in story acceptance criteria where behavior changes.

2. Architecture governance:
- Specify architecture constraints for layer boundaries touched.
- Mark `ADR required: yes` when changing boundary/data flow/contract/failure class.

3. Frugal architecture:
- Include cost/perf impact classification (`low|medium|high`).
- For perf-sensitive stories, include `make test-perf`.

4. Chaos/reliability operations:
- Risk-bearing stories must include failure hypothesis and lane wiring for:
  - `make test-hardening`
  - `make test-chaos`

5. Contract-first behavior:
- CLI/JSON/exit-code stories must state explicit invariants in acceptance criteria.

## Testing Tier Mapping (Mandatory)

When assigning `Test requirements:` and `Matrix wiring:` for each story, map to applicable dev-guides tiers explicitly:

1. Tier 1 Unit: pure package logic and parser/scorer unit coverage.
2. Tier 2 Integration: deterministic cross-component behavior with `-count=1`.
3. Tier 3 E2E: CLI invocation, JSON output, and exit code behavior.
4. Tier 4 Acceptance: scenario scripts for operator workflows and golden behavior.
5. Tier 5 Hardening: lock/atomic write/contention/retry/error-envelope resilience.
6. Tier 6 Chaos: controlled fault injection and resilience under failure.
7. Tier 7 Performance: benchmark and latency budget validation.
8. Tier 8 Soak: long-running stability and sustained contention.
9. Tier 9 Contract: schema, JSON shape, byte stability, exit contract compatibility.
10. Tier 10 UAT: install-path validation across source/release/homebrew flows.
11. Tier 11 Scenario: specification-driven outside-in scenario fixtures.
12. Tier 12 Cross-product Integration: compatibility checks with `Clyra-AI/proof`, Axym, and Gait contracts where touched.

High-risk stories must never stop at Tier 1-3 coverage only; include Tier 4/5/9 and add Tier 6/7/8 when risk profile warrants it.

## Test Matrix Wiring Contract

Every generated plan must include this section name exactly: `Test Matrix Wiring`.

The section must define:
- `Fast lane`: pre-push and rapid PR checks.
- `Core CI lane`: mandatory unit and integration checks.
- `Acceptance lane`: version-gated scenario scripts and operator-path checks.
- `Cross-platform lane`: Linux/macOS/Windows expectations for affected stories.
- `Risk lane`: hardening/chaos/performance/contract suites for high-risk stories.
- `Gating rule`: merge/release must block on required lane failure.

The matrix must explicitly map each story ID to one or more lanes.

## CI Pipeline Wiring Contract

In each plan, explicitly state where each required test set executes:

- PR pipeline: fast deterministic checks needed to review safely.
- Main pipeline: core integration/e2e/contract checks on push to main.
- Nightly pipeline: hardening/chaos/performance/soak suites.
- Release pipeline: signing/provenance/security and release acceptance gates.

Each story must identify pipeline placement for its required suites.

## Command Anchors

Include concrete story tasks that reference verifiable command surfaces, including:
- `axym collect --dry-run --json`
- `axym regress run --baseline <baseline-path> --frameworks eu-ai-act,soc2 --json`
- `axym verify --chain --json`
- `go test ./...`
- `go test ./... -count=1`

## Quality Gate for Output

Before finalizing, verify:
- every epic traces back to specific FR/NFR/AC statements in `axym.md`
- every story has concrete repo paths and executable commands
- acceptance criteria are deterministic and objectively testable
- test requirements match `dev_guides.md` and `architecture_guides.md` requirements
- ingestion/chain/dedup/regress stories include explicit semantic invariants
- every story includes architecture constraints, TDD first-failing-test requirement, and cost/perf impact
- contract surface map and story-level contract fields are complete and consistent
- high-risk stories include hardening/chaos lane wiring
- CLI contract stories include explicit `--json` and exit-code invariants
- sequence applies Wave 1 before Wave 2 for touched surfaces
- docs stories preserve README first-screen clarity, integration-first flow, lifecycle/path model, and docs source-of-truth linkage
- OSS/distribution stories include trust-baseline artifacts and maintainer/support context when applicable
- naming taxonomy/capability labels and first-10-min onboarding flow remain consistent with portfolio standards when cross-repo docs are touched
- matrix wiring exists for every story
- sequence is dependency-aware and executable end-to-end
- plan respects Axym boundaries (Prove product only; no Wrkr/Gait feature scope creep)

## Failure Mode

If `axym.md`, `dev_guides.md`, or `architecture_guides.md` lacks required planning inputs, write only:

- `No initial plan generated.`
- `Reason:` concise blocker summary.
- `Missing inputs:` exact missing fields/sections needed to proceed.

Do not fabricate plan details when source standards are incomplete.
