package governance

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	proofcanon "github.com/Clyra-AI/proof/canon"
	proofsign "github.com/Clyra-AI/proof/signing"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	ReasonGateSignature = "GATE_SIGNATURE_INVALID"
	ReasonGateExpired   = "GATE_TOKEN_EXPIRED"
	ReasonGateAuthority = "GATE_AUTHORITY_EXPANDED"
	ReasonGateParent    = "GATE_PARENT_MISMATCH"
	ReasonGateDuplicate = "GATE_TOKEN_DUPLICATE"
)

var bareDigest = regexp.MustCompile(`^[a-f0-9]{64}$`)

type GateResult struct {
	AuthorityLineage bool     `json:"authority_lineage"`
	ReasonCodes      []string `json:"reason_codes"`
	TokenID          string   `json:"token_id"`
}

type GateGap struct {
	TokenID  string `json:"token_id,omitempty"`
	Code     string `json:"code"`
	Severity string `json:"severity"`
}

type GateChainResult struct {
	AuthorityLineage bool      `json:"authority_lineage"`
	Tokens           []string  `json:"tokens"`
	Gaps             []GateGap `json:"gaps,omitempty"`
}

// VerifyGateChain verifies every signed gate and compares each delegation to
// its exact parent. Failures are retained as explicit high-severity gaps.
func VerifyGateChain(raws [][]byte, pub ed25519.PublicKey, now time.Time) GateChainResult {
	result := GateChainResult{AuthorityLineage: len(raws) > 0}
	parsed := make([]map[string]any, len(raws))
	for i, raw := range raws {
		var value map[string]any
		if err := json.Unmarshal(raw, &value); err != nil {
			result.AuthorityLineage = false
			result.Gaps = append(result.Gaps, GateGap{Code: ReasonMalformed, Severity: "high"})
			continue
		}
		parsed[i] = value
	}
	counts := map[string]int{}
	for _, value := range parsed {
		if value != nil {
			counts[stringValueGate(value["token_id"])]++
		}
	}
	for i, raw := range raws {
		if parsed[i] == nil {
			continue
		}
		id := stringValueGate(parsed[i]["token_id"])
		if counts[id] > 1 {
			result.AuthorityLineage = false
			result.Gaps = append(result.Gaps, GateGap{TokenID: id, Code: ReasonGateDuplicate, Severity: "high"})
			continue
		}
		parent := map[string]any(nil)
		if pid, ok := parsed[i]["parent_token_id"].(string); ok {
			for _, candidate := range parsed {
				if candidate != nil && candidate["token_id"] == pid {
					parent = candidate
					break
				}
			}
			if parent == nil {
				result.AuthorityLineage = false
				result.Gaps = append(result.Gaps, GateGap{TokenID: stringValueGate(parsed[i]["token_id"]), Code: ReasonGateParent, Severity: "high"})
				continue
			}
		}
		verified, err := VerifyGateArtifact(raw, pub, now, parent)
		if err != nil {
			result.AuthorityLineage = false
			result.Gaps = append(result.Gaps, GateGap{TokenID: verified.TokenID, Code: err.Error(), Severity: "high"})
			continue
		}
		result.Tokens = append(result.Tokens, verified.TokenID)
	}
	sort.Strings(result.Tokens)
	sort.Slice(result.Gaps, func(i, j int) bool {
		if result.Gaps[i].TokenID != result.Gaps[j].TokenID {
			return result.Gaps[i].TokenID < result.Gaps[j].TokenID
		}
		return result.Gaps[i].Code < result.Gaps[j].Code
	})
	return result
}

