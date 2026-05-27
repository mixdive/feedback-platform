import { useTranslation } from 'react-i18next'

export default function Splash() {
  const { t } = useTranslation()
  return (
    <div className="flex h-full w-full flex-col items-center justify-center gap-4 bg-zinc-50 dark:bg-zinc-950">
      <svg
        viewBox="0 0 60 46"
        role="img"
        aria-label="Mixdive"
        className="size-12 animate-pulse"
      >
        <title>Mixdive</title>
        <rect x="20" y="0" width="20" height="10" rx="5" fill="#0B2545" />
        <rect x="10" y="18" width="40" height="10" rx="5" fill="#0B2545" />
        <rect x="0" y="36" width="60" height="10" rx="5" fill="#2EC4B6" />
      </svg>
      <div className="text-sm text-zinc-500 dark:text-zinc-400">{t('common.loading')}</div>
    </div>
  )
}
