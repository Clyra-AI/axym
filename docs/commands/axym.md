# Axym Command Guide

Axym is a deterministic CLI for proving identity-governed action in software delivery for platform, security, and GRC teams that need local evidence collection, compliance mapping, and audit-ready bundles.

## Runtime boundary

Axym collects or ingests evidence from systems you already operate. Customer code, CI, MCP servers, model providers, sibling systems, and IAM/PAM/IGA systems stay upstream. Axym turns the resulting structured evidence into local proof records, compliance maps, gaps, and bundles that answer who acted, through which chain, against which target, under which policy and approval.

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
- `map` reports `5` covered controls out of `6` across `eu-ai-act,soc2`.
- `gaps` reports grade `C`, leaving SOC 2 `cc7` as the remaining sample gap.
- `bundle` emits identity-governance artifacts, keeps compliance incomplete (`complete=false`), and leaves `weak_record_count=1`.
- The identity-governance artifacts are `identity-chain-summary.json`, `ownership-register.json`, `privilege-drift-report.json`, and `delegated-chain-exceptions.json`.
- `verify --chain --json` reports an intact `6`-record chain.

## Real integration path

- Built-in collectors: `mcp`, `llmapi`, `webhook`, `githubactions`, `gitmeta`, `dbt`, `snowflake`, and `governanceevent`.
- Plugin collectors: `axym collect --json --plugin "<cmd>"`.
- Manual record append: `axym record add --input <record.json> --json`.
- Sibling ingest: `axym ingest --source wrkr --json --input <path>` and `axym ingest --source gait --json --input <path>`.

Public docs should not describe approvals, risk assessments, incidents, guardrails, or broader enterprise surfaces as default built-in clean-room capture unless that collector actually ships.
Public docs should also not position Axym as an IAM/PAM/IGA replacement or widen the wedge beyond software delivery.

## Commands

- `axym init --json`: creates local store scaffolding and policy defaults.
- `axym init --sample-pack ./axym-sample --json`: creates the local store plus a deterministic sample pack with machine-readable created files and next steps.
- `axym collect --dry-run --json`: validates fixture and environment readiness without writes.
- `axym collect --json`: runs built-in collectors and appends signed proof records from configured sources.
- `axym collect --json --plugin "<cmd>"`: runs a third-party collector protocol and promotes normalized collector JSONL (`source_type`, `source`, `source_product`, `record_type`, `agent_id`, `timestamp`, `event`, `metadata`, optional `relationship`, `controls`) into signed proof records while rejecting malformed payloads deterministically.
- `axym collect --json --governance-event-file ./events.jsonl`: promotes valid governance events to proof records with actor, downstream, owner, delegation, policy, and approval linkage when present.
- `axym record add --input <record.json> --json`: validates a user-supplied proof record payload, then signs and appends it to the local chain.
- `axym ingest --source wrkr --json --input <path>`: ingests Wrkr evidence with stateful drift tracking.
- `axym ingest --source gait --json --input <path>`: ingests Gait native/proof pack artifacts with translation.
- `axym map --frameworks eu-ai-act,soc2 --json`: deterministically maps chain evidence to framework controls.
- `axym gaps --frameworks eu-ai-act,soc2 --json`: ranks `covered`, `partial`, and `gap` outcomes with remediation and effort.
- `axym regress init --baseline ./tmp/regress-baseline.json --frameworks eu-ai-act,soc2 --json`: captures deterministic baseline coverage.
- `axym regress run --baseline ./tmp/regress-baseline.json --frameworks eu-ai-act,soc2 --json`: exits `5` on coverage drift with stable control output.
- `axym review --date 2026-09-15 --json`: emits a deterministic Daily Review Pack.
- `axym override create --bundle Q3-2026 --reason "fixture" --signer ops-key --json`: appends signed override evidence and artifacts.
- `axym replay --model payments-agent --tier A --json`: emits replay-certification evidence with deterministic blast-radius summaries.
- `axym bundle --audit Q3-2026 --frameworks eu-ai-act,soc2 --json`: assembles signed audit bundles with executive summary, identity-governance artifacts, OSCAL, and portable raw records.
- `axym verify --chain --json`: verifies append-only chain integrity plus Axym-managed record signatures.
- `axym verify --bundle ./axym-evidence --json`: verifies bundle manifest signatures, Axym-authored record signatures, and compliance completeness, including identity-governance artifact consistency, without writing store-managed temp artifacts.

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
