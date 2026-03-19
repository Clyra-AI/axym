# Axym — PRD v1

| Field | Value |
|-------|-------|
| Version | 1.1 |
| Status | Execution-ready |
| Owner | Product and Engineering |
| Last Updated | 2026-02-28 |

-----

## Executive Summary

> Launch-surface note (2026-03-19): this PRD describes the broader Axym product direction. The current OSS launch contract is narrower and must stay truthful in public docs. Built-in collection today is limited to `mcp`, `llmapi`, `webhook`, `githubactions`, `gitmeta`, `dbt`, `snowflake`, and `governanceevent`, plus explicit plugin/manual/sibling-ingest paths. Clean-room `collect --json` is a smoke test, not the supported first-value path. The supported install-time first-value journey is the local offline sample proof path introduced by `init --sample-pack <dir>`.

Axym is an open-source Go CLI that captures structured proof of AI system behavior and produces audit-ready compliance packages mapped to EU AI Act, SOC 2, SOX, PCI-DSS, and state AI regulations. It intercepts AI agent activity at the integration layer — CI/CD pipelines, MCP server calls, API gateways, and tool invocations — and produces signed, tamper-evident proof records that are automatically mapped to specific regulatory controls.

**Primary audiences:** The primary buyer is the **Head of GRC or Chief Compliance Officer** who must produce AI governance evidence for regulators and auditors. The primary user is the **GRC analyst or compliance engineer** who assembles audit packages. The champion is the **security or platform engineer** who integrates Axym into the pipeline.

**Why this exists:** Regulatory deadlines have converged — EU AI Act broad enforcement (August 2026), Colorado AI Act (effective February 1, 2026), Texas TRAIGA (effective September 1, 2025) — and all require technical evidence of AI system governance. The evidence doesn't exist in any structured form today. AI agent actions are scattered across logs, LLM API responses, git diffs, and ephemeral tool outputs. Axym transforms these operational signals into compliance artifacts.

**Differentiation:** GRC platforms (Vanta, Drata) manage compliance workflows but cannot generate technical evidence that AI systems are governed. LLM observability tools (LangSmith, Arize) capture operational telemetry but are not structured for compliance. Axym produces the evidence; GRC platforms and observability tools are complementary, not competitive.

**Positioning within Clyra AI:** Axym is the "Prove" step in See (Wrkr) -> Prove (Axym) -> Control (Gait). Axym is the product; the primitive underneath is `Clyra-AI/proof`. Axym imports Proof and adds compliance opinions on top: collection, framework mapping, gap detection, and bundle generation. Wrkr scan findings and Gait enforcement decisions flow into Axym as native proof records for unified compliance mapping. The products are independently useful but form a closed loop together.

-----

## Problem Statement

### The pain (stated by the buyer in their own words):

- **Head of GRC**: "The EU AI Act requires evidence of risk management, transparency, and human oversight for high-risk AI systems. My team has been maintaining a spreadsheet. The auditor wants technical controls and proof they're working. I have nothing machine-readable."
- **CISO**: "The board asked if we're AI Act compliant. I said we have policies. They asked if we can prove compliance under audit. I said I'd get back to them. That was three months ago."
- **VP Engineering**: "I'm told we need audit trails for our AI agents. Our agents log to CloudWatch. Nobody knows what the compliance team actually needs from those logs, and the compliance team can't read CloudWatch."
- **GRC Analyst**: "I spent three weeks manually reconstructing what our AI agent did for the auditor. I pulled API logs, git commits, Slack messages, and approval emails. I assembled it into a PDF. The auditor said it wasn't structured enough to map to controls. I need this automated yesterday."
- **External Auditor**: "My clients tell me they govern their AI. When I ask for evidence, they show me a policy PDF and logs I can't interpret. I need structured records I can map to control objectives, with integrity verification."

### The job to be done (JTBD):

**When** I'm responsible for AI compliance at an organization deploying AI agents and AI-assisted development tools in production, **I want to** automatically capture structured evidence of AI system behavior, decisions, and controls, and map that evidence to specific regulatory requirements, **so that** I can prove compliance when audited, identify governance gaps before the regulator does, and reduce audit preparation from weeks to hours.

### Why this wins (mapping to historical pattern):

| Pattern | Cloud/Container Precedent | Axym Application |
|---|---|---|
| Regulatory forcing function | SOX / PCI-DSS → cloud audit tools | EU AI Act / state AI laws → AI compliance evidence |
| Evidence > dashboards | SOC 2 evidence platforms (Vanta) | AI audit evidence bundles (Axym) |
| Neutral third party | AWS didn't build Vanta | OpenAI/Anthropic won't build cross-platform compliance evidence |
| Existing budget | GRC/audit budget (already allocated) | Same budget, new requirement (AI-specific controls) |
| Bottom-up distribution | Snyk CLI adopted by devs, bought by CISOs | `axym collect` adopted by engineers, funded by GRC |
| Artifact-first moat | Sigstore attestations became the standard | Proof records become the format auditors expect |
| Discovery before enforcement | Wiz: "see risk before changing anything" | `axym gaps`: see compliance gaps before the auditor does |

### The regulatory landscape (why the timing is now):

Regulatory effective dates in this section follow `regulatory_source.md` (canonical date source for all docs in this repo).

| Regulation | Effective | Key Evidence Requirements |
|---|---|---|
| EU AI Act (broad enforcement) | August 2, 2026 | Risk management systems, technical documentation, logging/traceability, human oversight mechanisms, accuracy/robustness measures |
| Texas TRAIGA | September 1, 2025 | Disclosure of AI-generated decisions, impact assessments, records of consequential decisions |
| Colorado AI Act | February 1, 2026 | Risk management policy, impact assessments, records of high-risk AI interactions |
| NIST AI 600-1 (guidance) | Q2 2026 | Agent identity, boundary enforcement, audit trails, containment evidence |
| SOC 2 + AI (updated guidance) | Rolling 2026 | AI-specific controls for logical access, system operations, change management |
| ISO 42001 (AI Management) | Adopted 2023, audits accelerating | AI risk assessment, performance monitoring, documentation requirements |

These aren't theoretical. Audit firms (Big 4, SOC 2 auditors) have updated their control matrices. The checklists now have AI-specific line items. Evidence is required. Most organizations have zero.

-----

## Product Overview

### One-liner:

**Axym is an open-source Go CLI that captures structured proof of AI system behavior and produces audit-ready compliance packages mapped to EU AI Act, SOC 2, SOX, PCI-DSS, and state AI regulations.**

### Core loop (the "15-minute time-to-value"):

```
Install → Connect → Collect → Map → Gaps → Bundle
  brew     agent      capture   compliance  find     signed
  install  frameworks proof     framework   missing  evidence
  Clyra-AI/tap/  + CI       records   mapping     controls package
  axym
```

### What Axym captures (the "evidence surface"):

**Layer 1: Agent Runtime Evidence (primary — this is the moat)**

| Signal | Source | What Axym Captures |
|---|---|---|
| Agent tool invocations | MCP server logs, tool call records | What tools the agent called, with what parameters, what was returned, timestamps |
| Agent decisions | LLM API responses (structured extraction) | Decision points, selected actions, reasoning artifacts (if available) |
| Agent permissions at execution time | Agent config files, runtime permission checks | What the agent was allowed to do vs. what it attempted |
| Guardrail activations | Guardrail frameworks (NeMo, Guardrails AI, custom) | When guardrails triggered, what they blocked, what they allowed |
| Human oversight events | Approval workflows, review records | Who approved what, when, and what they reviewed before approving |
| Agent boundary violations | Runtime monitoring, error logs | Attempts to exceed permissions, blocked actions, escalation events |
| Agent compiled actions | PTC-style compiled scripts, multi-step plans | Compound action hash, tool list, composite risk classification, execution plan structure. As model providers ship compiled/programmatic tool calling (LLM emits executable scripts instead of individual tool calls), the governance unit shifts from per-tool-call to per-script. Axym captures the plan (compiled action) and links it to the execution traces that follow |

**Layer 2: Development Lifecycle Evidence (secondary — enrichment)**

| Signal | Source | What Axym Captures |
|---|---|---|
| AI code authorship | Git metadata, AI tool config | Which code was AI-generated vs. human-written, review status |
| Model version and config | Deployment manifests, API configs | Which model version was running, what system prompts were active |
| Testing and evaluation records | CI/CD pipelines, eval frameworks | Test results, evaluation scores, benchmark data for AI components |
| Change management | Git history, deployment logs | Who deployed what AI system changes, when, with what approvals |

**Layer 3: Data Pipeline Evidence (from Clyra DNA — SOX/PCI-DSS)**

| Signal | Source | What Axym Captures |
|---|---|---|
| Data pipeline runs | dbt logs, orchestrator events | Git SHA, models executed, tables touched, Separation of Duties (SoD) roles |
| Query execution | Snowflake query history (digests only) | Query hashes (not queries), target tables, duration, row counts |
| Pipeline approvals | Change management integrations | Who approved the pipeline change, when, review evidence |
| Replay certification | Axym's replay engine | Replay tier (A/B/C), pass/fail, blast radius classification |

**Layer 4: External Tool Evidence (eval platforms, model registries, adversarial testing)**

| Signal | Source | What Axym Captures |
|---|---|---|
| Eval / benchmark results | Braintrust, Arize, LangSmith, custom eval harnesses | Evaluation scores, benchmark pass/fail, dataset version, model version under test |
| Model lifecycle events | MLflow, Hugging Face Hub, Weights & Biases, deployment pipelines | Model registration, version promotion, fine-tuning runs, checkpoint approvals, weight hash |
| Adversarial / red team results | Giskard, NVIDIA NeMo Red Team, custom red team tooling | Attack vectors tested, failure modes discovered, severity, affected model/agent, remediation status |

These surfaces are ingested through Axym's collector plugin architecture (FR12). Each collector reads existing outputs from the external tool — Axym never requires modifications to the source system. The records produced are standard proof records (`test_result`, `model_change`, `deployment`, `risk_assessment`) that flow into the same compliance mapping and evidence chain as all other Axym evidence. Eval results map to EU AI Act Article 15 (accuracy/robustness). Model provenance maps to Article 12 (record-keeping) and Article 13 (transparency). Red team results map to Article 9 (risk management) and Article 15.

**Layer 5: Organizational Controls Evidence (tertiary — compliance completeness)**

| Signal | Source | What Axym Captures |
|---|---|---|
| Risk assessment records | Risk register integrations, manual input | Documented risk assessments for AI systems, severity ratings |
| Training and awareness | LMS integrations, attestation records | Staff training on AI governance, acknowledgment records |
| Incident response records | Incident management tools, manual input | AI-related incidents, response actions, resolution evidence |
| Policy attestation | Document management, signature records | Current AI policies, review dates, acknowledgments |

### What Axym produces:

**1. Proof Records (the atomic unit)**

Every captured signal becomes a structured, signed proof record via `Clyra-AI/proof`:

```yaml
# Example: Single proof record produced by Axym
record_id: "prf-2026-09-15T10:30:00Z-a7f3b2c1"
record_version: "1.0"
timestamp: "2026-09-15T10:30:00Z"
source: "axym-mcp-collector"
source_product: "axym"
agent_id: "claude-code-payments-service"
record_type: "tool_invocation"

event:
  tool: "postgres_query"
  action: "SELECT"
  parameters:
    query_hash: "sha256:abc123..."  # query hash, not the query itself
    target: "payments.transactions"
    access_level: "read-only"
  result:
    status: "success"
    rows_returned: 142
    duration_ms: 230

controls:
  permissions_enforced: true
  approved_scope: "read-only on payments.*"
  within_scope: true
  guardrails_active:
    - name: "pii-filter"
      status: "active"
      triggered: false
    - name: "row-limit"
      status: "active"
      triggered: false
  human_oversight:
    approval_required: false   # read-only action per policy
    last_review: "2026-09-10T14:00:00Z"
    reviewer: "sam@acme.corp"

metadata:
  # Extensible key-value pairs. No compliance opinions in the record itself.
  # Compliance tagging is applied by axym when mapping records to frameworks.
  environment: "production"
  deployment_version: "v2.3.1"

integrity:
  record_hash: "sha256:def456..."
  previous_record_hash: "sha256:ghi789..."  # chain integrity
  signing_key_id: "clyra:prod-key-01:20260701"
  signature: "base64:..."
```

