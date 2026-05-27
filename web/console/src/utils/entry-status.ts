// Hardcoded entry status palette. Mirrors the Go enum in
// models/entry.go and the display metadata baked into api/shared.go's
// entryStatusDisplay map. Update all three in lockstep when adding or
// renaming a status.
//
// `closed` flags the terminal statuses — used to drive the row's
// closed-indicator dot on the entries list. Previously this lived on a
// per-state IsClosed flag; with states gone it's a property of the
// hardcoded enum.

import type { ApiEntryStatusValue } from '@/services/api'

export type EntryStatusInfo = {
  value: ApiEntryStatusValue
  title: string
  color: string
  icon: string
  closed: boolean
}

export const ENTRY_STATUSES: EntryStatusInfo[] = [
  { value: 'new', title: 'New', color: '#6366F1', icon: 'sparkles', closed: false },
  { value: 'evaluation', title: 'Evaluation', color: '#0EA5E9', icon: 'search', closed: false },
  { value: 'in-progress', title: 'In Progress', color: '#F59E0B', icon: 'loader', closed: false },
  { value: 'completed', title: 'Completed', color: '#10B981', icon: 'check', closed: true },
  { value: 'cancelled', title: 'Cancelled', color: '#E11D48', icon: 'x', closed: true },
]

const BY_VALUE: Record<string, EntryStatusInfo> = Object.fromEntries(
  ENTRY_STATUSES.map((s) => [s.value, s]),
)

export function entryStatusInfo(value: string | undefined): EntryStatusInfo {
  if (value && BY_VALUE[value]) return BY_VALUE[value]
  return ENTRY_STATUSES[0]
}
