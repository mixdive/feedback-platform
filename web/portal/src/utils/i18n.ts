import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import LanguageDetector from 'i18next-browser-languagedetector'
import dayjs from 'dayjs'
import localizedFormat from 'dayjs/plugin/localizedFormat'
import 'dayjs/locale/en'
import 'dayjs/locale/tr'

import en from '@/locales/en.json'
import tr from '@/locales/tr.json'

dayjs.extend(localizedFormat)

export const SUPPORTED_LANGUAGES = ['en', 'tr'] as const
export type SupportedLanguage = (typeof SUPPORTED_LANGUAGES)[number]

function syncDayjs(lng: string) {
  // Map the i18next BCP-47-ish tag ("en", "en-US", "tr-TR") to a dayjs
  // locale we shipped. Anything we don't recognize falls back to English.
  const base = lng.split('-')[0] as SupportedLanguage
  dayjs.locale(SUPPORTED_LANGUAGES.includes(base) ? base : 'en')
}

void i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    fallbackLng: 'en',
    supportedLngs: SUPPORTED_LANGUAGES,
    nonExplicitSupportedLngs: true,
    interpolation: { escapeValue: false },
    resources: {
      en: { translation: en },
      tr: { translation: tr },
    },
    detection: {
      order: ['localStorage', 'navigator'],
      lookupLocalStorage: 'mixdive_lang',
      caches: ['localStorage'],
    },
  })
  .then(() => syncDayjs(i18n.language))

i18n.on('languageChanged', syncDayjs)

export default i18n
