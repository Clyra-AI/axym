---
name: backlog-plan
description: Transform strategic recommendations in product/ideas.md into an execution-ready Axym backlog plan with epics, stories, tasks, repo paths, commands, acceptance criteria, and explicit test-matrix wiring.
disable-model-invocation: true
---

# Ideas to Backlog Plan (Axym)

Execute this workflow when asked to convert strategic feature recommendations into a concrete product backlog plan.

## Scope

- Repository root: `.`
- Input file: `./product/ideas.md`
- Standards sources of truth:
  - `./product/dev_guides.md`
  - `./product/architecture_guides.md`
- Structure references (match level of detail and style):
- `./product/PLAN_v1.md` (preferred local reference, if available)
- `../gait/product/PLAN_v1.md` (fallback reference, if available in your environment)
- Output file: `./product/PLAN_NEXT.md` (unless user specifies a different target)
- Planning only. Do not implement code or docs outside the target plan file.

## Preconditions

- `ideas.md` must contain strategic recommendations with evidence.
- Each recommendation should include:
- recommendation name
- why-now trigger
- strategic capability direction
- moat/benefit rationale
- source links

If these are missing, stop and output a gap note instead of inventing details.

- `product/dev_guides.md` and `product/architecture_guides.md` must exist and be readable.
- Both guides must provide enforceable planning constraints for:
  - testing/CI gates
  - determinism/contracts
  - architecture/TDD/chaos/frugal standards
- If either guide is missing or incomplete, stop with a blocker report.

## Workflow

1. Read `ideas.md` and extract candidate initiatives.
2. Read `product/dev_guides.md` and `product/architecture_guides.md` and lock constraints into planning decisions.
3. Read reference plans to mirror structure and detail level.
4. Cluster ideas into coherent epics (avoid one-idea-per-epic fragmentation).
5. Prioritize using `P0/P1/P2` based on contract risk reduction, moat expansion, adoption leverage, and sequencing dependency.
6. Produce execution-ready epics and stories.
7. For every story, include concrete tasks, repo paths, run commands, acceptance criteria, test requirements, CI matrix wiring, architecture governance fields, and contract-discipline fields (public/internal/shim/deprecated surface, versioning impact, structured-error impact, API symmetry/side-effect invariants, timeout/cancellation invariants when long-running paths are touched).
8. Build a `Contract Surface Map` section that lists stable public APIs, internal-only surfaces, compatibility shims, and active deprecations.
9. Build a plan-level test matrix section mapping stories to CI lanes (fast, integration, acceptance, cross-platform).
10. Ensure each story defines tests based on work type (schema, CLI, gate/policy, determinism, runtime, SDK, docs/examples, OSS/distribution where touched).
11. Add explicit boundaries and non-goals to prevent scope drift.
12. Add delivery sequencing section (phase/week-based minimum-now path) using two waves:
- Wave 1: contract/runtime correctness and architecture boundaries.
- Wave 2: docs, OSS hygiene, and distribution UX.
13. Add docs/DX readiness work where user-visible behavior changes: README first-screen clarity, integration-before-internals flow, lifecycle/path model, and source-of-truth linkage.
14. Add definition of done and release/exit gate criteria.
15. Write full plan to target file, overwriting prior contents.

## Handoff Contract (Planning -> Implementation)

- This skill intentionally leaves the generated plan file modified in the working tree.
- Expected follow-up is `backlog-implement` using that plan file on a new branch.
- If additional dirty files exist beyond the plan output, stop and scope/clean before implementation.

## Non-Negotiables

- Preserve Axym core contracts: determinism, offline-first defaults, fail-closed policy posture, schema stability, and exit code stability.
- Respect architecture boundaries: Go core is authoritative for enforcement/verification logic; Python remains thin adoption layer.
- Enforce both planning standards guides in all generated stories:
  - `product/dev_guides.md`
  - `product/architecture_guides.md`
- Avoid dashboard-first or hosted-only dependencies in backlog core.
- Do not include implementation code, pseudo-code, or ticket boilerplate.
- Do not recommend minor polish work as primary backlog items.
- Every story must include test requirements and explicit matrix wiring.
- Sequence work in two waves (Wave 1 contracts/runtime first, Wave 2 docs/OSS/distribution UX second).
- No story is complete without same-change test updates, except explicitly justified docs-only stories.

## Architecture Guides Enforcement Contract

For architecture/risk/adapter/failure stories, require wiring for:

- `make prepush-full`

For reliability/fault-tolerance stories, require wiring for:

- `make test-hardening`
- `make test-chaos`

For performance-sensitive stories, require wiring for:

- `make test-perf`

## Test Requirements by Work Type (Mandatory)

1. Schema or artifact contract work:
- Add schema validation tests.
- Add or update golden fixtures.
- Add compatibility or migration tests.

