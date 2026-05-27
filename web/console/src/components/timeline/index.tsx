import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  ArrowRight,
  CircleDashed,
  Eye,
  Github,
  GitMerge,
  Globe,
  Link2,
  Lock,
  type LucideIcon,
  MessageSquare,
  Package,
  Sparkles,
  Tag,
  Unlink2,
} from 'lucide-react'
import { Link } from 'react-router-dom'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'

dayjs.extend(relativeTime)

import Markdown from '@/components/markdown'
import StatusBadge from '@/components/status-badge'
import {
  API,
  type ApiActivity,
  type ApiActivityType,
  type ApiComment,
  type ApiEntryCreator,
  type ApiEntryTopic,
  type ApiRelease,
} from '@/services/api'
import { entryStatusInfo } from '@/utils/entry-status'
import { entryTypeInfo } from '@/utils/entry-type'
import { activityInfo } from '@/utils/activity'
import { message } from '@/utils/helpers'

// ACTIVITY_ICONS maps the icon string in `activityInfo()` to its
// lucide component. Keeps the palette icon-agnostic and centralises
// the visual mapping here.
const ACTIVITY_ICONS: Record<string, LucideIcon> = {
  sparkles: Sparkles,
  'arrow-right': ArrowRight,
  tag: Tag,
  package: Package,
  'link-2': Link2,
  'unlink-2': Unlink2,
  lock: Lock,
  eye: Eye,
  'git-merge': GitMerge,
  github: Github,
}

interface Props {
  entryId: string
  count: number
  topics: ApiEntryTopic[] | undefined
  releases: ApiRelease[] | undefined
}

// Timeline is the unified Console-side audit log for an entry. It
// merges activities (entry-created, status changes, topic edits,
// release assignments, relation add/remove, merges, internal-toggle)
// with comments by `createdAt`, oldest first, and renders each row in
// place. AI-authored rows ([[project_ai_provenance_two_axes]]) carry a
// Sparkles glyph on the timeline gutter so admins can scan provenance.
export default function Timeline({ entryId, count, topics, releases }: Props) {
  const queryClient = useQueryClient()

  const activitiesQ = useQuery({
    queryKey: ['console', 'activities', entryId],
    queryFn: () => API().console.listActivities(entryId),
    enabled: !!entryId,
  })
  const commentsQ = useQuery({
    queryKey: ['console', 'comments', entryId],
    queryFn: () => API().console.listComments(entryId),
    enabled: !!entryId,
  })

  const updateCommentMut = useMutation({
    mutationFn: ({ id, isInternal }: { id: string; isInternal: boolean }) =>
      API().console.updateComment(entryId, id, { isInternal }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['console', 'comments', entryId] })
    },
    onError: (e) => message(e),
  })

  const isLoading = activitiesQ.isLoading || commentsQ.isLoading
  const error = activitiesQ.error || commentsQ.error

  // Build the lookup maps once per render. Topics and releases come
  // pre-loaded by the parent so historical row labels (e.g. a topic
  // that's since been removed from the entry) still resolve.
  const topicById = new Map((topics ?? []).map((t) => [t.id, t]))
  const releaseById = new Map((releases ?? []).map((r) => [r.id, r]))

  type Row =
    | { kind: 'activity'; at: string; activity: ApiActivity }
    | { kind: 'comment'; at: string; comment: ApiComment }
  const rows: Row[] = [
    ...(activitiesQ.data?.data ?? []).map<Row>((a) => ({
      kind: 'activity',
      at: a.createdAt,
      activity: a,
    })),
    ...(commentsQ.data?.data ?? []).map<Row>((c) => ({
      kind: 'comment',
      at: c.createdAt,
      comment: c,
    })),
  ].sort((a, b) => (a.at < b.at ? -1 : a.at > b.at ? 1 : 0))

  const commentCount = commentsQ.data?.data.length ?? count

  return (
    <section className="space-y-4">
      <div className="flex items-center gap-2 border-b border-zinc-200 dark:border-zinc-800 pb-2">
        <MessageSquare className="size-4 text-zinc-500" />
        <h2 className="text-sm font-semibold uppercase tracking-wide text-zinc-600 dark:text-zinc-400">
          Activity ({commentCount} comments)
        </h2>
      </div>

      {isLoading && <p className="text-sm text-zinc-500">Loading…</p>}
      {error && <p className="text-sm text-rose-600">{(error as Error).message}</p>}

      {!isLoading && !error && rows.length === 0 && (
        <p className="text-sm text-zinc-500 italic">No activity yet.</p>
      )}

      {rows.length > 0 && (
        <ol className="space-y-3">
          {rows.map((row) =>
            row.kind === 'comment' ? (
              <CommentItem
                key={'c-' + row.comment.id}
                comment={row.comment}
                onToggleInternal={(isInternal) =>
                  updateCommentMut.mutate({ id: row.comment.id, isInternal })
                }
                isUpdating={
                  updateCommentMut.isPending &&
                  updateCommentMut.variables?.id === row.comment.id
                }
              />
            ) : (
              <ActivityItem
                key={'a-' + row.activity.id}
                activity={row.activity}
                topicById={topicById}
                releaseById={releaseById}
              />
            ),
          )}
        </ol>
      )}
    </section>
  )
}

