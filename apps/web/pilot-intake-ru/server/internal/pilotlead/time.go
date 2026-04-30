package pilotlead

import (
	"os"
	"regexp"
	"time"
)

var digestDatePattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

func MoscowLocation() *time.Location {
	loc, err := time.LoadLocation(DigestTZ)
	if err != nil {
		return time.FixedZone("GMT+3", 3*60*60)
	}
	return loc
}

func formatLocal(t time.Time, loc *time.Location) string {
	if loc == nil {
		loc = MoscowLocation()
	}
	return t.In(loc).Format(time.RFC3339)
}

func formatLocalDate(t time.Time, loc *time.Location) string {
	if loc == nil {
		loc = MoscowLocation()
	}
	return t.In(loc).Format("2006-01-02")
}

func TargetDigestDate(now time.Time, loc *time.Location) string {
	if override := os.Getenv("GOALRAIL_DIGEST_DATE"); digestDatePattern.MatchString(override) {
		return override
	}
	if loc == nil {
		loc = MoscowLocation()
	}
	return now.In(loc).AddDate(0, 0, -1).Format("2006-01-02")
}

func recordLocalDate(record LeadRecord, loc *time.Location) string {
	stored := stringValue(record, "submitted_date_local")
	if digestDatePattern.MatchString(stored) {
		return stored
	}

	submittedAt := stringValue(record, "submitted_at")
	if submittedAt == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, submittedAt)
	if err != nil {
		return ""
	}
	if loc == nil {
		loc = MoscowLocation()
	}
	return parsed.In(loc).Format("2006-01-02")
}

func retentionLocalDate(record LeadRecord, loc *time.Location) (string, bool) {
	if record == nil {
		return "", false
	}
	if stored, ok := parseLocalDate(stringValue(record, "submitted_date_local")); ok {
		return stored, true
	}

	submittedAt := stringValue(record, "submitted_at")
	if submittedAt == "" {
		return "", false
	}
	parsed, err := time.Parse(time.RFC3339, submittedAt)
	if err != nil {
		return "", false
	}
	if loc == nil {
		loc = MoscowLocation()
	}
	return parsed.In(loc).Format("2006-01-02"), true
}

func parseLocalDate(value string) (string, bool) {
	if !digestDatePattern.MatchString(value) {
		return "", false
	}
	parsed, err := time.Parse("2006-01-02", value)
	if err != nil {
		return "", false
	}
	return parsed.Format("2006-01-02"), true
}

func recordLocalTime(record LeadRecord, loc *time.Location) string {
	stored := stringValue(record, "submitted_at_local")
	if stored != "" {
		return stored
	}

	submittedAt := stringValue(record, "submitted_at")
	if submittedAt == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339, submittedAt)
	if err != nil {
		return ""
	}
	if loc == nil {
		loc = MoscowLocation()
	}
	return parsed.In(loc).Format("2006-01-02 15:04:05 -07:00")
}
