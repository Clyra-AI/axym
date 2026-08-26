package actioncontract

// This file is deliberately separate from proof-record ingestion. Action
// Contract artifacts are producer-owned documents and must remain byte
// identical; they are evidence, not Axym execution instructions.

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Clyra-AI/axym/core/store"
	proofcanon "github.com/Clyra-AI/proof/canon"
	proofsign "github.com/Clyra-AI/proof/signing"
)

const (
	NativeArtifactDir = "action-contract"
	ProposalDir       = "proposals"
	ActivationDir     = "activations"
)

const (
	ActivationReceiptSchemaID      = "https://axym.dev/schemas/v1/action-contract-activation-verification-receipt.schema.json"
	ActivationReceiptSchemaVersion = "1"
)

// ActivationVerificationReceipt is an Axym-signed handoff proving that the
// Gait activation was cryptographically checked before native persistence.
// The sidecar is independently verified on every store load; an editable
// native envelope cannot manufacture this trust boundary.
type ActivationVerificationReceipt struct {
	SchemaID                string              `json:"schema_id"`
	SchemaVersion           string              `json:"schema_version"`
	ArtifactID              string              `json:"artifact_id"`
	RawSHA256               string              `json:"raw_sha256"`
	ProducerSignatureDigest string              `json:"producer_signature_digest"`
	ConformanceDigest       string              `json:"conformance_digest"`
	ConformanceReasonCodes  []string            `json:"conformance_reason_codes,omitempty"`
	SignerKeyID             string              `json:"signer_key_id"`
	IssuedAt                string              `json:"issued_at"`
	Signature               proofsign.Signature `json:"signature"`
}

// NativeArtifactEnvelope is the Axym-side index for one untouched producer
// artifact. The payload itself is stored beside this envelope, never rebuilt
// from Data. References are copied as strings to preserve producer joins.
type NativeArtifactEnvelope struct {
	ArtifactType           string           `json:"artifact_type"`
	ArtifactID             string           `json:"artifact_id"`
	ContractID             string           `json:"contract_id"`
	ContractFamilyID       string           `json:"contract_family_id"`
	Revision               int              `json:"revision"`
	Producer               ProducerMetadata `json:"producer"`
	SchemaID               string           `json:"schema_id"`
	SchemaVersion          string           `json:"schema_version"`
	ContractSchemaVersion  string           `json:"contract_schema_version"`
	RawSHA256              string           `json:"raw_sha256"`
	CanonicalContentDigest string           `json:"canonical_content_digest,omitempty"`
	References             []string         `json:"references,omitempty"`
	ReportOnly             bool             `json:"report_only"`
	ActivationMode         string           `json:"activation_mode,omitempty"`
	NonBinding             bool             `json:"non_binding"`
	ConformanceReasonCodes []string         `json:"conformance_reason_codes,omitempty"`
	ConformanceDigest      string           `json:"conformance_digest,omitempty"`
}

type StoredArtifact struct {
	RelativePath string
	Raw          []byte
	Envelope     NativeArtifactEnvelope
}

func PersistProposal(root string, proposal Proposal) (StoredArtifact, error) {
	envelope := NativeArtifactEnvelope{
		ArtifactType: "proposal", ArtifactID: proposal.ArtifactID,
		ContractID: proposal.ContractID, ContractFamilyID: proposal.ContractFamilyID,
		Revision: proposal.Revision, Producer: proposal.Producer, SchemaID: proposal.SchemaID,
		SchemaVersion: proposal.SchemaVersion, ContractSchemaVersion: proposal.Producer.ContractSchemaVersion,
		RawSHA256: proposal.RawSHA256, CanonicalContentDigest: proposal.CanonicalContentDigest,
		References: sortedRefs(append(append(append([]string{}, proposal.SourceScanRefs...), proposal.CompositionRefs...), proposal.CreationEvidence...)),
		ReportOnly: proposal.ReportOnly, NonBinding: true,
	}
	return persist(root, ProposalDir, proposal.ArtifactID, proposal.Raw, envelope)
}

