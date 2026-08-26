package governance

import (
	"encoding/json"
	"testing"
	"time"
)

func TestProjectionProofIsSourceQualifiedAndNonAuthoritative(t *testing.T) {
	r, err := ToProofRecord("test_result", "judge", "judge", "j", time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), map[string]any{"verdict": "pass"}, []Ref{{Kind: "evidence", ID: "j", Digest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Source: "judge", SourceProduct: "judge", SchemaID: "judge/v1/evidence", SchemaVersion: "v1"}})
	if err != nil {
		t.Fatal(err)
	}
	if r.Event["execution_authority"] != false || r.Event["source_claim"] != true {
		t.Fatalf("unsafe projection: %#v", r.Event)
	}
}

func TestTelemetryAndJudgeAreBounded(t *testing.T) {
	raw, _ := json.Marshal(map[string]any{"spans": []any{}, "attestations": []any{}})
	got, err := IngestTelemetry(raw, time.Now(), time.Hour, "")
	if err != nil || len(got.SourceDigests) != 1 || got.IntegrityState != "absent" || got.FreshnessState != "absent" || len(got.ReasonCodes) != 1 || got.ReasonCodes[0] != ReasonTelemetryMissing {
		t.Fatalf("telemetry: %+v %v", got, err)
	}
	otlp, err := IngestOTLP([]byte(`{"resourceSpans":[]}`), OTLPOptions{})
	if err != nil || otlp.IntegrityState != "absent" || otlp.FreshnessState != "absent" || len(otlp.ReasonCodes) != 1 || otlp.ReasonCodes[0] != ReasonTelemetryMissing {
		t.Fatalf("empty OTLP telemetry was verified: %+v %v", otlp, err)
	}
	p, err := ProjectJudge(JudgeEvidence{ID: "j", ContractRef: Ref{ID: "c", Kind: "contract", Digest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Source: "wrkr", SourceProduct: "wrkr", SchemaID: "wrkr/v1/contract", SchemaVersion: "v1"}, Verdict: "pass", Source: "judge", Digest: "sha256:" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}, "c")
	if err != nil || !p.Advisory || p.ExecutionAuthority {
		t.Fatalf("judge projection: %+v %v", p, err)
	}
}
func TestTimelineReplayDeterministic(t *testing.T) {
	t1, e1 := ProjectTimeline("c", []Event{{ID: "b", ContractRef: Ref{ID: "c"}, Kind: "execution_succeeded", OccurredAt: "2026-01-04T00:00:00Z"}, {ID: "a", ContractRef: Ref{ID: "c"}, Kind: "execution_started", OccurredAt: "2026-01-03T00:00:00Z"}, {ID: "p", ContractRef: Ref{ID: "c"}, Kind: "proposed", OccurredAt: "2026-01-01T00:00:00Z"}, {ID: "v", ContractRef: Ref{ID: "c"}, Kind: "activated", OccurredAt: "2026-01-02T00:00:00Z"}})
	t2, e2 := ProjectTimeline("c", []Event{{ID: "a", ContractRef: Ref{ID: "c"}, Kind: "execution_started", OccurredAt: "2026-01-03T00:00:00Z"}, {ID: "b", ContractRef: Ref{ID: "c"}, Kind: "execution_succeeded", OccurredAt: "2026-01-04T00:00:00Z"}, {ID: "p", ContractRef: Ref{ID: "c"}, Kind: "proposed", OccurredAt: "2026-01-01T00:00:00Z"}, {ID: "v", ContractRef: Ref{ID: "c"}, Kind: "activated", OccurredAt: "2026-01-02T00:00:00Z"}})
	g1, _ := ProjectGraph(t1)
	g2, _ := ProjectGraph(t2)
	if e1 != nil || e2 != nil || t1.State.Status != t2.State.Status || g1.Nodes[0].ID != g2.Nodes[0].ID {
		t.Fatal("projection is not deterministic")
	}
}

