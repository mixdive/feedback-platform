import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  ChevronUp,
  Lock,
  MessageSquare,
  Rocket,
  Tag,
  ThumbsUp,
  Users,
} from 'lucide-react'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import clsx from 'clsx'

import { EntryTypeIcon } from '@/components/entry-type-badge'
import { API, type ApiDashboardEntryRow } from '@/services/api'
import { ENTRY_TYPES, entryTypeInfo } from '@/utils/entry-type'
import { ENTRY_STATUSES, entryStatusInfo } from '@/utils/entry-status'

dayjs.extend(relativeTime)

export default function DashboardPage() {
  const { data, isLoading, error } = useQuery({
    queryKey: ['console', 'dashboard'],
    queryFn: () => API().console.getDashboard(),
  })

  return (
    <div className="space-y-6">
      <div>
        <h1 className="text-2xl font-semibold">Dashboard</h1>
        <p className="mt-1 text-sm text-zinc-500 dark:text-zinc-400">
          Aggregate picture of feedback in this deployment. Numbers
          recompute on every visit — there is no snapshot to refresh.
        </p>
      </div>

      {isLoading && (
        <div className="rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 px-4 py-12 text-center text-sm text-zinc-500">
          Loading…
        </div>
      )}
      {error && (
        <div className="rounded-lg border border-rose-200 bg-rose-50 dark:border-rose-500/30 dark:bg-rose-500/10 px-4 py-12 text-center text-sm text-rose-700 dark:text-rose-300">
          {(error as Error).message}
        </div>
      )}

      {data && (
        <>
          <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-6 lg:gap-6">
            <KpiCard label="Entries" value={data.totalEntries} icon={MessageSquare} />
            <KpiCard label="Open" value={data.openEntries} icon={MessageSquare} tone="emerald" />
            <KpiCard label="Closed" value={data.closedEntries} icon={MessageSquare} tone="violet" />
            <KpiCard label="Votes" value={data.totalVotes} icon={ThumbsUp} tone="sky" />
            <KpiCard label="Users" value={data.totalUsers} icon={Users} tone="amber" />
            <KpiCard label="Releases" value={data.totalReleases} icon={Rocket} tone="rose" />
          </div>

          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            <Card title="Submissions — last 30 days">
              <TrendChart series={data.entriesPerDay} />
            </Card>

            <Card title="By entry type">
              <TypeBreakdown counts={data.entriesByType} total={data.totalEntries} />
            </Card>
          </div>

          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            <Card title="By status">
              <StatusBreakdown counts={data.entriesByStatus} total={data.totalEntries} />
            </Card>

            <Card title="Visibility">
              <div className="flex items-center gap-4">
                <VisibilityBar
                  publicCount={data.publicEntries}
                  internalCount={data.internalEntries}
                />
              </div>
              <dl className="mt-4 grid grid-cols-2 gap-3 text-xs">
                <div className="rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950/40 px-3 py-2">
                  <dt className="text-zinc-500">Public</dt>
                  <dd className="text-base font-semibold text-zinc-900 dark:text-zinc-100 tabular-nums">
                    {data.publicEntries}
                  </dd>
                </div>
                <div className="rounded-md border border-amber-200 dark:border-amber-500/40 bg-amber-50/60 dark:bg-amber-500/[0.07] px-3 py-2">
                  <dt className="text-amber-700 dark:text-amber-300 inline-flex items-center gap-1">
                    <Lock className="size-3" /> Internal
                  </dt>
                  <dd className="text-base font-semibold text-zinc-900 dark:text-zinc-100 tabular-nums">
                    {data.internalEntries}
                  </dd>
                </div>
                <div className="rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950/40 px-3 py-2">
                  <dt className="text-zinc-500 inline-flex items-center gap-1">
                    <Tag className="size-3" /> Topics
                  </dt>
                  <dd className="text-base font-semibold text-zinc-900 dark:text-zinc-100 tabular-nums">
                    {data.totalTopics}
                  </dd>
                </div>
                <div className="rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950/40 px-3 py-2">
                  <dt className="text-zinc-500 inline-flex items-center gap-1">
                    <MessageSquare className="size-3" /> Comments
                  </dt>
                  <dd className="text-base font-semibold text-zinc-900 dark:text-zinc-100 tabular-nums">
                    {data.totalComments}
                  </dd>
                </div>
              </dl>
            </Card>
          </div>

          <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
            <Card title="Top voted">
              <EntryRowList rows={data.topEntriesByVote} mode="vote" />
            </Card>
            <Card title="Most recent">
              <EntryRowList rows={data.recentEntries} mode="recent" />
            </Card>
          </div>
        </>
      )}
    </div>
  )
}

