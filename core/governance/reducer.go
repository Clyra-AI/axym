package governance

import (
	"fmt"
	"sort"
	"time"
)

type Event struct {
	ID           string `json:"id"`
	ContractRef  Ref    `json:"contract_ref"`
	Kind         string `json:"kind"`
	OccurredAt   string `json:"occurred_at"`
	Status       string `json:"status"`
	SourceDigest string `json:"source_digest"`
}
type State struct {
	ContractID    string   `json:"contract_id"`
	Status        string   `json:"status"`
	Events        []string `json:"events"`
	SourceDigests []string `json:"source_digests"`
	Complete      bool     `json:"complete"`
	ReasonCodes   []string `json:"reason_codes,omitempty"`
}

// Reduce sorts a copy and applies a small, explicit lifecycle state machine.
// Equal timestamps are broken by event ID, making replay deterministic.
func Reduce(contractID string, events []Event) (State, error) {
	return ReduceChecked(contractID, events)
}

// ReduceVerified is the production reducer boundary. Every event must carry
// a complete digest-bound contract reference and a non-empty source digest.
func ReduceVerified(contractID string, events []Event) (State, error) {
	for _, event := range events {
		if !validRef(event.ContractRef) || event.ContractRef.ID != contractID || !digestPattern.MatchString(event.SourceDigest) {
			return State{ContractID: contractID, Status: "unverifiable", ReasonCodes: []string{"EVIDENCE_DIGEST_OR_REFERENCE_MISSING"}}, fmt.Errorf("%s: digest-bound evidence required", ReasonTampered)
		}
	}
	return ReduceChecked(contractID, events)
}

