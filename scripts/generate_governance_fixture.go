//go:build ignore

package main

// Generates/checks the quarantined Axym producer fixture. Its inputs are the
// exact checked-in Wrkr/Gait artifacts; no private key is written.
import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/Clyra-AI/axym/core/governance"
	"github.com/Clyra-AI/axym/core/ingest/actioncontract"
)

const producerVersion = "v0.2.0"
const proposalPath = "testdata/action-contract-interop/v1/expected/compensation/pac-4b7f1402784256ce.json"
const activationPath = "testdata/action-contract-interop/v1/expected/compensation/activated-action-contract.json"
const lifecyclePath = "testdata/gait-action-contract-evidence/v1/compensation-required-started-completed/lifecycle.json"
const gatePath = "testdata/governance/v1/gait-gate/approval-exact.json"

type lifecycleRecord struct {
	RecordID   string `json:"record_id"`
	Kind       string `json:"kind"`
	OccurredAt string `json:"occurred_at"`
	Signature  struct {
		SignedDigest string `json:"signed_digest"`
	} `json:"signature"`
}

func main() {
	out := flag.String("out", "testdata/governance/v1/producer-fixture", "fixture output directory")
	update := flag.Bool("update", false, "write the fixture")
	check := flag.Bool("check", false, "check exact bytes and orphan files")
	flag.Parse()
	var err error
	if *check {
		err = checkFixture(*out)
	} else if *update {
		err = generate(*out)
	} else {
		err = fmt.Errorf("specify --update or --check")
	}
	if err != nil {
		panic(err)
	}
}

