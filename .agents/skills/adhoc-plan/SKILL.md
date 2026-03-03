---
name: adhoc-plan
description: Convert user-provided recommended work items into an execution-ready Axym backlog plan at a user-provided output path, with epics, stories, test requirements, and CI matrix wiring.
disable-model-invocation: true
---

# Recommendations to Backlog Plan (Axym)

Execute this workflow when the user asks to turn recommended items into a concrete backlog plan before implementation.

## Scope

- Repository root: `.`
- Recommendation source: user-provided recommended items for this run
- Standards sources of truth:
  - `./product/dev_guides.md`
  - `./product/architecture_guides.md`
- No dependency on `./product/ideas.md`
- Planning-only skill. Do not implement code in this workflow.

## Input Contract (Mandatory)

- `recommended_items`: structured list or raw text of recommended work to plan.
- `output_plan_path`: absolute or repo-relative file path where the generated plan will be written.

Validation rules:
- Both arguments are required.
- `output_plan_path` must resolve inside the repository and be writable.
- If either input is missing or invalid, stop and output a blocker report.

## Preconditions

- `./product/dev_guides.md` must exist and be readable.
- `./product/architecture_guides.md` must exist and be readable.
- Both guides must contain enforceable rules for:
  - testing and CI gating
  - determinism and contract stability
  - architecture/TDD/chaos/frugal governance requirements
- If guides are missing or incomplete, stop and output a blocker report.

## Workflow

1. Read `product/dev_guides.md` and `product/architecture_guides.md`; extract locked implementation and architecture constraints.
2. Parse `recommended_items` and normalize each item to:
- recommendation
- why
- strategic direction
- expected moat/benefit
3. Remove duplicates and out-of-scope items.
4. Cluster recommendations into coherent epics.
5. Prioritize with `P0/P1/P2` using contract risk, moat gain, adoption leverage, and dependency order.
6. Create execution-ready stories with:
- tasks
- repo paths
- run commands
- test requirements
- matrix wiring
- acceptance criteria
7. For every story, enforce architecture fields:
- architecture constraints
- ADR required (`yes/no`)
- TDD first failing test(s)
- cost/perf impact (`low|medium|high`)
- chaos/failure hypothesis (required for risk-bearing stories)
8. For every story, enforce contract-discipline fields:
- contract surface (`public|internal|shim|deprecated`)
- versioning/migration impact
- structured-error impact
- API symmetry/side-effect invariants
- timeout/cancellation invariants for long-running workflows
9. Add a `Contract Surface Map` section that lists stable public APIs, internal-only surfaces, compatibility shims, and active deprecations.
10. Add plan-level `Test Matrix Wiring`.
11. Add `Recommendation Traceability` mapping recommendations to epic/story IDs.
12. Add `Minimum-Now Sequence`, `Exit Criteria`, and `Definition of Done`, and enforce two-wave ordering:
- Wave 1: contract/runtime correctness and architecture boundaries.
- Wave 2: docs, OSS hygiene, and distribution UX.
13. Add docs/DX readiness work where user-visible behavior changes: README first-screen clarity, integration-before-internals flow, lifecycle/path model, and source-of-truth linkage.
14. Verify quality gates.
15. Overwrite `output_plan_path` with the final plan.

## Handoff Contract (Planning -> Implementation)

- This skill intentionally leaves `output_plan_path` modified in the working tree.
- Expected follow-up is `adhoc-implement` with the same `plan_path` on a new branch.
- If additional dirty files exist beyond the generated plan file, stop and scope/clean before implementation.

## Command Contract (JSON Required)

Use `axym` commands with `--json` whenever the plan needs machine-readable evidence, for example:

- `axym collect --dry-run --json`
- `axym regress init --baseline <baseline-bundle-path> --json`

## Non-Negotiables

- Preserve Axym contracts:
- determinism
- offline-first defaults
- fail-closed policy enforcement
- schema stability
- exit code stability
- Respect architecture boundaries:
- Go core authoritative for enforcement/verification
- Python remains thin adoption layer
- Enforce both standards guides in every generated plan:
  - `product/dev_guides.md`
  - `product/architecture_guides.md`
