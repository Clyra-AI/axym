# Axym

Axym is an open-source Go CLI for deterministic AI governance evidence collection, proof record emission, compliance mapping, and audit-ready bundle generation.

## Who it's for

Axym is built for platform, security, and GRC engineers who need to prove how AI systems behave without shipping evidence to a hosted service by default.

## Where Axym fits

Axym sits between your AI runtime and your audit output:

- Your code, CI system, model provider, MCP servers, and sibling systems emit events or records.
- Axym collects, ingests, or appends that evidence locally.
- Axym maps the resulting proof chain to frameworks, ranks gaps, and assembles audit bundles.

Axym does not pretend to own every upstream signal. The current OSS surface distinguishes built-in collectors, plugin collectors, manual proof-record append, and sibling ingest.

## Install

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

Requires Go `1.26.1` for source builds.

## Project docs

- [LICENSE](LICENSE): Apache-2.0 terms for use, redistribution, and contribution.
- [CHANGELOG.md](CHANGELOG.md): notable user-visible changes and release-facing notes.
- [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md): community expectations for issues, PRs, and discussions.
- [CONTRIBUTING.md](CONTRIBUTING.md): contribution flow, scope expectations, and local validation.
- [SECURITY.md](SECURITY.md): private reporting guidance and security boundary expectations.
- [docs/operator/quickstart.md](docs/operator/quickstart.md): operator walkthrough for smoke, sample, and real integration paths.
- [docs/operator/integration-model.md](docs/operator/integration-model.md): ownership boundaries, sync/async evidence flow, and integration model.

## Smoke test

Use this when you want to confirm the binary, local store, and deterministic command surface work on your machine.

```bash
./axym init --json
./axym collect --dry-run --json
```

Expected outcome:

- `init` creates `.axym` and `axym-policy.yaml`.
- `collect --dry-run` reports what Axym could capture without writing evidence.
- A fresh environment may still yield `captured: 0` on plain `collect --json`; that is a smoke test, not the supported first-value path.

## Sample proof path

Use this when you want a supported, offline, installed-binary demo that ends in non-empty evidence and non-empty compliance results.

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

- The sample pack is created locally with no repo fixture dependency and no network dependency.
- `collect` appends 3 governance-event-derived records.
- The full sample flow produces 5 covered controls out of 6 across `eu-ai-act,soc2`.
- `gaps` returns grade `C` for the published sample journey, not grade `F`.
- `bundle` succeeds and `verify --chain --json` reports an intact 5-record chain.

## Real integration path

Use this when you are wiring Axym into your actual runtime, CI, or sibling governance systems.

- Built-in collectors: `mcp`, `llmapi`, `webhook`, `githubactions`, `gitmeta`, `dbt`, `snowflake`, and `governanceevent`.
- Plugin collectors: `./axym collect --json --plugin "./my-collector"`.
- Manual record append: `./axym record add --input ./my-record.json --json`.
- Sibling ingest: `./axym ingest --source wrkr --json --input ./wrkr-records.jsonl` and `./axym ingest --source gait --json --input ./gait-pack`.

Approvals, risk assessments, incidents, guardrail activations, and similar evidence types are not claimed as default built-in clean-room capture today. Those arrive through built-in surfaces only when the corresponding source exists, or through plugin/manual/ingest paths.

Operator detail lives in [docs/operator/quickstart.md](docs/operator/quickstart.md) and [docs/operator/integration-model.md](docs/operator/integration-model.md).

## Supported surfaces today

