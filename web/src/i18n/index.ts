import { createI18n } from 'vue-i18n'
import en from './en'
import ja from './ja'
import zh from './zh'

const navLang = navigator.language.substring(0, 2)

function detectLocale(): string {
  if (navLang === 'zh') return 'zh'
  if (navLang === 'ja') return 'ja'
  return 'en'
}

const i18n = createI18n({
  locale: detectLocale(),
  fallbackLocale: 'en',
  legacy: false,
  messages: { en, ja, zh },
})

export default i18n
