# PLAN v1.0: Axym (Deterministic Evidence-to-Compliance Build Plan)

Date: 2026-02-28
Source of truth: `product/axym.md`, `product/dev_guides.md`, `product/architecture_guides.md`, `AGENTS.md`
Scope: Axym OSS CLI only (Prove product). No Wrkr/Gait feature implementation beyond documented ingestion and `Clyra-AI/proof` interoperability contracts.

This is an execution plan. Every story is implementation-ready with concrete paths, commands, acceptance criteria, test tiers, CI lane wiring, and architecture governance fields.

---

## Global Decisions (Locked)

- Axym runtime is Go-first and deterministic by default; no LLM calls in collect/map/gaps/verify paths.
- Toolchain contract is pinned: Go `1.25.7`, Python `3.13+` (tooling only), Node `22` (docs/UI only).
- `Clyra-AI/proof` is a hard dependency and interface contract: Axym must remain within one minor release of latest and never below minimum supported baseline.
- Zero evidence exfiltration is default. Evidence artifacts remain local unless user explicitly exports or integrates.
- Evidence is file-based, append-only, portable, and verifiable.
- CLI/API contracts are stable: `--json` on all commands, `--quiet` for CI, `--explain` on major diagnostic flows.
- Exit code contract is locked: `0,1,2,3,4,5,6,7,8`.
- Architecture boundaries are mandatory and testable: collect -> normalize/proof emit -> ingest/translate -> context enrich -> map/gaps -> review/regress -> bundle/export/verify.
- Go core is authoritative for enforcement/verification logic. Python remains a thin tooling layer.
- Structured parsing is required for structured artifacts (JSON/YAML/TOML); regex-only parsing is not acceptable for structured payloads.
- Fail-closed behavior is required for ambiguous high-risk conditions, invalid schema/input, verification failures, and unsafe output operations.
- Compliance mode defaults to evidence-loss budget `0` with durable write/queue guarantee and explicit degradation signals.
- Idempotent re-ingest and chain integrity are non-negotiable invariants.
- Rule IDs, reason codes, JSON schemas, and bundle layouts are versioned contract surfaces.
- Release integrity is mandatory: reproducible builds, checksums, SBOM, vulnerability scan, signing, provenance, in-pipeline verify-before-publish.

---

## Current Baseline (Observed)

Repository snapshot (2026-02-28):

- Present: `product/axym.md`, `product/dev_guides.md`, `product/architecture_guides.md`, `AGENTS.md`, `.agents/skills/*`.
- Missing runtime scaffold: no `go.mod`, `cmd/axym`, `core/`, `internal/`, `schemas/`, `scenarios/`, `scripts/`, `testinfra/`.
- Missing CI/release scaffold: no `.github/workflows`, no `.goreleaser.yaml`, no baseline `Makefile`.
- Missing user/runtime docs scaffold: no top-level `README.md`, `CONTRIBUTING.md`, `SECURITY.md`.
- Baseline command surface in repo today: planning/docs only; no buildable CLI.
- Gap to PRD is full implementation breadth across FR1-FR13, AC1-AC17, and NFR determinism/reliability/security/performance gates.

---

## Exit Criteria

Axym v1.0 is complete only when all criteria are automated and green:

1. AC1: `install -> init -> collect -> map` delivers a real framework map in <= 15 minutes for fixture environment.
2. AC2: `axym bundle --audit <name>` includes board-ready `executive-summary.pdf` with required fields.
3. AC3: auditor can run `proof verify --bundle` on Axym bundle without Axym installed.
4. AC4: `axym verify --chain` detects tamper and identifies exact break point.
5. AC5: `axym gaps` emits ranked, actionable gap output with remediation and effort.
6. AC6: built-in collector fixtures produce valid records at 100% expected-capture coverage.
7. AC7: non-blocking collection mode keeps agent flow alive on collector failures with explicit failure surfaces.
8. AC8: offline/no-egress run succeeds post-install with no unexpected outbound calls.
9. AC9: mixed-source chain (Wrkr + Gait + Axym) verifies via both `axym verify --chain` and `proof verify --chain`.
10. AC10: data pipeline evidence path includes SoD/freeze/query-tag semantics and replay certification outputs.
11. AC11: compliance mode fails closed on sink failure with deterministic reason signals.
12. AC12: OSCAL v1.1 export in bundle is valid and importable.
13. AC13: `axym regress run` exits `5` on control coverage drift with deterministic regressed-control output.
14. AC14: `axym review --date` emits complete Daily Review Pack including exception classes and grade distributions.
15. AC15: ticket attach path meets retry/DLQ behavior and SLA/SLO accounting.
16. AC16: override artifacts are signed, chain-linked, append-only, and visible in bundle.
17. AC17: third-party collector protocol works with strict schema rejection of malformed records.
18. NFR gate: deterministic outputs are byte-stable for same inputs (except explicit time/version fields).
19. NFR gate: zero default evidence exfiltration.
20. NFR gate: performance budgets hold (collect latency, map throughput, bundle/verify windows).
21. NFR gate: release integrity and security scans pass in release lane.

---

## Test Matrix Wiring

Tier model used in this plan (explicit mapping to `product/dev_guides.md`):

- Tier 1 Unit: isolated package/parser/scorer tests.
- Tier 2 Integration: deterministic cross-component tests (`-count=1`).
- Tier 3 E2E CLI: command invocation, `--json`, exit-code assertions.
- Tier 4 Acceptance: end-to-end operator workflows.
- Tier 5 Hardening: atomic writes, locking, retry/error-envelope resilience.
- Tier 6 Chaos: controlled fault injection and fail-closed/degradation behavior.
- Tier 7 Performance: benchmark/runtime budget checks.
- Tier 8 Soak: long-running stability and sustained contention checks.
- Tier 9 Contract: schema/JSON-shape/exit-code/artifact-byte compatibility.
- Tier 10 UAT: install-path and packaged-artifact validation.
- Tier 11 Scenario: specification-driven fixtures (`scenarios/axym/**`).
- Tier 12 Cross-product Integration: Wrkr/Gait/proof interoperability.

Tier alias notes for dev-guide alignment:

- Plan Tier 9 Contract maps to dev-guide Contract tier.
- Plan Tier 11 Scenario maps to dev-guide Scenario tier.
- Plan Tier 12 Cross-product Integration maps to dev-guide Cross-product Integration tier.

Lane definitions:

- Fast lane: pre-push checks (`make lint-fast`, focused Tier 1/Tier 9).
- Core CI lane: mandatory Tier 1-3 on PR/main.
- Acceptance lane: Tier 4 + Tier 11 required for workflow readiness.
- Cross-platform lane: impacted suites on Linux/macOS/Windows.
- Risk lane: Tier 5/6/7/8/9/12 for high-risk stories.

Pipeline placement contract:

- PR pipeline: Fast lane mandatory; selected Core CI; at least one non-Linux smoke lane; contract checks merge-blocking.
- Main pipeline: full Core CI + Acceptance + Tier 9 contracts.
- Nightly pipeline: Tier 5/6/7/8 + expanded platform-depth + Tier 12 interop.
- Release pipeline: Tier 4/9/10/12 + signing/provenance/SBOM/vuln scan + verify-before-publish.

Story-to-lane map:

| Story | Fast | Core CI | Acceptance | Cross-platform | Risk |
|---|---|---|---|---|---|
| 0.1 | Yes | Yes | No | Yes | No |
| 0.2 | Yes | Yes | No | Yes | No |
| 0.3 | Yes | Yes | Yes | Yes | Yes |
| 1.1 | Yes | Yes | Yes | Yes | No |
| 1.2 | Yes | Yes | Yes | Yes | Yes |
| 1.3 | Yes | Yes | Yes | Yes | Yes |
| 2.1 | Yes | Yes | Yes | Yes | Yes |
| 2.2 | Yes | Yes | Yes | Yes | Yes |
| 2.3 | Yes | Yes | Yes | Yes | Yes |
| 3.1 | Yes | Yes | Yes | Yes | Yes |
| 3.2 | Yes | Yes | Yes | Yes | Yes |
| 3.3 | Yes | Yes | Yes | Yes | Yes |
| 4.1 | Yes | Yes | Yes | Yes | Yes |
| 4.2 | Yes | Yes | Yes | Yes | Yes |
| 4.3 | Yes | Yes | Yes | Yes | Yes |
| 5.1 | Yes | Yes | Yes | Yes | Yes |
| 5.2 | Yes | Yes | Yes | Yes | Yes |
| 6.1 | Yes | Yes | Yes | Yes | Yes |
| 6.2 | Yes | Yes | Yes | Yes | Yes |
| 6.3 | Yes | Yes | Yes | Yes | Yes |
| 7.1 | Yes | Yes | Yes | Yes | No |
| 7.2 | Yes | Yes | Yes | Yes | Yes |
| 7.3 | Yes | Yes | Yes | Yes | Yes |
| 8.1 | No | Yes | Yes | Yes | Yes |
| 8.2 | Yes | Yes | Yes | Yes | Yes |

Gating rule:

- A story is not complete unless every lane marked `Yes` above is green in its assigned pipeline.
- Merge to `main` blocks on required lane failures.
- Release tag/publish blocks on any release-lane failure.

---

## Epic 0: Foundations, Scaffold, and Contract Rails

Objective: create a buildable Axym repository skeleton with pinned toolchains, deterministic lanes, and enforceable CI/security/release gates.
Traceability: FR11, NFR2-NFR5, dev guide Sections 2/4/5/8/10/11/16/17/19.

### Story 0.1: Bootstrap Axym runtime scaffold and contract directories
Priority: P0
Tasks:
- Create canonical layout: `cmd/axym/`, `core/`, `internal/`, `schemas/v1/`, `scripts/`, `testinfra/`, `scenarios/axym/`, `.github/workflows/`.
- Initialize Go module and root binary entrypoint.
- Add required tracked planning docs pattern support (`product/PLAN_*.md`) and repo hygiene guard scaffold.
- Add baseline docs required for OSS runtime consumption.
Repo paths:
- `go.mod`
- `cmd/axym/main.go`
- `core/`
- `internal/`
- `schemas/v1/`
- `scenarios/axym/`
- `README.md`
- `.gitignore`
Run commands:
- `go mod tidy`
- `go build ./cmd/axym`
- `go test ./...`
Test requirements:
- Tier 1: module/bootstrap tests for entrypoint/package load.
- Tier 9: repository hygiene contract checks for required/prohibited tracked artifacts.
Matrix wiring:
- Lanes: Fast, Core CI, Cross-platform.
- Pipeline placement: PR (Tier 1 + hygiene subset), Main (same checks on linux/macos/windows).
Acceptance criteria:
- `go build ./cmd/axym` passes on all target OS runners.
- Required scaffold paths exist and are test-validated.
Architecture constraints:
- Preserve boundary-aligned package roots from `product/architecture_guides.md` Section 2.
ADR required: no
TDD first failing test(s):
- `internal/bootstrap/bootstrap_test.go` fails until module entrypoint and package roots exist.
Cost/perf impact: low
Chaos/failure hypothesis:
- Not risk-bearing for runtime behavior; chaos lane not required.

### Story 0.2: Pin toolchains, dependency policy, and local command lanes
Priority: P0
Tasks:
- Pin Go/Python/Node versions in local and CI configs.
- Add `Makefile` lanes (`lint-fast`, `test-fast`, `test-contracts`, `test-scenarios`, `prepush`, `prepush-full`, `test-hardening`, `test-chaos`, `test-perf`).
- Add pre-commit hooks for secrets, formatting, and lint-fast.
- Pin initial dependency set including `Clyra-AI/proof` compatibility requirements.
Repo paths:
- `.tool-versions`
- `Makefile`
- `.pre-commit-config.yaml`
- `go.mod`
- `go.sum`
Run commands:
- `make lint-fast`
- `make test-fast`
- `make test-contracts`
- `go test ./...`
Test requirements:
- Tier 1: make-target and helper-script smoke tests.
- Tier 9: pinned-version/no-floating-tool contract checks.
Matrix wiring:
- Lanes: Fast, Core CI, Cross-platform.
- Pipeline placement: PR (lint-fast + test-fast), Main (full lane set on linux/macos/windows).
Acceptance criteria:
- Pinned toolchain checks fail deterministically on version drift.
- `Clyra-AI/proof` compatibility policy is machine-checked.
Architecture constraints:
- Node remains docs/tooling-only; no runtime dependency on Python/Node in core CLI behavior.
ADR required: no
TDD first failing test(s):
- `testinfra/contracts/toolchain_pin_test.go` for pinned versions and dependency policy.
Cost/perf impact: low
Chaos/failure hypothesis:
- Not risk-bearing for runtime behavior; chaos lane not required.

### Story 0.3: Wire PR/main/nightly/release pipelines with security and release integrity
Priority: P0
Tasks:
- Implement workflows for PR, protected branch, nightly, and release with pinned actions.
- Enforce workflow concurrency (`cancel-in-progress: true`) where required.
- Wire scanners (`golangci-lint`, `gosec`, `govulncheck`, CodeQL, docs-link checks).
- Add release pipeline steps: build -> checksums -> SBOM -> vuln scan -> signing -> provenance -> verify -> publish.
- Add branch-protection contract tests for required status mapping to PR-triggered workflows.
Repo paths:
- `.github/workflows/pr.yml`
- `.github/workflows/main.yml`
- `.github/workflows/nightly.yml`
- `.github/workflows/release.yml`
- `.goreleaser.yaml`
- `scripts/check_branch_protection_contract.sh`
- `testinfra/contracts/ci_contract_test.go`
Run commands:
- `go test ./... -count=1`
- `govulncheck -mode=binary ./cmd/axym`
- `make prepush-full`
- `sha256sum -c dist/checksums.txt`
Test requirements:
- Tier 2/3: workflow and CI command-path checks.
- Tier 9: required-check mapping, trigger, and contract tests.
- Tier 10: packaged install-path checks.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (fast + contract subset), Main (full core + contract), Nightly (security/perf/chaos expansions), Release (integrity gate sequence).
Acceptance criteria:
- Required checks map to PR-emitted statuses only.
- Release artifacts are signed, checksum-verified, SBOM-attested, and provenance-stamped before publish.
Architecture constraints:
- Security/release infrastructure must not weaken deterministic build or offline runtime defaults.
ADR required: no
TDD first failing test(s):
- `testinfra/contracts/ci_required_checks_test.go` for trigger/status-contract enforcement.
Cost/perf impact: medium
Chaos/failure hypothesis:
- Steady state: release pipeline produces verified signed artifacts.
- Fault: missing signature/provenance step or unmatched required status.
- Expected: pipeline fails closed before publish.
- Abort condition: any publish step reached without full verification evidence.

---

## Epic 1: Proof Record Construction, Emission, and Verification Core

Objective: implement deterministic record normalization, proof emission, append-only chain persistence, and verify surfaces.
Traceability: FR4, FR11, NFR1/NFR3/NFR4, AC4/AC8/AC11.

