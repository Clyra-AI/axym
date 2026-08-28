//go:build ignore

package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Clyra-AI/axym/core/governance"
	"github.com/Clyra-AI/axym/core/ingest/actioncontract"
	proofcanon "github.com/Clyra-AI/proof/canon"
	proofsign "github.com/Clyra-AI/proof/signing"
)

const (
	registerName       = "axym-authoritative-action-contract-register.json"
	packetName         = "axym-authoritative-action-contract-evidence-packet.json"
	manifestName       = governance.AuthoritativeReleaseManifestName
	keyName            = "axym-authoritative-action-contract-public-key.b64"
	registerSchemaName = "axym-authoritative-action-contract-register.schema.json"
	packetSchemaName   = "axym-authoritative-action-contract-evidence-packet.schema.json"
	bundleName         = "axym-authoritative-action-contract-bundle.tar.gz"
)

func main() {
	out := flag.String("out", "dist", "release output directory")
	proposalInput := flag.String("proposal-input", "", "explicit released Wrkr proposal artifact")
	activationInput := flag.String("activation-input", "", "explicit released Gait activation artifact")
	lifecycleInput := flag.String("lifecycle-input", "", "explicit released Gait authoritative lifecycle artifact")
	gateInput := flag.String("gate-input", "", "explicit released Gait gate artifact")
	gaitKeyInput := flag.String("gait-public-key", "", "explicit Gait lifecycle public key")
	gateKeyInput := flag.String("gate-public-key", "", "explicit Gait gate public key")
	tag := flag.String("tag", os.Getenv("GITHUB_REF_NAME"), "release tag")
	commit := flag.String("commit", os.Getenv("GITHUB_SHA"), "peeled release commit")
	workflowRef := flag.String("workflow-ref", os.Getenv("GITHUB_WORKFLOW_REF"), "workflow identity")
	runID := flag.String("run-id", os.Getenv("GITHUB_RUN_ID"), "workflow run identity")
	repository := flag.String("repository", os.Getenv("GITHUB_REPOSITORY"), "repository identity")
	flag.Parse()
	if strings.TrimSpace(*tag) == "" || strings.TrimSpace(*commit) == "" || strings.TrimSpace(*repository) == "" || *proposalInput == "" || *activationInput == "" || *lifecycleInput == "" || *gaitKeyInput == "" {
		fmt.Fprintln(os.Stderr, "authoritative release generation failed: tag, commit, repository, proposal-input, activation-input, lifecycle-input, and gait-public-key are required")
		os.Exit(1)
	}
	if err := generate(*out, *tag, *commit, *workflowRef, *runID, *repository, *proposalInput, *activationInput, *lifecycleInput, *gateInput, *gaitKeyInput, *gateKeyInput); err != nil {
		fmt.Fprintln(os.Stderr, "authoritative release generation failed:", err)
		os.Exit(1)
	}
}

