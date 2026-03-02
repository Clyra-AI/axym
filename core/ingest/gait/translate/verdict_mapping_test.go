package translate

import (
	"testing"

	"github.com/Clyra-AI/proof"
)

func TestTranslateMapsNativeTypesToProofRecordTypes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		nativeType string
		wantType   string
	}{
		{nativeType: NativeTypeTrace, wantType: "tool_invocation"},
		{nativeType: NativeTypeApprovalToken, wantType: "approval"},
		{nativeType: NativeTypeDelegationToken, wantType: "policy_enforcement"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.nativeType, func(t *testing.T) {
			t.Parallel()
			record, err := Translate(NativeRecord{
				Type:      tc.nativeType,
				Timestamp: "2026-02-28T20:00:00Z",
				AgentID:   "agent-a",
				Event: map[string]any{
					"decision": "allow",
				},
			})
			if err != nil {
				t.Fatalf("Translate: %v", err)
			}
			if record.RecordType != tc.wantType {
				t.Fatalf("record type mismatch: got %s want %s", record.RecordType, tc.wantType)
			}
			if got, _ := record.Metadata["gait_native_type"].(string); got != tc.nativeType {
				t.Fatalf("missing gait native type metadata: %+v", record.Metadata)
			}
		})
	}
}

func TestTranslatePreservesRelationshipEnvelope(t *testing.T) {
	t.Parallel()

	relationship := &proof.Relationship{
		ParentRef: &proof.RelationshipRef{Kind: "trace", ID: "parent-1"},
		EntityRefs: []proof.RelationshipRef{
			{Kind: "resource", ID: "resource-1"},
		},
	}
	record, err := Translate(NativeRecord{
		Type:         NativeTypeTrace,
		Timestamp:    "2026-02-28T20:00:00Z",
		Event:        map[string]any{"tool_name": "planner"},
		Relationship: relationship,
	})
	if err != nil {
		t.Fatalf("Translate: %v", err)
	}
	if record.Relationship == nil {
		t.Fatalf("expected relationship to be preserved")
	}
	if record.Relationship.ParentRef == nil || record.Relationship.ParentRef.ID != "parent-1" {
		t.Fatalf("parent relationship mismatch: %+v", record.Relationship)
	}
}

func TestTranslateRejectsUnsupportedNativeType(t *testing.T) {
	t.Parallel()

	_, err := Translate(NativeRecord{
		Type:      "unknown",
		Timestamp: "2026-02-28T20:00:00Z",
		Event:     map[string]any{"decision": "allow"},
	})
	if err == nil {
		t.Fatal("expected error for unsupported native type")
	}
}
