package grade

import (
	"fmt"

	"github.com/Clyra-AI/axym/core/compliance/coverage"
)

type Result struct {
	Letter string  `json:"letter"`
	Score  float64 `json:"score"`
	Reason string  `json:"reason"`
}

func Derive(report coverage.Report) Result {
	total := report.Summary.ControlCount
	if total == 0 {
		return Result{Letter: "F", Score: 0, Reason: "no controls evaluated"}
	}

	covered := report.Summary.CoveredCount
	partial := report.Summary.PartialCount
	gaps := report.Summary.GapCount

	score := (float64(covered)*1 + float64(partial)*0.5) / float64(total)
	letter := "F"
	switch {
	case gaps == 0 && covered == total:
		letter = "A"
	case gaps == 0 && score >= 0.85:
		letter = "B"
	case gaps <= max(1, total/5):
		letter = "C"
	case gaps <= max(1, total/2):
		letter = "D"
	case score > 0:
		letter = "E"
	default:
		letter = "F"
	}

	return Result{
		Letter: letter,
		Score:  round(score),
		Reason: fmt.Sprintf("weakest_link controls=%d covered=%d partial=%d gap=%d", total, covered, partial, gaps),
	}
}

func max(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func round(in float64) float64 {
	if in <= 0 {
		return 0
	}
	if in >= 1 {
		return 1
	}
	return float64(int(in*10000+0.5)) / 10000
}
