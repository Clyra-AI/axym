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
