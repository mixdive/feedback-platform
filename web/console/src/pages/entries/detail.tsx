import { useState } from 'react'
import { Link, useLocation, useNavigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ArrowLeft, ChevronUp, ExternalLink, Eye, Github, GitMerge, Loader2, Lock } from 'lucide-react'
import dayjs from 'dayjs'
import clsx from 'clsx'

import Button from '@/components/button'
import EntryTypeSelect from '@/components/entry-type-select'
import Markdown from '@/components/markdown'
import MergeDialog from '@/components/merge-dialog'
import Relations from '@/components/relations'
import Timeline from '@/components/timeline'
import ReleaseSelect from '@/components/release-select'
import StatusSelect from '@/components/status-select'
import TopicMultiSelect from '@/components/topic-multi-select'
import {
  API,
  type ApiEntry,
  type ApiEntryTypeValue,
  type ApiEntryCreator,
  type ApiEntryStatusValue,
  type ApiEntryTopic,
  type ApiRelease,
} from '@/services/api'
import { message } from '@/utils/helpers'
import { useDocumentTitle } from '@/utils/use-document-title'

function creatorLabel(c?: ApiEntryCreator): string {
  if (!c) return 'Anonymous'
  return c.name || c.username || 'Anonymous'
}

export default function EntryDetailPage() {
  const { id = '' } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const location = useLocation()
  const queryClient = useQueryClient()

  // Detail can be reached from several lists. The originating route passes
  // location.state.from so the back link stays semantically correct.
  const from = (location.state as { from?: string } | null)?.from
  const backTargets: Record<string, { to: string; label: string }> = {
    inbox: { to: '/inbox', label: 'Back to inbox' },
    'feature-requests': { to: '/feature-requests', label: 'Back to Feature Requests' },
    bugs: { to: '/bugs', label: 'Back to Bugs' },
    support: { to: '/support', label: 'Back to Support' },
  }
  const { to: backTo, label: backLabel } =
    (from && backTargets[from]) || { to: '/entry', label: 'Back to entries' }

  const { data: entry, isLoading, error } = useQuery({
    queryKey: ['console', 'entries', id],
    queryFn: () => API().console.getEntry(id),
    enabled: !!id,
  })

  const [activeTab, setActiveTab] = useState<'activity' | 'relations'>('activity')
  const [isMerging, setIsMerging] = useState(false)

  // Reflect the entry title in the browser tab while this page is open.
  useDocumentTitle(entry?.title)

  const { data: topicsData } = useQuery({
    queryKey: ['console', 'entry-topics'],
    queryFn: () => API().console.listEntryTopics(),
  })
  const topics = topicsData?.data

  const { data: releasesData } = useQuery({
    queryKey: ['console', 'releases'],
    queryFn: () => API().console.listReleases(),
  })
  const releases = releasesData?.data

  const voteMut = useMutation({
    mutationFn: () => API().console.toggleVote(id),
    onSuccess: (updated) => {
      queryClient.setQueryData(['console', 'entries', id], updated)
      void queryClient.invalidateQueries({ queryKey: ['console', 'entries'] })
      void queryClient.invalidateQueries({ queryKey: ['console', 'activities', id] })
    },
    onError: (e) => message(e),
  })

  // refreshAfterMetadataChange is the shared onSuccess for every
  // metadata-mutation hook on this page. Each of these mutations
  // emits an activity row server-side, so the timeline query is
  // invalidated alongside the entry-detail and entries-list caches.
  const refreshAfterMetadataChange = (updated: ApiEntry) => {
    queryClient.setQueryData(['console', 'entries', id], updated)
    void queryClient.invalidateQueries({ queryKey: ['console', 'entries'] })
    void queryClient.invalidateQueries({ queryKey: ['console', 'activities', id] })
  }

  const internalMut = useMutation({
    mutationFn: (isInternal: boolean) => API().console.updateEntry(id, { isInternal }),
    onSuccess: refreshAfterMetadataChange,
    onError: (e) => message(e),
  })

  const topicMut = useMutation({
    mutationFn: (topicIds: string[]) =>
      API().console.updateEntry(id, { topicIds }),
    onSuccess: refreshAfterMetadataChange,
    onError: (e) => message(e),
  })

  const entryTypeMut = useMutation({
    mutationFn: (entryType: ApiEntryTypeValue | '') =>
      API().console.updateEntry(id, { entryType: entryType || undefined }),
    onSuccess: refreshAfterMetadataChange,
    onError: (e) => message(e),
  })

  const statusMut = useMutation({
    mutationFn: (status: ApiEntryStatusValue) =>
      API().console.updateEntry(id, { status }),
    onSuccess: refreshAfterMetadataChange,
    onError: (e) => message(e),
  })

  const releaseMut = useMutation({
    mutationFn: (releaseId: string) => API().console.updateEntry(id, { releaseId }),
    onSuccess: refreshAfterMetadataChange,
    onError: (e) => message(e),
  })

  // Integrations status drives whether the "Create GitHub issue"
  // affordance renders. Cached at the page level so visiting another
  // entry doesn't refetch.
  const { data: integrationsData } = useQuery({
    queryKey: ['console', 'integrations'],
    queryFn: () => API().console.getIntegrations(),
  })
  const githubActive = !!integrationsData?.github?.active

  const githubIssueMut = useMutation({
    mutationFn: () => API().console.createGitHubIssue(id),
    onSuccess: (updated) => {
      refreshAfterMetadataChange(updated)
      if (updated.githubIssue?.url) {
        window.open(updated.githubIssue.url, '_blank', 'noopener,noreferrer')
        message('GitHub issue created', 'success')
      }
    },
    onError: (e) => message(e),
  })

  if (isLoading) {
    return <p className="text-zinc-500">Loading…</p>
  }

  if (error || !entry) {
    return (
      <div className="space-y-4">
        <Link
          to={backTo}
          className="inline-flex items-center gap-2 text-sm text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200"
        >
          <ArrowLeft className="size-4" />
          {backLabel}
        </Link>
        <p className="text-rose-600">
          {(error as Error | undefined)?.message ?? 'Feedback not found.'}
        </p>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="border-b border-zinc-200 dark:border-zinc-800 pb-4">
        <button
          type="button"
          onClick={() => navigate(backTo)}
          className="inline-flex items-center gap-2 text-sm text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200"
        >
          <ArrowLeft className="size-4" />
          {backLabel}
        </button>
        <h1 className="mt-2 text-2xl font-semibold break-words">{entry.title}</h1>
      </div>

      {entry.isInternal && (
        <div className="flex items-start gap-3 rounded-lg border border-amber-300 bg-amber-50 dark:border-amber-500/40 dark:bg-amber-500/10 px-4 py-3">
          <Lock className="mt-0.5 size-4 shrink-0 text-amber-700 dark:text-amber-300" />
          <div className="flex-1 text-sm text-amber-900 dark:text-amber-100">
            <div className="font-semibold">This entry is internal.</div>
            <p className="mt-0.5 text-amber-800/90 dark:text-amber-200/80">
              Hidden from the portal — only visible in the Console.
            </p>
          </div>
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_280px]">
        <div className="min-w-0 space-y-6">
          <div className="rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-6">
            <div className="flex items-start gap-5">
              <button
                type="button"
                aria-pressed={entry.isVoted}
                disabled={voteMut.isPending}
                onClick={() => voteMut.mutate()}
                className={clsx(
                  'flex shrink-0 self-start flex-col items-center justify-center rounded-md border w-16 h-16 transition-colors',
                  entry.isVoted
                    ? 'border-sky-500 bg-sky-50 text-sky-700 dark:border-sky-400 dark:bg-sky-500/10 dark:text-sky-300 hover:bg-sky-100 dark:hover:bg-sky-500/20'
                    : 'border-zinc-300 dark:border-zinc-700 text-zinc-700 dark:text-zinc-300 hover:bg-zinc-100 dark:hover:bg-zinc-800',
                  voteMut.isPending && 'opacity-50 cursor-not-allowed',
                )}
              >
                <ChevronUp className="size-5" />
                <span className="text-base font-semibold">{entry.voteCount}</span>
              </button>
              <div className="min-w-0 flex-1">
                {entry.description ? (
                  <Markdown>{entry.description}</Markdown>
                ) : (
                  <p className="text-sm italic text-zinc-500">No description.</p>
                )}
              </div>
            </div>
          </div>

          <div className="space-y-4">
            <div className="flex gap-1 border-b border-zinc-200 dark:border-zinc-800">
              <TabButton
                active={activeTab === 'activity'}
                onClick={() => setActiveTab('activity')}
              >
                Activity
              </TabButton>
              <TabButton
                active={activeTab === 'relations'}
                onClick={() => setActiveTab('relations')}
              >
                Relations ({entry.relations.length})
              </TabButton>
            </div>
            {activeTab === 'activity' ? (
              <Timeline
                entryId={entry.id}
                count={entry.commentCount}
                topics={topics}
                releases={releases}
              />
            ) : (
              <Relations entry={entry} />
            )}
          </div>
        </div>

        <Sidebar
          entry={entry}
          topics={topics}
          releases={releases}
          onToggleInternal={(v) => internalMut.mutate(v)}
          internalPending={internalMut.isPending}
          onChangeEntryType={(t) => entryTypeMut.mutate(t)}
          entryTypePending={entryTypeMut.isPending}
          onChangeStatus={(s) => statusMut.mutate(s)}
          statusPending={statusMut.isPending}
          onChangeTopics={(ids) => topicMut.mutate(ids)}
          topicPending={topicMut.isPending}
          onChangeRelease={(rid) => releaseMut.mutate(rid)}
          releasePending={releaseMut.isPending}
          onMerge={() => setIsMerging(true)}
          githubActive={githubActive}
          onCreateGitHubIssue={() => githubIssueMut.mutate()}
          githubPending={githubIssueMut.isPending}
        />
      </div>

      <MergeDialog
        entry={entry}
        open={isMerging}
        onClose={() => setIsMerging(false)}
      />
    </div>
  )
}

function Sidebar({
  entry,
  topics,
  releases,
  onToggleInternal,
  internalPending,
  onChangeEntryType,
  entryTypePending,
  onChangeStatus,
  statusPending,
  onChangeTopics,
  topicPending,
  onChangeRelease,
  releasePending,
  onMerge,
  githubActive,
  onCreateGitHubIssue,
  githubPending,
}: {
  entry: ApiEntry
  topics: ApiEntryTopic[] | undefined
  releases: ApiRelease[] | undefined
  onToggleInternal: (v: boolean) => void
  internalPending: boolean
  onChangeEntryType: (value: ApiEntryTypeValue | '') => void
  entryTypePending: boolean
  onChangeStatus: (value: ApiEntryStatusValue) => void
  statusPending: boolean
  onChangeTopics: (topicIds: string[]) => void
  topicPending: boolean
  onChangeRelease: (releaseId: string) => void
  releasePending: boolean
  onMerge: () => void
  githubActive: boolean
  onCreateGitHubIssue: () => void
  githubPending: boolean
}) {
  const entryTypeValue = entry.entryType?.value
  // Only feature requests + bugs can be published — support/other don't
  // belong in an engineering issue tracker. Hide-unavailable wins over
  // disable: when an entry doesn't qualify, no GitHub UI renders at all.
  const githubEligible =
    entryTypeValue === 'feature-request' || entryTypeValue === 'bug'
  const showGitHubSection =
    githubEligible && (githubActive || !!entry.githubIssue)
  const topicIds = entry.topics.map((t) => t.id)
  // Only planned releases are selectable; keep the currently-assigned
  // release in the option list even if it has flipped to completed so
  // the dropdown still renders the current value cleanly.
  const releaseOptions = (releases ?? []).filter(
    (r) => r.state === 'planned' || r.id === entry.release?.id,
  )
  return (
    <aside className="space-y-4 lg:sticky lg:top-4 lg:self-start">
      <div>
        {entry.isInternal ? (
          <span
            title="Internal — not visible on the Portal"
            aria-disabled="true"
            className="inline-flex w-full cursor-not-allowed items-center justify-center gap-2 rounded-md border border-zinc-200 px-3 h-9 text-sm font-medium text-zinc-400 dark:border-zinc-800 dark:text-zinc-600"
          >
            <ExternalLink className="size-4" />
            View on portal
          </span>
        ) : (
          <a
            href={`/entry/${entry.id}`}
            className="inline-flex w-full items-center justify-center gap-2 rounded-md border border-zinc-300 bg-white px-3 h-9 text-sm font-medium text-zinc-700 transition-colors hover:bg-zinc-50 dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200 dark:hover:bg-zinc-800"
          >
            <ExternalLink className="size-4" />
            View on portal
          </a>
        )}
        {entry.isInternal && (
          <p className="mt-1.5 text-xs text-zinc-500">
            Internal — not visible on the Portal.
          </p>
        )}
      </div>

      <Section title="Visibility">
        <div
          role="group"
          aria-label="Visibility"
          className="grid grid-cols-2 gap-1 rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-100 dark:bg-zinc-950 p-1"
        >
          <button
            type="button"
            aria-pressed={!entry.isInternal}
            disabled={internalPending}
            onClick={() => {
              if (entry.isInternal) onToggleInternal(false)
            }}
            className={clsx(
              'inline-flex items-center justify-center gap-1.5 rounded h-9 text-sm font-medium transition-colors',
              !entry.isInternal
                ? 'bg-white dark:bg-zinc-900 text-zinc-900 dark:text-zinc-100 shadow-sm ring-1 ring-zinc-200 dark:ring-zinc-800'
                : 'text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200',
              internalPending && 'opacity-60 cursor-not-allowed',
            )}
          >
            <Eye className="size-4" />
            Public
          </button>
          <button
            type="button"
            aria-pressed={entry.isInternal}
            disabled={internalPending}
            onClick={() => {
              if (!entry.isInternal) onToggleInternal(true)
            }}
            className={clsx(
              'inline-flex items-center justify-center gap-1.5 rounded h-9 text-sm font-medium transition-colors',
              entry.isInternal
                ? 'bg-amber-500 text-white dark:bg-amber-500 dark:text-white shadow-sm'
                : 'text-zinc-500 hover:text-amber-700 dark:hover:text-amber-300',
              internalPending && 'opacity-60 cursor-not-allowed',
            )}
          >
            <Lock className="size-4" />
            Internal
          </button>
        </div>
      </Section>

      <Section title="Entry Type">
        <EntryTypeSelect
          value={entry.entryType?.value ?? ''}
          onChange={onChangeEntryType}
          disabled={entryTypePending}
          allowEmpty
          emptyLabel="No entry type"
        />
      </Section>

      <Section title="Status">
        <StatusSelect
          value={entry.status.value}
          onChange={onChangeStatus}
          disabled={statusPending}
        />
      </Section>

      <TopicMultiSelect
        topics={topics}
        value={topicIds}
        aiIds={entry.aiTopicIds}
        onChange={onChangeTopics}
        disabled={topicPending}
      />

      <Section title="Release">
        <ReleaseSelect
          releases={releaseOptions}
          value={entry.release?.id ?? ''}
          onChange={onChangeRelease}
          disabled={releasePending}
        />
      </Section>

      {showGitHubSection && (
        <Section title="GitHub">
          {entry.githubIssue ? (
            <a
              href={entry.githubIssue.url}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex items-center gap-2 text-sm text-sky-600 hover:underline dark:text-sky-400"
            >
              <Github className="size-4" />
              View issue #{entry.githubIssue.number}
              <ExternalLink className="size-3" />
            </a>
          ) : (
            <button
              type="button"
              onClick={onCreateGitHubIssue}
              disabled={githubPending}
              className={clsx(
                'inline-flex w-full items-center justify-center gap-2 rounded-md border border-zinc-300 bg-white px-3 h-9 text-sm font-medium text-zinc-700 hover:bg-zinc-50 transition-colors dark:border-zinc-700 dark:bg-zinc-900 dark:text-zinc-200 dark:hover:bg-zinc-800',
                githubPending && 'opacity-60 cursor-not-allowed',
              )}
            >
              {githubPending ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <Github className="size-4" />
              )}
              {githubPending ? 'Creating…' : 'Create GitHub issue'}
            </button>
          )}
        </Section>
      )}

      <div className="rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-3 divide-y divide-zinc-100 dark:divide-zinc-800">
        <InfoRow label="Votes">
          <span className="text-sm">
            <span className="font-semibold">{entry.voteCount}</span>{' '}
            <span className="text-zinc-500">
              {entry.voteCount === 1 ? 'vote' : 'votes'}
            </span>
          </span>
        </InfoRow>
        <InfoRow label="Comments">
          <span className="text-sm">
            <span className="font-semibold">{entry.commentCount}</span>{' '}
            <span className="text-zinc-500">
              {entry.commentCount === 1 ? 'comment' : 'comments'}
            </span>
          </span>
        </InfoRow>
        <InfoRow label="Creator">
          <CreatorBlock creator={entry.creator} />
        </InfoRow>
        <InfoRow label="Created">
          <span className="text-sm text-zinc-700 dark:text-zinc-300" title={entry.createdAt}>
            {dayjs(entry.createdAt).format('MMM D, YYYY h:mm A')}
          </span>
        </InfoRow>
        {entry.updatedAt && entry.updatedAt !== entry.createdAt && (
          <InfoRow label="Updated">
            <span className="text-sm text-zinc-700 dark:text-zinc-300" title={entry.updatedAt}>
              {dayjs(entry.updatedAt).format('MMM D, YYYY h:mm A')}
            </span>
          </InfoRow>
        )}
      </div>

      <div className="pt-2">
        <Button variant="ghost" size="sm" onClick={onMerge} className="w-full">
          <GitMerge className="size-4" />
          Merge into…
        </Button>
      </div>
    </aside>
  )
}

function InfoRow({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-3 py-2 first:pt-0 last:pb-0">
      <span className="text-xs font-medium uppercase tracking-wide text-zinc-500">
        {label}
      </span>
      <div className="min-w-0 text-right">{children}</div>
    </div>
  )
}

function TabButton({
  active,
  onClick,
  children,
}: {
  active: boolean
  onClick: () => void
  children: React.ReactNode
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={clsx(
        '-mb-px border-b-2 px-3 py-2 text-sm transition-colors',
        active
          ? 'border-sky-500 text-zinc-900 dark:text-zinc-100'
          : 'border-transparent text-zinc-500 hover:text-zinc-800 dark:hover:text-zinc-200',
      )}
    >
      {children}
    </button>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-3">
      <div className="text-xs font-medium uppercase tracking-wide text-zinc-500">
        {title}
      </div>
      <div className="mt-2">{children}</div>
    </div>
  )
}

function CreatorBlock({ creator }: { creator?: ApiEntryCreator }) {
  if (!creator) {
    return <span className="text-sm text-zinc-500">Anonymous</span>
  }
  const initial = (creator.name || creator.username || '?')[0]?.toUpperCase() ?? '?'
  return (
    <div className="flex items-center gap-2">
      {creator.imageUrl ? (
        <img src={creator.imageUrl} alt="" className="size-7 rounded-full object-cover" />
      ) : (
        <div className="grid size-7 shrink-0 place-items-center rounded-full bg-zinc-200 dark:bg-zinc-800 text-xs font-semibold text-zinc-600 dark:text-zinc-300">
          {initial}
        </div>
      )}
      <div className="min-w-0 text-sm">
        <div className="truncate text-zinc-800 dark:text-zinc-200">
          {creatorLabel(creator)}
        </div>
        {creator.username && creator.name && (
          <div className="truncate text-xs text-zinc-500">@{creator.username}</div>
        )}
      </div>
    </div>
  )
}
