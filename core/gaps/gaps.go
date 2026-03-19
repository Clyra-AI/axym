package gaps

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Clyra-AI/axym/core/compliance/coverage"
	"github.com/Clyra-AI/axym/core/compliance/match"
	"github.com/Clyra-AI/axym/core/review/grade"
)

type Report struct {
	Gaps    []Item       `json:"gaps"`
	Summary Summary      `json:"summary"`
	Grade   grade.Result `json:"grade"`
}

type Summary struct {
	Total           int `json:"total"`
	GapCount        int `json:"gap_count"`
	PartialCount    int `json:"partial_count"`
	CoveredCount    int `json:"covered_count"`
	HighestPriority int `json:"highest_priority"`
	LowestPriority  int `json:"lowest_priority"`
}

type Item struct {
	Rank               int      `json:"rank"`
	FrameworkID        string   `json:"framework_id"`
	ControlID          string   `json:"control_id"`
	Title              string   `json:"title"`
	Status             string   `json:"status"`
	Priority           int      `json:"priority"`
	ReasonCodes        []string `json:"reason_codes"`
	MissingRecordTypes []string `json:"missing_record_types,omitempty"`
	MissingFields      []string `json:"missing_fields,omitempty"`
	Remediation        string   `json:"remediation"`
	Effort             string   `json:"effort"`
	EvidenceCount      int      `json:"evidence_count"`
	InvalidExcluded    int      `json:"invalid_excluded"`
}

func Build(coverageReport coverage.Report) Report {
	items := make([]Item, 0)
	summary := Summary{
		Total:        coverageReport.Summary.ControlCount,
		GapCount:     coverageReport.Summary.GapCount,
		PartialCount: coverageReport.Summary.PartialCount,
		CoveredCount: coverageReport.Summary.CoveredCount,
	}

	for _, frameworkCoverage := range coverageReport.Frameworks {
		for _, control := range frameworkCoverage.Controls {
			if control.Status == "covered" {
				continue
			}
			item := Item{
				FrameworkID:        frameworkCoverage.FrameworkID,
				ControlID:          control.ControlID,
				Title:              control.Title,
				Status:             control.Status,
				ReasonCodes:        append([]string(nil), control.ReasonCodes...),
				MissingRecordTypes: append([]string(nil), control.MissingRecordTypes...),
				MissingFields:      append([]string(nil), control.MissingFields...),
				EvidenceCount:      control.EvidenceCount,
				InvalidExcluded:    control.InvalidExcluded,
			}
			item.Priority = priority(item)
			item.Remediation, item.Effort = remediation(item)
			items = append(items, item)
		}
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].Priority != items[j].Priority {
			return items[i].Priority > items[j].Priority
		}
		if items[i].FrameworkID != items[j].FrameworkID {
			return items[i].FrameworkID < items[j].FrameworkID
		}
		return items[i].ControlID < items[j].ControlID
	})

	for i := range items {
		items[i].Rank = i + 1
	}
	if len(items) > 0 {
		summary.HighestPriority = items[0].Priority
		summary.LowestPriority = items[len(items)-1].Priority
	}

	return Report{
		Gaps:    items,
		Summary: summary,
		Grade:   grade.Derive(coverageReport),
	}
}

func Explain(report Report) []string {
	lines := make([]string, 0, len(report.Gaps)+1)
	lines = append(lines, fmt.Sprintf("grade=%s score=%.4f gaps=%d partial=%d covered=%d", report.Grade.Letter, report.Grade.Score, report.Summary.GapCount, report.Summary.PartialCount, report.Summary.CoveredCount))
	for _, item := range report.Gaps {
		lines = append(lines, fmt.Sprintf("%d. %s/%s status=%s priority=%d effort=%s remediation=%s", item.Rank, item.FrameworkID, item.ControlID, item.Status, item.Priority, item.Effort, item.Remediation))
	}
	return lines
}

func priority(item Item) int {
	severity := 10
	if item.Status == "gap" {
		severity = 20
	}
	return severity + len(item.MissingRecordTypes)*4 + len(item.MissingFields)*2 + item.InvalidExcluded*3 + identityPriority(item.ReasonCodes)
}

func remediation(item Item) (string, string) {
	missingTypes := strings.Join(item.MissingRecordTypes, ",")
	missingFields := strings.Join(item.MissingFields, ",")
	switch {
	case hasIdentityWeakness(item.ReasonCodes):
		return "Link governed actions to actor, executor, owner, delegation, policy, and approval evidence", effortFor(item, true)
	case missingTypes != "" && missingFields != "":
		return fmt.Sprintf("Collect evidence types [%s] and populate fields [%s]", missingTypes, missingFields), effortFor(item, true)
	case missingTypes != "":
		return fmt.Sprintf("Collect evidence types [%s]", missingTypes), effortFor(item, false)
	case missingFields != "":
		return fmt.Sprintf("Populate required fields [%s]", missingFields), effortFor(item, false)
	default:
		return "Increase control evidence frequency and validate schema completeness", effortFor(item, false)
	}
}

func identityPriority(reasons []string) int {
	total := 0
	for _, reason := range reasons {
		switch reason {
		case match.ReasonMissingActorLinkage,
			match.ReasonMissingDownstreamLinkage,
			match.ReasonMissingOwnerLinkage,
			match.ReasonMissingTargetLinkage,
			match.ReasonMissingPolicyBinding,
			match.ReasonMissingApprovalBinding,
			match.ReasonIncompleteDelegation,
			match.ReasonUnapprovedPrivilegeDrift:
			total += 5
		}
	}
	return total
}

func hasIdentityWeakness(reasons []string) bool {
	return identityPriority(reasons) > 0
}

func effortFor(item Item, typesAndFields bool) string {
	if item.Status == "gap" && (typesAndFields || len(item.MissingRecordTypes) >= 2 || len(item.MissingFields) >= 3) {
		return "high"
	}
	if item.Status == "gap" || item.Status == "partial" {
		return "medium"
	}
	return "low"
}
