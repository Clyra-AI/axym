package governance

import "sort"

type CompletenessStatus string

const (
	Complete     CompletenessStatus = "complete"
	Partial      CompletenessStatus = "partial"
	Gap          CompletenessStatus = "gap"
	Unverifiable CompletenessStatus = "unverifiable"
	OutOfScope   CompletenessStatus = "out_of_scope"
)

type Confidence string

const (
	High   Confidence = "high"
	Medium Confidence = "medium"
	Low    Confidence = "low"
	None   Confidence = "none"
)

type CompletenessInput struct {
	Readiness                bool
	Preconditions            bool
	ProposalSeen             bool
	ActivationSeen           bool
	ExecutionOutcome         string
	EffectValidated          bool
	Containment              string
	CompensationRequired     bool
	Compensation             string
	AuthorityLineage         bool
	Fresh                    bool
	CorrelationRefs          int
	CorrelationAuthoritative bool
	JudgeOnly                bool
	OutOfScope               bool
}
type Completeness struct {
	Status                CompletenessStatus `json:"status"`
	ReasonCodes           []string           `json:"reason_codes"`
	EvidenceQuality       Confidence         `json:"evidence_quality"`
	EnforcementCoverage   Confidence         `json:"enforcement_coverage"`
	CorrelationConfidence Confidence         `json:"correlation_confidence"`
}

const (
	ReasonReadinessMissing     = "READINESS_MISSING"
	ReasonPreconditionsMissing = "PRECONDITIONS_MISSING"
	ReasonSequenceIncomplete   = "PROPOSAL_ACTIVATION_SEQUENCE_INCOMPLETE"
	ReasonExecutionMissing     = "EXECUTION_OUTCOME_MISSING"
	ReasonEffectMissing        = "EFFECT_VALIDATION_MISSING"
	ReasonContainmentMissing   = "CONTAINMENT_ACK_MISSING"
	ReasonCompensationMissing  = "COMPENSATION_REQUIRED"
	ReasonAuthorityMissing     = "AUTHORITY_LINEAGE_UNVERIFIABLE"
	ReasonStaleEvidence        = "EVIDENCE_STALE"
	ReasonCorrelationWeak      = "CORRELATION_NON_AUTHORITATIVE"
	ReasonJudgeOnly            = "JUDGE_ONLY_ADVISORY"
	ReasonOutOfScopeInput      = "EVIDENCE_OUT_OF_SCOPE"
)

func EvaluateCompleteness(in CompletenessInput) Completeness {
	out := Completeness{Status: Complete, EvidenceQuality: High, EnforcementCoverage: High, CorrelationConfidence: High}
	add := func(r string) { out.ReasonCodes = append(out.ReasonCodes, r) }
	if in.OutOfScope {
		out.Status = OutOfScope
		add(ReasonOutOfScopeInput)
		return out
	}
	if in.JudgeOnly {
		out.Status = Unverifiable
		out.EvidenceQuality = Low
		out.EnforcementCoverage = None
		add(ReasonJudgeOnly)
	}
	if !in.Readiness {
		out.Status = Partial
		add(ReasonReadinessMissing)
	}
	if !in.Preconditions {
		out.Status = Partial
		add(ReasonPreconditionsMissing)
	}
	if !in.ProposalSeen || !in.ActivationSeen {
		out.Status = Partial
		add(ReasonSequenceIncomplete)
	}
	switch in.ExecutionOutcome {
	case "":
		out.Status = Partial
		add(ReasonExecutionMissing)
	case "failed":
		if in.CompensationRequired && in.Compensation != "completed" {
			out.Status = Partial
			add(ReasonCompensationMissing)
		}
	}
	if !in.EffectValidated {
		out.Status = Partial
		add(ReasonEffectMissing)
	}
	if in.Containment != "completed" && in.Containment != "not_required" {
		out.Status = Partial
		add(ReasonContainmentMissing)
		if in.Containment == "unresolved" {
			out.Status = Gap
		}
	}
	if !in.CompensationRequired && in.Compensation != "" && in.Compensation != "not_required" {
		out.Status = Partial
	}
	if !in.AuthorityLineage {
		out.Status = Unverifiable
		out.EvidenceQuality = Low
		add(ReasonAuthorityMissing)
	}
	if !in.Fresh {
		out.Status = Partial
		out.EvidenceQuality = Low
		add(ReasonStaleEvidence)
	}
	if in.CorrelationRefs == 0 || !in.CorrelationAuthoritative {
		out.CorrelationConfidence = Low
		add(ReasonCorrelationWeak)
		if out.Status == Complete {
			out.Status = Partial
		}
	}
	if in.JudgeOnly {
		out.Status = Unverifiable
	}
	if in.Containment == "unresolved" && out.Status != OutOfScope && out.Status != Unverifiable {
		out.Status = Gap
	}
	sort.Strings(out.ReasonCodes)
	return out
}

type Rollup struct {
	ContractID            string             `json:"contract_id"`
	Status                CompletenessStatus `json:"status"`
	EvidenceQuality       Confidence         `json:"evidence_quality"`
	EnforcementCoverage   Confidence         `json:"enforcement_coverage"`
	CorrelationConfidence Confidence         `json:"correlation_confidence"`
	ReasonCodes           []string           `json:"reason_codes"`
}

func RollupRegister(items map[string]Completeness) []Rollup {
	ids := make([]string, 0, len(items))
	for id := range items {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	out := make([]Rollup, 0, len(ids))
	for _, id := range ids {
		v := items[id]
		out = append(out, Rollup{ContractID: id, Status: v.Status, EvidenceQuality: v.EvidenceQuality, EnforcementCoverage: v.EnforcementCoverage, CorrelationConfidence: v.CorrelationConfidence, ReasonCodes: append([]string(nil), v.ReasonCodes...)})
	}
	return out
}
