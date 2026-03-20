# Axym

Axym is a deterministic CLI for platform, security, and GRC engineers who need local proof of identity-governed action in software delivery, compliance mapping, and audit-ready bundles.

## Where it fits

Axym sits after your runtime or CI evidence sources. Your code, providers, MCP servers, sibling systems, and IAM/PAM/IGA systems stay upstream; Axym collects, ingests, or appends structured evidence locally, then maps it and bundles it for audit use around the action-governance seam.

Operator walkthroughs live in [../../../docs/operator/quickstart.md](../../../docs/operator/quickstart.md) and [../../../docs/operator/integration-model.md](../../../docs/operator/integration-model.md).

## Install

- Homebrew: `brew install Clyra-AI/tap/axym` then `./axym version --json`
- Source: `go build ./cmd/axym` then `./axym version --json`
- Release binary: `./axym version --json`

## Smoke test

- `./axym init --json`
- `./axym collect --dry-run --json`

Expected outcome:

- `init` creates the local store and default policy.
- `collect --dry-run` reports deterministic would-capture output without writes.
- A clean environment may still produce `captured: 0` on plain `collect --json`; that is a smoke test, not the supported first-value path.

## Sample proof path

- `./axym init --sample-pack ./axym-sample --json`
- `./axym collect --json --governance-event-file ./axym-sample/governance/context_engineering.jsonl`
- `./axym record add --input ./axym-sample/records/approval.json --json`
- `./axym record add --input ./axym-sample/records/risk_assessment.json --json`
- `./axym map --frameworks eu-ai-act,soc2 --json`
- `./axym gaps --frameworks eu-ai-act,soc2 --json`
- `./axym bundle --audit sample --frameworks eu-ai-act,soc2 --json`
- `./axym verify --chain --json`

Expected outcome:

- The sample pack is created locally with no network access and no repo fixture dependency.
- The sample journey yields 5 covered controls out of 6 across `eu-ai-act,soc2`.
- `gaps` returns grade `C`, not grade `F`.
- `bundle` emits identity-governance artifacts and `verify --chain --json` succeeds with an intact 5-record chain.

## Real integration path

- Built-in collectors: `mcp`, `llmapi`, `webhook`, `githubactions`, `gitmeta`, `dbt`, `snowflake`, and `governanceevent`.
- Plugin collectors: `./axym collect --json --plugin "<cmd>"`.
- Manual record append: `./axym record add --input <record.json> --json`.
- Sibling ingest: `./axym ingest --source wrkr --input <path> --json` and `./axym ingest --source gait --input <path> --json`.

Public launch docs should not describe approvals, risk assessments, guardrails, incidents, or similar surfaces as default built-in clean-room capture unless that collector ships.
Public launch docs should also not position Axym as an IAM/PAM/IGA replacement or widen the wedge beyond software delivery.

## Commands

- `./axym collect --dry-run --json`: validates deterministic would-capture output with no writes.
- `./axym init --json`: initializes local store scaffolding and writes `axym-policy.yaml` defaults.
- `./axym init --sample-pack ./axym-sample --json`: additively materializes a deterministic sample pack with created files and next-step commands.
- `./axym collect --json`: runs built-in collectors and appends signed proof records from configured sources.
- `./axym record add --input <record.json> --json`: validates a proof record payload, then signs and appends it to the local chain with deterministic dedupe behavior.
- `./axym collect --json --plugin "<cmd>"`: runs third-party collector protocol (`stdin` config, `stdout` normalized collector JSONL with optional `relationship`).
- `./axym collect --json --governance-event-file <file.jsonl>`: ingests governance events and promotes valid events to proof records with actor/downstream/owner/policy/approval linkage when present.
- `./axym map --frameworks eu-ai-act,soc2 --json`: deterministically maps chain evidence to framework controls and emits explainable match rationale.
- `./axym gaps --frameworks eu-ai-act,soc2 --json`: computes deterministic `covered`/`partial`/`gap` ranking, remediation guidance, and auditability grade.
- `./axym map --policy-config ./axym-policy.yaml --json`: applies schema-validated policy defaults and threshold constraints; invalid policy input exits `6`.
- `./axym regress init --baseline ./tmp/regress-baseline.json --frameworks eu-ai-act,soc2 --json`: captures deterministic baseline coverage snapshots for regression checks.
- `./axym regress run --baseline ./tmp/regress-baseline.json --frameworks eu-ai-act,soc2 --json`: reports deterministic coverage drift and exits `5` when controls regress.
- `./axym review --date 2026-09-15 --json`: emits deterministic daily review pack output with exception classes and grade distribution summaries.
- `./axym override create --bundle Q3-2026 --reason "<reason>" --signer ops-key --json`: appends signed override artifacts and proof evidence.
- `./axym ingest --source wrkr --input <path> --json`: ingests sibling evidence from Wrkr/Gait payloads with deterministic append/dedupe/reject counters.
- `./axym replay --model payments-agent --tier A --json`: emits deterministic replay-certification evidence for review workflows.
- `./axym bundle --audit <name> --frameworks eu-ai-act,soc2 --json`: assembles deterministic signed audit bundles with executive summary (`.json` + `.pdf`), identity-governance artifacts, chain verification, and OSCAL export.
- `./axym map --json` and `./axym gaps --json`: default to frameworks `eu-ai-act,soc2` when `--frameworks` is omitted.
- `./axym map --frameworks eu-ai-act --min-coverage 0.80 --json`: enforces threshold policy and exits non-zero when coverage is below threshold.
- `./axym verify --chain --json`: verifies local append-only chain integrity plus Axym-managed record signatures.
- `./axym verify --bundle <path> --json`: verifies bundle manifest signatures, Axym-authored record signatures, and deterministic bundle compliance-completeness checks, including identity-governance artifact consistency, without writing store-managed temp artifacts.

## Contributor gates

- Fast local: `make lint-fast`, `make test-fast`, `make test-contracts`
- Extended local: `make lint-go`, `make test-security`, `make test-docs-links`, `make prepush-full`
- Hosted CI remains authoritative for required PR checks and GitHub-hosted CodeQL

## Release verification

- `./scripts/release_go_nogo.sh --dist-dir dist --binary-name axym`

## Exit codes

- `0` success
- `1` runtime failure
- `2` verification failure
- `3` policy/schema violation
- `4` approval required
- `5` regression drift (including threshold failures in `map`/`gaps`)
- `6` invalid input
- `7` dependency missing
- `8` unsafe operation blocked
