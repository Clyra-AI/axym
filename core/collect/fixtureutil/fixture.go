package fixtureutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Event struct {
	Timestamp string         `json:"timestamp"`
	AgentID   string         `json:"agent_id,omitempty"`
	Event     map[string]any `json:"event"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

type File struct {
	Events []Event `json:"events"`
}

func LoadEvents(dir string, fileName string) ([]Event, error) {
	if dir == "" {
		return nil, nil
	}
	path := filepath.Join(dir, fileName)
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read fixture %s: %w", path, err)
	}
	var fixture File
	if err := json.Unmarshal(raw, &fixture); err != nil {
		return nil, fmt.Errorf("decode fixture %s: %w", path, err)
	}
	return fixture.Events, nil
}

func ParseTimestamp(raw string, fallback time.Time) (time.Time, error) {
	if raw == "" {
		return fallback.UTC().Truncate(time.Second), nil
	}
	ts, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse timestamp %q: %w", raw, err)
	}
	return ts.UTC().Truncate(time.Second), nil
}
