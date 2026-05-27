import { Sparkles } from 'lucide-react'

import type { ApiEntryTopic } from '@/services/api'

// TopicBadge renders a small colored chip for one topic. Topics
// are console-only (never rendered on the Portal).
//
// appliedByAI marks per-entry assignment provenance — separate from
// ApiEntryTopic.source, which marks topics whose record was
// AI-created (the Settings list uses that one). A topic can be
// admin-created but AI-assigned to a specific entry, or vice versa.
export default function TopicBadge({
  topic,
  onRemove,
  appliedByAI,
  size = 'md',
}: {
  topic: Pick<ApiEntryTopic, 'title' | 'color'>
  onRemove?: () => void
  appliedByAI?: boolean
  size?: 'sm' | 'md'
}) {
  const color = topic.color || '#71717a'
  const sizeClass =
    size === 'sm' ? 'text-[10px] px-1.5 py-0.5' : 'text-xs px-2 py-0.5'
  return (
    <span
      className={
        'inline-flex items-center gap-1 rounded-full border font-medium ' + sizeClass
      }
      style={{
        borderColor: color + '66',
        backgroundColor: color + '1a',
        color,
      }}
    >
      <span className="size-1.5 rounded-full shrink-0" style={{ backgroundColor: color }} />
      {topic.title}
      {appliedByAI && (
        <Sparkles
          className={size === 'sm' ? 'size-2.5 shrink-0' : 'size-3 shrink-0'}
          aria-label="Applied by AI"
        />
      )}
      {onRemove && (
        <button
          type="button"
          onClick={onRemove}
          aria-label={`Remove ${topic.title}`}
          className="ml-0.5 inline-flex items-center justify-center rounded-full hover:bg-black/10 dark:hover:bg-white/10"
          style={{ color }}
        >
          <span aria-hidden className="text-[12px] leading-none px-0.5">×</span>
        </button>
      )}
    </span>
  )
}