### Story 1.1: Build record normalization, schema validation, and redaction pipeline
Priority: P0
Tasks:
- Implement event normalization contracts per source type into canonical intermediate model.
- Integrate schema validation and required-field checks prior to proof record creation.
- Implement configurable redaction actions (`hash`, `omit`, `mask`) before write/sign.
- Enforce rejection of malformed records with typed invalid-input errors.
Repo paths:
- `core/record/`
- `core/normalize/`
- `core/redact/`
- `schemas/v1/record/`
- `internal/integration/record/`
Run commands:
- `go test ./core/record/...`
- `go test ./... -count=1`
- `make test-contracts`
Test requirements:
- Tier 1: parser/normalizer/redaction unit tests.
- Tier 2: normalize->validate integration fixtures.
- Tier 9: schema compatibility + valid/invalid fixture contract tests.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform.
- Pipeline placement: PR (Tier 1 + contract subset), Main (Tier 2 + full contract suites).
Acceptance criteria:
- Invalid records never proceed to chain append.
- Redaction outputs are deterministic and reproducible for same input.
Architecture constraints:
- Keep normalization/record construction separate from collectors and from compliance mapping.
ADR required: no
TDD first failing test(s):
- `core/record/normalize_test.go` and `core/redact/redaction_rules_test.go`.
Cost/perf impact: medium
Chaos/failure hypothesis:
- Steady state: valid normalized records pass schema and redaction.
- Fault: malformed record or unsupported redaction rule.
- Expected: fail closed with exit `6` classification for invalid input.
- Abort condition: malformed record enters append path.

### Story 1.2: Implement deterministic proof emission, signing, chain append, and idempotent persistence
Priority: P0
Tasks:
- Build proof emitter using `proof.NewRecord()`, `proof.Sign()`, and `proof.AppendToChain()`.
- Implement append-only local store with atomic writes and fsync semantics in compliance mode.
- Implement idempotent dedupe key (`source_product + record_type + event_hash`) with bounded TTL index.
- Add explicit degradation/fail-closed policy handling for sink failures.
Repo paths:
- `core/proofemit/`
- `core/store/`
- `core/store/dedupe/`
- `core/policy/sink/`
- `internal/hardening/store/`
Run commands:
- `go test ./core/proofemit/... -count=1`
- `make prepush-full`
- `make test-hardening`
- `make test-chaos`
Test requirements:
- Tier 2: emitter/store integration tests.
- Tier 5: atomic write, lock contention, crash-safety, retry classification.
- Tier 6: sink-unavailable fault injection with fail-closed/advisory-mode assertions.
- Tier 9: deterministic chain/hash/signature and reason-code stability checks.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (Tier 1/2 subset), Main (Tier 2 + Tier 9), Nightly (Tier 5/6 contention/fault suites).
Acceptance criteria:
- Compliance mode returns success only after durable write or durable enqueue.
- Re-ingest of same record payload does not duplicate or reorder chain entries.
Architecture constraints:
- Proof emission layer is authoritative for chain append semantics; collectors cannot append directly.
ADR required: yes
TDD first failing test(s):
- `internal/hardening/store/atomic_append_test.go`.
- `internal/integration/proofemit/dedupe_idempotency_test.go`.
Cost/perf impact: high
Chaos/failure hypothesis:
- Steady state: append path preserves strict sequence and idempotent dedupe.
- Fault: disk full, fsync error, queue unavailable, stale lock.
- Expected: compliance mode fails closed; advisory mode emits explicit degradation signals.
- Abort condition: silent data loss or duplicate-chain append.
Semantic invariants:
- Same ordered input set -> same chain order and hashes.
- Re-ingest must not duplicate, reorder, or alter previous-hash linkage.
- `invalid_record`, `schema_error`, `mapping_error` are never treated as valid evidence.

### Story 1.3: Implement verify surfaces for chain and bundle cryptographic integrity
Priority: P0
Tasks:
- Implement `axym verify --chain` command path with precise break-point diagnostics.
- Implement cryptographic bundle verification delegation to proof primitives.
- Enforce typed verification failure envelopes and exit code `2`.
- Add unsafe output-path and marker trust checks for verify temporary artifacts.
Repo paths:
- `core/verify/`
- `cmd/axym/verify.go`
- `internal/e2e/verify/`
- `testinfra/contracts/verify_contract_test.go`
Run commands:
- `axym verify --chain --json`
- `axym verify --bundle ./fixtures/bundles/good --json`
- `go test ./... -count=1`
- `make prepush-full`
Test requirements:
- Tier 3: CLI verify behavior (`--json`, exits, reason codes).
- Tier 4: tamper/break-point acceptance fixtures.
- Tier 9: verify JSON envelope and exit-code contract stability.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (Tier 3 subset), Main (full Tier 3/4/9), Nightly (expanded tamper matrix).
Acceptance criteria:
- Chain tamper at any position is detected and reported with exact index/record ID.
- `verify --chain` and `proof verify --chain` agree on pass/fail for same chain.
Architecture constraints:
- Verification logic remains separate from mapping/compliance-opinion logic.
ADR required: no
TDD first failing test(s):
- `internal/e2e/verify/chain_breakpoint_test.go`.
Cost/perf impact: medium
Chaos/failure hypothesis:
- Steady state: verify succeeds on intact chain and valid bundle.
- Fault: hash mismatch, missing link, signature mismatch, unsafe temp path.
- Expected: fail closed with exit `2` (or exit `8` for unsafe operation) and typed reason.
- Abort condition: verification passes after intentional tamper.
Semantic invariants:
- Verification result is deterministic for fixed artifact set.
- Break-point identification is stable across repeated runs.

---

## Epic 2: Collector Acquisition and Plugin Runtime

Objective: implement deterministic adapter-first collection across required evidence surfaces and extensible plugin protocol.
Traceability: FR1, FR2, FR3, FR12, AC6/AC10/AC17, NFR1/NFR3.

### Story 2.1: Implement collector registry and runtime collectors (MCP, LLM middleware, webhook, CI, git)
Priority: P0
Tasks:
- Build collector interface, registry, and deterministic execution ordering.
- Implement built-in collectors: MCP logs, LLM middleware events, webhook, GitHub Actions events, git metadata.
- Emit only compliance-relevant metadata and hashed/redacted sensitive fields.
- Add collector-level explicit reason codes and per-source result summaries.
Repo paths:
- `core/collector/`
- `core/collect/`
- `core/collect/mcp/`
- `core/collect/llmapi/`
- `core/collect/webhook/`
- `core/collect/githubactions/`
- `core/collect/gitmeta/`
Run commands:
- `axym collect --dry-run --json`
- `axym collect --json`
- `go test ./core/collect/... -count=1`
- `make prepush-full`
Test requirements:
- Tier 1: collector parser/adapter units.
- Tier 2: collector->record integration fixtures.
- Tier 3: CLI collect dry-run vs write behavior.
- Tier 4: acceptance fixture for multi-collector run.
- Tier 9: stable collector summary schema and reason-code contracts.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (Tier 1/2 subset), Main (Tier 2/3/4/9), Nightly (expanded fixture scale).
Acceptance criteria:
- `axym collect --dry-run --json` reports deterministic would-capture output without writes.
- Built-in collector fixtures achieve expected capture parity and schema validity.
Architecture constraints:
- Collectors are adapters only; no compliance decision logic in collector packages.
ADR required: yes
TDD first failing test(s):
- `core/collect/registry_order_test.go`.
- `internal/integration/collect/multi_source_fixture_test.go`.
Cost/perf impact: medium
Chaos/failure hypothesis:
- Steady state: collectors emit deterministic normalized events.
- Fault: one collector fails or emits malformed payload.
- Expected: malformed payload rejected; other collectors continue deterministically; failure surfaced explicitly.
- Abort condition: collector failure silently drops signal or blocks deterministic ordering.

