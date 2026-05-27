import { Menu, MenuButton, MenuItem, MenuItems } from '@headlessui/react'
import { Check, ChevronDown, Globe } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import clsx from 'clsx'

import { SUPPORTED_LANGUAGES, type SupportedLanguage } from '@/utils/i18n'

// Header-level language switcher. Renders a Globe icon + the active
// language code; the dropdown lists every supported language with the
// current one checkmarked. Selection writes to localStorage via
// i18next's detector and broadcasts `languageChanged`, which the
// dayjs-sync hook in utils/i18n.ts picks up.
export default function LanguagePicker() {
  const { t, i18n } = useTranslation()
  const current = (i18n.resolvedLanguage || i18n.language || 'en').split('-')[0]

  return (
    <Menu as="div" className="relative">
      <MenuButton
        title={t('common.language')}
        className="inline-flex items-center gap-1 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-2 h-8 text-xs font-medium text-zinc-700 dark:text-zinc-200 hover:bg-zinc-50 dark:hover:bg-zinc-800 transition-colors"
      >
        <Globe className="size-4" />
        <span className="uppercase">{current}</span>
        <ChevronDown className="size-3 text-zinc-400" />
      </MenuButton>
      <MenuItems
        anchor="bottom end"
        className="z-50 mt-1 min-w-[10rem] rounded-md border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 py-1 text-sm shadow-lg focus:outline-none"
      >
        {SUPPORTED_LANGUAGES.map((lng) => (
          <MenuItem key={lng}>
            <button
              type="button"
              onClick={() => void i18n.changeLanguage(lng)}
              className={clsx(
                'flex w-full items-center gap-2 px-3 py-1.5 text-left transition-colors',
                'data-[focus]:bg-zinc-100 dark:data-[focus]:bg-zinc-800',
                current === lng
                  ? 'text-zinc-900 dark:text-zinc-100'
                  : 'text-zinc-600 dark:text-zinc-300',
              )}
            >
              <span className="flex-1">{t(`language.${lng}` as Lng)}</span>
              {current === lng && <Check className="size-4 text-brand-600 dark:text-brand-400" />}
            </button>
          </MenuItem>
        ))}
      </MenuItems>
    </Menu>
  )
}

type Lng = `language.${SupportedLanguage}`
