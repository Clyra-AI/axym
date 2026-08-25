package governance

import "testing"

func TestCompletenessMatrix(t *testing.T) {
	cases := []struct {
		name   string
		in     CompletenessInput
		want   CompletenessStatus
		reason string
	}{{"success", CompletenessInput{Readiness: true, Preconditions: true, ProposalSeen: true, ActivationSeen: true, ExecutionOutcome: "succeeded", EffectValidated: true, Containment: "completed", AuthorityLineage: true, Fresh: true, CorrelationRefs: 2, CorrelationAuthoritative: true}, Complete, ""}, {"failed compensated", CompletenessInput{Readiness: true, Preconditions: true, ProposalSeen: true, ActivationSeen: true, ExecutionOutcome: "failed", EffectValidated: true, Containment: "completed", CompensationRequired: true, Compensation: "completed", AuthorityLineage: true, Fresh: true, CorrelationRefs: 2, CorrelationAuthoritative: true}, Complete, ""}, {"judge", CompletenessInput{JudgeOnly: true}, Unverifiable, ReasonJudgeOnly}, {"stale", CompletenessInput{AuthorityLineage: true, Fresh: false}, Partial, ReasonStaleEvidence}, {"scope", CompletenessInput{OutOfScope: true}, OutOfScope, ReasonOutOfScopeInput}, {"unresolved containment", CompletenessInput{AuthorityLineage: true, Containment: "unresolved"}, Gap, ReasonContainmentMissing}}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := EvaluateCompleteness(tc.in)
			if got.Status != tc.want {
				t.Fatalf("status=%s want=%s reasons=%v", got.Status, tc.want, got.ReasonCodes)
			}
			if tc.reason != "" && !contains(got.ReasonCodes, tc.reason) {
				t.Fatalf("missing reason %s", tc.reason)
			}
		})
	}
}
func contains(values []string, want string) bool {
	for _, v := range values {
		if v == want {
			return true
		}
	}
	return false
}