This is a `proof.Record` from `Clyra-AI/proof`. The record schema, hash chain, signing protocol, and canonicalization spec are defined in `Clyra-AI/proof` and shared across all Clyra AI products. Axym produces proof records; it does not define the format.

**2. Proof Chain (the integrity mechanism)**

Records are chained via `Clyra-AI/proof` — each record's hash includes the previous record's hash, creating a tamper-evident sequence. If a record is modified or deleted, the chain breaks and verification flags it. This is not blockchain — it's a simple hash chain, the same mechanism used in certificate transparency logs. Boring, proven, auditable.

```
Record 1 → hash(record_1) →┐
Record 2 → hash(record_2 + prev_hash) →┐
Record 3 → hash(record_3 + prev_hash) →┐
...
Verification: axym verify --chain → delegates to proof.VerifyChain()
              proof verify --chain → "Chain intact. 1,427 records. No gaps."
```

**3. Compliance Map (the "how are we doing?" view)**

A structured report that maps collected proof records to specific regulatory controls. Framework definitions come from `Clyra-AI/proof/frameworks/`. Compliance mapping logic — matching records to controls, calculating coverage, detecting gaps — is Axym's domain.

```
┌──────────────────────────────────────────────────────────────┐
│ AXYM COMPLIANCE MAP — acme-corp — 2026-09-15                 │
│ Framework: EU AI Act (High-Risk Systems)                     │
├──────────────────────────────┬──────────┬──────────┬─────────┤
│ Requirement                  │ Status   │ Evidence │ Gaps    │
├──────────────────────────────┼──────────┼──────────┼─────────┤
│ Art. 9: Risk Management      │ PARTIAL  │ 342 recs │ 2 gaps  │
│  └ 9(2)(a) Risk identification│ COVERED │ 45 recs  │ —       │
│  └ 9(2)(b) Risk estimation    │ COVERED │ 89 recs  │ —       │
│  └ 9(2)(c) Risk evaluation    │ GAP     │ 0 recs   │ No eval │
│  └ 9(2)(d) Risk treatment     │ PARTIAL │ 208 recs │ 1 gap   │
│ Art. 12: Record-Keeping      │ COVERED  │ 1,427    │ —       │
│ Art. 13: Transparency        │ PARTIAL  │ 89 recs  │ 1 gap   │
│ Art. 14: Human Oversight     │ COVERED  │ 234 recs │ —       │
│ Art. 15: Accuracy/Robustness │ GAP      │ 12 recs  │ 3 gaps  │
├──────────────────────────────┴──────────┴──────────┴─────────┤
│ Auditability Grade: B                                        │
│ (Missing: risk evaluation records, 4 transparency disclosures, 3 accuracy gaps) │
└──────────────────────────────────────────────────────────────┘

Top Gaps:
  1. Art. 9(2)(c): No AI risk evaluation records found. Need documented
     risk evaluation methodology and periodic evaluation results.
  2. Art. 15(1): Only 12 accuracy benchmark records. Need continuous
     performance monitoring evidence for production AI systems.
  3. Art. 13(3)(d): Missing disclosure records for 4 of 7 customer-facing
     AI interactions.
```

**4. Audit Bundle (the deliverable)**

A portable, signed directory that a GRC analyst hands directly to the auditor. The bundle contains proof records verifiable with the standalone `proof verify` CLI — the auditor does not need Axym installed.

```
axym-evidence/
├── manifest.json                  # All files with SHA-256 hashes, schema versions, anchor refs
├── chain-verification.yaml        # Hash chain integrity verification
├── auditability-grade.yaml        # Overall grade (A-F) with per-section breakdown
├── boundary-contract.md           # Plain-language: what Axym proves vs. customer responsibility
├── summary/
│   ├── executive-summary.pdf      # One-page for the board
│   ├── compliance-map.yaml        # Framework coverage overview
│   └── oscal-v1.1/
│       └── component-definition.json  # OSCAL mapping to control objectives
├── evidence/
│   ├── eu-ai-act/
│   │   ├── article-9-risk-management/
│   │   │   ├── control-mapping.yaml
│   │   │   ├── proof-records.jsonl     # 342 linked records
│   │   │   └── gaps.yaml
│   │   ├── article-12-record-keeping/
│   │   │   ├── control-mapping.yaml
│   │   │   └── proof-records.jsonl     # 1,427 linked records
│   │   ├── article-13-transparency/
│   │   │   ├── control-mapping.yaml
│   │   │   ├── proof-records.jsonl
│   │   │   └── gaps.yaml
│   │   ├── article-14-human-oversight/
│   │   │   ├── control-mapping.yaml
│   │   │   └── proof-records.jsonl
│   │   └── article-15-accuracy-robustness/
│   │       ├── control-mapping.yaml
│   │       ├── proof-records.jsonl
│   │       └── gaps.yaml
│   ├── soc2/
│   │   ├── cc6-logical-access/
│   │   ├── cc7-system-operations/
│   │   └── cc8-change-management/
│   ├── sox/
│   │   ├── change-management/
│   │   ├── separation-of-duties/
│   │   └── access-controls/
│   └── state/
│       ├── texas-traiga/
│       └── colorado-ai-act/
├── raw-records/
│   ├── records-2026-09-01-to-2026-09-15.jsonl
│   └── chain.json                 # Full hash chain metadata
├── overrides/
│   └── override-*.yaml            # Signed exception artifacts (if any)
├── replay-certs/
│   └── replay-cert-*.yaml         # Replay certification records with tier/blast radius
├── methodology.yaml               # How evidence was collected, what was in scope
├── retention-matrix.json          # Retention policy per environment (dev/test/prod)
└── signatures/
    ├── bundle-signature.sig       # Ed25519 signature of entire bundle
    └── public-key.pem             # Verification key
```

**5. Gap Remediation Plan (the "fix it" path)**

For each identified compliance gap, Axym produces a structured remediation recommendation:

```yaml
gaps:
  - id: "gap-eu-ai-act-9-2-c"
    framework: "eu-ai-act"
    control: "article-9(2)(c)"
    title: "AI risk evaluation not documented"
    severity: "high"
    description: |
      Article 9(2)(c) requires documented risk evaluation methodology
      and periodic evaluation results. No evaluation records found in
      the proof chain for any production AI system.
    remediation:
      action: "Implement periodic AI risk evaluation"
      steps:
        - "Define risk evaluation methodology (template provided)"
        - "Run first evaluation for each production AI system"
        - "Add evaluation results to Axym evidence pipeline"
        - "Schedule quarterly re-evaluation in CI"
      template: "templates/risk-evaluation.yaml"
      estimated_effort: "2-3 days for initial setup"
    evidence_needed:
      - type: "risk_evaluation"
        minimum_frequency: "quarterly"
        required_fields: ["event.risk_category", "event.risk_severity", "controls.human_oversight"]
```

-----

## User Personas

### Primary Buyer: Head of GRC / Chief Compliance Officer (the "sponsor")

**Name archetype:** "Priya, Head of GRC at a 2,000-person fintech"

Priya reports to the General Counsel. Her team of 4 manages SOC 2, PCI-DSS, GDPR, and now the EU AI Act. Her auditor delivered the updated control matrix in June with 23 new AI-specific control objectives. Her team has been manually assembling evidence for two months. The board asked for an AI compliance status update at the last meeting. Priya showed a heat map that was mostly yellow and red.

**Priya's interaction with Axym:**

1. Sees the compliance gap report generated by her engineering team
2. Reviews the executive summary PDF
3. Presents to the board: "We have 73% coverage against EU AI Act requirements. Here are the 6 gaps and our remediation timeline."
4. Hands the audit bundle to the external auditor
5. Approves Clyra Platform budget for continuous evidence collection and dashboard
6. Adds "Axym evidence review" to monthly GRC cadence

### Primary User: GRC Analyst / Compliance Engineer (the "assembler")

**Name archetype:** "Jordan, Senior GRC Analyst"

Jordan is the person who actually builds audit packages. Before Axym, Jordan spent 3 weeks per audit cycle manually reconstructing AI system activity from logs, emails, and Slack messages. Jordan doesn't have access to production systems. Jordan knows what controls the auditor needs but can't get engineering to produce evidence in a usable format.

**Jordan's workflow with Axym:**

1. Receives proof records from engineering's Axym pipeline
2. Runs `axym map --frameworks eu-ai-act,soc2` to see coverage
3. Identifies gaps: "We're missing risk evaluation records and 4 transparency disclosures"
4. Files tickets with engineering for the gaps (Axym generates the ticket templates)
5. Runs `axym bundle --audit Q3-2026 --frameworks eu-ai-act,soc2` to generate the audit package
6. Reviews bundle, adds any manual proof records for non-automated controls
7. Delivers bundle to auditor
8. Auditor runs `proof verify --bundle ./axym-evidence/` — independently confirms integrity

### Champion: Security/Platform Engineer (the "integrator")

**Name archetype:** "Kai, Senior Platform Engineer"

Kai owns the CI/CD pipeline and developer tooling. The GRC team has been asking Kai for "AI audit logs" for months. Kai has been pointing them at CloudWatch and getting blank stares. Kai needs something that captures the right signals from the right places and produces output the compliance team can actually use — without Kai becoming a full-time compliance analyst.

**Kai's workflow with Axym:**

1. `brew install Clyra-AI/tap/axym`
2. `axym init` — configure evidence sources (CI pipeline, MCP servers, API gateway)
3. `axym collect --dry-run` — see what would be captured without writing anything
4. `axym collect` — start capturing proof records
5. Adds `axym collect` to CI pipeline as a post-deploy step
6. Adds `axym verify --chain` to weekly cron (integrity check)
7. Tells Jordan: "Your evidence is in `./evidence/`. Run `axym bundle` whenever you need it."
8. Never thinks about compliance again unless Axym alerts on a broken chain

Kai already runs Wrkr for AI tool discovery. Wrkr's scan findings are proof records in the same format. Kai adds `wrkr evidence` output to the same evidence directory — unified evidence chain, zero format translation.

### External User: Auditor (the "verifier")

**Name archetype:** "Deepa, Senior Manager at Big 4 audit firm"

Deepa audits 15+ clients per year. Since Q2 2026, her control matrix includes AI-specific objectives. Most clients hand her policy documents and CloudWatch screenshots. One client hands her an Axym bundle.

**Deepa's interaction with Axym:**

1. Receives `axym-evidence/` directory from Jordan
2. Opens `summary/executive-summary.pdf` — sees scope, coverage, gaps at a glance
3. Opens `evidence/eu-ai-act/article-12-record-keeping/control-mapping.yaml` — sees exactly how evidence maps to the control objective
4. Runs `proof verify --bundle ./axym-evidence/` — confirms cryptographic integrity of the proof chain (uses the standalone `proof` CLI, not Axym)
5. Samples 50 records from the proof chain and spot-checks against the control requirements
6. Closes the AI portion of the audit in 3 days instead of 3 weeks
7. Tells three other clients about Axym

-----

## Functional Requirements (LVP v1)

### FR1: Evidence Collection from Agent Runtime

- Capture tool invocation records from MCP server interactions (parameters, results, timestamps)
- Capture agent decision records from structured LLM API responses (action selected, alternatives considered if available)
- Capture guardrail activation records (what triggered, what was blocked/allowed)
- Capture permission enforcement records (what was attempted vs. what was allowed)
- Support collection from: MCP server logs (required), LLM API middleware (required), custom webhook (required), CI/CD pipeline events (required)
- Collection is non-blocking — never interferes with agent operation
- All collected events are written as proof records via `proof.NewRecord()` with deterministic schemas

