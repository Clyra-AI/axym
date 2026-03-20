# PLAN Axym: Proof Integrity and Ingress Contract Hardening

Date: 2026-03-20
Source of truth: user-provided 2026-03-20 code-review findings and fix-wave guidance, `product/axym.md`, `product/dev_guides.md`, `product/architecture_guides.md`, `AGENTS.md`
Scope: Axym OSS CLI only. Convert the current review blockers into an execution-ready remediation backlog covering proof integrity verification, fail-closed proof ingress validation, collector relationship-envelope preservation, machine-readable CLI contract cleanup, cancellation propagation, and required docs/ADR synchronization. No new product-scope expansion beyond Axym's existing CLI and `Clyra-AI/proof` interoperability.

---

## Global Decisions (Locked)

- This plan supersedes the older `PLAN_NEXT.md` theme. The new scope is release-blocker remediation, not broader product reframing work.
- Contract/runtime correctness must land before docs, onboarding, or distribution-language changes.
- Axym record validity is a three-part contract for Axym-authored evidence: proof-schema-valid record payload, valid record signature, and valid append-only chain linkage.
- Fail-closed enforcement points are mandatory at the append boundary, verify boundary, bundle build/verify boundary, and collector/runtime boundary.
- `record add` must not be the only validation gate. The shared append path must reject schema-invalid proof payloads so future callers cannot bypass CLI-only checks.
- The collector/plugin protocol remains adapter-first. `collect --plugin` must not accept arbitrary pre-signed `proof.Record` payloads as a bypass around normalization, redaction, or policy enforcement.
- Relationship-envelope preservation is required across supported collector ingress. If Axym accepts a relationship-bearing collector payload, it must preserve `parent_ref`, `entity_refs`, `policy_ref`, `agent_chain`, `edges`, and additive relationship extras instead of silently flattening them away.
- Bundle verification must stay offline and deterministic. If per-record signature verification needs exported public-key material, Axym must ship a versioned bundle artifact for that purpose rather than relying on any network lookup.
- Public contract changes must be additive wherever possible. Exit code vocabulary remains `0,1,2,3,4,5,6,7,8`.
- Verification-only JSON fields such as `break_index` and `break_point` must never appear on non-verification errors.
- Long-running CLI paths must propagate cancellation and timeout state from the process boundary through `collect`, `ingest`, and plugin execution.
- Public docs are source-of-truth for supported runtime behavior. They must match shipped plugin protocol shape, verify semantics, and manual proof-ingress guarantees exactly.

---

## Current Baseline (Observed)

- `go build ./cmd/axym` and `go test ./... -count=1` currently pass, including internal integration and e2e lanes.
- `core/verify/verify.go` and `core/verify/bundle/verify.go` delegate to `proof.VerifyChain()` only, so record-signature tampering is not detected by `verify --chain` or `verify --bundle`.
- `core/bundle/bundle.go` gates bundle creation on the same chain-only verification path, so Axym can ship bundles containing records with invalid signatures.
- `cmd/axym/record.go` performs shallow field checks only, then calls `store.Append`; `core/store/store.go` signs and appends without a full `proof.ValidateRecord` gate at the shared append boundary.
- `core/collector/types.go` has no `Relationship` field on `Candidate`, `core/collect/plugin/collector.go` has no `relationship` parser, and `core/collect/runner.go` never forwards relationship data into normalization.
- `cmd/axym/output.go` serializes `break_index` unconditionally, so non-verify failures surface a phantom `break_index: 0`.
- `cmd/axym/collect.go`, `cmd/axym/ingest.go`, `cmd/axym/root.go`, and `cmd/axym/main.go` use `context.Background()` or `Execute()` instead of a signal-aware command context.
- Existing green areas remain strong: Wrkr ingest already rejects unsupported proof record types, Gait native translation preserves relationships within the ingest path, and map/gaps/regress/review logic already has healthy deterministic coverage.
- Current public docs drift from shipped behavior in two important places: plugin collector docs still describe raw `[]proof.Record` emission, and verification docs imply signed/tamper-evident guarantees stronger than the runtime currently enforces.

---

## Exit Criteria

