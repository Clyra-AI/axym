package governance

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/proof"
)

const VerifiedLifecycleRegistryFile = "gait-verified-lifecycle.json"

type VerifiedLifecycleEntry struct {
	RecordID             string                    `json:"record_id"`
	RecordHash           string                    `json:"record_hash"`
	SourceArtifactDigest string                    `json:"source_artifact_digest"`
	Receipt              store.VerificationReceipt `json:"receipt"`
}

// RegisterVerifiedLifecycle is written only by the Gait ingest path after a
// lifecycle aggregate has passed Gait's trusted verification/config boundary.
func RegisterVerifiedLifecycle(root string, record proof.Record, signingKey proof.SigningKey) error {
	entry, ok := lifecycleRegistryEntry(record)
	if !ok {
		return fmt.Errorf("record is not a verified Gait lifecycle aggregate")
	}
	receipt, err := store.VerifyLifecycleReceipt(&record, signingKey.Public)
	if err != nil {
		return fmt.Errorf("verify lifecycle receipt: %w", err)
	}
	entry.Receipt = receipt
	path := filepath.Join(root, VerifiedLifecycleRegistryFile)
	entries := map[string]VerifiedLifecycleEntry{}
	// #nosec G304 -- path is derived from the managed store root and fixed registry filename.
	if raw, err := os.ReadFile(path); err == nil {
		var existing []VerifiedLifecycleEntry
		if err := json.Unmarshal(raw, &existing); err != nil {
			return fmt.Errorf("decode verified lifecycle registry: %w", err)
		}
		for _, candidate := range existing {
			if candidate.RecordID != "" {
				entries[candidate.RecordID] = candidate
			}
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	entries[entry.RecordID] = entry
	keys := make([]string, 0, len(entries))
	for key := range entries {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	ordered := make([]VerifiedLifecycleEntry, 0, len(keys))
	for _, key := range keys {
		ordered = append(ordered, entries[key])
	}
	raw, err := json.MarshalIndent(ordered, "", "  ")
	if err != nil {
		return err
	}
	return store.WriteJSONAtomic(path, append(raw, '\n'), true)
}

func VerifyRegisteredLifecycle(root string, record proof.Record, publicKey proof.PublicKey) bool {
	entry, ok := lifecycleRegistryEntry(record)
	if !ok {
		return false
	}
	receipt, err := store.VerifyLifecycleReceipt(&record, publicKey.Public)
	if err != nil {
		return false
	}
	entry.Receipt = receipt
	// #nosec G304 -- path is derived from the managed store root and fixed registry filename.
	raw, err := os.ReadFile(filepath.Join(root, VerifiedLifecycleRegistryFile))
	if err != nil {
		return false
	}
	var entries []VerifiedLifecycleEntry
	if json.Unmarshal(raw, &entries) != nil {
		return false
	}
	for _, candidate := range entries {
		if candidate == entry {
			return true
		}
	}
	return false
}

// VerifyLifecycleRecordsWithRegistry is the bundle/governance boundary for
// lifecycle records, including stores that have no separately retained Wrkr
// proposal. Every lifecycle aggregate must carry a valid receipt and an
// exact persisted registry entry before it can be treated as evidence.
func VerifyLifecycleRecordsWithRegistry(root string, records []proof.Record, publicKey proof.PublicKey) error {
	for _, record := range records {
		if !lifecycleRecord(record) {
			continue
		}
		if _, err := store.VerifyLifecycleReceipt(&record, publicKey.Public); err != nil {
			return fmt.Errorf("lifecycle record %s lacks a valid ingest receipt: %w", record.RecordID, err)
		}
		if _, ok := lifecycleRegistryEntry(record); !ok || !VerifyRegisteredLifecycle(root, record, publicKey) {
			return fmt.Errorf("lifecycle record %s lacks trusted Gait verification registry entry", record.RecordID)
		}
	}
	return nil
}

func lifecycleRegistryEntry(record proof.Record) (VerifiedLifecycleEntry, bool) {
	if record.Metadata == nil || record.SourceProduct != "gait" || record.Integrity.RecordHash == "" {
		return VerifiedLifecycleEntry{}, false
	}
	if stringFieldAC(record.Metadata, "evidence_kind") != "gait_lifecycle" || stringFieldAC(record.Metadata, "gait_verification_state") != "verified" || !boolFieldAC(record.Metadata, "gait_authoritative") || boolFieldAC(record.Metadata, "gait_fixture_only") {
		return VerifiedLifecycleEntry{}, false
	}
	source := stringFieldAC(record.Metadata, "gait_source_artifact_digest")
	if !validDigestAC(source) || !validDigestAC(record.Integrity.RecordHash) {
		return VerifiedLifecycleEntry{}, false
	}
	return VerifiedLifecycleEntry{RecordID: record.RecordID, RecordHash: record.Integrity.RecordHash, SourceArtifactDigest: source}, true
}