func PersistActivation(root string, activation Activation, conformance *ConformanceResult, verification ...ActivationVerification) (StoredArtifact, error) {
	if activation.DevelopmentSigning {
		return StoredArtifact{}, fmt.Errorf("development-signed activation cannot be persisted as trusted evidence")
	}
	if conformance == nil {
		return StoredArtifact{}, fmt.Errorf("activation conformance result is required")
	}
	if strings.TrimSpace(root) == "" || filepath.Base(activation.ArtifactID) != activation.ArtifactID || strings.ContainsAny(activation.ArtifactID, `/\\`) || len(activation.Raw) == 0 {
		return StoredArtifact{}, fmt.Errorf("action contract artifact identity and bytes are required")
	}
	conformanceDigestValue, err := conformanceDigest(*conformance)
	if err != nil {
		return StoredArtifact{}, err
	}
	if len(verification) != 1 || !verification[0].Result.Valid || !verification[0].Result.SignatureVerified {
		return StoredArtifact{}, fmt.Errorf("activation verification context is required")
	}
	if err := VerifyActivationSignature(activation, verification[0].PublicKey); err != nil {
		return StoredArtifact{}, fmt.Errorf("activation producer signature is invalid: %w", err)
	}
	envelope := NativeArtifactEnvelope{
		ArtifactType: "activation", ArtifactID: activation.ArtifactID,
		ContractID: activation.ContractID, ContractFamilyID: activation.ContractFamilyID,
		Revision: activation.Revision, Producer: activation.Producer, SchemaID: activation.SchemaID,
		SchemaVersion: activation.SchemaVersion, ContractSchemaVersion: activation.Producer.ContractSchemaVersion,
		RawSHA256: activation.RawSHA256, References: sortedRefs(activation.AuthorityRefs),
		ReportOnly: activation.ReportOnly, ActivationMode: activation.ActivationMode,
		NonBinding: true, ConformanceDigest: conformanceDigestValue,
	}
	envelope.ConformanceReasonCodes = sortedRefs(conformance.ReasonCodes)
	payloadPath := filepath.Join(root, NativeArtifactDir, ActivationDir, activation.ArtifactID+".json")
	envelopePath := filepath.Join(root, NativeArtifactDir, ActivationDir, activation.ArtifactID+".envelope.json")
	receiptPath := filepath.Join(root, NativeArtifactDir, ActivationDir, activation.ArtifactID+".verification.json")
	snapshots := make([]fileSnapshot, 0, 3)
	for _, path := range []string{payloadPath, envelopePath, receiptPath} {
		snapshot, snapshotErr := snapshotFile(path)
		if snapshotErr != nil {
			return StoredArtifact{}, snapshotErr
		}
		snapshots = append(snapshots, snapshot)
	}
	rollback := func() {
		restoreFileSnapshots(snapshots)
	}
	item, err := persist(root, ActivationDir, activation.ArtifactID, activation.Raw, envelope)
	if err != nil {
		rollback()
		return StoredArtifact{}, err
	}
	axymStore, err := store.New(store.Config{RootDir: root})
	if err != nil {
		rollback()
		return StoredArtifact{}, err
	}
	key, err := axymStore.SigningKey()
	if err != nil {
		rollback()
		return StoredArtifact{}, err
	}
	receipt := ActivationVerificationReceipt{
		SchemaID: ActivationReceiptSchemaID, SchemaVersion: ActivationReceiptSchemaVersion,
		ArtifactID: activation.ArtifactID, RawSHA256: RawDigest(activation.Raw),
		ProducerSignatureDigest: activation.Signature.SignedDigest, ConformanceDigest: envelope.ConformanceDigest, ConformanceReasonCodes: append([]string(nil), envelope.ConformanceReasonCodes...), SignerKeyID: key.KeyID,
		IssuedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	digest, err := activationReceiptDigest(receipt)
	if err != nil {
		rollback()
		return StoredArtifact{}, err
	}
	receipt.Signature, err = proofsign.SignDigestHex(key.Private, strings.TrimPrefix(digest, "sha256:"))
	if err != nil {
		rollback()
		return StoredArtifact{}, fmt.Errorf("sign activation verification receipt: %w", err)
	}
	receiptRaw, err := json.MarshalIndent(receipt, "", "  ")
	if err != nil {
		rollback()
		return StoredArtifact{}, err
	}
	receiptRaw = append(receiptRaw, '\n')
	if err := store.WriteJSONAtomic(receiptPath, receiptRaw, true); err != nil {
		rollback()
		return StoredArtifact{}, err
	}
	return item, nil
}

type fileSnapshot struct {
	path   string
	raw    []byte
	mode   os.FileMode
	exists bool
}

func snapshotFile(path string) (fileSnapshot, error) {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return fileSnapshot{path: path}, nil
	}
	if err != nil {
		return fileSnapshot{}, err
	}
	if !info.Mode().IsRegular() {
		return fileSnapshot{}, fmt.Errorf("activation sidecar is not a regular file: %s", path)
	}
	// #nosec G304 -- paths are constructed from the validated activation ID
	// under the caller-controlled managed store root.
	raw, err := os.ReadFile(path)
	if err != nil {
		return fileSnapshot{}, err
	}
	return fileSnapshot{path: path, raw: raw, mode: info.Mode().Perm(), exists: true}, nil
}

