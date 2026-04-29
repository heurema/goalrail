<?php
declare(strict_types=1);

require_once __DIR__ . '/pilot-mail.php';

const LEAD_SUBJECT_PREFIX = 'Пилот';
const LEAD_LOG = '/srv/goalrail/pilot/leads/leads.jsonl';
const DIGEST_TZ = 'Europe/Moscow';

function target_date(DateTimeZone $tz): string
{
    $override = getenv('GOALRAIL_DIGEST_DATE');
    if (is_string($override) && preg_match('/^\d{4}-\d{2}-\d{2}$/', $override) === 1) {
        return $override;
    }

    return (new DateTimeImmutable('yesterday', $tz))->format('Y-m-d');
}

function record_local_date(array $record, DateTimeZone $tz): string
{
    $stored = $record['submitted_date_local'] ?? '';
    if (is_string($stored) && preg_match('/^\d{4}-\d{2}-\d{2}$/', $stored) === 1) {
        return $stored;
    }

    $submittedAt = $record['submitted_at'] ?? '';
    if (!is_string($submittedAt) || trim($submittedAt) === '') {
        return '';
    }

    try {
        return (new DateTimeImmutable($submittedAt))->setTimezone($tz)->format('Y-m-d');
    } catch (Exception) {
        return '';
    }
}

function record_local_time(array $record, DateTimeZone $tz): string
{
    $stored = $record['submitted_at_local'] ?? '';
    if (is_string($stored) && trim($stored) !== '') {
        return $stored;
    }

    $submittedAt = $record['submitted_at'] ?? '';
    if (!is_string($submittedAt) || trim($submittedAt) === '') {
        return '';
    }

    try {
        return (new DateTimeImmutable($submittedAt))->setTimezone($tz)->format('Y-m-d H:i:s P');
    } catch (Exception) {
        return '';
    }
}

function digest_records(string $date, DateTimeZone $tz): array
{
    if (!is_readable(LEAD_LOG)) {
        return [];
    }

    $records = [];
    $handle = fopen(LEAD_LOG, 'r');
    if ($handle === false) {
        fwrite(STDERR, "lead_log_unavailable\n");
        exit(2);
    }

    while (($line = fgets($handle)) !== false) {
        $record = json_decode($line, true);
        if (!is_array($record)) {
            continue;
        }

        if (record_local_date($record, $tz) === $date) {
            $records[] = $record;
        }
    }

    fclose($handle);
    return $records;
}

function text_field(array $record, string $key): string
{
    $value = $record[$key] ?? '';
    return is_string($value) ? trim($value) : '';
}

$tz = new DateTimeZone(DIGEST_TZ);
$date = target_date($tz);
$records = digest_records($date, $tz);

if ($records === []) {
    echo "no_leads date={$date}\n";
    exit(0);
}

$lines = [
    "Заявки с RU лендинга GoalRail за {$date} (GMT+3).",
    '',
    'Всего: ' . count($records),
    '',
];

foreach ($records as $index => $record) {
    $number = $index + 1;
    $lines[] = "#{$number}";
    $lines[] = 'Время: ' . record_local_time($record, $tz);
    $lines[] = 'Email: ' . text_field($record, 'email');
    $lines[] = 'Source: ' . text_field($record, 'source');
    $lines[] = 'Page: ' . text_field($record, 'page');
    $lines[] = '';
}

$body = implode("\n", $lines);
$subject = LEAD_SUBJECT_PREFIX . " — заявки за {$date}";
if (getenv('GOALRAIL_DIGEST_DRY_RUN') === 'yes') {
    echo "would_send date={$date} count=" . count($records) . "\n";
    exit(0);
}

try {
    $leadRecipient = pilot_mail_recipient();
    $transport = pilot_send_text_email($leadRecipient, $subject, $body);
} catch (PilotMailException) {
    fwrite(STDERR, "mail_unavailable\n");
    exit(1);
}

echo "sent date={$date} count=" . count($records) . " transport={$transport}\n";