### FR2: Evidence Collection from Development Lifecycle

- Capture deployment events for AI system components (what was deployed, by whom, with what approvals)
- Capture model version changes (which model, which version, which configuration)
- Capture test/evaluation results for AI components from CI pipelines
- Capture code review records for AI-generated code (who reviewed, what was the review status)
- Capture model lifecycle events from model registries (MLflow, Hugging Face Hub, Weights & Biases): model registration, version promotion, fine-tuning completion, checkpoint approval, weight file hash
- Capture eval and benchmark results from eval platforms (Braintrust, Arize, LangSmith, custom harnesses): evaluation scores, benchmark pass/fail, dataset version, model version under test, regression from previous eval
- Capture adversarial testing and red team results (Giskard, NVIDIA NeMo Red Team, custom tooling): attack vectors tested, failure modes discovered, severity classification, affected model/agent, remediation status

### FR3: Evidence Collection from Data Pipelines (Clyra DNA)

- Capture dbt pipeline runs: Git SHA, models executed, tables touched, SoD roles, policy version
- Capture Snowflake query history as canonicalized digests (SQL canonicalization from `proof.Canonicalize()`): query_id, start_time, end_time, sql_digest, rows_affected, warehouse_name
- Snowflake enrichment: query ACCOUNT_USAGE.QUERY_HISTORY filtered by QUERY_TAG=change_id; retry/backoff up to 15 minutes for eventual consistency; enrichment lag → `decision.pass=false` with `ENRICHMENT_LAG`; missing QUERY_TAG on matching queries → `decision.pass=false` with `MISSING_QUERY_TAG`
- Capture pipeline approval events from change management integrations
- **SoD enforcement:** derive requestor/approver/deployer from ticket + CI; validate SoD constraints (requestor ≠ deployer, approval chain ordering); SoD violation → `decision.pass=false` with `SOD_REQUESTOR_DEPLOYER`
- **Freeze window enforcement:** policy-defined deployment freeze periods; writes during freeze → `decision.pass=false` with `FREEZE_WINDOW`
- Produce `data_pipeline_run` proof records for SOX and PCI-DSS compliance with explicit `decision.pass` and enumerated `reason_codes[]`