1. `axym verify --chain --json` fails with exit `2` and stable verification reason codes when any stored record signature is missing or invalid, even if chain hashes remain intact.
2. `axym bundle --json` and `axym verify --bundle --json` fail closed on invalid or unverifiable record signatures for Axym-authored bundles without any network dependency.
3. Axym emits or embeds the deterministic public-key material needed for offline bundle record-signature verification through an additive, versioned artifact path.
4. `axym record add --json` rejects unknown record types and type-schema-invalid proof payloads deterministically, and rejected payloads do not mutate chain count or head hash.
5. Manual proof ingress uses one authoritative validation gate that future callers cannot bypass accidentally.
6. `axym collect --json --plugin "<cmd>"` preserves supported relationship-envelope data end-to-end instead of silently flattening it away.
7. Non-verify JSON errors omit verification breakpoint fields; verify failures continue to expose breakpoint data when applicable.
8. `collect`, `ingest`, and plugin execution honor process cancellation and configured timeouts end-to-end without ambiguous partial behavior.
9. Contract, acceptance, and release-risk lanes include signature-tamper, schema-invalid manual append, relationship round-trip, and JSON envelope regressions as blocking checks.
10. `product/axym.md`, `README.md`, `docs/`, and `docs-site/public/llm/*.md` describe shipped verification, plugin, and manual-ingest behavior truthfully and consistently.

---

## Recommendation Traceability

| Recommendation | Why | Strategic direction | Expected moat/benefit | Planned stories |
|---|---|---|---|---|
| Verify record signatures, not just chain hashes | Invalid signatures currently pass `verify --chain`, `verify --bundle`, and bundle build | Restore cryptographic record-integrity enforcement at the verification boundary | Re-establishes the signed/tamper-evident proof guarantee and auditor trust | `W1-S1`, `W3-S1` |
| Reject schema-invalid manual records before append | Unsupported or malformed proof payloads can currently enter the canonical chain through `record add` | Make proof ingress fail closed at the authoritative append boundary | Prevents poisoned chains and future bypasses from new callers | `W1-S2`, `W3-S1` |
| Preserve full relationship envelopes through the collector/plugin boundary | Plugin-collected provenance is silently flattened today | Keep Axym's collector path truthful to its published relationship-preservation claim | Stronger provenance continuity and better sibling-product alignment | `W2-S1`, `W3-S1` |
| Remove phantom verification fields from generic JSON errors | Machine-readable consumers currently see misleading `break_index: 0` on unrelated errors | Tighten CLI JSON envelope stability and semantics | Cleaner CI/wrapper integrations and lower contract ambiguity | `W2-S2` |
| Propagate cancellation/timeouts through the CLI runtime boundary | Long-running collect/ingest flows still ignore signal-aware command context | Make long-running paths bounded, interruptible, and deterministic | Safer CI automation and better operational reliability | `W2-S2` |
| Backstop the blocker classes in required lanes | The repo-wide suite was green while these holes remained open | Turn review-discovered failure classes into permanent gating tests | Prevents silent regression and improves release confidence | `W1-S1`, `W1-S2`, `W2-S1`, `W2-S2`, `W3-S1` |

---

## Test Matrix Wiring

Lane definitions:

- Fast lane: `make lint-fast`, targeted package/unit tests, and focused contract tests under `testinfra/contracts` for the story's public surface.
- Core CI lane: `make test-fast`, targeted `internal/integration/...` and `internal/e2e/...` suites, `make test-contracts`, and `make test-adapter-parity` when adapter boundaries change.
- Acceptance lane: `go test ./testinfra/acceptance/... -count=1`, affected `internal/e2e/...` suites, and `make test-scenarios` when user-visible workflows or docs storylines change.
- Cross-platform lane: `.github/workflows/pr.yml` and `.github/workflows/main.yml` matrix coverage on Linux, macOS, and Windows for stories that affect CLI contract shape, path behavior, plugin execution, or bundle layout.
- Risk lane: `make prepush-full`, `make codeql`, targeted hardening/chaos coverage, and any release-go/no-go checks affected by the story.

Story-to-lane map:

