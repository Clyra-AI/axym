package store

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Clyra-AI/proof"
	proofcanon "github.com/Clyra-AI/proof/canon"
	proofsign "github.com/Clyra-AI/proof/signing"
)

// LifecycleReceiptMetadataKey is intentionally not accepted as an ordinary
// user assertion. Store.AppendVerifiedLifecycle verifies its signature and
// binds it to the exact aggregate payload before signing the chain record.
const LifecycleReceiptMetadataKey = "gait_verification_receipt"

const (
	LifecycleReceiptSchemaID      = "https://axym.dev/schemas/v1/governance/gait-verification-receipt.schema.json"
	LifecycleReceiptSchemaVersion = "1"
)

// VerificationReceipt is the ingest-specific cryptographic handoff between
// the verified Gait adapter and the Axym proof store. It is signed with the
// store key only after the Gait pack has passed its trusted producer checks.
// The signature covers the aggregate's content digest, so a caller cannot
// change lifecycle metadata or references and retain the attestation.
type VerificationReceipt struct {
	SchemaID             string              `json:"schema_id"`
	SchemaVersion        string              `json:"schema_version"`
	ReceiptID            string              `json:"receipt_id"`
	RecordID             string              `json:"record_id"`
	RecordContentDigest  string              `json:"record_content_digest"`
	SourceArtifactDigest string              `json:"source_artifact_digest"`
	EvidenceSetID        string              `json:"evidence_set_id"`
	ProducerVersion      string              `json:"producer_version"`
	SourceCommit         string              `json:"source_commit"`
	IssuedAt             string              `json:"issued_at"`
	SignerKeyID          string              `json:"signer_key_id"`
	Signature            proofsign.Signature `json:"signature"`
}

// SignLifecycleReceipt attaches an ingest receipt to a translated lifecycle
// aggregate. Callers must invoke this only after the Gait verifier returned
// authoritative evidence; the store enforces the cryptographic boundary.
func SignLifecycleReceipt(record *proof.Record, key proof.SigningKey, issuedAt string) error {
	if !isGaitLifecycleRecord(record) {
		return fmt.Errorf("record is not a gait lifecycle aggregate")
	}
	if _, ok := record.Metadata[LifecycleReceiptMetadataKey]; ok {
		return fmt.Errorf("gait lifecycle verification receipt already present")
	}
	contentDigest, err := lifecycleRecordContentDigest(record)
	if err != nil {
		return err
	}
	receipt := VerificationReceipt{
		SchemaID:             LifecycleReceiptSchemaID,
		SchemaVersion:        LifecycleReceiptSchemaVersion,
		ReceiptID:            "gait-ingest-receipt-v1:" + strings.TrimPrefix(contentDigest, "sha256:")[:16],
		RecordID:             record.RecordID,
		RecordContentDigest:  contentDigest,
		SourceArtifactDigest: stringValue(record.Metadata, "gait_source_artifact_digest"),
		EvidenceSetID:        stringValue(record.Metadata, "gait_evidence_set_id"),
		ProducerVersion:      stringValue(record.Metadata, "gait_producer_version"),
		SourceCommit:         stringValue(record.Metadata, "gait_source_commit"),
		IssuedAt:             issuedAt,
		SignerKeyID:          key.KeyID,
	}
	if receipt.SourceArtifactDigest == "" || receipt.EvidenceSetID == "" || receipt.ProducerVersion == "" || receipt.SourceCommit == "" || receipt.IssuedAt == "" {
		return fmt.Errorf("gait lifecycle provenance is incomplete")
	}
	digest, err := receiptDigest(receipt)
	if err != nil {
		return err
	}
	signature, err := proofsign.SignDigestHex(key.Private, strings.TrimPrefix(digest, "sha256:"))
	if err != nil {
		return fmt.Errorf("sign gait lifecycle verification receipt: %w", err)
	}
	receipt.Signature = signature
	raw, err := json.Marshal(receipt)
	if err != nil {
		return err
	}
	var value any
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	record.Metadata[LifecycleReceiptMetadataKey] = value
	return nil
}