### Story 2.2: Implement data pipeline collectors with SoD/freeze/enrichment and replay-cert input semantics
Priority: P0
Tasks:
- Implement dbt and Snowflake digest-first collectors with canonicalization.
- Implement SoD enforcement and freeze-window evaluation (`decision.pass`, reason codes).
- Implement enrichment lag and missing-query-tag behavior (`ENRICHMENT_LAG`, `MISSING_QUERY_TAG`).
- Emit `data_pipeline_run` records and replay-certification inputs for later replay story.
Repo paths:
- `core/collect/dbt/`
- `core/collect/snowflake/`
- `core/policy/sod/`
- `core/policy/freeze/`
- `internal/integration/datapipeline/`
Run commands:
- `axym collect --json`
- `go test ./core/collect/dbt/... -count=1`
- `go test ./core/collect/snowflake/... -count=1`
- `make prepush-full`
- `make test-hardening`
- `make test-chaos`
Test requirements:
- Tier 1: parser/canonicalization and policy-evaluator units.
- Tier 2: dbt/snowflake integration with deterministic fixtures.
- Tier 4: AC10 acceptance workflow fixture.
- Tier 5: retry/backoff and enrichment-lag hardening tests.
- Tier 6: sink/query-source outages and delayed consistency chaos tests.
- Tier 9: reason-code and decision envelope contract checks.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (Tier 1/2 subset), Main (Tier 2/4/9), Nightly (Tier 5/6 stress suites).
Acceptance criteria:
- SoD and freeze violations deterministically emit `decision.pass=false` with stable reason codes.
- Digest-only capture policy is enforced; no raw sensitive payload output.
Architecture constraints:
- Data pipeline policy evaluation remains in policy/evaluator layer, not collector transport code.
ADR required: yes
TDD first failing test(s):
- `core/policy/sod/sod_violation_test.go`.
- `internal/integration/datapipeline/enrichment_lag_contract_test.go`.
Cost/perf impact: high
Chaos/failure hypothesis:
- Steady state: enrichment and policy evaluation produce deterministic decision envelopes.
- Fault: delayed query history, missing tags, source outage.
- Expected: explicit fail/degradation reason codes; no implicit success on undecidable paths.
- Abort condition: undecidable high-risk path returns pass.

### Story 2.3: Implement third-party collector protocol and governance-event promotion path
Priority: P1
Tasks:
- Implement external collector protocol (`stdin CollectorConfig`, `stdout JSONL proof records`).
- Add collector process isolation, timeout, deterministic stderr/error mapping.
- Implement governance-event ingestion (stdout/webhook/file JSONL) and promotion to signed proof records.
- Add strict schema rejection for malformed third-party output.
Repo paths:
- `core/collect/plugin/`
- `core/collect/governanceevent/`
- `schemas/v1/governance_event/`
- `internal/e2e/plugin/`
Run commands:
- `axym collect --json`
- `go test ./core/collect/plugin/... -count=1`
- `go test ./internal/e2e/plugin -count=1`
- `make prepush-full`
Test requirements:
- Tier 1: plugin protocol parser/error-mapping units.
- Tier 2: plugin execution and governance-event promotion integration.
- Tier 3: CLI plugin invocation and malformed-output exit behavior.
- Tier 9: plugin contract/schema stability tests.
- Tier 11: AC17 scenario fixture.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (Tier 1/2), Main (Tier 2/3/9/11), Nightly (faulted plugin timing/failure matrix).
Acceptance criteria:
- Malformed plugin output is rejected with clear typed errors and never appended to chain.
- Valid plugin records are indistinguishable from built-in collector records in downstream map/bundle flows.
Architecture constraints:
- Plugin runtime cannot bypass normalization/schema/proof-emission gates.
ADR required: yes
TDD first failing test(s):
- `internal/e2e/plugin/malformed_jsonl_rejected_test.go`.
Cost/perf impact: medium
Chaos/failure hypothesis:
- Steady state: plugin output is validated and promoted deterministically.
- Fault: hung plugin, malformed JSONL, partial output stream.
- Expected: deterministic timeout/error classification, no chain corruption.
- Abort condition: partially invalid plugin stream produces partially appended chain entries.

---

## Epic 3: Sibling Ingestion, Translation, and Chain Stitching

Objective: ingest Wrkr and Gait evidence safely, preserve provenance/relationships, and maintain chain invariants across sessions and re-ingest.
Traceability: FR10, FR4, FR11, AC9, NFR3/NFR4.

### Story 3.1: Implement Wrkr ingest path with privilege-drift analysis and idempotent state
Priority: P0
Tasks:
- Implement Wrkr proof-record ingest path for supported record types.
- Persist ingest state (`.axym/wrkr-last-ingest.json`) and compute privilege drift deltas.
- Map unapproved privilege escalations to governance gaps with stable reason classes.
- Enforce dedupe/idempotent ingest and deterministic ordering.
Repo paths:
- `core/ingest/wrkr/`
- `core/ingest/state/`
- `core/review/privilegedrift/`
- `internal/integration/ingest/wrkr/`
Run commands:
- `axym ingest --source wrkr --json`
- `go test ./core/ingest/wrkr/... -count=1`
- `make prepush-full`
- `make test-hardening`
Test requirements:
- Tier 2: Wrkr ingest integration with stateful delta fixtures.
- Tier 4: privilege-drift acceptance scenarios.
- Tier 5: idempotent re-ingest and state-lock hardening tests.
- Tier 9: ingest reason-code and output-contract stability.
- Tier 12: cross-product proof-chain interoperability checks.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (Tier 2 subset), Main (Tier 2/4/9), Nightly (Tier 5 + Tier 12 matrix).
Acceptance criteria:
- First ingest establishes baseline; subsequent ingest reports deterministic privilege drift.
- Re-ingesting same Wrkr payload does not produce duplicate chain side effects.
Architecture constraints:
- Ingest logic remains separate from collector runtime; drift analysis remains in review/compliance layers.
ADR required: yes
TDD first failing test(s):
- `internal/integration/ingest/wrkr/idempotent_reingest_test.go`.
Cost/perf impact: medium
Chaos/failure hypothesis:
- Steady state: Wrkr ingest preserves sequence and provenance while producing stable drift analysis.
- Fault: repeated inputs, partial state write, conflicting state file lock.
- Expected: deterministic no-dup ingest with explicit lock/retry classification.
- Abort condition: duplicate chain append on repeated ingest.
Semantic invariants:
- Re-ingest of same record set is idempotent.
- Previous-hash linkage remains stable regardless of ingest source mix.
- Drift comparison is deterministic for same baseline and incoming set.

### Story 3.2: Implement Gait pack ingest and native-type translation contracts
Priority: P0
Tasks:
- Implement PackSpec reader (zip, extracted dir, explicit path).
- Translate native Gait types (`trace`, `approval_token`, `delegation_token`) to proof record types.
- Implement compiled-action translation/synthesis and relationship-envelope preservation.
- Ingest passthrough `proof_records.jsonl` when present without translation.
Repo paths:
- `core/ingest/gait/`
- `core/ingest/gait/translate/`
- `core/ingest/gait/pack/`
- `internal/integration/ingest/gait/`
Run commands:
- `axym ingest --source gait --json`
- `go test ./core/ingest/gait/... -count=1`
- `make prepush-full`
- `make test-contracts`
Test requirements:
- Tier 1: translator mapping units per verdict/token type.
- Tier 2: pack reader + translator integration fixtures.
- Tier 4: mixed native+proof passthrough acceptance workflow.
- Tier 9: translation schema compatibility and reason-code contracts.
- Tier 12: gait-proof interoperability suite.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (Tier 1/2), Main (Tier 2/4/9), Nightly (Tier 12 expanded packs).
Acceptance criteria:
- Translation mapping is deterministic and reversible at field-contract level.
- Relationship envelopes and provenance references are preserved/synthesized per contract.
Architecture constraints:
- Translation is mechanical and deterministic; no compliance-opinion branching in translator package.
ADR required: yes
TDD first failing test(s):
- `core/ingest/gait/translate/verdict_mapping_test.go`.
- `internal/integration/ingest/gait/relationship_preservation_test.go`.
Cost/perf impact: high
Chaos/failure hypothesis:
- Steady state: Gait pack ingestion yields deterministic translated proof records.
- Fault: missing pack entry, signature mismatch, unsupported schema version.
- Expected: fail closed with typed dependency/input errors; no partial unsafe append.
- Abort condition: invalid pack content is partially accepted into chain.
Semantic invariants:
- Translation output for identical source artifacts is byte-stable.
- Passthrough proof records remain unchanged except controlled ingestion metadata.
- Chain linkage preserves source ordering guarantees.