func restoreFileSnapshots(snapshots []fileSnapshot) {
	for _, snapshot := range snapshots {
		if snapshot.exists {
			if err := os.WriteFile(snapshot.path, snapshot.raw, snapshot.mode); err == nil {
				_ = os.Chmod(snapshot.path, snapshot.mode)
			}
			continue
		}
		_ = os.Remove(snapshot.path)
	}
}

func persist(root, kind, id string, raw []byte, envelope NativeArtifactEnvelope) (StoredArtifact, error) {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(id) == "" || len(raw) == 0 {
		return StoredArtifact{}, fmt.Errorf("action contract artifact identity and bytes are required")
	}
	if filepath.Base(id) != id || strings.ContainsAny(id, `/\\`) {
		return StoredArtifact{}, fmt.Errorf("unsafe action contract artifact id")
	}
	if envelope.RawSHA256 == "" {
		envelope.RawSHA256 = RawDigest(raw)
	}
	rel := filepath.ToSlash(filepath.Join(NativeArtifactDir, kind, id+".json"))
	envRel := filepath.ToSlash(filepath.Join(NativeArtifactDir, kind, id+".envelope.json"))
	payloadPath := filepath.Join(root, filepath.FromSlash(rel))
	payloadExisted := false
	// #nosec G304 -- payloadPath is derived from a validated artifact ID under the managed store root.
	if existing, err := os.ReadFile(payloadPath); err == nil {
		payloadExisted = true
		if string(existing) != string(raw) {
			return StoredArtifact{}, fmt.Errorf("action contract artifact identity already contains different bytes: %s", id)
		}
	} else if !os.IsNotExist(err) {
		return StoredArtifact{}, err
	}
	if err := store.WriteJSONAtomic(payloadPath, raw, true); err != nil {
		return StoredArtifact{}, err
	}
	envRaw, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return StoredArtifact{}, err
	}
	envRaw = append(envRaw, '\n')
	if err := store.WriteJSONAtomic(filepath.Join(root, filepath.FromSlash(envRel)), envRaw, true); err != nil {
		if !payloadExisted {
			_ = os.Remove(payloadPath)
		}
		return StoredArtifact{}, err
	}
	return StoredArtifact{RelativePath: rel, Raw: append([]byte(nil), raw...), Envelope: envelope}, nil
}

