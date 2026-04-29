<?php
declare(strict_types=1);

require_once __DIR__ . '/pilot-mail.php';

const LEAD_SUBJECT = 'Пилот — заявка с RU лендинга';
const LEAD_LOG = '/srv/goalrail/pilot/leads/leads.jsonl';
const MAX_BODY_BYTES = 8192;

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

function store_lead_once(array $record, string $email): bool
{
    $dir = dirname(LEAD_LOG);
    if (!is_dir($dir) && !mkdir($dir, 0770, true) && !is_dir($dir)) {
        respond(500, ['ok' => false, 'error' => 'log_unavailable']);
    }

    $handle = fopen(LEAD_LOG, 'c+');
    if ($handle === false) {
        respond(500, ['ok' => false, 'error' => 'log_unavailable']);
    }

    if (!flock($handle, LOCK_EX)) {
        fclose($handle);
        respond(500, ['ok' => false, 'error' => 'log_unavailable']);
    }

    $normalizedEmail = strtolower($email);
    rewind($handle);
    while (($line = fgets($handle)) !== false) {
        $existing = json_decode($line, true);
        if (!is_array($existing)) {
            continue;
        }

        $existingEmail = $existing['email'] ?? '';
        if (is_string($existingEmail) && strtolower(trim($existingEmail)) === $normalizedEmail) {
            flock($handle, LOCK_UN);
            fclose($handle);
            return false;
        }
    }

    $line = json_encode($record, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES) . PHP_EOL;
    if ($line === false || fseek($handle, 0, SEEK_END) !== 0 || fwrite($handle, $line) === false) {
        flock($handle, LOCK_UN);
        fclose($handle);
        respond(500, ['ok' => false, 'error' => 'log_unavailable']);
    }

    fflush($handle);
    flock($handle, LOCK_UN);
    fclose($handle);
    return true;
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

$stored = store_lead_once($record, $email);
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
    pilot_send_text_email($leadRecipient, LEAD_SUBJECT, $body, $email);
} catch (PilotMailException) {
    respond(500, ['ok' => false, 'error' => 'mail_unavailable']);
}

respond(200, ['ok' => true, 'duplicate' => false]);
