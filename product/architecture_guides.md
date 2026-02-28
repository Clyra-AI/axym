# Axym Architecture Execution Guide

Version: 1.0
Status: Normative
Scope: Axym only (`/Users/tr/axym`)

This document is the actionable architecture companion to `/Users/tr/axym/product/dev_guides.md`.

`dev_guides.md` defines toolchain/process standards.
This file defines architecture execution rules, PR gates, and required artifacts.

## 1) Purpose

Use this guide to make architecture decisions auditable and repeatable, not opinion-based.

Every architecture-impacting change MUST show:

1. What contract changes (if any)
2. What failure modes are introduced/changed
3. What cost/performance impact is expected
4. What tests prove behavior under normal and fault conditions

## 2) Architecture Baseline (Axym)

Axym remains a deterministic, evidence-first, append-only compliance pipeline that produces verifiable artifacts through `Clyra-AI/proof`.

Required boundaries (do not collapse):

- Source adapters and collector acquisition
- Event normalization and record construction
- Proof emission (sign + chain append)
- Sibling ingestion and translation (Wrkr/Gait)
- Context enrichment and control matching
- Compliance mapping and gap detection
- Daily review and regression evaluation
- Bundle assembly/export and verification

Any PR that crosses boundaries directly MUST include an ADR (see Section 9).

### 2.1 Boundary Clarification (Current Axym v1 Contract)

Use this map to keep architecture reviews concrete while implementation is built:

| Boundary | Expected package area (target layout) |
|---|---|
| Source adapters | `core/collect/*`, `core/collector/*` |
| Normalization + proof emission | `core/proofemit/*`, `core/record/*` |
| Ingestion/translation | `core/ingest/*` |
| Compliance mapping | `core/compliance/*`, `core/map/*` |
| Gap/review/regression | `core/gaps/*`, `core/review/*`, `core/regress/*` |
| Bundle/export/verify | `core/bundle/*`, `core/export/*`, `core/verify/*` |

If package layout differs, preserve these boundaries as interfaces and document the mapping in PR notes.

## 3) TDD Standard (Required)

### 3.1 Red-Green-Refactor Contract

For behavior changes, teams MUST follow:

1. Add/adjust failing test(s) encoding intended behavior.
2. Implement minimal code to pass.
3. Refactor while keeping tests green.

### 3.2 Minimum test additions by change type

| Change type | Minimum required tests |
|---|---|
| Collector/parser changes | unit parser tests + contract fixtures/goldens |
| CLI output/exit changes | CLI contract tests (`--json`, exits, reason codes) |
| Mapping/gap logic | deterministic coverage tests + regression fixtures |
| Proof/chain behavior | chain/signature contract tests + tamper tests |
| Gait/Wrkr ingest translation | translation tests + legacy compatibility assertions |
| Replay/review/ticketing behavior | failure-mode tests (retry/DLQ/degradation) + scenario checks |
| Concurrency/state changes | contention/lifecycle tests + atomic write safety |

### 3.3 TDD evidence required in PR description

Every architecture-impacting PR MUST include:

- `Test intent`: what behavior was encoded first
- `Commands run`: exact command list
- `Result`: pass/fail summary

Minimum command baseline:

```bash
make lint-fast
make test-fast
make test-contracts
make test-scenarios
```

If architecture boundaries, mapper logic, adapters, or failure semantics changed:

```bash
make prepush-full
```

If Make targets are not yet scaffolded, use equivalent scripts and update this guide in the same PR.

## 4) Beyond 12-Factor: Cloud-Native Execution Factors

Axym is a CLI/runtime pipeline, but cloud-native execution factors still apply.

The following are REQUIRED where applicable:

### 4.1 Contract-first interfaces

- CLI JSON keys and exit codes are API contracts.
- Bundle file layout and manifest schema are API contracts.
- Contract changes MUST be explicit, versioned, and documented in the same PR.

### 4.2 Telemetry-first operability

- Emit machine-readable rationale (`--json`, reason codes).
- Degradation and data-quality signals MUST be explicit (`SINK_UNAVAILABLE`, `ENRICHMENT_LAG`, `CHAIN_SESSION_GAP`, `ATTACH_FAILED`).

### 4.3 AuthN/AuthZ and least privilege

- Default source access is read-only.
- New network paths MUST be explicit and opt-in.
- Compliance mode must fail closed on sink unavailability.

### 4.4 Externalized config, deterministic defaults

- Runtime config via flags/env/config files.
- Defaults MUST preserve deterministic local behavior and append-only integrity.

### 4.5 Immutable build and supply-chain integrity

- Reproducible builds, pinned dependencies, signed artifacts, SBOM/provenance.
- No floating `@latest` in CI-critical tooling.

### 4.6 Policy as code

- Compliance thresholds, mappings, and risk/context behavior are deterministic configuration.
- Rule IDs and reason codes must remain stable.

### 4.7 Failure visibility and graceful degradation

- Partial failure behavior MUST be explicit in outputs.
- Ambiguous high-risk paths MUST fail closed.

### 4.8 Backpressure and bounded work

- New loops/fan-out must define limits and deterministic ordering.
- Backpressure and retry behavior must declare caps and DLQ semantics.

## 5) Frugal Architecture and FinOps-by-Design

Frugality is a first-class non-functional requirement.

### 5.1 Required cost posture

