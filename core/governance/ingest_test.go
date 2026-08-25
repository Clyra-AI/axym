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
	if err != nil || len(got.SourceDigests) != 1 {
		t.Fatalf("telemetry: %+v %v", got, err)
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