- No dashboard-first scope in core backlog.
- No minor polish as primary backlog.
- Every story must include tests and matrix wiring.
- Sequence work in two waves (Wave 1 contracts/runtime first, Wave 2 docs/OSS/distribution UX second).

## Architecture Guides Enforcement Contract

For stories touching architecture/risk/adapter/failure semantics, plan wiring must include:

- `make prepush-full`

For reliability/fault-tolerance stories, plan wiring must include:

- `make test-hardening`
- `make test-chaos`

For performance-sensitive stories, plan wiring must include:

- `make test-perf`

## Test Requirements by Work Type (Mandatory)

1. Schema/artifact changes:
- schema validation tests
- fixture/golden updates
- compatibility/migration tests

2. CLI behavior changes:
- help/usage tests
- `--json` stability tests
- exit-code contract tests
- `axym version` discoverability and minimal dependency install-path checks when install/version surfaces are touched

3. Gate/policy/fail-closed changes:
- deterministic allow/block/require_approval fixtures
- fail-closed undecidable-path tests
- reason-code stability checks
- For stories that clean/reset output paths, require `non-empty + non-managed => fail` tests
- Require marker trust tests (`marker must be regular file`; reject symlink/directory)

4. Determinism/hash/sign/packaging changes:
- byte-stability repeat-run tests
- canonicalization/digest checks
- verify/diff determinism tests
- `make test-contracts` when applicable

5. Job runtime/state/concurrency changes:
- lifecycle tests
- crash-safe/atomic-write tests
- contention/concurrency tests
- chaos suites when applicable
- end-to-end timeout/cancellation propagation checks for long-running workflows

6. SDK/adapter boundary changes:
- wrapper error-mapping tests
- adapter parity/conformance tests
- structured machine-readable error envelope tests for library/SDK consumers
- extension-point compatibility tests when enterprise integration seams are introduced

7. Voice/context-proof changes:
- relevant scenario acceptance suites as applicable

8. Docs/examples changes:
- docs consistency checks
- storyline/smoke checks when user flow changes
- README first-screen checks (what it is, for whom, integration path, first value)
- source-of-truth checks between repo docs and generated/public docs

## Test Matrix Wiring Contract (Plan-Level)

The plan must include a `Test Matrix Wiring` section with:

- Fast lane
- Core CI lane
- Acceptance lane
- Cross-platform lane
- Risk lane
- Merge/release gating rule

Every story must declare its lane wiring.

## Plan Format Contract

Required sections:

1. `# PLAN <name>: <theme>`
2. `Date`, `Source of truth`, `Scope`
3. `Global Decisions (Locked)`
4. `Current Baseline (Observed)`
5. `Exit Criteria`
6. `Recommendation Traceability`
7. `Contract Surface Map`
8. `Test Matrix Wiring`
9. Epic sections with objectives and stories
10. `Minimum-Now Sequence` (with Wave 1/Wave 2 ordering)
11. `Explicit Non-Goals`
12. `Definition of Done`

Story template:

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
- Optional: `Dependencies:`, `Risks:`

## Quality Gate

Before finalizing:

- Every recommendation maps to at least one epic/story.
- Every story is actionable without guesswork.
- Acceptance criteria are testable and deterministic.
- Paths are real and repo-relevant.
- Test requirements match story type.
- Matrix wiring exists for every story.
- Every story maps to enforceable rules from both guides (`dev_guides.md`, `architecture_guides.md`).
- Contract surface map and story-level contract fields are complete and consistent.
- High-risk stories include hardening/chaos lane wiring.
- CLI contract stories include explicit `--json` and exit-code invariants.
- `Minimum-Now Sequence` applies Wave 1 before Wave 2 for touched surfaces.
- Docs stories preserve README first-screen clarity, integration-first flow, and docs source-of-truth linkage.
- OSS/distribution stories include trust-baseline artifacts and support/maintainer context when applicable.
- Sequence is dependency-aware and implementation-ready.

## Failure Mode

If inputs are missing or recommendations are not plan-ready, write only:

- `No backlog plan generated.`
- `Reason:` concise blocker summary.
- `Missing inputs:` exact required fields.

Do not fabricate backlog content.