func generate(root string) error {
	proposalRaw, err := os.ReadFile(proposalPath)
	if err != nil {
		return err
	}
	activationRaw, err := os.ReadFile(activationPath)
	if err != nil {
		return err
	}
	lifecycleRaw, err := os.ReadFile(lifecyclePath)
	if err != nil {
		return err
	}
	gateRaw, err := os.ReadFile(gatePath)
	if err != nil {
		return err
	}
	proposal, err := actioncontract.ParseProposal(proposalRaw)
	if err != nil {
		return err
	}
	if !actioncontract.AcceptableSemanticProposal(actioncontract.ValidateProposal(proposal, actioncontract.ValidationOptions{})) {
		return fmt.Errorf("Wrkr proposal validation failed")
	}
	var lifecycle struct {
		Records []lifecycleRecord `json:"records"`
	}
	if err = json.Unmarshal(lifecycleRaw, &lifecycle); err != nil || len(lifecycle.Records) == 0 {
		return fmt.Errorf("Gait lifecycle fixture invalid")
	}
	register, _, err := governance.RegisterAndPackets([]actioncontract.Proposal{proposal}, nil)
	if err != nil {
		return err
	}
	contract := register.Contracts[0]
	contractRef := contract.CausalRef
	axis := map[string]string{"proposal_ingested": "readiness", "activation_requested": "confirmation", "precondition_evaluated": "preconditions", "decision_ready": "approval", "activated": "enforcement", "execution_started": "execution", "execution_succeeded": "outcome", "effect_recorded": "effect", "effect_validated": "effect", "containment_requested": "containment", "containment_completed": "containment", "compensation_required": "compensation", "compensation_started": "compensation", "compensation_completed": "compensation"}
	evidence := make([]governance.Evidence, 0, len(lifecycle.Records)+7)
	for _, record := range lifecycle.Records {
		kind := axis[record.Kind]
		if kind == "" {
			continue
		}
		if len(record.Signature.SignedDigest) != 64 {
			return fmt.Errorf("unsigned lifecycle record %s", record.RecordID)
		}
		ref := governance.Ref{ID: record.RecordID, Kind: kind, Digest: "sha256:" + record.Signature.SignedDigest, Source: "gait", SourceProduct: "gait", SchemaID: "https://gait.dev/schemas/v1/runtime-lifecycle-record.schema.json", SchemaVersion: "1"}
		evidence = append(evidence, governance.Evidence{Kind: kind, Ref: ref, OccurredAt: record.OccurredAt, ContractRef: contractRef, Provenance: ref})
	}
	for i, item := range []struct{ kind, id, digest, path, schema string }{{"authority", "gact-4aad73ff9f3c7e5a", actioncontract.RawDigest(activationRaw), activationPath, actioncontract.ActivationSchemaID}, {"credential", "gact-4aad73ff9f3c7e5a", actioncontract.RawDigest(activationRaw), activationPath, actioncontract.ActivationSchemaID}, {"delegation", "d2b27b6d9992e51ac9660de5", actioncontract.RawDigest(gateRaw), gatePath, "https://gait.dev/schemas/v1/gate-approval.schema.json"}, {"resource_lifecycle", "gait-lifecycle", actioncontract.RawDigest(lifecycleRaw), lifecyclePath, "https://gait.dev/schemas/v1/runtime-lifecycle-record.schema.json"}, {"proof", "gait-proof", actioncontract.RawDigest(lifecycleRaw), lifecyclePath, "https://gait.dev/schemas/v1/runtime-lifecycle-record.schema.json"}, {"freshness", "gait-freshness", actioncontract.RawDigest(lifecycleRaw), lifecyclePath, "https://gait.dev/schemas/v1/runtime-lifecycle-record.schema.json"}, {"correlation", "gait-correlation", actioncontract.RawDigest(lifecycleRaw), lifecyclePath, "https://gait.dev/schemas/v1/runtime-lifecycle-record.schema.json"}} {
		ref := governance.Ref{ID: item.id, Kind: item.kind, Digest: item.digest, Source: item.path, SourceProduct: "gait", SchemaID: item.schema, SchemaVersion: "1"}
		evidence = append(evidence, governance.Evidence{Kind: item.kind, Ref: ref, OccurredAt: fmt.Sprintf("2026-07-19T01:00:%02dZ", i), ContractRef: contractRef, Provenance: ref})
	}
	packet, err := governance.BuildPacket(contract, evidence)
	if err != nil {
		return err
	}
	seedSum := sha256.Sum256([]byte("axym-governed-producer-fixture-v1"))
	seed := seedSum[:]
	packet, err = governance.SignPacket(packet, ed25519.NewKeyFromSeed(seed))
	if err != nil {
		return err
	}
	registerRaw, _ := json.MarshalIndent(register, "", "  ")
	packetRaw, _ := json.MarshalIndent(packet, "", "  ")
	if err = write(root, "action-contract-register.json", append(registerRaw, '\n')); err != nil {
		return err
	}
	if err = write(root, "action-contract-evidence-packet.json", append(packetRaw, '\n')); err != nil {
		return err
	}
	priv := ed25519.NewKeyFromSeed(seed)
	public := priv.Public().(ed25519.PublicKey)
	if err = write(root, "fixture-signing-key.public.b64", []byte(base64.StdEncoding.EncodeToString(priv.Public().(ed25519.PublicKey))+"\n")); err != nil {
		return err
	}
	for name, src := range map[string]string{"action-contract-register.schema.json": "schemas/v1/governance/action-contract-register.schema.json", "action-contract-evidence-packet.schema.json": "schemas/v1/governance/action-contract-evidence-packet.schema.json"} {
		raw, e := os.ReadFile(src)
		if e != nil {
			return e
		}
		if e = write(filepath.Join(root, "schemas"), name, raw); e != nil {
			return e
		}
	}
	files := []string{"action-contract-evidence-packet.json", "action-contract-register.json", "fixture-signing-key.public.b64", "schemas/action-contract-evidence-packet.schema.json", "schemas/action-contract-register.schema.json"}
	sort.Strings(files)
	entries := make([]map[string]string, 0, len(files))
	for _, name := range files {
		raw, e := os.ReadFile(filepath.Join(root, name))
		if e != nil {
			return e
		}
		sum := sha256.Sum256(raw)
		entries = append(entries, map[string]string{"path": name, "sha256": "sha256:" + hex.EncodeToString(sum[:])})
	}
	manifest := map[string]any{"fixture_version": "1", "fixture_only": true, "quarantine": true, "authoritative": false, "non_authoritative": true, "producer": map[string]string{"name": "axym", "version": producerVersion, "kind": "governed_action_contract_projection"}, "signing": map[string]any{"fixture_test_only": true, "non_authoritative": true, "derivation": "sha256(axym-governed-producer-fixture-v1)", "public_key_sha256": actioncontract.RawDigest(public)}, "sources": map[string]any{"wrkr_proposal": sourceEntry(proposalPath, proposalRaw), "gait_activation": sourceEntry(activationPath, activationRaw), "gait_lifecycle": sourceEntry(lifecyclePath, lifecycleRaw), "gait_gate": sourceEntry(gatePath, gateRaw), "wrkr_manifest": manifestEntry("testdata/action-contract-interop/v1/expected/fixture-manifest.json"), "gait_manifest": manifestEntry("testdata/gait-action-contract-evidence/v1/fixture-manifest.json")}, "artifacts": map[string]string{"register": "sha256:" + sha(registerRaw), "packet": "sha256:" + sha(packetRaw)}, "files": entries}
	manifestRaw, _ := json.MarshalIndent(manifest, "", "  ")
	return write(root, "manifest.json", append(manifestRaw, '\n'))
}
func sourceEntry(path string, raw []byte) map[string]string {
	return map[string]string{"path": path, "sha256": actioncontract.RawDigest(raw)}
}
func manifestEntry(path string) map[string]string {
	raw, _ := os.ReadFile(path)
	return sourceEntry(path, raw)
}
func sha(raw []byte) string { sum := sha256.Sum256(raw); return hex.EncodeToString(sum[:]) }
func write(root, name string, data []byte) error {
	if err := os.MkdirAll(filepath.Join(root, filepath.Dir(name)), 0700); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, name), data, 0600)
}
func checkFixture(root string) error {
	tmp, err := os.MkdirTemp("", "axym-governance-fixture-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if err = generate(tmp); err != nil {
		return err
	}
	return compareTree(tmp, root, "")
}
func compareTree(expected, actual, rel string) error {
	entries, err := os.ReadDir(filepath.Join(expected, rel))
	if err != nil {
		return err
	}
	actualEntries, err := os.ReadDir(filepath.Join(actual, rel))
	if err != nil {
		return err
	}
	actualSet := map[string]bool{}
	for _, e := range actualEntries {
		actualSet[e.Name()] = true
	}
	for _, e := range entries {
		name := filepath.ToSlash(filepath.Join(rel, e.Name()))
		if e.IsDir() {
			if err = compareTree(expected, actual, name); err != nil {
				return err
			}
			delete(actualSet, e.Name())
			continue
		}
		left, _ := os.ReadFile(filepath.Join(expected, name))
		right, e2 := os.ReadFile(filepath.Join(actual, name))
		if e2 != nil || string(left) != string(right) {
			return fmt.Errorf("fixture drift: %s", name)
		}
		delete(actualSet, e.Name())
	}
	if rel == "" {
		delete(actualSet, "README.md")
	}
	if len(actualSet) > 0 {
		return fmt.Errorf("orphan fixture file: %s", filepath.Join(rel, sortedKeys(actualSet)[0]))
	}
	return nil
}
func sortedKeys(in map[string]bool) []string {
	out := make([]string, 0, len(in))
	for k := range in {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
