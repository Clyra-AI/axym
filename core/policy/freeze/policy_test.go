package freeze

import (
	"testing"
	"time"
)

func TestEvaluateFreezeWindowViolation(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 2, 28, 12, 0, 0, 0, time.UTC)
	decision := Evaluate(at, []Window{{
		Start: time.Date(2026, 2, 28, 11, 0, 0, 0, time.UTC),
		End:   time.Date(2026, 2, 28, 13, 0, 0, 0, time.UTC),
	}})
	if decision.Pass {
		t.Fatal("expected pass=false")
	}
	if len(decision.ReasonCodes) != 1 || decision.ReasonCodes[0] != ReasonFreezeWindow {
		t.Fatalf("unexpected reasons: %+v", decision.ReasonCodes)
	}
}
