package governance

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	proofcanon "github.com/Clyra-AI/proof/canon"
	proofsign "github.com/Clyra-AI/proof/signing"
)

const ReasonControlSignature = "CONTROL_SIGNATURE_INVALID"

var controlDigestPattern = regexp.MustCompile(`^sha256:[a-f0-9]{64}$`)

type ControlLifecycleResult struct {
	Scenario         string   `json:"scenario"`
	Statuses         []string `json:"statuses"`
	AuthorityLineage bool     `json:"authority_lineage"`
	Quarantine       bool     `json:"quarantine"`
	Authoritative    bool     `json:"authoritative"`
}

func VerifyControlLifecycle(raw []byte, pub ed25519.PublicKey) (ControlLifecycleResult, error) {
	var doc struct {
		Records []map[string]any `json:"records"`
	}
	if json.Unmarshal(raw, &doc) != nil || len(doc.Records) == 0 {
		return ControlLifecycleResult{}, fmt.Errorf("%s", ReasonMalformed)
	}
	seen := map[string]bool{}
	last := time.Time{}
	controls := map[string]map[string]any{}
	out := ControlLifecycleResult{Quarantine: true}
	for _, r := range doc.Records {
		id, _ := r["record_id"].(string)
		if id == "" || seen[id] {
			return out, fmt.Errorf("%s", ReasonTampered)
		}
		seen[id] = true
		ts, _ := time.Parse(time.RFC3339Nano, stringGate(r["occurred_at"]))
		if ts.IsZero() || (!last.IsZero() && ts.Before(last)) {
			return out, fmt.Errorf("%s", ReasonMalformed)
		}
		last = ts
		if err := verifyOuterRecord(r, pub); err != nil {
			return out, err
		}
		if c, ok := r["control"].(map[string]any); ok {
			if err := verifyNestedControl(c, pub); err != nil {
				return out, fmt.Errorf("nested:%w", err)
			}
			if err := verifyControlPhase(c, controls); err != nil {
				return out, err
			}
			if k, ok := r["kind"].(string); ok {
				out.Statuses = append(out.Statuses, k)
			}
		}
	}
	sort.Strings(out.Statuses)
	out.AuthorityLineage = true
	return out, nil
}

