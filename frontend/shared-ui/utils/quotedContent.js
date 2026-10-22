// Runs against the RAW HTML stored on the message - before vue-letter
// renders. vue-letter prefixes every id/class with a random "msg_XYZ_" at
// render time, so any CSS that needs to match the rendered DOM must use
// attribute-suffix selectors (see .hide-quoted-text in main.scss).
//
// Scope: <blockquote> covers Gmail/Apple/Thunderbird *replies*. Gmail
// *forwards* wrap content in <div class="gmail_quote gmail_quote_container">
// instead. The id/class markers below cover the Microsoft variants
// (Outlook desktop/web/mac, Hotmail) and Yahoo Mail.
export const QUOTE_MARKERS = [
  '<blockquote',
  'id="divRplyFwdMsg"',
  'id="appendonsend"',
  'id="OLK_SRC_BODY_SECTION"',
  'class="OutlookMessageHeader"',
  'class="yahoo_quoted"',
  'gmail_quote_container'
]

// Our own "cited message" feature (agents/visitors quoting an earlier message from the
// timeline) wraps its content in a <blockquote class="quoted-message">. That is not an
// email thread quote, so it must not trigger the show/hide-quoted-text toggle - strip it out
// before testing for the real markers below.
const CITED_MESSAGE_BLOCKQUOTE = /<blockquote\b[^>]*\bclass="[^"]*\bquoted-message\b[^"]*"[^>]*>[\s\S]*?<\/blockquote>/gi

export const containsQuoteMarkers = (html) => {
  if (!html) return false
  const withoutCitedMessages = html.replace(CITED_MESSAGE_BLOCKQUOTE, '')
  return QUOTE_MARKERS.some((marker) => withoutCitedMessages.includes(marker))
}
