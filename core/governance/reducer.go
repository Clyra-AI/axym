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

func ReduceChecked(contractID string, events []Event) (State, error) {
	in := append([]Event(nil), events...)
	sort.SliceStable(in, func(i, j int) bool {
		if in[i].OccurredAt == in[j].OccurredAt {
			return in[i].ID < in[j].ID
		}
		return in[i].OccurredAt < in[j].OccurredAt
	})
	s := State{ContractID: contractID, Status: "unknown", Events: []string{}, SourceDigests: []string{}}
	seen := map[string]bool{}
	seenDigests := map[string]bool{}
	allowed := map[string]map[string]bool{
		"unknown": {"proposed": true, "registered": true}, "proposed": {"approved": true, "activated": true}, "active": {"execution_started": true, "stop_requested": true, "revocation_requested": true},
		"executing": {"execution_succeeded": true, "execution_failed": true, "effect_validated": true, "effect_rejected": true, "stop_requested": true, "revocation_requested": true}, "failed": {"compensated": true, "contained": true}, "succeeded": {"effect_validated": true, "contained": true}, "stopping": {"stop_acknowledged": true, "contained": true}, "revoking": {"revocation_acknowledged": true, "contained": true}, "contained": {"compensated": true},
	}
	for _, e := range in {
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
		case "proposed", "registered", "approved", "activated", "execution_started", "execution_succeeded", "effect_validated", "execution_failed", "effect_rejected", "stop_requested", "stop_acknowledged", "revocation_requested", "revocation_acknowledged", "contained", "compensated":
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
		case "contained":
			s.Status = "contained"
		case "compensated":
			s.Status = "compensated"
		case "stop_requested", "revocation_requested":
			s.Status = "stopping"
		case "stop_acknowledged", "revocation_acknowledged":
			s.Status = "contained"
		}
	}
	s.Complete = s.Status == "succeeded" || s.Status == "failed" || s.Status == "contained" || s.Status == "compensated"
	return s, nil
}
