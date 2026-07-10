import { useEffect } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { ArrowLeft, ChevronRight, FileText, Link as LinkIcon } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import Markdown from '@/components/markdown'
import StatusBadge from '@/components/status-badge'
import { API } from '@/services/api'
import { message } from '@/utils/helpers'

// ChangelogDetailPage renders one release on its own URL so it can
// be shared from social media. Slug accepts version name or release
// ID; the backend resolves both.
export default function ChangelogDetailPage() {
  const { t, i18n } = useTranslation()
  const { slug = '' } = useParams<{ slug: string }>()

  const { data: release, isLoading, error } = useQuery({
    queryKey: ['portal', 'changelog', slug],
    queryFn: () => API().portal.getChangelogRelease(slug),
    enabled: !!slug,
  })

  useEffect(() => {
    if (release) {
      const suffix = release.title ? ` — ${release.title}` : ''
      document.title = `${release.versionName}${suffix} · ${t('changelog.titleSuffix')}`
    }
    return () => {
      document.title = t('changelog.titleSuffix')
    }
  }, [release, t])

  const handleCopyLink = async () => {
    try {
      await navigator.clipboard.writeText(window.location.href)
      message(t('changelog.copied'), 'success')
    } catch (err) {
      message(err)
    }
  }

  if (isLoading) {
    return <p className="text-sm text-zinc-500">{t('common.loading')}</p>
  }

  if (error || !release) {
    return (
      <div className="space-y-4">
        <Link
          to="/changelog"
          className="inline-flex items-center gap-2 text-sm text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200"
        >
          <ArrowLeft className="size-4" />
          {t('changelog.backToChangelog')}
        </Link>
        <div className="rounded-lg border border-dashed border-zinc-300 dark:border-zinc-700 p-12 text-center">
          <h2 className="text-lg font-semibold">{t('changelog.notFoundTitle')}</h2>
          <p className="mt-2 text-sm text-zinc-500">
            {(error as Error | undefined)?.message ?? t('changelog.notFoundBody')}
          </p>
        </div>
      </div>
    )
  }

  return (
    <article className="space-y-6">
      <Link
        to="/changelog"
        className="inline-flex items-center gap-2 text-sm text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200"
      >
        <ArrowLeft className="size-4" />
        {t('changelog.backToChangelog')}
      </Link>

      <header className="space-y-3">
        <div className="flex flex-wrap items-baseline gap-3">
          <h1 className="text-3xl font-semibold text-zinc-900 dark:text-zinc-50">
            {release.versionName}
          </h1>
          <span
            className={
              release.state === 'completed'
                ? 'rounded px-1.5 py-0.5 text-[10px] uppercase tracking-wider bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
                : 'rounded px-1.5 py-0.5 text-[10px] uppercase tracking-wider bg-sky-100 text-sky-700 dark:bg-sky-500/15 dark:text-sky-300'
            }
          >
            {release.state === 'completed'
              ? t('changelog.stateReleased')
              : t('changelog.statePlanned')}
          </span>
          <span className="text-sm text-zinc-500 dark:text-zinc-400">
            {formatLongDate(release.releaseDate, i18n.language, t('common.noDate'))}
          </span>
          <button
            type="button"
            onClick={handleCopyLink}
            className="ml-auto inline-flex items-center gap-1 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-2 py-1 text-xs text-zinc-600 dark:text-zinc-300 hover:bg-zinc-50 dark:hover:bg-zinc-800"
            title={t('changelog.copyLink')}
          >
            <LinkIcon className="size-3.5" />
            {t('changelog.copyLink')}
          </button>
        </div>
        {release.title && (
          <h2 className="text-xl font-medium text-zinc-700 dark:text-zinc-300">
            {release.title}
          </h2>
        )}
      </header>

      {release.description && (
        <Markdown className="text-zinc-700 dark:text-zinc-300">
          {release.description}
        </Markdown>
      )}

      {release.pdfFileUrl && (
        <a
          href={release.pdfFileUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-flex items-center gap-2 rounded-md border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 px-3 py-2 text-sm text-zinc-700 dark:text-zinc-200 hover:bg-zinc-50 dark:hover:bg-zinc-800"
        >
          <FileText className="size-4 text-zinc-500" />
          <span className="font-medium">
            {release.pdfFileName || t('changelog.downloadPdf')}
          </span>
          {release.pdfFileSize ? (
            <span className="text-xs text-zinc-500 tabular-nums">
              {formatBytes(release.pdfFileSize)}
            </span>
          ) : null}
        </a>
      )}

      {release.entries.length > 0 && (
        <section className="space-y-2">
          <h2 className="text-xs font-semibold uppercase tracking-wider text-zinc-500 dark:text-zinc-400">
            {t('changelog.included')}
          </h2>
          <div className="rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 divide-y divide-zinc-200 dark:divide-zinc-800">
            {release.entries.map((e) => (
              <Link
                key={e.id}
                to={`/entry/${e.id}`}
                className="flex items-start gap-3 px-4 py-3 hover:bg-zinc-50 dark:hover:bg-zinc-950/40"
              >
                <div className="min-w-0 flex-1">
                  <div className="font-medium text-zinc-900 dark:text-zinc-100">
                    {e.title}
                  </div>
                  {e.description && (
                    <p className="mt-0.5 line-clamp-2 text-sm text-zinc-500">
                      {plainPreview(e.description)}
                    </p>
                  )}
                </div>
                <div className="mt-0.5 shrink-0">
                  <StatusBadge status={e.status} />
                </div>
                <ChevronRight className="size-4 shrink-0 text-zinc-400" />
              </Link>
            ))}
          </div>
        </section>
      )}

      {release.entries.length === 0 && !release.description && (
        <p className="text-sm text-zinc-500">
          {release.state === 'completed'
            ? t('changelog.noNotesCompleted')
            : t('changelog.noNotesPlanned')}
        </p>
      )}
    </article>
  )
}

function formatLongDate(isoDate: string | undefined, locale: string, fallback: string): string {
  if (!isoDate) return fallback
  const [y, m, d] = isoDate.split('-').map(Number)
  if (!y || !m || !d) return isoDate
  const date = new Date(Date.UTC(y, m - 1, d))
  return date.toLocaleDateString(locale, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  })
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}

function plainPreview(s: string): string {
  return s
    .replace(/```[\s\S]*?```/g, '')
    .replace(/`([^`]+)`/g, '$1')
    .replace(/!\[[^\]]*\]\([^)]*\)/g, '')
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(/[\*_~#>]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
}