- Prefer static/local analysis over remote calls.
- Keep non-deterministic enrichment optional.
- Avoid hidden daemons for OSS baseline workflows.

### 5.2 Required PR cost note

PRs affecting collect/map/review/bundle performance MUST include:

- CPU impact estimate (low/medium/high)
- Memory impact estimate (low/medium/high)
- I/O/network impact estimate (low/medium/high)
- Expected runtime delta for representative scenarios

### 5.3 Performance guardrails

When touching hot paths, run:

```bash
make test-perf
```

If budgets regress, PR MUST include mitigation or approved exception.

## 6) Chaos Engineering Operating Standard

Chaos is a reliability gate for risk-bearing paths.

### 6.1 Required hypothesis format

Each chaos addition MUST define:

- `Steady state`: measurable normal behavior
- `Fault`: injected failure mode
- `Expected`: deterministic failure class/exit/recovery behavior
- `Abort condition`: when to stop

### 6.2 Minimum chaos coverage triggers

Add/extend chaos tests when introducing:

- New external dependency path
- New sink/write path or fsync contract
- New retry/backoff behavior
- New chain stitching/session-boundary behavior
- New ticket attach/DLQ workflow

### 6.3 Required commands

```bash
make test-chaos
make test-hardening
```

## 7) Fowler-Style App Architecture Governance

Axym favors explicit boundaries and evolutionary fitness functions.

### 7.1 Design rules

- Prefer simple modular design over speculative abstractions.
- Isolate provider/tool schemas behind adapter boundaries.
- Keep collectors as adapters, not compliance decision engines.
- Avoid boundary leakage (for example: mapper reading raw source payloads directly).
- No shared mutable global state across layers.

### 7.2 Fitness functions (must stay green)

- Determinism and byte-stability checks
- Contract and schema compatibility checks
- Fail-closed safety checks
- Scenario acceptance checks
- Cross-product interoperability checks (Wrkr/Gait/proof)

## 8) System Architecture Best Practices (Operational)

### 8.1 Reliability

- Define expected failure behavior per command (exit code + reason class).
- Keep dependency outages visible in JSON outputs and Daily Review artifacts.

### 8.2 Security

- No secret extraction in output artifacts.
- No default scan/evidence-data exfiltration.
- Signature and chain verification are release-blocking integrity checks.

### 8.3 Data and evidence

- Evidence artifacts are portable, verifiable, and reproducible.
- Chain integrity remains independently verifiable via `proof verify`.

### 8.4 Cross-product interoperability

- `Clyra-AI/proof` contracts are non-negotiable interfaces.
- Wrkr/Gait ingest compatibility regressions are blocking.

### 8.5 Failure/Degradation Matrix (Axym v1)

| Condition | Expected behavior class | Expected signal surface |
|---|---|---|
| `collect` in compliance mode and sink unavailable | Fail closed | exit `1`, reason `SINK_UNAVAILABLE` |
| `collect` in advisory/shadow mode and sink unavailable | Graceful degradation with explicit warning | exit `0`, degradation reason in JSON + Daily Review |
| Collector emits malformed proof records | Fail closed for malformed payload, continue other collectors deterministically | rejected record count + typed validation errors |
| `verify --chain` detects break | Fail closed | exit `2`, exact break point |
| `verify --bundle` cryptographic or manifest failure | Fail closed | exit `2`, typed verification failure |
| `regress run` detects control coverage drift | Deterministic drift failure | exit `5`, list of regressed controls |
| Invalid framework/mapping/policy input | Fail closed | exit `6`, invalid input/schema violation |
| Missing required dependency/config source | Fail closed | exit `7`, dependency missing |
| Unsafe bundle output path | Fail closed | exit `8`, unsafe operation blocked |

When modifying any row behavior, update:

- CLI/error contract tests
- command reference docs
- ADR section (per Section 9)

## 9) Required ADR for Architecture-Impacting Changes

Create/update an ADR section in PR description (or link to product ADR file) when:

- Changing boundaries or data flow between architecture layers
- Introducing a new provider/adapter dependency
- Changing contract semantics or mapping model behavior
- Altering failure handling class for any command

ADR minimum template:

```text
Title:
Context:
Decision:
Alternatives considered:
Tradeoffs:
Rollback plan:
Validation plan (commands + acceptance criteria):
```

## 10) PR Gate Checklist (Must Pass)

- [ ] Boundary impact assessed; ADR provided when required
- [ ] TDD evidence included (test intent + commands run)
- [ ] Contract impact documented (JSON keys, exits, schemas)
- [ ] Failure-mode deltas documented (fail-open/fail-closed rationale)
- [ ] Cost/perf impact documented
- [ ] Required lanes executed for scope (`prepush`/`prepush-full` + risk lanes)
- [ ] Docs updated for user-visible behavior changes in same PR

## 11) Command Matrix (Architecture-focused)

| Change scope | Required command set |
|---|---|
| Standard behavior change | `make prepush` |
| Architecture/mapping/adapter/failure change | `make prepush-full` |
| Contract/schema/bundle change | `make test-contracts` + `make test-scenarios` |
| Reliability/fault-tolerance change | `make test-hardening` + `make test-chaos` |
| Performance-sensitive change | `make test-perf` |

## 12) Non-goals

- This guide does not replace contributor onboarding docs.
- This guide does not redefine Axym product scope.
- This guide does not allow non-deterministic behavior in default collect/map/bundle/proof paths.
