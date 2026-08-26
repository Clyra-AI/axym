package store

import (
	"errors"
	"testing"
	"time"

	"github.com/Clyra-AI/proof"
)

func TestAppendRejectsCallerLifecycleSpoof(t *testing.T) {
	st, err := New(Config{RootDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	record := testLifecycleAggregate(t)
	if _, err := st.Append(record, "spoof"); err == nil {
		t.Fatal("caller-authored Gait lifecycle metadata was appended")
	}
}

func TestAppendVerifiedLifecycleRequiresBoundReceipt(t *testing.T) {
	st, err := New(Config{RootDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	key, err := st.SigningKey()
	if err != nil {
		t.Fatal(err)
	}
	record := testLifecycleAggregate(t)
	if _, err := st.AppendVerifiedLifecycle(record, "lifecycle"); err == nil {
		t.Fatal("lifecycle without receipt was appended")
	}
	if err := SignLifecycleReceipt(record, key, "2026-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	record.Event["gait_execution"] = "failed"
	if _, err := st.AppendVerifiedLifecycle(record, "tampered"); err == nil {
		t.Fatal("tampered lifecycle retained receipt")
	}
	record.Event["gait_execution"] = "succeeded"
	if _, err := st.AppendVerifiedLifecycle(record, "lifecycle"); err != nil {
		t.Fatalf("verified lifecycle append failed: %v", err)
	}
}

func TestAppendVerifiedLifecycleRollsBackWhenReceiptCommitFails(t *testing.T) {
	st, err := New(Config{RootDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	key, err := st.SigningKey()
	if err != nil {
		t.Fatal(err)
	}
	record := testLifecycleAggregate(t)
	if err := SignLifecycleReceipt(record, key, "2026-01-01T00:00:00Z"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.AppendVerifiedLifecycle(record, "lifecycle", func(proof.Record) error {
		return errors.New("registry unavailable")
	}); err == nil {
		t.Fatal("append succeeded despite registry commit failure")
	}
	chain, err := st.LoadChain()
	if err != nil {
		t.Fatal(err)
	}
	if len(chain.Records) != 0 {
		t.Fatalf("lifecycle append was not rolled back: %d records", len(chain.Records))
	}
}

func testLifecycleAggregate(t *testing.T) *proof.Record {
	t.Helper()
	record, err := proof.NewRecord(proof.RecordOpts{
		Timestamp:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		Source:        "gait",
		SourceProduct: "gait",
		AgentID:       "gait://v1",
		Type:          "test_result",
		Event: map[string]any{
			"contract_ref":   map[string]any{"id": "contract-1"},
			"evidence_refs":  []map[string]any{{"id": "execution-1", "kind": "execution"}},
			"gait_execution": "succeeded",
		},
		Metadata: map[string]any{
			"evidence_kind":                 "gait_lifecycle",
			"gait_evidence_set_id":          "gait_lifecycle_v1:1234567890abcdef",
			"gait_producer_version":         "v1.7.0",
			"gait_source_commit":            "0123456789012345678901234567890123456789",
			"gait_verification_state":       "verified",
			"gait_authoritative":            true,
			"gait_fixture_only":             false,
			"gait_source_artifact_digest":   "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
			"gait_source_artifact_digests":  []string{"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
			"gait_derived_evidence_digests": []string{"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	record.Integrity.RecordHash = ""
	return record
}
