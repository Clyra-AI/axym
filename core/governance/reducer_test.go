package governance

import "testing"

func TestReduceIsDeterministicAndBounded(t *testing.T) {
	c := Contract{ID: "c", FamilyID: "f", Revision: 1, Action: "deploy", Target: "prod", Environment: "prod", PolicyDigest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Provenance: Ref{Digest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Source: "wrkr", SourceProduct: "wrkr", SchemaID: "wrkr/v1/contract", SchemaVersion: "v1"}}
	if err := VerifyLineage(c, []Ref{{Kind: "target", ID: "prod", Digest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Source: "gait", SourceProduct: "gait", SchemaID: "gait/v1/target", SchemaVersion: "v1"}}); err != nil {
		t.Fatal(err)
	}
	if err := VerifyLineage(c, []Ref{{Kind: "target", ID: "staging", Digest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Source: "gait", SourceProduct: "gait", SchemaID: "gait/v1/target", SchemaVersion: "v1"}}); err == nil {
		t.Fatal("authority expansion accepted")
	}
	events := []Event{{ID: "b", ContractRef: Ref{ID: "c"}, Kind: "execution_succeeded", OccurredAt: "2026-01-04T00:00:00Z"}, {ID: "a", ContractRef: Ref{ID: "c"}, Kind: "execution_started", OccurredAt: "2026-01-03T00:00:00Z"}, {ID: "p", ContractRef: Ref{ID: "c"}, Kind: "proposed", OccurredAt: "2026-01-01T00:00:00Z"}, {ID: "v", ContractRef: Ref{ID: "c"}, Kind: "activated", OccurredAt: "2026-01-02T00:00:00Z"}}
	got, err := Reduce("c", events)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "succeeded" || !got.Complete || got.Events[2] != "a" {
		t.Fatalf("unexpected state %+v", got)
	}
}
