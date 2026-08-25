# Axym Command Guide

Axym is a deterministic CLI for proving identity-governed action in software delivery for platform, security, and GRC teams that need local evidence collection, compliance mapping, and audit-ready bundles.

## Runtime boundary

Axym collects or ingests evidence from systems you already operate. Customer code, CI, MCP servers, model providers, sibling systems, and IAM/PAM/IGA systems stay upstream. Axym turns the resulting structured evidence into local proof records, compliance maps, gaps, and bundles that answer who acted, through which chain, against which target, under which policy and approval.
Gait enforces runtime controls. Wrkr inventories ownership and posture. Proof verifies signatures and chain integrity. Axym correlates, maps, and packages that evidence.
Axym does not block, freeze, broker, or sandbox execution.

Operator walkthroughs live in [../operator/quickstart.md](../operator/quickstart.md) and [../operator/integration-model.md](../operator/integration-model.md).

## Install paths

Homebrew:

```bash
brew install Clyra-AI/tap/axym
axym version --json
```

Source:

```bash
go build ./cmd/axym
./axym version --json
```

Release binary:

```bash
./axym version --json
```

If you installed via Homebrew, replace `./axym` with `axym` in the commands below.

## Smoke test

Use this when you want to confirm the binary and local environment are wired correctly.

```bash
./axym init --json
./axym collect --dry-run --json
```

Expected outcome:

- `init` creates the local store and default policy.
- `collect --dry-run` shows deterministic would-capture output without writes.
- A fresh environment may still return `captured: 0` on plain `collect --json`; that validates the smoke path, but it is not the published first-value result.

## Sample proof path

Use this when you want a supported offline demo that ends with non-empty evidence and a non-empty compliance result.

First value is evidence + ranked gaps + intact local verification, not full audit completeness.

```bash
./axym init --sample-pack ./axym-sample --json
./axym collect --json --governance-event-file ./axym-sample/governance/context_engineering.jsonl
./axym record add --input ./axym-sample/records/approval.json --json
./axym record add --input ./axym-sample/records/risk_assessment.json --json
./axym map --frameworks eu-ai-act,soc2 --json
./axym gaps --frameworks eu-ai-act,soc2 --json
./axym bundle --audit sample --frameworks eu-ai-act,soc2 --json
./axym verify --chain --json
```

Expected outcome:

- The sample pack is created locally with no network dependency and no repo fixture dependency.
- `collect` captures `4` governance events from the bundled sample pack.
- The local chain ends with `6` total records after the manual approval and risk assessment append.
- `map` reports `6` covered controls out of `10` across `eu-ai-act,soc2`.
- `gaps` reports grade `C`, leaving controls `article-15`, `article-26`, `cc7.1`, `cc8.1` as the remaining sample gaps.
- `bundle` emits identity-governance artifacts, keeps compliance incomplete (`complete=false`), and leaves `weak_record_count=1`.
- The identity-governance artifacts are `identity-chain-summary.json`, `ownership-register.json`, `privilege-drift-report.json`, and `delegated-chain-exceptions.json`.
- `verify --chain --json` reports an intact `6`-record chain.

## Real integration path

- Built-in collectors: `mcp`, `llmapi`, `webhook`, `githubactions`, `gitmeta`, `dbt`, `snowflake`, and `governanceevent`.
- Plugin collectors: `axym collect --json --plugin "<cmd>"`.
- Manual record append: `axym record add --input <record.json> --json`.
- Authoritative contract: [../../schemas/v1/record/README.md](../../schemas/v1/record/README.md).
- Sibling ingest: `axym ingest --source wrkr --json --input <path>` and `axym ingest --gait-pack <path> --json`.
- Action Contract compatibility: `axym action-contract consume <proposed_action_contract.json> --json` validates exactly one Wrkr v3 proposal, preserves producer-native IDs/refs, and reports a non-authoritative receipt conforming to Axym's versioned consumer-receipt schema. The standalone conformance entrypoint is `axym-action-contract-consumer <proposed_action_contract.json>`; see [the versioned schema contract](../../schemas/v1/action_contract/README.md).
- Stable today: built-in collection, plugin collection, manual record append, sibling ingest, and `map`/`gaps`/`bundle`/`verify`.
- Internal detail: package names, workflow step ordering, and helper placement are not public extension points.
- Deprecated surface: none documented in launch docs today.

Public docs should not describe approvals, risk assessments, incidents, guardrails, or broader enterprise surfaces as default built-in clean-room capture unless that collector actually ships.
Public docs should also not position Axym as an IAM/PAM/IGA replacement or widen the wedge beyond software delivery.

## Commands

### Governance evidence projections

`axym governance ingest --kind telemetry --input <otlp.json> --json` ingests
offline OTLP/third-party trace and boundary-attestation JSON into a
digest-bound, read-only projection. `--kind judge` accepts Judge evidence as
an advisory projection only: it cannot establish execution authority, override
Axym compliance results, or be translated into an execution record. Both paths
fail closed for malformed, stale, tampered, and out-of-scope input.

Released Gait v1.5 control-extension fixtures are mirrored under
`testdata/governance/v1/gait-control` by the offline import/check generator.
They are explicitly synthetic, quarantined, and non-authoritative; Axym
preserves their causal references and statuses without creating execution or
compliance authority.