| Story | Fast | Core CI | Acceptance | Cross-platform | Risk |
|---|---|---|---|---|---|
| W1-S1 | Yes | Yes | Yes | Yes | Yes |
| W1-S2 | Yes | Yes | Yes | Yes | Yes |
| W2-S1 | Yes | Yes | Yes | Yes | Yes |
| W2-S2 | Yes | Yes | Yes | Yes | Yes |
| W3-S1 | Yes | Yes | Yes | No | No |

Merge/release gating rule:

- Wave `W1` and `W2` stories are merge-blocking runtime work. `W3` docs work must not merge ahead of them.
- Any story marked `Risk: Yes` is release-blocking until its Fast, Core CI, Acceptance, Cross-platform, and Risk lanes are green.
- Any story that changes CLI JSON envelopes, bundle artifacts, or verification semantics must pass contract tests and cross-platform CLI smoke in the same merge window.
- Public docs updates are incomplete until `make test-docs-consistency`, `make test-docs-storyline`, `make test-docs-links`, and the relevant docs parity contract tests are green.

---

## Epic W1: Proof Integrity and Fail-Closed Manual Ingress

Objective: make record integrity and manual proof ingress trustworthy before touching richer provenance or launch-facing docs.

### Story W1-S1: Add signature-aware chain and bundle verification
Priority: P0
Tasks:
- Add deterministic per-record signature verification to the chain verification path, using Axym-managed public-key material instead of hash-link checks alone.
- Introduce an additive, versioned bundle artifact for exported record-signing public keys so `verify --bundle` can validate record signatures offline.
- Make `bundle` fail closed when source-chain signatures are missing or invalid, not just when previous hashes are broken.
- Extend machine-readable verification output and stable reason codes to distinguish record-signature failure from chain-link failure without changing exit code vocabulary.
- Refresh bundle fixtures and contract tests so signature tamper is release-blocking.
Repo paths:
- `core/verify/verify.go`
- `core/verify/bundle/verify.go`
- `core/bundle/bundle.go`
- `core/store/store.go`
- `cmd/axym/verify.go`
- `cmd/axym/verify_test.go`
- `cmd/axym/bundle_test.go`
- `core/verify/verify_test.go`
- `core/verify/bundle/verify_test.go`
- `internal/e2e/verify/chain_breakpoint_test.go`
- `internal/e2e/bundleverify/oscal_schema_validation_test.go`
- `internal/integration/bundle/context_engineering_bundle_test.go`
- `testinfra/contracts/verify_contract_test.go`
- `testinfra/contracts/bundle_contract_test.go`
- `schemas/v1/bundle/schema.go`
- `schemas/v1/bundle/executive-summary-v1.schema.json`
- `fixtures/bundles/good/`
Run commands:
- `go build -o ./axym ./cmd/axym`
- `go test ./core/verify ./core/verify/bundle ./core/bundle ./cmd/axym -count=1`
- `go test ./internal/e2e/verify ./internal/e2e/bundleverify ./internal/integration/bundle -count=1`
- `go test ./testinfra/contracts -run 'Verify|Bundle' -count=1`
- `./axym collect --fixture-dir fixtures/collectors --store-dir ./.axym-sig --json`
- `./axym verify --chain --store-dir ./.axym-sig --json`
- `./axym bundle --audit sig-test --store-dir ./.axym-sig --output ./axym-evidence --json`
- `./axym verify --bundle ./axym-evidence --json`
Test requirements:
- Determinism/hash/sign/packaging changes: byte-stability repeat-run checks, canonicalization/digest checks, verify determinism checks, and `make test-contracts`.
- Schema/artifact changes: validation tests and fixture/golden updates for any additive bundle signing-key artifact.
- CLI behavior changes: `--json` stability and exit-code contract tests for `verify` and `bundle`.
- Gate/policy/fail-closed changes: explicit signature-tamper fail-closed tests.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Required gates: `make test-contracts`, targeted verify/bundle suites, `make prepush-full`, and release go/no-go checks if bundle artifact layout changes.
Acceptance criteria:
- A record-signature corruption that leaves chain hashes intact still causes `verify --chain` to fail with exit `2` and a stable verification reason.
- `bundle` and `verify --bundle` fail closed on invalid or unverifiable record signatures for Axym-authored bundles.
- Offline bundle verification uses additive bundle key material only and does not introduce network lookups.
- Repeat bundle builds with identical inputs remain deterministic after the new artifact is added.
Stable/internal boundary notes:
- Public: `verify --chain`, `verify --bundle`, JSON reason codes, and bundle artifact layout are contract surfaces.
- Internal: helper/package placement for key loading and signature iteration is implementation detail.
Migration expectations:
- Additive only. The top-level proof-record format and exit code vocabulary must not break.
- Bundle fixtures and docs must be refreshed in the same backlog so new verification semantics are visible immediately.
Integration hooks:
- CI and release automation using `axym verify --chain --json`, `axym bundle --json`, and `axym verify --bundle --json`.
- Auditor/operator workflows consuming portable bundle artifacts for offline validation.
Dependencies:
- None.
Risks:
- If key-export semantics are left implicit, bundle verification will remain partially unverifiable and the contract hole will persist under a new name.

