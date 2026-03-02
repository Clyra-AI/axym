package privilegedrift

import "sort"

const ReasonClassUnapprovedPrivilegeEscalation = "WRKR_PRIVILEGE_ESCALATION_UNAPPROVED"

type Observation struct {
	Principal string
	Privilege string
	Approved  bool
	RecordID  string
}

type Gap struct {
	Status      string `json:"status"`
	ReasonClass string `json:"reason_class"`
	Principal   string `json:"principal"`
	Privilege   string `json:"privilege"`
	RecordID    string `json:"record_id,omitempty"`
}

func Analyze(baseline map[string][]string, observations []Observation) (map[string][]string, []Gap) {
	updated := cloneBaseline(baseline)
	gaps := make([]Gap, 0)

	for _, obs := range observations {
		if obs.Principal == "" || obs.Privilege == "" {
			continue
		}
		privs := ensurePrincipal(updated, obs.Principal)
		if hasPrivilege(privs, obs.Privilege) {
			continue
		}
		updated[obs.Principal] = insertPrivilege(privs, obs.Privilege)
		if obs.Approved {
			continue
		}
		gaps = append(gaps, Gap{
			Status:      "gap",
			ReasonClass: ReasonClassUnapprovedPrivilegeEscalation,
			Principal:   obs.Principal,
			Privilege:   obs.Privilege,
			RecordID:    obs.RecordID,
		})
	}

	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].Principal != gaps[j].Principal {
			return gaps[i].Principal < gaps[j].Principal
		}
		if gaps[i].Privilege != gaps[j].Privilege {
			return gaps[i].Privilege < gaps[j].Privilege
		}
		return gaps[i].RecordID < gaps[j].RecordID
	})
	return updated, gaps
}

func cloneBaseline(in map[string][]string) map[string][]string {
	out := map[string][]string{}
	for principal, privileges := range in {
		copied := append([]string(nil), privileges...)
		sort.Strings(copied)
		out[principal] = copied
	}
	return out
}

func ensurePrincipal(state map[string][]string, principal string) []string {
	if _, ok := state[principal]; !ok {
		state[principal] = []string{}
	}
	return state[principal]
}

func hasPrivilege(privs []string, candidate string) bool {
	i := sort.SearchStrings(privs, candidate)
	return i < len(privs) && privs[i] == candidate
}

func insertPrivilege(privs []string, candidate string) []string {
	i := sort.SearchStrings(privs, candidate)
	if i < len(privs) && privs[i] == candidate {
		return privs
	}
	privs = append(privs, "")
	copy(privs[i+1:], privs[i:])
	privs[i] = candidate
	return privs
}