const TONE_BG: Record<string, string> = {
  default: 'bg-white dark:bg-zinc-900',
  emerald: 'bg-emerald-50/60 dark:bg-emerald-500/[0.07]',
  violet: 'bg-violet-50/60 dark:bg-violet-500/[0.07]',
  sky: 'bg-sky-50/60 dark:bg-sky-500/[0.07]',
  amber: 'bg-amber-50/60 dark:bg-amber-500/[0.07]',
  rose: 'bg-rose-50/60 dark:bg-rose-500/[0.07]',
}

const TONE_ICON: Record<string, string> = {
  default: 'text-zinc-400',
  emerald: 'text-emerald-600 dark:text-emerald-300',
  violet: 'text-violet-600 dark:text-violet-300',
  sky: 'text-sky-600 dark:text-sky-300',
  amber: 'text-amber-600 dark:text-amber-300',
  rose: 'text-rose-600 dark:text-rose-300',
}

function KpiCard({
  label,
  value,
  icon: Icon,
  tone = 'default',
}: {
  label: string
  value: number
  icon: typeof MessageSquare
  tone?: keyof typeof TONE_BG
}) {
  return (
    <div
      className={clsx(
        'rounded-xl border border-zinc-200 dark:border-zinc-800 px-4 py-3',
        TONE_BG[tone],
      )}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="text-xs text-zinc-500">{label}</span>
        <Icon className={clsx('size-4', TONE_ICON[tone])} />
      </div>
      <div className="mt-1 text-2xl font-semibold tabular-nums">{value}</div>
    </div>
  )
}

function Card({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-4">
      <h2 className="text-sm font-semibold text-zinc-700 dark:text-zinc-200">
        {title}
      </h2>
      <div className="mt-3">{children}</div>
    </section>
  )
}

function TrendChart({
  series,
}: {
  series: { date: string; count: number }[]
}) {
  if (series.length === 0) {
    return <p className="text-sm text-zinc-500">No data yet.</p>
  }
  const max = Math.max(1, ...series.map((d) => d.count))
  const width = 520
  const height = 140
  const padding = 8
  const stepX = (width - padding * 2) / Math.max(1, series.length - 1)
  const pointAt = (i: number, v: number): [number, number] => [
    padding + i * stepX,
    height - padding - (v / max) * (height - padding * 2),
  ]
  const points = series.map((d, i) => pointAt(i, d.count))
  const path = points
    .map(([x, y], i) => `${i === 0 ? 'M' : 'L'}${x.toFixed(1)},${y.toFixed(1)}`)
    .join(' ')
  const areaPath =
    `M${padding},${height - padding} ` +
    points.map(([x, y]) => `L${x.toFixed(1)},${y.toFixed(1)}`).join(' ') +
    ` L${(width - padding).toFixed(1)},${height - padding} Z`

  const total = series.reduce((sum, d) => sum + d.count, 0)
  const first = series[0]?.date
  const last = series[series.length - 1]?.date

  return (
    <div>
      <div className="text-xs text-zinc-500 mb-2 flex items-center justify-between">
        <span>{total} new in 30 days</span>
        <span className="tabular-nums">peak: {max}</span>
      </div>
      <svg
        viewBox={`0 0 ${width} ${height}`}
        preserveAspectRatio="none"
        className="w-full h-32"
      >
        <path d={areaPath} fill="#0ea5e9" opacity="0.12" />
        <path d={path} fill="none" stroke="#0ea5e9" strokeWidth="1.5" />
        {points.map(([x, y], i) => (
          <circle
            key={i}
            cx={x}
            cy={y}
            r="1.8"
            fill="#0ea5e9"
            opacity={series[i].count > 0 ? 1 : 0}
          />
        ))}
      </svg>
      <div className="mt-1 flex justify-between text-[10px] text-zinc-500">
        <span>{first && dayjs(first).format('MMM D')}</span>
        <span>{last && dayjs(last).format('MMM D')}</span>
      </div>
    </div>
  )
}

