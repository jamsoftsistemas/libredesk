// Converts plain-text URLs into clickable anchors for messages rendered via
// v-html. Only used for content_type === 'text' bodies - HTML content types
// go through vue-letter, which handles links on its own.

const URL_REGEX = /(https?:\/\/[^\s<]+)/gi

const ESCAPE_MAP = { '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;' }

const escapeHtml = (text) => text.replace(/[&<>"']/g, (ch) => ESCAPE_MAP[ch])

// Trim trailing punctuation that's almost always sentence punctuation, not
// part of the URL (e.g. "visit https://example.com." or "(see https://x.com)").
const TRAILING_PUNCTUATION = /[.,!?:;)\]'"]+$/

export const linkifyText = (text) => {
  const escaped = escapeHtml(text || '')
  return escaped.replace(URL_REGEX, (url) => {
    const trailingMatch = url.match(TRAILING_PUNCTUATION)
    const trailing = trailingMatch ? trailingMatch[0] : ''
    const href = trailing ? url.slice(0, -trailing.length) : url
    if (!href) return url
    return `<a href="${href}" target="_blank" rel="noopener noreferrer">${href}</a>${trailing}`
  })
}
