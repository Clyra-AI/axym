# Axym Operator Quickstart

This guide is for platform, security, and GRC operators wiring Axym into a real environment. It separates three paths that serve different goals: `smoke test`, `sample proof path`, and `real integration path`.

See the boundary model in [integration-model.md](integration-model.md) and the companion diagram in [integration-boundary.mmd](integration-boundary.mmd).

## Install

Homebrew:

```bash
brew install Clyra-AI/tap/axym
./axym version --json
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

- The sample pack is created locally with no network fetch and no repo fixture dependency.
- The resulting chain contains 5 records: 3 governance-event decisions, 1 approval, and 1 risk assessment.
- `map` reports 5 covered controls out of 6 across `eu-ai-act,soc2`.
- `gaps` returns grade `C`, leaving SOC 2 `cc7` as the remaining sample gap.
- `bundle` succeeds and `verify --chain --json` reports an intact local chain.

## Real integration path

Use this when you are connecting Axym to your actual runtime, CI, or sibling systems.

- Built-in collectors: `mcp`, `llmapi`, `webhook`, `githubactions`, `gitmeta`, `dbt`, `snowflake`, and `governanceevent`.
- Plugin collectors: `./axym collect --json --plugin "<cmd>"`.
- Manual record append: `./axym record add --input <record.json> --json`.
- Sibling ingest: `./axym ingest --source wrkr --input <path> --json` and `./axym ingest --source gait --input <path> --json`.

This is the path to use when you want evidence from your own CI, runtime, provider, or sibling governance systems. Approvals, risk assessments, incidents, and similar evidence classes are not claimed as default built-in clean-room capture unless that collector ships.

## Failure handling and expected outputs

- `collect --dry-run --json` is expected to report per-source `would_capture` and `reason_codes` without side effects.
- `collect --json` reports per-source `status`, `captured`, `rejected`, and `failures`; zero capture is valid when no real inputs exist.
- `map --json` and `gaps --json` emit deterministic summaries even when coverage is incomplete.
- `verify --chain --json` validates local append-only integrity.
- `verify --bundle <path> --json` validates bundle cryptographic integrity plus Axym compliance completeness state.

## Validation

```bash
make lint-go
make test-security
make test-docs-links
make prepush-full
```
