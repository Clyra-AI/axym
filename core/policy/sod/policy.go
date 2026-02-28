package sod

import "strings"

const (
	ReasonRequestorDeployer = "SOD_REQUESTOR_DEPLOYER"
	ReasonMissingActor      = "SOD_MISSING_ACTOR"
)

type Input struct {
	Requestor string
	Approver  string
	Deployer  string
}

type Decision struct {
	Pass        bool     `json:"pass"`
	ReasonCodes []string `json:"reason_codes,omitempty"`
}

func Evaluate(in Input) Decision {
	reasons := []string{}
	requestor := strings.TrimSpace(strings.ToLower(in.Requestor))
	deployer := strings.TrimSpace(strings.ToLower(in.Deployer))
	approver := strings.TrimSpace(strings.ToLower(in.Approver))
	if requestor == "" || deployer == "" || approver == "" {
		reasons = append(reasons, ReasonMissingActor)
	}
	if requestor != "" && deployer != "" && requestor == deployer {
		reasons = append(reasons, ReasonRequestorDeployer)
	}
	if len(reasons) == 0 {
		return Decision{Pass: true}
	}
	return Decision{Pass: false, ReasonCodes: reasons}
}
