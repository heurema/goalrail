import { describe, expect, it } from 'vitest';

import { resolveInitialLocale } from './locale';

describe('resolveInitialLocale', () => {
  it('prefers query param, then Vite default locale, then navigator language, then EN fallback', () => {
    expect(resolveInitialLocale({ search: '?lng=ru', envDefault: 'en', navigatorLanguage: 'en-US' })).toBe('ru');
    expect(resolveInitialLocale({ search: '?lng=fr', envDefault: 'ru', navigatorLanguage: 'en-US' })).toBe('ru');
    expect(resolveInitialLocale({ search: '', envDefault: '', navigatorLanguage: 'ru-RU' })).toBe('ru');
    expect(resolveInitialLocale({ search: '?lng=fr', envDefault: 'de', navigatorLanguage: 'fr-FR' })).toBe('en');
  });
});
