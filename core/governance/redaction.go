package governance

import (
	"encoding/json"
	"strings"
)

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

// RedactRegister applies the customer-safe projection to governed contract
// axes while retaining IDs and references needed for stable joins.
func RedactRegister(register Register) Register {
	register = redactRegisterValue(register)
	if digest, err := Digest(register.Contracts); err == nil {
		register.SourceDigest = digest
	}
	return register
}

// RedactPacket applies the same projection to packet evidence. Digest and
// signature are cleared because callers must recompute them after redaction.
func RedactPacket(packet Packet) Packet {
	packet = redactPacketValue(packet)
	packet.Digest = ""
	packet.Signature = nil
	return packet
}

func redactRegisterValue(value Register) Register {
	redacted := redactJSON(value)
	_ = json.Unmarshal(redacted, &value)
	return value
}

func redactPacketValue(value Packet) Packet {
	redacted := redactJSON(value)
	_ = json.Unmarshal(redacted, &value)
	return value
}

func redactJSON(value any) []byte {
	raw, err := json.Marshal(value)
	if err != nil {
		return raw
	}
	var object any
	if json.Unmarshal(raw, &object) != nil {
		return raw
	}
	redactJSONValue(object, "")
	redacted, err := json.Marshal(object)
	if err != nil {
		return raw
	}
	return redacted
}

func redactJSONValue(value any, key string) {
	if sensitiveProjectionKey(key) {
		return
	}
	switch typed := value.(type) {
	case map[string]any:
		for childKey, child := range typed {
			if sensitiveProjectionKey(childKey) || (governedAxisKey(key) && sensitiveAxisValueKey(childKey)) {
				typed[childKey] = stableRedactedDigest(child)
				continue
			}
			redactJSONValue(child, childKey)
		}
	case []any:
		for _, child := range typed {
			redactJSONValue(child, key)
		}
	}
}

func sensitiveProjectionKey(key string) bool {
	normalized := strings.NewReplacer(".", "", "-", "", "_", "", "/", "").Replace(strings.ToLower(strings.TrimSpace(key)))
	for _, marker := range []string{"apikey", "token", "secret", "password", "prompt", "input", "output", "dbstatement", "databasestatement", "sql"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func governedAxisKey(key string) bool {
	normalized := strings.NewReplacer(".", "", "-", "", "_", "", "/", "").Replace(strings.ToLower(strings.TrimSpace(key)))
	for _, marker := range []string{"authorization", "authority", "precondition", "confirmation", "approval", "credential", "delegation", "effect", "compensation", "outcome", "judge"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func sensitiveAxisValueKey(key string) bool {
	normalized := strings.NewReplacer(".", "", "-", "", "_", "", "/", "").Replace(strings.ToLower(strings.TrimSpace(key)))
	for _, marker := range []string{"value", "content", "constraint", "detail", "explanation", "sourceuri", "token", "secret", "payload", "statement"} {
		if strings.Contains(normalized, marker) {
			return true
		}
	}
	return false
}

func stableRedactedDigest(value any) string {
	digest, err := Digest(value)
	if err != nil {
		return "[REDACTED]"
	}
	return digest
}
