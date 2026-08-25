package main

import (
	"bytes"
	"encoding/json"
	"github.com/Clyra-AI/axym/core/governance"
	"github.com/Clyra-AI/axym/core/store"
	"os"
	"path/filepath"
	"testing"
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