### Story 3.3: Implement session-boundary chain stitching and gap signaling
Priority: P1
Tasks:
- Detect session boundaries via checkpoint markers/session IDs/timestamp discontinuities.
- Verify cross-session linkage and emit `CHAIN_SESSION_GAP` with exact missing windows.
- Grade gap-window records per auditability rules and surface in Daily Review inputs.
- Keep deterministic stitching even with interleaved source sessions.
Repo paths:
- `core/ingest/stitch/`
- `core/review/sessiongap/`
- `internal/hardening/stitch/`
Run commands:
- `go test ./core/ingest/stitch/... -count=1`
- `make prepush-full`
- `make test-hardening`
- `make test-chaos`
Test requirements:
- Tier 2: session boundary detection integration tests.
- Tier 5: checkpoint/lock contention hardening.
- Tier 6: crash-resume and missing-window chaos injection.
- Tier 9: gap signal schema/reason-code stability.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (Tier 2 subset), Main (Tier 2/9), Nightly (Tier 5/6 stress).
Acceptance criteria:
- Missing session windows are reported deterministically with exact range metadata.
- Intact multi-session chains verify as continuous sequence.
Architecture constraints:
- Stitching layer must not mutate source records; it emits linkage/gap observations only.
ADR required: yes
TDD first failing test(s):
- `internal/hardening/stitch/session_gap_detection_test.go`.
Cost/perf impact: medium
Chaos/failure hypothesis:
- Steady state: stitched sessions form a verifiable continuous chain.
- Fault: checkpoint loss, crash between sessions, out-of-order replay.
- Expected: explicit gap windows and fail/degrade signaling; no silent continuity claim.
- Abort condition: chain reported intact despite synthetic gap fixture.
Semantic invariants:
- Stitching does not reorder records.
- Gap range computation is deterministic for fixed input timeline.

---

## Epic 4: Compliance Mapping, Gap Detection, and Auditability Grading

Objective: deterministically map evidence to controls, compute coverage/gaps, and produce actionable compliance outputs.
Traceability: FR5, FR7, FR13, AC5/AC13/AC14, NFR3.

### Story 4.1: Implement framework loader, control matcher, and context enricher
Priority: P0
Tasks:
- Load framework definitions from `Clyra-AI/proof/frameworks/*.yaml` with strict schema checks.
- Implement control matcher using record type + required fields + frequency constraints.
- Implement context enricher for `data_class`, `endpoint_class`, `risk_level`, and `discovery_method` weighting.
- Persist deterministic explainability fields for why a record matched/partially matched/failed.
Repo paths:
- `core/compliance/framework/`
- `core/compliance/match/`
- `core/compliance/context/`
- `internal/integration/compliance/match/`
Run commands:
- `axym map --frameworks eu-ai-act,soc2 --json`
- `go test ./core/compliance/... -count=1`
- `make prepush-full`
Test requirements:
- Tier 1: matcher/enricher units and weighting tests.
- Tier 2: framework loader + matcher integration fixtures.
- Tier 4: map workflow acceptance fixtures.
- Tier 9: mapping reason-code and output-shape stability.
- Tier 11: spec scenarios for context-weighted control mapping.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (Tier 1/2), Main (Tier 2/4/9/11), Nightly (extended framework corpus).
Acceptance criteria:
- Matching decisions are deterministic and explainable per record/control pair.
- Context-weight changes are visible in output rationale without breaking schema stability.
Architecture constraints:
- Compliance semantics live in `core/compliance/*`; collectors/ingestors remain opinion-free.
ADR required: yes
TDD first failing test(s):
- `core/compliance/match/context_weighting_test.go`.
Cost/perf impact: medium
Chaos/failure hypothesis:
- Steady state: same records/frameworks produce same control outcomes.
- Fault: invalid framework file or missing required fields.
- Expected: fail closed with typed schema/input errors; no implicit fallback mapping.
- Abort condition: mapping succeeds on invalid framework definition.

### Story 4.2: Implement coverage, gap ranking, remediation planning, and auditability grade engine
Priority: P0
Tasks:
- Implement control coverage status (`covered`, `partial`, `gap`) with deterministic ranking.
- Implement remediation recommendation templates with effort estimates and required evidence fields.
- Implement record-level and bundle-level auditability grade derivation (A-F rules).
- Emit grade/ranking rationale in `--json` and `--explain` outputs.
Repo paths:
- `core/gaps/`
- `core/compliance/coverage/`
- `core/review/grade/`
- `schemas/v1/gaps/`
- `internal/integration/gaps/`
Run commands:
- `axym gaps --json`
- `axym gaps --explain`
- `go test ./core/gaps/... -count=1`
- `make prepush-full`
- `make test-perf`
Test requirements:
- Tier 1: status/ranking/grade rules units.
- Tier 2: coverage/gap integration tests.
- Tier 3: CLI `gaps` behavior and exit semantics.
- Tier 7: ranking/grade throughput benchmarks.
- Tier 9: gap schema + grade reason contract tests.
- Tier 11: gap-remediation scenario fixtures.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (Tier 1/3 subset), Main (Tier 2/3/9/11), Nightly (Tier 7 perf).
Acceptance criteria:
- Coverage and ranking outputs are byte-stable for fixed input corpus.
- Bundle grade equals deterministic weakest-link derivation.
Architecture constraints:
- Gap/ranking and grade logic must be deterministic and separate from bundle rendering concerns.
ADR required: no
TDD first failing test(s):
- `core/gaps/ranking_determinism_test.go`.
- `core/review/grade/weakest_link_test.go`.
Cost/perf impact: high
Chaos/failure hypothesis:
- Steady state: deterministic ranking and grade derivation under nominal corpus size.
- Fault: large corpus and tie-heavy ranking set.
- Expected: stable tie-breakers and bounded runtime under budget.
- Abort condition: non-deterministic ranking order across identical runs.

### Story 4.3: Implement threshold policy and invalid-evidence boundary enforcement for map/gaps flows
Priority: P0
Tasks:
- Enforce framework coverage thresholds from policy config.
- Ensure invalid evidence classes (`invalid_record`, `schema_error`, `mapping_error`) do not count as valid coverage.
- Add deterministic non-zero exit behavior when thresholds/regression boundaries fail.
- Surface typed reason codes and failing controls in machine output.
Repo paths:
- `core/compliance/threshold/`
- `cmd/axym/map.go`
- `cmd/axym/gaps.go`
- `testinfra/contracts/compliance_exit_contract_test.go`
Run commands:
- `axym map --frameworks eu-ai-act --json`
- `axym gaps --json`
- `go test ./... -count=1`
- `make prepush-full`
Test requirements:
- Tier 3: CLI threshold and exit-code tests.
- Tier 4: acceptance fixtures for below-threshold and invalid-evidence scenarios.
- Tier 9: exit-code, reason-code, JSON contract stability.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (Tier 3 subset), Main (Tier 3/4/9), Nightly (invalid-boundary fuzz fixtures).
Acceptance criteria:
- Threshold failures are deterministic and explainable in `--json` output.
- Invalid evidence classes never contribute to control coverage.
Architecture constraints:
- Threshold and invalid-boundary logic remains in compliance layer; no collector-side bypass.
ADR required: no
TDD first failing test(s):
- `testinfra/contracts/invalid_evidence_not_counted_test.go`.
Cost/perf impact: low
Chaos/failure hypothesis:
- Steady state: threshold evaluation correctly reflects validated evidence only.
- Fault: mixed valid/invalid evidence input corpus.
- Expected: fail closed for invalid classes and deterministic threshold result.
- Abort condition: invalid evidence increases measured coverage.

---

## Epic 5: Bundle Assembly, Export, and Bundle Verification Semantics

Objective: produce deterministic, signed, audit-ready bundles and verify both cryptographic and compliance completeness contracts.
Traceability: FR6, FR11, AC2/AC3/AC12/AC16, NFR3/NFR4.

