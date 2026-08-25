package governance

import "strings"

// RedactTelemetrySpan removes sensitive OTLP attributes before a span can be
// projected into a signed Proof record. Classification is punctuation
// insensitive so authorization, api-key and database.statement variants are
// treated consistently.
func RedactTelemetrySpan(span TraceSpan) TraceSpan {
	out := span
	out.Attributes = make(map[string]string, len(span.Attributes))
	for key, value := range span.Attributes {
		if telemetrySensitiveKey(key) {
			out.Attributes[key] = "[REDACTED]"
		} else {
			out.Attributes[key] = value
		}
	}
	return out
}

func telemetrySensitiveKey(key string) bool {
	normalized := strings.NewReplacer(".", "", "-", "", "_", "", "/", "").Replace(strings.ToLower(strings.TrimSpace(key)))
	for _, marker := range []string{"authorization", "authheader", "apikey", "token", "secret", "password", "prompt", "input", "output", "dbstatement", "databasestatement", "sql", "query"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}