**Replay certification (from Clyra DNA's production-hardened architecture):**

- Replay sources are always the original artifacts (dbt compiled SQL, manifest, run results) — never from the proof record itself
- Replay Bundles are encrypted with customer KMS and stored in customer storage; proof records hold hash pointers only
- Execute replay in customer-defined sandbox (e.g., `CLYRA_REPLAY_DB.CLYRA_SANDBOX` for Snowflake; scratch schema for Postgres); never replay into production
- Compare rowcount + digests between original and replay; emit signed `replay_certification` proof record with blast_radius summary (tables_touched, row_digest_counts, dependency_graph_summary)
- **Replay Tiers (shown in bundle and compliance map):**
  - **Tier A (Strong):** deterministic baseline — Snowflake TIME TRAVEL ≥7 days + snapshot at T0; bit-identical match
  - **Tier B (Representative):** seeded fixture; representative equivalence
  - **Tier C (Advisory):** anchors/order only; no fixture available
- DB replay: whitelisted DML only; DDL recorded as advisory (digest only, non-replayable)
- HTTP replay: fetch encrypted stub, simulate responses, compare status/size/digests
- LLM replay: metadata-only verification (provider/model allowlist check); no semantic guarantee

### FR4: Proof Record Integrity

- Every proof record includes: unique ID, timestamp, source, source product, event data, control evidence, cryptographic hash, chain link, signature
- **Graph-linkable relationship envelope (implemented in `Clyra-AI/proof`, H0 data architecture for H3 PolicyGraph).** Every proof record includes an optional `relationship` field (`proof.Relationship`) carrying graph-structured context. The envelope is already implemented in proof, Gait, and Wrkr — Axym ingests and preserves it. Fields: `parent_ref` (typed node reference — `kind` + `id` — linking to the record's parent: session, trace, run, intent, policy, agent, or evidence), `entity_refs[]` (typed references to all entities involved — agents, tools, resources, policies, evidence — each with `kind` and `id`), `policy_ref` (full policy lineage: `policy_id`, `policy_version`, `policy_digest` as SHA-256, `matched_rule_ids[]`), `agent_chain[]` (ordered delegation hops with `identity` and `role`: requester, delegator, delegate), `edges[]` (typed graph edges with `kind`, `from`, `to` — edge kinds: `calls`, `governed_by`, `delegates_to`, `targets`, `derived_from`, `emits_evidence`). Legacy compatibility fields (`parent_record_id`, `related_entity_ids[]`, `agent_lineage[]`) are preserved for backward compatibility. All fields are nullable with `omitempty`. Each relationship component has an `Extra` field (`map[string]json.RawMessage`) preserving additive fields for future graph extensions without breaking existing records. Gait emits relationship envelopes on TraceRecords, SessionEvents, ApprovalAuditRecords, and DelegationAuditRecords. Wrkr emits them on findings, risk assessments, attack paths, and posture scores via `proofmap.go`. When Axym ingests these records, the relationship data is preserved in the evidence chain — so when PolicyGraph ships (H3), it launches pre-populated with 18+ months of graph-structured governance topology accumulated from day-one proof records across all three sibling products
- Records are created via `proof.NewRecord()` and signed via `proof.Sign()` from the `Clyra-AI/proof` module
- Records are chained via `proof.AppendToChain()` — each record's hash includes the previous record's hash
- Chain integrity can be verified at any time (`axym verify --chain` delegates to `proof.VerifyChain()`)
- Broken chains are detected and flagged with the exact break point
- Records are immutable once written (append-only evidence store)
- Proof records are signed with Ed25519 (default) or cosign (Sigstore)

### FR5: Compliance Framework Mapping

- Map proof records to regulatory controls for: EU AI Act (Articles 9, 12, 13, 14, 15), SOC 2 (CC6, CC7, CC8), SOX (change management, SoD, access controls), PCI-DSS (Requirement 10), Texas TRAIGA, Colorado AI Act, ISO 42001 (AI risk assessment), NIST AI 600-1 (boundary enforcement)
- Framework definitions are loaded from `Clyra-AI/proof/frameworks/*.yaml` — shared across all Clyra AI products
- Mapping logic lives in Axym: matching proof records to required record types, calculating coverage, detecting gaps
- Each control maps to: required proof record types, minimum frequency, required fields
- **Controls as linkable references (H0 data architecture for H3 PolicyGraph).** Framework control identifiers are stored as structured references (`framework_id` + `control_id` + `version`), not flat string labels. "EU AI Act Article 14" becomes a traversable reference (`{framework: "eu-ai-act", control: "art-14", version: "2024"}`) that can connect to every proof record mapped to that control. This makes compliance mappings graph-queryable when PolicyGraph ships — "show me every proof record across every agent that maps to Article 14" becomes a reference lookup, not a string search
- Produce coverage report: which controls have evidence, which have gaps, what percentage is covered
- Gap analysis includes: what's missing, what record type is needed, remediation steps
- **OSCAL v1.1 export:** generate OSCAL component-definition JSON mapping proof records to control objectives (SOX CM/CC, PCI-DSS Req-10, EU AI Act articles). Shipped with every audit bundle. Human-readable snippet alongside machine-readable JSON. OSCAL mappings are derived from `Clyra-AI/proof/frameworks/*.yaml` definitions
- **Auditability grade (A–F):** each proof record and each bundle receives a deterministic auditability grade based on evidence completeness. Grade is reproducible — same inputs always produce same grade. Surfaced in compliance map, bundle summary, and Daily Review. Derivation rules (per record):
  - **A (Exemplary):** all criteria met — signature present and valid, chain link intact, all required fields populated, enrichment complete (no `ENRICHMENT_LAG`), SoD validated where applicable, replay Tier A where applicable
  - **B (Strong):** signature valid, chain intact, all required fields populated. Minor gaps: enrichment lag within tolerance, replay Tier B, or one optional field missing
  - **C (Adequate):** signature valid, chain intact, required fields present. Notable gaps: enrichment incomplete, SoD not validated, replay Tier C, or multiple optional fields missing
  - **D (Weak):** chain intact but signature missing or unverifiable, OR required fields missing. Record exists but auditability is compromised
  - **F (Failing):** chain link broken, record unsigned and unverifiable, or critical required fields absent. Record cannot be relied upon for compliance evidence
  - Bundle grade is the lowest grade among its constituent records (weakest link). A bundle with 99 A-grade records and 1 D-grade record is graded D
- **Record-type-to-framework mapping pattern.** Proof's 15 record types cover the full evidence surface. The table below shows which record types satisfy which framework categories — the specific control-level mapping lives in `Clyra-AI/proof/frameworks/*.yaml`, not in Axym code:

  | Proof Record Type | EU AI Act | SOC 2 | SOX | PCI-DSS | State AI Laws |
  |---|---|---|---|---|---|
  | `tool_invocation` | Art. 12 (logging) | CC7 (operations) | — | Req. 10 (audit trails) | Disclosure |
  | `decision` | Art. 14 (human oversight) | CC7 (operations) | — | — | Impact records |
  | `guardrail_activation` | Art. 9 (risk mgmt) | CC6 (access) | — | — | Risk controls |
  | `permission_check` | Art. 9 (risk mgmt) | CC6 (access) | Access controls | Req. 10 (monitoring) | — |
  | `human_oversight` | Art. 14 (oversight) | CC7 (operations) | SoD | — | Oversight records |
  | `policy_enforcement` | Art. 9 (risk mgmt) | CC6 (access), CC8 (change) | Change mgmt | — | Risk controls |
  | `scan_finding` | Art. 9 (risk mgmt) | CC7 (operations) | — | — | Risk assessment |
  | `risk_assessment` | Art. 9 (risk mgmt) | CC7 (operations) | — | — | Impact assessment |
  | `deployment` | Art. 15 (robustness) | CC8 (change) | Change mgmt | — | — |
  | `model_change` | Art. 15 (robustness) | CC8 (change) | Change mgmt | — | — |
  | `test_result` | Art. 15 (accuracy) | CC8 (change) | — | — | — |
  | `incident` | Art. 9 (risk mgmt) | CC7 (operations) | — | Req. 10 (monitoring) | Disclosure |
  | `data_pipeline_run` | Art. 12 (logging) | CC7 (operations) | Change mgmt, SoD | Req. 10 (audit trails) | — |
  | `replay_certification` | Art. 12, 15 (verification) | CC7, CC8 | — | — | — |
  | `approval` | Art. 14 (oversight) | CC6 (access) | SoD, Access | — | Oversight records |
  | `compiled_action` | Art. 12 (logging), Art. 14 (oversight) | CC7 (operations), CC8 (change) | Change mgmt | Req. 10 (audit trails) | Disclosure |

  This is the mapping *pattern* — which record types are relevant to which framework *categories*. The actual control-by-control requirements (minimum frequency, required fields, evidence thresholds) are defined in the framework YAML files and may change as regulatory guidance evolves. Axym treats them as configuration, not code. **Note:** proof ships framework YAMLs for EU AI Act Art. 9/12/13/14/15, SOC 2 CC6/CC7/CC8, SOX sox-cm, PCI-DSS Req-10, Texas TRAIGA, Colorado AI Act, ISO 42001, and NIST AI 600-1. Control-level depth varies — Art. 9 and 12 have the deepest sub-control mappings at launch; Art. 13, 15, and CC8 ship with category-level mappings and are refined as regulatory guidance matures.

- **Context-aware compliance mapping.** Axym's `ControlMatcher` considers context metadata — `data_class`, `endpoint_class`, and `risk_level` — when mapping records to controls. A `permission_check` record for a tool accessing `data_class: pii` or `data_class: credentials` maps to stricter controls (Art. 9 risk management, CC6 logical access) with higher evidence weight than the same record type accessing `data_class: internal`. Context metadata flows from three sources: (1) Wrkr discovery data ingested via FR10 — `data_class` and `endpoint_class` per tool; (2) Gait policy verdicts — risk classification from policy evaluation; (3) `axym-policy.yaml` system declarations — `risk_level` per AI system. This prevents a common compliance failure mode: treating all agent actions as equivalent regardless of what they touch. A tool invocation on a public docs endpoint and one on a production credentials store produce the same record type but have fundamentally different compliance implications. Context-aware mapping reflects this without requiring separate record types for every combination
- **Discovery method as compliance weight factor.** When proof records arrive with `discovery_method` in metadata (`static`, `webmcp`, `a2a`, `dynamic_mcp`), the ContextEnricher factors discovery method into evidence weight. Statically declared tools — present in agent config at deploy time with governance from the start — satisfy controls with full evidence weight. Dynamically discovered tools — encountered at runtime via WebMCP, A2A capability cards, or dynamic MCP registration — satisfy the same controls with reduced evidence weight because they were not part of the original risk assessment. A `permission_check` record for a statically declared tool with `data_class: pii` provides stronger Art. 9 coverage than the same record for a WebMCP-discovered tool. Coverage reports distinguish static vs. dynamic evidence. Gap analysis flags controls where all evidence comes from dynamically discovered tools as partial coverage, prompting the organization to formally assess and declare those tools

### FR6: Audit Bundle Generation

- Generate a portable, self-contained directory of proof records mapped to selected frameworks
- Bundle includes: executive summary (PDF), compliance map, proof records linked to controls, gap analysis, raw records, hash chain verification, methodology description, OSCAL v1.1 component-definition JSON, auditability grade summary
- Bundle includes `manifest.json` listing all files with SHA-256 hashes, schema versions, and anchor references; `proof verify` validates the manifest offline
- Bundle includes `boundary-contract.md` — plain-language statement clarifying what Axym proves vs. customer responsibility
- Bundle is cryptographically signed as a unit (Ed25519, tamper-evident)
- Bundle is versioned — can generate bundles for different time ranges and framework combinations
- Bundle includes `retention-matrix.json` — retention policy per environment (dev/test/prod) documenting how long evidence is stored and when it can be purged
- Bundle format is documented and stable (auditors can build tooling against it)
- All proof records in the bundle are verifiable with the standalone `proof verify` CLI
- Override artifacts: documented exceptions are included in the bundle with signer identity, justification, and timestamp — auditors see both the exception and its authorization

### FR7: Gap Detection, Daily Review, and Remediation

- Continuously monitor proof record coverage against selected frameworks
- Alert when coverage drops below configurable thresholds
- For each gap: describe what's missing, what control it affects, remediation steps, effort estimate
- Generate ticket templates (Jira, Linear, GitHub Issues) for engineering remediation
- Provide templates for common record types that require manual input (risk assessments, training records)
- **Daily Review Pack (from Clyra DNA):** `axym review --date <date>` generates a structured review of the specified 24-hour period (default: yesterday):
  - Exceptions: SoD violations, missing approvals, missing/mismatched tags, enrichment failures, attach failures, replay mismatches, freeze-window violations
  - Per-record auditability grades for all new records
  - Replay tier distribution (how many A/B/C certifications)
  - Export: CSV/PDF via CLI flags (`--format csv`, `--format pdf`)
  - Maps to PCI-DSS Req-10 daily review and SOX change management review requirements

### FR8: Ticket Integration and Evidence Attach

- Auto-attach proof records and verify links to Jira/ServiceNow change tickets by `change_id`
- Retries with exponential backoff; rate-limit handling (429/5xx)
- On exhaustion: DLQ + alert + local backup evidence for manual attach
- **Attach SLA/SLO:** 100% attached within 24h of ticket closure (compliance SLA); ≥95% attached within 10 minutes (operational SLO)
- Attach failures visible in Daily Review with retry/DLQ status

### FR9: Override and Exception Handling

- `axym override create --bundle <id> --reason "<justification>" --signer <key>` — create a signed exception for a specific proof record or bundle
- Overrides are signed artifacts: signer identity, timestamp, justification, scope, expiry
- Override artifacts are included in audit bundles alongside the original records — auditors see both the exception and its authorization
- Override reason codes are enumerated and auditable
- Override history is append-only and chain-linked (cannot be silently removed)

### FR10: Ingestion of Sibling Product Records

- Ingest proof records produced by Wrkr (`scan_finding`, `risk_assessment`, `approval`, `lifecycle_transition`) and include them in compliance maps and bundles — these are native `Clyra-AI/proof` records, no translation needed
- **Privilege drift mapping:** When consecutive Wrkr scans are ingested, Axym compares the permission surface between scans (new tools, new endpoint_class grants, new data_class access, autonomy level changes) and maps detected changes to access governance controls: SOC 2 CC6 (logical access changes monitored), SOX access controls (privilege changes detected and approved). Unapproved privilege escalations — a tool gaining `proc.exec` or `credentials` access without a corresponding `approval` record in the chain — are surfaced as compliance gaps in the Daily Review and gap analysis. **State storage:** Previous ingest state is stored in `.axym/wrkr-last-ingest.json` locally, keyed on `(agent_id, org)`. In CI, the state can be loaded from a committed artifact or cache key. Comparison requires at least two ingested scans — the first ingest establishes the baseline, subsequent ingests detect drift
- Ingest Gait artifacts from packs (PackSpec v1 ZIPs) by reading both native Gait types and derived proof records:
  - **Native Gait types (translation required):** Gait's shipped format uses `gait.gate.trace`, `gait.gate.approval_token`, and `gait.gate.delegation_token` — typed structs with Ed25519 signatures and JCS canonicalization, but not `Clyra-AI/proof` records. The GaitIngestor translates these to proof record types:
    - `gait.gate.trace` (verdict=block) → `policy_enforcement` proof record
    - `gait.gate.trace` (verdict=allow) → `permission_check` proof record
    - `gait.gate.trace` (verdict=dry_run) → `policy_enforcement` proof record (with `dry_run: true` in event metadata — policy evaluated but not enforced). **Compliance mapping note:** Axym's framework mapping logic distinguishes `dry_run: true` records from actual enforcement when calculating control coverage. Dry_run records demonstrate policy evaluation (evidence the policy exists and was applied) but do not satisfy controls requiring active enforcement. Coverage reports show dry_run records separately
    - `gait.gate.trace` (verdict=require_approval) → `guardrail_activation` proof record
    - `gait.gate.approval_token` → `approval` proof record (extract approver identity, scope, expiry, signature)
    - `gait.gate.delegation_token` → `delegation` proof record (extract delegator identity, delegatee identity, scope, chain depth, both policy digests, constraints, verdict). Delegation records map to Art. 14 (human oversight of delegation chains), CC6 (logical access — who authorized the delegatee), and SOX access controls. When delegation chain depth exceeds policy limits, the record captures the violation. Cross-agent delegation proof chains are linked: the delegator's record references the delegatee's chain via hash pointer, enabling auditors to trace the full accountability path across agent boundaries
  - **Gait source types the GaitIngestor reads (key fields shown — full definitions in `gait/core/schema/v1/gate/types.go` and `gait/core/schema/v1/pack/types.go`):**

    ```go
    // Gate evaluation result — the primary enforcement record
    // Schema: gait.gate.result v1.0.0
    type GateResult struct {
        Verdict     string   `json:"verdict"`      // "allow" | "block" | "dry_run" | "require_approval"
        ReasonCodes []string `json:"reason_codes"`
        Violations  []string `json:"violations,omitempty"`
    }

    // Trace record — signed audit trail of a gate evaluation
    // Schema: gait.gate.trace v1.0.0
    type TraceRecord struct {
        TraceID          string     `json:"trace_id"`          // SHA-256(policy:intent:verdict)[:12]
        ToolName         string     `json:"tool_name"`
        Verdict          string     `json:"verdict"`
        IntentDigest     string     `json:"intent_digest"`     // SHA-256 hex
        PolicyDigest     string     `json:"policy_digest"`     // SHA-256 hex
        Violations       []string   `json:"violations,omitempty"`
        ApprovalTokenRef string     `json:"approval_token_ref,omitempty"`
        DelegationRef    *DelegationRef   `json:"delegation_ref,omitempty"`
        SkillProvenance  *SkillProvenance `json:"skill_provenance,omitempty"`
        Signature        *Signature `json:"signature,omitempty"` // Ed25519
    }

    // Intent request — what the agent wanted to do
    // Schema: gait.gate.intent_request v1.0.0
    type IntentRequest struct {
        ToolName        string              `json:"tool_name"`
        Args            map[string]any      `json:"args"`
        Targets         []IntentTarget      `json:"targets"`          // Kind, Value, Operation, EndpointClass, Destructive
        ArgProvenance   []IntentArgProvenance `json:"arg_provenance,omitempty"`
        SkillProvenance *SkillProvenance    `json:"skill_provenance,omitempty"`
        Delegation      *IntentDelegation   `json:"delegation,omitempty"`
        Context         IntentContext       `json:"context"`          // Identity, Workspace, RiskClass
    }

    // Approval token — signed human approval for a specific intent
    // Schema: gait.gate.approval_token v1.0.0
    type ApprovalToken struct {
        TokenID                 string    `json:"token_id"`
        ApproverIdentity        string    `json:"approver_identity"`
        ReasonCode              string    `json:"reason_code"`
        IntentDigest            string    `json:"intent_digest"`
        PolicyDigest            string    `json:"policy_digest"`
        DelegationBindingDigest string    `json:"delegation_binding_digest,omitempty"`
        Scope                   []string  `json:"scope"`             // array, ≥1 entry
        ExpiresAt               time.Time `json:"expires_at"`
        Signature               *Signature `json:"signature,omitempty"` // Ed25519
    }

    // Delegation token — signed authority transfer
    // Schema: gait.gate.delegation_token v1.0.0
    type DelegationToken struct {
        TokenID           string    `json:"token_id"`
        DelegatorIdentity string    `json:"delegator_identity"`
        DelegateIdentity  string    `json:"delegate_identity"`
        Scope             []string  `json:"scope"`              // array, ≥1 entry
        ScopeClass        string    `json:"scope_class,omitempty"`
        IntentDigest      string    `json:"intent_digest,omitempty"`
        PolicyDigest      string    `json:"policy_digest,omitempty"`
        ExpiresAt         time.Time `json:"expires_at"`
        Signature         *Signature `json:"signature,omitempty"` // Ed25519
    }

    // Pack manifest — the container for all artifacts
    // Schema: gait.pack.manifest v1.0.0
    type Manifest struct {
        SchemaID        string      `json:"schema_id"`
        SchemaVersion   string      `json:"schema_version"`
        PackID          string      `json:"pack_id"`          // SHA-256 of JCS-canonical manifest
        PackType        string      `json:"pack_type"`        // "run" | "job" | "call"
        SourceRef       string      `json:"source_ref,omitempty"`
        Contents        []PackEntry `json:"contents"`
        Signatures      []Signature `json:"signatures,omitempty"`
    }

    type PackEntry struct {
        Path   string `json:"path"`
        SHA256 string `json:"sha256"`
        Type   string `json:"type"`             // "json" | "jsonl" | "zip" | "blob"
    }

    // Shared signature struct — same across all signed artifacts
    type Signature struct {
        Alg          string `json:"alg"`           // "ED25519"
        KeyID        string `json:"key_id"`        // SHA-256 hex of public key bytes
        Sig          string `json:"sig"`           // base64-encoded
        SignedDigest string `json:"signed_digest"` // hex digest of signed content
    }
    ```

  - **Compiled action artifacts (PTC-style scripts):** When Gait evaluates a compound action — a compiled script containing multiple tool calls emitted by programmatic tool calling (PTC) or equivalent multi-step agent planners — the GaitIngestor translates the script-level evaluation into a `compiled_action` proof record. The record captures: script hash (SHA-256 of the compiled action), tool list (ordered sequence of tools in the plan), composite risk classification (highest risk tool in the chain), script structure summary (step count, conditionals, loops), and the gate verdict for the script as a whole. The `compiled_action` record links to the individual `policy_enforcement`/`permission_check` trace records for each tool call within the script via `event.execution_trace_refs[]`. This provides auditors with both the plan (what was intended) and the execution (what happened) in one evidence chain. If Gait evaluated the script as a unit (script-level intent mode), one `compiled_action` record is emitted. If Gait evaluated individual calls within the script, the GaitIngestor correlates them by session/job ID and synthesizes the `compiled_action` record at ingestion time
  - **Derived proof records (no translation):** If Gait packs contain a `proof_records.jsonl` file, those are already `Clyra-AI/proof` format and are ingested directly
  - **Session journals:** Ingest `gait.runpack.session_journal` records — session-level evidence of tool call sequences, verdict sequences, and checkpoint chains. Session events map to compliance controls for change management and access governance
  - **Job lifecycle:** Ingest Gait job state transitions (running → paused → decision_needed → completed) as evidence of human-in-the-loop governance when jobs require approval gates
- Gait and proof share cryptographic foundations (Ed25519, JCS/RFC 8785, SHA-256) so signature verification works across both native and translated records
- **Two complementary integrity models (not a bridging problem):** Gait uses manifest+signature integrity (PackSpec v1 manifests list every file with SHA-256 digests, signed as a unit) — this provides point-in-time artifact integrity ("this evidence package is intact"). Proof's hash chains provide stream integrity ("nothing was deleted or reordered over time"). These serve different purposes and coexist naturally. Gait packs arrive with manifest+signature integrity already verified. Axym then appends the translated proof records to its continuous evidence chain using `proof.AppendToChain()` — the chain link is established at ingestion time, not at Gait's production time. Both integrity guarantees are preserved: the pack's manifest proves the records weren't tampered with before ingestion, and the chain proves they weren't tampered with after. Session checkpoint chains (`prev_checkpoint_digest`) provide additional sequence integrity within Gait's native format
- **Checkpoint-aware evidence chain stitching.** Durable agents that checkpoint, pause, crash, and resume produce proof record chains that span multiple sessions. When ingesting multi-session evidence, Axym detects session boundaries (checkpoint markers, session ID transitions, timestamp gaps), verifies the chain link across boundaries, and reports any gaps. If a chain has a gap between sessions — evidence lost during a crash or incomplete checkpoint — the gap window is flagged in the Daily Review with `CHAIN_SESSION_GAP` reason code, the exact missing record range, and the affected time window. Gap records receive auditability grade D (chain intact but evidence incomplete for the gap period). For intact cross-session chains, Axym stitches records into a continuous evidence sequence regardless of how many sessions produced them
- **Relationship envelope preservation.** When ingesting proof records from Wrkr and Gait, the GaitIngestor and WrkrIngestor preserve the full `relationship` envelope — `parent_ref`, `entity_refs`, `policy_ref`, `agent_chain`, `edges`, and all legacy compatibility fields. Translated records (from Gait's native types) have relationship envelopes synthesized from the source record's policy lineage and agent context fields. This ensures the unified evidence chain carries the complete graph topology from all three products, ready for PolicyGraph queries at H3
- Unified evidence chain: Wrkr discovery records, Axym collection records, and Gait enforcement records (translated at ingestion) can be appended to the same hash chain

### FR11: CLI Experience

- `axym init` — interactive setup (evidence sources, frameworks, signing key)
- `axym collect` — run evidence collection cycle (one-shot or CI-triggered)
- `axym collect --dry-run` — show what would be captured without writing
- `axym map --frameworks [list]` — show compliance coverage for selected frameworks
- `axym gaps` — show all compliance gaps ranked by severity
- `axym bundle --audit [name] --frameworks [list] --from [date] --to [date]` — generate audit bundle
- `axym review --date [date]` — generate Daily Review Pack for the specified date (default: yesterday)
- `axym verify --chain` — verify proof chain integrity (delegates to `Clyra-AI/proof`)
- `axym verify --bundle [path]` — verify audit bundle integrity: cryptographic verification (signatures, hashes, chain — delegates to `proof.Verify()`) **plus** compliance completeness checks (required record types present per framework, field coverage, auditability grade recalculation). `proof verify --bundle` performs cryptographic verification only — no compliance opinions. Auditors use `proof verify`; GRC analysts use `axym verify` for the full picture
- `axym record add --type [type] --file [path]` — add manual proof record (for non-automated controls)
- `axym override create --bundle [id] --reason [text] --signer [key]` — create signed exception
- `axym ingest --source [wrkr|gait|path]` — ingest proof records from sibling products or external sources. For `gait`: accepts PackSpec v1 ZIP files (extracted automatically), directories of extracted pack files, or individual pack paths; Gait's `gait.gate.trace` and token types are translated to proof record format at ingestion time. For `wrkr`: reads wrkr's proof record output directory. For `path`: reads JSONL files containing proof-format records
- `axym replay --model [name] --tier [A|B|C]` — run replay certification for a specific pipeline model
- `axym regress init --baseline [bundle-path]` — establish compliance baseline from a known-good state
- `axym regress run --baseline [path] --frameworks [list]` — check for compliance regression (exit 5 on drift)
- All commands: `--json` flag for machine-readable output
- All commands: `--quiet` flag for CI usage
- All commands: `--explain` flag for verbose diagnostic output
- All commands use the shared exit code vocabulary defined in `Clyra-AI/proof`: `0` success, `2` verification failure, `5` regression drift, etc.

### FR12: Collector Plugin Architecture

- Collectors are plugins that capture evidence from specific sources
- Built-in collectors: MCP server logs, MCP gateway audit logs (Kong, Docker, MintMCP), LLM API middleware (OpenAI, Anthropic), CI/CD events (GitHub Actions), Git metadata, dbt pipeline logs, Snowflake query history, eval platform results (Braintrust, Arize, LangSmith), model registry events (MLflow, Hugging Face Hub, W&B), adversarial/red team results (Giskard, NeMo Red Team)
- Collector interface: `Collect(config CollectorConfig) ([]proof.Record, error)`
- Third-party collectors can be built as standalone binaries implementing the collector protocol: Axym invokes the binary, passes `CollectorConfig` as JSON on stdin, reads `[]proof.Record` as JSONL on stdout. Exit code 0 = success, non-zero = failure (stderr contains error message). Go plugins (`plugin` package) are supported as a secondary option but are fragile — same Go version, OS, and build flags required
- Collector registry documents available collectors and their configuration
- **Adapter-first design:** Collectors read existing outputs — MCP server log files, LLM API response middleware, CI pipeline event webhooks, git metadata, dbt run results. Collectors never require modifications to the source system. No SDK installation in agent frameworks, no upstream PRs to MCP servers, no changes to CI pipeline definitions beyond adding a post-step. The moment a collector requires the source to change how it works, adoption stalls. This is the same pattern Gait proved with eight reference integrations: wrap existing tools, never ask them to change.
- **MCP gateway audit log ingestion.** MCP gateways (Kong, Docker MCP Gateway, MintMCP, Lunar) centralize agent-to-tool access control and produce structured audit logs documenting which agents called which tools, when, with what auth, and whether the call was allowed or blocked. The `MCPGatewayCollector` reads these logs and translates entries into `permission_check` proof records (access allowed) and `policy_enforcement` proof records (access blocked). This is valuable because many enterprises will deploy MCP gateways before adopting Gait — the gateway handles auth and rate limiting, Axym produces the signed compliance evidence. Supported log formats: Kong gateway JSON logs, Docker MCP Gateway interceptor events, MintMCP audit trail exports. The collector extracts: agent identity (from gateway auth context), tool name, action, verdict, timestamp, and gateway policy version. Gateway-sourced records are tagged with `evidence_source: mcp_gateway` in metadata so compliance mapping can distinguish them from Gait-sourced enforcement evidence
- **Governance event ingestion.** As agent frameworks adopt lightweight governance telemetry (structured JSONL events describing governance decisions — tool gating, permission checks, approval requests, policy evaluations), Axym should ingest these events alongside its existing collector outputs. The `GovernanceEventCollector` reads governance event streams from: (1) stdout JSONL from agent framework middleware (same adapter-first pattern — no SDK required, just pipe output); (2) webhook endpoint for real-time event push; (3) file-based JSONL for batch ingestion. Governance events are lightweight, unsigned, real-time signals. The collector validates them against `Clyra-AI/proof`'s governance event JSON Schema, promotes valid events to signed proof records via `proof.NewRecord()` + `proof.Sign()`, and appends them to the evidence chain. Invalid events are logged and rejected. This creates a two-tier evidence model: governance events are the real-time adoption surface (easy for agent frameworks to emit — 3 lines of code to write JSONL to stdout), proof records are the compliance-grade durable form. The promotion from event to record is where signing, chain-linking, and schema validation happen. This is the top-of-funnel play for the proof record format: lower the barrier to producing governance telemetry, then graduate it into signed compliance evidence

### FR13: Compliance Regression

- `axym regress init --baseline [bundle-path]` — establish a known-good compliance state as a CI baseline from an existing audit bundle or compliance map
- `axym regress run --baseline [path] --frameworks [list]` — compare current compliance coverage against baseline
- If coverage drops below the baseline for any control that was previously covered, exit code `5` (regression drift detected)
- Regression checks are deterministic: same proof records + same baseline = same result
- Regression fixtures are portable files that can be committed to the repo and run in CI
- **Use case:** After passing an audit, run `axym regress init` to capture the passing state. Add `axym regress run` to CI. If a team removes evidence collection, disables a guardrail, or breaks the proof chain, CI fails before the auditor notices.
- Maps to the cross-product regression pattern: Gait converts bad runs into enforcement regression fixtures, Wrkr converts bad scans into posture regression fixtures, Axym converts compliance gaps into coverage regression fixtures. Same exit code (`5`), same CI semantics, different governance domain.

-----

## Non-Functional Requirements

### NFR1: Data Sovereignty & Minimization

- **Zero data exfiltration.** Axym never sends proof data outside the user's environment. No telemetry on evidence contents.
- **Data minimization by design.** Proof records capture proofs, not payloads. Query hashes, not queries. Action summaries, not full LLM responses. Enough to prove compliance, not enough to reconstruct sensitive data. Canonicalization for sensitive fields (SQL, URLs, prompts) is handled by `proof.Canonicalize()`.
- **Configurable redaction.** Users can define redaction rules for sensitive fields before records are written.
- Optional anonymous usage telemetry (command counts only, opt-in).

### NFR2: Performance

- Evidence collection adds < 50ms latency per event (non-blocking by default, async write)
- Compliance mapping for 10,000 records against 4 frameworks: under 30 seconds
- Bundle generation for 10,000 records: under 2 minutes
- Chain verification for 100,000 records: under 60 seconds

### NFR3: Reliability & Durability (Clyra DNA)

- Evidence collection failures are logged but never interrupt agent operation
- Failed collection attempts are queued and retried (configurable retry policy)
- Evidence store is append-only — no mechanism to delete or modify records via the CLI
- All outputs (records, maps, bundles) are schema-validated before writing
- Deterministic: same proof records + same framework mappings = same compliance map and bundle
- **Compliance mode (default for production):** Evidence Loss Budget = 0. Collection returns 200 OK only after durable write (fsync to PVC) or enqueue to durable queue (SQS/Kafka/PubSub/Redis-AOF). If upstream cannot backpressure, fail closed (policy breach). At-least-once delivery with idempotent ingest (dedupe by composite key: `source_product + record_type + event_hash`, TTL 7 days). Re-ingesting records from Wrkr or Gait that are already in the chain is safe — deduplication prevents double-chaining while preserving the original chain link
- **Degradation policy:** `on_sink_unavailable` = `fail_closed` (default) | `advisory_only` | `shadow`. Per-collector overrides allowed. Degradation events emit `SINK_UNAVAILABLE` reason code and are surfaced in Daily Review
- **Best-effort mode (dev/test only):** bounded in-memory spool with oldest-drop. Clearly labeled — never for production compliance evidence
- Metrics/alerts: spool_depth, blocked_ms, enqueue_failures, dlq_depth, verify_pass_rate

### NFR4: Security

- Proof records are signed individually via `proof.Sign()` (per-record integrity)
- Proof chain provides sequence integrity via `proof.AppendToChain()` (tamper detection)
- Audit bundles are signed as a whole (bundle integrity)
- Signing supports: Ed25519 (default), Sigstore (cosign), customer-managed keys
- Axym requests minimum source permissions: read-only access to logs and metadata
- Evidence store directory permissions are restricted (owner read-write only)

### NFR5: Extensibility

- **Collector plugins** for adding new evidence sources
- **Framework mappings** are YAML files in `Clyra-AI/proof/frameworks/` (add new regulations without code changes in any repo)
- **Record types** are extensible via `Clyra-AI/proof`'s custom type mechanism — users define JSON Schema, Axym validates against it
- **Bundle formatters** for custom output formats (JSON, CSV, PDF templates)

-----

## Goals

1. **15-minute time-to-first-value.** From `brew install Clyra-AI/tap/axym` to seeing a compliance coverage map with real proof records from your AI systems.
2. **Become the evidence format auditors trust.** Axym evidence bundles contain proof records verifiable by the standalone `proof` CLI. Auditors learn one verification tool that works across all Clyra AI products and any third-party tool that emits proof records.
3. **Close the translation gap.** Engineering produces technical signals. GRC needs compliance evidence. Axym is the translator — it speaks both languages.
4. **Compliance as continuous, not annual.** Evidence is captured continuously, not assembled in a panic before the audit. The audit bundle is always ready.
5. **Bottom-up adoption through engineering trust.** Open source, local-only, non-blocking. Engineers integrate it because it's low-friction. GRC teams fund it because it solves their evidence problem.
6. **Feed the shared primitive.** Axym is the heaviest producer of proof records. Every Axym installation generates records in the shared `Clyra-AI/proof` format. As Axym adoption grows, the volume of proof records in the ecosystem grows, and the format earns standard status through ubiquity.
7. **Continuous engagement, not audit-time panic.** The Daily Review Pack (FR7) creates a recurring daily touchpoint — exceptions, grade changes, and replay results demand attention whether or not an audit is imminent. Compliance regression in CI (FR13) fails builds when evidence coverage drops, creating immediate engineering work. Gap alerts trigger ticket creation. The audit bundle is the deliverable, but the daily cadence is the retention: Axym produces work every day, not just at audit time. The pattern: Wrkr creates weekly PR work, Axym creates daily review work, Gait creates per-action enforcement work — three cadences that keep the governance loop active.

## Non-Goals (v1)

1. **Not a GRC platform.** Axym does not manage compliance workflows, assign control owners, or track policy approvals. It produces evidence that GRC platforms consume.
2. **Not a runtime enforcement engine.** Axym does not block agent actions in real-time — that's Gait's job. Axym does evaluate evidence-time policy (SoD validation, freeze windows, enrichment completeness) and marks `decision.pass=false` when policy is violated, but this is evidence classification, not runtime blocking.
3. **Not an observability tool.** Axym does not replace LangSmith, Arize, or Datadog for operational debugging. It captures compliance-grade proof, not operational telemetry.
4. **Not a scanner.** Axym does not discover AI tools. Discovery is Wrkr's job. Axym captures evidence of what discovered tools actually do.
5. **Not an AI risk assessment tool.** Axym captures and presents evidence of risk management. The actual risk assessment is a human exercise — Axym provides templates and captures the output, but doesn't automate judgment.
6. **Not a legal opinion.** Framework mappings are informational. "Consult qualified legal counsel for compliance determinations." Axym provides evidence infrastructure, not legal advice.
7. **Not a SaaS product (v1).** The open-source CLI is the product. CLI `axym collect` runs one-shot or as a CI post-step (self-hosted). This PRD covers OSS scope only. Clyra Platform and Enterprise tiers are defined in the roadmap and risk register sections but their FRs, NFRs, ACs, and DoD are out of scope for this document.
8. **Not cross-platform on day one.** GitHub Actions is the primary CI integration. Jenkins, GitLab CI, and CircleCI follow in v1.1+.
9. **Not a primitive.** The proof record format, hash chain, and signing protocol live in `Clyra-AI/proof`. Axym is the reference consumer and the heaviest producer, but it does not own the format. The primitive is independent of any single product.

-----

## Acceptance Criteria (LVP v1 "Done" Definition)

### AC1: The "15-Minute Demo"

A platform engineer with a production AI agent using MCP servers and deployed via GitHub Actions can:

- Install Axym (`brew install Clyra-AI/tap/axym`)
- Run `axym init` with source configuration
- Run `axym collect` to capture proof records from existing sources
- Run `axym map --frameworks eu-ai-act` to see compliance coverage
- See a compliance map with real proof records mapped to regulatory controls
- Total elapsed time: under 15 minutes

### AC2: The "Board Slide"

The output of `axym bundle --audit Q3-2026` includes an `executive-summary.pdf` that a Head of GRC can present to a board, containing: number of AI systems in scope, proof records collected, compliance coverage percentage by framework, top gaps with remediation status.

### AC3: The "Auditor Handoff"

The output of `axym bundle --audit Q3-2026 --frameworks eu-ai-act,soc2` produces a directory that a GRC analyst can hand to an external auditor, containing: structured proof records mapped to specific control objectives, gap analysis, hash chain verification, and bundle signature. The auditor can independently verify bundle integrity with `proof verify --bundle` — no Axym installation required.

### AC4: The "Chain Integrity"

`axym verify --chain` on a proof store of 10,000+ records either confirms "Chain intact. N records. No gaps." or identifies the exact break point. Manually tampering with a single record causes verification to fail with the specific record identified.

### AC5: The "Gap Alert"

After removing or simulating absence of a required proof record type (e.g., no risk evaluation records), `axym gaps` correctly identifies the missing control, the affected regulatory articles, and produces a remediation recommendation with effort estimate.

### AC6: The "Collector Test"

For each built-in collector (MCP server, LLM API middleware, GitHub Actions, Git metadata, dbt pipeline), a test fixture exists with known events, and Axym correctly captures and structures 100% of them into valid proof records.

### AC7: The "Non-Blocking Guarantee"

Evidence collection can be configured to run alongside a production AI agent. If the evidence collection system fails (disk full, permission error, schema validation failure), the agent operation continues unaffected. Failures are logged and queued for retry.

### AC8: The "Zero Egress" Audit

A network-isolated run (no outbound internet after install) completes successfully. Proof records and bundles exist only on local filesystem. No DNS lookups to non-configured domains during collection or bundle generation.

### AC9: The "Cross-Product Chain"

Proof records from Wrkr (scan findings), Gait (enforcement decisions, translated from native `gait.gate.trace` and token types at ingestion time), and Axym (collected evidence) are appended to the same chain. `axym verify --chain` and `proof verify --chain` both validate the mixed-source chain. The chain doesn't care which tool produced which record or whether the record was native proof format or translated from Gait's PackSpec artifacts. Gait's manifest+signature integrity is preserved in the source pack; the proof chain link is established when Axym ingests and translates the records.

### AC10: The "Data Pipeline Evidence"

Axym's dbt collector captures a pipeline run, produces `data_pipeline_run` proof records with canonicalized SQL digests, SoD validation (requestor ≠ deployer), and Snowflake query history enrichment. The records map to SOX change management controls. The replay engine executes in a sandbox, certifies a Tier A replay, and produces a `replay_certification` proof record with blast radius summary. A SoD violation produces `decision.pass=false` with `SOD_REQUESTOR_DEPLOYER`. A missing QUERY_TAG produces `decision.pass=false` with `MISSING_QUERY_TAG`. Both are surfaced in the Daily Review Pack.

### AC11: The "Compliance Mode Guarantee"

In compliance mode, evidence collection returns success only after durable write. Simulating a sink failure (disk full, queue unavailable) causes Axym to fail closed — no silent evidence loss. The failure appears in the Daily Review with `SINK_UNAVAILABLE` reason code. The evidence loss budget is verifiably zero: `axym verify --chain` confirms no gaps in the sequence.

### AC12: The "OSCAL Export"

`axym bundle --audit Q3-2026 --frameworks sox,pci-dss` includes an `oscal-v1.1/component-definition.json` that maps proof records to SOX and PCI-DSS control objectives in valid OSCAL v1.1 format. An auditor using OSCAL-compatible tools can import the mapping directly.

### AC13: The "Compliance Regression"

After generating a passing compliance bundle (coverage ≥ 80% for EU AI Act), `axym regress init` captures the baseline. Simulating evidence loss (removing a collector, breaking the proof chain, or deleting records for a required control) causes `axym regress run` to exit with code `5` and report exactly which controls regressed. The regression fixture is a portable file that runs identically in CI and locally. Same proof records + same baseline = same pass/fail result, deterministically.

### AC14: The "Daily Review"

`axym review --date 2026-09-15` produces a Daily Review Pack covering the preceding 24 hours. The pack lists: all exceptions (SoD violations, missing approvals, enrichment failures, freeze-window violations), per-record auditability grades for all new records, replay tier distribution, and attach status for ticket integrations. Output is available as structured JSON (`--json`) and as CSV/PDF export. When no evidence was collected in the period, the review reports an empty day with zero exceptions rather than failing. The review content matches exactly what FR7 specifies.

### AC15: The "Ticket Attach"

A test fixture Jira instance has a change ticket `CHANGE-1234`. `axym collect` captures proof records for a deployment and tags them with `change_id: CHANGE-1234`. Axym auto-attaches the proof records to the ticket within 10 minutes (operational SLO). Simulating a Jira 429 rate-limit response causes Axym to retry with exponential backoff and succeed on the next attempt. Simulating sustained Jira unavailability (5xx for >10 minutes) causes the record to enter the DLQ with an alert. The DLQ'd record appears in the Daily Review with `ATTACH_FAILED` status. After Jira recovers, `axym collect` retries and attaches the evidence within the 24h compliance SLA. All attached records are 100% accounted for — none silently lost.

### AC16: The "Override Exception"

An audit bundle for Q3-2026 has a gap: no `risk_assessment` records for a specific AI system. `axym override create --bundle Q3-2026 --reason "Risk assessment conducted verbally with CISO on 2026-08-15, documented in RISK-789" --signer ops-key` produces a signed override artifact. Re-generating the bundle now includes the override alongside the gap — auditors see both the gap and its signed justification with signer identity, timestamp, and expiry. `axym verify --chain` confirms the override is chain-linked and append-only. Attempting to delete the override leaves a chain gap detected by verification.

### AC17: The "Custom Collector"

A third-party collector built as a standalone binary implementing the collector protocol (receives `CollectorConfig` as JSON on stdin, emits `[]proof.Record` as JSONL on stdout) is registered with Axym. The collector captures events from a custom internal tool, produces valid proof records with correct schema validation. `axym collect` discovers and runs the custom collector alongside built-in collectors. The custom collector's records appear in compliance maps and audit bundles indistinguishable from built-in collector output. A collector that returns malformed records (missing required fields, invalid record type) is rejected at schema validation with a clear error — the malformed records do not enter the proof chain.

-----

## Tech Stack & Architecture

### Design Principles

1. **Evidence as artifact.** Every output is a file — structured, versioned, signed, portable. Not a dashboard you have to log into. Not a SaaS you depend on.
2. **Shared primitive, product opinions.** The proof record format, hash chain, and signing live in `Clyra-AI/proof` (zero compliance opinions). Axym imports `Clyra-AI/proof` and adds compliance logic on top: collection, framework mapping, gap detection, bundle generation. This separation means the format can be adopted independently by any tool without buying into Axym.
3. **Deterministic pipeline.** Zero LLMs in the evidence chain. Pattern matching, schema validation, cryptographic hashing. Same inputs → same outputs, always. A compliance tool that hallucinates evidence is worse than no tool at all.
4. **Minimal data, maximum proof.** Capture the minimum data needed to prove compliance. Hashes, not payloads. Summaries, not transcripts. The proof record proves the control was in place without creating a new data liability.
5. **Boring technology.** Go. Single static binary. Infrastructure standard.

### Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                         axym CLI                              │
│                           (Go)                                │
├──────────┬───────────┬──────────┬──────────┬─────────────────┤
│  init    │  collect   │  map     │  gaps    │  bundle         │
│          │           │          │          │                 │
│  config  │  ┌───────┐│  framework│ coverage │  audit package  │
│  wizard  │  │collect││  mapping │ analysis │  generation     │
│          │  │engine ││          │          │  + signing      │
│          │  └──┬────┘│          │          │                 │
│          │     │     │          │          │                 │
├──────────┴─────┴─────┴──────────┴──────────┴─────────────────┤
│  imports github.com/Clyra-AI/proof                                  │
├──────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐       │
│  │ Record Schema │  │ Hash Chain   │  │ Signing      │       │
│  │              │  │              │  │              │       │
│  │ NewRecord()  │  │ Append()     │  │ Sign()       │       │
│  │ Validate()   │  │ Verify()     │  │ Verify()     │       │
│  │ Canonicalize │  │ Range()      │  │ Cosign()     │       │
│  │ Schema ver.  │  │ Integrity    │  │ Ed25519      │       │
│  └──────────────┘  └──────────────┘  └──────────────┘       │
│                                                              │
│  Clyra-AI/proof — shared across Wrkr, Axym, Gait                   │
│  Zero compliance opinions. Just: create, chain, sign, verify. │
│                                                              │
│  Also consumed by:                                            │
│  - Agent frameworks (emit records from inside agent code)     │
│  - MCP servers (emit records per tool invocation)             │
│  - CI pipelines (emit records per deployment)                 │
│  - GRC platforms (consume and verify records)                 │
│                                                              │
└──────────────────────────────────────────────────────────────┘

Collection Engine (collector plugins — part of axym CLI):
  ├── MCPCollector           → MCP server interaction logs
  ├── MCPGatewayCollector    → Kong / Docker / MintMCP gateway audit logs
  │                            → permission_check + policy_enforcement records
  ├── LLMAPICollector        → OpenAI / Anthropic API middleware
  ├── GitHubActionsCollector → CI/CD pipeline events
  ├── GitMetadataCollector   → Commits, reviews, deployments
  ├── GuardrailCollector     → Guardrail activation records
  ├── DbtCollector           → dbt pipeline runs (Clyra DNA)
  ├── SnowflakeCollector     → Query history digests (Clyra DNA)
  ├── EvalCollector          → Braintrust / Arize / LangSmith eval results → test_result records
  ├── ModelRegistryCollector → MLflow / HF Hub / W&B model events → model_change + deployment records
  ├── RedTeamCollector       → Giskard / NeMo Red Team outputs → test_result + risk_assessment records
  ├── WebhookCollector       → Custom webhook endpoint for any source
  ├── GovernanceEventCollector → Lightweight governance event JSONL from agent frameworks
  │                            (stdout pipe, webhook, or file — validates and promotes to
  │                            signed proof records; adoption funnel for proof format)
  └── ManualCollector        → CLI-based manual evidence entry

  Each collector uses Clyra-AI/proof to produce valid proof records.

Compliance Mapper (part of axym CLI — NOT in Clyra-AI/proof):
  reads: Clyra-AI/proof/frameworks/*.yaml (shared definitions)
  ├── ControlMatcher         → Match proof records to controls (context-aware: considers data_class,
  │                            endpoint_class, risk_level from Wrkr discovery, Gait verdicts,
  │                            and axym-policy.yaml system declarations)
  ├── ContextEnricher        → Resolves context metadata for each record: merges Wrkr discovery
  │                            data (tool → data_class/endpoint_class), Gait risk classification,
  │                            axym-policy.yaml system risk_level, and discovery_method
  │                            (static/webmcp/a2a/dynamic_mcp) into a unified context
  │                            that ControlMatcher uses for weighted framework mapping
  ├── CoverageCalculator     → Percentage coverage per control
  └── GapDetector            → Missing record types and frequencies

Bundle Generator (part of axym CLI — NOT in Clyra-AI/proof):
  ├── EvidenceAssembler      → Collect and organize records by framework
  ├── PDFRenderer            → Executive summary PDF
  ├── BundleSigner           → Uses proof.Sign() from Clyra-AI/proof
  └── BundleVerifier         → Delegates to proof.Verify()

Sibling Record Ingestion (part of axym CLI):
  ├── WrkrIngestor           → Reads Wrkr proof records (scan_finding, risk_assessment, approval, lifecycle_transition)
  ├── GaitIngestor           → Reads Gait packs (PackSpec v1 ZIPs):
  │     ├── TraceTranslator  → gait.gate.trace → policy_enforcement/permission_check/guardrail_activation
  │     ├── TokenTranslator  → gait.gate.approval_token → approval
  │     ├── DelegationTranslator → gait.gate.delegation_token → delegation (cross-agent chain links)
  │     ├── CompiledActionTranslator → Script-level evaluations → compiled_action proof records
  │     │                      (links plan hash to execution trace refs; synthesizes from
  │     │                      correlated per-call traces when script-level eval not available)
  │     ├── SessionIngestor  → gait.runpack.session_journal → session-level evidence
  │     ├── JobIngestor      → Job state transitions → human-oversight evidence
  │     ├── ChainStitcher    → Detect session boundaries, verify cross-session chain links,
  │     │                      flag gaps with CHAIN_SESSION_GAP in Daily Review
  │     └── ProofPassthrough → proof_records.jsonl (if present) → direct ingestion
  ├── GovernanceEventIngestor → Reads lightweight governance event JSONL from agent framework
  │                            middleware (stdout, webhook, or file). Validates against governance
  │                            event JSON Schema, promotes to signed proof records via
  │                            proof.NewRecord() + proof.Sign(). Adoption funnel: events are
  │                            easy to emit, records are compliance-grade
  └── GenericIngestor        → Reads any valid Clyra-AI/proof records from a directory

Ticket Integration (part of axym CLI — Clyra DNA):
  ├── JiraBridge             → Auto-attach proof + verify link to Jira change tickets
  ├── ServiceNowBridge       → Auto-attach proof + verify link to ServiceNow change records
  └── DLQ + RetryEngine      → Exponential backoff, rate-limit handling, dead-letter queue

Daily Review Engine (part of axym CLI — Clyra DNA):
  ├── ExceptionAggregator    → SoD violations, missing tags, enrichment failures
  ├── ReplayTierSummarizer   → Tier A/B/C distribution
  ├── GradeCalculator        → Auditability grade (A-F) per record and aggregate
  └── ReviewPackExporter     → CSV/PDF/email output

Replay Engine (part of axym CLI — Clyra DNA):
  ├── DbtReplay              → Fetch compiled SQL, execute in sandbox, compare digests
  ├── HttpReplay             → Fetch encrypted stubs, simulate responses
  ├── LlmReplay              → Metadata verification only (allowlist check)
  └── ReplayCertEmitter      → Emit signed replay_certification proof records
```

**The boundary rule:** If it's about the shape, integrity, or signing of evidence — it's in `Clyra-AI/proof`. If it's about what the evidence *means* for compliance — it's in `axym` CLI. Framework definitions (what controls exist) live in `Clyra-AI/proof`. Framework mapping logic (does this record satisfy this control?) lives in `axym`.

### Tech Choices

| Component | Choice | Why |
|---|---|---|
| Language | Go | Same as Gait, Wrkr, and Clyra-AI/proof. Single static binary. Infrastructure standard. No runtime dependencies. |
| CLI framework | `cobra` + `viper` | Standard for Go CLIs (kubectl, gh, hugo). Consistent with Gait. |
| Proof records | `github.com/Clyra-AI/proof` (Go module) | Shared primitive. Records, chain, signing, canonicalization, verification, framework definitions. |
| Evidence store | Append-only JSONL files on local filesystem | No database dependency. Portable. `git`-friendly. Enterprise teams can back up however they want. |
| Schema validation | Via `Clyra-AI/proof` (JSON Schema validation) | Proof records are validated at creation time by the shared module. |
| Compliance mappings | YAML configuration files from `Clyra-AI/proof/frameworks/` | Shared across products. Non-engineers can read and review. New frameworks added without code changes. |
| PDF generation | `jung-kurt/gofpdf` or `pdfcpu` | For executive summary and audit report outputs. Pure Go, no external dependencies. |
| Git operations | `go-git/go-git` | For git metadata collection. Pure Go git implementation. |
| CI distribution | GitHub Action (`Clyra-AI/axym-action@v1`) | Primary integration point. Also Docker image for other CI systems. |
| Testing | Go stdlib `testing` + `testify` | Consistent with Gait and Clyra-AI/proof. |
| Distribution | `goreleaser` → Homebrew tap (`Clyra-AI/tap/axym`) + GitHub releases + Docker image | Single static binary for every platform. |

### Data Model (core entities)

Axym uses `proof.Record` from `Clyra-AI/proof` as its atomic unit. The types below are Axym-specific — compliance opinions that live in the axym CLI, not in the shared primitive.

```go
// axym-specific types — compliance opinions on top of proof.Record

// ComplianceTag maps a proof record to regulatory controls.
// Applied by axym during mapping, not stored in the proof record itself.
type ComplianceTag struct {
    Framework string   `json:"framework"`  // "eu-ai-act" | "soc2" | "sox" | etc.
    Controls  []string `json:"controls"`   // specific control identifiers
}

// ComplianceCoverage tracks evidence coverage for a single control.
type ComplianceCoverage struct {
    Framework        string          `json:"framework"`
    ControlID        string          `json:"control_id"`
    Status           CoverageStatus  `json:"status"` // "covered" | "partial" | "gap"
    RecordCount      int             `json:"record_count"`
    LastRecordDate   *time.Time      `json:"last_record_date,omitempty"`
    Gaps             []ComplianceGap `json:"gaps,omitempty"`
}

type CoverageStatus string
const (
    Covered CoverageStatus = "covered"
    Partial CoverageStatus = "partial"
    Gap     CoverageStatus = "gap"
)

// ComplianceGap describes a specific evidence gap.
type ComplianceGap struct {
    ID          string          `json:"id"`
    ControlID   string          `json:"control_id"`
    Title       string          `json:"title"`
    Severity    string          `json:"severity"` // "critical" | "high" | "medium" | "low"
    Description string          `json:"description"`
    Remediation RemediationPlan `json:"remediation"`
}

// RemediationPlan describes how to fix a compliance gap.
type RemediationPlan struct {
    Action         string               `json:"action"`
    Steps          []string             `json:"steps"`
    TemplatePath   string               `json:"template_path,omitempty"`
    EstimatedEffort string              `json:"estimated_effort"`
    EvidenceNeeded []EvidenceRequirement `json:"evidence_needed"`
}

// EvidenceRequirement describes what proof records are needed.
type EvidenceRequirement struct {
    RecordType       string   `json:"record_type"`
    MinimumFrequency string   `json:"minimum_frequency"`
    RequiredFields   []string `json:"required_fields"`
}

// CollectorConfig defines configuration for a collector plugin.
type CollectorConfig struct {
    Type       string         `json:"type"`
    Enabled    bool           `json:"enabled"`
    Source     string         `json:"source"`
    Config     map[string]any `json:"config"`
    Redactions []Redaction    `json:"redactions,omitempty"`
}

// Redaction defines a field redaction rule.
type Redaction struct {
    Field  string `json:"field"`
    Action string `json:"action"` // "hash" | "omit" | "mask"
}

// Collector is the interface that collector plugins implement.
type Collector interface {
    Name() string
    Collect(config CollectorConfig) ([]proof.Record, error)
}
```

### The `axym-policy.yaml` (convention for repos)

This is the file that declares which AI systems are in scope, what evidence collection is active, and what frameworks apply. It's the bridge between engineering and GRC.

```yaml
# axym-policy.yaml — AI compliance evidence policy for this repo/org
# Reviewed and approved by: @priya (2026-09-10)
version: "1.0"
org: acme-corp

# AI systems in scope for compliance
systems:
  - id: "payments-agent"
    type: "claude-code-agent"
    risk_level: "high"           # EU AI Act classification
    environment: "production"
    owner: "@kai"
    mcp_servers:
      - "postgres-payments"
      - "stripe-api"
    frameworks:
      - "eu-ai-act"
      - "soc2"
      - "sox"
      - "texas-traiga"

  - id: "docs-assistant"
    type: "openai-assistant"
    risk_level: "limited"
    environment: "production"
    owner: "@frontend-team"
    frameworks:
      - "eu-ai-act"
      - "soc2"

  - id: "etl-pipeline"
    type: "dbt-pipeline"
    risk_level: "high"           # SOX-relevant
    environment: "production"
    owner: "@data-team"
    frameworks:
      - "sox"
      - "pci-dss"

# Evidence collection configuration
collection:
  sources:
    - type: "mcp-server"
      config:
        log_path: "/var/log/mcp/"
    - type: "github-actions"
      config:
        workflows: ["deploy.yml", "ai-tests.yml"]
    - type: "llm-api"
      config:
        middleware: "axym-middleware"
    - type: "dbt"
      config:
        project_dir: "/opt/dbt/acme"
        snowflake_account: "acme.us-east-1"
    - type: "eval"
      config:
        platform: "braintrust"           # or "arize", "langsmith", "custom"
        api_endpoint: "https://api.braintrust.dev"
        project: "payments-agent-evals"
    - type: "model-registry"
      config:
        platform: "mlflow"               # or "huggingface", "wandb"
        tracking_uri: "https://mlflow.internal"
    - type: "red-team"
      config:
        platform: "giskard"              # or "nemo-red-team", "custom"
        results_path: "./red-team-results/"

  redaction:
    - field: "event.parameters.query"
      action: "hash"             # Replace with SHA-256 hash via proof.Canonicalize()
    - field: "event.result.data"
      action: "omit"             # Remove entirely

  retention:
    days: 730                     # 2 years (EU AI Act minimum)

# Sibling product integration
siblings:
  wrkr:
    evidence_path: "./wrkr-evidence/"    # Ingest Wrkr scan findings
  gait:
    evidence_path: "./gait-out/"          # Ingest Gait enforcement records

# Compliance thresholds (CLI: exit non-zero when coverage drops below threshold)
thresholds:
  eu-ai-act:
    minimum_coverage: 80         # percent — axym gaps exits non-zero if below threshold
  soc2:
    minimum_coverage: 90
  sox:
    minimum_coverage: 95
```

-----

## Rollout Plan

### Week 1–4: Ship the "Evidence Gap" demo

The open-source CLI that does: collect from one source → produce proof records → map to EU AI Act → show gaps.

Distribution:

- Binary: `brew install Clyra-AI/tap/axym` + GitHub releases (goreleaser)
- Docker image: `ghcr.io/Clyra-AI/axym`
- GitHub repo with comprehensive README
- One blog post: "We ran an AI compliance audit on ourselves. Here's what was missing."
- Target audience: GRC professionals and security engineers searching for EU AI Act compliance tools
- Submit to r/netsec, r/compliance, GRC-focused LinkedIn groups

### Week 5–8: Add multi-source + bundle generation

- Multiple collector plugins (MCP, LLM API, GitHub Actions, Git, dbt)
- `axym bundle` for full audit package generation
- `axym verify` for chain and bundle integrity
- `axym ingest` for consuming Wrkr and Gait proof records
- Ticket integration: Jira/ServiceNow auto-attach with DLQ and retry
- Second blog post: "How we reduced AI audit prep from 3 weeks to 3 hours"
- GitHub Action published to marketplace
- Reach out to SOC 2 auditors for feedback on bundle format

### Week 9–12: Community + design partners

- 5–10 design partners at companies facing EU AI Act audits
- Framework mapping review with GRC consultants
- Auditor feedback loop (are the bundles useful? what's missing?)
- Plugin authoring guide for custom collectors
- First external contributor adds a new collector or framework mapping (to `Clyra-AI/proof`)
- `axym-policy.yaml` convention documented

### Week 13–20: Cross-product integration

- Demonstrate full See → Prove → Control loop: Wrkr scan → Axym evidence → Gait enforcement → back to Axym
- Publish integration guide: "Running the Clyra AI governance stack"
- Pitch the proof record format to agent framework maintainers via `Clyra-AI/proof` (lightweight SDKs for Python, TypeScript wrapping the JSON Schema + signing protocol)
- Target: first external tools emitting proof records that Axym can ingest

## Success Metrics (6-Month Targets)

| Metric | Target |
|---|---|
| GitHub stars | 1,500+ |
| Weekly active CLI users (`axym`) | 300+ |
| Proof records generated (total across Axym users) | 500,000+ |
| Audit bundles generated | 100+ |
| Design partners (paid pilot) | 8 |
| Framework mappings contributed by community (to `Clyra-AI/proof`) | 3+ new frameworks |
| External collector contributions | 5+ |
| Blog posts / case studies written by users | 5+ |
| Auditor endorsements / testimonials | 3+ |
| First paying Axym customer (OSS support/services or pilot) | Month 6 |
| First audit successfully closed using Axym evidence | Month 4 |
| Cross-product chains (Wrkr + Axym + Gait records in same chain) | 10+ organizations |

**The ecosystem health metric:** proof records from Wrkr and Gait that flow into Axym bundles. When this number grows, the Clyra AI governance loop is working.

-----

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Regulatory requirements change or get clarified after launch | High | Medium | Framework mappings are YAML in `Clyra-AI/proof`. New YAML file = new framework support. Community can contribute updated mappings within days. One PR to `Clyra-AI/proof`, all three products benefit. |
| Audit firms don't accept Axym evidence format | Medium | High | Evidence bundles contain proof records verifiable by the standalone `proof` CLI. Bundle format designed to be conservative (JSONL + PDF = universally readable). Auditor portal in Enterprise tier. Design partner program includes auditor feedback. |
| GRC platform incumbents add AI evidence features | Medium | High | Axym produces the evidence; GRC platforms consume it. Position as complementary, not competitive. Build export adapters to Vanta/Drata/ServiceNow. They need the evidence supply that Axym provides. Vanta/Drata manage compliance *workflows* (assign owners, track status, pull screenshots); they cannot generate *technical evidence* that AI agents are governed — they have no collector for MCP server logs, no proof chain, no framework-specific record-type mapping. Axym's moat is the evidence production layer: collectors at the agent runtime, signed proof records, and deterministic framework mapping. If incumbents add AI evidence, they'll likely add shallow integrations (API log ingestion, LLM call counts) — not the structured, control-mapped, chain-verified evidence that auditors need. |
| Engineering teams resist adding evidence collection to pipelines | Medium | Medium | Non-blocking design. < 50ms per event. Open source for trust. Frame as "protect yourself" not "comply with policy." The champion (Kai) adoption path avoids top-down mandate. |
| Framework mappings are legally challenged as incorrect | Medium | High | Ship as "informational, not legal advice." Clearly disclaim. Partner with GRC firms for Enterprise-tier validated mappings. Community review for open source mappings. |
| Evidence volume creates storage cost concerns | Low | Medium | Data minimization by design (hashes, not payloads). Configurable retention. Evidence store is local filesystem — teams manage storage as they do logs. |
| Competitors copy the proof record format | Medium | Low | This is actually a win. If the format becomes a standard, Clyra AI is the reference implementation. Network effects from tooling ecosystem (auditor tools, GRC integrations) built on the format. |
| Axym can't ingest evidence from Gait's existing packs | Low | Medium | Gait's native types (`gait.gate.trace`, `gait.gate.approval_token`, `gait.gate.delegation_token`) use the same crypto foundations (Ed25519, JCS, SHA-256) as proof records. GaitIngestor translates native types to proof records mechanically. If Gait packs include `proof_records.jsonl`, those are ingested directly. Integration guide documents both paths. |
| EU AI Act enforcement is weaker than expected | Low | Medium | SOC 2 AI controls, SOX, PCI-DSS, and state laws provide independent demand. The "prove compliance" need exists regardless of enforcement intensity. Clyra DNA extends Axym into data pipeline governance (SOX/PCI-DSS) beyond AI-specific regulations. |
| Open source contributors submit inaccurate framework mappings | Medium | Medium | Framework mappings require review before merge to `Clyra-AI/proof`. Validated mappings with professional attestation are an Enterprise feature. Clearly mark community-contributed mappings as "community-reviewed, not legally validated." |
| Layer 4 collectors (eval, model registry, red team) are thin — listed in PRD but implementation depth is unproven | High | Medium | v1 priority is Layer 1 (agent runtime) and Layer 3 (data pipelines from Clyra DNA). Layer 4 collectors ship as "beta" with minimal config: read output files/API, extract structured fields, emit proof records. Design partner feedback in weeks 9-12 determines which Layer 4 collectors need depth (e.g., Braintrust eval collector may need dataset version tracking, MLflow collector may need model lineage). Thin is acceptable — the collector interface is simple (`Collect() → []proof.Record`), so depth can be added incrementally without breaking the architecture. The risk is not shipping thin collectors; the risk is *not shipping them at all* and missing the evidence surface. |
| Free/paid line is undefined — risk of giving away too much or gating too early | Medium | High | **OSS (free forever):** Full CLI — `collect`, `map`, `gaps`, `bundle`, `verify`, `regress`, `replay`, `ingest`. Everything a platform engineer needs to produce evidence, generate bundles, and verify integrity locally. No usage limits, no org size limits, no feature gating. Evidence production never requires a license. The proof record format, YAML compliance mappings, and JSON Schema spec are radically free to produce — gating them would kill format adoption. **Clyra Platform (paid):** One unified product, not separate Pro tiers. Dashboard (compliance coverage over time), continuous collection (always-on daemon, not CLI-triggered), change detection between collection cycles, Slack/Teams alerts on coverage drops and gap detection, GRC platform integrations (Vanta/Drata/ServiceNow push), SIEM export (Splunk/Sentinel/Elastic), multi-framework view, team access with RBAC, SSO/SCIM, API access, nightly Daily Review generation with email delivery. The upgrade trigger: "I ran this once and the gaps were alarming — now I need to watch continuously and share with my team." Platform captures GRC and security budget. The developer runs `axym config set platform.endpoint` and the same commands feed organizational infrastructure. **Clyra Enterprise:** Replay certification tiers (A/B/C) — producing evidence is free, certifying that evidence is replay-verifiable is Enterprise. Validated compliance mappings with professional attestation (reviewed by qualified legal and GRC professionals — distinct from community-reviewed YAML). Auditor portal (read-only access for external audit firms). Audit-ready package generation in auditor-specific deliverable formats. Custom framework development for industry-specific or proprietary compliance frameworks. Evidence retention and archival policies meeting regulatory retention requirements. Executive reporting with board-ready summaries. Dedicated support with SLA. Data residency. The upgrade trigger: "The auditor needs to see this" or "The board needs to sign off on this." Enterprise captures GRC and audit budget. **The principle: never gate the proof format itself.** Never gate the ability to produce or locally verify a proof record. Never gate the YAML compliance mappings or the JSON Schema spec. These are the format adoption surfaces. What you charge for is organizational meaning, continuous verification, and auditor trust built on top of that free format. |
