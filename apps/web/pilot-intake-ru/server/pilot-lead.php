<?php
declare(strict_types=1);

require_once __DIR__ . '/pilot-mail.php';

const LEAD_SUBJECT = 'Пилот — заявка с RU лендинга';
const LEAD_LOG = '/srv/goalrail/pilot/leads/leads.jsonl';
const MAX_BODY_BYTES = 8192;
const NOTIFICATION_STATUS_RECEIVED = 'received';
const NOTIFICATION_STATUS_NOTIFIED = 'notified';
const NOTIFICATION_STATUS_FAILED = 'notification_failed';
const NOTIFICATION_ERROR_MAIL_UNAVAILABLE = 'mail_unavailable';

function respond(int $status, array $payload): never
{
    http_response_code($status);
    header('Content-Type: application/json; charset=UTF-8');
    header('X-Content-Type-Options: nosniff');
    echo json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    exit;
}

function request_data(): array
{
    $raw = file_get_contents('php://input');
    if ($raw === false) {
        respond(400, ['ok' => false, 'error' => 'invalid_request']);
    }

    if (strlen($raw) > MAX_BODY_BYTES) {
        respond(413, ['ok' => false, 'error' => 'request_too_large']);
    }

    $contentType = strtolower($_SERVER['CONTENT_TYPE'] ?? '');
    if (str_contains($contentType, 'application/json')) {
        $decoded = json_decode($raw, true);
        if (!is_array($decoded)) {
            respond(400, ['ok' => false, 'error' => 'invalid_json']);
        }
        return $decoded;
    }

    if (str_contains($contentType, 'application/x-www-form-urlencoded') || str_contains($contentType, 'multipart/form-data')) {
        return $_POST;
    }

    if ($raw !== '') {
        parse_str($raw, $parsed);
        if (is_array($parsed) && $parsed !== []) {
            return $parsed;
        }
    }

    return $_POST;
}

function field(array $data, string $key): string
{
    $value = $data[$key] ?? '';
    return is_string($value) ? trim($value) : '';
}

function log_unavailable(): never
{
    respond(500, ['ok' => false, 'error' => 'log_unavailable']);
}

function normalized_email(string $email): string
{
    return strtolower(trim($email));
}

function open_lead_log()
{
    $dir = dirname(LEAD_LOG);
    if (!is_dir($dir) && !mkdir($dir, 0770, true) && !is_dir($dir)) {
        log_unavailable();
    }

    $handle = fopen(LEAD_LOG, 'c+');
    if ($handle === false) {
        log_unavailable();
    }

    if (!flock($handle, LOCK_EX)) {
        fclose($handle);
        log_unavailable();
    }

    return $handle;
}

function close_lead_log($handle): void
{
    flock($handle, LOCK_UN);
    fclose($handle);
}

function read_lead_log_entries($handle): array
{
    $entries = [];
    rewind($handle);
    while (($line = fgets($handle)) !== false) {
        $decoded = json_decode($line, true);
        $entries[] = [
            'raw' => $line,
            'record' => is_array($decoded) ? $decoded : null,
        ];
    }

    return $entries;
}

function lead_record_email(array $record): string
{
    $email = $record['email'] ?? '';
    return is_string($email) ? normalized_email($email) : '';
}

function lead_record_status(array $record): ?string
{
    if (!array_key_exists('notification_status', $record)) {
        return null;
    }

    $status = $record['notification_status'];
    return is_string($status) && $status !== '' ? $status : null;
}

function is_retryable_attempt_status(?string $status): bool
{
    return $status === NOTIFICATION_STATUS_FAILED;
}

function is_markable_attempt_status(?string $status): bool
{
    return $status === NOTIFICATION_STATUS_RECEIVED || $status === 'pending' || $status === NOTIFICATION_STATUS_FAILED;
}

function lead_attempt_state(array $entries, string $normalizedEmail): array
{
    $retryIndex = null;

    foreach ($entries as $index => $entry) {
        $record = $entry['record'];
        if (!is_array($record) || lead_record_email($record) !== $normalizedEmail) {
            continue;
        }

        $status = lead_record_status($record);
        if ($status === NOTIFICATION_STATUS_NOTIFIED || $status === null) {
            return ['kind' => 'duplicate'];
        }

        if (is_retryable_attempt_status($status)) {
            $retryIndex = $index;
            continue;
        }

        return ['kind' => 'duplicate'];
    }

    if ($retryIndex !== null) {
        return ['kind' => 'retry', 'index' => $retryIndex];
    }

    return ['kind' => 'new'];
}

