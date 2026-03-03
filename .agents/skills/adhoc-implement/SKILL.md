---
name: adhoc-implement
description: Implement a user-specified Axym backlog plan end-to-end with strict branch bootstrap, story-by-story execution, required test-matrix wiring, CodeQL validation, and final DoD/acceptance revalidation.
disable-model-invocation: true
---

# Adhoc Plan Implementation (Axym)

Execute this workflow for: "implement this plan file", "run plan from <path>", or "execute backlog from a custom plan doc."

## Scope

- Repository: `.`
- Mandatory input argument: `plan_path`
- `plan_path` must point to a specific plan document provided by the user
- No default fallback to `product/PLAN_NEXT.md`
- Planning input only; this skill performs implementation work in repo

## Input Contract (Mandatory)

- Required: `plan_path`
- Accepted forms:
- absolute path
- repo-relative path
- Input must resolve to an existing readable file
- If `plan_path` is missing or invalid, stop with blocker report

## Preconditions

- Plan file includes required structure:
- `Global Decisions (Locked)`
- `Exit Criteria`
- `Test Matrix Wiring`
- Story sections with `Tasks`, `Repo paths`, `Run commands`, `Test requirements`, `Matrix wiring`, `Acceptance criteria`
- If structure is incomplete, stop and report missing sections

## Git Bootstrap Contract (Mandatory)

Run in order before implementation:

1. `git fetch origin main`
2. `git checkout main`
3. `git pull --ff-only origin main`
4. `git checkout -b codex/adhoc-<plan-scope>`

Rules:
- If worktree is dirty before step 1:
- Allow only the plan-handoff case where all modified files are planning outputs and include `plan_path`.
- Planning-output allowlist: `./product/PLAN_NEXT.md`, `./product/PLAN_v1.0.md`, and selected `plan_path`.
- In the allowlist case, require current branch is already `main`, run step 1, skip steps 2-3, and create the branch from fetched `origin/main` with `git checkout -b codex/adhoc-<plan-scope> origin/main` to preserve plan edits on an up-to-date base.
- Otherwise stop and report blocker.
- If unexpected unrelated changes appear during execution, stop immediately and ask how to proceed
- Do not auto-commit or auto-push unless explicitly requested by the user

## Workflow

1. Parse plan and build execution queue by dependency and priority (`P0 -> P1 -> P2`).
2. Run baseline before first edit:
- `make lint-fast`
- `make test-fast`
- Record failures as pre-existing vs introduced.
3. Implement one story at a time (no parallel story execution).
4. For each story:
- implement scoped code/docs/tests only
- keep orchestration thin; move parsing/persistence/reporting/policy logic into focused packages when boundary stories are in scope
- run story `Run commands`
- run story `Test requirements`
- run story `Matrix wiring` lanes
- mark complete only when acceptance criteria pass
5. Run epic-level validation after epic completion.
6. Run plan-level validation:
- `make prepush-full` (preferred), or
- `make prepush` plus `make codeql`
- Never finish without CodeQL unless explicitly waived by the user.
7. Revalidate all implemented work against:
- story acceptance criteria
- plan Definition of Done
- plan Exit Criteria
- Output `met/not met` with command evidence for each item.

## Execution Waves (Mandatory)

- Execute in two waves to keep blast radius controlled:
- Wave 1: contract/runtime correctness and architecture boundaries.
- Wave 2: docs, OSS hygiene, and distribution UX.
- Do not start Wave 2 work for a touched surface before Wave 1 acceptance criteria are met for that surface.

## Command Contract (JSON Required)

When collecting evidence or emitting machine-readable status, use `axym` commands with `--json`, for example:

- `axym collect --dry-run --json`
- `axym regress run --baseline <baseline-path> --frameworks eu-ai-act,soc2 --json`

## Test Requirements by Work Type (Mandatory)

1. Schema/artifact contract changes:
- schema validation tests
- fixture/golden updates
- compatibility or migration tests
- `make test-contracts`

2. CLI behavior changes (flags/JSON/exits):
- `cmd/axym/*_test.go` command coverage
- `--json` stability checks
- exit-code contract checks
- `axym version` discoverability and minimal dependency install-path checks when install/version surfaces are touched

