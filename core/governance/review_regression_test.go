package governance

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"testing"
	"time"
)

func reviewRef() Ref {
	return Ref{ID: "c", Kind: "contract", Digest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Source: "gait", SourceProduct: "gait", SchemaID: "gait/v1/contract", SchemaVersion: "1"}
}
func TestReviewSignedDigestAndTelemetryRegression(t *testing.T) {
	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	b := BoundaryAttestation{SchemaID: "boundary/v1", SchemaVersion: "1", ID: "b", Boundary: "x", ContractRef: reviewRef(), ObservedAt: "2026-01-01T00:00:00Z", FreshUntil: "2026-01-02T00:00:00Z", Source: "gait", Advisory: true}
	b, _ = SignBoundary(b, priv)
	b.Boundary = "modified"
	b.Digest, _ = Digest(b)
	if e := VerifyBoundary(b, pub, time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC), "c"); e == nil || e.Error() != ReasonTampered {
		t.Fatalf("boundary stale signature accepted: %v", e)
	}
	j := JudgeEvidence{SchemaID: "judge/v1", SchemaVersion: "1", ID: "j", ContractRef: reviewRef(), Verdict: "pass", ObservedAt: "2026-01-01T00:00:00Z", FreshUntil: "2026-01-02T00:00:00Z", Source: "judge", ProviderVersion: "1", Advisory: true}
	j, _ = SignJudge(j, priv)
	j.Verdict = "modified"
	j.Digest, _ = Digest(j)
	if e := VerifyJudge(j, pub, time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC), "c"); e == nil || e.Error() != ReasonTampered {
		t.Fatalf("judge stale signature accepted: %v", e)
	}
}
func TestReviewTelemetryDigestScopeAndFreshness(t *testing.T) {
	s := TraceSpan{TraceID: "0123456789abcdef0123456789abcdef", SpanID: "0123456789abcdef", StartTime: "2026-01-01T00:00:00Z", EndTime: "2026-01-01T00:00:01Z", Source: "otel", Attributes: map[string]string{"contract.id": "wrong"}}
	s.Digest, _ = Digest(s)
	raw, _ := json.Marshal(map[string]any{"spans": []TraceSpan{s}})
	r, e := IngestTelemetry(raw, time.Date(2026, 1, 1, 0, 0, 2, 0, time.UTC), time.Hour, "expected")
	if e != nil || !containsReason(r.ReasonCodes, ReasonOutOfScope) {
		t.Fatalf("scope mismatch missing: %+v %v", r, e)
	}
	s.Attributes = nil
	s.Digest = "sha256:" + "0" + "123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	raw, _ = json.Marshal(map[string]any{"spans": []TraceSpan{s}})
	r, e = IngestTelemetry(raw, time.Date(2026, 1, 1, 0, 0, 2, 0, time.UTC), time.Hour, "")
	if e != nil || !containsReason(r.ReasonCodes, ReasonTampered) {
		t.Fatalf("tampered digest missing: %+v %v", r, e)
	}
	r, e = IngestTelemetry(raw, time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC), time.Hour, "")
	if e != nil || !containsReason(r.ReasonCodes, ReasonStale) {
		t.Fatalf("stale telemetry missing: %+v %v", r, e)
	}
}
func containsReason(v []string, w string) bool {
	for _, x := range v {
		if x == w {
			return true
		}
	}
	return false
}
