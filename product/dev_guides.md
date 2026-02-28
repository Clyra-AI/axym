# Axym Development Standards

Version: 1.0
Status: Normative
Scope: Axym (`/Users/tr/axym`)

This document defines Axym's unified development infrastructure standards: toolchains, CI pipelines, testing, linting, security scanning, release integrity, performance budgets, and repo hygiene.

This is a toolchain and process specification, not an architecture description. For architecture execution rules (boundaries, ADR gates, failure matrices, and architecture PR checklists), see `/Users/tr/axym/product/architecture_guides.md`.

## 1) Enforcement-First Rule

Treat this document and CI/scripts as one contract.

- If a normative statement changes, update enforcement in the same PR.
- Workflow checks, branch-protection contracts, and docs parity tests are merge-blocking.
- Command/flag/exit-code changes in CLI code must include docs updates in the same PR.

## 2) Language Toolchains

### 2.1 Go

- Version policy: pin in `go.mod`; track latest stable within 2 minor releases.
- Current pin target: `1.25.7`.
- Module layout: `cmd/<binary>/` entry points, `core/` library packages, `internal/` non-exported packages.
- Build: `go build ./cmd/axym`.
- Version injection: `-ldflags "-s -w -X main.version={{ .Version }}"`.
- Dependency pinning: exact versions in `go.mod`; `go.sum` required.

### 2.2 Python

- Version: `3.13+`.
- Package manager: `uv`.
- Project config: `pyproject.toml` with `requires-python = ">=3.13"`.
- Dev dependencies via optional extras and lockfile.

### 2.3 Node / TypeScript

- Version: Node `22` LTS.
- Use `npm ci` for deterministic install.
- Scope: docs tooling and optional local UI helpers only, never core runtime logic.

## 3) Cross-Repo Interoperability

Axym is part of a shared dependency graph rooted in `Clyra-AI/proof`.

### 3.1 Pinned versions (current)

| Component | Version | Scope |
|---|---|---|
| Go | `1.25.7` | Axym repo + CI |
| `Clyra-AI/proof` | `>= v0.4.5` | Axym proof primitive dependency |
| Python | `3.13` | scripts and tooling |
| Node | `22` LTS | docs/tooling |

### 3.2 proof tracking policy

- Absolute minimum supported proof version: `v0.3.0`.
- Axym must track proof within 1 minor release of latest tag.
- `replace` directives for `Clyra-AI/proof` are local-dev only and never committed to `main`.

### 3.3 Shared dependency conventions

Pin transitive dependencies critical to proof compatibility:

- `github.com/gowebpki/jcs` (`v1.0.1`) for RFC 8785 canonicalization.
- `github.com/santhosh-tekuri/jsonschema/v5` (`v5.3.1`) for schema validation.
- `gopkg.in/yaml.v3` (`v3.0.1`) for framework/config parsing.
- `github.com/spf13/cobra` (`v1.8.1`) for CLI contract consistency.

## 4) Linting and Formatting

### 4.1 Go

| Tool | Purpose | Execution |
|---|---|---|
| `gofmt` | formatting | `gofmt -w .` |
| `go vet` | static analysis | `go vet ./...` |
| `golangci-lint` | lint aggregator | `golangci-lint run ./...` |
| `gosec` | security static analysis | `gosec ./...` |
| `govulncheck` | vuln database | `govulncheck -mode=binary ./cmd/axym` |

### 4.2 Python

- `ruff check`, `ruff format`
- `mypy` in strict mode
- `bandit -q -r <package>`

### 4.3 Markdown/docs

- Internal link validation in CI.
- CLI/docs parity checks (commands, flags, exits, reason codes).
- Diagram syntax validation for Mermaid where used.

### 4.4 Pre-commit hooks

At minimum enforce:

- whitespace and EOF normalization
- secret/key detection
- Go format + module hygiene
- lint-fast lane

## 5) Testing Matrix

Tests are organized into tiers by scope and risk.

### Tier 1 — Unit

