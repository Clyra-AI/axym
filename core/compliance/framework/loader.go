package framework

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Clyra-AI/proof"
	proofframework "github.com/Clyra-AI/proof/core/framework"
)

const (
	ReasonFrameworkMissing = "FRAMEWORK_MISSING"
	ReasonFrameworkLoad    = "FRAMEWORK_LOAD_FAILED"
)

type Definition struct {
	ID       string    `json:"id"`
	Version  string    `json:"version"`
	Title    string    `json:"title"`
	Controls []Control `json:"controls"`
}

type Control struct {
	FrameworkID         string   `json:"framework_id"`
	ID                  string   `json:"id"`
	Title               string   `json:"title"`
	RequiredRecordTypes []string `json:"required_record_types"`
	RequiredFields      []string `json:"required_fields"`
	MinimumFrequency    string   `json:"minimum_frequency"`
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

func LoadMany(frameworkIDs []string) ([]Definition, error) {
	ids := normalizeIDs(frameworkIDs)
	if len(ids) == 0 {
		return nil, &Error{
			ReasonCode: ReasonFrameworkMissing,
			Message:    "at least one framework id is required",
		}
	}

	definitions := make([]Definition, 0, len(ids))
	for _, id := range ids {
		fw, err := proof.LoadFramework(id)
		if err != nil {
			return nil, &Error{
				ReasonCode: ReasonFrameworkLoad,
				Message:    fmt.Sprintf("load framework %q", id),
				Err:        err,
			}
		}
		definition := Definition{
			ID:      strings.TrimSpace(fw.Framework.ID),
			Version: strings.TrimSpace(fw.Framework.Version),
			Title:   strings.TrimSpace(fw.Framework.Title),
		}
		flattenControls(definition.ID, fw.Controls, &definition.Controls)
		sort.Slice(definition.Controls, func(i, j int) bool {
			return definition.Controls[i].ID < definition.Controls[j].ID
		})
		definitions = append(definitions, definition)
	}
	sort.Slice(definitions, func(i, j int) bool {
		return definitions[i].ID < definitions[j].ID
	})
	return definitions, nil
}

func normalizeIDs(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, candidate := range in {
		raw := strings.TrimSpace(candidate)
		if raw == "" {
			continue
		}
		id := raw
		if !isLikelyPath(raw) {
			id = strings.ToLower(raw)
		}
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func isLikelyPath(value string) bool {
	if value == "" {
		return false
	}
	if filepath.IsAbs(value) {
		return true
	}
	if strings.HasPrefix(value, ".") {
		return true
	}
	return strings.ContainsAny(value, `/\`)
}

func flattenControls(frameworkID string, controls []proofframework.Control, out *[]Control) {
	for _, control := range controls {
		recordTypes := append([]string(nil), control.RequiredRecordTypes...)
		fields := append([]string(nil), control.RequiredFields...)
		sort.Strings(recordTypes)
		sort.Strings(fields)
		*out = append(*out, Control{
			FrameworkID:         frameworkID,
			ID:                  strings.TrimSpace(control.ID),
			Title:               strings.TrimSpace(control.Title),
			RequiredRecordTypes: recordTypes,
			RequiredFields:      fields,
			MinimumFrequency:    strings.TrimSpace(control.MinimumFrequency),
		})
		flattenControls(frameworkID, control.Children, out)
	}
}
