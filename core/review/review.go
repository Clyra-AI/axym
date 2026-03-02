package review

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Clyra-AI/axym/core/collect/snowflake"
	"github.com/Clyra-AI/axym/core/ingest/stitch"
	"github.com/Clyra-AI/axym/core/policy/freeze"
	"github.com/Clyra-AI/axym/core/policy/sod"
	"github.com/Clyra-AI/axym/core/store"
	"github.com/Clyra-AI/proof"
)

const (
	ExceptionSoD             = "sod"
	ExceptionApprovals       = "approvals"
	ExceptionEnrichment      = "enrichment"
	ExceptionAttach          = "attach"
	ExceptionReplay          = "replay"
	ExceptionFreeze          = "freeze"
	ExceptionChainSessionGap = "chain-session-gap"

	gradeHigh    = "high"
	gradeMedium  = "medium"
	gradeLow     = "low"
	gradeUnknown = "unknown"
)

var exceptionOrder = []string{
	ExceptionSoD,
	ExceptionApprovals,
	ExceptionEnrichment,
	ExceptionAttach,
	ExceptionReplay,
	ExceptionFreeze,
	ExceptionChainSessionGap,
}

var gradeOrder = []string{gradeHigh, gradeMedium, gradeLow, gradeUnknown}
var replayTierOrder = []string{"A", "B", "C", "unknown"}
var attachStatusOrder = []string{"attached", "retry", "dlq", "failed", "unknown"}
var attachSLAOrder = []string{"within_sla", "breached_sla", "unknown"}

type Request struct {
	StoreDir string
	Date     time.Time
}

type Pack struct {
	Date                   string         `json:"date"`
	Empty                  bool           `json:"empty"`
	RecordCount            int            `json:"record_count"`
	Exceptions             []Count        `json:"exceptions"`
	GradeDistribution      []Count        `json:"grade_distribution"`
	ReplayTierDistribution []Count        `json:"replay_tier_distribution"`
	AttachStatus           []Count        `json:"attach_status"`
	AttachSLA              []Count        `json:"attach_sla"`
	DegradationFlags       []string       `json:"degradation_flags"`
	Records                []RecordReview `json:"records"`
}