```bash
./axym init --json
./axym collect --dry-run --json
./axym collect --json --plugin "./my-collector"
./axym collect --json --governance-event-file ./events.jsonl
./axym record add --input ./my-record.json --json
./axym ingest --source wrkr --json --input ./wrkr-records.jsonl
./axym ingest --source gait --json --input ./gait-pack
./axym map --frameworks eu-ai-act,soc2 --json
./axym gaps --frameworks eu-ai-act,soc2 --json
./axym regress init --baseline ./tmp/regress-baseline.json --frameworks eu-ai-act,soc2 --json
./axym regress run --baseline ./tmp/regress-baseline.json --frameworks eu-ai-act,soc2 --json
./axym review --date 2026-09-15 --json
./axym review --date 2026-09-15 --format csv
./axym override create --bundle Q3-2026 --reason "fixture" --signer ops-key --json
./axym replay --model payments-agent --tier A --json
./axym bundle --audit Q3-2026 --frameworks eu-ai-act,soc2 --json
./axym verify --chain --json
./axym verify --bundle ./axym-evidence --json
```

## Context engineering evidence

Axym can capture governance-relevant context engineering events without storing raw prompt bodies by default. The supported additive event types are:

- `instruction_rewrite`
- `context_reset`
- `knowledge_import`

Example:

```bash
./axym collect --json --governance-event-file ./fixtures/governance/context_engineering.jsonl
```

These events are intended to carry digest-first fields such as `previous_hash`, `current_hash`, `artifact_digest`, `artifact_kind`, `source_uri`, and `reason_code`.

## Contributor gate model

Local-fast checks:

```bash
make lint-fast
make test-fast
make test-contracts
```

Local-extended checks:

```bash
make lint-go
make test-security
make test-docs-links
make prepush-full
```

Hosted CI remains authoritative for pull-request workflow enforcement and CodeQL analysis through GitHub Actions.

`collect` emits deterministic per-source summaries (`sources[]`) with `reason_codes`, supports non-blocking collector failures, and keeps malformed plugin/governance payloads out of the proof chain.

Built-in collection is limited to the collectors listed above. Plugin collection, manual record append, and sibling ingest are separate supported paths and should not be described as one default behavior.

`ingest` supports deterministic sibling ingest from Wrkr and Gait. Wrkr ingest persists drift baseline state in `.axym/wrkr-last-ingest.json`; Gait ingest supports zip/extracted/explicit-path packs and translates `trace`, `approval_token`, and `delegation_token` native records to proof records while preserving relationship envelopes.

`map` deterministically matches chain evidence to framework controls and emits per-control rationale for `covered`/`partial`/`gap` outcomes.

`init` bootstraps local store material and an `axym-policy.yaml` file with deterministic defaults. `init --sample-pack <dir>` additively materializes a local sample proof path with machine-readable created files and next-step commands.

`record add` appends a user-supplied proof record JSON payload to the local chain with deterministic dedupe semantics.

`map`/`gaps` default to frameworks from `axym-policy.yaml` when present, otherwise `eu-ai-act,soc2`. Invalid policy config fails closed with exit `6`.

`regress init` captures deterministic per-control coverage baselines. `regress run` compares current coverage to baseline and exits `5` on drift with stable `regressed_controls` output.

`gaps` ranks `partial`/`gap` controls with deterministic remediation and auditability grade output; `--min-coverage` or `--policy-config` can enforce fail-closed coverage thresholds.

`review` emits deterministic daily exception packs with fixed exception classes (`sod`, `approvals`, `enrichment`, `attach`, `replay`, `freeze`, `chain-session-gap`), per-record auditability, replay tier distributions, and attach SLA/status envelopes.

`override create` appends signed override evidence records and append-only override artifacts under `.axym/overrides/`.

`replay` emits `replay_certification` proof records with deterministic tier classification and blast-radius summary fields.

`bundle` assembles deterministic artifact sets (`manifest.json`, `chain-verification.yaml`, `auditability-grade.yaml`, `executive-summary.json`, `executive-summary.pdf`, OSCAL export, and retention/boundary contracts), signs the manifest with local proof keys, and enforces managed output path safety.

`verify --bundle` reports cryptographic integrity plus deterministic Axym compliance-completeness checks (required record classes, field-coverage state, grade recomputation, and OSCAL schema validation) without creating store-managed temp artifacts.

Release verification uses:

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
