package regress

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Clyra-AI/axym/core/compliance/coverage"
	"github.com/Clyra-AI/axym/core/store"
	regressschema "github.com/Clyra-AI/axym/schemas/v1/regress"
)

const (
	BaselineVersion = "v1"
	DefaultFileName = "regress-baseline.json"

	ReasonInvalidInput   = "invalid_input"
	ReasonBaselineRead   = "REGRESS_BASELINE_READ_FAILED"
	ReasonBaselineWrite  = "REGRESS_BASELINE_WRITE_FAILED"
	ReasonBaselineSchema = "REGRESS_BASELINE_SCHEMA_FAILED"
)

type Error struct {
	ReasonCode string
	Message    string
	ExitCode   int
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

type Baseline struct {
	Version    string              `json:"version"`
	CapturedAt string              `json:"captured_at,omitempty"`
	Frameworks []FrameworkBaseline `json:"frameworks"`
}

type FrameworkBaseline struct {
	FrameworkID string            `json:"framework_id"`
	Coverage    float64           `json:"coverage"`
	Controls    []ControlBaseline `json:"controls"`
}

type ControlBaseline struct {
	ControlID   string   `json:"control_id"`
	Status      string   `json:"status"`
	ReasonCodes []string `json:"reason_codes,omitempty"`
}

type InitRequest struct {
	BaselinePath   string
	CoverageReport coverage.Report
	Now            time.Time
}

type InitResult struct {
	Path         string `json:"path"`
	Frameworks   int    `json:"frameworks"`
	ControlCount int    `json:"control_count"`
}

type RunRequest struct {
	BaselinePath   string
	CoverageReport coverage.Report
}

type Drift struct {
	FrameworkID         string   `json:"framework_id"`
	ControlID           string   `json:"control_id"`
	BaselineStatus      string   `json:"baseline_status"`
	CurrentStatus       string   `json:"current_status"`
	BaselineReasonCodes []string `json:"baseline_reason_codes,omitempty"`
	CurrentReasonCodes  []string `json:"current_reason_codes,omitempty"`
	Reason              string   `json:"reason"`
}

type RunResult struct {
	BaselinePath      string  `json:"baseline_path"`
	DriftDetected     bool    `json:"drift_detected"`
	RegressedControls []Drift `json:"regressed_controls,omitempty"`
}

func Init(req InitRequest) (InitResult, error) {
	path, err := resolveBaselinePath(req.BaselinePath)
	if err != nil {
		return InitResult{}, wrapInvalidInput("resolve baseline path", err)
	}
	baseline := buildBaseline(req.CoverageReport, req.Now)
	raw, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		return InitResult{}, &Error{ReasonCode: ReasonBaselineWrite, Message: "marshal baseline", ExitCode: 1, Err: err}
	}
	if err := regressschema.ValidateBaseline(raw); err != nil {
		return InitResult{}, &Error{ReasonCode: ReasonBaselineSchema, Message: "baseline schema validation failed", ExitCode: 6, Err: err}
	}
	if err := store.WriteJSONAtomic(path, raw, true); err != nil {
		return InitResult{}, &Error{ReasonCode: ReasonBaselineWrite, Message: "write baseline", ExitCode: 1, Err: err}
	}
	controlCount := 0
	for _, framework := range baseline.Frameworks {
		controlCount += len(framework.Controls)
	}
	return InitResult{Path: path, Frameworks: len(baseline.Frameworks), ControlCount: controlCount}, nil
}

func Run(req RunRequest) (RunResult, error) {
	path, err := resolveBaselinePath(req.BaselinePath)
	if err != nil {
		return RunResult{}, wrapInvalidInput("resolve baseline path", err)
	}
	baseline, err := readBaseline(path)
	if err != nil {
		return RunResult{}, err
	}

	current := buildBaseline(req.CoverageReport, time.Time{})
	drift := diff(baseline, current)
	return RunResult{
		BaselinePath:      path,
		DriftDetected:     len(drift) > 0,
		RegressedControls: drift,
	}, nil
}

func readBaseline(path string) (Baseline, error) {
	// #nosec G304 -- baseline path is explicit user input.
	raw, err := os.ReadFile(path)
	if err != nil {
		return Baseline{}, &Error{ReasonCode: ReasonBaselineRead, Message: "read baseline", ExitCode: 6, Err: err}
	}
	if err := regressschema.ValidateBaseline(raw); err != nil {
		return Baseline{}, &Error{ReasonCode: ReasonBaselineSchema, Message: "baseline schema validation failed", ExitCode: 6, Err: err}
	}
	var baseline Baseline
	if err := json.Unmarshal(raw, &baseline); err != nil {
		return Baseline{}, &Error{ReasonCode: ReasonBaselineRead, Message: "decode baseline", ExitCode: 6, Err: err}
	}
	return baseline, nil
}

