<?php
declare(strict_types=1);

class PilotMailException extends RuntimeException
{
}

const PILOT_MAIL_TO_DEFAULT = 'hello@goalrail.dev';
const PILOT_MAIL_RECIPIENT_OVERRIDE = '/srv/goalrail/pilot/backend/lead-recipient.local';
const PILOT_RESEND_API_KEY_FILE = '/srv/goalrail/pilot/backend/resend-api-key.local';
const PILOT_RESEND_API_URL = 'https://api.resend.com/emails';
const PILOT_RESEND_FROM = 'GoalRail Pilot <noreply@skill7.dev>';
const PILOT_POSTFIX_FROM = 'GoalRail Pilot <noreply@pilot.goalrail.ru>';
const PILOT_POSTFIX_ENVELOPE_FROM = 'noreply@pilot.goalrail.ru';

function pilot_mail_validate_email(string $email): bool
{
    return $email !== ''
        && strlen($email) <= 254
        && !str_contains($email, "\r")
        && !str_contains($email, "\n")
        && filter_var($email, FILTER_VALIDATE_EMAIL) !== false;
}

function pilot_mail_recipient(): string
{
    if (!is_readable(PILOT_MAIL_RECIPIENT_OVERRIDE)) {
        return PILOT_MAIL_TO_DEFAULT;
    }

    $recipient = trim((string) file_get_contents(PILOT_MAIL_RECIPIENT_OVERRIDE));
    if (!pilot_mail_validate_email($recipient)) {
        throw new PilotMailException('recipient_unavailable');
    }

    return $recipient;
}

function pilot_resend_api_key(): ?string
{
    if (!is_readable(PILOT_RESEND_API_KEY_FILE)) {
        return null;
    }

    $apiKey = trim((string) file_get_contents(PILOT_RESEND_API_KEY_FILE));
    if (
        $apiKey === ''
        || strlen($apiKey) > 512
        || str_contains($apiKey, "\r")
        || str_contains($apiKey, "\n")
        || !str_starts_with($apiKey, 're_')
    ) {
        throw new PilotMailException('resend_key_unavailable');
    }

    return $apiKey;
}

function pilot_resend_post_json(string $apiKey, string $json): array
{
    if (function_exists('curl_init')) {
        $handle = curl_init(PILOT_RESEND_API_URL);
        if ($handle === false) {
            throw new PilotMailException('resend_unavailable');
        }

        curl_setopt_array($handle, [
            CURLOPT_POST => true,
            CURLOPT_HTTPHEADER => [
                'Authorization: Bearer ' . $apiKey,
                'Content-Type: application/json',
            ],
            CURLOPT_POSTFIELDS => $json,
            CURLOPT_RETURNTRANSFER => true,
            CURLOPT_TIMEOUT => 10,
        ]);

        $body = curl_exec($handle);
        $status = (int) curl_getinfo($handle, CURLINFO_RESPONSE_CODE);
        $error = curl_error($handle);
        curl_close($handle);

        if ($body === false || $status === 0) {
            throw new PilotMailException($error !== '' ? 'resend_unavailable' : 'resend_no_response');
        }

        return [$status, is_string($body) ? $body : ''];
    }

    $context = stream_context_create([
        'http' => [
            'method' => 'POST',
            'header' => implode("\r\n", [
                'Authorization: Bearer ' . $apiKey,
                'Content-Type: application/json',
            ]),
            'content' => $json,
            'ignore_errors' => true,
            'timeout' => 10,
        ],
    ]);

    $body = file_get_contents(PILOT_RESEND_API_URL, false, $context);
    if ($body === false) {
        throw new PilotMailException('resend_unavailable');
    }

    $status = 0;
    foreach ($http_response_header ?? [] as $header) {
        if (preg_match('/^HTTP\/\S+\s+(\d{3})\b/', $header, $matches) === 1) {
            $status = (int) $matches[1];
            break;
        }
    }

    return [$status, $body];
}

function pilot_send_resend_email(string $apiKey, string $to, string $subject, string $text, ?string $replyTo): void
{
    $payload = [
        'from' => PILOT_RESEND_FROM,
        'to' => [$to],
        'subject' => $subject,
        'text' => $text,
    ];

    if ($replyTo !== null && pilot_mail_validate_email($replyTo)) {
        $payload['reply_to'] = $replyTo;
    }

    $json = json_encode($payload, JSON_UNESCAPED_UNICODE | JSON_UNESCAPED_SLASHES);
    if ($json === false) {
        throw new PilotMailException('resend_payload_unavailable');
    }

    [$status, $body] = pilot_resend_post_json($apiKey, $json);
    if ($status < 200 || $status >= 300) {
        throw new PilotMailException('resend_rejected');
    }

    $decoded = json_decode($body, true);
    if (!is_array($decoded) || !isset($decoded['id'])) {
        throw new PilotMailException('resend_unexpected_response');
    }
}

function pilot_encode_subject(string $subject): string
{
    return '=?UTF-8?B?' . base64_encode($subject) . '?=';
}

function pilot_send_postfix_email(string $to, string $subject, string $text, ?string $replyTo): void
{
    $headers = [
        'MIME-Version: 1.0',
        'Content-Type: text/plain; charset=UTF-8',
        'From: ' . PILOT_POSTFIX_FROM,
        'X-Mailer: GoalRail pilot fallback mail transport',
    ];

    if ($replyTo !== null && pilot_mail_validate_email($replyTo)) {
        $headers[] = 'Reply-To: ' . $replyTo;
    }

    if (!mail($to, pilot_encode_subject($subject), $text, implode("\r\n", $headers), '-f' . PILOT_POSTFIX_ENVELOPE_FROM)) {
        throw new PilotMailException('mail_unavailable');
    }
}

function pilot_send_text_email(string $to, string $subject, string $text, ?string $replyTo = null): string
{
    if (!pilot_mail_validate_email($to)) {
        throw new PilotMailException('recipient_unavailable');
    }

    $apiKey = pilot_resend_api_key();
    if ($apiKey !== null) {
        pilot_send_resend_email($apiKey, $to, $subject, $text, $replyTo);
        return 'resend';
    }

    pilot_send_postfix_email($to, $subject, $text, $replyTo);
    return 'postfix';
}