// LoadStored returns all producer-native artifacts in deterministic order.
// It verifies the envelope-to-payload digest and refuses malformed metadata.
func LoadStored(root string) ([]StoredArtifact, error) {
	base := filepath.Join(root, NativeArtifactDir)
	var out []StoredArtifact
	if entries, err := os.ReadDir(base); err == nil {
		for _, entry := range entries {
			if !entry.IsDir() || (entry.Name() != ProposalDir && entry.Name() != ActivationDir && entry.Name() != "packets" && entry.Name() != "timelines" && entry.Name() != "graphs") {
				return nil, fmt.Errorf("unsupported action contract store entry: %s", entry.Name())
			}
		}
	} else if !os.IsNotExist(err) {
		return nil, err
	}
	for _, kind := range []string{ProposalDir, ActivationDir} {
		dir := filepath.Join(base, kind)
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, err
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() || !strings.HasSuffix(name, ".envelope.json") {
				if entry.IsDir() || (!strings.HasSuffix(name, ".json") && !strings.HasSuffix(name, ".envelope.json")) {
					return nil, fmt.Errorf("unexpected action contract store file: %s", name)
				}
				continue
			}
			if entry.Type()&os.ModeSymlink != 0 {
				return nil, fmt.Errorf("symlinked action contract envelope is not allowed: %s", name)
			}
			id := strings.TrimSuffix(name, ".envelope.json")
			payloadPath := filepath.Join(dir, id+".json")
			payloadInfo, statErr := os.Lstat(payloadPath)
			if statErr != nil || payloadInfo.Mode()&os.ModeSymlink != 0 || !payloadInfo.Mode().IsRegular() {
				return nil, fmt.Errorf("invalid action contract payload: %s", id)
			}
			// #nosec G304 -- name comes from a regular envelope entry under the managed artifact directory.
			envRaw, err := os.ReadFile(filepath.Join(dir, name))
			if err != nil {
				return nil, err
			}
			var env NativeArtifactEnvelope
			decoder := json.NewDecoder(bytes.NewReader(envRaw))
			decoder.DisallowUnknownFields()
			if err := rejectDuplicateJSONKeys(envRaw); err != nil {
				return nil, fmt.Errorf("decode action contract envelope %s: %w", name, err)
			}
			if err := decoder.Decode(&env); err != nil {
				return nil, fmt.Errorf("decode action contract envelope %s: %w", name, err)
			}
			var trailing any
			if err := decoder.Decode(&trailing); err != io.EOF {
				return nil, fmt.Errorf("decode action contract envelope %s: trailing input", name)
			}
			// #nosec G304 -- payloadPath is the validated sibling payload for this managed envelope.
			raw, err := os.ReadFile(payloadPath)
			if err != nil {
				return nil, err
			}
			if env.ArtifactID != id || env.RawSHA256 != RawDigest(raw) {
				return nil, fmt.Errorf("action contract envelope digest mismatch: %s", name)
			}
			if err := validateStoredEnvelope(kind, env, raw); err != nil {
				return nil, err
			}
			if kind == ActivationDir {
				activation, err := ParseActivation(raw)
				if err != nil {
					return nil, fmt.Errorf("parse stored activation %s: %w", id, err)
				}
				if err := verifyStoredActivationReceipt(root, activation, env); err != nil {
					return nil, err
				}
			}
			rel := filepath.ToSlash(filepath.Join(NativeArtifactDir, kind, id+".json"))
			out = append(out, StoredArtifact{RelativePath: rel, Raw: raw, Envelope: env})
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || strings.HasSuffix(entry.Name(), ".envelope.json") || strings.HasSuffix(entry.Name(), ".verification.json") {
				continue
			}
			envelopeName := strings.TrimSuffix(entry.Name(), ".json") + ".envelope.json"
			if _, err := os.Stat(filepath.Join(dir, envelopeName)); os.IsNotExist(err) {
				return nil, fmt.Errorf("orphan action contract payload: %s", entry.Name())
			}
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RelativePath < out[j].RelativePath })
	return out, nil
}

