package identitygovernance

import (
	"encoding/json"
	"sort"
	"strings"

	"github.com/Clyra-AI/axym/core/compliance/match"
	"github.com/Clyra-AI/axym/core/normalize"
	"github.com/Clyra-AI/proof"
)

type Digest struct {
	RecordCount                  int `json:"record_count"`
	WeakRecordCount              int `json:"weak_record_count"`
	OwnerCount                   int `json:"owner_count"`
	PrivilegeDriftFindingCount   int `json:"privilege_drift_finding_count"`
	DelegatedChainExceptionCount int `json:"delegated_chain_exception_count"`
}

type RecordSummary struct {
	RecordID             string   `json:"record_id"`
	RecordType           string   `json:"record_type"`
	ActorIdentity        string   `json:"actor_identity,omitempty"`
	DownstreamIdentity   string   `json:"downstream_identity,omitempty"`
	OwnerIdentity        string   `json:"owner_identity,omitempty"`
	PolicyDigest         string   `json:"policy_digest,omitempty"`
	ApprovalTokenRef     string   `json:"approval_token_ref,omitempty"`
	TargetKind           string   `json:"target_kind,omitempty"`
	TargetID             string   `json:"target_id,omitempty"`
	DelegationChainDepth int      `json:"delegation_chain_depth"`
	WeakReasonCodes      []string `json:"weak_reason_codes,omitempty"`
}

type ChainSummary struct {
	Version string          `json:"version"`
	Digest  Digest          `json:"digest"`
	Records []RecordSummary `json:"records"`
}

type OwnershipEntry struct {
	OwnerIdentity   string   `json:"owner_identity"`
	RecordCount     int      `json:"record_count"`
	RecordIDs       []string `json:"record_ids"`
	ApprovalRefs    []string `json:"approval_refs,omitempty"`
	DownstreamUsers []string `json:"downstream_identities,omitempty"`
}

type OwnershipRegister struct {
	Version string           `json:"version"`
	Entries []OwnershipEntry `json:"entries"`
}

type PrivilegeDriftFinding struct {
	RecordID           string `json:"record_id"`
	ActorIdentity      string `json:"actor_identity,omitempty"`
	DownstreamIdentity string `json:"downstream_identity,omitempty"`
	Privilege          string `json:"privilege"`
	Approved           bool   `json:"approved"`
	ApprovalTokenRef   string `json:"approval_token_ref,omitempty"`
	Status             string `json:"status"`
}

type PrivilegeDriftReport struct {
	Version  string                  `json:"version"`
	Findings []PrivilegeDriftFinding `json:"findings"`
}

type DelegatedChainException struct {
	RecordID             string   `json:"record_id"`
	RecordType           string   `json:"record_type"`
	ActorIdentity        string   `json:"actor_identity,omitempty"`
	DownstreamIdentity   string   `json:"downstream_identity,omitempty"`
	DelegationChainDepth int      `json:"delegation_chain_depth"`
	ReasonCodes          []string `json:"reason_codes"`
}

type DelegatedChainExceptions struct {
	Version    string                    `json:"version"`
	Exceptions []DelegatedChainException `json:"exceptions"`
}

type Artifacts struct {
	Digest                   Digest
	ChainSummary             ChainSummary
	OwnershipRegister        OwnershipRegister
	PrivilegeDriftReport     PrivilegeDriftReport
	DelegatedChainExceptions DelegatedChainExceptions
}

