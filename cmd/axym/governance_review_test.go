package main

import (
	"bytes"
	"encoding/json"
	"github.com/Clyra-AI/axym/core/governance"
	"github.com/Clyra-AI/axym/core/store"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGovernanceVerifyAtMalformedIsInvalidInput(t *testing.T) {
	if _, e := parseGovernanceTime("not-time"); e == nil {
		t.Fatal("malformed verify-at accepted")
	}
}
func TestGovernanceTelemetryInvalidBatchDoesNotAppend(t *testing.T) {
	dir := t.TempDir()
	valid := governance.TraceSpan{TraceID: "0123456789abcdef0123456789abcdef", SpanID: "0123456789abcdef", StartTime: "2026-01-01T00:00:00Z", EndTime: "2026-01-01T00:00:01Z", Source: "otel", Attributes: map[string]string{}}
	valid.Digest, _ = governance.Digest(valid)
	invalid := valid
	invalid.SpanID = "bad"
	raw, _ := json.Marshal(map[string]any{"spans": []governance.TraceSpan{valid, invalid}})
	input := filepath.Join(dir, "telemetry.json")
	if e := os.WriteFile(input, raw, 0600); e != nil {
		t.Fatal(e)
	}
	var out, err bytes.Buffer
	code := execute([]string{"governance", "emit", "--kind", "telemetry", "--input", input, "--store-dir", filepath.Join(dir, "store"), "--json"}, &out, &err)
	if code == 0 {
		t.Fatalf("invalid batch accepted: %s", out.String())
	}
	st, e := store.OpenReadOnly(store.Config{RootDir: filepath.Join(dir, "store")})
	if e == nil {
		chain, e := st.LoadChain()
		if e != nil {
			t.Fatal(e)
		}
		if len(chain.Records) != 0 {
			t.Fatalf("partial append occurred: %d", len(chain.Records))
		}
	}
}

func TestGovernanceEmitTelemetryPersistsRedactedTrace(t *testing.T) {
	dir := t.TempDir()
	now := time.Now().UTC()
	s := governance.TraceSpan{TraceID: "0123456789abcdef0123456789abcdef", SpanID: "0123456789abcdef", StartTime: now.Add(-time.Second).Format(time.RFC3339Nano), EndTime: now.Format(time.RFC3339Nano), Source: "otel", Attributes: map[string]string{"authorization": "Bearer raw-auth", "gen_ai.prompt": "raw-prompt", "db.statement": "SELECT raw-db"}}
	s.Digest, _ = governance.Digest(s)
	raw, _ := json.Marshal(map[string]any{"spans": []governance.TraceSpan{s}})
	input := filepath.Join(dir, "telemetry.json")
	if e := os.WriteFile(input, raw, 0600); e != nil {
		t.Fatal(e)
	}
	storeDir := filepath.Join(dir, "store")
	var out, err bytes.Buffer
	if code := execute([]string{"governance", "emit", "--kind", "telemetry", "--input", input, "--store-dir", storeDir, "--json"}, &out, &err); code != 0 {
		t.Fatalf("emit failed code=%d out=%s err=%s", code, out.String(), err.String())
	}
	st, e := store.OpenReadOnly(store.Config{RootDir: storeDir})
	if e != nil {
		t.Fatal(e)
	}
	chain, e := st.LoadChain()
	if e != nil || len(chain.Records) != 1 {
		t.Fatalf("chain load: %d %v", len(chain.Records), e)
	}
	encoded, _ := json.Marshal(chain.Records[0])
	for _, rawValue := range []string{"raw-auth", "raw-prompt", "SELECT raw-db"} {
		if bytes.Contains(encoded, []byte(rawValue)) {
			t.Fatalf("raw value persisted: %s", rawValue)
		}
	}
	trace := chain.Records[0].Event["trace"].(map[string]any)
	orig := trace["digest"].(string)
	trace["digest"] = ""
	recomputed, e := governance.Digest(trace)
	if e != nil || recomputed != orig {
		t.Fatalf("embedded digest mismatch: %s %s %v", recomputed, orig, e)
	}
}