func verifyStoredActivationReceipt(root string, activation Activation, envelope NativeArtifactEnvelope) error {
	receiptPath := filepath.Join(root, NativeArtifactDir, ActivationDir, activation.ArtifactID+".verification.json")
	if info, statErr := os.Lstat(receiptPath); statErr != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return fmt.Errorf("stored activation verification receipt is missing: %s", activation.ArtifactID)
	}
	// #nosec G304 -- receiptPath is derived from the already validated artifact ID.
	raw, err := os.ReadFile(receiptPath)
	if err != nil {
		return fmt.Errorf("stored activation verification receipt is missing: %s", activation.ArtifactID)
	}
	if err := rejectDuplicateJSONKeys(raw); err != nil {
		return fmt.Errorf("decode activation verification receipt %s: %w", activation.ArtifactID, err)
	}
	receipt, err := decodeActivationVerificationReceipt(raw)
	if err != nil {
		return fmt.Errorf("decode activation verification receipt %s: %w", activation.ArtifactID, err)
	}
	if receipt.SchemaID != ActivationReceiptSchemaID || receipt.SchemaVersion != ActivationReceiptSchemaVersion || receipt.ArtifactID != activation.ArtifactID || receipt.RawSHA256 != RawDigest(activation.Raw) || receipt.ProducerSignatureDigest != activation.Signature.SignedDigest || receipt.ConformanceDigest == "" || receipt.ConformanceDigest != envelope.ConformanceDigest || !sameStrings(receipt.ConformanceReasonCodes, envelope.ConformanceReasonCodes) || receipt.SignerKeyID == "" || receipt.IssuedAt == "" {
		return fmt.Errorf("activation verification receipt mismatch: %s", activation.ArtifactID)
	}
	if _, err := time.Parse(time.RFC3339Nano, receipt.IssuedAt); err != nil {
		return fmt.Errorf("activation verification receipt timestamp is invalid: %s", activation.ArtifactID)
	}
	publicKey, keyID, err := activationReceiptPublicKey(root)
	if err != nil || receipt.SignerKeyID != keyID {
		return fmt.Errorf("activation verification receipt signer mismatch: %s", activation.ArtifactID)
	}
	digest, err := activationReceiptDigest(receipt)
	if err != nil || receipt.Signature.SignedDigest != strings.TrimPrefix(digest, "sha256:") {
		return fmt.Errorf("activation verification receipt digest mismatch: %s", activation.ArtifactID)
	}
	valid, err := proofsign.VerifyDigestHex(publicKey, receipt.Signature)
	if err != nil || !valid || receipt.Signature.KeyID != keyID {
		return fmt.Errorf("activation verification receipt signature invalid: %s", activation.ArtifactID)
	}
	return nil
}

func decodeActivationVerificationReceipt(raw []byte) (ActivationVerificationReceipt, error) {
	var receipt ActivationVerificationReceipt
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&receipt); err != nil {
		return ActivationVerificationReceipt{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return ActivationVerificationReceipt{}, fmt.Errorf("trailing JSON")
	}
	return receipt, nil
}

