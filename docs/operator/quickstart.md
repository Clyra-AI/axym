# Axym Operator Quickstart

This guide is for platform, security, and GRC operators wiring Axym into a real environment. Axym's job is to prove identity-governed action in software delivery, not to replace IAM, PAM, or IGA systems. It separates three paths that serve different goals: `smoke test`, `sample proof path`, and `real integration path`.

See the boundary model in [integration-model.md](integration-model.md) and the companion diagram in [integration-boundary.mmd](integration-boundary.mmd).

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

## Smoke test

Use this first if you want to verify the binary, local store, and deterministic command surface on your machine.

```bash
./axym init --json
./axym collect --dry-run --json
```

Expected outcome:

- `init` creates `.axym` and `axym-policy.yaml`.
- `collect --dry-run` shows deterministic `would_capture` output without writing evidence.
- A clean environment may still return `captured: 0` on plain `collect --json`; that confirms the smoke path, but it is not the supported first-value flow.

## Sample proof path

Use this when you want a supported offline path that produces non-empty local evidence and non-empty compliance output on a fresh install.

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

Use this when you are connecting Axym to your actual runtime, CI, or sibling systems.

- Built-in collectors: `mcp`, `llmapi`, `webhook`, `githubactions`, `gitmeta`, `dbt`, `snowflake`, and `governanceevent`.
- Plugin collectors: `./axym collect --json --plugin "<cmd>"`.
- Manual record append: `./axym record add --input <record.json> --json` after Axym validates the proof payload and signs/links it locally.
- Authoritative contract: [../../schemas/v1/record/README.md](../../schemas/v1/record/README.md).
- Sibling ingest: `./axym ingest --source wrkr --input <path> --json` and `./axym ingest --source gait --input <path> --json`.
- Stable today: built-in collection, plugin collection, manual record append, sibling ingest, and `map`/`gaps`/`bundle`/`verify`.
- Internal detail: package names, workflow step ordering, and helper placement are not public extension points.
- Deprecated surface: none documented in launch docs today.

This is the path to use when you want evidence from your own CI, runtime, provider, or sibling governance systems. Approvals, risk assessments, incidents, and similar evidence classes are not claimed as default built-in clean-room capture unless that collector ships.
Use this path when you need to prove which non-human identity acted, which owner approved it, which target was touched, and which policy or approval bound the action.

## Failure handling and expected outputs

- `collect --dry-run --json` is expected to report per-source `would_capture` and `reason_codes` without side effects.
- `collect --json` reports per-source `status`, `captured`, `rejected`, and `failures`; zero capture is valid when no real inputs exist.
- `map --json` and `gaps --json` emit deterministic summaries even when coverage is incomplete.
- `verify --chain --json` validates local append-only integrity plus Axym-managed record signatures.
- `verify --bundle <path> --json` validates manifest signatures, Axym-authored record signatures, and Axym compliance completeness state, including the identity-chain summary, ownership register, privilege-drift report, and delegated-chain exceptions artifacts.

Contributor and release validation lives in [../../README.md](../../README.md) and [../../CONTRIBUTING.md](../../CONTRIBUTING.md).
