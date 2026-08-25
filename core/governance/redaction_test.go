package governance

import "testing"

func TestRedactTelemetrySpanSensitiveOTLPClasses(t *testing.T) {
	span := TraceSpan{Attributes: map[string]string{
		"http.request.header.authorization": "Bearer raw",
		"authorization":                     "raw-auth",
		"api-key":                           "raw-api-key",
		"x_api_key":                         "raw-x-api-key",
		"gen_ai.prompt":                     "raw-prompt",
		"input.content":                     "raw-input",
		"output.content":                    "raw-output",
		"db.statement":                      "SELECT secret",
		"database-statement":                "UPDATE secret",
		"sql.query":                         "SELECT raw",
		"safe.attribute":                    "safe",
	}}
	got := RedactTelemetrySpan(span)
	for key, value := range got.Attributes {
		if key != "safe.attribute" && value != "[REDACTED]" {
			t.Fatalf("sensitive attribute %q leaked: %q", key, value)
		}
	}
	if got.Attributes["safe.attribute"] != "safe" {
		t.Fatal("safe attribute changed")
	}
}

func TestRedactedTelemetryDigestRecomputes(t *testing.T) {
	s := TraceSpan{TraceID: "0123456789abcdef0123456789abcdef", SpanID: "0123456789abcdef", StartTime: "2026-01-01T00:00:00Z", EndTime: "2026-01-01T00:00:01Z", Source: "otel", Attributes: map[string]string{"authorization": "raw"}}
	r := RedactTelemetrySpan(s)
	r.Digest = ""
	d, e := Digest(r)
	if e != nil {
		t.Fatal(e)
	}
	r.Digest = d
	check := r
	check.Digest = ""
	again, e := Digest(check)
	if e != nil || again != d {
		t.Fatalf("redacted digest mismatch: %s %s %v", d, again, e)
	}
}
