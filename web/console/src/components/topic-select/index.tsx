import { Menu, MenuButton, MenuItem, MenuItems } from '@headlessui/react'
import { Check, ChevronDown, Plus } from 'lucide-react'
import clsx from 'clsx'

import TopicBadge from '@/components/topic-badge'
import type { ApiEntryTopic } from '@/services/api'

// TopicSelect is the single-select picker for assigning ONE topic
// to an entry. The DB field is an array (room to grow), but the
// v0.1 UI enforces a single-pick contract here. Picking another
// topic replaces the current selection.
//
// onChange receives the new topicId or '' when the user clears.
// align controls dropdown positioning so the same component can
// render in a left-aligned form and a right-aligned filter bar.
export default function TopicSelect({
  topics,
  value,
  aiIds,
  onChange,
  disabled,
  placeholder = 'Add topic',
  align = 'left',
  emptyLabel = 'No topic',
}: {
  topics: ApiEntryTopic[] | undefined
  value: string
  aiIds?: string[]
  onChange: (id: string) => void
  disabled?: boolean
  placeholder?: string
  align?: 'left' | 'right'
  emptyLabel?: string
}) {
  const selected = topics?.find((t) => t.id === value)
  const isAIApplied = !!(value && aiIds && aiIds.includes(value))

  return (
    <div className="flex flex-wrap items-center gap-1.5">
      {selected && (
        <TopicBadge
          topic={selected}
          appliedByAI={isAIApplied}
          onRemove={disabled ? undefined : () => onChange('')}
        />
      )}
      <Menu as="div" className="relative">
        <MenuButton
          disabled={disabled || !topics}
          className="inline-flex items-center gap-1 rounded-full border border-dashed border-zinc-300 dark:border-zinc-600 px-2 py-0.5 text-xs text-zinc-600 dark:text-zinc-300 hover:bg-zinc-100 dark:hover:bg-zinc-800 disabled:opacity-50 disabled:cursor-not-allowed"
        >
          <Plus className="size-3" />
          {selected ? 'Change' : placeholder}
          <ChevronDown className="size-3" />
        </MenuButton>
        <MenuItems
          anchor={align === 'left' ? 'bottom start' : 'bottom end'}
          className="z-30 mt-1 w-64 origin-top rounded-md border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-900 shadow-lg ring-1 ring-black/5 focus:outline-none max-h-80 overflow-auto"
        >
          <div className="px-3 py-2 text-xs font-semibold text-zinc-500 dark:text-zinc-400 border-b border-zinc-100 dark:border-zinc-800">
            Topic
          </div>
          {(topics ?? []).length === 0 && (
            <div className="px-3 py-2 text-xs text-zinc-500">
              No topics configured.
            </div>
          )}
          {topics && topics.length > 0 && (
            <MenuItem>
              {({ focus }) => (
                <button
                  type="button"
                  onClick={() => onChange('')}
                  className={clsx(
                    'flex w-full items-center gap-2 px-3 py-2 text-left text-sm',
                    focus
                      ? 'bg-zinc-100 dark:bg-zinc-800'
                      : 'text-zinc-800 dark:text-zinc-100',
                  )}
                >
                  <Check
                    className={clsx(
                      'size-4 shrink-0',
                      value === '' ? 'opacity-100' : 'opacity-0',
                    )}
                  />
                  <span className="truncate flex-1 italic text-zinc-500">{emptyLabel}</span>
                </button>
              )}
            </MenuItem>
          )}
          {topics?.map((t) => {
            const isSelected = value === t.id
            return (
              <MenuItem key={t.id}>
                {({ focus }) => (
                  <button
                    type="button"
                    onClick={() => onChange(t.id)}
                    className={clsx(
                      'flex w-full items-center gap-2 px-3 py-2 text-left text-sm',
                      focus
                        ? 'bg-zinc-100 dark:bg-zinc-800'
                        : 'text-zinc-800 dark:text-zinc-100',
                    )}
                  >
                    <Check
                      className={clsx(
                        'size-4 shrink-0',
                        isSelected ? 'opacity-100' : 'opacity-0',
                      )}
                    />
                    <span
                      className="size-2 rounded-full shrink-0"
                      style={{ backgroundColor: t.color || '#71717a' }}
                    />
                    <span className="truncate flex-1">{t.title}</span>
                  </button>
                )}
              </MenuItem>
            )
          })}
        </MenuItems>
      </Menu>
    </div>
  )
}
