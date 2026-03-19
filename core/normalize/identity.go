package normalize

import (
	"encoding/json"
	"strings"

	"github.com/Clyra-AI/proof"
)

type IdentityView struct {
	ActorIdentity      string                `json:"actor_identity,omitempty"`
	DownstreamIdentity string                `json:"downstream_identity,omitempty"`
	OwnerIdentity      string                `json:"owner_identity,omitempty"`
	PolicyDigest       string                `json:"policy_digest,omitempty"`
	ApprovalTokenRef   string                `json:"approval_token_ref,omitempty"`
	TargetKind         string                `json:"target_kind,omitempty"`
	TargetID           string                `json:"target_id,omitempty"`
	DelegationChain    []proof.AgentChainHop `json:"delegation_chain,omitempty"`
}

func (v IdentityView) Empty() bool {
	return strings.TrimSpace(v.ActorIdentity) == "" &&
		strings.TrimSpace(v.DownstreamIdentity) == "" &&
		strings.TrimSpace(v.OwnerIdentity) == "" &&
		strings.TrimSpace(v.PolicyDigest) == "" &&
		strings.TrimSpace(v.ApprovalTokenRef) == "" &&
		strings.TrimSpace(v.TargetKind) == "" &&
		strings.TrimSpace(v.TargetID) == "" &&
		len(v.DelegationChain) == 0
}

func IdentityViewFromRecord(record *proof.Record) IdentityView {
	if record == nil {
		return IdentityView{}
	}
	view := DeriveIdentityView(record.AgentID, record.Event, record.Metadata, record.Relationship)
	if view.OwnerIdentity == "" && record.Controls.HumanOversight != nil {
		view.OwnerIdentity = strings.TrimSpace(record.Controls.HumanOversight.Reviewer)
	}
	return view
}

func DeriveIdentityView(agentID string, event map[string]any, metadata map[string]any, relationship *proof.Relationship) IdentityView {
	view := IdentityView{
		ActorIdentity: firstString(
			stringFromMap(event, "actor_identity"),
			stringFromMap(metadata, "actor_identity"),
			stringFromMap(event, "actor_id"),
			stringFromMap(metadata, "actor_id"),
			stringFromMap(event, "principal_id"),
			stringFromMap(metadata, "principal_id"),
		),
		DownstreamIdentity: firstString(
			stringFromMap(event, "downstream_identity"),
			stringFromMap(metadata, "downstream_identity"),
			lastAgentChainIdentity(relationship),
			stringFromMap(event, "delegate_identity"),
			stringFromMap(metadata, "delegate_identity"),
			strings.TrimSpace(agentID),
		),
		OwnerIdentity: firstString(
			stringFromMap(event, "owner_identity"),
			stringFromMap(metadata, "owner_identity"),
			stringFromMap(event, "approver_identity"),
			stringFromMap(metadata, "approver_identity"),
		),
		PolicyDigest: firstString(
			stringFromMap(event, "policy_digest"),
			stringFromMap(metadata, "policy_digest"),
			policyDigestFromRelationship(relationship),
		),
		ApprovalTokenRef: firstString(
			stringFromMap(event, "approval_token_ref"),
			stringFromMap(metadata, "approval_token_ref"),
			stringFromMap(event, "context_approval_ref"),
			stringFromMap(metadata, "context_approval_ref"),
			stringFromMap(event, "approval_ref"),
			stringFromMap(metadata, "approval_ref"),
		),
		TargetKind: firstString(
			stringFromMap(event, "target_kind"),
			stringFromMap(metadata, "target_kind"),
		),
		TargetID: firstString(
			stringFromMap(event, "target_id"),
			stringFromMap(metadata, "target_id"),
		),
		DelegationChain: firstNonEmptyChain(
			chainFromAny(event["delegation_chain"]),
			chainFromAny(metadata["delegation_chain"]),
			cloneAgentChain(nilIfEmptyAgentChain(relationship)),
		),
	}

	if view.TargetID == "" {
		targetID := firstString(
			stringFromMap(event, "tool_name"),
			stringFromMap(event, "tool"),
			stringFromMap(event, "scope"),
			stringFromMap(event, "privilege"),
			stringFromMap(metadata, "scope"),
		)
		if targetID != "" {
			view.TargetID = targetID
		}
	}
	if view.TargetKind == "" && view.TargetID != "" {
		switch {
		case stringFromMap(event, "tool_name") != "" || stringFromMap(event, "tool") != "":
			view.TargetKind = "tool"
		case stringFromMap(event, "privilege") != "" || stringFromMap(event, "scope") != "" || stringFromMap(metadata, "scope") != "":
			view.TargetKind = "privilege"
		}
	}
	if view.TargetKind == "" || view.TargetID == "" {
		kind, id := targetFromRelationship(relationship)
		if view.TargetKind == "" {
			view.TargetKind = kind
		}
		if view.TargetID == "" {
			view.TargetID = id
		}
	}

	if view.ActorIdentity == "" && len(view.DelegationChain) > 0 {
		view.ActorIdentity = strings.TrimSpace(view.DelegationChain[0].Identity)
	}
	if view.DownstreamIdentity == "" && len(view.DelegationChain) > 0 {
		view.DownstreamIdentity = strings.TrimSpace(view.DelegationChain[len(view.DelegationChain)-1].Identity)
	}
	if view.ActorIdentity == "" {
		view.ActorIdentity = view.DownstreamIdentity
	}
	if view.DownstreamIdentity == "" {
		view.DownstreamIdentity = view.ActorIdentity
	}
	if len(view.DelegationChain) == 0 {
		switch {
		case view.ActorIdentity != "" && view.DownstreamIdentity != "" && view.ActorIdentity != view.DownstreamIdentity:
			view.DelegationChain = []proof.AgentChainHop{
				{Identity: view.ActorIdentity, Role: "requester"},
				{Identity: view.DownstreamIdentity, Role: "delegate"},
			}
		case view.DownstreamIdentity != "":
			view.DelegationChain = []proof.AgentChainHop{{Identity: view.DownstreamIdentity, Role: "delegate"}}
		}
	}
	return view
}