// authorLabel collapses the User projection into the single string we
// surface on each row. Same precedence as the comments author label.
function authorLabel(a?: ApiEntryCreator): string {
  if (!a) return 'Anonymous'
  return a.name || a.username || 'Anonymous'
}

function CommentItem({
  comment,
  onToggleInternal,
  isUpdating,
}: {
  comment: ApiComment
  onToggleInternal: (isInternal: boolean) => void
  isUpdating: boolean
}) {
  const author = comment.author
  const initial = (author?.name || author?.username || '?')[0]?.toUpperCase() ?? '?'
  const isInternal = comment.isInternal
  return (
    <li className="rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-4">
      <div className="flex items-start gap-3">
        {author?.imageUrl ? (
          <img
            src={author.imageUrl}
            alt=""
            className="size-8 shrink-0 rounded-full object-cover"
          />
        ) : (
          <div className="grid size-8 shrink-0 place-items-center rounded-full bg-zinc-200 dark:bg-zinc-800 text-xs font-semibold text-zinc-600 dark:text-zinc-300">
            {initial}
          </div>
        )}
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5">
            <span className="text-sm font-medium text-zinc-800 dark:text-zinc-200">
              {authorLabel(author)}
            </span>
            <span className="text-xs text-zinc-500" title={comment.createdAt}>
              {dayjs(comment.createdAt).format('MMM D, YYYY · h:mm A')}
            </span>
            {isInternal && (
              <span className="inline-flex items-center gap-1 rounded bg-amber-100 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-amber-800 dark:bg-amber-900/40 dark:text-amber-200">
                <Lock className="size-3" />
                Internal
              </span>
            )}
          </div>
          <div className="mt-2">
            <Markdown>{comment.body}</Markdown>
          </div>
          <div className="mt-3 flex items-center gap-2">
            <button
              type="button"
              onClick={() => onToggleInternal(!isInternal)}
              disabled={isUpdating}
              className="inline-flex items-center gap-1.5 rounded-md border border-zinc-200 px-2 py-1 text-xs text-zinc-600 hover:bg-zinc-50 disabled:opacity-50 dark:border-zinc-800 dark:text-zinc-300 dark:hover:bg-zinc-800"
              title={
                isInternal
                  ? 'Make this comment visible on the public portal'
                  : 'Hide this comment from the public portal'
              }
            >
              {isInternal ? (
                <>
                  <Globe className="size-3.5" />
                  Make public
                </>
              ) : (
                <>
                  <Lock className="size-3.5" />
                  Make internal
                </>
              )}
            </button>
          </div>
        </div>
      </div>
    </li>
  )
}

function ActivityItem({
  activity,
  topicById,
  releaseById,
}: {
  activity: ApiActivity
  topicById: Map<string, ApiEntryTopic>
  releaseById: Map<string, ApiRelease>
}) {
  const info = activityInfo(activity.type)
  const Icon = ACTIVITY_ICONS[info.icon] ?? CircleDashed
  const isAI = activity.source === 'ai'
  const actor = activity.actor
  const actorName =
    isAI ? 'AI' : actor ? authorLabel(actor) : activity.source === 'admin' ? 'An admin' : 'Someone'
  return (
    <li className="flex items-start gap-3 rounded-lg border border-transparent px-2 py-1.5 hover:border-zinc-200 dark:hover:border-zinc-800">
      <div
        className={
          'mt-0.5 grid size-7 shrink-0 place-items-center rounded-full ' +
          (isAI
            ? 'bg-violet-100 text-violet-700 dark:bg-violet-500/15 dark:text-violet-300'
            : 'bg-zinc-100 text-zinc-600 dark:bg-zinc-800 dark:text-zinc-300')
        }
        title={isAI ? 'Applied by AI' : undefined}
      >
        {isAI ? <Sparkles className="size-3.5" /> : <Icon className="size-3.5" />}
      </div>
      <div className="min-w-0 flex-1 text-sm text-zinc-700 dark:text-zinc-300">
        <div className="flex flex-wrap items-center gap-x-1.5 gap-y-1">
          <span className="font-medium text-zinc-900 dark:text-zinc-100">{actorName}</span>
          <ActivityPayload
            activity={activity}
            topicById={topicById}
            releaseById={releaseById}
          />
          <span className="text-xs text-zinc-500" title={activity.createdAt}>
            · {dayjs(activity.createdAt).fromNow()}
          </span>
        </div>
      </div>
    </li>
  )
}