func VerifyGateArtifact(raw []byte, pub ed25519.PublicKey, now time.Time, root map[string]any) (GateResult, error) {
	var v map[string]any
	if err := json.Unmarshal(raw, &v); err != nil {
		return GateResult{}, fmt.Errorf("%s", ReasonMalformed)
	}
	id := stringValueGate(v["token_id"])
	if err := validateGateShape(v); err != nil {
		return GateResult{TokenID: id}, err
	}
	parentID, hasParentID := v["parent_token_id"]
	parentDigest, hasParentDigest := v["parent_token_digest"]
	if hasParentID || hasParentDigest {
		pid, idOK := parentID.(string)
		pd, digestOK := parentDigest.(string)
		if !idOK || strings.TrimSpace(pid) == "" || !digestOK || strings.TrimSpace(pd) == "" {
			return GateResult{TokenID: id}, fmt.Errorf("%s", ReasonGateParent)
		}
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
	if keyID, _ := sig["key_id"].(string); keyID != proofsign.KeyID(pub) {
		return GateResult{TokenID: id}, fmt.Errorf("%s", ReasonGateSignature)
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
	created, _ := time.Parse(time.RFC3339Nano, stringValueGate(v["created_at"]))
	exp, _ := time.Parse(time.RFC3339Nano, stringValueGate(v["expires_at"]))
	if exp.IsZero() || (!now.IsZero() && !now.Before(exp)) {
		return GateResult{TokenID: id}, fmt.Errorf("%s", ReasonGateExpired)
	}
	if !exp.After(created) {
		return GateResult{TokenID: id}, fmt.Errorf("%s", ReasonMalformed)
	}
	if !now.IsZero() && now.Before(created) {
		return GateResult{TokenID: id}, fmt.Errorf("%s", ReasonGateExpired)
	}
	if root != nil {
		rootCopy := cloneGateMap(root)
		if err := gateSubset(rootCopy, v); err != nil {
			return GateResult{TokenID: id}, err
		}
	} else if hasParentID || hasParentDigest {
		// A token carrying any parent reference is never a root token. This
		// also closes the non-string parent-id case that cannot be resolved by
		// the chain lookup below.
		return GateResult{TokenID: id}, fmt.Errorf("%s", ReasonGateParent)
	}
	return GateResult{AuthorityLineage: true, TokenID: id, ReasonCodes: []string{}}, nil
}

// validateGateShape enforces the minimum signed gate contract before any
// authority comparison. A valid signature over an incomplete root is still
// not an authority grant: its schema, identities, delegation scope, and
// validity window must be explicit and non-empty.
func validateGateShape(v map[string]any) error {
	for _, key := range []string{"schema_id", "schema_version", "producer_version", "token_id", "created_at", "expires_at"} {
		value, ok := v[key].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s", ReasonMalformed)
		}
	}
	schemaID := v["schema_id"].(string)
	switch schemaID {
	case "gait.gate.delegation_token":
		if v["schema_version"] != "1.0.0" {
			return fmt.Errorf("%s", ReasonMalformed)
		}
		for _, key := range []string{"delegator_identity", "delegate_identity"} {
			value, ok := v[key].(string)
			if !ok || strings.TrimSpace(value) == "" {
				return fmt.Errorf("%s", ReasonMalformed)
			}
		}
		if v["delegator_identity"] == v["delegate_identity"] {
			return fmt.Errorf("%s", ReasonMalformed)
		}
	case "gait.gate.approval_token":
		if v["schema_version"] != "1.0.0" {
			return fmt.Errorf("%s", ReasonMalformed)
		}
		value, ok := v["approver_identity"].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s", ReasonMalformed)
		}
		for _, key := range []string{"reason_code", "intent_digest", "policy_digest"} {
			value, ok := v[key].(string)
			if !ok || strings.TrimSpace(value) == "" {
				return fmt.Errorf("%s", ReasonMalformed)
			}
		}
	default:
		return fmt.Errorf("%s", ReasonMalformed)
	}
	if _, err := time.Parse(time.RFC3339Nano, v["created_at"].(string)); err != nil {
		return fmt.Errorf("%s", ReasonMalformed)
	}
	if _, err := time.Parse(time.RFC3339Nano, v["expires_at"].(string)); err != nil {
		return fmt.Errorf("%s", ReasonMalformed)
	}
	scope, ok := v["scope"].([]any)
	if !ok || len(scope) == 0 {
		return fmt.Errorf("%s", ReasonMalformed)
	}
	for _, item := range scope {
		value, ok := item.(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s", ReasonMalformed)
		}
	}
	return nil
}