type Count struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type RecordReview struct {
	RecordID         string   `json:"record_id"`
	RecordType       string   `json:"record_type"`
	Timestamp        string   `json:"timestamp"`
	Auditability     string   `json:"auditability"`
	ExceptionClasses []string `json:"exception_classes"`
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

func Build(req Request) (Pack, error) {
	storeDir := strings.TrimSpace(req.StoreDir)
	if storeDir == "" {
		storeDir = ".axym"
	}

	evidenceStore, err := store.New(store.Config{RootDir: storeDir})
	if err != nil {
		return Pack{}, &Error{ReasonCode: "REVIEW_STORE_UNAVAILABLE", Message: "initialize local store", Err: err}
	}
	chain, err := evidenceStore.LoadChain()
	if err != nil {
		return Pack{}, &Error{ReasonCode: "REVIEW_CHAIN_READ_FAILED", Message: "load chain", Err: err}
	}

	day := req.Date.UTC()
	if day.IsZero() {
		day = time.Now().UTC()
	}
	return BuildFromRecords(chain.Records, day), nil
}

func BuildFromRecords(records []proof.Record, day time.Time) Pack {
	dayStart := time.Date(day.UTC().Year(), day.UTC().Month(), day.UTC().Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)

	target := make([]proof.Record, 0)
	for _, record := range records {
		ts := record.Timestamp.UTC()
		if (ts.Equal(dayStart) || ts.After(dayStart)) && ts.Before(dayEnd) {
			target = append(target, record)
		}
	}

	sort.Slice(target, func(i, j int) bool {
		ti := target[i].Timestamp.UTC()
		tj := target[j].Timestamp.UTC()
		if !ti.Equal(tj) {
			return ti.Before(tj)
		}
		if target[i].RecordID != target[j].RecordID {
			return target[i].RecordID < target[j].RecordID
		}
		return target[i].Integrity.RecordHash < target[j].Integrity.RecordHash
	})

	exceptionCounts := initializeCounts(exceptionOrder)
	gradeCounts := initializeCounts(gradeOrder)
	replayCounts := initializeCounts(replayTierOrder)
	attachStatusCounts := initializeCounts(attachStatusOrder)
	attachSLACounts := initializeCounts(attachSLAOrder)

	recordRows := make([]RecordReview, 0, len(target))
	var hasAttach bool
	var hasReplay bool
	var hasEnrichment bool

	for _, record := range target {
		recordClasses := classifyExceptions(record)
		for _, class := range recordClasses {
			exceptionCounts[class]++
		}

		auditability := deriveAuditability(record, recordClasses)
		if _, ok := gradeCounts[auditability]; !ok {
			auditability = gradeUnknown
		}
		gradeCounts[auditability]++

		if record.RecordType == "replay_certification" {
			hasReplay = true
			tier := strings.ToUpper(strings.TrimSpace(stringFromMap(record.Event, "tier")))
			if _, ok := replayCounts[tier]; !ok {
				tier = "unknown"
			}
			replayCounts[tier]++
		}

		if isTicketAttachmentRecord(record) {
			hasAttach = true
			status := strings.ToLower(strings.TrimSpace(stringFromMap(record.Event, "status")))
			if _, ok := attachStatusCounts[status]; !ok {
				status = "unknown"
			}
			attachStatusCounts[status]++

			slaWithin, ok := boolFromRecord(record, "sla_within")
			slaKey := "unknown"
			if ok {
				if slaWithin {
					slaKey = "within_sla"
				} else {
					slaKey = "breached_sla"
				}
			}
			attachSLACounts[slaKey]++
		}

		if record.RecordType == "data_pipeline_run" {
			hasEnrichment = true
		}

		recordRows = append(recordRows, RecordReview{
			RecordID:         record.RecordID,
			RecordType:       record.RecordType,
			Timestamp:        record.Timestamp.UTC().Format(time.RFC3339),
			Auditability:     auditability,
			ExceptionClasses: recordClasses,
		})
	}

	degradationFlags := []string{}
	if len(target) > 0 {
		if !hasEnrichment {
			degradationFlags = append(degradationFlags, "missing_enrichment_feed")
		}
		if !hasAttach {
			degradationFlags = append(degradationFlags, "missing_attach_feed")
		}
		if !hasReplay {
			degradationFlags = append(degradationFlags, "missing_replay_feed")
		}
	}
	sort.Strings(degradationFlags)

	return Pack{
		Date:                   dayStart.Format("2006-01-02"),
		Empty:                  len(target) == 0,
		RecordCount:            len(target),
		Exceptions:             toOrderedCounts(exceptionOrder, exceptionCounts),
		GradeDistribution:      toOrderedCounts(gradeOrder, gradeCounts),
		ReplayTierDistribution: toOrderedCounts(replayTierOrder, replayCounts),
		AttachStatus:           toOrderedCounts(attachStatusOrder, attachStatusCounts),
		AttachSLA:              toOrderedCounts(attachSLAOrder, attachSLACounts),
		DegradationFlags:       degradationFlags,
		Records:                recordRows,
	}
}

func classifyExceptions(record proof.Record) []string {
	classSet := map[string]struct{}{}
	for _, reason := range reasonCodesFromRecord(record) {
		switch reason {
		case sod.ReasonMissingActor, sod.ReasonRequestorDeployer:
			classSet[ExceptionSoD] = struct{}{}
		case freeze.ReasonFreezeWindow:
			classSet[ExceptionFreeze] = struct{}{}
		case snowflake.ReasonEnrichmentLag, snowflake.ReasonMissingTag:
			classSet[ExceptionEnrichment] = struct{}{}
		case stitch.ReasonChainSessionGap:
			classSet[ExceptionChainSessionGap] = struct{}{}
		case "TICKET_RATE_LIMITED", "TICKET_REMOTE_ERROR", "TICKET_DLQ", "TICKET_REJECTED":
			classSet[ExceptionAttach] = struct{}{}
		case "REPLAY_FAILED", "REPLAY_MISMATCH":
			classSet[ExceptionReplay] = struct{}{}
		}
	}

	if record.RecordType == "approval" {
		decision := strings.ToLower(strings.TrimSpace(stringFromMap(record.Event, "decision")))
		approved, hasApproved := boolFromMap(record.Event, "approved")
		if decision != "" && decision != "allow" {
			classSet[ExceptionApprovals] = struct{}{}
		}
		if hasApproved && !approved {
			classSet[ExceptionApprovals] = struct{}{}
		}
	}

	if isTicketAttachmentRecord(record) {
		status := strings.ToLower(strings.TrimSpace(stringFromMap(record.Event, "status")))
		if status != "" && status != "attached" {
			classSet[ExceptionAttach] = struct{}{}
		}
	}

	if record.RecordType == "replay_certification" {
		if pass, ok := boolFromMap(record.Event, "pass"); ok && !pass {
			classSet[ExceptionReplay] = struct{}{}
		}
		status := strings.ToLower(strings.TrimSpace(stringFromMap(record.Event, "status")))
		if status != "" && status != "certified" {
			classSet[ExceptionReplay] = struct{}{}
		}
	}

	out := make([]string, 0, len(classSet))
	for _, class := range exceptionOrder {
		if _, ok := classSet[class]; ok {
			out = append(out, class)
		}
	}
	return out
}

func deriveAuditability(record proof.Record, classes []string) string {
	for _, field := range []string{"auditability", "auditability_grade"} {
		if v := strings.ToLower(strings.TrimSpace(stringFromMap(record.Metadata, field))); v != "" {
			if v == gradeHigh || v == gradeMedium || v == gradeLow {
				return v
			}
		}
		if v := strings.ToLower(strings.TrimSpace(stringFromMap(record.Event, field))); v != "" {
			if v == gradeHigh || v == gradeMedium || v == gradeLow {
				return v
			}
		}
	}
	if len(classes) == 0 {
		return gradeHigh
	}
	for _, class := range classes {
		if class == ExceptionReplay || class == ExceptionAttach || class == ExceptionChainSessionGap {
			return gradeLow
		}
	}
	return gradeMedium
}

func initializeCounts(keys []string) map[string]int {
	out := make(map[string]int, len(keys))
	for _, key := range keys {
		out[key] = 0
	}
	return out
}

func toOrderedCounts(order []string, counts map[string]int) []Count {
	out := make([]Count, 0, len(order))
	for _, key := range order {
		out = append(out, Count{Name: key, Count: counts[key]})
	}
	return out
}

func reasonCodesFromRecord(record proof.Record) []string {
	set := map[string]struct{}{}
	addReasonCodes(set, record.Event["reason_codes"])
	addReasonCodes(set, record.Metadata["reason_codes"])
	out := make([]string, 0, len(set))
	for reason := range set {
		out = append(out, reason)
	}
	sort.Strings(out)
	return out
}

func addReasonCodes(set map[string]struct{}, value any) {
	switch typed := value.(type) {
	case []string:
		for _, reason := range typed {
			trimmed := strings.TrimSpace(reason)
			if trimmed != "" {
				set[trimmed] = struct{}{}
			}
		}
	case []any:
		for _, raw := range typed {
			if str, ok := raw.(string); ok {
				trimmed := strings.TrimSpace(str)
				if trimmed != "" {
					set[trimmed] = struct{}{}
				}
			}
		}
	}
}

func stringFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	raw, ok := m[key]
	if !ok {
		return ""
	}
	value, ok := raw.(string)
	if !ok {
		return ""
	}
	return value
}

func boolFromMap(m map[string]any, key string) (bool, bool) {
	if m == nil {
		return false, false
	}
	raw, ok := m[key]
	if !ok {
		return false, false
	}
	value, ok := raw.(bool)
	if !ok {
		return false, false
	}
	return value, true
}

func boolFromRecord(record proof.Record, key string) (bool, bool) {
	if value, ok := boolFromMap(record.Event, key); ok {
		return value, true
	}
	if value, ok := boolFromMap(record.Metadata, key); ok {
		return value, true
	}
	return false, false
}

func isTicketAttachmentRecord(record proof.Record) bool {
	if record.RecordType == "ticket_attachment" {
		return true
	}
	kind := strings.ToLower(strings.TrimSpace(stringFromMap(record.Event, "kind")))
	category := strings.ToLower(strings.TrimSpace(stringFromMap(record.Event, "category")))
	return kind == "ticket_attachment" || category == "ticket_attachment"
}