### Story W1-S2: Enforce full proof-schema validation on manual append
Priority: P0
Tasks:
- Move full proof-record validation into the authoritative append boundary used by `record add`, so unknown record types and type-schema-invalid payloads fail before signing or chain mutation.
- Keep path/read/decode and missing-input failures separate from schema/record-type failures in the CLI error taxonomy.
- Preserve deterministic dedupe behavior for valid manual payloads while ensuring rejected payloads do not mutate chain count, head hash, or dedupe state.
- Add contract tests proving that invalid manual payloads cannot be legitimized later by `verify --chain`, `bundle`, or downstream mapping workflows.
- Refresh sample/manual proof fixtures if any currently rely on lenient ingress behavior.
Repo paths:
- `cmd/axym/record.go`
- `cmd/axym/record_test.go`
- `core/store/store.go`
- `core/store/atomic_test.go`
- `internal/integration/record/normalize_validate_test.go`
- `testinfra/contracts/record_schema_contract_test.go`
- `testinfra/contracts/cli_output_contract_test.go`
- `testinfra/contracts/verify_contract_test.go`
- `fixtures/records/decision.json`
Run commands:
- `go build -o ./axym ./cmd/axym`
- `go test ./cmd/axym ./core/store ./internal/integration/record -count=1`
- `go test ./testinfra/contracts -run 'Record|CLIOutput|Verify' -count=1`
- `go test ./testinfra/acceptance/... -count=1`
- `./axym record add --input fixtures/records/decision.json --json`
- `./axym verify --chain --json`
Test requirements:
- Schema/artifact changes: proof-schema validation tests and fixture updates where needed.
- CLI behavior changes: `record add --json` envelope and exit-code contract tests.
- Gate/policy/fail-closed changes: rejected payloads must not mutate append-only state.
- Acceptance checks: operator/manual proof path remains usable with valid payloads.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Required gates: targeted `record`/`store` suites, contract tests, and acceptance smoke on the manual proof path.
Acceptance criteria:
- `record add` rejects unknown record types and typed-schema-invalid payloads deterministically before signing or appending.
- Invalid manual payloads do not change chain count, head hash, or later verification results.
- Valid manual payloads continue to append and dedupe deterministically.
- Machine-readable errors distinguish invalid file/input issues from schema/record-type violations.
Stable/internal boundary notes:
- Public: `record add --json`, exit behavior, and accepted proof payload semantics are contract surfaces.
- Internal: whether validation sits in `cmd/axym/record.go`, `core/store`, or a shared helper is internal as long as future callers cannot bypass it.
Migration expectations:
- Stricter validation may reject previously accepted invalid payloads. That is an intentional fail-closed correction, not a relaxed compatibility surface.
- Docs and quickstart/manual examples must be updated in `W3-S1`.
Integration hooks:
- Operator and CI workflows using `axym record add --input <record.json> --json` for approvals, risk assessments, and other manual proof records.
- Sample-pack and quickstart flows that rely on deterministic manual record append.
Dependencies:
- None.
Risks:
- CLI-only validation is insufficient; this story is incomplete unless the shared append path is safe for future callers too.

---

## Epic W2: Collector Boundary Preservation and CLI Runtime Contract Cleanup

Objective: preserve supported provenance at the collect boundary and remove remaining machine-readable/runtime contract drift after proof integrity is hardened.