2. CLI surface work (flags, args, `--json`, exits):
- Add command tests for help/usage behavior.
- Add `--json` stability tests.
- Add exit code contract tests.
- Add `axym version` discoverability and minimal dependency install-path checks when install/version surfaces are touched.

3. Gate or policy semantics:
- Add deterministic allow/block/require_approval and `decision.pass` fixture tests.
- Add fail-closed tests for evaluator-missing or undecidable paths.
- Add reason code stability checks.
- Add regression input-boundary tests (`invalid_record`/`schema_error`/`mapping_error` must not be treated as valid evidence).
- Add chain/dedup preservation tests (re-ingest must not duplicate or reorder records; previous-hash linkage must remain stable).
- For stories that clean/reset output paths, require `non-empty + non-managed => fail` tests.
- Require marker trust tests (`marker must be regular file`; reject symlink/directory).

4. Determinism, hashing, signing, packaging:
- Add byte-stability tests for repeated runs with identical input.
- Add canonicalization and digest stability tests.
- Add verify/diff determinism tests.

5. Job runtime, state, concurrency, persistence:
- Add pause/resume/cancel/checkpoint lifecycle tests.
- Add crash-safe/atomic write tests.
- Add concurrent execution and contention tests.
- Add end-to-end timeout/cancellation propagation checks for long-running workflows.

6. SDK or adapter boundary work:
- Add wrapper behavior/error-mapping tests.
- Add adapter conformance tests against canonical sidecar/gate path.
- Preserve Go-authoritative decision boundary tests.
- Add structured machine-readable error envelope tests for library/SDK consumers.
- Add extension-point compatibility tests when enterprise integration seams are introduced.

7. Docs/examples contract changes:
- Add command-smoke checks for documented flows.
- Add docs-versus-CLI parity checks where possible.
- Update acceptance scripts if docs alter required operator path.
- Add README first-screen checks (what it is, for whom, integration path, first value).
- Add source-of-truth checks between repo docs and generated/public docs.

## Test Matrix Wiring Contract (Plan-Level)

Every generated plan must include a section named `Test Matrix Wiring` with:

- `Fast lane`: pre-push or quick CI checks required for each epic.
- `Core CI lane`: required unit/integration UAT checks in default CI.
- `Acceptance lane`: deterministic acceptance scripts required before merge or release.
- `Cross-platform lane`: Linux/macOS/Windows expectations for affected stories.
- `Risk lane`: extra suites for high-risk stories (policy, determinism, security, portability).
- `Gating rule`: merge/release block conditions tied to failed required lanes.

## Plan Format Contract

Required top sections:

1. `# PLAN <name>: <theme>`
2. `Date`, `Source of truth`, `Scope`
3. `Global Decisions (Locked)`
4. `Current Baseline (Observed)`
5. `Exit Criteria`
6. `Contract Surface Map`
7. `Test Matrix Wiring`
8. `Epic` sections with `Objective` and `Story` breakdowns
9. `Minimum-Now Sequence` (phased execution with Wave 1/Wave 2)
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
- Optional when needed:
- `Dependencies:`
- `Risks:`
- `Semantic invariants:` (required for stories touching ingestion/dedup/chain/regress behavior)

## Quality Gate for Output

Before finalizing, verify:

- Every epic maps back to at least one idea from `ideas.md`.
- Every story is actionable without guesswork.
- Acceptance criteria are testable and deterministic.
- Paths are real and repo-relevant.
- Test requirements match story work type.
- Matrix wiring is present for every story.
- Every story maps to enforceable rules from both guides (`dev_guides.md`, `architecture_guides.md`).
- Contract surface map and story-level contract fields are complete and consistent.
- High-risk stories include hardening/chaos lane wiring.
- CLI contract stories include explicit `--json` and exit-code invariants.
- `Minimum-Now Sequence` applies Wave 1 before Wave 2 for touched surfaces.
- Docs stories preserve README first-screen clarity, integration-first flow, and docs source-of-truth linkage.
- OSS/distribution stories include trust-baseline artifacts and support/maintainer context when applicable.
- Sequence is dependency-aware.
- Plan stays strategic and execution-relevant (not cosmetic).

## Command Anchors

- Include concrete plan tasks that reference verifiable CLI surfaces, for example:
  - `axym collect --dry-run --json`
  - `axym regress run --baseline <baseline-path> --frameworks eu-ai-act,soc2 --json`
  - `axym verify --chain --json`

## Failure Mode

If `ideas.md` lacks strategic quality or evidence, write only:

- `No backlog plan generated.`
- `Reason:` concise blocker summary.
- `Missing inputs:` exact missing fields required to proceed.

Do not fabricate backlog content when source strategy quality is insufficient.
