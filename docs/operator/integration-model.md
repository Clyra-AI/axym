# Axym Integration Model

Axym does not replace your runtime, CI, provider, or upstream identity systems. It sits at the evidence boundary: reading existing signals, translating them into proof records, and then producing compliance views and audit bundles for identity-governed action in software delivery.
Gait enforces runtime controls. Wrkr inventories ownership and posture. Proof verifies signatures and chain integrity. Axym correlates, maps, and packages that evidence.
Axym does not block, freeze, broker, or sandbox execution.

See the linked diagram in [integration-boundary.mmd](integration-boundary.mmd).

## Ownership boundaries

### Customer code and infrastructure

- Own the business workflow, AI application logic, deployment topology, and environment-specific integrations.
- Emit source artifacts such as CI logs, governance events, approvals, replay outputs, and sibling system exports.

### Axym

- Collects or ingests supported evidence surfaces locally.
- Normalizes evidence into proof records and appends them to the local proof chain.
- Exposes one additive identity-governance view across native collection, manual append, Wrkr ingest, and Gait ingest.
- Maps records to frameworks, ranks gaps, and assembles audit bundles.
- Verifies chain and bundle integrity without shipping evidence to a hosted service by default.

### Tool providers and upstream systems

- Continue to own model execution, MCP behavior, CI orchestration, incident systems, and approval systems.
- Continue to own identity lifecycle, credential issuance, entitlements, and interactive access control when IAM, PAM, or IGA systems are present.
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
- Best for explicit approvals, risk assessments, or other high-signal records that already exist in a structured form and already know actor/downstream/owner/policy linkage.
- Authoritative contract: [../../schemas/v1/record/README.md](../../schemas/v1/record/README.md).

### Sibling ingest

- Invoked with `./axym ingest --source wrkr --input <path> --json` or `./axym ingest --gait-pack <path> --json`.
- Best when you already have compatible evidence or translated packs from other Clyra products and need one normalized identity-chain view across them.
- `--gait-pack` is the source-of-truth path for Gait authorization bundles and structured control artifacts.
- Gait v1.5 lifecycle evidence requires `--gait-lifecycle-verification <config.json>` containing caller-owned trusted-key, exact lineage, digest, time-window, and fixture-policy fields; Axym never derives verification authority from the lifecycle pack. The config contract is [versioned and strict](../../schemas/v1/gait/lifecycle-verification-config-v1.schema.json), with a [safe illustrative example](../../schemas/v1/gait/lifecycle-verification-config.example.json).
- Authoritative lifecycle ingestion attaches a signed, digest-bound verification receipt before the aggregate is appended. Manual `record add` cannot assert Gait lifecycle metadata, and bundle/governance projections reverify the receipt and persisted record hash before promoting execution, effect, or containment axes.
- Lifecycle registry commit is coupled to the append boundary with rollback on registry failure, so an authoritative chain record is not left without its durable verification receipt.

## Public surface notes

- Stable today: built-in collection, plugin collection, manual record append, sibling ingest, and `map`/`gaps`/`bundle`/`verify`.

### Action Contract compatibility

- Axym can consume exactly one Wrkr v3 `proposed_action_contract` with `./axym action-contract consume <path> --json`.
- Exact producer-native proposal bytes and reference envelopes can be retained in the managed store with `--store-dir`; bundles carry and verify those bytes, envelopes, governed register, signed packet, timeline, and graph.
- The consumer preserves producer-native bytes, IDs, revisions, supersession, authority/precondition/confirmation/compensation fields, and evidence references.
- A proposal is report-only evidence, never execution authority. `context_only` activation is explicitly non-binding; `enforce_floor` conformance reports structural preservation/tightening only.
- Gait `activated_action_contract` artifacts are validated against the Axym-owned schema and can be verified with the typed `core/ingest/actioncontract` package when the activation public key and exact proposal bytes are supplied.
- Activation verification also requires current-selection evidence for the exact family/revision and an explicit UTC evaluation time. Development-signed activations remain unverifiable even when a test-only allowance is supplied.
- `enforce_floor` is not interpreted as tightening by mode alone: only an explicit value projection can prove `tightened`; otherwise the exact proposal binding is reported conservatively.
- Exact verified Gait lifecycle artifacts now project execution, effect, containment, compensation, freshness, and correlation evidence into the governed packet. Telemetry and Judge projections remain advisory, non-authoritative, and explicitly labelled; they never grant execution authority. See [the versioned schema contract](../../schemas/v1/action_contract/README.md).
- Internal detail: package names, workflow step ordering, and helper placement are not public extension points.
- Deprecated surface: none documented in launch docs today.

## Sync vs async operator flows

### Sync paths

- `collect --dry-run` for immediate environment and would-capture validation.
- `collect` against local or mounted artifacts that are available at command time.
- `record add` when an operator or workflow already has the exact proof payload to append and wants Axym to validate/sign/link it locally.
- `record add` uses Axym's public manual-input envelope while `Clyra-AI/proof` owns the shared proof-record semantics and record-type-specific validation rules.

### Async paths

- CI, deployment, dbt, or Snowflake artifacts written out by upstream systems and collected later.
- Wrkr and Gait exports that are ingested after the source system run completes.
- Governance-event files emitted by another runtime component and collected in a later step.

## Choosing the right path first

- Start with the `smoke test` when you need install and command-surface confidence.
- Use the `sample proof path` when you need a supported offline first-value demo that ends in evidence, ranked gaps, and intact local verification rather than full audit completeness.
- Move to the `real integration path` once you are ready to connect actual runtime, CI, plugin, manual, or sibling evidence sources.

## Failure handling

- Zero capture on clean-room `collect --json` is expected when no real inputs are present.
- Per-source `reason_codes` explain empty, degraded, or failed collection paths.
- `map` and `gaps` stay deterministic even when the result is incomplete.
- `verify --chain` validates append-only local integrity and Axym-managed record signatures.
- `verify --bundle` validates portable bundle manifest signatures, Axym-authored record signatures, and reports compliance completeness explicitly, including identity-governance artifact consistency.
- When present, it recomputes `authorization-register.json`, `insurance-evidence-profile.json`, `credential-posture-register.json`, `freeze-window-coverage.json`, `kill-switch-coverage.json`, `enforcement-explain-register.json`, `sandbox-coverage.json`, and `control-maturity.json`.
