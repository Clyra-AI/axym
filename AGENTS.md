# AGENTS.md - Axym Repository Guide

Version: 1.0  
Last Updated: 2026-02-28  
Scope: This repository (`axym`) only.

## 1) Scope and Intent

- Build and maintain **Axym** only (the "Prove" product in See -> Prove -> Control).
- Do not implement Wrkr or Gait product features in this repo, except required interoperability via shared `Clyra-AI/proof` contracts and documented ingestion paths.
- Treat these docs as authoritative for product and engineering behavior:
  - `product/axym.md`
  - `product/dev_guides.md`
  - `product/architecture_guides.md`

## 2) Product North Star

Axym is an open-source Go CLI that captures structured AI governance evidence, emits signed/tamper-evident proof records, maps evidence to compliance controls, and produces audit-ready bundles.

Every change should improve one or more of:

- Evidence coverage (runtime, lifecycle, data pipeline, sibling ingest)
- Compliance clarity (control coverage, gap quality, auditability grade)
- Deterministic evidence output (signed proof records, chain integrity)
- Time-to-value (fast install, useful map/gaps/bundle output)

## 3) Non-Negotiable Engineering Constraints

- Deterministic pipeline only in default paths: **no LLM calls** in collect/map/gap/verify logic.
- Zero evidence exfiltration by default: records remain in user environment.
- Evidence is file-based and verifiable: output must be portable and auditable.
- Same input -> same output (maps/gaps/bundles), barring explicit timestamp/version fields.
- Prefer boring, auditable implementations over clever abstractions.

## 4) Required Architecture Boundaries

Keep changes aligned to these logical layers:

- Source adapters and collector acquisition
- Event normalization and proof record construction
- Proof emission (record creation, signing, chain append)
- Sibling ingestion/translation (Wrkr records, Gait pack translation)
- Context enrichment and compliance matching
- Coverage/gap/review/regression evaluation
- Bundle assembly/export/verification

Do not collapse these boundaries in ways that reduce testability or determinism.

## 5) Collector and Ingestion Best Practices

Collection must prioritize structured parsing over brittle text matching.

- Parse JSON/YAML/TOML and typed artifacts with schema-backed decoders when possible.
- Avoid regex-only logic for structured evidence payloads.
- Never extract raw secrets; only capture compliance-relevant metadata and hashed/redacted values.
- Every collector and ingestor should return stable, explainable outputs with explicit reason codes.

Minimum high-priority surfaces to preserve:

- MCP/tool invocation evidence
- LLM API middleware evidence
- CI/CD and deployment evidence
- Data pipeline evidence (dbt/Snowflake-style digest-first records)
- Ticket attachment/retry/DLQ evidence
- Wrkr proof ingestion
- Gait pack ingestion and trace/token translation
- Daily review and replay certification signals

## 6) Compliance, Integrity, and Gap Rules

Preserve these conventions in data model and behavior:

- Compliance mapping from proof records to framework controls is deterministic.
- Gap status remains explicit (`covered`, `partial`, `gap`) and actionable.
- Auditability grade derivation remains deterministic and explainable.
- Ingested/translated records must preserve provenance and relationship fields.
- Re-ingestion must be idempotent where required (no unintended duplicate chain effects).

Outputs should remain ranked and actionable (focus on material compliance risk, not noisy bulk output).

## 7) Proof and Contract Requirements

- Emit and validate records via `Clyra-AI/proof` primitives.
- Keep proof record semantics consistent (`tool_invocation`, `decision`, `policy_enforcement`, `approval`, `risk_assessment`, etc., as defined by proof/framework contracts).
- Maintain chain integrity and verifiability.
- Treat exit codes as API surface:
  - `0` success
  - `1` runtime failure
  - `2` verification failure
  - `3` policy/schema violation
  - `4` approval required
  - `5` regression drift
  - `6` invalid input
  - `7` dependency missing
  - `8` unsafe operation blocked

CLI output expectations:

- `--json` for machine-readable output
- `--explain` for human-readable rationale
- `--quiet` for CI-friendly operation

## 8) Toolchain and Dependency Standards

- Go is primary runtime language (single static binary model).
- Target toolchain versions:
  - Go `1.26.1`
  - Python `3.13+` (scripts/tools)
  - Node `22+` (docs/UI only; not core runtime logic)
- Use exact/pinned dependency versions where applicable.
- Avoid floating `@latest` in CI/build tooling.
- Keep `Clyra-AI/proof` dependency current with org policy (within one minor release of latest, and never below minimum supported baseline).
- For shared YAML config behavior, keep compatibility with `gopkg.in/yaml.v3`.

## 9) Testing and Validation Expectations

Any behavior change should include or update tests at the right level.

- Unit: isolated parser/collector/mapper behavior
- Integration: cross-component deterministic behavior
- E2E: CLI behavior, output contracts, and exit codes
- Scenario/spec tests: outside-in fixtures validating intended product behavior
- Contract tests: schema/output compatibility and stable artifacts
- Cross-product integration tests: Wrkr/Gait/proof interoperability where touched

Determinism requirements:

- Use no-cache flags where appropriate (for example `-count=1` in non-unit tiers).
- Golden outputs must be byte-stable unless intentionally updated.
- Keep benchmark/perf checks reproducible.

## 10) Security, Privacy, and Repo Hygiene

- Never commit secrets, credentials, generated binaries, or transient reports.
- Keep records and logs scrubbed of secret values.
- Treat sink unavailability, attach failures, and chain gaps as first-class reliability/compliance risks.
- Prefer fail-closed behavior for ambiguous high-risk conditions.

## 11) Documentation and Change Hygiene

- Keep docs and CLI behavior aligned.
- If commands, flags, exit codes, schema fields, mapping semantics, or integrity behavior change, update docs in `product/` and any user-facing command docs in the same PR.
- Keep terminology consistent with Axym domain language: evidence collection, compliance mapping, gap detection, auditability, proof records, bundles, regression.

## 12) Pull Request Checklist (Agent and Human)

- [ ] Change is in scope for Axym (not Wrkr/Gait product logic)
- [ ] Deterministic behavior preserved
- [ ] No evidence-data exfiltration introduced
- [ ] Proof/exit-code contracts preserved or explicitly versioned
- [ ] Tests added/updated at the correct layer
- [ ] Docs updated for externally visible changes
- [ ] Dependency/toolchain changes are pinned and justified
