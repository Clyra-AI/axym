---
name: adhoc-plan
description: Convert user-provided recommended work items into an execution-ready Axym backlog plan at a user-provided output path, with epics, stories, test requirements, and CI matrix wiring.
disable-model-invocation: true
---

# Recommendations to Backlog Plan (Axym)

Execute this workflow when the user asks to turn recommended items into a concrete backlog plan before implementation.

## Scope

- Repository root: `/Users/tr/axym`
- Recommendation source: user-provided recommended items for this run
- No dependency on `/Users/tr/axym/product/ideas.md`
- Planning-only skill. Do not implement code in this workflow.

## Input Contract (Mandatory)

- `recommended_items`: structured list or raw text of recommended work to plan.
- `output_plan_path`: absolute or repo-relative file path where the generated plan will be written.

Validation rules:
- Both arguments are required.
- `output_plan_path` must resolve inside the repository and be writable.
- If either input is missing or invalid, stop and output a blocker report.

## Workflow

1. Parse `recommended_items` and normalize each item to:
- recommendation
- why
- strategic direction
- expected moat/benefit
2. Remove duplicates and out-of-scope items.
3. Cluster recommendations into coherent epics.
4. Prioritize with `P0/P1/P2` using contract risk, moat gain, adoption leverage, and dependency order.
5. Sequence work in dependency-driven waves:
- Use `Wave 1 .. Wave N` and corresponding epic IDs `W1 .. WN`, where `N >= 1`.
- Create only one wave when scope is small and a split adds no implementation value.
- Create multiple waves when dependency order, risk reduction, or reviewability benefits from staging.
- When both classes exist, contract/runtime correctness and architecture-boundary work must complete in earlier waves before docs, OSS hygiene, onboarding, or distribution UX waves.
6. Create execution-ready stories with:
- tasks
- repo paths
- run commands
- test requirements
- matrix wiring
- acceptance criteria
- If a story affects public surfaces, include stable/internal boundary notes, migration expectations, and where users integrate Axym into their workflow or pipeline.
7. Add plan-level `Test Matrix Wiring`.
8. Add `Recommendation Traceability` mapping recommendations to epic/story IDs.
9. Add `Minimum-Now Sequence`, `Exit Criteria`, and `Definition of Done`, with the explicit wave order and rationale.
10. Verify quality gates.
11. Overwrite `output_plan_path` with the final plan.

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
- source adapters and collector acquisition
- event normalization and proof record construction
- proof emission
- sibling ingestion and translation
- context enrichment and compliance matching
- coverage/gap/review/regression evaluation
- bundle assembly/export/verification
- Treat architecture as enforceable code boundaries, not doc-only intent.
- Prefer thin orchestration and focused packages for parsing, persistence, reporting, and policy logic.
- Make side effects explicit in names/signatures and avoid ambiguous `plan` vs `apply` or `read` vs `read+validate` semantics.
- Public-surface stories must cover versioning/deprecation expectations, machine-readable error behavior, and install/version discoverability where relevant.
- Long-running workflow stories must include cancellation/timeout propagation expectations.
- Prefer extension points over enterprise forks when the recommendation implies customization pressure.
- No dashboard-first scope in core backlog.
- No minor polish as primary backlog.
- Every story must include tests and matrix wiring.
- Use dependency-driven wave sequencing instead of a fixed two-wave template.
- It may be 1 wave or many waves depending on complexity, dependencies, and implementation risk.
- When both contract/runtime and docs/onboarding/distribution classes exist, all contract/runtime waves must precede later docs/onboarding/distribution waves.

## Test Requirements by Work Type (Mandatory)

1. Schema/artifact changes:
- schema validation tests
- fixture/golden updates
- compatibility/migration tests

2. CLI behavior changes:
- help/usage tests
- `--json` stability tests
- exit-code contract tests

3. Gate/policy/fail-closed changes:
- deterministic `covered`/`partial`/`gap` fixtures
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
- workflow lifecycle tests for multi-stage collect/map/gap/bundle/verify paths when applicable
- crash-safe/atomic-write tests
- contention/concurrency tests
- chaos suites when applicable
- end-to-end timeout/cancellation propagation checks for long-running workflows

6. SDK/adapter boundary changes:
- wrapper error-mapping tests
- adapter parity/conformance tests

7. Scenario/context changes:
- relevant scenario acceptance suites as applicable

8. Docs/examples changes:
- docs consistency checks
- storyline/smoke checks when user flow changes
- README/quickstart/integration coverage checks when public docs change
- docs source-of-truth sync tasks for `/Users/tr/axym/README.md`, `/Users/tr/axym/docs/`, `/Users/tr/axym/docs-site/public/llms.txt`, and `/Users/tr/axym/docs-site/public/llm/*.md`
- OSS trust-baseline updates when public launch/support expectations change

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
7. `Test Matrix Wiring`
8. Epic sections with objectives and stories
9. `Minimum-Now Sequence`
10. `Explicit Non-Goals`
11. `Definition of Done`

Epic and wave numbering contract:

- Use `## Epic W1: ...`, `## Epic W2: ...`, through `## Epic WN: ...` as needed.
- Story IDs under each epic must follow the same wave prefix, for example `W3-S2`.
- Do not invent extra waves unless they improve dependency ordering, risk control, or reviewability.

Story template:

- `### Story <ID>: <title>`
- `Priority:`
- `Tasks:`
- `Repo paths:`
- `Run commands:`
- `Test requirements:`
- `Matrix wiring:`
- `Acceptance criteria:`
- Optional: `Dependencies:`, `Risks:`

## Quality Gate

Before finalizing:

- Every recommendation maps to at least one epic/story.
- Every story is actionable without guesswork.
- Acceptance criteria are testable and deterministic.
- Paths are real and repo-relevant.
- Test requirements match story type.
- Matrix wiring exists for every story.
- Sequence is dependency-aware and implementation-ready.
- Wave numbering is explicit, sequential, and justified by dependency order.
- Earlier waves cover contract/runtime and architecture-boundary work before later docs/OSS/onboarding/distribution waves when both classes are present.
- Public/internal boundaries, integration hooks, and side effects are explicit where stories affect user-facing surfaces.
- Launch-facing plans include OSS trust-baseline and docs source-of-truth work when relevant.

## Failure Mode

If inputs are missing or recommendations are not plan-ready, write only:

- `No backlog plan generated.`
- `Reason:` concise blocker summary.
- `Missing inputs:` exact required fields.

Do not fabricate backlog content.
