package redact

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Clyra-AI/proof"
)

type Action string

const (
	ActionHash Action = "hash"
	ActionOmit Action = "omit"
	ActionMask Action = "mask"
)

type Rule struct {
	Path   string
	Action Action
}

type Config struct {
	EventRules    []Rule
	MetadataRules []Rule
}

func Apply(event map[string]any, metadata map[string]any, cfg Config) (map[string]any, map[string]any, error) {
	redactedEvent, err := cloneObject(event)
	if err != nil {
		return nil, nil, fmt.Errorf("clone event: %w", err)
	}
	redactedMetadata, err := cloneObject(metadata)
	if err != nil {
		return nil, nil, fmt.Errorf("clone metadata: %w", err)
	}

	if err := applyRules(redactedEvent, cfg.EventRules); err != nil {
		return nil, nil, fmt.Errorf("apply event rules: %w", err)
	}
	if err := applyRules(redactedMetadata, cfg.MetadataRules); err != nil {
		return nil, nil, fmt.Errorf("apply metadata rules: %w", err)
	}

	return redactedEvent, redactedMetadata, nil
}

func applyRules(obj map[string]any, rules []Rule) error {
	sorted := append([]Rule(nil), rules...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Path == sorted[j].Path {
			return sorted[i].Action < sorted[j].Action
		}
		return sorted[i].Path < sorted[j].Path
	})
	for _, rule := range sorted {
		path := strings.TrimSpace(rule.Path)
		if path == "" {
			return fmt.Errorf("rule path is required")
		}
		switch rule.Action {
		case ActionHash, ActionOmit, ActionMask:
		default:
			return fmt.Errorf("unsupported redaction action %q for path %q", rule.Action, path)
		}
		segments := strings.Split(path, ".")
		if err := applyRule(obj, segments, rule.Action); err != nil {
			return err
		}
	}
	return nil
}

func applyRule(node map[string]any, segments []string, action Action) error {
	if len(segments) == 0 {
		return nil
	}
	key := segments[0]
	value, exists := node[key]
	if !exists {
		return nil
	}
	if len(segments) > 1 {
		next, ok := value.(map[string]any)
		if !ok {
			return fmt.Errorf("redaction path %q references non-object value", strings.Join(segments, "."))
		}
		return applyRule(next, segments[1:], action)
	}

	switch action {
	case ActionOmit:
		delete(node, key)
	case ActionMask:
		node[key] = "***"
	case ActionHash:
		h, err := hashValue(value)
		if err != nil {
			return err
		}
		node[key] = h
	}
	return nil
}

func hashValue(value any) (string, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal value for hash: %w", err)
	}
	canonical, err := proof.Canonicalize(raw, proof.DomainJSON)
	if err != nil {
		return "", fmt.Errorf("canonicalize value for hash: %w", err)
	}
	sum := sha256.Sum256(canonical)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func cloneObject(in map[string]any) (map[string]any, error) {
	if in == nil {
		return map[string]any{}, nil
	}
	raw, err := json.Marshal(in)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, err
	}
	if out == nil {
		return map[string]any{}, nil
	}
	return out, nil
}
