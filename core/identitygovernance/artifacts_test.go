package identitygovernance

import (
	"testing"

	"github.com/Clyra-AI/axym/core/normalize"
	"github.com/Clyra-AI/proof"
)

func TestPrivilegeDriftFindingPrefersExplicitEventApproval(t *testing.T) {
	t.Parallel()

	finding, ok := privilegeDriftFinding(proof.Record{
		RecordID:   "rec-1",
		RecordType: "scan_finding",
		Event: map[string]any{
			"privilege": "admin",
			"approved":  false,
		},
		Metadata: map[string]any{
			"approved": true,
		},
	}, normalize.IdentityView{})
	if !ok {
		t.Fatal("expected privilege drift finding")
	}
	if finding.Approved {
		t.Fatalf("expected explicit event.approved=false to win, got approved=true")
	}
	if finding.Status != "unapproved" {
		t.Fatalf("expected unapproved status, got %q", finding.Status)
	}
}
