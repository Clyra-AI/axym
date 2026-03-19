package translate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Clyra-AI/axym/core/normalize"
	"github.com/Clyra-AI/proof"
)

const (
	NativeTypeTrace           = "trace"
	NativeTypeApprovalToken   = "approval_token"
	NativeTypeDelegationToken = "delegation_token"

	ReasonUnsupportedNativeType = "GAIT_UNSUPPORTED_NATIVE_TYPE"
	ReasonInvalidNativeRecord   = "GAIT_INVALID_NATIVE_RECORD"
)

type NativeRecord struct {
	Type         string              `json:"type"`
	Timestamp    string              `json:"timestamp"`
	AgentID      string              `json:"agent_id,omitempty"`
	Source       string              `json:"source,omitempty"`
	Event        map[string]any      `json:"event"`
	Metadata     map[string]any      `json:"metadata,omitempty"`
	Relationship *proof.Relationship `json:"relationship,omitempty"`
}

type Error struct {
	ReasonCode string
	Message    string
	Err        error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return fmt.Sprintf("%s: %s", e.ReasonCode, e.Message)
	}
	return fmt.Sprintf("%s: %s: %v", e.ReasonCode, e.Message, e.Err)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func Translate(native NativeRecord) (*proof.Record, error) {
	recordType, ok := recordTypeForNativeType(strings.TrimSpace(native.Type))
	if !ok {
		return nil, &Error{
			ReasonCode: ReasonUnsupportedNativeType,
			Message:    fmt.Sprintf("unsupported native type %q", native.Type),
		}
	}
	event := cloneMap(native.Event)
	if len(event) == 0 {
		return nil, &Error{
			ReasonCode: ReasonInvalidNativeRecord,
			Message:    "native event is required",
		}
	}
	metadata := cloneMap(native.Metadata)
	metadata["gait_native_type"] = strings.TrimSpace(native.Type)
	metadata["gait_translation"] = "v1"

	if compiled, ok := event["compiled_action"]; ok {
		digest, err := digestCompiledAction(compiled)
		if err != nil {
			return nil, &Error{
				ReasonCode: ReasonInvalidNativeRecord,
				Message:    "compiled_action digest synthesis failed",
				Err:        err,
			}
		}
		metadata["compiled_action_digest"] = digest
	}

	timestamp, err := parseTimestamp(native.Timestamp)
	if err != nil {
		return nil, &Error{
			ReasonCode: ReasonInvalidNativeRecord,
			Message:    "native timestamp is required and must be RFC3339",
			Err:        err,
		}
	}

	source := strings.TrimSpace(native.Source)
	if source == "" {
		source = "gait"
	}
	view := normalize.DeriveIdentityView(strings.TrimSpace(native.AgentID), event, metadata, native.Relationship)
	event, metadata, relationship := normalize.ApplyIdentityView(event, metadata, native.Relationship, view)
	record, err := proof.NewRecord(proof.RecordOpts{
		Timestamp:     timestamp,
		Source:        source,
		SourceProduct: "gait",
		AgentID:       strings.TrimSpace(native.AgentID),
		Type:          recordType,
		Event:         event,
		Metadata:      metadata,
		Relationship:  relationship,
		Controls: proof.Controls{
			PermissionsEnforced: true,
			ApprovedScope:       "gait-pack",
		},
	})
	if err != nil {
		return nil, &Error{
			ReasonCode: ReasonInvalidNativeRecord,
			Message:    "build translated proof record",
			Err:        err,
		}
	}
	return record, nil
}

func recordTypeForNativeType(nativeType string) (string, bool) {
	switch nativeType {
	case NativeTypeTrace:
		return "tool_invocation", true
	case NativeTypeApprovalToken:
		return "approval", true
	case NativeTypeDelegationToken:
		return "policy_enforcement", true
	default:
		return "", false
	}
}

func parseTimestamp(raw string) (time.Time, error) {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(raw))
	if err != nil {
		return time.Time{}, err
	}
	return t.UTC().Truncate(time.Second), nil
}

func digestCompiledAction(compiled any) (string, error) {
	raw, err := json.Marshal(compiled)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:]), nil
}

func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return map[string]any{}
	}
	raw, _ := json.Marshal(in)
	var out map[string]any
	_ = json.Unmarshal(raw, &out)
	if out == nil {
		return map[string]any{}
	}
	return out
}
