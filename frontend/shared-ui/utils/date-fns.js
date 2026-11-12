// Wrapper around date-fns that applies the app locale (e.g. "out" instead of "oct")
// to every format call. Import from '@shared-ui/utils/date-fns' instead of 'date-fns'.
import {
  format as dfFormat,
  formatDistanceToNow as dfFormatDistanceToNow,
  formatDistance as dfFormatDistance
} from 'date-fns'
import { ptBR, enUS, es, fr, de } from 'date-fns/locale'

export * from 'date-fns'

const LOCALES = { pt: ptBR, en: enUS, es, fr, de }

const detect = () => {
  const lang = (typeof navigator !== 'undefined' && navigator.language) || 'pt-BR'
  return LOCALES[lang.slice(0, 2).toLowerCase()] || ptBR
}

let currentLocale = detect()

// Call with the app language (e.g. 'pt-BR'). Falls back to pt-BR.
export const setDateLocale = (lang) => {
  currentLocale = LOCALES[String(lang || '').slice(0, 2).toLowerCase()] || ptBR
}

export const getDateLocale = () => currentLocale

export const format = (date, fmt, options = {}) =>
  dfFormat(date, fmt, { locale: currentLocale, ...options })

export const formatDistanceToNow = (date, options = {}) =>
  dfFormatDistanceToNow(date, { locale: currentLocale, ...options })

export const formatDistance = (a, b, options = {}) =>
  dfFormatDistance(a, b, { locale: currentLocale, ...options })