func generate(out, tag, commit, workflowRef, runID, repository, proposalInput, activationInput, lifecycleInput, gateInput, gaitKeyInput, gateKeyInput string) error {
	proposalRaw, err := os.ReadFile(proposalInput)
	if err != nil {
		return err
	}
	activationRaw, err := os.ReadFile(activationInput)
	if err != nil {
		return err
	}
	lifecycleRaw, err := os.ReadFile(lifecycleInput)
	if err != nil {
		return err
	}
	var gateRaw []byte
	if gateInput != "" {
		gateRaw, err = os.ReadFile(gateInput)
		if err != nil {
			return err
		}
	}
	proposal, err := actioncontract.ParseProposal(proposalRaw)
	if err != nil {
		return err
	}
	if !actioncontract.AcceptableSemanticProposal(actioncontract.ValidateProposal(proposal, actioncontract.ValidationOptions{})) {
		return fmt.Errorf("Wrkr proposal validation failed")
	}
	activation, err := actioncontract.ParseActivation(activationRaw)
	if err != nil {
		return err
	}
	gaitPublic, err := readPublicKey(gaitKeyInput)
	if err != nil {
		return fmt.Errorf("read Gait public key: %w", err)
	}
	gatePublic := gaitPublic
	if gateKeyInput != "" {
		gatePublic, err = readPublicKey(gateKeyInput)
		if err != nil {
			return fmt.Errorf("read gate public key: %w", err)
		}
	}
	if err := actioncontract.VerifyActivationSignature(activation, gaitPublic); err != nil {
		return fmt.Errorf("activation signature verification failed: %w", err)
	}
	register, _, err := governance.RegisterAndPackets([]actioncontract.Proposal{proposal}, nil)
	if err != nil {
		return err
	}
	contract := register.Contracts[0]
	contractRef := contract.CausalRef
	var source struct {
		Records []json.RawMessage `json:"records"`
	}
	if err := json.Unmarshal(lifecycleRaw, &source); err != nil || len(source.Records) == 0 {
		return fmt.Errorf("Gait lifecycle input invalid")
	}
	axis := map[string]string{"proposal_ingested": "readiness", "activation_requested": "confirmation", "precondition_evaluated": "preconditions", "decision_ready": "approval", "activated": "enforcement", "execution_started": "execution", "execution_succeeded": "outcome", "effect_recorded": "effect", "effect_validated": "effect", "containment_requested": "containment", "containment_completed": "containment", "compensation_required": "compensation", "compensation_started": "compensation", "compensation_completed": "compensation"}
	evidence := make([]governance.Evidence, 0, len(source.Records)+7)
	firstAt := ""
	for _, rawItem := range source.Records {
		var item struct {
			RecordID   string `json:"record_id"`
			Kind       string `json:"kind"`
			OccurredAt string `json:"occurred_at"`
			Signature  struct {
				SignedDigest string `json:"signed_digest"`
			} `json:"signature"`
			ContractRef struct {
				ID     string `json:"id"`
				Digest string `json:"digest"`
			} `json:"contract_ref"`
			ContractFamilyID string `json:"contract_family_id"`
			Revision         int    `json:"revision"`
			ActivationRef    struct {
				ID     string `json:"id"`
				Digest string `json:"digest"`
			} `json:"activation_ref"`
		}
		if err := json.Unmarshal(rawItem, &item); err != nil {
			return fmt.Errorf("invalid lifecycle record")
		}
		if firstAt == "" {
			firstAt = item.OccurredAt
		}
		if kind := axis[item.Kind]; kind != "" {
			if item.RecordID == "" || len(item.Signature.SignedDigest) != 64 || item.ContractRef.ID != proposal.ContractID || item.ContractRef.Digest != proposal.CanonicalContentDigest || item.ContractFamilyID != proposal.ContractFamilyID || item.Revision != proposal.Revision || (item.ActivationRef.ID != "" && item.ActivationRef.ID != activation.ArtifactID) {
				return fmt.Errorf("unsigned lifecycle record %s", item.RecordID)
			}
			if err := verifySignedLifecycleRecord(rawItem, gaitPublic); err != nil {
				return fmt.Errorf("lifecycle record %s verification failed: %w", item.RecordID, err)
			}
			ref := governance.Ref{ID: item.RecordID, Kind: kind, Digest: "sha256:" + item.Signature.SignedDigest, Source: "gait", SourceProduct: "gait", SchemaID: "https://gait.dev/schemas/v1/runtime-lifecycle-record.schema.json", SchemaVersion: "1"}
			attrs := lifecycleAttrs(item.Kind)
			evidence = append(evidence, governance.Evidence{Kind: kind, Ref: ref, OccurredAt: item.OccurredAt, ContractRef: contractRef, Attributes: attrs, Provenance: ref})
		}
	}
	var gate map[string]any
	if len(gateRaw) > 0 {
		if err := json.Unmarshal(gateRaw, &gate); err != nil {
			return err
		}
		lifecycleAt, err := time.Parse(time.RFC3339Nano, firstAt)
		if err != nil {
			return fmt.Errorf("lifecycle timestamp invalid: %w", err)
		}
		if err := governance.ValidateGateContractBinding(gate, proposal.ContractID, proposal.ContractFamilyID, proposal.Revision, proposal.CanonicalContentDigest, lifecycleAt); err != nil {
			return err
		}
		if _, err := governance.VerifyGateArtifact(gateRaw, gatePublic, lifecycleAt, nil); err != nil {
			return fmt.Errorf("gate signature verification failed: %w", err)
		}
	}
	extra := []struct{ kind, id, digest, path, schema string }{{"authority", activation.ArtifactID, actioncontract.RawDigest(activationRaw), filepath.Base(activationInput), actioncontract.ActivationSchemaID}, {"credential", activation.ArtifactID, actioncontract.RawDigest(activationRaw), filepath.Base(activationInput), actioncontract.ActivationSchemaID}, {"resource_lifecycle", "gait-lifecycle", actioncontract.RawDigest(lifecycleRaw), filepath.Base(lifecycleInput), "https://gait.dev/schemas/v1/runtime-lifecycle-record.schema.json"}, {"proof", "gait-proof", actioncontract.RawDigest(lifecycleRaw), filepath.Base(lifecycleInput), "https://gait.dev/schemas/v1/runtime-lifecycle-record.schema.json"}, {"freshness", "gait-freshness", actioncontract.RawDigest(lifecycleRaw), filepath.Base(lifecycleInput), "https://gait.dev/schemas/v1/runtime-lifecycle-record.schema.json"}, {"correlation", "gait-correlation", actioncontract.RawDigest(lifecycleRaw), filepath.Base(lifecycleInput), "https://gait.dev/schemas/v1/runtime-lifecycle-record.schema.json"}}
	if len(gateRaw) > 0 {
		extra = append(extra, struct{ kind, id, digest, path, schema string }{"delegation", stringField(gate, "token_id"), actioncontract.RawDigest(gateRaw), filepath.Base(gateInput), "https://gait.dev/schemas/v1/gate-approval.schema.json"})
	}
	for _, item := range extra {
		if item.id == "" {
			return fmt.Errorf("authoritative input missing relationship identity")
		}
		ref := governance.Ref{ID: item.id, Kind: item.kind, Digest: item.digest, Source: item.path, SourceProduct: "gait", SchemaID: item.schema, SchemaVersion: "1"}
		evidence = append(evidence, governance.Evidence{Kind: item.kind, Ref: ref, OccurredAt: firstAt, ContractRef: contractRef, Provenance: ref})
	}
	packet, err := governance.BuildPacket(contract, evidence)
	if err != nil {
		return err
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	register, err = governance.SignRegister(register, priv)
	if err != nil {
		return err
	}
	packet.Signature = nil
	packet.Digest = ""
	packet, err = governance.SignPacket(packet, priv)
	if err != nil {
		return err
	}
	registerBytes, err := marshal(register)
	if err != nil {
		return err
	}
	packetBytes, err := marshal(packet)
	if err != nil {
		return err
	}
	keyBytes := []byte(base64.StdEncoding.EncodeToString(pub) + "\n")
	registerSchema, err := os.ReadFile("schemas/v1/governance/action-contract-register.schema.json")
	if err != nil {
		return err
	}
	packetSchema, err := os.ReadFile("schemas/v1/governance/action-contract-evidence-packet.schema.json")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(out, 0o755); err != nil {
		return err
	}
	files := map[string][]byte{registerName: registerBytes, packetName: packetBytes, keyName: keyBytes, registerSchemaName: registerSchema, packetSchemaName: packetSchema}
	bundleBytes, err := bundle(files)
	if err != nil {
		return err
	}
	files[bundleName] = bundleBytes
	for name, raw := range files {
		if err := os.WriteFile(filepath.Join(out, name), raw, 0o644); err != nil {
			return err
		}
	}
	manifest := map[string]any{
		"manifest_version": "v1", "release_tag": tag, "peeled_commit": commit,
		"authoritative": true, "fixture_only": false, "non_authoritative": false, "quarantine": false,
		"producer":  map[string]string{"name": "axym", "version": tag, "kind": "release_owner_authoritative_governance"},
		"generator": map[string]string{"name": "generate_authoritative_governance_release.go", "version": "v1"},
		"workflow":  map[string]string{"name": "release", "ref": workflowRef, "run_id": runID, "repository": repository},
		"signing":   map[string]string{"algorithm": "ed25519", "key_origin": "release_time_generated", "public_key_path": keyName, "public_key_sha256": actioncontract.RawDigest(keyBytes)},
		"artifacts": map[string]string{"register": actioncontract.RawDigest(registerBytes), "packet": actioncontract.RawDigest(packetBytes), "bundle": actioncontract.RawDigest(bundleBytes)},
		"schemas":   []map[string]string{{"path": registerSchemaName, "sha256": actioncontract.RawDigest(registerSchema), "schema_id": governance.RegisterSchemaID, "schema_version": governance.SchemaVersion}, {"path": packetSchemaName, "sha256": actioncontract.RawDigest(packetSchema), "schema_id": governance.PacketSchemaID, "schema_version": governance.SchemaVersion}},
		"files":     releaseFiles(files),
	}
	manifestBytes, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}
	manifestBytes = append(manifestBytes, '\n')
	return os.WriteFile(filepath.Join(out, manifestName), manifestBytes, 0o644)
}

