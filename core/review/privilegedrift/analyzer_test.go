package privilegedrift

import "testing"

func TestAnalyzeProducesDeterministicUnapprovedGaps(t *testing.T) {
	t.Parallel()

	baseline := map[string][]string{
		"agent-a": {"read"},
	}
	observations := []Observation{
		{Principal: "agent-a", Privilege: "write", Approved: false, RecordID: "r2"},
		{Principal: "agent-a", Privilege: "read", Approved: false, RecordID: "r1"},
		{Principal: "agent-b", Privilege: "admin", Approved: false, RecordID: "r3"},
	}

	updated, gaps := Analyze(baseline, observations)

	if len(gaps) != 2 {
		t.Fatalf("gap count mismatch: got %d", len(gaps))
	}
	if gaps[0].Principal != "agent-a" || gaps[0].Privilege != "write" {
		t.Fatalf("unexpected first gap: %+v", gaps[0])
	}
	if gaps[1].Principal != "agent-b" || gaps[1].Privilege != "admin" {
		t.Fatalf("unexpected second gap: %+v", gaps[1])
	}
	if got := updated["agent-a"]; len(got) != 2 || got[0] != "read" || got[1] != "write" {
		t.Fatalf("agent-a baseline mismatch: %#v", got)
	}
}

func TestAnalyzeSkipsApprovedNewPrivilege(t *testing.T) {
	t.Parallel()

	updated, gaps := Analyze(nil, []Observation{
		{Principal: "agent-a", Privilege: "write", Approved: true, RecordID: "r1"},
	})

	if len(gaps) != 0 {
		t.Fatalf("expected no gaps, got %+v", gaps)
	}
	if got := updated["agent-a"]; len(got) != 1 || got[0] != "write" {
		t.Fatalf("updated baseline mismatch: %#v", got)
	}
}