func ReduceChecked(contractID string, events []Event) (State, error) {
	in := append([]Event(nil), events...)
	sort.SliceStable(in, func(i, j int) bool {
		return eventBefore(in[i], in[j])
	})
	s := State{ContractID: contractID, Status: "unknown", Events: []string{}, SourceDigests: []string{}}
	seen := map[string]bool{}
	seenDigests := map[string]bool{}
	allowed := map[string]map[string]bool{
		"unknown": {"proposed": true, "registered": true}, "proposed": {"approved": true, "activated": true}, "active": {"execution_started": true, "execution_blocked": true, "stop_requested": true, "revocation_requested": true}, "executing": {"execution_succeeded": true, "execution_failed": true, "execution_blocked": true, "effect_recorded": true, "effect_validated": true, "effect_rejected": true, "effect_unresolved": true, "stop_requested": true, "revocation_requested": true}, "failed": {"compensation_required": true, "compensation_not_required": true, "compensation_started": true, "compensation_failed": true, "compensation_unresolved": true, "compensated": true, "contained": true, "containment_partial": true, "containment_unresolved": true}, "succeeded": {"effect_recorded": true, "effect_validated": true, "effect_rejected": true, "effect_unresolved": true, "compensation_required": true, "compensation_not_required": true, "compensation_started": true, "compensation_failed": true, "compensation_unresolved": true, "compensated": true, "contained": true, "containment_requested": true, "containment_partial": true, "containment_unresolved": true}, "effect_recorded": {"effect_validated": true, "effect_rejected": true, "effect_unresolved": true, "compensation_required": true, "compensation_not_required": true, "compensation_started": true, "compensation_failed": true, "compensation_unresolved": true, "compensated": true}, "compensation_required": {"effect_recorded": true, "effect_validated": true, "effect_rejected": true, "effect_unresolved": true, "compensation_started": true, "compensation_failed": true, "compensation_unresolved": true, "compensated": true, "containment_requested": true, "containment_partial": true, "containment_unresolved": true, "contained": true}, "compensating": {"effect_recorded": true, "effect_validated": true, "effect_rejected": true, "effect_unresolved": true, "compensated": true, "compensation_failed": true, "compensation_unresolved": true, "contained": true, "containment_requested": true, "containment_partial": true, "containment_unresolved": true}, "containing": {"containment_partial": true, "containment_unresolved": true, "contained": true, "compensation_required": true, "compensation_not_required": true, "compensation_started": true, "compensation_failed": true, "compensation_unresolved": true, "compensated": true}, "blocked": {"compensation_required": true, "compensation_not_required": true, "compensation_started": true, "compensation_failed": true, "compensation_unresolved": true, "compensated": true, "contained": true}, "partial": {"compensation_required": true, "compensation_not_required": true, "compensation_started": true, "compensation_failed": true, "compensation_unresolved": true, "compensated": true}, "gap": {"compensation_required": true, "compensation_not_required": true, "compensation_started": true, "compensation_failed": true, "compensation_unresolved": true, "compensated": true}, "compensated": {"containment_requested": true, "containment_partial": true, "containment_unresolved": true, "contained": true}, "stopping": {"stop_acknowledged": true, "contained": true}, "revoking": {"revocation_acknowledged": true, "contained": true}, "contained": {"compensation_required": true, "compensation_not_required": true, "compensation_started": true, "compensated": true},
	}
	// Effects may arrive after a completed compensation action. Keep this
	// explicit so valid producer history is not rejected by replay ordering.
	for _, kind := range []string{"effect_recorded", "effect_validated", "effect_rejected", "effect_unresolved"} {
		allowed["compensated"][kind] = true
	}
	// A contract may be executed repeatedly. Once a run is terminal, a new
	// activation starts a fresh run while retaining the prior event history.
	for _, status := range []string{"succeeded", "failed", "blocked", "compensated", "contained"} {
		allowed[status]["activated"] = true
	}
	for _, e := range in {
		if (e.ContractRef.Digest != "" || e.ContractRef.SchemaID != "" || e.SourceDigest != "") && (!validRef(e.ContractRef) || e.ContractRef.ID != contractID || !digestPattern.MatchString(e.SourceDigest)) {
			return s, fmt.Errorf("%s: digest-bound evidence required", ReasonTampered)
		}
		if e.ContractRef.ID != contractID {
			return s, fmt.Errorf("%s: foreign event %s", ReasonOutOfScope, e.ID)
		}
		if e.ID == "" {
			return s, fmt.Errorf("%s: event id required", ReasonMalformed)
		}
		if seen[e.ID] {
			return s, fmt.Errorf("%s: duplicate event %s", ReasonTampered, e.ID)
		}
		if _, err := time.Parse(time.RFC3339Nano, e.OccurredAt); err != nil {
			return s, fmt.Errorf("%s: event timestamp %s", ReasonMalformed, e.ID)
		}
		if e.SourceDigest != "" && !digestPattern.MatchString(e.SourceDigest) {
			return s, fmt.Errorf("%s: event digest %s", ReasonTampered, e.ID)
		}
		if e.SourceDigest != "" && seenDigests[e.SourceDigest] {
			return s, fmt.Errorf("%s: duplicate digest", ReasonTampered)
		}
		if e.Kind == "" {
			return s, fmt.Errorf("%s: event kind %s", ReasonMalformed, e.ID)
		}
		switch e.Kind {
		case "proposed", "registered", "approved", "activated", "execution_started", "execution_succeeded", "execution_failed", "execution_blocked", "effect_recorded", "effect_validated", "effect_rejected", "effect_unresolved", "containment_requested", "containment_partial", "containment_unresolved", "compensation_required", "compensation_not_required", "compensation_started", "compensation_failed", "compensation_unresolved", "compensated", "stop_requested", "stop_acknowledged", "revocation_requested", "revocation_acknowledged", "contained":
		default:
			return s, fmt.Errorf("%s: unknown lifecycle kind %s", ReasonMalformed, e.Kind)
		}
		if !allowed[s.Status][e.Kind] && (s.Status != "unknown" || e.Kind != "registered") {
			return s, fmt.Errorf("%s: illegal transition %s -> %s", ReasonMalformed, s.Status, e.Kind)
		}
		seen[e.ID] = true
		s.Events = append(s.Events, e.ID)
		if e.SourceDigest != "" {
			s.SourceDigests = append(s.SourceDigests, e.SourceDigest)
			seenDigests[e.SourceDigest] = true
		}
		switch e.Kind {
		case "proposed", "registered":
			s.Status = "proposed"
		case "approved", "activated":
			s.Status = "active"
		case "execution_started":
			s.Status = "executing"
		case "execution_succeeded", "effect_validated":
			s.Status = "succeeded"
		case "execution_failed", "effect_rejected":
			s.Status = "failed"
		case "execution_blocked":
			s.Status = "blocked"
			s.ReasonCodes = append(s.ReasonCodes, "GAP_EXECUTION_BLOCKED")
		case "effect_unresolved":
			s.Status = "gap"
			s.ReasonCodes = append(s.ReasonCodes, "GAP_EFFECT")
		case "effect_recorded":
			s.Status = "effect_recorded"
		case "compensation_required":
			s.Status = "compensation_required"
			s.ReasonCodes = append(s.ReasonCodes, "REQUIRED_COMPENSATION_MISSING")
		case "compensation_not_required":
			// Preserve the execution outcome. This disposition is not a
			// claim that compensation occurred.
		case "compensation_started":
			s.Status = "compensating"
		case "compensation_failed", "compensation_unresolved":
			s.Status = "gap"
			s.ReasonCodes = append(s.ReasonCodes, "GAP_COMPENSATION")
		case "contained":
			if e.Status == "unresolved" || e.Status == "unknown" || e.Status == "gap" {
				s.Status = "gap"
				s.ReasonCodes = append(s.ReasonCodes, "GAP_CONTAINMENT")
			} else {
				s.Status = "contained"
			}
		case "containment_requested":
			s.Status = "containing"
		case "containment_partial":
			s.Status = "partial"
			s.ReasonCodes = append(s.ReasonCodes, "GAP_CONTAINMENT_PARTIAL")
		case "containment_unresolved":
			s.Status = "gap"
			s.ReasonCodes = append(s.ReasonCodes, "GAP_CONTAINMENT")
		case "compensated":
			s.Status = "compensated"
			// A verified completion discharges the earlier requirement. Preserve
			// unrelated lifecycle gaps, but do not report compensation missing.
			s.ReasonCodes = removeReason(s.ReasonCodes, "REQUIRED_COMPENSATION_MISSING")
		case "stop_requested", "revocation_requested":
			s.Status = "stopping"
		case "stop_acknowledged", "revocation_acknowledged":
			s.Status = "contained"
		}
	}
	s.Complete = (s.Status == "succeeded" || s.Status == "failed" || s.Status == "contained" || s.Status == "compensated") && !hasBlockingGap(s.ReasonCodes)
	return s, nil
}

// eventBefore compares RFC3339 instants rather than their textual offsets.
// Keep the original timestamp on Event for producer provenance; only replay
// ordering needs the normalized instant.
func eventBefore(left, right Event) bool {
	lt, lerr := time.Parse(time.RFC3339Nano, left.OccurredAt)
	rt, rerr := time.Parse(time.RFC3339Nano, right.OccurredAt)
	if lerr == nil && rerr == nil {
		if lt.Equal(rt) {
			return left.ID < right.ID
		}
		return lt.Before(rt)
	}
	if left.OccurredAt == right.OccurredAt {
		return left.ID < right.ID
	}
	return left.OccurredAt < right.OccurredAt
}

func removeReason(reasons []string, target string) []string {
	filtered := reasons[:0]
	for _, reason := range reasons {
		if reason != target {
			filtered = append(filtered, reason)
		}
	}
	return filtered
}

func hasBlockingGap(reasons []string) bool {
	for _, reason := range reasons {
		switch reason {
		case "GAP_CONTAINMENT", "GAP_CONTAINMENT_PARTIAL", "GAP_EFFECT", "GAP_EXECUTION_BLOCKED", "GAP_COMPENSATION":
			return true
		}
	}
	return false
}
