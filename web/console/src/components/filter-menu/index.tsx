import { Menu, MenuButton, MenuItem, MenuItems } from '@headlessui/react'
import { Check, ChevronDown } from 'lucide-react'
import clsx from 'clsx'
import type { ReactNode } from 'react'

export type FilterOption = {
  id: string
  label: string
  hint?: string
  icon?: ReactNode
}

export default function FilterMenu({
  label,
  options,
  value,
  onChange,
  align = 'right',
}: {
  label: string
  options: FilterOption[]
  value: string
  onChange: (id: string) => void
  align?: 'left' | 'right'
}) {
  return (
    <Menu as="div" className="relative">
      <MenuButton className="inline-flex items-center gap-1 text-sm text-zinc-600 dark:text-zinc-300 hover:text-zinc-900 dark:hover:text-zinc-50 focus:outline-none">
        {label}
        <ChevronDown className="size-3.5" />
      </MenuButton>
      <MenuItems
        anchor={align === 'left' ? 'bottom start' : 'bottom end'}
        className="z-30 mt-1 w-64 origin-top rounded-md border border-zinc-200 dark:border-zinc-700 bg-white dark:bg-zinc-900 shadow-lg ring-1 ring-black/5 focus:outline-none max-h-80 overflow-auto"
      >
        <div className="px-3 py-2 text-xs font-semibold text-zinc-500 dark:text-zinc-400 border-b border-zinc-100 dark:border-zinc-800">
          Filter by {label.toLowerCase()}
        </div>
        {options.map((o) => {
          const selected = o.id === value
          return (
            <MenuItem key={o.id || '__empty__'}>
              {({ focus }) => (
                <button
                  type="button"
                  onClick={() => onChange(o.id)}
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
                      selected ? 'opacity-100' : 'opacity-0',
                    )}
                  />
                  {o.icon && <span className="shrink-0">{o.icon}</span>}
                  <span className="truncate flex-1">{o.label}</span>
                  {o.hint && (
                    <span className="text-xs text-zinc-500 dark:text-zinc-400 shrink-0">
                      {o.hint}
                    </span>
                  )}
                </button>
              )}
            </MenuItem>
          )
        })}
      </MenuItems>
    </Menu>
  )
}
