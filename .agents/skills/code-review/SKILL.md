---
name: code-review
description: Perform Codex-style full-repository review for Axym (not PR-limited), with severity-ranked findings focused on regressions, fail-closed safety, determinism, portability, and docs/CLI contract correctness.
disable-model-invocation: true
---

# Full-Repo Code Review (Axym)

Execute this workflow for: "review the codebase", "audit repo health", "run a full code review", or "find risks in Axym."

## Reviewer Personality

- Contract-first: behavior and guarantees over style.
- Boundary-enforced: architecture boundaries must be visible in package/API contracts, not docs-only claims.
- Regression-first: look for latent breakage paths.
- Fail-closed safety bias: block safety/control weakening.
- Scenario-driven: each finding includes concrete break path and impact.
- Portability-aware: Linux/macOS/CI/toolchain/path behavior.
- Signal over noise: findings-first, severity-ranked output.

## Scope

- Repository root: `.`
- Review entire repo, not only current diffs.
- Prioritize high-risk surfaces first, then remaining components.

## High-Risk Surfaces (Priority Order)

1. `core/source`, `core/detect`, `core/identity`, `core/risk`, `core/proof`, `core/regress`
2. `cmd/axym` CLI behavior, flags, exit codes, JSON outputs
3. `core/mcp` and adapter boundaries
4. `sdk/python` wrapper behavior and error mapping
5. `schemas/v1` and compatibility-sensitive artifacts
6. Public API/contract surfaces (stable/internal/shim/deprecated map, schema/version policy, migration paths)
7. `docs`, `README.md`, and `docs-site` command/contract accuracy and onboarding coherence

## Workflow

1. Build repository map and contract map from code/tests/help text, explicitly classifying stable/internal/shim/deprecated surfaces.
2. Validate contract compatibility before logic review: schema/version policy, CLI/JSON shape, and exit-code behavior.
3. Run baseline validation where feasible (lint/build/tests) and record gaps if not run.
4. Review each subsystem for:
   - Controls enforced at execution boundary (runtime/policy layer), not only prompt/UI boundary checks
   - Architecture boundary erosion (orchestration layer absorbing parser/persistence/report/policy logic)
   - Authoritative core drift (policy/signing/verification duplicated in wrappers instead of one core runtime)
   - Hidden side effects in API names/signatures
   - Asymmetric API semantics (`read` vs `read+validate`, `plan` vs `apply`)
   - Side-effect safety pattern gaps (plan/apply split, out-of-band stop, scoped approvals, destructive budgets)
   - Missing timeout/cancellation propagation in long-running paths
   - Missing extension seams that force enterprise forks
   - Safety/control bypasses
   - Fail-open handling on ambiguous high-risk paths (must fail closed)
   - Destructive filesystem operations on user-supplied paths without trusted ownership checks (regular-file marker validation, no marker-name-only trust)
   - Determinism or reproducibility breaks
   - Integrity verification weakening
   - Crash-safety/state handling gaps (atomic writes, lock strategy, contention behavior, permission checks)
   - Error contract gaps (taxonomy + stable JSON envelope + retryability hints)
   - False-green test/CI paths
   - Risk-tiered CI gate drift (fast/core/acceptance/cross-platform/chaos/perf) or missing merge policy ties
   - Portability/toolchain/path assumptions
   - Finding-class boundary leaks (invalid/non-record events entering mapped evidence state)
   - Chain-state clobbering (re-ingest duplicates, reordered records, or broken previous-hash linkage)
   - Additive evolution violations (breaking changes where additive fields + dual-reader compatibility are expected)
   - Schema/CLI contract drift
   - Missing/ambiguous schema versioning policy and migration expectations for changed contracts
   - Missing structured machine-readable errors for SDK/library users
   - Missing deterministic operator diagnostics/correlation (`doctor`-style checks, correlation IDs, local structured logs)
   - Version/install discoverability gaps (`axym version`, minimal dependency install path)
   - Adoption-path gaps (single quickstart path, expected outputs, integration diagrams, troubleshooting-first docs)
   - README first-screen clarity gaps (what it is, for whom, integration point, first value path)
   - Docs lifecycle/path model drift or docs source-of-truth ambiguity
   - Governance artifact drift (`ADR`, risk register, explicit non-goals, definition-of-done)
   - OSS trust-baseline gaps (`CONTRIBUTING`, `CHANGELOG`, `CODE_OF_CONDUCT`, issue/PR templates, security policy links)
   - Release-integrity/post-merge monitoring gaps (supply-chain verification, post-merge regression watch)
   - Cross-repo taxonomy/onboarding drift (README flow, naming labels, first-10-min path)
   - Docs/examples that do not match real behavior
5. Verify findings with concrete evidence (file refs, commands, test output).
6. Rank findings by severity and confidence.
7. Report minimum blocker set for safe release posture.

## Severity Model

- P0: release blocker, severe safety/integrity break, high reputational risk.
- P1: major behavioral regression or control bypass with real user impact.
- P2: meaningful correctness/portability/docs-contract issue.
- P3: minor maintainability concern.

## Finding Format

- `Severity`: P0/P1/P2/P3
- `Title`: short and action-oriented
- `Location`: file + line
- `Problem`: what is wrong
- `Break Scenario`: concrete failure path
- `Impact`: user/safety/CI/compliance effect
- `Fix Direction`: minimal safe correction

## Review Rules

- Findings are primary output; summaries stay brief.
- Treat execution-boundary-only controls (present in UI/prompt but absent in runtime enforcement) as at least `P1`.
- Treat fail-open behavior on ambiguous high-risk paths as at least `P1`.
- Treat recursive cleanup/delete on caller-selected paths with weak ownership gating as at least `P1`.
- Treat finding-boundary leaks that can cause false drift/exit `5` as at least `P1`.
- Treat lifecycle-state clobbering that reintroduces removed identities as at least `P2`.
- Treat missing machine-readable library errors where library integration is a supported path as at least `P2`.
- Treat missing additive compatibility on contract evolution (no dual-reader/migration path) as at least `P1`.
- Do not report style nits unless they cause runtime/contract risk.
- Do not claim tests/commands were run if they were not.
- Separate facts from inference.
- Provide a two-wave fix order for material findings:
  - Wave 1: contract/runtime correctness and architecture boundaries
  - Wave 2: docs/OSS hygiene/distribution UX
- If no findings, explicitly state `No material findings` and list residual risks/testing gaps.

## Command Anchors

- `axym collect --dry-run --json` to verify baseline runtime diagnostics and dependency posture.
- `axym regress run --baseline <baseline-path> --frameworks eu-ai-act,soc2 --json` to validate policy verdict/exit behavior.
- `axym verify --chain --json` to check artifact integrity and signature status.

## Output Contract

1. `Findings` (required, ordered by severity)
2. `Subsystem Coverage` (Green/Yellow/Red per major area)
3. `Open Questions / Assumptions` (if any)
4. `Residual Risk / Testing Gaps`
5. `Final Judgment`:
   - technical health today
   - minimum blockers (if any)
   - top 3 risk concentrations
6. `Fix Wave Plan` (when findings exist):
   - Wave 1 blockers (contract/runtime/architecture boundaries)
   - Wave 2 blockers (docs/OSS/distribution UX)