// VerifyLifecycleReceipt verifies both the receipt signature and its binding
// to the current aggregate. A valid chain/store signature alone is not enough.
func VerifyLifecycleReceipt(record *proof.Record, publicKey ed25519.PublicKey) (VerificationReceipt, error) {
	if !isGaitLifecycleRecord(record) {
		return VerificationReceipt{}, fmt.Errorf("record is not a gait lifecycle aggregate")
	}
	raw, err := json.Marshal(record.Metadata[LifecycleReceiptMetadataKey])
	if err != nil {
		return VerificationReceipt{}, fmt.Errorf("decode gait lifecycle verification receipt: %w", err)
	}
	var receipt VerificationReceipt
	if err := json.Unmarshal(raw, &receipt); err != nil {
		return VerificationReceipt{}, fmt.Errorf("decode gait lifecycle verification receipt: %w", err)
	}
	if receipt.SchemaID != LifecycleReceiptSchemaID || receipt.SchemaVersion != LifecycleReceiptSchemaVersion || receipt.RecordID != record.RecordID || receipt.SignerKeyID != proofsign.KeyID(publicKey) || receipt.RecordContentDigest == "" || receipt.SourceArtifactDigest == "" || receipt.EvidenceSetID == "" || receipt.ProducerVersion == "" || receipt.SourceCommit == "" || receipt.IssuedAt == "" {
		return VerificationReceipt{}, fmt.Errorf("gait lifecycle verification receipt is malformed")
	}
	wantContent, err := lifecycleRecordContentDigest(record)
	if err != nil || wantContent != receipt.RecordContentDigest {
		return VerificationReceipt{}, fmt.Errorf("gait lifecycle verification receipt content mismatch")
	}
	wantReceiptDigest, err := receiptDigest(receipt)
	if err != nil || receipt.Signature.SignedDigest != strings.TrimPrefix(wantReceiptDigest, "sha256:") {
		return VerificationReceipt{}, fmt.Errorf("gait lifecycle verification receipt digest mismatch")
	}
	ok, err := proofsign.VerifyDigestHex(publicKey, receipt.Signature)
	if err != nil || !ok {
		return VerificationReceipt{}, fmt.Errorf("gait lifecycle verification receipt signature invalid")
	}
	for _, field := range []struct{ name, key string }{
		{"source artifact digest", "gait_source_artifact_digest"},
		{"evidence set id", "gait_evidence_set_id"},
		{"producer version", "gait_producer_version"},
		{"source commit", "gait_source_commit"},
	} {
		if stringValue(record.Metadata, field.key) != receiptValue(receipt, field.key) {
			return VerificationReceipt{}, fmt.Errorf("gait lifecycle verification receipt %s mismatch", field.name)
		}
	}
	return receipt, nil
}

func receiptValue(receipt VerificationReceipt, key string) string {
	switch key {
	case "gait_source_artifact_digest":
		return receipt.SourceArtifactDigest
	case "gait_evidence_set_id":
		return receipt.EvidenceSetID
	case "gait_producer_version":
		return receipt.ProducerVersion
	case "gait_source_commit":
		return receipt.SourceCommit
	default:
		return ""
	}
}

func lifecycleRecordContentDigest(record *proof.Record) (string, error) {
	if record == nil {
		return "", fmt.Errorf("record is nil")
	}
	copy := *record
	copy.Integrity = proof.Integrity{}
	metadata := map[string]any{}
	for key, value := range record.Metadata {
		if key != LifecycleReceiptMetadataKey {
			metadata[key] = value
		}
	}
	copy.Metadata = metadata
	raw, err := json.Marshal(copy)
	if err != nil {
		return "", err
	}
	digest, err := proofcanon.DigestJCS(raw)
	if err != nil {
		return "", err
	}
	return "sha256:" + strings.TrimPrefix(digest, "sha256:"), nil
}

func receiptDigest(receipt VerificationReceipt) (string, error) {
	copy := receipt
	copy.Signature = proofsign.Signature{}
	raw, err := json.Marshal(copy)
	if err != nil {
		return "", err
	}
	digest, err := proofcanon.DigestJCS(raw)
	if err != nil {
		return "", err
	}
	return "sha256:" + strings.TrimPrefix(digest, "sha256:"), nil
}

func stringValue(object map[string]any, key string) string {
	value, _ := object[key].(string)
	return strings.TrimSpace(value)
}

func isGaitLifecycleRecord(record *proof.Record) bool {
	return record != nil && record.SourceProduct == "gait" && record.Metadata != nil && stringValue(record.Metadata, "evidence_kind") == "gait_lifecycle"
}
