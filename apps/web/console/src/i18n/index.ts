import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';

import { resolveInitialLocale, supportedLocales } from './locale';
import { defaultNS, resources } from './resources';

const initialLocale = resolveInitialLocale({
  search: typeof window === 'undefined' ? '' : window.location.search,
  envDefault: import.meta.env.VITE_GOALRAIL_DEFAULT_LOCALE,
  navigatorLanguage: typeof navigator === 'undefined' ? undefined : navigator.language,
});

void i18n.use(initReactI18next).init({
  resources,
  lng: initialLocale,
  fallbackLng: 'en',
  supportedLngs: supportedLocales,
  defaultNS,
  ns: [defaultNS],
  interpolation: {
    escapeValue: false,
  },
});

export default i18n;