- `axym init --json`: creates local store scaffolding and policy defaults.
- `axym init --sample-pack ./axym-sample --json`: creates the local store plus a deterministic sample pack with machine-readable created files and next steps.
- `axym collect --dry-run --json`: validates fixture and environment readiness without writes and preserves per-source `reason_codes`, including `governanceevent: ["NO_INPUT"]` when no governance-event files are supplied.
- `axym collect --json`: runs built-in collectors and appends signed proof records from configured sources.
- `axym collect --json --plugin "<cmd>"`: runs a third-party collector protocol and promotes normalized collector JSONL (`source_type`, `source`, `source_product`, `record_type`, `agent_id`, `timestamp`, `event`, `metadata`, optional `relationship`, `controls`) into signed proof records while rejecting malformed payloads deterministically.
- `axym collect --json --governance-event-file ./events.jsonl`: promotes valid governance events to proof records with actor, downstream, owner, delegation, policy, and approval linkage when present.
- `axym record add --input <record.json> --json`: validates the public manual-input contract, normalizes compatibility-only `record_version: "1.0"` payloads to `v1`, then signs and appends the record to the local chain. Shared proof-record semantics remain owned by `Clyra-AI/proof`; see [../../schemas/v1/record/README.md](../../schemas/v1/record/README.md).
- `axym ingest --source wrkr --json --input <path>`: ingests Wrkr evidence with stateful drift tracking.
- `axym ingest --gait-pack <path> --json`: ingests Gait native/proof packs plus authorization bundles and structured control artifacts with deterministic translation and bundle counts.
- `axym ingest --source gait --input <path> --gait-lifecycle-verification <config.json> --json`: verifies a caller-owned Gait v1.5 lifecycle context (trusted key, exact lineage digests, validity window, and optional fixture allowance) before lifecycle ingestion. The config must conform to [the versioned schema](../../schemas/v1/gait/lifecycle-verification-config-v1.schema.json); see the [illustrative example](../../schemas/v1/gait/lifecycle-verification-config.example.json).
- `axym action-contract consume <path> --json`: consumes one report-only Wrkr proposal; it never treats the proposal as execution authority and does not claim execution/effect/containment evidence.
- Action Contract verification is fail-closed for symlinked paths, stale/ambiguous selection evidence, absent evaluation time, and development signing; machine-readable reasons are stable and never contain filesystem/parser text.
- `axym map --frameworks eu-ai-act,soc2 --json`: deterministically maps chain evidence to framework controls.
- `axym gaps --frameworks eu-ai-act,soc2 --json`: ranks `covered`, `partial`, and `gap` outcomes with remediation and effort.
- `axym regress init --baseline ./tmp/regress-baseline.json --frameworks eu-ai-act,soc2 --json`: captures deterministic baseline coverage.
- `axym regress run --baseline ./tmp/regress-baseline.json --frameworks eu-ai-act,soc2 --json`: exits `5` on coverage drift or control-maturity regression with stable output.
- `axym review --date 2026-09-15 --json`: emits a deterministic Daily Review Pack.
- `axym override create --bundle Q3-2026 --reason "fixture" --signer ops-key --json`: appends signed override evidence and artifacts.
- `axym replay --model payments-agent --tier A --json`: emits replay-certification evidence with deterministic blast-radius summaries.
- `axym bundle --audit Q3-2026 --frameworks eu-ai-act,soc2 --json`: assembles signed audit bundles with executive summary, identity-governance artifacts, OSCAL, portable raw records, and when Gait control evidence exists: `authorization-register.json`, `insurance-evidence-profile.json`, `credential-posture-register.json`, `freeze-window-coverage.json`, `kill-switch-coverage.json`, `enforcement-explain-register.json`, `sandbox-coverage.json`, and `control-maturity.json`.
- `axym verify --chain --json`: verifies append-only chain integrity plus Axym-managed record signatures.
- `axym verify --bundle ./axym-evidence --json`: verifies bundle manifest signatures, Axym-authored record signatures, and compliance completeness, including identity-governance artifact consistency, without writing store-managed temp artifacts.
  It recomputes `authorization-register.json`, `insurance-evidence-profile.json`, `credential-posture-register.json`, `freeze-window-coverage.json`, `kill-switch-coverage.json`, `enforcement-explain-register.json`, `sandbox-coverage.json`, and `control-maturity.json` when they are declared in the bundle.

### Framework evidence-set mapping

Axym supports both legacy Proof framework controls and Proof `v0.6.1` evidence sets. Legacy `required_record_types` keep their existing alternative-match behavior. For `evidence_sets`, Axym treats sets as alternatives, requires every `required_record_type` within one set, and applies that set's `source_products`, `required_fields`, and `minimum_frequency` constraints together. A control is covered when any one set is complete.

When no set is complete, Axym deterministically reports the closest partial or gap alternative by matched-type completeness, then missing types and fields; the evidence-set ID breaks any remaining tie. `map --json` exposes the selected set as an `evidence_set=<id>` prefix in `rationale`. A multi-type set with only some required types remains `partial` and includes `REQUIRED_RECORD_TYPES_NOT_MET` in `reason_codes`. `gaps`, generated bundles, and `verify --bundle` use the same selected-set requirements, so remediation and audit artifacts do not flatten alternatives into false coverage.

## Contributor checks

Fast local checks:

```bash
make lint-fast
make test-fast
make test-contracts
```

Extended local checks:

```bash
make prepush-full
```

Required tools for `make prepush-full`: `golangci-lint`, `gosec`, and `codeql`.

Maintainer and release-manager verification:

```bash
make release-local
make release-go-nogo-local
./scripts/release_go_nogo.sh --dist-dir dist --binary-name axym
```

Additional required tools for `make release-local` and `make release-go-nogo-local`: `syft` and `cosign`.

Hosted CI remains authoritative for pull-request required checks and GitHub-hosted CodeQL analysis.

## Release verification

```bash
./scripts/release_go_nogo.sh --dist-dir dist --binary-name axym
```

## Exit codes

- `0` success
- `1` runtime failure
- `2` verification failure
- `3` policy/schema violation
- `4` approval required
- `5` regression drift
- `6` invalid input
- `7` dependency missing
- `8` unsafe operation blocked
