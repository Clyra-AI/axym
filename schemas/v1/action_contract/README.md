# Axym Action Contract Compatibility

Axym consumes, but does not own, the currently released producer artifacts:

- Wrkr `proposed_action_contract` artifact schema `1`, embedded contract schema `3`.
- Gait `activated_action_contract` artifact schema `1` from the merged activation boundary at commit `0897620`.

The three producer schemas plus the Axym-owned consumer-receipt schema are
published in Axym's versioned schema tree and embedded by the consumer
package. The nine Wrkr artifact bytes under
`testdata/action-contract-interop/v1/expected/` are pinned to the released
Wrkr fixture manifest (`v1.14.0`, fixture version `1`). Axym preserves the
producer-native envelope and contract map; it does not translate a report-only
proposal into authority.

Activation verification requires an exact current-selection assertion for the
proposal family/revision, an explicit UTC evaluation time, and a public key.
Development-signed activations are rejected by default; an explicit test-only
allowance still produces `unverifiable`, never an authentic/verified result.
`enforce_floor` conformance reports whether required proposal fields remain
bound or are explicitly excepted. A tightened result is emitted only when an
actual control projection proves a stricter value; activation mode alone is
not evidence of tightening. `context_only` is labeled separately as
non-binding context, and an unverifiable input can never be `valid: true`.
Execution, effects, containment, compensation events, telemetry authenticity,
and policy correctness are intentionally out of scope until released producer
artifacts exist for those surfaces.

The executable conformance consumer is built with:

```bash
go build -o axym-action-contract-consumer ./cmd/axym-action-contract-consumer
export WRKR_AXYM_ACTION_CONTRACT_CONSUMER="$PWD/axym-action-contract-consumer"
"$WRKR_AXYM_ACTION_CONTRACT_CONSUMER" <one-proposed-action-contract.json>
```

Every receipt sets `self_attestation: false` and contains the raw artifact
SHA-256, producer IDs, revision, correlation references, schema versions, and
deterministic semantic classification. The exact Gait activation golden is
blocked until the corresponding Gait v1.4 fixture is committed and released;
the current generated activation input is unit-test-only and is not claimed as
producer compatibility evidence.
