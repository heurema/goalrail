package pilotlead

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultLeadRetentionDays = 90
	MinLeadRetentionDays     = 7
	MaxLeadRetentionDays     = 365
)

func RunPurge(config Config, store *Store, output io.Writer) error {
	config = config.withDefaults()
	if output == nil {
		output = io.Discard
	}
	if store == nil {
		store = NewStore(config.LeadLogPath, config.Now)
	}

	retentionDays, err := leadRetentionDays(os.Getenv("GOALRAIL_LEAD_RETENTION_DAYS"))
	if err != nil {
		_, _ = fmt.Fprintf(output, "invalid_retention\n")
		return err
	}

	loc := MoscowLocation()
	cutoff := config.Now().In(loc).AddDate(0, 0, -retentionDays).Format("2006-01-02")
	dryRun := os.Getenv("GOALRAIL_PURGE_CONFIRM") != "yes"
	result, err := store.PurgeBeforeLocalDate(cutoff, loc, dryRun)
	if err != nil {
		_, _ = fmt.Fprintf(output, "lead_log_unavailable\n")
		return err
	}

	verb := "purged"
	if dryRun {
		verb = "would_purge"
	}
	_, _ = fmt.Fprintf(
		output,
		"%s retention_days=%d cutoff=%s purged=%d kept=%d kept_unknown_date=%d\n",
		verb,
		retentionDays,
		cutoff,
		result.Purged,
		result.Kept,
		result.KeptUnknownDate,
	)
	return nil
}

func leadRetentionDays(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return DefaultLeadRetentionDays, nil
	}
	days, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("invalid retention days: %q", value)
	}
	if days < MinLeadRetentionDays || days > MaxLeadRetentionDays {
		return 0, fmt.Errorf("retention days outside %d-%d: %d", MinLeadRetentionDays, MaxLeadRetentionDays, days)
	}
	return days, nil
}
