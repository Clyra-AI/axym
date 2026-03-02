package sessiongap

import (
	"sort"

	"github.com/Clyra-AI/axym/core/ingest/stitch"
)

type Signal struct {
	ReasonCode      string `json:"reason_code"`
	Status          string `json:"status"`
	Auditability    string `json:"auditability"`
	GapStart        string `json:"gap_start"`
	GapEnd          string `json:"gap_end"`
	MissingSeconds  int64  `json:"missing_seconds"`
	BoundaryTrigger string `json:"boundary_trigger"`
}

func BuildSignals(gaps []stitch.Gap) []Signal {
	if len(gaps) == 0 {
		return []Signal{}
	}
	signals := make([]Signal, 0, len(gaps))
	for _, gap := range gaps {
		status := "gap"
		auditability := "low"
		if gap.MissingSeconds <= 5*60 {
			status = "partial"
			auditability = "medium"
		}
		signals = append(signals, Signal{
			ReasonCode:      gap.ReasonCode,
			Status:          status,
			Auditability:    auditability,
			GapStart:        gap.GapStart,
			GapEnd:          gap.GapEnd,
			MissingSeconds:  gap.MissingSeconds,
			BoundaryTrigger: gap.BoundaryTrigger,
		})
	}
	sort.Slice(signals, func(i, j int) bool {
		if signals[i].GapStart != signals[j].GapStart {
			return signals[i].GapStart < signals[j].GapStart
		}
		return signals[i].GapEnd < signals[j].GapEnd
	})
	return signals
}