func ApplyIdentityView(event map[string]any, metadata map[string]any, relationship *proof.Relationship, view IdentityView) (map[string]any, map[string]any, *proof.Relationship) {
	if event == nil {
		event = map[string]any{}
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	if view.Empty() {
		return event, metadata, cloneRelationship(relationship)
	}

	rel := cloneRelationship(relationship)
	if rel == nil {
		rel = &proof.Relationship{}
	}

	setStringIfMissing(event, "actor_identity", view.ActorIdentity)
	setStringIfMissing(event, "downstream_identity", view.DownstreamIdentity)
	setStringIfMissing(event, "owner_identity", view.OwnerIdentity)
	setStringIfMissing(event, "policy_digest", view.PolicyDigest)
	setStringIfMissing(event, "approval_token_ref", view.ApprovalTokenRef)
	setStringIfMissing(event, "target_kind", view.TargetKind)
	setStringIfMissing(event, "target_id", view.TargetID)
	if len(view.DelegationChain) > 0 && !mapHasKey(event, "delegation_chain") {
		event["delegation_chain"] = chainToAny(view.DelegationChain)
	}

	if view.PolicyDigest != "" {
		if rel.PolicyRef == nil {
			rel.PolicyRef = &proof.PolicyRef{}
		}
		if strings.TrimSpace(rel.PolicyRef.PolicyDigest) == "" {
			rel.PolicyRef.PolicyDigest = view.PolicyDigest
		}
	}
	if len(view.DelegationChain) > 0 && len(rel.AgentChain) == 0 {
		rel.AgentChain = cloneAgentChain(view.DelegationChain)
	}
	appendEntityRef(rel, "agent", view.ActorIdentity)
	appendEntityRef(rel, "agent", view.DownstreamIdentity)
	appendEntityRef(rel, relationshipEntityKind(view.TargetKind), view.TargetID)

	if relationshipEmpty(rel) {
		return event, metadata, nil
	}
	return event, metadata, rel
}

func chainToAny(hops []proof.AgentChainHop) []map[string]any {
	out := make([]map[string]any, 0, len(hops))
	for _, hop := range hops {
		identity := strings.TrimSpace(hop.Identity)
		role := strings.TrimSpace(hop.Role)
		if identity == "" {
			continue
		}
		item := map[string]any{"identity": identity}
		if role != "" {
			item["role"] = role
		}
		out = append(out, item)
	}
	return out
}

func chainFromAny(value any) []proof.AgentChainHop {
	items, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]proof.AgentChainHop, 0, len(items))
	for _, item := range items {
		entry, ok := item.(map[string]any)
		if !ok {
			continue
		}
		identity := strings.TrimSpace(stringFromMap(entry, "identity"))
		if identity == "" {
			continue
		}
		out = append(out, proof.AgentChainHop{
			Identity: identity,
			Role:     strings.TrimSpace(stringFromMap(entry, "role")),
		})
	}
	return out
}

