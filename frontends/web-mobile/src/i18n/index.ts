import { createI18n } from 'vue-i18n'

export interface LanguageInfo {
  code: string
  name: string
}

// Автоматическое обнаружение и загрузка всех языковых файлов из папки ./locales/
const localeFiles = import.meta.glob('./locales/*.json', { eager: true }) as Record<string, any>

export const availableLanguages: LanguageInfo[] = []
const messages: Record<string, any> = {}

for (const path in localeFiles) {
  const match = path.match(/\/([a-zA-Z0-9_-]+)\.json$/)
  if (match && match[1]) {
    const code = match[1]
    const content = localeFiles[path].default || localeFiles[path]
    messages[code] = content
    availableLanguages.push({
      code,
      name: content.locale_name || code.toUpperCase()
    })
  }
}

const savedLocale = localStorage.getItem('boyan_locale')
const defaultLocale = (savedLocale && messages[savedLocale])
  ? savedLocale
  : (navigator.language.startsWith('ru') && messages['ru'] ? 'ru' : (messages['en'] ? 'en' : Object.keys(messages)[0]))

export const i18n = createI18n({
  legacy: false,
  locale: defaultLocale,
  fallbackLocale: 'en',
  messages,
})

export function setLocale(code: string) {
  if (messages[code]) {
    i18n.global.locale.value = code as any
    localStorage.setItem('boyan_locale', code)
    document.documentElement.setAttribute('lang', code)
  }
}