3. Gate/policy/fail-closed changes:
- deterministic allow/block/require_approval and `decision.pass` fixtures
- fail-closed undecidable-path tests
- reason-code stability checks
- regression input-boundary tests (`invalid_record`/`schema_error`/`mapping_error` must not be treated as valid evidence)
- chain/dedup preservation tests (re-ingest must not duplicate or reorder records; previous-hash linkage must remain stable)
- filesystem boundary tests for user-supplied output paths (`non-empty + non-managed => fail`)
- ownership marker trust tests (`marker must be regular file`; reject symlink/directory)

4. Determinism/hash/sign/pack changes:
- byte-stability repeat-run tests
- canonicalization/digest stability checks
- verify/diff determinism tests
- `make test-contracts` when applicable

5. Job runtime/state/concurrency changes:
- lifecycle tests (submit/checkpoint/pause/resume/cancel)
- atomic write/crash safety tests
- contention/concurrency tests
- chaos lanes when scoped
- end-to-end cancellation/timeout propagation tests across CLI -> orchestration -> adapters

6. SDK/adapter boundary changes:
- wrapper behavior/error-mapping tests
- adapter conformance/parity tests
- `make test-adapter-parity` when applicable
- structured machine-readable error envelope tests for SDK/library paths
- extension-point compatibility tests when new enterprise integration seams are introduced

7. Voice/context changes:
- `relevant scenario acceptance suites` as applicable

8. Docs/examples changes:
- `make test-docs-consistency`
- `make test-docs-storyline` when flow changes
- README first-screen checks (what it is, who it is for, integration path, first value)
- source-of-truth checks between repo docs and generated/public docs
- problem -> solution framing and integration-before-internals checks for touched docs

## Test Matrix Wiring (Enforcement)

Every story must map to and run required lanes:

- Fast lane: `make lint-fast`, `make test-fast`
- Core lane: targeted unit/integration suites
- Acceptance lane: relevant `make test-*-acceptance` targets
- Cross-platform lane: preserve Linux/macOS/Windows behavior on touched surfaces
- Risk lane: determinism/safety/security/perf suites as required

No story is complete if any required lane is skipped or failing.

## Surgical Docs Sync Rule

- If a story changes user-visible behavior, update only impacted docs in the same story:
- `./README.md`
- `./docs/`
- `./docs-site/public/llms.txt`
- `./docs-site/public/llm/*.md`

Doc updates must preserve:
- first-screen README clarity for value and integration path
- integration guidance before internal architecture detail
- one canonical file/state lifecycle path model (diagram + path narrative) for touched workflows
- source-of-truth linkage between repo docs and generated/public docs

If internal-only behavior with no user-visible impact, avoid unnecessary doc churn.

## Safety Rules

- Preserve determinism, offline-first defaults, fail-closed enforcement, schema stability, and exit-code stability.
- Keep side effects explicit in API names/signatures and preserve symmetrical API semantics.
- Never weaken unapproved posture => regression failure behavior.
- Do not allow recursive cleanup on user-supplied paths without explicit ownership validation tests.
- No destructive git operations unless explicitly requested.
- No silent skips of required tests/checks.
- Keep changes tightly scoped to active story.

## Quality Rules

- Claims must be evidence-backed by executed commands/tests.
- Do not claim tests ran if they were not run.
- Tests must use temp dirs for generated artifacts; do not leak test outputs into tracked source paths.
- If docs/CLI drift occurs due to user-visible changes, patch docs in same story.
- For touched contracts, ensure public/internal/shim/deprecated API classification and schema version/migration notes are updated.

## Blocker Handling

If blocked:
1. Stop blocked story immediately.
2. Report exact blocker and affected acceptance criteria.
3. Continue only independent unblocked stories.
4. End with minimum unblock actions.

## Completion Criteria

Implementation is complete only when all are true:

- All non-blocked in-scope stories are implemented.
- Required story tests and matrix lanes pass.
- Plan Definition of Done is satisfied.
- Plan Exit Criteria is satisfied.
- CodeQL validation is green.

## Expected Output

- Execution summary: completed/deferred/blocked stories
- Change log: key files per story
- Validation log: commands and pass/fail
- Revalidation report: acceptance criteria + DoD + exit criteria (`met/not met` with evidence)
- Wave status: Wave 1 vs Wave 2 completion for touched surfaces
- Residual risk: remaining gaps and next required actions
