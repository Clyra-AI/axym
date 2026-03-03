---
name: app-audit
description: Run an evidence-based audit for Axym OSS project across product, architecture, DX, security, and GTM readiness; output a strict go/no-go report with P0-P3 blockers and launch risk by technology, messaging, and expectations.
disable-model-invocation: true
---

# Axym OSS Audit

Execute this workflow when asked to perform app review, release readiness, architecture clarity, or public-launch audit for Axym.

## Scope

- Repository: `.`
- Analyze current code/docs only; do not invent features/markets.
- Default mode is read-only unless user explicitly asks for fixes.

## Workflow

1. Build whiteboard mental model from onboarding to sustained use, including the first-10-min path.
2. Map personas, JTBD, and MVP user-story coverage.
3. List required inputs/config/secrets/dependencies and setup friction.
4. Evaluate “aha” moments per primary user story.
5. Run technical validation on affected surfaces (build/lint/tests as needed), including architecture-boundary enforcement in code.
6. Audit docs for integration clarity:
   - where Axym sits in runtime path
   - what is customer code vs Axym vs tool/provider
   - sync vs async behavior and failure handling
   - README first screen: what it is, who it is for, integration path, first value
   - problem -> solution framing before primitive/feature lists
   - lifecycle/path model clarity and docs source-of-truth linkage (repo docs vs generated/public docs)
7. Compare stated product intent vs implemented behavior, including API surface map quality (stable/internal/shim/deprecated), schema/versioning policy clarity, and structured machine-readable error support.
8. Assess security posture and fail-closed guarantees.
   - explicitly test filesystem mutation boundaries on user-supplied output paths
   - confirm cleanup/reset flows reject non-managed dirs and reject marker symlink/directory types
9. Assess OSS readiness baseline and governance context:
   - `CONTRIBUTING`, `CHANGELOG`, `CODE_OF_CONDUCT`, issue/PR templates, and security policy links
   - maintainer/support expectations and standalone vs ecosystem dependency clarity
10. Assess market wedge sharpness for existing personas only.
11. Produce final go/no-go verdict with minimum blocker set and two-wave remediation order.

## Non-Negotiables

- Evidence-first: every claim must cite command output or file path.
- Boundary-first: explicitly separate ownership and integration points.
- Incident-first: lead with failure scenarios and operational impact.
- No cosmetics as blockers.
- Distinguish facts vs inference.
- Recommend fixes in two waves: Wave 1 contracts/runtime/architecture boundaries, then Wave 2 docs/OSS/distribution UX.

## Command Anchors

- `axym collect --dry-run --json` to capture environment diagnostics in machine-readable form.
- `axym bundle --audit sample --frameworks eu-ai-act,soc2 --json` to inspect artifact envelope integrity and payload shape.
- `axym regress run --baseline <baseline-path> --frameworks eu-ai-act,soc2 --json` to validate fail-closed policy behavior.

## Severity

- P0: release blocker / high reputational risk
- P1: major launch risk
- P2: meaningful gap, non-blocking for project
- P3: polish

## Output Contract

- Section 1: End-to-end product model
- Section 2: Persona/story coverage map
- Section 3: Inputs/config friction table
- Section 4: Aha analysis
- Section 5: Technical audit + release readiness
- Section 6: Business/market fit assessment
- Section 7: Final verdict (go/no-go) + top 3 launch risks
- Section 8: Fix wave plan (Wave 1 blockers, Wave 2 blockers)

Each section must include:
- Findings
- Evidence references
- Risk color (Green/Yellow/Red)
- Blockers (if any)
- Minimum fix set (only release-critical)
