package readiness

import "github.com/heurema/goalrail/apps/cli/internal/spine"

func Score(findings []spine.ReadinessFinding) int {
	score := 0
	for _, finding := range findings {
		weight := weightFor(finding.Check)
		switch finding.Status {
		case spine.FindingStatusPass:
			score += weight
		case spine.FindingStatusWarn:
			score += weight / 2
		}
	}

	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

func Status(score int) string {
	switch {
	case score >= 85:
		return spine.ReadinessStatusReady
	case score >= 60:
		return spine.ReadinessStatusNeedsWork
	default:
		return spine.ReadinessStatusNotReady
	}
}
