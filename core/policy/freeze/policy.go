package freeze

import "time"

const ReasonFreezeWindow = "FREEZE_WINDOW"

type Window struct {
	Start time.Time
	End   time.Time
}

type Decision struct {
	Pass        bool     `json:"pass"`
	ReasonCodes []string `json:"reason_codes,omitempty"`
}

func Evaluate(at time.Time, windows []Window) Decision {
	instant := at.UTC()
	for _, window := range windows {
		start := window.Start.UTC()
		end := window.End.UTC()
		if instant.Equal(start) || instant.Equal(end) || (instant.After(start) && instant.Before(end)) {
			return Decision{Pass: false, ReasonCodes: []string{ReasonFreezeWindow}}
		}
	}
	return Decision{Pass: true}
}