func Build(records []proof.Record) Artifacts {
	recordSummaries := make([]RecordSummary, 0, len(records))
	ownership := map[string]*OwnershipEntry{}
	privilegeFindings := make([]PrivilegeDriftFinding, 0)
	delegatedExceptions := make([]DelegatedChainException, 0)
	weakRecords := 0

	for _, record := range sortedRecords(records) {
		view := normalize.IdentityViewFromRecord(&record)
		_, weaknessReasons := match.IdentityWeaknesses(record)
		if len(weaknessReasons) > 0 {
			weakRecords++
		}

		recordSummaries = append(recordSummaries, RecordSummary{
			RecordID:             strings.TrimSpace(record.RecordID),
			RecordType:           strings.ToLower(strings.TrimSpace(record.RecordType)),
			ActorIdentity:        strings.TrimSpace(view.ActorIdentity),
			DownstreamIdentity:   strings.TrimSpace(view.DownstreamIdentity),
			OwnerIdentity:        strings.TrimSpace(view.OwnerIdentity),
			PolicyDigest:         strings.TrimSpace(view.PolicyDigest),
			ApprovalTokenRef:     strings.TrimSpace(view.ApprovalTokenRef),
			TargetKind:           strings.TrimSpace(view.TargetKind),
			TargetID:             strings.TrimSpace(view.TargetID),
			DelegationChainDepth: len(view.DelegationChain),
			WeakReasonCodes:      append([]string(nil), weaknessReasons...),
		})

		if owner := strings.TrimSpace(view.OwnerIdentity); owner != "" {
			entry := ownership[owner]
			if entry == nil {
				entry = &OwnershipEntry{OwnerIdentity: owner}
				ownership[owner] = entry
			}
			entry.RecordCount++
			entry.RecordIDs = append(entry.RecordIDs, strings.TrimSpace(record.RecordID))
			if ref := strings.TrimSpace(view.ApprovalTokenRef); ref != "" {
				entry.ApprovalRefs = append(entry.ApprovalRefs, ref)
			}
			if downstream := strings.TrimSpace(view.DownstreamIdentity); downstream != "" {
				entry.DownstreamUsers = append(entry.DownstreamUsers, downstream)
			}
		}

		if privilegeFinding, ok := privilegeDriftFinding(record, view); ok {
			privilegeFindings = append(privilegeFindings, privilegeFinding)
		}
		if delegatedChainException, ok := delegatedChainException(record, view, weaknessReasons); ok {
			delegatedExceptions = append(delegatedExceptions, delegatedChainException)
		}
	}

	ownershipEntries := make([]OwnershipEntry, 0, len(ownership))
	for _, owner := range sortedOwnershipKeys(ownership) {
		entry := ownership[owner]
		entry.RecordIDs = uniqueSorted(entry.RecordIDs)
		entry.ApprovalRefs = uniqueSorted(entry.ApprovalRefs)
		entry.DownstreamUsers = uniqueSorted(entry.DownstreamUsers)
		ownershipEntries = append(ownershipEntries, *entry)
	}

	sort.Slice(privilegeFindings, func(i, j int) bool {
		return privilegeFindings[i].RecordID < privilegeFindings[j].RecordID
	})
	sort.Slice(delegatedExceptions, func(i, j int) bool {
		return delegatedExceptions[i].RecordID < delegatedExceptions[j].RecordID
	})

	digest := Digest{
		RecordCount:                  len(recordSummaries),
		WeakRecordCount:              weakRecords,
		OwnerCount:                   len(ownershipEntries),
		PrivilegeDriftFindingCount:   len(privilegeFindings),
		DelegatedChainExceptionCount: len(delegatedExceptions),
	}

	return Artifacts{
		Digest: digest,
		ChainSummary: ChainSummary{
			Version: "v1",
			Digest:  digest,
			Records: recordSummaries,
		},
		OwnershipRegister: OwnershipRegister{
			Version: "v1",
			Entries: ownershipEntries,
		},
		PrivilegeDriftReport: PrivilegeDriftReport{
			Version:  "v1",
			Findings: privilegeFindings,
		},
		DelegatedChainExceptions: DelegatedChainExceptions{
			Version:    "v1",
			Exceptions: delegatedExceptions,
		},
	}
}

func MarshalIndent(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func privilegeDriftFinding(record proof.Record, view normalize.IdentityView) (PrivilegeDriftFinding, bool) {
	if strings.ToLower(strings.TrimSpace(record.RecordType)) != "scan_finding" {
		return PrivilegeDriftFinding{}, false
	}
	privilege := firstString(strings.TrimSpace(view.TargetID), stringFromMap(record.Event, "privilege"))
	if privilege == "" {
		return PrivilegeDriftFinding{}, false
	}
	approved, _ := boolFromMap(record.Event, "approved")
	if !approved {
		approved, _ = boolFromMap(record.Metadata, "approved")
	}
	status := "approved"
	if !approved {
		status = "unapproved"
	}
	return PrivilegeDriftFinding{
		RecordID:           strings.TrimSpace(record.RecordID),
		ActorIdentity:      strings.TrimSpace(view.ActorIdentity),
		DownstreamIdentity: strings.TrimSpace(view.DownstreamIdentity),
		Privilege:          privilege,
		Approved:           approved,
		ApprovalTokenRef:   strings.TrimSpace(view.ApprovalTokenRef),
		Status:             status,
	}, true
}

func delegatedChainException(record proof.Record, view normalize.IdentityView, reasons []string) (DelegatedChainException, bool) {
	relevant := make([]string, 0, len(reasons))
	for _, reason := range reasons {
		switch reason {
		case match.ReasonMissingActorLinkage,
			match.ReasonMissingDownstreamLinkage,
			match.ReasonIncompleteDelegation,
			match.ReasonMissingApprovalBinding,
			match.ReasonMissingOwnerLinkage,
			match.ReasonMissingPolicyBinding:
			relevant = append(relevant, reason)
		}
	}
	if len(relevant) == 0 {
		return DelegatedChainException{}, false
	}
	return DelegatedChainException{
		RecordID:             strings.TrimSpace(record.RecordID),
		RecordType:           strings.ToLower(strings.TrimSpace(record.RecordType)),
		ActorIdentity:        strings.TrimSpace(view.ActorIdentity),
		DownstreamIdentity:   strings.TrimSpace(view.DownstreamIdentity),
		DelegationChainDepth: len(view.DelegationChain),
		ReasonCodes:          uniqueSorted(relevant),
	}, true
}

func sortedRecords(records []proof.Record) []proof.Record {
	out := append([]proof.Record(nil), records...)
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Timestamp.Equal(out[j].Timestamp) {
			return out[i].Timestamp.Before(out[j].Timestamp)
		}
		if out[i].RecordID != out[j].RecordID {
			return out[i].RecordID < out[j].RecordID
		}
		return out[i].Integrity.RecordHash < out[j].Integrity.RecordHash
	})
	return out
}

func sortedOwnershipKeys(entries map[string]*OwnershipEntry) []string {
	out := make([]string, 0, len(entries))
	for key := range entries {
		out = append(out, key)
	}
	sort.Strings(out)
	return out
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

func stringFromMap(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	value, _ := m[key].(string)
	return strings.TrimSpace(value)
}

func boolFromMap(m map[string]any, key string) (bool, bool) {
	if m == nil {
		return false, false
	}
	value, ok := m[key].(bool)
	return value, ok
}

func firstString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
