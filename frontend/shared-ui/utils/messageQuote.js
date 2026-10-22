import { convertTextToHtml, getTextFromHTML } from './string.js'

const SNIPPET_MAX_LENGTH = 120

// Single-line plain-text preview shown in the WhatsApp-style quote chip above the composer.
export function buildQuoteSnippet(content, contentType) {
  const text = contentType === 'text' ? content || '' : getTextFromHTML(content || '')
  const collapsed = text.replace(/\s+/g, ' ').trim()
  if (collapsed.length <= SNIPPET_MAX_LENGTH) return collapsed
  return `${collapsed.slice(0, SNIPPET_MAX_LENGTH - 1).trimEnd()}…`
}

// Builds the compact, WhatsApp-style quote block merged into the outgoing message when an
// agent quotes another message. Kept the same across channels since they all go through the
// shared rich-text composer.
export function buildQuoteHtml({ authorName, content, contentType }) {
  const body = contentType === 'text' ? convertTextToHtml(content || '') : content || ''
  return (
    `<blockquote class="quoted-message" style="margin:0 0 8px;padding:1px 0 1px 10px;border-left:3px solid currentColor;opacity:0.7;">` +
    (authorName ? `<p style="margin:0;">${convertTextToHtml(authorName)}</p>` : '') +
    body +
    `</blockquote>`
  )
}