- Isolated component tests (`go test ./...`).
- Run on every PR.

### Tier 2 — Integration

- Cross-component deterministic tests (collector -> record -> mapper).
- Run on PR and protected-branch pushes.

### Tier 3 — E2E CLI

- Build binary, invoke CLI via `exec.Command`, assert JSON outputs and exit codes.
- Run on protected-branch pushes.

### Tier 4 — Acceptance

- Blessed workflows end-to-end (`init`, `collect`, `map`, `gaps`, `bundle`, `verify`, `regress`).
- Release-gating and protected-branch lane.

### Tier 5 — Hardening

- Atomic write safety, lock contention, retry/backoff classification, error-envelope contracts.
- Nightly and release-gating.

### Tier 6 — Chaos

- Sink failures, queue outages, ticketing 429/5xx storms, chain-session gaps.
- Nightly and release-gating.

### Tier 7 — Performance

- Benchmark and runtime budget regressions (p50/p95/p99).
- Nightly and release-gating.

### Tier 8 — Contract

- Deterministic artifact bytes, stable exit codes, stable JSON shapes, schema compatibility.
- Blocking on protected-branch pushes.

### Tier 9 — Scenario

- Outside-in scenario fixtures under `scenarios/axym/**`.
- Validates implementation against externally defined expected outcomes.
- Blocking for protected-branch and release lanes.

### Tier 10 — Cross-product Integration

- End-to-end chain across Wrkr + Gait + Axym + proof verify.
- Confirms translated Gait artifacts and native proof records coexist in one valid chain.
- Nightly and release-gating.

## 6) Scenario and Simulation Standards

- Scenario fixtures are specification artifacts, not generated test noise.
- Changes to expected outcomes require CODEOWNERS/human review.
- Simulated services are boundary-level (`httptest`, fixture repos, canned API responses), not interface mocks.
- No real credentials in CI; all external dependencies must be simulated or fixture-backed.

## 7) Coverage Gates

| Scope | Threshold | Enforcement |
|---|---|---|
| Go core packages (`core/`, `cmd/`) | `>= 85%` | required in CI |
| Go all packages (per-package) | `>= 75%` | required in CI with explicit allowlist |
| Python tooling packages | `>= 85%` | required in CI |

Use deterministic coverage collection:

- Go: `-coverprofile=coverage.out`
- Python: `pytest --cov=<package> --cov-report=term-missing`

## 8) CI Pipeline Architecture

### 8.1 PR pipeline

- At least one fast merge-blocking lane on every `pull_request`.
- Include deterministic lint + unit + contract checks.
- Include at least one secondary-platform smoke lane.
- All PR workflows must set `concurrency` with `cancel-in-progress: true`.

### 8.2 Protected-branch pipeline

- Run full deterministic matrix after merge.
- Foundational contract/determinism checks are never path-gated.

### 8.3 Nightly pipelines

- Hardening
- Performance
- Extended security/compliance
- Platform-depth matrix

### 8.4 Release pipeline

Required sequence:

1. Release-gated acceptance and contract suites
2. Reproducible artifact build
3. Checksum generation and verification
4. SBOM generation
5. Artifact/SBOM vulnerability scan
6. Signing and provenance generation
7. In-pipeline verification before publish
8. Publish only after all gates pass

## 9) Branch Protection Contract

| Setting | Value |
|---|---|
| Required status checks | must map to `pull_request` workflows and tracked required-checks contract |
| Strict status checks | enabled |
| Required conversation resolution | enabled |
| Linear history | enabled |
| Force pushes | disabled |
| Branch deletion | disabled |
| Enforce admins | enabled |

Rules:

- Required checks must never reference non-PR jobs.
- Workflow renames/trigger changes must update contract tests in same PR.

## 10) Security Scanning

### 10.1 Static and dependency security

- CodeQL for Go/Python on PR + protected branch.
- `gosec`, `bandit`, `govulncheck` in CI lanes.
- Secret detection in pre-commit and CI fallback checks.