// ActivityPayload renders the per-type readable description.
// Falls back to the palette's generic verb when no specialised
// rendering is available (e.g. future analyzer types).
function ActivityPayload({
  activity,
  topicById,
  releaseById,
}: {
  activity: ApiActivity
  topicById: Map<string, ApiEntryTopic>
  releaseById: Map<string, ApiRelease>
}) {
  const t = activity.type
  switch (t) {
    case 'entry-created':
      return <span>created this feedback</span>
    case 'status-changed': {
      const from = entryStatusInfo(activity.fromValue)
      const to = entryStatusInfo(activity.toValue)
      return (
        <span className="inline-flex items-center gap-1.5">
          changed status from <StatusBadge status={from} /> to <StatusBadge status={to} />
        </span>
      )
    }
    case 'entry-type-changed': {
      const from = entryTypeInfo(activity.fromValue)
      const to = entryTypeInfo(activity.toValue)
      return (
        <span className="inline-flex items-center gap-1.5">
          changed entry type
          {from && (
            <>
              from <TypeChip title={from.title} color={from.color} />
            </>
          )}
          {to && (
            <>
              to <TypeChip title={to.title} color={to.color} />
            </>
          )}
          {!from && !to && <span className="text-zinc-500 italic">to none</span>}
        </span>
      )
    }
    case 'topic-added':
    case 'topic-removed': {
      const topic = activity.targetId ? topicById.get(activity.targetId) : undefined
      const label = topic?.title ?? activity.targetId ?? 'a topic'
      const color = topic?.color || '#71717a'
      return (
        <span className="inline-flex items-center gap-1.5">
          {t === 'topic-added' ? 'added topic' : 'removed topic'}
          <span
            className="inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium"
            style={{ borderColor: color + '66', backgroundColor: color + '1a', color }}
          >
            <span className="size-1.5 rounded-full shrink-0" style={{ backgroundColor: color }} />
            {label}
          </span>
        </span>
      )
    }
    case 'release-set': {
      const r = activity.targetId ? releaseById.get(activity.targetId) : undefined
      const label = r?.versionName || r?.title || activity.targetId || 'a release'
      return (
        <span className="inline-flex items-center gap-1.5">
          assigned release <Chip>{label}</Chip>
        </span>
      )
    }
    case 'release-cleared': {
      const r = activity.fromValue ? releaseById.get(activity.fromValue) : undefined
      const label = r?.versionName || r?.title || activity.fromValue
      return (
        <span>
          cleared release{label ? <> (was <Chip>{label}</Chip>)</> : null}
        </span>
      )
    }
    case 'relation-added':
    case 'relation-removed': {
      const verb = t === 'relation-added' ? 'linked' : 'unlinked'
      const kind = t === 'relation-added' ? activity.toValue : activity.fromValue
      const kindLabel = kind === 'duplicate' ? 'duplicate of' : 'related to'
      return (
        <span className="inline-flex items-center gap-1.5">
          {verb} as {kindLabel}{' '}
          {activity.targetId ? (
            <Link
              to={`/entry/${activity.targetId}`}
              className="text-sky-600 hover:underline dark:text-sky-400"
            >
              another entry
            </Link>
          ) : (
            'another entry'
          )}
        </span>
      )
    }
    case 'internal-enabled':
      return <span>marked this feedback as internal</span>
    case 'internal-disabled':
      return <span>marked this feedback as public</span>
    case 'merged-into': {
      const title = activity.toValue
      return (
        <span className="inline-flex items-center gap-1.5">
          merged this into{' '}
          {activity.targetId ? (
            <Link
              to={`/entry/${activity.targetId}`}
              className="text-sky-600 hover:underline dark:text-sky-400"
            >
              {title || 'the target entry'}
            </Link>
          ) : (
            <Chip>{title || 'the target entry'}</Chip>
          )}
        </span>
      )
    }
    case 'github-issue-created': {
      const url = activity.targetId || activity.toValue
      return (
        <span className="inline-flex items-center gap-1.5">
          published this as{' '}
          {url ? (
            <a
              href={url}
              target="_blank"
              rel="noreferrer noopener"
              className="text-sky-600 hover:underline dark:text-sky-400"
            >
              a GitHub issue
            </a>
          ) : (
            'a GitHub issue'
          )}
        </span>
      )
    }
    default:
      return <span>{activityInfo(t as ApiActivityType).verb}</span>
  }
}

function Chip({ children }: { children: React.ReactNode }) {
  return (
    <span className="inline-flex items-center rounded-full border border-zinc-200 bg-zinc-100 px-2 py-0.5 text-xs font-medium text-zinc-700 dark:border-zinc-700 dark:bg-zinc-800 dark:text-zinc-300">
      {children}
    </span>
  )
}

function TypeChip({ title, color }: { title: string; color: string }) {
  return (
    <span
      className="inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium"
      style={{ borderColor: color + '66', backgroundColor: color + '1a', color }}
    >
      {title}
    </span>
  )
}
