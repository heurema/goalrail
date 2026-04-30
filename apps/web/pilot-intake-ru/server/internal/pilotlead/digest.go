package pilotlead

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func RunDigest(ctx context.Context, config Config, store *Store, mailer Mailer, output io.Writer) error {
	config = config.withDefaults()
	if output == nil {
		output = io.Discard
	}
	if store == nil {
		store = NewStore(config.LeadLogPath, config.Now)
	}
	if mailer == nil {
		mailer = NewTransportMailer(config)
	}

	loc := MoscowLocation()
	date := TargetDigestDate(config.Now(), loc)
	records, err := store.DigestRecords(date, loc)
	if err != nil {
		_, _ = fmt.Fprintf(output, "lead_log_unavailable\n")
		return err
	}
	if len(records) == 0 {
		_, _ = fmt.Fprintf(output, "no_leads date=%s\n", date)
		return nil
	}

	body := digestBody(date, records, loc)
	subject := DigestSubjectStart + " — заявки за " + date
	if os.Getenv("GOALRAIL_DIGEST_DRY_RUN") == "yes" {
		_, _ = fmt.Fprintf(output, "would_send date=%s count=%d\n", date, len(records))
		return nil
	}

	recipient, err := MailRecipient(config)
	if err != nil {
		_, _ = fmt.Fprintf(output, "mail_unavailable\n")
		return err
	}
	transport, err := mailer.SendText(ctx, recipient, subject, body, "")
	if err != nil {
		_, _ = fmt.Fprintf(output, "mail_unavailable\n")
		return err
	}
	_, _ = fmt.Fprintf(output, "sent date=%s count=%d transport=%s\n", date, len(records), transport)
	return nil
}

func digestBody(date string, records []LeadRecord, loc *time.Location) string {
	lines := []string{
		"Заявки с RU лендинга GoalRail за " + date + " (GMT+3).",
		"",
		fmt.Sprintf("Всего: %d", len(records)),
		"",
	}
	for index, record := range records {
		lines = append(lines,
			fmt.Sprintf("#%d", index+1),
			"Время: "+recordLocalTime(record, loc),
			"Email: "+strings.TrimSpace(stringValue(record, "email")),
			"Source: "+strings.TrimSpace(stringValue(record, "source")),
			"Page: "+strings.TrimSpace(stringValue(record, "page")),
			"",
		)
	}
	return strings.Join(lines, "\n")
}