func marshal(value any) ([]byte, error) {
	raw, err := json.MarshalIndent(value, "", "  ")
	return append(raw, '\n'), err
}

func readPublicKey(path string) (ed25519.PublicKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil || len(key) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid public key")
	}
	return ed25519.PublicKey(key), nil
}

func verifySignedLifecycleRecord(raw []byte, public ed25519.PublicKey) error {
	var value map[string]json.RawMessage
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}
	signatureRaw, ok := value["signature"]
	if !ok {
		return fmt.Errorf("signature missing")
	}
	var signature proofsign.Signature
	if err := json.Unmarshal(signatureRaw, &signature); err != nil {
		return err
	}
	if signature.KeyID != proofsign.KeyID(public) {
		return fmt.Errorf("signature key mismatch")
	}
	if len(signature.SignedDigest) != 64 {
		return fmt.Errorf("signed digest is invalid")
	}
	value["record_id"] = json.RawMessage(`""`)
	value["signature"] = json.RawMessage(`{"alg":"","key_id":"","sig":""}`)
	canonical, err := json.Marshal(value)
	if err != nil {
		return err
	}
	digest, err := proofcanon.DigestJCS(canonical)
	if err != nil || digest != signature.SignedDigest {
		return fmt.Errorf("signed digest mismatch")
	}
	valid, err := proofsign.VerifyDigestHex(public, signature)
	if err != nil || !valid {
		return fmt.Errorf("signature invalid")
	}
	return nil
}

