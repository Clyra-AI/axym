# Axym

Axym is an open-source Go CLI for deterministic proof of identity-governed action in software delivery: who initiated, who executed, which target was touched, which owner approved it, and which policy governed it.

## Who it's for

Axym is built for platform, security, and GRC engineers who need local evidence collection, compliance mapping, and audit-ready bundles without shipping evidence to a hosted service by default.

## Where Axym fits

Axym sits between your runtime and your audit output:

- Your code, CI system, model provider, MCP servers, and sibling systems emit events or records.
- Axym collects, ingests, or appends that evidence locally.
- Axym maps the resulting proof chain to frameworks, ranks gaps, and assembles audit bundles.

Axym does not replace IAM, PAM, or IGA systems. The OSS CLI distinguishes built-in collection, plugin collection, manual proof-record append, and sibling ingest.

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

If you installed via Homebrew, replace `./axym` with `axym` in the commands below.

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

Use this when you want a supported, offline, installed-binary demo that ends in non-empty evidence and ranked compliance output.

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
- `map` reports `5` covered controls out of `6` across `eu-ai-act,soc2`.
- `gaps` reports grade `C`, leaving SOC 2 `cc7` as the remaining sample gap.
- `bundle` emits identity-governance artifacts, keeps compliance incomplete (`complete=false`), and leaves `weak_record_count=1`.
- The identity-governance artifacts are `identity-chain-summary.json`, `ownership-register.json`, `privilege-drift-report.json`, and `delegated-chain-exceptions.json`.
- `verify --chain --json` reports an intact `6`-record chain.

## Real integration path

Use this when you are wiring Axym into your actual runtime, CI, or sibling governance systems.

- Built-in collectors: `mcp`, `llmapi`, `webhook`, `githubactions`, `gitmeta`, `dbt`, `snowflake`, and `governanceevent`.
- Plugin collectors: `./axym collect --json --plugin "./my-collector"`.
- Manual record append: `./axym record add --input ./my-record.json --json`.
- Sibling ingest: `./axym ingest --source wrkr --json --input ./wrkr-records.jsonl` and `./axym ingest --source gait --json --input ./gait-pack`.

Approvals, risk assessments, incidents, guardrail activations, and similar evidence types are not claimed as default built-in clean-room capture. Those arrive through built-in surfaces only when the corresponding source exists, or through plugin, manual, or ingest paths.

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

`collect` emits deterministic per-source summaries (`sources[]`) with `reason_codes`, supports non-blocking collector failures, and keeps malformed plugin and governance payloads out of the proof chain.

`ingest` supports deterministic sibling ingest from Wrkr and Gait. Wrkr ingest persists drift baseline state in `.axym/wrkr-last-ingest.json`; Gait ingest supports zip, extracted, and explicit-path packs while preserving relationship envelopes.

`map` deterministically matches chain evidence to framework controls and emits per-control rationale for `covered`, `partial`, and `gap` outcomes.

`gaps` ranks `partial` and `gap` controls with deterministic remediation and auditability grade output.

`bundle` assembles deterministic artifact sets, signs the manifest with local proof keys, and enforces managed output path safety.

`verify --chain` reports deterministic local integrity for both append-only chain linkage and Axym-managed record signatures.

`verify --bundle` reports manifest-signature verification, per-record signature verification for Axym-authored bundles, and deterministic compliance-completeness checks without creating store-managed temp artifacts.

## Context engineering evidence

Axym can capture governance-relevant context engineering events without storing raw prompt bodies by default. Supported additive event types include `instruction_rewrite`, `context_reset`, and `knowledge_import`.

Example:

```bash
./axym collect --json --governance-event-file ./fixtures/governance/context_engineering.jsonl
```

These events can carry digest-first fields such as `previous_hash`, `current_hash`, `artifact_digest`, `artifact_kind`, `source_uri`, and `reason_code`.

## Contributor gate model

Fast local checks:

```bash
make lint-fast
make test-fast
make test-contracts
```

Normal contributors can usually stop here unless they are changing public docs, CI contracts, release behavior, or other launch-facing surfaces.

Full local gate for public-surface, CI, or release-adjacent changes:

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

Hosted CI remains authoritative for pull-request workflow enforcement and GitHub-hosted CodeQL analysis.

## Support and security

- Public GitHub issues are the default path for bugs, questions, and feature requests.
- Security-sensitive reports must use GitHub Security Advisories as the private reporting path described in [SECURITY.md](SECURITY.md).
- If GitHub Security Advisories are unavailable, open a minimal public issue without exploit details and reference [SECURITY.md](SECURITY.md).
- Maintainer support for the OSS CLI is best-effort and async.

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
