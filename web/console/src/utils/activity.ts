// Activity palette. Mirrors the Go enum + display metadata for each
// ActivityType emitted by the backend. Update this file when adding a
// new activity type to models/activity.go.
//
// `icon` names line up with lucide-react components; the consumer maps
// them by string so this module stays icon-agnostic.

import type { ApiActivityType } from '@/services/api'

export type ActivityInfo = {
  value: ApiActivityType
  // Short verb phrase used in the timeline row prefix, e.g.
  // "{actor} {verb}". For transitions ("changed status"), keep the verb
  // generic — the per-row component renders the from/to chips after it.
  verb: string
  icon: string
}

export const ACTIVITIES: ActivityInfo[] = [
  { value: 'entry-created', verb: 'created this feedback', icon: 'sparkles' },
  { value: 'status-changed', verb: 'changed status', icon: 'arrow-right' },
  { value: 'entry-type-changed', verb: 'changed entry type', icon: 'arrow-right' },
  { value: 'topic-added', verb: 'added topic', icon: 'tag' },
  { value: 'topic-removed', verb: 'removed topic', icon: 'tag' },
  { value: 'release-set', verb: 'assigned release', icon: 'package' },
  { value: 'release-cleared', verb: 'cleared release', icon: 'package' },
  { value: 'relation-added', verb: 'linked', icon: 'link-2' },
  { value: 'relation-removed', verb: 'unlinked', icon: 'unlink-2' },
  { value: 'internal-enabled', verb: 'marked as internal', icon: 'lock' },
  { value: 'internal-disabled', verb: 'marked as public', icon: 'eye' },
  { value: 'merged-into', verb: 'merged into another entry', icon: 'git-merge' },
  { value: 'github-issue-created', verb: 'created a GitHub issue', icon: 'github' },
]

const BY_VALUE: Record<string, ActivityInfo> = Object.fromEntries(
  ACTIVITIES.map((a) => [a.value, a]),
)

export function activityInfo(value: ApiActivityType): ActivityInfo {
  return BY_VALUE[value] ?? { value, verb: value, icon: 'circle' }
}