### 10.2 Release security

| Tool | Purpose |
|---|---|
| Grype | release artifact vulnerability scanning |
| Syft | SPDX SBOM generation |
| cosign | OIDC keyless signing |
| SLSA provenance | build attestation |

### 10.3 Verification commands

```bash
sha256sum -c dist/checksums.txt
cosign verify-blob --certificate dist/checksums.txt.pem \
  --signature dist/checksums.txt.sig dist/checksums.txt
```

## 11) Release Integrity

- Tooling: GoReleaser `v2.13.3`.
- Platforms: linux/darwin/windows x amd64/arm64.
- Artifacts: static binaries + archives + checksums + signatures + SBOM + provenance.
- Versioning: semver tags `vX.Y.Z` with build-time version injection.

## 12) Performance Budget Framework

Define and enforce committed budgets:

- `perf/bench_baseline.json` for benchmark regressions.
- `perf/runtime_slo_budgets.json` for CLI p50/p95/p99.
- `perf/resource_budgets.json` for memory/CPU/allocation budgets.

Deterministic benchmark requirements:

- `GOMAXPROCS=1`
- 5 samples
- median aggregation

## 13) Repo Hygiene

### 13.1 Required tracked files

- `product/axym.md`
- `product/architecture_guides.md`
- `product/dev_guides.md`
- `product/PLAN_*.md` (when planning docs are used)

### 13.2 Prohibited tracked files

- build output directories
- generated coverage artifacts
- built binaries
- ephemeral performance reports

### 13.3 Validation

- Enforce repo hygiene checks in `make lint` (or equivalent lane).
- Remediation via `git rm --cached <path>`.

## 14) Documentation Standards

- Docs are executable contract surfaces; parity with CLI is required.
- User-facing command docs must include `--json` and exit-code behavior.
- Buyer-facing docs should include FAQ sections where relevant.
- Keep machine-oriented docs (`llms.txt`, structured indexes/sitemaps) current when docs sites are used.

## 15) Determinism Standards

| Area | Standard |
|---|---|
| Benchmarks | `GOMAXPROCS=1` |
| Hardening/E2E | `-count=1` |
| Golden files | byte-for-byte CI match |
| JSON canonicalization | RFC 8785 (JCS) for signed/digested JSON |
| Archive determinism | fixed timestamps and stable ordering |
| Exit codes | stable API contract |

## 16) Exit Code Contract

Axym uses the shared Clyra exit contract:

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Internal/runtime failure |
| 2 | Verification failure |
| 3 | Policy block |
| 4 | Approval required |
| 5 | Regression failed |
| 6 | Invalid input |
| 7 | Dependency missing |
| 8 | Unsafe operation blocked |

All commands must support `--json`.
Most commands should support `--explain`.

## 17) Schema Management

- Store schemas under `schemas/v1/`.
- Keep one JSON Schema file per artifact type.
- Maintain paired `valid_*` and `invalid_*` fixtures.
- Backward-compatible changes stay in major version; breaking changes require new major schema version.
- Schema changes must trigger contract test updates.

## 18) PR Requirements

PR templates must require:

- `make fmt && make lint && make test` (or equivalent lanes) passed
- hardening review (error classes, exits, determinism, security/privacy)
- docs updates for user-visible behavior
- explicit note of contract impact and migration behavior when applicable

## 19) Standard Command Lanes

| Lane | Purpose |
|---|---|
| `make lint-fast` | fast lint and hygiene checks |
| `make test-fast` | fast deterministic test lane |
| `make test-contracts` | contract/schema/exit stability checks |
| `make test-scenarios` | scenario spec conformance |
| `make prepush` | default local push gate |
| `make prepush-full` | architecture/risk/release-risk local gate |
| `make test-hardening` | resilience and fault-tolerance checks |
| `make test-chaos` | fault injection and degradation validation |
| `make test-perf` | budget and regression checks |

If lane names differ in implementation, map them explicitly in this guide and CI docs.
