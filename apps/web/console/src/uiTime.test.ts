import { describe, expect, it } from 'vitest';

import { formatCalmTimestamp } from './uiTime';

function localISO(year: number, monthIndex: number, day: number, hour: number, minute: number, second = 0) {
  return new Date(year, monthIndex, day, hour, minute, second).toISOString();
}

describe('uiTime', () => {
  it('formats recent English timestamps without seconds or raw ISO strings', () => {
    const now = new Date(2026, 4, 8, 14, 30, 45);

    expect(formatCalmTimestamp(localISO(2026, 4, 8, 14, 30, 20), { now, locale: 'en' })).toBe('just now');
    expect(formatCalmTimestamp(localISO(2026, 4, 8, 14, 25, 0), { now, locale: 'en' })).toBe('5 min ago');
    expect(formatCalmTimestamp(localISO(2026, 4, 8, 12, 10, 0), { now, locale: 'en' })).toBe('2 h ago');
    expect(formatCalmTimestamp(localISO(2026, 4, 8, 7, 20, 15), { now, locale: 'en' })).toBe('Today 07:20');
    expect(formatCalmTimestamp(localISO(2026, 4, 7, 9, 10, 30), { now, locale: 'en' })).toBe('Yesterday 09:10');
    expect(formatCalmTimestamp(localISO(2026, 4, 6, 18, 5, 30), { now, locale: 'en' })).toBe('6 May');
  });

  it('formats Russian timestamps with the same calm rules', () => {
    const now = new Date(2026, 4, 8, 14, 30, 45);

    expect(formatCalmTimestamp(localISO(2026, 4, 8, 14, 30, 20), { now, locale: 'ru' })).toBe('только что');
    expect(formatCalmTimestamp(localISO(2026, 4, 8, 14, 25, 0), { now, locale: 'ru' })).toBe('5 мин назад');
    expect(formatCalmTimestamp(localISO(2026, 4, 8, 12, 10, 0), { now, locale: 'ru' })).toBe('2 ч назад');
    expect(formatCalmTimestamp(localISO(2026, 4, 8, 7, 20, 15), { now, locale: 'ru' })).toBe('Сегодня 07:20');
    expect(formatCalmTimestamp(localISO(2026, 4, 7, 9, 10, 30), { now, locale: 'ru' })).toBe('Вчера 09:10');
    expect(formatCalmTimestamp(localISO(2026, 4, 6, 18, 5, 30), { now, locale: 'ru' })).toMatch(/^6 мая$/);
  });

  it('does not echo invalid raw timestamps into normal UI labels', () => {
    expect(formatCalmTimestamp('2026-05-not-a-date', { now: new Date(2026, 4, 8), locale: 'en' })).toBe('unknown time');
    expect(formatCalmTimestamp('2026-05-not-a-date', { now: new Date(2026, 4, 8), locale: 'ru' })).toBe('время неизвестно');
  });
});
