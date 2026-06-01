import { useEffect, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import {
  ArrowLeft,
  Bug,
  ChevronUp,
  ExternalLink,
  Lightbulb,
  Lock,
  MessageSquare,
  Sparkles,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import dayjs from 'dayjs'
import clsx from 'clsx'

import Button from '@/components/button'
import { markdownToPlainText } from '@/components/markdown'
import MarkdownEditor from '@/components/markdown-editor'
import StatusBadge from '@/components/status-badge'
import {
  API,
  type ApiEntry,
  type ApiEntryTypeValue,
  type ApiEntryCreator,
  type ApiFindSimilarMatch,
} from '@/services/api'
import { authResolved } from '@/store'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { useSiteConfig } from '@/store/site/hooks'
import { message } from '@/utils/helpers'
import { entryTypeInfo } from '@/utils/entry-type'

type AllowedNewType = 'feature-request' | 'bug'

const TITLE_MAX = 100

// Two-step flow surfaced in the UI:
//   step 1 → title + type pick (the user's first decisions).
//   step 2 → description + submit (the full details).
// The 'matches' phase is an interstitial between them and is not
// counted as its own step.
type Phase = 'title' | 'matches' | 'form'

function creatorLabel(c: ApiEntryCreator | undefined, fallback: string): string {
  if (!c) return fallback
  return c.name || c.username || fallback
}

// DEFAULT_TEMPLATE_LANGUAGE mirrors the backend's
// models.DefaultTemplateLanguage — when the visitor's active language
// has no template for the chosen entry type, we render this
// language's copy before falling back to the per-type placeholder.
const DEFAULT_TEMPLATE_LANGUAGE = 'en'

export default function NewEntryPage() {
  const { t, i18n } = useTranslation()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const dispatch = useAppDispatch()
  const me = useAppSelector((s) => s.auth.user)
  const siteConfig = useSiteConfig()

  const [phase, setPhase] = useState<Phase>('title')
  const [entryType, setEntryType] = useState<AllowedNewType>('feature-request')
  const [title, setTitle] = useState('')
  const [description, setDescription] = useState('')
  // Tracks whether the user has edited the description away from the
  // pre-filled template. Once true, we never re-apply the template on
  // phase changes (so navigating between matches → form → matches does
  // not stomp the user's work).
  const [descriptionTouched, setDescriptionTouched] = useState(false)
  const [isInternal, setIsInternal] = useState(false)
  const [matches, setMatches] = useState<ApiFindSimilarMatch[]>([])

  // Active language for template selection. resolvedLanguage handles
  // BCP-47 region tags ("en-US" → "en"); fall back to "en" if i18next
  // hasn't initialized yet for some reason.
  const activeLang = (
    i18n.resolvedLanguage ||
    i18n.language ||
    DEFAULT_TEMPLATE_LANGUAGE
  ).split('-')[0]

  // Admin-configured per-(entry-type, language) markdown template.
  // Tries the visitor's active language first, falls back to the
  // default-language template, then to empty (which means the
  // description field shows the per-type placeholder copy).
  const perLang = siteConfig?.entryTypeTemplates?.[entryType]
  const template =
    perLang?.[activeLang] ?? perLang?.[DEFAULT_TEMPLATE_LANGUAGE] ?? ''

  // Apply the per-entry-type template the first time we reach the form
  // phase, only if the user hasn't touched the description yet.
  useEffect(() => {
    if (phase !== 'form') return
    if (descriptionTouched) return
    if (description !== '') return
    if (!template) return
    setDescription(template)
  }, [phase, descriptionTouched, description, template])

  // If the user switches the entry type OR the active language while
  // still on the title page and the description is still untouched,
  // drop any template carry-over from a previous pick so the
  // form-phase effect can reapply the right template cleanly.
  useEffect(() => {
    if (descriptionTouched) return
    setDescription('')
  }, [entryType, activeLang, descriptionTouched])

  // Visibility toggle is admin/editor-only — the option stays invisible
  // to regular portal users by design.
  const canSetInternal =
    !!me && me.roles.some((r) => r === 'admin' || r === 'editor')

  // Block submit when the caller has hit their per-user open
  // feature-request cap.
  const atFeatureRequestLimit =
    entryType === 'feature-request' &&
    !!me &&
    !me.featureRequestQuota.unlimited &&
    me.featureRequestQuota.used >= me.featureRequestQuota.max

  const findSimilarMut = useMutation({
    mutationFn: (body: { title: string; description?: string }) =>
      API().portal.findSimilar(body),
    onSuccess: (res) => {
      setMatches(res.data)
      setPhase(res.data.length > 0 ? 'matches' : 'form')
    },
    onError: (e) => message(e),
  })

  const submitMut = useMutation({
    mutationFn: (body: {
      title: string
      description?: string
      entryType?: ApiEntryTypeValue
      isInternal?: boolean
    }) => API().portal.submit(body),
    onSuccess: (entry) => {
      message(t('new.submitted'), 'success')
      void queryClient.invalidateQueries({ queryKey: ['portal', 'entries'] })
      API()
        .auth.me()
        .then((res) => dispatch(authResolved(res.user)))
        .catch(() => {})
      navigate(`/entry/${entry.id}`)
    },
    onError: (e) => message(e),
  })

  const voteMut = useMutation({
    mutationFn: (id: string) => API().portal.addVote(id),
    onSuccess: (updated) => {
      setMatches((prev) =>
        prev.map((m) =>
          m.entry.id === updated.id
            ? { ...m, entry: { ...m.entry, isVoted: updated.isVoted, voteCount: updated.voteCount } }
            : m,
        ),
      )
      void queryClient.invalidateQueries({ queryKey: ['portal', 'entries'] })
    },
    onError: (e) => message(e),
  })

  const continueFromTitle = () => {
    const tr = title.trim()
    if (tr.length < 3) return
    findSimilarMut.mutate({ title: tr })
  }

  const submit = () => {
    const tr = title.trim()
    if (tr.length < 3) return
    submitMut.mutate({
      title: tr,
      description: description.trim() || undefined,
      entryType,
      isInternal: canSetInternal && isInternal ? true : undefined,
    })
  }

  const handleUpvote = (entry: ApiEntry) => {
    if (!me) {
      message(t('new.signInToVote'), 'info')
      return
    }
    if (entry.isVoted) return
    voteMut.mutate(entry.id)
  }

  const typeInfo = entryTypeInfo(entryType)
  const typeTitle = typeInfo ? t(typeInfo.titleKey) : ''
  const formHeading = typeInfo
    ? t('new.formHeading', { type: typeTitle })
    : t('new.formHeadingFallback')
  const descriptionPlaceholder = t(`new.descriptionPlaceholder.${entryType}` as const)
  const formPhaseCopy = t(`new.formPhase.${entryType}` as const)
  const stepNumber = phase === 'form' ? 2 : 1

  return (
    <div className="space-y-6">
      <div>
        <Link
          to={phase === 'title' ? '/' : '#'}
          onClick={(e) => {
            if (phase !== 'title') {
              e.preventDefault()
              setPhase('title')
            }
          }}
          className="inline-flex items-center gap-1 text-sm text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200"
        >
          <ArrowLeft className="size-4" />
          {phase === 'title' ? t('new.stepBack') : t('new.stepChangeTitle')}
        </Link>
      </div>

      {phase === 'title' && (
        <div className="space-y-4">
          <div>
            <StepIndicator current={stepNumber} />
            <h1 className="mt-2 text-2xl font-semibold">{t('new.titleHeading')}</h1>
            <p className="mt-1 text-sm text-zinc-500">{t('new.titleSubtitle')}</p>
          </div>
          <form
            onSubmit={(e) => {
              e.preventDefault()
              continueFromTitle()
            }}
            className="space-y-4 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-5"
          >
            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-2">
                {t('new.fieldType')}
              </label>
              <TypeSegmentedControl value={entryType} onChange={setEntryType} />
            </div>
            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-2">
                {t('new.fieldShortSummary')}
              </label>
              <input
                value={title}
                onChange={(e) => setTitle(e.target.value.slice(0, TITLE_MAX))}
                maxLength={TITLE_MAX}
                placeholder={t('new.placeholderShortSummary')}
                autoFocus
                className="block w-full h-12 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-base focus:border-sky-500 focus:outline-none"
              />
              <div className="mt-1 text-right text-xs text-zinc-400">
                {title.length} / {TITLE_MAX}
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <Button type="button" variant="ghost" onClick={() => navigate('/')}>
                {t('common.cancel')}
              </Button>
              <Button
                type="submit"
                isLoading={findSimilarMut.isPending}
                disabled={title.trim().length < 3}
              >
                {t('common.continue')}
              </Button>
            </div>
          </form>
        </div>
      )}

      {phase === 'matches' && (
        <div className="space-y-4">
          <div className="flex items-center gap-2">
            <Sparkles className="size-5 text-sky-500" />
            <h1 className="text-xl font-semibold">{t('new.matchesHeading')}</h1>
          </div>
          <p className="text-sm text-zinc-500">
            {t('new.matchesIntro', { count: matches.length, query: title.trim() })}
          </p>

          <ul className="space-y-3">
            {matches.map(({ entry, reason }) => (
              <li
                key={entry.id}
                className="rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-4"
              >
                <div className="flex items-start gap-3">
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-medium">{entry.title}</span>
                      {entry.status && <StatusBadge status={entry.status} />}
                    </div>
                    {entry.description && (
                      <p className="mt-1 line-clamp-2 text-sm text-zinc-600 dark:text-zinc-400">
                        {markdownToPlainText(entry.description)}
                      </p>
                    )}
                    {reason && (
                      <p className="mt-1 text-xs text-sky-700 dark:text-sky-300">
                        {reason}
                      </p>
                    )}
                    <div className="mt-2 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-zinc-500">
                      <span className="inline-flex items-center gap-1">
                        <ChevronUp className="size-3.5" />
                        {entry.voteCount} {t('entryDetail.voteCountLabel', { count: entry.voteCount })}
                      </span>
                      <span>·</span>
                      <span className="inline-flex items-center gap-1">
                        <MessageSquare className="size-3.5" />
                        {entry.commentCount}
                      </span>
                      <span>·</span>
                      <span>{t('common.byUser', { name: creatorLabel(entry.creator, t('common.anonymous')) })}</span>
                      <span>·</span>
                      <span>{dayjs(entry.createdAt).format('LL')}</span>
                    </div>
                  </div>
                  <div className="flex shrink-0 flex-col gap-2">
                    <Button
                      type="button"
                      variant="primary"
                      onClick={() => handleUpvote(entry)}
                      isLoading={voteMut.isPending && voteMut.variables === entry.id}
                      className="!h-8 !px-3"
                      disabled={entry.isVoted}
                    >
                      <ChevronUp className="size-4" />
                      {entry.isVoted ? t('new.matchesVoted') : t('new.matchesVote')}
                    </Button>
                    <a
                      href={`/entry/${entry.id}`}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center justify-center gap-1.5 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 h-8 px-3 text-sm text-zinc-700 dark:text-zinc-200 hover:bg-zinc-50 dark:hover:bg-zinc-800 transition-colors"
                    >
                      <ExternalLink className="size-3.5" />
                      {t('new.matchesOpen')}
                    </a>
                  </div>
                </div>
              </li>
            ))}
          </ul>

          <div className="rounded-lg border border-dashed border-zinc-300 dark:border-zinc-700 p-5 text-center">
            <p className="text-sm text-zinc-600 dark:text-zinc-300">
              {t('new.noneMatch')}
            </p>
            <Button
              type="button"
              variant="primary"
              onClick={() => setPhase('form')}
              className="mt-3"
            >
              {t('new.createNew')}
            </Button>
          </div>
        </div>
      )}

      {phase === 'form' && (
        <div className="space-y-4">
          <div>
            <StepIndicator current={stepNumber} />
            <h1 className="mt-2 text-2xl font-semibold">{formHeading}</h1>
            <p className="mt-1 text-sm text-zinc-500">{formPhaseCopy}</p>
          </div>

          <form
            onSubmit={(e) => {
              e.preventDefault()
              submit()
            }}
            className="space-y-5 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-5"
          >
            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-2">
                {t('new.fieldTitle')}
              </label>
              <input
                value={title}
                onChange={(e) => setTitle(e.target.value.slice(0, TITLE_MAX))}
                maxLength={TITLE_MAX}
                placeholder={t('new.placeholderShortSummary')}
                className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none"
              />
            </div>

            {entryType === 'feature-request' &&
              me &&
              !me.featureRequestQuota.unlimited &&
              me.featureRequestQuota.max > 0 && (
                <FeatureRequestQuotaHint
                  used={me.featureRequestQuota.used}
                  max={me.featureRequestQuota.max}
                />
              )}

            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-2">
                {t('new.fieldDescription')}
              </label>
              <MarkdownEditor
                value={description}
                onChange={(next) => {
                  setDescription(next)
                  if (!descriptionTouched) setDescriptionTouched(true)
                }}
                placeholder={descriptionPlaceholder}
                rows={6}
                upload={
                  siteConfig?.uploadsEnabled
                    ? (file) => API().portal.uploadFile(file)
                    : undefined
                }
              />
            </div>

            {canSetInternal && (
              <div>
                <label className="block text-xs font-medium text-zinc-500 mb-2">
                  {t('new.fieldVisibility')}
                </label>
                <label className="flex items-start gap-3 rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-900/40 p-3 cursor-pointer">
                  <input
                    type="checkbox"
                    checked={isInternal}
                    onChange={(e) => setIsInternal(e.target.checked)}
                    className="mt-0.5 size-4 rounded border-zinc-300 dark:border-zinc-700 text-amber-600 focus:ring-amber-500"
                  />
                  <span className="text-sm">
                    <span className="inline-flex items-center gap-1 font-medium text-zinc-900 dark:text-zinc-100">
                      <Lock className="size-3.5" />
                      {t('new.internalLabel')}
                    </span>
                    <span className="block text-xs text-zinc-500">
                      {t('new.internalDescription')}
                    </span>
                  </span>
                </label>
              </div>
            )}

            <div className="flex items-center justify-end gap-2">
              <Button type="button" variant="ghost" onClick={() => navigate('/')}>
                {t('common.cancel')}
              </Button>
              <Button
                type="submit"
                isLoading={submitMut.isPending}
                disabled={title.trim().length < 3 || atFeatureRequestLimit}
              >
                {t('common.submit')}
              </Button>
            </div>
          </form>
        </div>
      )}
    </div>
  )
}

function StepIndicator({ current }: { current: 1 | 2 }) {
  const { t } = useTranslation()
  const steps: { n: 1 | 2; label: string }[] = [
    { n: 1, label: t('new.stepSummary') },
    { n: 2, label: t('new.stepDetails') },
  ]
  return (
    <div className="flex items-center gap-2 text-xs">
      {steps.map((s, i) => (
        <div key={s.n} className="flex items-center gap-2">
          <span
            className={clsx(
              'inline-flex size-5 items-center justify-center rounded-full text-[10px] font-semibold',
              s.n === current
                ? 'bg-sky-500 text-white'
                : s.n < current
                  ? 'bg-sky-100 text-sky-700 dark:bg-sky-500/20 dark:text-sky-300'
                  : 'bg-zinc-200 text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400',
            )}
          >
            {s.n}
          </span>
          <span
            className={clsx(
              'font-medium',
              s.n === current
                ? 'text-zinc-900 dark:text-zinc-100'
                : 'text-zinc-500',
            )}
          >
            {s.label}
          </span>
          {i < steps.length - 1 && (
            <span className="mx-1 h-px w-6 bg-zinc-300 dark:bg-zinc-700" />
          )}
        </div>
      ))}
      <span className="ml-1 text-zinc-400">
        {t('new.stepCaption', { current, total: 2 })}
      </span>
    </div>
  )
}

function TypeSegmentedControl({
  value,
  onChange,
}: {
  value: AllowedNewType
  onChange: (next: AllowedNewType) => void
}) {
  const { t } = useTranslation()
  const options: { value: AllowedNewType; label: string; Icon: typeof Lightbulb }[] = [
    { value: 'feature-request', label: t('new.typeFeatureRequest'), Icon: Lightbulb },
    { value: 'bug', label: t('new.typeBug'), Icon: Bug },
  ]
  return (
    <div className="inline-flex rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-0.5 text-sm shadow-soft">
      {options.map(({ value: v, label, Icon }) => (
        <button
          key={v}
          type="button"
          onClick={() => onChange(v)}
          aria-pressed={value === v}
          className={clsx(
            'inline-flex items-center gap-1.5 rounded-md px-3 py-1.5 font-medium transition-colors',
            value === v
              ? 'bg-brand-50 text-brand-700 dark:bg-brand-500/15 dark:text-brand-300'
              : 'text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-300',
          )}
        >
          <Icon className="size-4" />
          {label}
        </button>
      ))}
    </div>
  )
}

function FeatureRequestQuotaHint({ used, max }: { used: number; max: number }) {
  const { t } = useTranslation()
  const remaining = Math.max(0, max - used)
  const atLimit = used >= max
  return (
    <p
      className={clsx(
        'text-xs',
        atLimit
          ? 'text-rose-600 dark:text-rose-400'
          : 'text-zinc-500 dark:text-zinc-400',
      )}
    >
      {atLimit
        ? t('featureRequestQuota.atLimit', { max })
        : t('featureRequestQuota.hint', { remaining, max })}
    </p>
  )
}