### Story 5.1: Implement deterministic bundle assembler, manifest, and safe output path contracts
Priority: P0
Tasks:
- Implement bundle layout generator with stable ordering and fixed timestamp strategy for deterministic archives.
- Generate `manifest.json`, `chain-verification.yaml`, `auditability-grade.yaml`, `boundary-contract.md`, `retention-matrix.json`.
- Enforce output safety rules (`non-empty + non-managed => fail`, managed marker must be regular file).
- Add signed bundle artifact generation and raw-record inclusion contract.
Repo paths:
- `core/bundle/`
- `core/export/manifest/`
- `core/export/safety/`
- `schemas/v1/bundle/`
- `internal/integration/bundle/`
Run commands:
- `axym bundle --audit Q3-2026 --frameworks eu-ai-act,soc2 --json`
- `go test ./core/bundle/... -count=1`
- `make prepush-full`
- `make test-perf`
Test requirements:
- Tier 2: bundle assembly integration tests.
- Tier 4: audit handoff acceptance fixtures.
- Tier 7: bundle generation performance budget checks.
- Tier 9: bundle manifest/schema/byte-stability contracts.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (Tier 2/9 subset), Main (Tier 2/4/9), Nightly (Tier 7 scale benchmarks).
Acceptance criteria:
- Repeated bundle generation with same inputs yields byte-stable artifact set (except explicit version/time fields where documented).
- Unsafe output paths fail with exit `8` and typed reason.
Architecture constraints:
- Bundle assembly consumes compliance outputs but does not recompute mapping semantics.
ADR required: yes
TDD first failing test(s):
- `core/export/safety/non_managed_output_rejected_test.go`.
- `internal/integration/bundle/byte_stability_test.go`.
Cost/perf impact: high
Chaos/failure hypothesis:
- Steady state: bundle generation is deterministic and safe.
- Fault: symlink marker, pre-populated foreign directory, interrupted write.
- Expected: fail closed with no partial unsafe bundle overwrite.
- Abort condition: bundle writer mutates unmanaged directory.
Semantic invariants:
- Bundle manifest digest set is deterministic for fixed inputs.
- Output safety guards never downgrade to warn-only in default mode.

### Story 5.2: Implement OSCAL export and `verify --bundle` compliance-completeness checks
Priority: P1
Tasks:
- Generate `oscal-v1.1/component-definition.json` from mapped controls and evidence links.
- Implement `axym verify --bundle` compliance completeness checks (required record types, field coverage, grade recomputation).
- Keep cryptographic verification delegation to proof, then layer Axym compliance opinions on top.
- Add executive summary/report contract checks tied to bundle contents.
Repo paths:
- `core/export/oscal/`
- `core/verify/bundle/`
- `cmd/axym/bundle.go`
- `cmd/axym/verify.go`
- `internal/e2e/bundleverify/`
Run commands:
- `axym bundle --audit Q3-2026 --frameworks sox,pci-dss --json`
- `axym verify --bundle ./axym-evidence --json`
- `go test ./... -count=1`
- `make prepush-full`
Test requirements:
- Tier 3: CLI bundle/verify behavior, JSON, and exits.
- Tier 4: AC3/AC12 acceptance fixtures.
- Tier 9: OSCAL schema compatibility and verify-output contract checks.
- Tier 10: release-artefact UAT verifies exported bundle with standalone proof tool.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (Tier 3 subset), Main (Tier 3/4/9), Release (Tier 10 UAT + verify gate).
Acceptance criteria:
- OSCAL export validates against v1.1 schema fixture.
- `axym verify --bundle` reports cryptographic status and compliance completeness in one deterministic envelope.
Architecture constraints:
- Maintain boundary that proof verifies integrity; Axym provides compliance interpretation.
ADR required: no
TDD first failing test(s):
- `internal/e2e/bundleverify/oscal_schema_validation_test.go`.
Cost/perf impact: medium
Chaos/failure hypothesis:
- Steady state: bundle verification combines crypto + completeness deterministically.
- Fault: missing required control evidence or invalid OSCAL node.
- Expected: fail closed with exit `2` or `6` and typed reason surface.
- Abort condition: verify passes with known missing required record classes.

---

## Epic 6: Daily Review, Ticket Attachment, Override, and Replay Certification

Objective: deliver operational compliance cadence with daily exception packs, ticket attach reliability, signed overrides, and replay certification evidence.
Traceability: FR7, FR8, FR9, FR3 replay subsection, AC14/AC15/AC16/AC10.

### Story 6.1: Implement Daily Review Pack engine and export surfaces
Priority: P0
Tasks:
- Aggregate daily exceptions (SoD, approvals, enrichment, attach, replay, freeze, chain-session-gap).
- Emit per-record auditability grades and replay-tier distributions.
- Implement `axym review --date` with `--json`, CSV, and PDF outputs.
- Ensure empty-day behavior returns deterministic successful review with zero exceptions.
Repo paths:
- `core/review/`
- `core/review/export/`
- `cmd/axym/review.go`
- `internal/e2e/review/`
Run commands:
- `axym review --date 2026-09-15 --json`
- `axym review --date 2026-09-15 --format csv`
- `go test ./... -count=1`
- `make prepush-full`
- `make test-hardening`
Test requirements:
- Tier 2: review aggregation integration tests.
- Tier 3: CLI review command contract tests.
- Tier 4: AC14 acceptance fixture.
- Tier 5: retry/backfill hardening tests for delayed inputs.
- Tier 9: review JSON/CSV/PDF shape and reason-code stability.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (Tier 2/3 subset), Main (Tier 2/3/4/9), Nightly (Tier 5 delayed-input resilience).
Acceptance criteria:
- Daily pack includes all required exception classes and grade distributions.
- Empty-day report is deterministic and non-failing.
Architecture constraints:
- Review aggregation reads persisted evidence outputs; it does not mutate source records.
ADR required: no
TDD first failing test(s):
- `internal/e2e/review/empty_day_contract_test.go`.
Cost/perf impact: medium
Chaos/failure hypothesis:
- Steady state: review pack generation succeeds with complete exception classification.
- Fault: missing upstream enrichment/attach feeds for day window.
- Expected: explicit degradation flags in review output; no silent omission.
- Abort condition: missing feed results in falsely complete review pack.

### Story 6.2: Implement Jira/ServiceNow attach bridge with retry, DLQ, and SLA accounting
Priority: P1
Tasks:
- Implement ticket bridges keyed by `change_id` with deterministic payload schema.
- Add retry/backoff policy, rate-limit handling, DLQ persistence, and local backup evidence link.
- Emit attach outcomes to Daily Review and explicit attach status envelopes.
- Enforce attach SLA/SLO counters and deterministic reporting windows.
Repo paths:
- `core/ticket/`
- `core/ticket/jira/`
- `core/ticket/servicenow/`
- `core/ticket/dlq/`
- `internal/hardening/ticket/`
Run commands:
- `axym collect --json`
- `go test ./core/ticket/... -count=1`
- `make prepush-full`
- `make test-hardening`
- `make test-chaos`
Test requirements:
- Tier 2: attach bridge integration tests with canned API fixtures.
- Tier 4: AC15 acceptance workflow fixture.
- Tier 5: retry/DLQ hardening tests.
- Tier 6: 429/5xx chaos storm and recovery scenarios.
- Tier 9: attach status/reason-code contract tests.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (Tier 2 subset), Main (Tier 2/4/9), Nightly (Tier 5/6 storm matrix).
Acceptance criteria:
- 429 retry path succeeds deterministically when fixture recovers.
- Sustained failure routes to DLQ and appears in Daily Review with typed status.
Architecture constraints:
- Ticket adapters remain external-boundary modules; core evidence chain semantics stay local and deterministic.
ADR required: yes
TDD first failing test(s):
- `internal/hardening/ticket/rate_limit_retry_test.go`.
- `internal/hardening/ticket/dlq_visibility_test.go`.
Cost/perf impact: medium
Chaos/failure hypothesis:
- Steady state: attach within SLA/SLO with deterministic status accounting.
- Fault: sustained 5xx, repeated 429, partial network outage.
- Expected: bounded retries, DLQ fallback, explicit review visibility.
- Abort condition: failed attachment not accounted for in review outputs.

