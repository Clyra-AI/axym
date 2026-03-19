package record

import (
	"encoding/json"
	"fmt"

	"github.com/Clyra-AI/axym/core/normalize"
	"github.com/Clyra-AI/axym/core/redact"
	recordschema "github.com/Clyra-AI/axym/schemas/v1/record"
	"github.com/Clyra-AI/proof"
)

type BuildInput struct {
	Normalized normalize.Record
	Redaction  redact.Config
}

func NormalizeAndBuild(in normalize.Input, cfg redact.Config) (*proof.Record, error) {
	normalized, err := normalize.Normalize(in)
	if err != nil {
		return nil, NewInvalidInputError(ReasonMappingError, "normalize input", err)
	}
	return Build(BuildInput{Normalized: normalized, Redaction: cfg})
}

func Build(in BuildInput) (*proof.Record, error) {
	event, metadata, err := redact.Apply(in.Normalized.Event, in.Normalized.Metadata, in.Redaction)
	if err != nil {
		return nil, NewInvalidInputError(ReasonMappingError, "redaction failed", err)
	}
	event, metadata = canonicalizeEventAndMetadata(event, metadata)

	payload := map[string]any{
		"source":         in.Normalized.Source,
		"source_product": in.Normalized.SourceProduct,
		"record_type":    in.Normalized.RecordType,
		"agent_id":       in.Normalized.AgentID,
		"timestamp":      in.Normalized.Timestamp.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"event":          event,
		"controls": map[string]any{
			"permissions_enforced": in.Normalized.Controls.PermissionsEnforced,
			"approved_scope":       in.Normalized.Controls.ApprovedScope,
		},
	}
	if metadata != nil {
		payload["metadata"] = metadata
	}
	payloadRaw, err := json.Marshal(payload)
	if err != nil {
		return nil, NewInvalidInputError(ReasonInvalidRecord, "marshal normalized payload", err)
	}

	if err := recordschema.ValidateNormalized(payloadRaw); err != nil {
		return nil, NewInvalidInputError(ReasonSchemaError, "normalized payload failed schema validation", err)
	}

	r, err := proof.NewRecord(proof.RecordOpts{
		Source:        in.Normalized.Source,
		SourceProduct: in.Normalized.SourceProduct,
		AgentID:       in.Normalized.AgentID,
		Type:          in.Normalized.RecordType,
		Timestamp:     in.Normalized.Timestamp,
		Event:         event,
		Metadata:      metadata,
		Controls: proof.Controls{
			PermissionsEnforced: in.Normalized.Controls.PermissionsEnforced,
			ApprovedScope:       in.Normalized.Controls.ApprovedScope,
		},
	})
	if err != nil {
		return nil, NewInvalidInputError(ReasonInvalidRecord, "create proof record", err)
	}
	if err := proof.ValidateRecord(r); err != nil {
		return nil, NewInvalidInputError(ReasonSchemaError, "proof record failed schema validation", err)
	}
	if r.Integrity.RecordHash == "" {
		return nil, NewInvalidInputError(ReasonInvalidRecord, "record hash is empty", fmt.Errorf("missing record hash"))
	}
	return r, nil
}
