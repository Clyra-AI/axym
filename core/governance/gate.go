package governance

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	proofcanon "github.com/Clyra-AI/proof/canon"
	proofsign "github.com/Clyra-AI/proof/signing"
	"regexp"
	"sort"
	"time"
)

const (
	ReasonGateSignature = "GATE_SIGNATURE_INVALID"
	ReasonGateExpired   = "GATE_TOKEN_EXPIRED"
	ReasonGateAuthority = "GATE_AUTHORITY_EXPANDED"
	ReasonGateParent    = "GATE_PARENT_MISMATCH"
)

var bareDigest = regexp.MustCompile(`^[a-f0-9]{64}$`)

type GateResult struct {
	AuthorityLineage bool     `json:"authority_lineage"`
	ReasonCodes      []string `json:"reason_codes"`
	TokenID          string   `json:"token_id"`
}

func VerifyGateArtifact(raw []byte, pub ed25519.PublicKey, now time.Time, root map[string]any) (GateResult, error) {
	var v map[string]any
	if err := json.Unmarshal(raw, &v); err != nil {
		return GateResult{}, fmt.Errorf("%s", ReasonMalformed)
	}
	id, _ := v["token_id"].(string)
	if id == "" {
		return GateResult{}, fmt.Errorf("%s", ReasonMalformed)
	}
	if revoked, ok := v["revoked"].(bool); ok && revoked {
		return GateResult{TokenID: id}, fmt.Errorf("%s", ReasonGateParent)
	}
	sig, ok := v["signature"].(map[string]any)
	if !ok {
		return GateResult{}, fmt.Errorf("%s", ReasonGateSignature)
	}
	signed, _ := sig["signed_digest"].(string)
	if !bareDigest.MatchString(signed) || len(pub) != ed25519.PublicKeySize {
		return GateResult{}, fmt.Errorf("%s", ReasonGateSignature)
	}
	delete(v, "signature")
	canon, err := json.Marshal(v)
	if err != nil {
		return GateResult{}, err
	}
	d, err := proofcanon.DigestJCS(canon)
	if err != nil || d != signed {
		return GateResult{}, fmt.Errorf("%s", ReasonGateSignature)
	}
	var ps proofsign.Signature
	b, _ := json.Marshal(sig)
	_ = json.Unmarshal(b, &ps)
	valid, err := proofsign.VerifyDigestHex(pub, ps)
	if err != nil || !valid {
		return GateResult{}, fmt.Errorf("%s", ReasonGateSignature)
	}
	exp, _ := time.Parse(time.RFC3339Nano, stringValueGate(v["expires_at"]))
	if exp.IsZero() || (!now.IsZero() && !now.Before(exp)) {
		return GateResult{TokenID: id}, fmt.Errorf("%s", ReasonGateExpired)
	}
	if root != nil {
		if err := gateSubset(root, v); err != nil {
			return GateResult{TokenID: id}, err
		}
	}
	return GateResult{AuthorityLineage: true, TokenID: id, ReasonCodes: []string{}}, nil
}
func gateSubset(root, child map[string]any) error {
	if pid, ok := child["parent_token_id"].(string); ok {
		if rid, _ := root["token_id"].(string); pid != rid {
			return fmt.Errorf("%s", ReasonGateParent)
		}
	}
	if pd, ok := child["parent_token_digest"].(string); ok {
		if s, ok := root["signature"].(map[string]any); ok {
			if sd, _ := s["signed_digest"].(string); pd != "sha256:"+sd {
				return fmt.Errorf("%s", ReasonGateParent)
			}
		}
	}
	if r, ok := root["expires_at"].(string); ok {
		if c, ok := child["expires_at"].(string); ok {
			rt, _ := time.Parse(time.RFC3339Nano, r)
			ct, _ := time.Parse(time.RFC3339Nano, c)
			if ct.After(rt) {
				return fmt.Errorf("%s", ReasonGateAuthority)
			}
		}
	}
	for _, key := range []string{"contract_digest", "policy_digest", "origin_authority_digest"} {
		if r, ok := root[key].(string); ok {
			if c, ok := child[key].(string); ok && r != c {
				return fmt.Errorf("%s", ReasonGateParent)
			}
		}
	}
	for _, key := range []string{"action_classes", "target_scope", "environment_scope", "data_classes", "network_destinations"} {
		if !subsetGate(stringsGate(root[key]), stringsGate(child[key])) {
			return fmt.Errorf("%s", ReasonGateAuthority)
		}
	}
	for _, key := range []string{"max_operations", "max_targets", "max_descendant_depth"} {
		if r, ok := root[key].(float64); ok {
			if c, ok := child[key].(float64); ok && c > r {
				return fmt.Errorf("%s", ReasonGateAuthority)
			}
		}
	}
	return nil
}
func stringsGate(v any) []string {
	a, _ := v.([]any)
	o := make([]string, 0, len(a))
	for _, x := range a {
		if s, ok := x.(string); ok {
			o = append(o, s)
		}
	}
	sort.Strings(o)
	return o
}
func subsetGate(root, child []string) bool {
	m := map[string]bool{}
	for _, v := range root {
		m[v] = true
	}
	for _, v := range child {
		if !m[v] {
			return false
		}
	}
	return true
}
func stringValueGate(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}