### Story 6.3: Implement signed overrides and replay certification flows
Priority: P1
Tasks:
- Implement `axym override create` signed artifact flow (scope, signer, expiry, reason).
- Ensure override artifacts are append-only and included in bundles.
- Implement replay engine tiers A/B/C and `replay_certification` proof record emission.
- Support replay blast-radius summary fields and deterministic tier classification.
Repo paths:
- `core/override/`
- `core/replay/`
- `cmd/axym/override.go`
- `cmd/axym/replay.go`
- `internal/integration/override_replay/`
Run commands:
- `axym override create --bundle Q3-2026 --reason "fixture" --signer ops-key --json`
- `axym replay --model payments-agent --tier A --json`
- `go test ./... -count=1`
- `make prepush-full`
- `make test-hardening`
Test requirements:
- Tier 2: override/replay integration tests.
- Tier 3: CLI override/replay contract tests.
- Tier 4: AC16 and replay workflow acceptance fixtures.
- Tier 5: append-only and replay-write crash-safety tests.
- Tier 9: override/replay schema and exit contract checks.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (Tier 2/3 subset), Main (Tier 2/3/4/9), Nightly (Tier 5 resilience + replay corpus).
Acceptance criteria:
- Override deletion/tamper is detected by chain verification.
- Replay certification outputs include deterministic tier and blast-radius summary fields.
Architecture constraints:
- Override and replay emit evidence; they do not bypass proof emission/sign/chain layers.
ADR required: yes
TDD first failing test(s):
- `internal/integration/override_replay/override_append_only_test.go`.
- `core/replay/tier_classification_test.go`.
Cost/perf impact: high
Chaos/failure hypothesis:
- Steady state: overrides and replay certs are signed and chain-linked.
- Fault: attempted override removal or replay artifact mismatch.
- Expected: verification detects tamper; replay emits explicit failure classification.
- Abort condition: override removal not detected by verify path.
Semantic invariants:
- Override history is append-only.
- Replay tier classification is deterministic for fixed fixture inputs.

---

## Epic 7: CLI Surface Contracts and Regression Enforcement

Objective: deliver complete command surface with stable JSON/exits and deterministic regression gates for CI.
Traceability: FR11, FR13, AC1/AC4/AC13, dev guide exit/CLI contract sections.

### Story 7.1: Implement root CLI contract and shared output envelope
Priority: P0
Tasks:
- Implement root command scaffolding and shared output envelope types.
- Enforce `--json`, `--quiet`, `--explain` behavior contracts across commands.
- Implement stable error envelope with reason-class and exit-code mapping.
- Add command help/usage parity tests.
Repo paths:
- `cmd/axym/root.go`
- `cmd/axym/output.go`
- `internal/e2e/cli/`
- `testinfra/contracts/cli_output_contract_test.go`
Run commands:
- `axym --help`
- `axym collect --dry-run --json`
- `go test ./... -count=1`
Test requirements:
- Tier 3: CLI help/flag/exit behavior tests.
- Tier 9: JSON shape and exit-contract compatibility tests.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform.
- Pipeline placement: PR (Tier 3 subset), Main (full Tier 3/9 across OS matrix).
Acceptance criteria:
- All commands support `--json`; major commands support `--explain`; CI flows honor `--quiet`.
- Exit code contract remains stable for baseline fixture corpus.
Architecture constraints:
- Output contract is centralized; command handlers must not diverge envelope semantics.
ADR required: no
TDD first failing test(s):
- `testinfra/contracts/cli_output_contract_test.go`.
Cost/perf impact: low
Chaos/failure hypothesis:
- Not risk-bearing beyond contract validation.

### Story 7.2: Implement complete command surface and policy/config initialization flows
Priority: P0
Tasks:
- Implement `init`, `collect`, `map`, `gaps`, `bundle`, `review`, `verify`, `record add`, `override create`, `ingest`, `replay` command handlers.
- Implement `axym-policy.yaml` read/validate/apply path with deterministic defaults.
- Add docs parity checks for command/flag surfaces.
- Add machine-readable rationale fields for major decision paths.
Repo paths:
- `cmd/axym/init.go`
- `cmd/axym/collect.go`
- `cmd/axym/map.go`
- `cmd/axym/gaps.go`
- `cmd/axym/bundle.go`
- `cmd/axym/review.go`
- `cmd/axym/record.go`
- `cmd/axym/ingest.go`
- `core/config/`
- `schemas/v1/config/`
Run commands:
- `axym init --json`
- `axym collect --dry-run --json`
- `axym verify --chain --json`
- `go test ./... -count=1`
- `make prepush-full`
Test requirements:
- Tier 3: command-level e2e contracts (`--json`, help, exits).
- Tier 4: operator acceptance scripts across primary command paths.
- Tier 9: command/flag schema and docs-parity contract checks.
- Tier 11: scenario fixtures for AC1 flow.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (Tier 3 subset), Main (Tier 3/4/9/11), Nightly (expanded command matrix).
Acceptance criteria:
- Primary command surface is complete and stable per PRD.
- Config validation fails closed with exit `6` on invalid policy/schema input.
Architecture constraints:
- Command handlers orchestrate layers; no cross-layer business logic collapse.
ADR required: yes
TDD first failing test(s):
- `internal/e2e/cli/command_surface_contract_test.go`.
Cost/perf impact: medium
Chaos/failure hypothesis:
- Steady state: commands return deterministic envelopes for valid/invalid inputs.
- Fault: malformed policy, missing dependencies, invalid flags.
- Expected: typed fail-closed exits (`6`/`7`) with stable reason classes.
- Abort condition: invalid configuration silently falls back to permissive defaults.

### Story 7.3: Implement compliance regression baseline/init/run contracts
Priority: P0
Tasks:
- Implement `axym regress init --baseline <bundle-path>` baseline capture.
- Implement `axym regress run --baseline <path> --frameworks <list>` drift detection.
- Emit deterministic regressed-control details and exit `5` on drift.
- Add portable regression fixture format for CI/local parity.
Repo paths:
- `core/regress/`
- `cmd/axym/regress.go`
- `schemas/v1/regress/`
- `internal/integration/regress/`
Run commands:
- `axym regress init --baseline ./fixtures/bundles/passing --json`
- `axym regress run --baseline <baseline-path> --frameworks eu-ai-act,soc2 --json`
- `go test ./... -count=1`
- `make prepush-full`
- `make test-hardening`
- `make test-chaos`
Test requirements:
- Tier 2: baseline/read/write integration tests.
- Tier 3: regress CLI behavior and exit-code tests.
- Tier 4: AC13 regression acceptance fixtures.
- Tier 5: concurrent baseline access and atomic update hardening tests.
- Tier 6: corrupt/missing baseline chaos tests.
- Tier 9: regress schema + exit `5` contract stability.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (Tier 2/3 subset), Main (Tier 2/3/4/9), Nightly (Tier 5/6 drift stress).
Acceptance criteria:
- Drifted controls are listed deterministically with stable reason fields.
- Same baseline + same evidence set always returns same pass/fail result.
Architecture constraints:
- Regression engine consumes map/gap outputs; it does not re-implement mapping semantics.
ADR required: no
TDD first failing test(s):
- `internal/integration/regress/exit5_on_drift_test.go`.
Cost/perf impact: medium
Chaos/failure hypothesis:
- Steady state: regress run detects only real coverage drift.
- Fault: baseline file corruption, missing framework entries, partial baseline write.
- Expected: deterministic fail-closed behavior with typed reason classes.
- Abort condition: drift exists but command exits `0`.
Semantic invariants:
- Regression result is deterministic for fixed baseline + evidence set.
- Baseline updates preserve versioned schema compatibility.