func verifyControlPhase(c map[string]any, prior map[string]map[string]any) error {
	cmd, _ := c["command"].(string)
	phase, _ := c["phase"].(string)
	if cmd == "" || phase == "" {
		return fmt.Errorf("%s", ReasonMalformed)
	}
	prev := prior[cmd]
	if (cmd == "capability_invalidation" || cmd == "descendant_invalidation") && prev == nil {
		prev = prior["external_revocation"]
	}
	if phase == "requested" || phase == "attempted" {
		if prev != nil {
			return fmt.Errorf("%s", ReasonGateParent)
		}
		prior[cmd] = c
		return nil
	}
	if prev == nil {
		return fmt.Errorf("%s", ReasonGateParent)
	}
	if phase == "acknowledged" || phase == "failed" || phase == "denied" || phase == "invalidated" {
		if cmd == "capability_invalidation" || cmd == "descendant_invalidation" {
			return nil
		}
		if !sameControlEvidenceRef(c["causal_ref"], prev) {
			return fmt.Errorf("%s", ReasonGateParent)
		}
		if !sameString(c["boundary_id"], prev["boundary_id"]) || !sameString(c["resource_id"], prev["resource_id"]) || !sameString(c["adapter_identity"], prev["adapter_identity"]) {
			return fmt.Errorf("%s", ReasonGateParent)
		}
		if !scopeSubsetControl(c["affected_scope"], prev["affected_scope"]) {
			return fmt.Errorf("%s", ReasonGateAuthority)
		}
		if cmd != "external_revocation" {
			delete(prior, cmd)
		}
		return nil
	}
	return fmt.Errorf("%s", ReasonMalformed)
}
func sameString(a, b any) bool { sa, _ := a.(string); sb, _ := b.(string); return sa != "" && sa == sb }
func sameControlRef(a, b any) bool {
	am, aok := a.(map[string]any)
	bm, bok := b.(map[string]any)
	if !aok || !bok {
		return false
	}
	for _, k := range []string{"id", "digest", "schema_id", "schema_version", "source_product"} {
		if am[k] != bm[k] {
			return false
		}
	}
	return true
}
func sameControlEvidenceRef(a any, prev map[string]any) bool {
	m, ok := a.(map[string]any)
	if !ok {
		return false
	}
	id, _ := prev["evidence_id"].(string)
	return id != "" && m["id"] == id
}
func scopeSubsetControl(child, parent any) bool {
	ca, _ := child.([]any)
	pa, _ := parent.([]any)
	m := map[string]bool{}
	for _, v := range pa {
		if s, ok := v.(string); ok {
			m[s] = true
		}
	}
	for _, v := range ca {
		if s, ok := v.(string); ok && !m[s] {
			return false
		}
	}
	return true
}
func verifyOuterRecord(r map[string]any, pub ed25519.PublicKey) error {
	id, _ := r["record_id"].(string)
	sig, okSig := r["signature"].(map[string]any)
	if !okSig || id == "" {
		return fmt.Errorf("%s", ReasonControlSignature)
	}
	var signature proofsign.Signature
	signatureRaw, err := json.Marshal(sig)
	if err != nil || json.Unmarshal(signatureRaw, &signature) != nil || len(signature.SignedDigest) != 64 {
		return fmt.Errorf("%s", ReasonControlSignature)
	}
	r["record_id"] = ""
	r["signature"] = map[string]any{"alg": "", "key_id": "", "sig": ""}
	signable, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("%s", ReasonControlSignature)
	}
	digest, err := proofcanon.DigestJCS(signable)
	if err != nil {
		return fmt.Errorf("%s", ReasonControlSignature)
	}
	digest = strings.TrimPrefix(digest, "sha256:")
	if signature.SignedDigest != digest || id != "gait-lr-"+digest[:16] {
		return fmt.Errorf("%s", ReasonControlSignature)
	}
	verified, err := proofsign.VerifyDigestHex(pub, signature)
	if err != nil || !verified {
		return fmt.Errorf("%s", ReasonControlSignature)
	}
	return nil
}
func verifyNestedControl(c map[string]any, pub ed25519.PublicKey) error {
	evidenceID := stringGate(c["evidence_id"])
	declaredDigest := stringGate(c["canonical_content_digest"])
	if stringGate(c["schema_id"]) != "https://gait.dev/schemas/v1/action-contract/control-event-evidence.schema.json" || evidenceID == "" || !controlDigestPattern.MatchString(declaredDigest) {
		return fmt.Errorf("%s", ReasonMalformed)
	}
	for _, key := range []string{"event_ref", "causal_ref", "control_ref"} {
		if ref, ok := c[key].(map[string]any); !ok || stringGate(ref["id"]) == "" || !controlDigestPattern.MatchString(stringGate(ref["digest"])) || stringGate(ref["schema_id"]) == "" || stringGate(ref["schema_version"]) == "" || stringGate(ref["source_product"]) == "" {
			return fmt.Errorf("%s", ReasonMalformed)
		}
	}
	raw, err := json.Marshal(c)
	if err != nil {
		return fmt.Errorf("%s", ReasonControlSignature)
	}
	var signable map[string]any
	if json.Unmarshal(raw, &signable) != nil {
		return fmt.Errorf("%s", ReasonControlSignature)
	}
	delete(signable, "evidence_id")
	delete(signable, "canonical_content_digest")
	provenance, ok := signable["provenance"].(map[string]any)
	if !ok {
		return fmt.Errorf("%s", ReasonControlSignature)
	}
	signatureMap, ok := provenance["signature"].(map[string]any)
	if !ok {
		return fmt.Errorf("%s", ReasonControlSignature)
	}
	var signature proofsign.Signature
	signatureRaw, err := json.Marshal(signatureMap)
	if err != nil || json.Unmarshal(signatureRaw, &signature) != nil {
		return fmt.Errorf("%s", ReasonControlSignature)
	}
	delete(provenance, "signature")
	signableRaw, err := json.Marshal(signable)
	if err != nil {
		return fmt.Errorf("%s", ReasonControlSignature)
	}
	digest, err := proofcanon.DigestJCS(signableRaw)
	if err != nil {
		return fmt.Errorf("%s", ReasonControlSignature)
	}
	digest = strings.TrimPrefix(digest, "sha256:")
	if declaredDigest != "sha256:"+digest || evidenceID != "gait-control-"+digest[:16] || signature.SignedDigest != digest {
		return fmt.Errorf("%s", ReasonControlSignature)
	}
	declaredKey, err := base64.StdEncoding.DecodeString(stringGate(provenance["public_key"]))
	if err != nil || !bytes.Equal(declaredKey, pub) {
		return fmt.Errorf("%s", ReasonControlSignature)
	}
	verified, err := proofsign.VerifyDigestHex(pub, signature)
	if err != nil || !verified {
		return fmt.Errorf("%s", ReasonControlSignature)
	}
	return nil
}
func stringGate(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
