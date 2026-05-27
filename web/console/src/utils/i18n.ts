import i18n from 'i18next'
import { initReactI18next } from 'react-i18next'
import LanguageDetector from 'i18next-browser-languagedetector'

void i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    fallbackLng: 'en',
    supportedLngs: ['en'],
    interpolation: { escapeValue: false },
    keySeparator: false,
    nsSeparator: false,
    resources: {
      en: { translation: {} },
    },
  })

export default i18n