---

## Epic 8: Acceptance Closure, Reliability Hardening, and Release Readiness

Objective: close AC coverage with scenario-driven validation and enforce resilience/performance/release integrity gates before v1.0 cut.
Traceability: AC1-AC17, NFR2-NFR5, dev/architecture gate contracts.

### Story 8.1: Build scenario and acceptance harness covering AC1-AC17 and cross-product flows
Priority: P0
Tasks:
- Create scenario fixtures under `scenarios/axym/**` for each acceptance criterion.
- Build acceptance runner scripts and deterministic golden-output verification.
- Add cross-product scenarios for Wrkr/Gait ingest + unified chain verification.
- Ensure scenario fixture updates are reviewed as contract changes.
Repo paths:
- `scenarios/axym/`
- `internal/scenarios/`
- `scripts/validate_scenarios.sh`
- `testinfra/acceptance/`
Run commands:
- `go test ./internal/scenarios -count=1 -tags=scenario`
- `make test-scenarios`
- `axym verify --chain --json`
Test requirements:
- Tier 4: end-to-end acceptance suites for AC1-AC17.
- Tier 11: scenario fixtures as external behavior specs.
- Tier 12: cross-product chain acceptance fixtures.
Matrix wiring:
- Lanes: Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: Main (Tier 4/11 baseline), Nightly (expanded scenario corpus), Release (Tier 4/12 must pass).
Acceptance criteria:
- Every AC has at least one deterministic acceptance fixture and pass/fail command.
- Cross-product fixtures verify mixed chain behavior with proof parity.
Architecture constraints:
- Scenarios validate boundaries as black-box behavior; no test-only boundary bypass.
ADR required: no
TDD first failing test(s):
- `internal/scenarios/ac13_regression_spec_test.go` and corresponding AC fixture tests.
Cost/perf impact: medium
Chaos/failure hypothesis:
- Steady state: acceptance harness catches behavioral regressions pre-merge.
- Fault: fixture drift or nondeterministic output order.
- Expected: deterministic test failure requiring explicit fixture review.
- Abort condition: scenario suite passes despite known AC regression.

### Story 8.2: Add hardening/chaos/perf/soak suites and finalize docs/release gate parity
Priority: P0
Tasks:
- Implement Tier 5/6/7/8 suites for lock contention, sink outages, retry storms, performance budgets, and soak stability.
- Add docs parity checks for command/flag/exit-code surfaces and operator workflows.
- Finalize release gate scripts for UAT install paths (source, release binary, Homebrew).
- Wire final release go/no-go checks including signing/provenance verification commands.
Repo paths:
- `internal/hardening/`
- `internal/chaos/`
- `perf/bench_baseline.json`
- `perf/runtime_slo_budgets.json`
- `perf/resource_budgets.json`
- `docs/commands/`
- `scripts/release_go_nogo.sh`
Run commands:
- `make test-hardening`
- `make test-chaos`
- `make test-perf`
- `go test ./... -count=1`
- `sha256sum -c dist/checksums.txt`
- `cosign verify-blob --certificate dist/checksums.txt.pem --signature dist/checksums.txt.sig dist/checksums.txt`
- `make prepush-full`
Test requirements:
- Tier 5: resilience and atomicity contract suite.
- Tier 6: controlled chaos suite.
- Tier 7: performance budget suite.
- Tier 8: soak/stability suite.
- Tier 9: docs/CLI parity and contract stability checks.
- Tier 10: install-path UAT suites.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Pipeline placement: PR (fast contract subset), Main (core + docs parity), Nightly (Tier 5/6/7/8), Release (Tier 9/10 + signing/provenance gates).
Acceptance criteria:
- Budget regressions and chaos failures are merge/release blocking where lanes are required.
- Release gate verifies checksums, signatures, SBOM, and provenance before publish.
Architecture constraints:
- Hardening/chaos coverage must target real boundary behavior, not mocks that bypass failure classes.
ADR required: no
TDD first failing test(s):
- `internal/hardening/sink_unavailable_fail_closed_test.go`.
- `internal/chaos/retry_storm_classification_test.go`.
Cost/perf impact: high
Chaos/failure hypothesis:
- Steady state: system remains deterministic and fail-closed under bounded failures.
- Fault: injected sink outages, lock starvation, external API storms, perf-budget pressure.
- Expected: explicit failure classes, bounded retries, no silent evidence loss.
- Abort condition: resilience lane green while known failure class is untested.

---

## Minimum-Now Sequence

Phase 0 (Week 1): foundation bootstrap

- Deliver Stories `0.1`, `0.2`, `0.3`.
- Outcome: buildable scaffold, pinned lanes, CI/release/security guardrails.

Phase 1 (Week 2-3): proof core first

- Deliver Stories `1.1`, `1.2`, `1.3`.
- Outcome: deterministic record->sign->append->verify core with fail-closed persistence semantics.

Phase 2 (Week 4-5): collector breadth and plugin adoption path

- Deliver Stories `2.1`, `2.2`, `2.3`.
- Outcome: required evidence surfaces collecting deterministic proof records including data-pipeline semantics.

Phase 3 (Week 6-7): sibling ingestion and chain continuity

- Deliver Stories `3.1`, `3.2`, `3.3`.
- Outcome: Wrkr/Gait ingest, translation, and cross-session stitching with idempotent chain invariants.

Phase 4 (Week 8-9): compliance meaning and audit outputs

- Deliver Stories `4.1`, `4.2`, `4.3`, `5.1`, `5.2`.
- Outcome: deterministic map/gaps/grade + signed bundle/oscal/verify completeness.

Phase 5 (Week 10-11): operations and governance loops

- Deliver Stories `6.1`, `6.2`, `6.3`, `7.1`, `7.2`, `7.3`.
- Outcome: daily review, ticket attach, override/replay, full CLI surface, and regression gates.

Phase 6 (Week 12): acceptance closure and release readiness

- Deliver Stories `8.1`, `8.2`.
- Outcome: AC1-AC17 coverage complete, hardening/chaos/perf/soak green, release gate ready for v1.0 tag.

Dependency notes:

- Epic 1 depends on Epic 0.
- Epics 2 and 3 depend on Epic 1.
- Epic 4 depends on Epics 1-3.
- Epic 5 depends on Epic 4 and verify core from Epic 1.
- Epic 6 depends on Epics 2-5.
- Epic 7 depends on all prior runtime epics.
- Epic 8 depends on all implementation epics.

---

## Explicit Non-Goals

- No Wrkr or Gait product-feature implementation beyond Axym ingestion/translation contracts.
- No hosted-only or dashboard-first dependencies in v1.0 core.
- No non-deterministic/LLM-based compliance inference in default runtime paths.
- No evidence payload exfiltration or raw-secret extraction.
- No replacement of `Clyra-AI/proof` record semantics, chain model, or signing contracts.
- No scope expansion into legal-opinion automation or GRC workflow ownership systems.
- No speculative architecture collapse across required boundaries.

---

## Definition of Done

- Every story acceptance criterion is automated and passing in all required lanes.
- CLI contract stability (`--json`, exits, reason codes, help surfaces) is enforced by tests.
- Determinism contracts are green for records, maps, gaps, bundles, and regress results.
- Fail-closed behavior is verified for high-risk ambiguous and dependency-failure paths.
- Schema contracts are versioned and validated with valid/invalid fixtures.
- Ingestion, dedupe, chain-linkage, and regression semantic invariants are covered and passing.
- Cross-product interoperability suites pass for Wrkr/Gait/proof integration points.
- Performance budgets are met or explicitly exception-approved with mitigation.
- Release integrity pipeline produces verified signed artifacts with SBOM and provenance.
- Docs and CLI parity checks pass for all user-visible command/flag/exit behavior.
