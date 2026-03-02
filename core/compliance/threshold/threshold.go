package threshold

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Clyra-AI/proof"
)

const (
	ReasonThresholdNotMet = "COVERAGE_THRESHOLD_NOT_MET"
	ReasonInvalidEvidence = "INVALID_EVIDENCE_CLASS"
)

var invalidEvidenceClasses = map[string]struct{}{
	"invalid_record": {},
	"schema_error":   {},
	"mapping_error":  {},
}

type PolicyConfig struct {
	GlobalMinCoverage    *float64           `json:"global_min_coverage,omitempty"`
	FrameworkMinCoverage map[string]float64 `json:"framework_min_coverage,omitempty"`
}

type FrameworkCoverage struct {
	FrameworkID     string   `json:"framework_id"`
	Coverage        float64  `json:"coverage"`
	FailingControls []string `json:"failing_controls,omitempty"`
}

type Failure struct {
	FrameworkID      string   `json:"framework_id"`
	RequiredCoverage float64  `json:"required_coverage"`
	ActualCoverage   float64  `json:"actual_coverage"`
	FailingControls  []string `json:"failing_controls,omitempty"`
}

type Evaluation struct {
	Passed   bool      `json:"passed"`
	Failures []Failure `json:"failures,omitempty"`
}

func LoadPolicy(path string) (PolicyConfig, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return PolicyConfig{}, nil
	}
	// #nosec G304 -- policy path is explicit user input from CLI.
	raw, err := os.ReadFile(trimmed)
	if err != nil {
		return PolicyConfig{}, fmt.Errorf("read policy config: %w", err)
	}
	var cfg PolicyConfig
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return PolicyConfig{}, fmt.Errorf("decode policy config: %w", err)
	}
	cfg.FrameworkMinCoverage = normalizeFrameworkThresholds(cfg.FrameworkMinCoverage)
	if cfg.GlobalMinCoverage != nil {
		value := clamp01(*cfg.GlobalMinCoverage)
		cfg.GlobalMinCoverage = &value
	}
	return cfg, nil
}

func Evaluate(frameworks []FrameworkCoverage, policy PolicyConfig, override *float64) Evaluation {
	failures := make([]Failure, 0)
	for _, framework := range frameworks {
		minimum, enabled := ResolveMinimum(policy, framework.FrameworkID, override)
		if !enabled {
			continue
		}
		if framework.Coverage+1e-9 >= minimum {
			continue
		}
		controls := append([]string(nil), framework.FailingControls...)
		sort.Strings(controls)
		failures = append(failures, Failure{
			FrameworkID:      framework.FrameworkID,
			RequiredCoverage: minimum,
			ActualCoverage:   framework.Coverage,
			FailingControls:  controls,
		})
	}
	sort.Slice(failures, func(i, j int) bool {
		if failures[i].FrameworkID != failures[j].FrameworkID {
			return failures[i].FrameworkID < failures[j].FrameworkID
		}
		return failures[i].RequiredCoverage > failures[j].RequiredCoverage
	})
	return Evaluation{Passed: len(failures) == 0, Failures: failures}
}

func ResolveMinimum(policy PolicyConfig, frameworkID string, override *float64) (float64, bool) {
	if override != nil {
		return clamp01(*override), true
	}
	if value, ok := policy.FrameworkMinCoverage[strings.ToLower(strings.TrimSpace(frameworkID))]; ok {
		return value, true
	}
	if policy.GlobalMinCoverage != nil {
		return clamp01(*policy.GlobalMinCoverage), true
	}
	return 0, false
}

func IsInvalidEvidenceClass(record proof.Record) (bool, string) {
	codes := append(extractReasonCodes(record.Metadata), extractReasonCodes(record.Event)...)
	for _, code := range codes {
		if _, ok := invalidEvidenceClasses[code]; ok {
			return true, code
		}
	}
	return false, ""
}

func extractReasonCodes(container map[string]any) []string {
	if container == nil {
		return nil
	}
	out := make([]string, 0)
	push := func(value string) {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized != "" {
			out = append(out, normalized)
		}
	}
	if value, ok := container["reason_code"].(string); ok {
		push(value)
	}
	if value, ok := container["reason"].(string); ok {
		push(value)
	}
	if values, ok := container["reason_codes"].([]any); ok {
		for _, value := range values {
			if code, ok := value.(string); ok {
				push(code)
			}
		}
	}
	if values, ok := container["reason_codes"].([]string); ok {
		for _, value := range values {
			push(value)
		}
	}
	sort.Strings(out)
	result := make([]string, 0, len(out))
	for i, code := range out {
		if i == 0 || code != out[i-1] {
			result = append(result, code)
		}
	}
	return result
}

func normalizeFrameworkThresholds(in map[string]float64) map[string]float64 {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]float64, len(in))
	for frameworkID, value := range in {
		trimmed := strings.ToLower(strings.TrimSpace(frameworkID))
		if trimmed == "" {
			continue
		}
		out[trimmed] = clamp01(value)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func clamp01(in float64) float64 {
	if in < 0 {
		return 0
	}
	if in > 1 {
		return 1
	}
	return in
}