func TestEvidenceEventMappingPreservesTerminalSemantics(t *testing.T) {
	checks := []struct {
		kind, state, want string
	}{
		{"execution", "failed", "execution_failed"},
		{"execution", "gap", "execution_failed"},
		{"execution", "blocked", "execution_blocked"},
		{"effect", "rejected", "effect_rejected"},
		{"effect", "gap", "effect_rejected"},
		{"effect", "unknown", "effect_unresolved"},
		{"effect", "unresolved", "effect_unresolved"},
		{"containment", "unresolved", "containment_unresolved"},
		{"containment", "gap", "containment_unresolved"},
		{"containment", "partial", "containment_partial"},
		{"compensation", "started", "compensation_started"},
		{"compensation", "completed", "compensated"},
		{"compensation", "failed", "compensation_failed"},
		{"compensation", "unresolved", "compensation_unresolved"},
		{"compensation", "required", "compensation_required"},
		{"compensation", "not_required", "compensation_not_required"},
	}
	for _, check := range checks {
		if got := EvidenceEventKind(check.kind, check.state); got != check.want {
			t.Errorf("EvidenceEventKind(%q, %q) = %q, want %q", check.kind, check.state, got, check.want)
		}
	}
	ref := verifiedRef("c")
	packet := Packet{Evidence: []Evidence{
		{Kind: "execution", Ref: Ref{ID: "e", Kind: "execution", Digest: ref.Digest, Source: "gait", SourceProduct: "gait", SchemaID: "gait/v1/execution", SchemaVersion: "1"}, ContractRef: ref, Attributes: map[string]string{"state": "failed"}, OccurredAt: "2026-01-01T00:00:00Z"},
		{Kind: "effect", Ref: Ref{ID: "f", Kind: "effect_event", Digest: ref.Digest, Source: "gait", SourceProduct: "gait", SchemaID: "gait/v1/effect", SchemaVersion: "1"}, ContractRef: ref, Attributes: map[string]string{"state": "rejected"}, OccurredAt: "2026-01-01T00:00:01Z"},
		{Kind: "containment", Ref: Ref{ID: "g", Kind: "containment", Digest: ref.Digest, Source: "gait", SourceProduct: "gait", SchemaID: "gait/v1/containment", SchemaVersion: "1"}, ContractRef: ref, Attributes: map[string]string{"state": "unresolved"}, OccurredAt: "2026-01-01T00:00:02Z"},
		{Kind: "compensation", Ref: Ref{ID: "h", Kind: "compensation", Digest: ref.Digest, Source: "gait", SourceProduct: "gait", SchemaID: "gait/v1/compensation", SchemaVersion: "1"}, ContractRef: ref, Attributes: map[string]string{"state": "completed"}, OccurredAt: "2026-01-01T00:00:03Z"},
	}}
	events := EventsFromPacket(packet)
	if len(events) != 4 || events[0].Kind != "execution_failed" || events[1].Kind != "effect_rejected" || events[2].Kind != "containment_unresolved" || events[2].Status != "unresolved" || events[3].Kind != "compensated" {
		t.Fatalf("terminal semantics were not preserved: %+v", events)
	}
}

func TestPacketCompletenessUsesTerminalStateIndependentOfRefSorting(t *testing.T) {
	contractRef := verifiedRef("c")
	packet, err := BuildPacket(Contract{ID: "c", FamilyID: "f", Revision: 1, Action: "a", Target: "t", Environment: "e", Provenance: contractRef}, []Evidence{
		{Kind: "execution", Ref: Ref{ID: "z-start", Kind: "execution", Digest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", Source: "gait", SourceProduct: "gait", SchemaID: "gait/v1/execution", SchemaVersion: "1"}, ContractRef: contractRef, Attributes: map[string]string{"state": "started"}, OccurredAt: "2026-01-01T00:00:01Z", Provenance: contractRef},
		{Kind: "execution", Ref: Ref{ID: "a-complete", Kind: "execution", Digest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc", Source: "gait", SourceProduct: "gait", SchemaID: "gait/v1/execution", SchemaVersion: "1"}, ContractRef: contractRef, Attributes: map[string]string{"state": "succeeded", "terminal": "true"}, OccurredAt: "2026-01-01T00:00:02Z", Provenance: contractRef},
	})
	if err != nil {
		t.Fatal(err)
	}
	if packet.AxisStates["execution"] != "present" {
		t.Fatalf("terminal execution state was overwritten by ID order: %+v", packet.AxisStates)
	}
}
