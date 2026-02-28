package sink

import "strings"

type Mode string

const (
	ModeFailClosed   Mode = "fail_closed"
	ModeAdvisoryOnly Mode = "advisory_only"
	ModeShadow       Mode = "shadow"
)

type Decision struct {
	Allow      bool
	Degraded   bool
	ReasonCode string
	Message    string
}

func NormalizeMode(mode Mode) Mode {
	switch Mode(strings.ToLower(strings.TrimSpace(string(mode)))) {
	case ModeAdvisoryOnly:
		return ModeAdvisoryOnly
	case ModeShadow:
		return ModeShadow
	default:
		return ModeFailClosed
	}
}

func OnSinkFailure(mode Mode, err error) Decision {
	if err == nil {
		return Decision{Allow: true}
	}
	m := NormalizeMode(mode)
	switch m {
	case ModeAdvisoryOnly, ModeShadow:
		return Decision{
			Allow:      true,
			Degraded:   true,
			ReasonCode: "SINK_UNAVAILABLE",
			Message:    err.Error(),
		}
	default:
		return Decision{
			Allow:      false,
			Degraded:   false,
			ReasonCode: "SINK_UNAVAILABLE",
			Message:    err.Error(),
		}
	}
}
