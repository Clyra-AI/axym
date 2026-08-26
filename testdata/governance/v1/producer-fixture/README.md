# Axym governed Action Contract producer fixture

This quarantined, fixture-only pack contains a deterministic governed register
and signed evidence packet, the public verification key, and the exact schemas
used to validate them. It is a producer-native Axym projection for later Proof
import; it is not a Proof-authored assessment and grants no runtime authority.

Regenerate and verify it with:

```bash
go run ./scripts/generate_governance_fixture.go --update
go run ./scripts/generate_governance_fixture.go --check
GOCACHE=/tmp/axym-gocache go test ./core/governance -run CheckedInProducerFixturePack -count=1
```