function lead_log_line(array $record): ?string
{
    $encoded = json_encode($record, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    return $encoded === false ? null : $encoded . PHP_EOL;
}

function write_all($handle, string $content): bool
{
    $offset = 0;
    $length = strlen($content);

    while ($offset < $length) {
        $written = fwrite($handle, substr($content, $offset));
        if ($written === false || $written === 0) {
            return false;
        }
        $offset += $written;
    }

    return true;
}

function rewrite_lead_log($handle, array $entries): bool
{
    if (ftruncate($handle, 0) === false || rewind($handle) === false) {
        return false;
    }

    foreach ($entries as $entry) {
        $raw = $entry['raw'] ?? '';
        if (!is_string($raw) || !write_all($handle, $raw)) {
            return false;
        }
    }

    return fflush($handle);
}

function append_lead_record($handle, array $record): bool
{
    $line = lead_log_line($record);
    if ($line === null || fseek($handle, 0, SEEK_END) !== 0 || !write_all($handle, $line)) {
        return false;
    }

    return fflush($handle);
}

function prepare_attempt_record(array $record, string $attemptedAt): array
{
    $record['notification_status'] = NOTIFICATION_STATUS_RECEIVED;
    $record['notification_attempted_at'] = $attemptedAt;
    $record['notification_updated_at'] = $attemptedAt;
    unset($record['notification_transport'], $record['notification_error']);

    return $record;
}

function record_lead_notification_attempt(array $record, string $email, string $attemptedAt): bool
{
    $handle = open_lead_log();
    $ok = true;
    $stored = false;

    $entries = read_lead_log_entries($handle);
    $state = lead_attempt_state($entries, normalized_email($email));

    if ($state['kind'] === 'duplicate') {
        close_lead_log($handle);
        return false;
    }

    $attemptRecord = prepare_attempt_record($record, $attemptedAt);
    if ($state['kind'] === 'retry') {
        $index = $state['index'];
        $existing = $entries[$index]['record'] ?? [];
        $updated = is_array($existing) ? array_merge($attemptRecord, $existing) : $attemptRecord;
        $updated = prepare_attempt_record($updated, $attemptedAt);
        $line = lead_log_line($updated);
        if ($line === null) {
            $ok = false;
        } else {
            $entries[$index] = ['raw' => $line, 'record' => $updated];
            $ok = rewrite_lead_log($handle, $entries);
            $stored = $ok;
        }
    } else {
        $ok = append_lead_record($handle, $attemptRecord);
        $stored = $ok;
    }

    close_lead_log($handle);
    if (!$ok) {
        log_unavailable();
    }

    return $stored;
}

function mark_lead_notification_result(string $email, string $status, ?string $transport, ?string $error): bool
{
    $handle = open_lead_log();
    $entries = read_lead_log_entries($handle);
    $targetIndex = null;
    $normalizedEmail = normalized_email($email);

    foreach ($entries as $index => $entry) {
        $record = $entry['record'];
        if (!is_array($record) || lead_record_email($record) !== $normalizedEmail) {
            continue;
        }

        if (is_markable_attempt_status(lead_record_status($record))) {
            $targetIndex = $index;
        }
    }

    if ($targetIndex === null) {
        close_lead_log($handle);
        return false;
    }

    $record = $entries[$targetIndex]['record'];
    $record['notification_status'] = $status;
    $record['notification_updated_at'] = gmdate('c');

    if ($transport !== null) {
        $record['notification_transport'] = $transport;
    } else {
        unset($record['notification_transport']);
    }

    if ($error !== null) {
        $record['notification_error'] = $error;
    } else {
        unset($record['notification_error']);
    }

    $line = lead_log_line($record);
    if ($line === null) {
        close_lead_log($handle);
        return false;
    }

    $entries[$targetIndex] = ['raw' => $line, 'record' => $record];
    $ok = rewrite_lead_log($handle, $entries);
    close_lead_log($handle);

    return $ok;
}

if (($_SERVER['REQUEST_METHOD'] ?? '') !== 'POST') {
    respond(405, ['ok' => false, 'error' => 'method_not_allowed']);
}

$data = request_data();

$honeypot = field($data, 'website');
if ($honeypot !== '') {
    respond(400, ['ok' => false, 'error' => 'invalid_request']);
}

$email = field($data, 'email');
if (
    $email === ''
    || strlen($email) > 254
    || str_contains($email, "\r")
    || str_contains($email, "\n")
    || filter_var($email, FILTER_VALIDATE_EMAIL) === false
) {
    respond(400, ['ok' => false, 'error' => 'invalid_email']);
}

$source = substr(field($data, 'source') ?: 'ru-pilot', 0, 80);
$page = substr(field($data, 'page') ?: 'pilot.goalrail.ru', 0, 120);
$submittedAt = gmdate('c');
$localSubmittedAt = (new DateTimeImmutable('now', new DateTimeZone('Europe/Moscow')))->format('c');
$localSubmittedDate = substr($localSubmittedAt, 0, 10);
$userAgent = substr($_SERVER['HTTP_USER_AGENT'] ?? '', 0, 240);

$record = [
    'submitted_at' => $submittedAt,
    'submitted_at_local' => $localSubmittedAt,
    'submitted_date_local' => $localSubmittedDate,
    'email' => $email,
    'source' => $source,
    'page' => $page,
    'user_agent' => $userAgent,
];

$notificationAttemptedAt = gmdate('c');
$stored = record_lead_notification_attempt($record, $email, $notificationAttemptedAt);
if (!$stored) {
    respond(200, ['ok' => true, 'duplicate' => true]);
}

$body = implode("\n", [
    'Новая заявка с RU лендинга GoalRail.',
    '',
    'Email: ' . $email,
    'Source: ' . $source,
    'Page: ' . $page,
    'Submitted at: ' . $submittedAt,
    '',
    'Ответьте на это письмо, чтобы написать посетителю напрямую.',
]);

try {
    $leadRecipient = pilot_mail_recipient();
    $transport = pilot_send_text_email($leadRecipient, LEAD_SUBJECT, $body, $email);
    if (!mark_lead_notification_result($email, NOTIFICATION_STATUS_NOTIFIED, $transport, null)) {
        log_unavailable();
    }
} catch (PilotMailException) {
    mark_lead_notification_result($email, NOTIFICATION_STATUS_FAILED, null, NOTIFICATION_ERROR_MAIL_UNAVAILABLE);
    respond(500, ['ok' => false, 'error' => 'mail_unavailable']);
}

respond(200, ['ok' => true, 'duplicate' => false]);