func stringFromMap(m map[string]any, key string) string {
	if len(m) == 0 {
		return ""
	}
	raw, ok := m[key]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return strings.TrimSpace(value)
}

func firstString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func firstNonEmptyChain(chains ...[]proof.AgentChainHop) []proof.AgentChainHop {
	for _, chain := range chains {
		if len(chain) > 0 {
			return cloneAgentChain(chain)
		}
	}
	return nil
}

func nilIfEmptyAgentChain(relationship *proof.Relationship) []proof.AgentChainHop {
	if relationship == nil || len(relationship.AgentChain) == 0 {
		return nil
	}
	return relationship.AgentChain
}

func cloneAgentChain(in []proof.AgentChainHop) []proof.AgentChainHop {
	if len(in) == 0 {
		return nil
	}
	out := make([]proof.AgentChainHop, 0, len(in))
	for _, hop := range in {
		identity := strings.TrimSpace(hop.Identity)
		if identity == "" {
			continue
		}
		out = append(out, proof.AgentChainHop{
			Identity: identity,
			Role:     strings.TrimSpace(hop.Role),
			Extra:    cloneRawMap(hop.Extra),
		})
	}
	return out
}

func cloneRelationship(in *proof.Relationship) *proof.Relationship {
	if in == nil {
		return nil
	}
	out := &proof.Relationship{
		ParentRecordID:   strings.TrimSpace(in.ParentRecordID),
		RelatedRecordIDs: append([]string(nil), in.RelatedRecordIDs...),
		RelatedEntityIDs: append([]string(nil), in.RelatedEntityIDs...),
		AgentLineage:     cloneAgentLineage(in.AgentLineage),
		Extra:            cloneRawMap(in.Extra),
	}
	if in.ParentRef != nil {
		out.ParentRef = &proof.RelationshipRef{
			Kind:  strings.TrimSpace(in.ParentRef.Kind),
			ID:    strings.TrimSpace(in.ParentRef.ID),
			Extra: cloneRawMap(in.ParentRef.Extra),
		}
	}
	if len(in.EntityRefs) > 0 {
		out.EntityRefs = make([]proof.RelationshipRef, 0, len(in.EntityRefs))
		for _, ref := range in.EntityRefs {
			if strings.TrimSpace(ref.Kind) == "" || strings.TrimSpace(ref.ID) == "" {
				continue
			}
			out.EntityRefs = append(out.EntityRefs, proof.RelationshipRef{
				Kind:  strings.TrimSpace(ref.Kind),
				ID:    strings.TrimSpace(ref.ID),
				Extra: cloneRawMap(ref.Extra),
			})
		}
	}
	if in.PolicyRef != nil {
		out.PolicyRef = &proof.PolicyRef{
			PolicyID:       strings.TrimSpace(in.PolicyRef.PolicyID),
			PolicyVersion:  strings.TrimSpace(in.PolicyRef.PolicyVersion),
			PolicyDigest:   strings.TrimSpace(in.PolicyRef.PolicyDigest),
			MatchedRuleIDs: append([]string(nil), in.PolicyRef.MatchedRuleIDs...),
			Extra:          cloneRawMap(in.PolicyRef.Extra),
		}
	}
	if len(in.AgentChain) > 0 {
		out.AgentChain = cloneAgentChain(in.AgentChain)
	}
	if len(in.Edges) > 0 {
		out.Edges = make([]proof.RelationshipEdge, 0, len(in.Edges))
		for _, edge := range in.Edges {
			out.Edges = append(out.Edges, proof.RelationshipEdge{
				Kind: strings.TrimSpace(edge.Kind),
				From: proof.RelationshipRef{
					Kind:  strings.TrimSpace(edge.From.Kind),
					ID:    strings.TrimSpace(edge.From.ID),
					Extra: cloneRawMap(edge.From.Extra),
				},
				To: proof.RelationshipRef{
					Kind:  strings.TrimSpace(edge.To.Kind),
					ID:    strings.TrimSpace(edge.To.ID),
					Extra: cloneRawMap(edge.To.Extra),
				},
				Extra: cloneRawMap(edge.Extra),
			})
		}
	}
	return out
}