function TypeBreakdown({
  counts,
  total,
}: {
  counts: Record<string, number>
  total: number
}) {
  if (total === 0) {
    return <p className="text-sm text-zinc-500">No entries yet.</p>
  }
  return (
    <ul className="space-y-2">
      {ENTRY_TYPES.map((t) => {
        const n = counts[t.value] ?? 0
        const pct = total > 0 ? Math.round((n / total) * 100) : 0
        return (
          <li key={t.value} className="space-y-1">
            <div className="flex items-center justify-between text-xs">
              <span className="inline-flex items-center gap-1.5">
                <EntryTypeIcon name={t.icon} color={t.color} className="size-3.5" />
                <span className="font-medium text-zinc-700 dark:text-zinc-200">
                  {t.title}
                </span>
              </span>
              <span className="tabular-nums text-zinc-500">
                {n} <span className="text-zinc-400">· {pct}%</span>
              </span>
            </div>
            <div className="h-2 w-full overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-800">
              <div
                className="h-full rounded-full"
                style={{ width: `${pct}%`, backgroundColor: t.color }}
              />
            </div>
          </li>
        )
      })}
    </ul>
  )
}

function StatusBreakdown({
  counts,
  total,
}: {
  counts: Record<string, number>
  total: number
}) {
  if (total === 0) {
    return <p className="text-sm text-zinc-500">No entries yet.</p>
  }
  return (
    <ul className="space-y-2">
      {ENTRY_STATUSES.map((s) => {
        const n = counts[s.value] ?? 0
        const pct = total > 0 ? Math.round((n / total) * 100) : 0
        return (
          <li key={s.value} className="space-y-1">
            <div className="flex items-center justify-between text-xs">
              <span className="inline-flex items-center gap-1.5">
                <span
                  className="inline-block size-2 rounded-full"
                  style={{ backgroundColor: s.color }}
                />
                <span className="font-medium text-zinc-700 dark:text-zinc-200">
                  {s.title}
                </span>
              </span>
              <span className="tabular-nums text-zinc-500">
                {n} <span className="text-zinc-400">· {pct}%</span>
              </span>
            </div>
            <div className="h-2 w-full overflow-hidden rounded-full bg-zinc-100 dark:bg-zinc-800">
              <div
                className="h-full rounded-full"
                style={{ width: `${pct}%`, backgroundColor: s.color }}
              />
            </div>
          </li>
        )
      })}
    </ul>
  )
}

function VisibilityBar({
  publicCount,
  internalCount,
}: {
  publicCount: number
  internalCount: number
}) {
  const total = publicCount + internalCount
  if (total === 0) {
    return <p className="text-sm text-zinc-500">No entries yet.</p>
  }
  const publicPct = (publicCount / total) * 100
  return (
    <div className="w-full h-3 rounded-full overflow-hidden bg-zinc-100 dark:bg-zinc-800 flex">
      <div
        title={`Public · ${publicCount}`}
        className="bg-emerald-500"
        style={{ width: `${publicPct}%` }}
      />
      <div
        title={`Internal · ${internalCount}`}
        className="bg-amber-500"
        style={{ width: `${100 - publicPct}%` }}
      />
    </div>
  )
}

function EntryRowList({
  rows,
  mode,
}: {
  rows: ApiDashboardEntryRow[]
  mode: 'vote' | 'recent'
}) {
  if (rows.length === 0) {
    return <p className="text-sm text-zinc-500">No entries yet.</p>
  }
  return (
    <ul className="divide-y divide-zinc-200 dark:divide-zinc-800">
      {rows.map((e) => {
        const typeInfo = entryTypeInfo(e.entryType)
        const statusInfo = entryStatusInfo(e.status)
        return (
          <li
            key={e.id}
            className="flex items-center gap-3 py-2 first:pt-0 last:pb-0"
          >
            {typeInfo && (
              <EntryTypeIcon
                name={typeInfo.icon}
                color={typeInfo.color}
                className="size-4 shrink-0"
              />
            )}
            <Link
              to={`/entry/${e.id}`}
              className="min-w-0 flex-1 truncate text-sm text-zinc-800 dark:text-zinc-100 hover:underline"
            >
              {e.title}
            </Link>
            {e.isInternal && (
              <Lock className="size-3.5 shrink-0 text-amber-600 dark:text-amber-300" />
            )}
            <span
              className="shrink-0 inline-flex items-center gap-1 text-[10px] uppercase tracking-wide font-medium"
              style={{ color: statusInfo.color }}
            >
              <span
                className="inline-block size-1.5 rounded-full"
                style={{ backgroundColor: statusInfo.color }}
              />
              {statusInfo.title}
            </span>
            {mode === 'vote' ? (
              <span className="shrink-0 inline-flex items-center gap-1 text-xs text-zinc-500 tabular-nums">
                <ChevronUp className="size-3.5" />
                {e.voteCount}
              </span>
            ) : (
              <span className="shrink-0 text-xs text-zinc-500 tabular-nums">
                {dayjs(e.createdAt).fromNow()}
              </span>
            )}
          </li>
        )
      })}
    </ul>
  )
}
