package replay

import "testing"

func TestTierClassificationDeterministic(t *testing.T) {
	t.Parallel()

	input := RiskInput{ProductionCritical: true, DataSensitivity: "high", PublicExposure: false}
	first := ClassifyTier(input)
	second := ClassifyTier(input)
	if first != "A" || second != "A" {
		t.Fatalf("tier classification mismatch: first=%s second=%s", first, second)
	}
	if first != second {
		t.Fatalf("classification must be deterministic: first=%s second=%s", first, second)
	}
}

func TestTierClassificationCoverage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		input RiskInput
		want  string
	}{
		{name: "high risk", input: RiskInput{ProductionCritical: true, DataSensitivity: "high"}, want: "A"},
		{name: "medium risk", input: RiskInput{ProductionCritical: true, DataSensitivity: "low"}, want: "B"},
		{name: "low risk", input: RiskInput{ProductionCritical: false, DataSensitivity: "low"}, want: "C"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ClassifyTier(tc.input); got != tc.want {
				t.Fatalf("ClassifyTier() = %s want %s", got, tc.want)
			}
		})
	}
}
