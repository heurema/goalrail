import type { ConsoleLocale } from './resources';

export const supportedLocales = ['en', 'ru'] as const satisfies readonly ConsoleLocale[];

export function isSupportedLocale(value: string | null | undefined): value is ConsoleLocale {
  return supportedLocales.includes(value as ConsoleLocale);
}

export function resolveInitialLocale(input: {
  search?: string;
  envDefault?: string;
  navigatorLanguage?: string;
}): ConsoleLocale {
  const queryLocale = new URLSearchParams(input.search ?? '').get('lng');
  if (isSupportedLocale(queryLocale)) {
    return queryLocale;
  }

  const envLocale = input.envDefault?.trim().toLowerCase();
  if (isSupportedLocale(envLocale)) {
    return envLocale;
  }

  const browserLocale = input.navigatorLanguage?.trim().toLowerCase().split('-')[0];
  if (isSupportedLocale(browserLocale)) {
    return browserLocale;
  }

  return 'en';
}

export function updateLocaleQueryParam(locale: ConsoleLocale) {
  if (typeof window === 'undefined') {
    return;
  }

  const url = new URL(window.location.href);
  url.searchParams.set('lng', locale);
  window.history.replaceState(window.history.state, '', `${url.pathname}${url.search}${url.hash}`);
}
