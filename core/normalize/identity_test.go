package normalize

import (
	"testing"

	"github.com/Clyra-AI/proof"
)

func TestTargetFromRelationshipIgnoresAgentRefs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		relationship *proof.Relationship
		wantKind     string
		wantID       string
	}{
		{
			name: "agent refs alone do not become targets",
			relationship: &proof.Relationship{
				EntityRefs: []proof.RelationshipRef{
					{Kind: "agent", ID: "svc://actor"},
					{Kind: "agent", ID: "svc://delegate"},
				},
			},
		},
		{
			name: "resource target is preserved after agent refs",
			relationship: &proof.Relationship{
				EntityRefs: []proof.RelationshipRef{
					{Kind: "agent", ID: "svc://actor"},
					{Kind: "resource", ID: "deploy://prod"},
				},
			},
			wantKind: "resource",
			wantID:   "deploy://prod",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			kind, id := targetFromRelationship(tc.relationship)
			if kind != tc.wantKind || id != tc.wantID {
				t.Fatalf("targetFromRelationship() = (%q, %q), want (%q, %q)", kind, id, tc.wantKind, tc.wantID)
			}
		})
	}
}
