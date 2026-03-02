package coverage

import (
	"sort"

	"github.com/Clyra-AI/axym/core/compliance/match"
)

type Report struct {
	Frameworks []FrameworkCoverage `json:"frameworks"`
	Summary    Summary             `json:"summary"`
}

type Summary struct {
	FrameworkCount int `json:"framework_count"`
	ControlCount   int `json:"control_count"`
	CoveredCount   int `json:"covered_count"`
	PartialCount   int `json:"partial_count"`
	GapCount       int `json:"gap_count"`
}

type FrameworkCoverage struct {
	FrameworkID    string            `json:"framework_id"`
	Coverage       float64           `json:"coverage"`
	CoveredCount   int               `json:"covered_count"`
	PartialCount   int               `json:"partial_count"`
	GapCount       int               `json:"gap_count"`
	Controls       []ControlCoverage `json:"controls"`
	FailingControl []string          `json:"failing_controls,omitempty"`
}

type ControlCoverage struct {
	FrameworkID        string   `json:"framework_id"`
	ControlID          string   `json:"control_id"`
	Title              string   `json:"title"`
	Status             string   `json:"status"`
	ReasonCodes        []string `json:"reason_codes"`
	EvidenceCount      int      `json:"evidence_count"`
	MissingRecordTypes []string `json:"missing_record_types,omitempty"`
	MissingFields      []string `json:"missing_fields,omitempty"`
	InvalidExcluded    int      `json:"invalid_excluded"`
}

func Build(result match.Result) Report {
	frameworks := make([]FrameworkCoverage, 0, len(result.Frameworks))
	summary := Summary{FrameworkCount: len(result.Frameworks)}

	for _, frameworkResult := range result.Frameworks {
		frameworkCoverage := FrameworkCoverage{
			FrameworkID: frameworkResult.ID,
			Controls:    make([]ControlCoverage, 0, len(frameworkResult.Controls)),
		}
		for _, control := range frameworkResult.Controls {
			cc := buildControlCoverage(control)
			frameworkCoverage.Controls = append(frameworkCoverage.Controls, cc)
			summary.ControlCount++
			switch cc.Status {
			case match.ControlStatusCovered:
				frameworkCoverage.CoveredCount++
				summary.CoveredCount++
			case match.ControlStatusPartial:
				frameworkCoverage.PartialCount++
				summary.PartialCount++
				frameworkCoverage.FailingControl = append(frameworkCoverage.FailingControl, cc.ControlID)
			default:
				frameworkCoverage.GapCount++
				summary.GapCount++
				frameworkCoverage.FailingControl = append(frameworkCoverage.FailingControl, cc.ControlID)
			}
		}
		if len(frameworkCoverage.Controls) > 0 {
			frameworkCoverage.Coverage = round(float64(frameworkCoverage.CoveredCount) / float64(len(frameworkCoverage.Controls)))
		}
		sort.Strings(frameworkCoverage.FailingControl)
		frameworks = append(frameworks, frameworkCoverage)
	}

	sort.Slice(frameworks, func(i, j int) bool {
		return frameworks[i].FrameworkID < frameworks[j].FrameworkID
	})
	return Report{Frameworks: frameworks, Summary: summary}
}

func buildControlCoverage(control match.ControlResult) ControlCoverage {
	recordTypes := make(map[string]struct{}, len(control.Evidence))
	missingFields := make(map[string]struct{})
	for _, evidence := range control.Evidence {
		recordTypes[evidence.RecordType] = struct{}{}
		for _, field := range evidence.Missing {
			missingFields[field] = struct{}{}
		}
	}
	missingTypes := make([]string, 0)
	for _, requiredType := range control.RequiredRecordTypes {
		if _, ok := recordTypes[requiredType]; ok {
			continue
		}
		missingTypes = append(missingTypes, requiredType)
	}
	sort.Strings(missingTypes)

	missing := make([]string, 0, len(missingFields))
	for field := range missingFields {
		missing = append(missing, field)
	}
	sort.Strings(missing)

	return ControlCoverage{
		FrameworkID:        control.FrameworkID,
		ControlID:          control.ControlID,
		Title:              control.Title,
		Status:             control.Status,
		ReasonCodes:        append([]string(nil), control.ReasonCodes...),
		EvidenceCount:      len(control.Evidence),
		MissingRecordTypes: missingTypes,
		MissingFields:      missing,
		InvalidExcluded:    control.InvalidExcluded,
	}
}

func round(in float64) float64 {
	if in <= 0 {
		return 0
	}
	if in >= 1 {
		return 1
	}
	return float64(int(in*10000+0.5)) / 10000
}