### Story W2-S1: Preserve relationship envelopes through the collector/plugin path
Priority: P1
Tasks:
- Extend the collector candidate model to carry `*proof.Relationship` without collapsing the adapter-first boundary.
- Extend plugin output parsing to accept relationship-envelope data while still rejecting malformed or bypass-style payloads that try to skip normalization.
- Thread candidate relationship data through `core/collect/runner` into normalization/proof emission so stored records preserve `parent_ref`, `entity_refs`, `policy_ref`, `agent_chain`, `edges`, and additive extras when present.
- Add round-trip tests for plugin-collected relationship envelopes and re-run mixed-source integration coverage to ensure no regressions in existing governance/Wrkr/Gait paths.
- Keep the normalized identity view additive on top of preserved relationship data; do not replace the relationship envelope with Axym-only flattened fields.
Repo paths:
- `core/collector/types.go`
- `core/collect/plugin/collector.go`
- `core/collect/plugin/collector_test.go`
- `core/collect/runner.go`
- `core/normalize/identity.go`
- `core/record/builder.go`
- `cmd/axym/collect_test.go`
- `internal/e2e/plugin/empty_metadata_roundtrip_test.go`
- `internal/integration/collect/multi_source_fixture_test.go`
- `testinfra/contracts/collect_contract_test.go`
Run commands:
- `go build -o ./axym ./cmd/axym`
- `go test ./core/collect/... ./core/collector ./core/normalize ./core/record ./cmd/axym -count=1`
- `go test ./internal/e2e/plugin ./internal/integration/collect -count=1`
- `go test ./testinfra/contracts -run 'Collect' -count=1`
- `./axym collect --json --plugin "<cmd>"`
- `./axym verify --chain --json`
Test requirements:
- SDK/adapter boundary changes: plugin protocol conformance tests and adapter parity coverage.
- CLI behavior changes: collect JSON summary stability with relationship-bearing plugin inputs.
- Scenario/context changes: round-trip provenance preservation fixtures.
- Determinism checks: repeated collection with identical plugin output yields stable stored relationship content.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Required gates: `make test-adapter-parity`, plugin e2e, collect integration, contract tests, and release-risk suites.
Acceptance criteria:
- A plugin that emits a valid relationship envelope stores that envelope without silent loss.
- Existing plugins that do not emit `relationship` remain compatible and deterministic.
- Malformed relationship payloads fail closed with typed collector errors rather than degrading into partially preserved provenance.
- Axym still uses normalized candidate flow; `collect --plugin` does not become a raw proof-record append backdoor.
Stable/internal boundary notes:
- Public: plugin collector protocol fields and stored relationship semantics are supported contract surfaces.
- Internal: candidate struct shape and helper placement remain implementation detail.
Migration expectations:
- Additive only. Existing plugins remain valid; relationship-bearing plugins gain a supported preservation path after this story.
- Public protocol docs must be updated in `W3-S1` to match the shipped shape exactly.
Integration hooks:
- Plugin author workflows built around `axym collect --json --plugin "<cmd>"`.
- CI/operator collection flows that depend on plugin-provided provenance surviving into the proof chain.
Dependencies:
- `W1-S2`
Risks:
- If relationship fields are only partially threaded, Axym will keep claiming provenance preservation while continuing to drop critical graph edges.

