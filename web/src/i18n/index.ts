import { createI18n } from 'vue-i18n'
import en from './en'
import zh from './zh'

const navLang = navigator.language.substring(0, 2)

const i18n = createI18n({
  locale: navLang === 'zh' ? 'zh' : 'en',
  fallbackLocale: 'en',
  legacy: false,
  messages: { en, zh },
})

export default i18n