func buildBaseline(report coverage.Report, now time.Time) Baseline {
	frameworks := make([]FrameworkBaseline, 0, len(report.Frameworks))
	for _, frameworkCoverage := range report.Frameworks {
		controls := make([]ControlBaseline, 0, len(frameworkCoverage.Controls))
		for _, control := range frameworkCoverage.Controls {
			controls = append(controls, ControlBaseline{
				ControlID:   control.ControlID,
				Status:      normalizeStatus(control.Status),
				ReasonCodes: uniqueSorted(control.ReasonCodes),
			})
		}
		sort.Slice(controls, func(i, j int) bool {
			return controls[i].ControlID < controls[j].ControlID
		})
		frameworks = append(frameworks, FrameworkBaseline{
			FrameworkID: strings.ToLower(strings.TrimSpace(frameworkCoverage.FrameworkID)),
			Coverage:    frameworkCoverage.Coverage,
			Controls:    controls,
		})
	}
	sort.Slice(frameworks, func(i, j int) bool {
		return frameworks[i].FrameworkID < frameworks[j].FrameworkID
	})
	baseline := Baseline{
		Version:    BaselineVersion,
		CapturedAt: now.UTC().Format(time.RFC3339),
		Frameworks: frameworks,
	}
	if now.IsZero() {
		baseline.CapturedAt = ""
	}
	return baseline
}

func diff(baseline Baseline, current Baseline) []Drift {
	currentFrameworks := make(map[string]FrameworkBaseline, len(current.Frameworks))
	for _, framework := range current.Frameworks {
		currentFrameworks[framework.FrameworkID] = framework
	}
	out := make([]Drift, 0)
	for _, framework := range baseline.Frameworks {
		currentFramework, ok := currentFrameworks[framework.FrameworkID]
		if !ok {
			for _, control := range framework.Controls {
				out = append(out, Drift{
					FrameworkID:         framework.FrameworkID,
					ControlID:           control.ControlID,
					BaselineStatus:      control.Status,
					CurrentStatus:       "gap",
					BaselineReasonCodes: append([]string(nil), control.ReasonCodes...),
					Reason:              "framework_missing_from_current",
				})
			}
			continue
		}
		currentControls := make(map[string]ControlBaseline, len(currentFramework.Controls))
		for _, control := range currentFramework.Controls {
			currentControls[control.ControlID] = control
		}
		for _, baselineControl := range framework.Controls {
			currentControl, ok := currentControls[baselineControl.ControlID]
			if !ok {
				out = append(out, Drift{
					FrameworkID:         framework.FrameworkID,
					ControlID:           baselineControl.ControlID,
					BaselineStatus:      baselineControl.Status,
					CurrentStatus:       "gap",
					BaselineReasonCodes: append([]string(nil), baselineControl.ReasonCodes...),
					Reason:              "control_missing_from_current",
				})
				continue
			}
			if statusRank(currentControl.Status) <= statusRank(baselineControl.Status) {
				continue
			}
			out = append(out, Drift{
				FrameworkID:         framework.FrameworkID,
				ControlID:           baselineControl.ControlID,
				BaselineStatus:      baselineControl.Status,
				CurrentStatus:       currentControl.Status,
				BaselineReasonCodes: append([]string(nil), baselineControl.ReasonCodes...),
				CurrentReasonCodes:  append([]string(nil), currentControl.ReasonCodes...),
				Reason:              "coverage_drift_detected",
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].FrameworkID != out[j].FrameworkID {
			return out[i].FrameworkID < out[j].FrameworkID
		}
		return out[i].ControlID < out[j].ControlID
	})
	return out
}

func resolveBaselinePath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", fmt.Errorf("baseline path is required")
	}
	info, err := os.Stat(trimmed)
	if err == nil {
		if info.IsDir() {
			return filepath.Join(trimmed, DefaultFileName), nil
		}
		return trimmed, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	if strings.HasSuffix(trimmed, string(os.PathSeparator)) {
		return filepath.Join(trimmed, DefaultFileName), nil
	}
	return trimmed, nil
}

func wrapInvalidInput(message string, err error) error {
	return &Error{ReasonCode: ReasonInvalidInput, Message: message, ExitCode: 6, Err: err}
}

func normalizeStatus(status string) string {
	value := strings.ToLower(strings.TrimSpace(status))
	switch value {
	case "covered", "partial", "gap":
		return value
	default:
		return "gap"
	}
}

func statusRank(status string) int {
	switch normalizeStatus(status) {
	case "covered":
		return 0
	case "partial":
		return 1
	default:
		return 2
	}
}

func uniqueSorted(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	sort.Strings(out)
	return out
}