### Story W2-S2: Clean up JSON error envelopes and propagate signal-aware cancellation
Priority: P2
Tasks:
- Make verification breakpoint fields verification-only in the JSON envelope, using `omitempty` or a verify-specific error payload shape.
- Thread signal-aware context from `main` to root and subcommands using `signal.NotifyContext`, `ExecuteContext`, and `cmd.Context()`.
- Replace `context.Background()` in `collect` and `ingest` with command-scoped context so plugin timeouts, process interrupts, and CI cancellation propagate correctly.
- Add CLI and hardening tests that prove timeout/cancel paths do not create ambiguous partial behavior and that non-verify errors omit misleading breakpoint fields.
- Keep help, usage, and exit-code contracts stable while tightening machine-readable semantics.
Repo paths:
- `cmd/axym/output.go`
- `cmd/axym/root.go`
- `cmd/axym/main.go`
- `cmd/axym/collect.go`
- `cmd/axym/ingest.go`
- `cmd/axym/root_test.go`
- `cmd/axym/collect_test.go`
- `cmd/axym/ingest_test.go`
- `internal/e2e/cli/command_surface_contract_test.go`
- `internal/e2e/cli/help_usage_contract_test.go`
- `internal/e2e/plugin/malformed_jsonl_rejected_test.go`
- `internal/hardening/sink_unavailable_fail_closed_test.go`
- `testinfra/contracts/cli_output_contract_test.go`
- `testinfra/contracts/verify_contract_test.go`
Run commands:
- `go build -o ./axym ./cmd/axym`
- `go test ./cmd/axym ./internal/e2e/cli ./internal/e2e/plugin -count=1`
- `go test ./testinfra/contracts -run 'CLIOutput|Verify' -count=1`
- `make test-hardening`
- `./axym verify --json`
- `./axym ingest --source gait --input /tmp/does-not-exist --json`
Test requirements:
- CLI behavior changes: help/usage tests, `--json` stability tests, exit-code contract tests.
- Job runtime/state/concurrency changes: timeout/cancellation propagation checks for long-running workflows.
- Gate/policy/fail-closed changes: ensure cancellation does not create silent partial success.
- Cross-platform smoke expectations for signal-aware command execution.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance, Cross-platform, Risk.
- Required gates: command-surface contracts, hardening smoke, plugin timeout/cancel tests, and release-risk lane.
Acceptance criteria:
- Non-verify errors no longer emit `break_index: 0` or other misleading verification-only fields.
- `collect` and `ingest` honor command cancellation and configured timeouts end-to-end.
- Timeout/cancel handling does not introduce silent partial success or unstable JSON envelopes.
- Help, usage, and exit-code vocabulary remain unchanged unless explicitly versioned.
Stable/internal boundary notes:
- Public: CLI JSON error envelope, timeout/cancel behavior, and reason-code stability are contract surfaces.
- Internal: context plumbing and helper layering remain implementation detail.
Migration expectations:
- Additive/cleanup only. Existing consumers should see stricter semantics, not new opaque failure classes.
Integration hooks:
- CLI automation, wrappers, and CI jobs that parse `--json` output from `collect`, `ingest`, and `verify`.
- Long-running plugin-backed collection and sibling-ingest workflows that need bounded cancellation semantics.
Dependencies:
- None.
Risks:
- Cross-platform cancellation behavior must be validated at smoke level without introducing Unix-only assumptions into merge gates.

---

## Epic W3: Docs, ADR, and Launch-Facing Contract Sync

Objective: update every public/source-of-truth document only after runtime behavior and tests are stable.

