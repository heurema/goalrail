import type { ConsoleLocale } from './i18n/resources';

interface CalmTimestampOptions {
  now?: Date;
  locale?: ConsoleLocale;
}

const MINUTE_MS = 60 * 1000;
const HOUR_MS = 60 * MINUTE_MS;
const RECENT_HOUR_LIMIT = 6;

const TIME_LABELS = {
  en: {
    justNow: 'just now',
    minAgo: (value: number) => `${value} min ago`,
    hoursAgo: (value: number) => `${value} h ago`,
    today: (time: string) => `Today ${time}`,
    yesterday: (time: string) => `Yesterday ${time}`,
    unknown: 'unknown time',
    dateLocale: 'en-GB',
  },
  ru: {
    justNow: 'только что',
    minAgo: (value: number) => `${value} мин назад`,
    hoursAgo: (value: number) => `${value} ч назад`,
    today: (time: string) => `Сегодня ${time}`,
    yesterday: (time: string) => `Вчера ${time}`,
    unknown: 'время неизвестно',
    dateLocale: 'ru-RU',
  },
} as const;

export function formatCalmTimestamp(value: string, options: CalmTimestampOptions = {}) {
  const locale = options.locale ?? 'en';
  const labels = TIME_LABELS[locale];
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return labels.unknown;
  }

  const now = options.now ?? new Date();
  const diffMs = now.getTime() - date.getTime();
  if (diffMs >= 0 && diffMs < MINUTE_MS) {
    return labels.justNow;
  }

  if (diffMs >= MINUTE_MS && diffMs < HOUR_MS) {
    return labels.minAgo(Math.max(1, Math.floor(diffMs / MINUTE_MS)));
  }

  if (diffMs >= HOUR_MS && diffMs < RECENT_HOUR_LIMIT * HOUR_MS) {
    return labels.hoursAgo(Math.floor(diffMs / HOUR_MS));
  }

  const time = formatLocalTime(date, labels.dateLocale);
  if (isSameLocalDate(date, now)) {
    return labels.today(time);
  }

  if (isYesterday(date, now)) {
    return labels.yesterday(time);
  }

  return new Intl.DateTimeFormat(labels.dateLocale, {
    day: 'numeric',
    month: 'short',
  }).format(date).replace('.', '');
}

function formatLocalTime(date: Date, locale: string) {
  return new Intl.DateTimeFormat(locale, {
    hour: '2-digit',
    minute: '2-digit',
    hour12: false,
  }).format(date);
}

function isSameLocalDate(left: Date, right: Date) {
  return left.getFullYear() === right.getFullYear()
    && left.getMonth() === right.getMonth()
    && left.getDate() === right.getDate();
}

function isYesterday(date: Date, now: Date) {
  const yesterday = new Date(now);
  yesterday.setDate(now.getDate() - 1);
  return isSameLocalDate(date, yesterday);
}
