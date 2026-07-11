// Display naming for a user across the Portal. Precedence is
// username → name → caller fallback, so a person reads the same wherever
// they appear (entry author, comment author, avatar).
//
// The backend already merges every account type (Email → Custom → Google)
// into the EntryCreator wire shape (see api/shared.go's BuildEntryCreator),
// filling name/username from whichever account carries them. This helper
// only decides which of the two to show. Keep the precedence in lockstep
// with the Console copy. The fallback is passed in so it can be localised
// (e.g. t('common.anonymous')).

import type { ApiEntryCreator } from '@/services/api'

// userDisplayName resolves the label shown for a user. Username wins over
// name; when neither is set (both empty across all accounts) the caller's
// fallback is used.
export function userDisplayName(u: ApiEntryCreator | undefined | null, fallback: string): string {
  if (!u) return fallback
  return u.username || u.name || fallback
}

// userInitial is the single-letter avatar glyph, following the same
// precedence so the circle matches the label. Falls back to '?'.
export function userInitial(u: ApiEntryCreator | undefined | null): string {
  const label = u ? u.username || u.name || '' : ''
  return (label[0] || '?').toUpperCase()
}