### Story W3-S1: Align product, operator, and protocol docs with shipped behavior
Priority: P1
Tasks:
- Update `product/axym.md`, `README.md`, `docs/commands/axym.md`, `docs/operator/quickstart.md`, `docs/operator/integration-model.md`, `docs/operator/integration-boundary.mmd`, `docs-site/public/llms.txt`, and `docs-site/public/llm/axym.md` to describe the shipped runtime accurately.
- Update ADRs that now have changed contract semantics: collector/plugin protocol (`ADR-0002`) and bundle/verification behavior (`ADR-0005`).
- Remove or correct any doc claim that plugin collectors emit raw `[]proof.Record` if the shipped runtime still uses normalized candidate promotion.
- Document the stricter `record add` contract, the signature-aware verify/bundle behavior, and any additive bundle signing-key artifact introduced by `W1-S1`.
- Run docs source-of-truth checks so README, docs, docs-site, and command parity remain synchronized.
Repo paths:
- `product/axym.md`
- `README.md`
- `docs/commands/axym.md`
- `docs/operator/quickstart.md`
- `docs/operator/integration-model.md`
- `docs/operator/integration-boundary.mmd`
- `docs/adr/ADR-0002-collector-runtime-and-plugin-protocol.md`
- `docs/adr/ADR-0005-bundle-assembly-and-verification.md`
- `docs-site/public/llms.txt`
- `docs-site/public/llm/axym.md`
- `testinfra/contracts/command_docs_parity_contract_test.go`
- `testinfra/contracts/product_identity_contract_test.go`
- `scripts/check_docs_consistency.sh`
- `scripts/check_docs_storyline.sh`
- `scripts/check_docs_links.sh`
Run commands:
- `make test-docs-consistency`
- `make test-docs-storyline`
- `make test-docs-links`
- `go test ./testinfra/contracts -run 'CommandDocsParity|ProductIdentity|CLIOutput|Verify' -count=1`
- `go test ./testinfra/acceptance/... -count=1`
Test requirements:
- Docs/examples changes: docs consistency checks, storyline/smoke checks, README/quickstart/integration coverage checks, and docs source-of-truth sync tasks for `README.md`, `docs/`, `docs-site/public/llms.txt`, and `docs-site/public/llm/*.md`.
- Public-surface stories: machine-readable error behavior, integration hooks, and migration notes must be documented explicitly.
Matrix wiring:
- Lanes: Fast, Core CI, Acceptance.
- Required gates: docs consistency, docs storyline, docs links, docs parity contract tests.
Acceptance criteria:
- All public docs describe manual proof ingress as schema-validated and fail-closed.
- All public docs describe `verify --chain` and `verify --bundle` as signature-aware where shipped.
- Plugin protocol docs match the actual supported runtime shape and relationship-preservation semantics exactly.
- No public doc widens Axym beyond its existing CLI/proof scope or implies a raw plugin append path that the runtime does not support.
Stable/internal boundary notes:
- Public: docs are source-of-truth for supported integration hooks and machine-readable runtime semantics.
- Internal: helper/package placement and implementation detail remain out of scope for public docs.
Migration expectations:
- Docs must describe any additive bundle artifact or stricter validation behavior introduced in earlier waves.
- No undocumented fallback behavior may remain after this story closes.
Integration hooks:
- README and quickstart users following install-time verification and manual proof-ingress paths.
- Plugin authors and operators using command docs, operator docs, and docs-site content as their integration reference.
Dependencies:
- `W1-S1`
- `W1-S2`
- `W2-S1`
- `W2-S2`
Risks:
- Shipping docs early recreates the same trust gap under new wording; this story must remain last.

---

## Minimum-Now Sequence

1. `W1-S1` first. Signature-aware verify/bundle behavior is the highest-risk release blocker and defines the offline integrity contract for every later story.
2. `W1-S2` second. The authoritative append boundary must reject invalid proof payloads before the team broadens or documents any ingress guarantees.
3. `W2-S1` third. Relationship preservation depends on a trustworthy ingress/append contract and becomes the runtime truth that docs later describe.
4. `W2-S2` fourth. JSON envelope cleanup and signal-aware cancellation should land after the main integrity surfaces are stable so contract tests can lock the final CLI runtime seam.
5. `W3-S1` last. Public docs, ADRs, and docs-site content should only change once the runtime and tests are authoritative.

Wave rationale:

- `W1` is pure contract/runtime correctness at the proof and append boundaries.
- `W2` is still runtime work, but it focuses on adapter preservation and CLI execution-boundary hygiene once the core integrity rules are fixed.
- `W3` is docs/source-of-truth sync and therefore intentionally follows all runtime waves.

---

## Explicit Non-Goals

- No new product scope beyond Axym's existing OSS CLI and documented `Clyra-AI/proof` interoperability.
- No new dashboard, service, or default network dependency.
- No new `ingest --source` surface such as `agnt`; supported generic ingress remains `collect --governance-event-file`, manual proof append, and existing proof-format paths.
- No top-level proof-record format fork or Axym-only record envelope.
- No exit-code vocabulary expansion beyond the existing `0,1,2,3,4,5,6,7,8` contract.
- No docs-only repositioning ahead of runtime truth.

---

## Definition of Done

- Every recommendation in this plan maps to at least one completed story with green required lanes.
- Signature tamper, schema-invalid manual append, plugin relationship loss, and misleading JSON error-envelope regressions are all covered by blocking tests.
- `verify`, `bundle`, `record add`, `collect --plugin`, and `ingest` behavior are deterministic, fail-closed where required, and accurately documented.
- Any additive artifact introduced for offline signature verification is schema-validated, manifest-covered, deterministic across repeated builds, and documented.
- Public docs and ADRs match shipped runtime behavior across `product/`, `README.md`, `docs/`, and `docs-site`.
- No new nondeterminism, default exfiltration path, unsafe output-path behavior, or exit-code drift is introduced while closing these blockers.
