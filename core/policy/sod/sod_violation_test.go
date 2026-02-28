package sod

import "testing"

func TestSoDViolationRequestorEqualsDeployer(t *testing.T) {
	t.Parallel()

	decision := Evaluate(Input{Requestor: "alice", Approver: "bob", Deployer: "alice"})
	if decision.Pass {
		t.Fatal("expected pass=false")
	}
	if len(decision.ReasonCodes) == 0 || decision.ReasonCodes[0] != ReasonRequestorDeployer {
		t.Fatalf("expected %s reason, got %+v", ReasonRequestorDeployer, decision.ReasonCodes)
	}
}
