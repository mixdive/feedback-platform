import { useState } from 'react'
import { Popover, PopoverButton, PopoverPanel } from '@headlessui/react'
import { Search, Settings, Sparkles } from 'lucide-react'
import clsx from 'clsx'

import type { ApiEntryTopic } from '@/services/api'

// TopicMultiSelect is the GitHub-Issues-style multi-select label
// picker for the Console entry-detail sidebar. The card shows the
// currently-assigned topics as colored chips; a gear button opens
// a filterable checklist popover that lets the admin add or remove
// topics. Each checkbox toggle fires onChange with the next full
// list of topic IDs.
export default function TopicMultiSelect({
  topics,
  value,
  aiIds,
  onChange,
  disabled,
}: {
  topics: ApiEntryTopic[] | undefined
  value: string[]
  aiIds?: string[]
  onChange: (ids: string[]) => void
  disabled?: boolean
}) {
  const [filter, setFilter] = useState('')
  const byId = new Map((topics ?? []).map((t) => [t.id, t]))
  const selected = value
    .map((id) => byId.get(id))
    .filter((t): t is ApiEntryTopic => !!t)
  const q = filter.trim().toLowerCase()
  const filtered = (topics ?? []).filter(
    (t) =>
      !q ||
      t.title.toLowerCase().includes(q) ||
      (t.description ?? '').toLowerCase().includes(q),
  )

  const toggle = (id: string) => {
    if (value.includes(id)) onChange(value.filter((v) => v !== id))
    else onChange([...value, id])
  }

  return (
    <div className="rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-3">
      <Popover className="relative">
        <div className="flex items-center justify-between gap-2">
          <span className="text-xs font-medium uppercase tracking-wide text-zinc-500">
            Topic
          </span>
          <PopoverButton
            disabled={disabled || !topics}
            aria-label="Manage topics"
            className="inline-flex items-center justify-center rounded p-1 text-zinc-500 hover:bg-zinc-100 hover:text-zinc-800 dark:hover:bg-zinc-800 dark:hover:text-zinc-200 disabled:opacity-50 disabled:cursor-not-allowed"
          >
            <Settings className="size-4" />
          </PopoverButton>
        </div>
        <div className="mt-2">
          {selected.length === 0 ? (
            <span className="text-sm italic text-zinc-500">None yet</span>
          ) : (
            <div className="flex flex-wrap gap-1.5">
              {selected.map((t) => {
                const color = t.color || '#71717a'
                const isAI = aiIds?.includes(t.id)
                return (
                  <span
                    key={t.id}
                    className="inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium"
                    style={{
                      borderColor: color + '66',
                      backgroundColor: color + '1a',
                      color,
                    }}
                  >
                    <span
                      className="size-1.5 rounded-full shrink-0"
                      style={{ backgroundColor: color }}
                    />
                    {t.title}
                    {isAI && <Sparkles className="size-3 shrink-0" aria-label="Applied by AI" />}
                  </span>
                )
              })}
            </div>
          )}
        </div>
        <PopoverPanel
          anchor={{ to: 'bottom end', gap: 4 }}
          className="z-30 w-80 origin-top rounded-md border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-900 shadow-lg ring-1 ring-black/5 focus:outline-none"
        >
          <div className="px-3 py-2 text-xs font-semibold text-zinc-500 dark:text-zinc-400 border-b border-zinc-100 dark:border-zinc-800">
            Apply topics to this entry
          </div>
          <div className="border-b border-zinc-100 dark:border-zinc-800 p-2">
            <div className="relative">
              <Search className="pointer-events-none absolute left-2 top-1/2 size-4 -translate-y-1/2 text-zinc-400" />
              <input
                type="text"
                autoFocus
                value={filter}
                onChange={(e) => setFilter(e.target.value)}
                placeholder="Filter topics"
                className="block w-full h-9 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 pl-8 pr-3 text-sm focus:border-sky-500 focus:outline-none"
              />
            </div>
          </div>
          <div className="max-h-72 overflow-auto">
            {(topics ?? []).length === 0 && (
              <div className="px-3 py-3 text-xs text-zinc-500">No topics configured.</div>
            )}
            {(topics ?? []).length > 0 && filtered.length === 0 && (
              <div className="px-3 py-3 text-xs text-zinc-500">No matches.</div>
            )}
            {filtered.map((t) => {
              const isSelected = value.includes(t.id)
              return (
                <button
                  key={t.id}
                  type="button"
                  onClick={() => toggle(t.id)}
                  className={clsx(
                    'flex w-full items-start gap-2 px-3 py-2 text-left text-sm border-b border-zinc-50 last:border-b-0 dark:border-zinc-800/50',
                    'hover:bg-zinc-50 dark:hover:bg-zinc-800/60',
                  )}
                >
                  <input
                    type="checkbox"
                    checked={isSelected}
                    readOnly
                    tabIndex={-1}
                    className="mt-0.5 size-4 shrink-0 accent-sky-600"
                  />
                  <span
                    className="mt-1 size-2.5 rounded-full shrink-0"
                    style={{ backgroundColor: t.color || '#71717a' }}
                  />
                  <span className="min-w-0 flex-1">
                    <span className="block font-semibold text-zinc-900 dark:text-zinc-100">
                      {t.title}
                    </span>
                    {t.description && (
                      <span className="block text-xs text-zinc-500">{t.description}</span>
                    )}
                  </span>
                </button>
              )
            })}
          </div>
        </PopoverPanel>
      </Popover>
    </div>
  )
}