func gateSubset(root, child map[string]any) error {
	if _, hasID := child["parent_token_id"]; !hasID {
		if _, hasDigest := child["parent_token_digest"]; hasDigest {
			return fmt.Errorf("%s", ReasonGateParent)
		}
	}
	if pid, ok := child["parent_token_id"].(string); ok {
		if rid, _ := root["token_id"].(string); pid != rid {
			return fmt.Errorf("%s", ReasonGateParent)
		}
		pd, ok := child["parent_token_digest"].(string)
		if !ok || pd == "" {
			return fmt.Errorf("%s", ReasonGateParent)
		}
		if s, ok := root["signature"].(map[string]any); ok {
			if sd, _ := s["signed_digest"].(string); pd != "sha256:"+sd {
				return fmt.Errorf("%s", ReasonGateParent)
			}
		} else {
			return fmt.Errorf("%s", ReasonGateParent)
		}
	}
	if pd, ok := child["parent_token_digest"].(string); ok {
		if s, ok := root["signature"].(map[string]any); ok {
			if sd, _ := s["signed_digest"].(string); pd != "sha256:"+sd {
				return fmt.Errorf("%s", ReasonGateParent)
			}
		}
		delegator, childHasDelegator := child["delegator_identity"].(string)
		delegate, rootHasDelegate := root["delegate_identity"].(string)
		if !childHasDelegator || !rootHasDelegate || strings.TrimSpace(delegator) == "" || strings.TrimSpace(delegate) == "" || delegator != delegate {
			return fmt.Errorf("%s", ReasonGateAuthority)
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
		// These bindings define the authority's governing context. A child
		// cannot drop one while retaining a valid signature: every binding
		// present on the parent must be carried forward byte-for-byte.
		if _, parentHas := root[key]; parentHas {
			r, parentOK := root[key].(string)
			c, childOK := child[key].(string)
			if !parentOK || !childOK || strings.TrimSpace(r) == "" || c != r {
				return fmt.Errorf("%s", ReasonGateParent)
			}
		}
	}
	for _, key := range []string{"action_classes", "scope", "target_scope", "environment_scope", "data_classes", "network_destinations"} {
		if !subsetGate(stringsGate(root[key]), stringsGate(child[key])) {
			return fmt.Errorf("%s", ReasonGateAuthority)
		}
	}
	for _, key := range []string{"max_operations", "max_targets", "max_descendant_depth"} {
		if parentValue, parentHas := root[key]; parentHas {
			r, parentOK := parentValue.(float64)
			c, childOK := child[key].(float64)
			if !parentOK || !childOK || c < 0 || c > r {
				return fmt.Errorf("%s", ReasonGateAuthority)
			}
		}
	}
	if rootMax, ok := root["max_descendant_depth"].(float64); ok {
		childMax, childHasMax := child["max_descendant_depth"].(float64)
		childDepth, childHasDepth := child["depth"].(float64)
		rootDepth, rootHasDepth := root["depth"].(float64)
		if !rootHasDepth {
			rootDepth = 0
		}
		// A descendant consumes one level on every hop. Requiring both the
		// absolute depth and the strictly smaller remaining budget prevents a
		// child from repeating the parent's allowance indefinitely.
		if !childHasMax || !childHasDepth || childDepth != rootDepth+1 || childDepth > rootDepth+rootMax || childMax >= rootMax {
			return fmt.Errorf("%s", ReasonGateAuthority)
		}
	}
	return nil
}

func cloneGateMap(in map[string]any) map[string]any {
	raw, err := json.Marshal(in)
	if err != nil {
		return nil
	}
	var out map[string]any
	if json.Unmarshal(raw, &out) != nil {
		return nil
	}
	return out
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
