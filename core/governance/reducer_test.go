package governance

import "testing"

func TestReduceIsDeterministicAndBounded(t *testing.T) {
	c := Contract{ID: "c", FamilyID: "f", Revision: 1, Action: "deploy", Target: "prod", Environment: "prod", PolicyDigest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Provenance: Ref{Digest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Source: "wrkr", SourceProduct: "wrkr", SchemaID: "wrkr/v1/contract", SchemaVersion: "v1"}}
	if err := VerifyLineage(c, []Ref{{Kind: "target", ID: "prod", Digest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Source: "gait", SourceProduct: "gait", SchemaID: "gait/v1/target", SchemaVersion: "v1"}}); err != nil {
		t.Fatal(err)
	}
	if err := VerifyLineage(c, []Ref{{Kind: "target", ID: "staging", Digest: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef", Source: "gait", SourceProduct: "gait", SchemaID: "gait/v1/target", SchemaVersion: "v1"}}); err == nil {
		t.Fatal("authority expansion accepted")
	}
	events := []Event{{ID: "b", ContractRef: Ref{ID: "c"}, Kind: "execution_succeeded", OccurredAt: "2026-01-04T00:00:00Z"}, {ID: "a", ContractRef: Ref{ID: "c"}, Kind: "execution_started", OccurredAt: "2026-01-03T00:00:00Z"}, {ID: "p", ContractRef: Ref{ID: "c"}, Kind: "proposed", OccurredAt: "2026-01-01T00:00:00Z"}, {ID: "v", ContractRef: Ref{ID: "c"}, Kind: "activated", OccurredAt: "2026-01-02T00:00:00Z"}}
	got, err := Reduce("c", events)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "succeeded" || !got.Complete || got.Events[2] != "a" {
		t.Fatalf("unexpected state %+v", got)
	}
}

func TestReduceUnresolvedContainmentIsNotComplete(t *testing.T) {
	ref := verifiedRef("c")
	digests := []string{
		"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		"sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		"sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd",
		"sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee",
	}
	events := []Event{
		{ID: "p", ContractRef: ref, Kind: "proposed", OccurredAt: "2026-01-01T00:00:00Z", SourceDigest: digests[0]},
		{ID: "a", ContractRef: ref, Kind: "activated", OccurredAt: "2026-01-01T00:00:01Z", SourceDigest: digests[1]},
		{ID: "s", ContractRef: ref, Kind: "execution_started", OccurredAt: "2026-01-01T00:00:02Z", SourceDigest: digests[2]},
		{ID: "x", ContractRef: ref, Kind: "execution_succeeded", OccurredAt: "2026-01-01T00:00:03Z", SourceDigest: digests[3]},
		{ID: "c", ContractRef: ref, Kind: "contained", Status: "unresolved", OccurredAt: "2026-01-01T00:00:04Z", SourceDigest: digests[4]},
	}
	state, err := ReduceVerified("c", events)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != "gap" || state.Complete {
		t.Fatalf("unresolved containment promoted to completion: %+v", state)
	}
}

func TestReduceUnresolvedEffectIsNotValidated(t *testing.T) {
	ref := verifiedRef("c")
	events := []Event{
		{ID: "p", ContractRef: ref, Kind: "proposed", OccurredAt: "2026-01-01T00:00:00Z", SourceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{ID: "a", ContractRef: ref, Kind: "activated", OccurredAt: "2026-01-01T00:00:01Z", SourceDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		{ID: "s", ContractRef: ref, Kind: "execution_started", OccurredAt: "2026-01-01T00:00:02Z", SourceDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
		{ID: "x", ContractRef: ref, Kind: "execution_succeeded", OccurredAt: "2026-01-01T00:00:03Z", SourceDigest: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
		{ID: "e", ContractRef: ref, Kind: "effect_unresolved", Status: "unknown", OccurredAt: "2026-01-01T00:00:04Z", SourceDigest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},
	}
	state, err := ReduceVerified("c", events)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != "gap" || state.Complete {
		t.Fatalf("unresolved effect promoted to validation: %+v", state)
	}
}

func TestReduceCompensationLifecycleIsRepresented(t *testing.T) {
	ref := verifiedRef("c")
	events := []Event{
		{ID: "p", ContractRef: ref, Kind: "proposed", OccurredAt: "2026-01-01T00:00:00Z", SourceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{ID: "a", ContractRef: ref, Kind: "activated", OccurredAt: "2026-01-01T00:00:01Z", SourceDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		{ID: "s", ContractRef: ref, Kind: "execution_started", OccurredAt: "2026-01-01T00:00:02Z", SourceDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
		{ID: "x", ContractRef: ref, Kind: "execution_failed", OccurredAt: "2026-01-01T00:00:03Z", SourceDigest: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
		{ID: "q", ContractRef: ref, Kind: "compensation_required", Status: "required", OccurredAt: "2026-01-01T00:00:04Z", SourceDigest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},
		{ID: "r", ContractRef: ref, Kind: "compensation_started", Status: "started", OccurredAt: "2026-01-01T00:00:05Z", SourceDigest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{ID: "k", ContractRef: ref, Kind: "compensated", Status: "completed", OccurredAt: "2026-01-01T00:00:06Z", SourceDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
	}
	state, err := ReduceVerified("c", events)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != "compensated" || !state.Complete {
		t.Fatalf("compensation lifecycle was not completed: %+v", state)
	}
	for _, reason := range state.ReasonCodes {
		if reason == "REQUIRED_COMPENSATION_MISSING" {
			t.Fatalf("completed compensation retained missing requirement: %+v", state)
		}
	}

	events[len(events)-1].Kind = "compensation_unresolved"
	events[len(events)-1].Status = "unresolved"
	state, err = ReduceVerified("c", events)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != "gap" || state.Complete {
		t.Fatalf("unresolved compensation promoted to completion: %+v", state)
	}

	notRequired := events[:4]
	notRequired = append(notRequired, Event{ID: "n", ContractRef: ref, Kind: "compensation_not_required", Status: "not_required", OccurredAt: "2026-01-01T00:00:04Z", SourceDigest: "sha256:2222222222222222222222222222222222222222222222222222222222222222"})
	state, err = ReduceVerified("c", notRequired)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != "failed" || !state.Complete {
		t.Fatalf("not-required disposition changed or obscured execution outcome: %+v", state)
	}

	postContainment := []Event{
		{ID: "p", ContractRef: ref, Kind: "proposed", OccurredAt: "2026-01-02T00:00:00Z", SourceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{ID: "a", ContractRef: ref, Kind: "activated", OccurredAt: "2026-01-02T00:00:01Z", SourceDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		{ID: "s", ContractRef: ref, Kind: "execution_started", OccurredAt: "2026-01-02T00:00:02Z", SourceDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
		{ID: "x", ContractRef: ref, Kind: "execution_failed", OccurredAt: "2026-01-02T00:00:03Z", SourceDigest: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
		{ID: "c", ContractRef: ref, Kind: "contained", Status: "completed", OccurredAt: "2026-01-02T00:00:04Z", SourceDigest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},
		{ID: "q", ContractRef: ref, Kind: "compensation_required", Status: "required", OccurredAt: "2026-01-02T00:00:05Z", SourceDigest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{ID: "r", ContractRef: ref, Kind: "compensation_started", Status: "started", OccurredAt: "2026-01-02T00:00:06Z", SourceDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		{ID: "k", ContractRef: ref, Kind: "compensated", Status: "completed", OccurredAt: "2026-01-02T00:00:07Z", SourceDigest: "sha256:2222222222222222222222222222222222222222222222222222222222222222"},
	}
	state, err = ReduceVerified("c", postContainment)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != "compensated" || !state.Complete {
		t.Fatalf("compensation after containment was rejected: %+v", state)
	}

	postCompensationContainment := append(append([]Event(nil), postContainment[:4]...),
		Event{ID: "q", ContractRef: ref, Kind: "compensation_required", Status: "required", OccurredAt: "2026-01-02T00:00:04Z", SourceDigest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},
		Event{ID: "r", ContractRef: ref, Kind: "compensation_started", Status: "started", OccurredAt: "2026-01-02T00:00:05Z", SourceDigest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		Event{ID: "k", ContractRef: ref, Kind: "compensated", Status: "completed", OccurredAt: "2026-01-02T00:00:06Z", SourceDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		Event{ID: "c", ContractRef: ref, Kind: "contained", Status: "completed", OccurredAt: "2026-01-02T00:00:07Z", SourceDigest: "sha256:2222222222222222222222222222222222222222222222222222222222222222"},
	)
	state, err = ReduceVerified("c", postCompensationContainment)
	if err != nil || state.Status != "contained" || !state.Complete {
		t.Fatalf("containment after compensation was rejected: state=%+v err=%v", state, err)
	}
}

func TestReduceBlockedAndPartialOutcomesRemainIncomplete(t *testing.T) {
	ref := verifiedRef("c")
	base := []Event{
		{ID: "p", ContractRef: ref, Kind: "proposed", OccurredAt: "2026-01-01T00:00:00Z", SourceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{ID: "a", ContractRef: ref, Kind: "activated", OccurredAt: "2026-01-01T00:00:01Z", SourceDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
	}
	blocked := append(append([]Event(nil), base...), Event{ID: "b", ContractRef: ref, Kind: "execution_blocked", Status: "blocked", OccurredAt: "2026-01-01T00:00:02Z", SourceDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"})
	state, err := ReduceVerified("c", blocked)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != "blocked" || state.Complete {
		t.Fatalf("blocked execution promoted to completion: %+v", state)
	}
	blockedComp := append(append([]Event(nil), blocked...),
		Event{ID: "q", ContractRef: ref, Kind: "compensation_required", Status: "required", OccurredAt: "2026-01-01T00:00:03Z", SourceDigest: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
		Event{ID: "r", ContractRef: ref, Kind: "compensation_started", Status: "started", OccurredAt: "2026-01-01T00:00:04Z", SourceDigest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},
		Event{ID: "k", ContractRef: ref, Kind: "compensated", Status: "completed", OccurredAt: "2026-01-01T00:00:05Z", SourceDigest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
	)
	state, err = ReduceVerified("c", blockedComp)
	if err != nil || state.Status != "compensated" || state.Complete {
		t.Fatalf("compensation after blocked execution was rejected: state=%+v err=%v", state, err)
	}
	partial := append(append([]Event(nil), base...),
		Event{ID: "s", ContractRef: ref, Kind: "execution_started", OccurredAt: "2026-01-01T00:00:02Z", SourceDigest: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
		Event{ID: "x", ContractRef: ref, Kind: "execution_succeeded", OccurredAt: "2026-01-01T00:00:03Z", SourceDigest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},
		Event{ID: "m", ContractRef: ref, Kind: "containment_partial", Status: "partial", OccurredAt: "2026-01-01T00:00:04Z", SourceDigest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
	)
	state, err = ReduceVerified("c", partial)
	if err != nil {
		t.Fatal(err)
	}
	if state.Status != "partial" || state.Complete {
		t.Fatalf("partial containment promoted to completion: %+v", state)
	}
	partialComp := append(append([]Event(nil), partial...),
		Event{ID: "q", ContractRef: ref, Kind: "compensation_required", Status: "required", OccurredAt: "2026-01-01T00:00:05Z", SourceDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		Event{ID: "r", ContractRef: ref, Kind: "compensation_started", Status: "started", OccurredAt: "2026-01-01T00:00:06Z", SourceDigest: "sha256:2222222222222222222222222222222222222222222222222222222222222222"},
		Event{ID: "k", ContractRef: ref, Kind: "compensated", Status: "completed", OccurredAt: "2026-01-01T00:00:07Z", SourceDigest: "sha256:3333333333333333333333333333333333333333333333333333333333333333"},
	)
	state, err = ReduceVerified("c", partialComp)
	if err != nil || state.Status != "compensated" || state.Complete {
		t.Fatalf("compensation after partial containment was rejected or hid the gap: state=%+v err=%v", state, err)
	}
	unresolved := append(append([]Event(nil), base...),
		Event{ID: "s", ContractRef: ref, Kind: "execution_started", OccurredAt: "2026-01-01T00:00:02Z", SourceDigest: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
		Event{ID: "x", ContractRef: ref, Kind: "execution_succeeded", OccurredAt: "2026-01-01T00:00:03Z", SourceDigest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},
		Event{ID: "u", ContractRef: ref, Kind: "containment_unresolved", Status: "unresolved", OccurredAt: "2026-01-01T00:00:04Z", SourceDigest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		Event{ID: "q", ContractRef: ref, Kind: "compensation_required", Status: "required", OccurredAt: "2026-01-01T00:00:05Z", SourceDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		Event{ID: "r", ContractRef: ref, Kind: "compensation_started", Status: "started", OccurredAt: "2026-01-01T00:00:06Z", SourceDigest: "sha256:2222222222222222222222222222222222222222222222222222222222222222"},
		Event{ID: "k", ContractRef: ref, Kind: "compensated", Status: "completed", OccurredAt: "2026-01-01T00:00:07Z", SourceDigest: "sha256:3333333333333333333333333333333333333333333333333333333333333333"},
	)
	state, err = ReduceVerified("c", unresolved)
	if err != nil || state.Status != "compensated" || state.Complete {
		t.Fatalf("compensation after unresolved containment was rejected or hid the gap: state=%+v err=%v", state, err)
	}
	interleaved := append(append([]Event(nil), base...),
		Event{ID: "s", ContractRef: ref, Kind: "execution_started", OccurredAt: "2026-01-01T00:00:02Z", SourceDigest: "sha256:4444444444444444444444444444444444444444444444444444444444444444"},
		Event{ID: "x", ContractRef: ref, Kind: "execution_succeeded", OccurredAt: "2026-01-01T00:00:03Z", SourceDigest: "sha256:5555555555555555555555555555555555555555555555555555555555555555"},
		Event{ID: "q", ContractRef: ref, Kind: "compensation_required", Status: "required", OccurredAt: "2026-01-01T00:00:04Z", SourceDigest: "sha256:6666666666666666666666666666666666666666666666666666666666666666"},
		Event{ID: "r", ContractRef: ref, Kind: "compensation_started", Status: "started", OccurredAt: "2026-01-01T00:00:05Z", SourceDigest: "sha256:7777777777777777777777777777777777777777777777777777777777777777"},
		Event{ID: "c", ContractRef: ref, Kind: "containment_requested", Status: "requested", OccurredAt: "2026-01-01T00:00:06Z", SourceDigest: "sha256:8888888888888888888888888888888888888888888888888888888888888888"},
		Event{ID: "k", ContractRef: ref, Kind: "compensated", Status: "completed", OccurredAt: "2026-01-01T00:00:07Z", SourceDigest: "sha256:9999999999999999999999999999999999999999999999999999999999999999"},
		Event{ID: "d", ContractRef: ref, Kind: "contained", Status: "completed", OccurredAt: "2026-01-01T00:00:08Z", SourceDigest: "sha256:abababababababababababababababababababababababababababababababab"},
	)
	state, err = ReduceVerified("c", interleaved)
	if err != nil || state.Status != "contained" || !state.Complete {
		t.Fatalf("verified containment/compensation interleave was rejected: state=%+v err=%v", state, err)
	}
}

func TestReduceEffectTransitionsCanInterleaveCompensation(t *testing.T) {
	ref := verifiedRef("c")
	events := []Event{
		{ID: "p", ContractRef: ref, Kind: "proposed", OccurredAt: "2026-01-03T00:00:00Z", SourceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{ID: "a", ContractRef: ref, Kind: "activated", OccurredAt: "2026-01-03T00:00:01Z", SourceDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		{ID: "s", ContractRef: ref, Kind: "execution_started", OccurredAt: "2026-01-03T00:00:02Z", SourceDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
		{ID: "x", ContractRef: ref, Kind: "execution_succeeded", OccurredAt: "2026-01-03T00:00:03Z", SourceDigest: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
		{ID: "q", ContractRef: ref, Kind: "compensation_required", Status: "required", OccurredAt: "2026-01-03T00:00:04Z", SourceDigest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},
		{ID: "r", ContractRef: ref, Kind: "effect_recorded", Status: "recorded", OccurredAt: "2026-01-03T00:00:05Z", SourceDigest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{ID: "v", ContractRef: ref, Kind: "effect_validated", Status: "validated", OccurredAt: "2026-01-03T00:00:06Z", SourceDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		{ID: "b", ContractRef: ref, Kind: "compensation_started", Status: "started", OccurredAt: "2026-01-03T00:00:07Z", SourceDigest: "sha256:2222222222222222222222222222222222222222222222222222222222222222"},
		{ID: "k", ContractRef: ref, Kind: "compensated", Status: "completed", OccurredAt: "2026-01-03T00:00:08Z", SourceDigest: "sha256:3333333333333333333333333333333333333333333333333333333333333333"},
	}
	state, err := ReduceVerified("c", events)
	if err != nil || state.Status != "compensated" || !state.Complete {
		t.Fatalf("effect/compensation interleave was rejected: state=%+v err=%v", state, err)
	}

	// A producer may emit effect evidence after compensation completes. The
	// reducer must accept both transitions and preserve the validated outcome.
	postCompensation := []Event{
		{ID: "p", ContractRef: ref, Kind: "proposed", OccurredAt: "2026-01-04T00:00:00Z", SourceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{ID: "a", ContractRef: ref, Kind: "activated", OccurredAt: "2026-01-04T00:00:01Z", SourceDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		{ID: "s", ContractRef: ref, Kind: "execution_started", OccurredAt: "2026-01-04T00:00:02Z", SourceDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
		{ID: "x", ContractRef: ref, Kind: "execution_failed", OccurredAt: "2026-01-04T00:00:03Z", SourceDigest: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
		{ID: "q", ContractRef: ref, Kind: "compensation_required", Status: "required", OccurredAt: "2026-01-04T00:00:04Z", SourceDigest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},
		{ID: "b", ContractRef: ref, Kind: "compensation_started", Status: "started", OccurredAt: "2026-01-04T00:00:05Z", SourceDigest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{ID: "k", ContractRef: ref, Kind: "compensated", Status: "completed", OccurredAt: "2026-01-04T00:00:06Z", SourceDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		{ID: "r", ContractRef: ref, Kind: "effect_recorded", Status: "recorded", OccurredAt: "2026-01-04T00:00:07Z", SourceDigest: "sha256:2222222222222222222222222222222222222222222222222222222222222222"},
		{ID: "v", ContractRef: ref, Kind: "effect_validated", Status: "validated", OccurredAt: "2026-01-04T00:00:08Z", SourceDigest: "sha256:3333333333333333333333333333333333333333333333333333333333333333"},
	}
	state, err = ReduceVerified("c", postCompensation)
	if err != nil || state.Status != "succeeded" || !state.Complete {
		t.Fatalf("effect after compensation was rejected: state=%+v err=%v", state, err)
	}
	for _, reason := range state.ReasonCodes {
		if reason == "REQUIRED_COMPENSATION_MISSING" {
			t.Fatalf("effect after completed compensation retained missing requirement: %+v", state)
		}
	}
}

func TestReduceOrdersEquivalentOffsetTimestampsByInstant(t *testing.T) {
	ref := verifiedRef("c")
	events := []Event{
		{ID: "p", ContractRef: ref, Kind: "proposed", OccurredAt: "2026-01-05T00:00:00Z", SourceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{ID: "a", ContractRef: ref, Kind: "activated", OccurredAt: "2026-01-05T00:00:01Z", SourceDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		// 10:00+02:00 is 08:00Z and must precede 08:30Z despite its text.
		{ID: "s", ContractRef: ref, Kind: "execution_started", OccurredAt: "2026-01-05T10:00:00+02:00", SourceDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
		{ID: "x", ContractRef: ref, Kind: "execution_succeeded", OccurredAt: "2026-01-05T08:30:00Z", SourceDigest: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
	}
	state, err := ReduceVerified("c", events)
	if err != nil || state.Status != "succeeded" || !state.Complete {
		t.Fatalf("equivalent offset timestamps replayed out of order: state=%+v err=%v", state, err)
	}
	if len(state.Events) != 4 || state.Events[2] != "s" || state.Events[3] != "x" {
		t.Fatalf("producer transitions were not ordered by instant: %+v", state.Events)
	}
}

func TestReduceAllowsRepeatedTerminalLifecycleRuns(t *testing.T) {
	ref := verifiedRef("c")
	events := []Event{
		{ID: "p", ContractRef: ref, Kind: "proposed", OccurredAt: "2026-01-06T00:00:00Z", SourceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
		{ID: "a1", ContractRef: ref, Kind: "activated", OccurredAt: "2026-01-06T00:00:01Z", SourceDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
		{ID: "s1", ContractRef: ref, Kind: "execution_started", OccurredAt: "2026-01-06T00:00:02Z", SourceDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
		{ID: "x1", ContractRef: ref, Kind: "execution_succeeded", OccurredAt: "2026-01-06T00:00:03Z", SourceDigest: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
		{ID: "c1", ContractRef: ref, Kind: "contained", Status: "completed", OccurredAt: "2026-01-06T00:00:04Z", SourceDigest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},
		{ID: "a2", ContractRef: ref, Kind: "activated", OccurredAt: "2026-01-06T00:00:05Z", SourceDigest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
		{ID: "s2", ContractRef: ref, Kind: "execution_started", OccurredAt: "2026-01-06T00:00:06Z", SourceDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
		{ID: "x2", ContractRef: ref, Kind: "execution_succeeded", OccurredAt: "2026-01-06T00:00:07Z", SourceDigest: "sha256:2222222222222222222222222222222222222222222222222222222222222222"},
	}
	state, err := ReduceVerified("c", events)
	if err != nil || state.Status != "succeeded" || !state.Complete {
		t.Fatalf("repeated lifecycle run was rejected: state=%+v err=%v", state, err)
	}
}

func TestReduceAllowsVerifiedRestartAfterFailedAndBlockedRuns(t *testing.T) {
	ref := verifiedRef("c")
	for _, terminal := range []struct {
		name   string
		status string
		kind   string
	}{
		{"failed", "failed", "execution_failed"},
		{"blocked", "blocked", "execution_blocked"},
	} {
		t.Run(terminal.name, func(t *testing.T) {
			events := []Event{
				{ID: "p", ContractRef: ref, Kind: "proposed", OccurredAt: "2026-01-07T00:00:00Z", SourceDigest: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
				{ID: "a1", ContractRef: ref, Kind: "activated", OccurredAt: "2026-01-07T00:00:01Z", SourceDigest: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"},
				{ID: "s1", ContractRef: ref, Kind: "execution_started", OccurredAt: "2026-01-07T00:00:02Z", SourceDigest: "sha256:cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},
				{ID: "x1", ContractRef: ref, Kind: terminal.kind, OccurredAt: "2026-01-07T00:00:03Z", SourceDigest: "sha256:dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"},
				{ID: "a2", ContractRef: ref, Kind: "activated", OccurredAt: "2026-01-07T00:00:04Z", SourceDigest: "sha256:eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},
				{ID: "s2", ContractRef: ref, Kind: "execution_started", OccurredAt: "2026-01-07T00:00:05Z", SourceDigest: "sha256:ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"},
				{ID: "x2", ContractRef: ref, Kind: "execution_succeeded", OccurredAt: "2026-01-07T00:00:06Z", SourceDigest: "sha256:1111111111111111111111111111111111111111111111111111111111111111"},
			}
			state, err := ReduceVerified("c", events)
			if err != nil || state.Status != "succeeded" {
				t.Fatalf("verified restart after %s was rejected: state=%+v err=%v", terminal.status, state, err)
			}
		})
	}
}