func cloneAgentLineage(in []proof.AgentLineageHop) []proof.AgentLineageHop {
	if len(in) == 0 {
		return nil
	}
	out := make([]proof.AgentLineageHop, 0, len(in))
	for _, hop := range in {
		if strings.TrimSpace(hop.AgentID) == "" {
			continue
		}
		out = append(out, proof.AgentLineageHop{
			AgentID:            strings.TrimSpace(hop.AgentID),
			DelegatedBy:        strings.TrimSpace(hop.DelegatedBy),
			DelegationRecordID: strings.TrimSpace(hop.DelegationRecordID),
			Extra:              cloneRawMap(hop.Extra),
		})
	}
	return out
}

func cloneRawMap(in map[string]json.RawMessage) map[string]json.RawMessage {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]json.RawMessage, len(in))
	for key, value := range in {
		out[key] = append(json.RawMessage(nil), value...)
	}
	return out
}

func mapHasKey(m map[string]any, key string) bool {
	if len(m) == 0 {
		return false
	}
	_, ok := m[key]
	return ok
}

func setStringIfMissing(m map[string]any, key string, value string) {
	if strings.TrimSpace(value) == "" || mapHasKey(m, key) {
		return
	}
	m[key] = strings.TrimSpace(value)
}

func appendEntityRef(relationship *proof.Relationship, kind string, id string) {
	if relationship == nil {
		return
	}
	kind = strings.TrimSpace(kind)
	id = strings.TrimSpace(id)
	if kind == "" || id == "" {
		return
	}
	for _, ref := range relationship.EntityRefs {
		if strings.TrimSpace(ref.Kind) == kind && strings.TrimSpace(ref.ID) == id {
			return
		}
	}
	relationship.EntityRefs = append(relationship.EntityRefs, proof.RelationshipRef{Kind: kind, ID: id})
}

func relationshipEntityKind(kind string) string {
	switch strings.TrimSpace(kind) {
	case "agent", "tool", "resource", "policy", "run", "trace", "delegation", "evidence":
		return strings.TrimSpace(kind)
	case "":
		return ""
	default:
		return "resource"
	}
}

func policyDigestFromRelationship(relationship *proof.Relationship) string {
	if relationship == nil || relationship.PolicyRef == nil {
		return ""
	}
	return strings.TrimSpace(relationship.PolicyRef.PolicyDigest)
}

func lastAgentChainIdentity(relationship *proof.Relationship) string {
	if relationship == nil || len(relationship.AgentChain) == 0 {
		return ""
	}
	return strings.TrimSpace(relationship.AgentChain[len(relationship.AgentChain)-1].Identity)
}

func targetFromRelationship(relationship *proof.Relationship) (string, string) {
	if relationship == nil {
		return "", ""
	}
	for _, ref := range relationship.EntityRefs {
		kind := strings.TrimSpace(ref.Kind)
		id := strings.TrimSpace(ref.ID)
		if kind == "" || id == "" {
			continue
		}
		switch kind {
		case "agent", "identity", "owner", "policy", "approver":
			continue
		default:
			return kind, id
		}
	}
	return "", ""
}

func relationshipEmpty(relationship *proof.Relationship) bool {
	if relationship == nil {
		return true
	}
	return relationship.ParentRef == nil &&
		len(relationship.EntityRefs) == 0 &&
		relationship.PolicyRef == nil &&
		len(relationship.AgentChain) == 0 &&
		len(relationship.Edges) == 0 &&
		strings.TrimSpace(relationship.ParentRecordID) == "" &&
		len(relationship.RelatedRecordIDs) == 0 &&
		len(relationship.RelatedEntityIDs) == 0 &&
		len(relationship.AgentLineage) == 0 &&
		len(relationship.Extra) == 0
}