func lifecycleAttrs(kind string) map[string]string {
	state := map[string]string{"execution_started": "started", "execution_blocked": "blocked", "execution_succeeded": "succeeded", "execution_failed": "failed", "effect_recorded": "recorded", "effect_validated": "validated", "containment_requested": "requested", "containment_completed": "completed", "containment_partial": "partial", "containment_unresolved": "gap", "compensation_required": "required", "compensation_started": "started", "compensation_completed": "completed", "compensation_failed": "failed"}[kind]
	attrs := map[string]string{}
	if state != "" {
		attrs["state"] = state
	}
	if kind == "execution_blocked" || kind == "execution_succeeded" || kind == "execution_failed" || kind == "effect_validated" || kind == "containment_completed" || kind == "containment_partial" || kind == "containment_unresolved" || kind == "compensation_completed" || kind == "compensation_failed" {
		attrs["terminal"] = "true"
	}
	return attrs
}

func stringField(value map[string]any, key string) string {
	result, _ := value[key].(string)
	return result
}

func releaseFiles(files map[string][]byte) []map[string]string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]map[string]string, 0, len(names))
	for _, name := range names {
		out = append(out, map[string]string{"path": name, "sha256": actioncontract.RawDigest(files[name])})
	}
	return out
}

func bundle(files map[string][]byte) ([]byte, error) {
	var buf strings.Builder
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		header := &tar.Header{Name: "authoritative-governance/" + name, Mode: 0o644, Size: int64(len(files[name])), ModTime: time.Unix(0, 0), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(header); err != nil {
			return nil, err
		}
		if _, err := tw.Write(files[name]); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}
