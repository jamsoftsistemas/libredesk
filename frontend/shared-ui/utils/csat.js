const RATING_EMOJI = { 1: '😢', 2: '😕', 3: '😊', 4: '😃', 5: '🤩' }
const RATING_TEXT_KEY = {
  1: 'globals.terms.poor',
  2: 'globals.terms.fair',
  3: 'globals.terms.good',
  4: 'globals.terms.great',
  5: 'globals.terms.excellent'
}

export const CSAT_RATING_VALUES = [1, 2, 3, 4, 5]

// overrides is the inbox's `config.csat_ratings`: either empty/absent (use defaults) or
// exactly 5 entries ({emoji, label}) ordered from rating 1 to 5.
function overrideFor (rating, overrides) {
  return Array.isArray(overrides) && overrides.length === 5 ? overrides[rating - 1] : null
}

export function csatRatingEmoji (rating, overrides) {
  const override = overrideFor(rating, overrides)
  return (override && override.emoji) || RATING_EMOJI[rating] || ''
}

// Returns the final display label (already translated/resolved), unlike the override's raw text.
export function csatRatingLabel (rating, overrides, t) {
  const override = overrideFor(rating, overrides)
  if (override && override.label) return override.label
  const key = RATING_TEXT_KEY[rating]
  return key ? t(key) : ''
}

// Builds the ordered {value, emoji, text} list the CSAT rating picker renders.
export function buildCSATRatings (overrides, t) {
  return CSAT_RATING_VALUES.map((value) => ({
    value,
    emoji: csatRatingEmoji(value, overrides),
    text: csatRatingLabel(value, overrides, t)
  }))
}
