# Axym Integration Model

Axym does not replace your runtime, CI, or provider. It sits at the evidence boundary: reading existing signals, translating them into proof records, and then producing compliance views and audit bundles.

See the linked diagram in [integration-boundary.mmd](integration-boundary.mmd).

## Ownership boundaries

### Customer code and infrastructure

- Own the business workflow, AI application logic, deployment topology, and environment-specific integrations.
- Emit source artifacts such as CI logs, governance events, approvals, replay outputs, and sibling system exports.

### Axym

- Collects or ingests supported evidence surfaces locally.
- Normalizes evidence into proof records and appends them to the local proof chain.
- Maps records to frameworks, ranks gaps, and assembles audit bundles.
- Verifies chain and bundle integrity without shipping evidence to a hosted service by default.

### Tool providers and upstream systems

- Continue to own model execution, MCP behavior, CI orchestration, incident systems, and approval systems.
- Provide the raw or structured artifacts that Axym reads through built-in collectors, plugins, manual append, or sibling ingest.

## Evidence path types

### Built-in collection

- Invoked with `./axym collect --json`.
- Supports the shipped built-in collectors only: `mcp`, `llmapi`, `webhook`, `githubactions`, `gitmeta`, `dbt`, `snowflake`, and `governanceevent`.
- Best for sources Axym already knows how to parse deterministically.

### Plugin collection

- Invoked with `./axym collect --json --plugin "<cmd>"`.
- Best when your source is not covered by a built-in collector but can emit deterministic JSONL proof-candidate output.

### Manual record append

- Invoked with `./axym record add --input <record.json> --json`.
- Best for explicit approvals, risk assessments, or other high-signal records that already exist in a structured form.

### Sibling ingest

- Invoked with `./axym ingest --source wrkr --input <path> --json` or `./axym ingest --source gait --input <path> --json`.
- Best when you already have compatible evidence or translated packs from other Clyra products.

## Sync vs async operator flows

### Sync paths

- `collect --dry-run` for immediate environment and would-capture validation.
- `collect` against local or mounted artifacts that are available at command time.
- `record add` when an operator or workflow already has the exact proof payload to append.

### Async paths

- CI, deployment, dbt, or Snowflake artifacts written out by upstream systems and collected later.
- Wrkr and Gait exports that are ingested after the source system run completes.
- Governance-event files emitted by another runtime component and collected in a later step.

## Choosing the right path first

- Start with the `smoke test` when you need install and command-surface confidence.
- Use the `sample proof path` when you need a supported offline first-value demo that ends in non-empty evidence.
- Move to the `real integration path` once you are ready to connect actual runtime, CI, plugin, manual, or sibling evidence sources.

## Failure handling

- Zero capture on clean-room `collect --json` is expected when no real inputs are present.
- Per-source `reason_codes` explain empty, degraded, or failed collection paths.
- `map` and `gaps` stay deterministic even when the result is incomplete.
- `verify --chain` validates append-only local integrity.
- `verify --bundle` validates portable bundle integrity and reports compliance completeness explicitly.

