package spine

const (
	ReadinessStatusReady     = "ready"
	ReadinessStatusNeedsWork = "needs_work"
	ReadinessStatusNotReady  = "not_ready"

	FindingStatusPass = "pass"
	FindingStatusWarn = "warn"
	FindingStatusFail = "fail"
)

type ReadinessReport struct {
	Score                  int                 `json:"score"`
	Status                 string              `json:"status"`
	Findings               []ReadinessFinding  `json:"findings"`
	Evidence               []ReadinessEvidence `json:"evidence"`
	RecommendedNextActions []string            `json:"recommended_next_actions"`
}

type ReadinessFinding struct {
	Check   string `json:"check"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type ReadinessEvidence struct {
	Check  string   `json:"check"`
	Paths  []string `json:"paths,omitempty"`
	Detail string   `json:"detail,omitempty"`
}