func activationReceiptPublicKey(root string) (ed25519.PublicKey, string, error) {
	managed, err := store.OpenReadOnly(store.Config{RootDir: root})
	if err == nil {
		key, keyErr := managed.SigningKey()
		if keyErr == nil {
			return key.Public, key.KeyID, nil
		}
	}
	// Bundles intentionally carry only the public record key, not the managed
	// store's private signing-key.json. Use that authenticated bundle key when
	// verifying the copied activation receipt.
	// #nosec G304 -- the explicit bundle/store root is supplied by the caller;
	// the fixed filename is not derived from untrusted artifact data.
	raw, readErr := os.ReadFile(filepath.Join(root, "record-signing-key.json"))
	if readErr != nil {
		if err != nil {
			return nil, "", fmt.Errorf("load activation receipt signing key: %w", err)
		}
		return nil, "", fmt.Errorf("load activation receipt signing key: %w", readErr)
	}
	var payload struct {
		KeyID  string `json:"key_id"`
		Public string `json:"public"`
	}
	if decodeErr := json.Unmarshal(raw, &payload); decodeErr != nil {
		return nil, "", decodeErr
	}
	public, decodeErr := base64.StdEncoding.DecodeString(payload.Public)
	if decodeErr != nil || len(public) != ed25519.PublicKeySize || payload.KeyID == "" {
		return nil, "", fmt.Errorf("invalid activation receipt public key")
	}
	return ed25519.PublicKey(public), payload.KeyID, nil
}

func activationReceiptDigest(receipt ActivationVerificationReceipt) (string, error) {
	copy := receipt
	copy.Signature = proofsign.Signature{}
	raw, err := json.Marshal(copy)
	if err != nil {
		return "", err
	}
	return proofcanon.DigestJCS(raw)
}

func conformanceDigest(result ConformanceResult) (string, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return "", err
	}
	return proofcanon.DigestJCS(raw)
}

func validateStoredEnvelope(kind string, env NativeArtifactEnvelope, raw []byte) error {
	if env.RawSHA256 != RawDigest(raw) || env.ArtifactID == "" || env.ContractID == "" || env.ContractFamilyID == "" || env.Revision < 1 || env.Producer.Name == "" || env.SchemaID == "" || env.SchemaVersion == "" || env.ContractSchemaVersion == "" || !env.NonBinding {
		return fmt.Errorf("incomplete action contract envelope metadata")
	}
	switch kind {
	case ProposalDir:
		p, err := ParseProposal(raw)
		if err != nil {
			return fmt.Errorf("stored proposal envelope payload mismatch: %w", err)
		}
		if env.ArtifactType != "proposal" || env.Producer != p.Producer || env.SchemaID != ProposalSchemaID || env.SchemaVersion != ProposalSchemaVersion || env.ContractSchemaVersion != ProposalContractVersion || env.ArtifactID != p.ArtifactID || env.ContractID != p.ContractID || env.ContractFamilyID != p.ContractFamilyID || env.Revision != p.Revision || env.CanonicalContentDigest != p.CanonicalContentDigest || !env.ReportOnly || !p.ReportOnly || !sameStrings(env.References, sortedRefs(append(append(append([]string{}, p.SourceScanRefs...), p.CompositionRefs...), p.CreationEvidence...))) {
			return fmt.Errorf("stored proposal envelope metadata mismatch")
		}
	case ActivationDir:
		a, err := ParseActivation(raw)
		if err != nil {
			return fmt.Errorf("stored activation envelope payload mismatch: %w", err)
		}
		if env.ArtifactType != "activation" || env.Producer != a.Producer || env.SchemaID != ActivationSchemaID || env.SchemaVersion != ActivationSchemaVersion || env.ContractSchemaVersion != ActivationContractVersion || env.ArtifactID != a.ArtifactID || env.ContractID != a.ContractID || env.ContractFamilyID != a.ContractFamilyID || env.Revision != a.Revision || env.ReportOnly != a.ReportOnly || env.ActivationMode != a.ActivationMode || !sameStrings(env.References, sortedRefs(a.AuthorityRefs)) {
			return fmt.Errorf("stored activation envelope metadata mismatch")
		}
	default:
		return fmt.Errorf("unsupported action contract artifact directory: %s", kind)
	}
	return nil
}

func sameStrings(left, right []string) bool {
	left = sortedRefs(left)
	right = sortedRefs(right)
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func sortedRefs(in []string) []string {
	seen := map[string]struct{}{}
	for _, value := range in {
		if value = strings.TrimSpace(value); value != "" {
			seen[value] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for value := range seen {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
