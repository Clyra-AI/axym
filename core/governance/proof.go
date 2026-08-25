package governance

import (
	"encoding/json"
	"fmt"
	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/axym/core/store/dedupe"
	"github.com/Clyra-AI/proof"
	"sort"
	"time"
)

// ToProofRecord creates an Axym-owned observation. Source-qualified claims
// and digest-bound relationship refs make the record auditable without
// granting execution authority.
func ToProofRecord(recordType, source, sourceProduct, id string, occurredAt time.Time, payload map[string]any, refs []Ref) (*proof.Record, error) {
	if occurredAt.IsZero() || id == "" || source == "" || sourceProduct == "" {
		return nil, fmt.Errorf("invalid projection identity")
	}
	rel := &proof.Relationship{}
	for _, r := range refs {
		if !validRef(r) {
			return nil, fmt.Errorf("invalid relationship reference %s", r.ID)
		}
		rel.EntityRefs = append(rel.EntityRefs, proof.RelationshipRef{Kind: r.Kind, ID: r.ID, Digest: r.Digest, SchemaID: r.SchemaID, SchemaVersion: r.SchemaVersion, SourceProduct: r.SourceProduct})
	}
	sort.Slice(rel.EntityRefs, func(i, j int) bool {
		if rel.EntityRefs[i].Kind == rel.EntityRefs[j].Kind {
			return rel.EntityRefs[i].ID < rel.EntityRefs[j].ID
		}
		return rel.EntityRefs[i].Kind < rel.EntityRefs[j].Kind
	})
	payload = clonePayload(payload)
	payload["projection_id"] = id
	payload["source_product"] = sourceProduct
	payload["source_claim"] = true
	payload["execution_authority"] = false
	if _, err := json.Marshal(payload); err != nil {
		return nil, err
	}
	return proof.NewRecord(proof.RecordOpts{Timestamp: occurredAt.UTC(), Source: source, SourceProduct: sourceProduct, AgentID: "axym://governance", Type: recordType, Event: payload, Relationship: rel, Controls: proof.Controls{}})
}

// AppendProjection uses the same Store.Append signing, chaining, and dedupe
// path as every other Axym record. No alternate key or chain format exists.
func AppendProjection(st *store.Store, rec *proof.Record) (store.AppendResult, error) {
	if st == nil || rec == nil {
		return store.AppendResult{}, fmt.Errorf("store and record are required")
	}
	key, err := dedupe.BuildKey(rec.SourceProduct, rec.RecordType, rec.Event)
	if err != nil {
		return store.AppendResult{}, err
	}
	return st.Append(rec, key)
}
func clonePayload(in map[string]any) map[string]any {
	o := map[string]any{}
	for k, v := range in {
		o[k] = v
	}
	return o
}
