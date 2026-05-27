import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ChevronUp, ExternalLink, X } from 'lucide-react'
import dayjs from 'dayjs'
import clsx from 'clsx'

import EntryTypeBadge from '@/components/entry-type-badge'
import Comments from '@/components/comments'
import Markdown from '@/components/markdown'
import StatusBadge from '@/components/status-badge'
import { API, type ApiEntryCreator } from '@/services/api'
import { message } from '@/utils/helpers'

function creatorLabel(c?: ApiEntryCreator): string {
  if (!c) return 'Anonymous'
  return c.name || c.username || 'Anonymous'
}

interface Props {
  entryId: string | null
  onClose: () => void
}

export default function EntryLightbox({ entryId, onClose }: Props) {
  const queryClient = useQueryClient()
  const isOpen = !!entryId

  const { data: entry, isLoading, error } = useQuery({
    queryKey: ['console', 'entries', entryId ?? ''],
    queryFn: () => API().console.getEntry(entryId ?? ''),
    enabled: !!entryId,
  })

  const voteMut = useMutation({
    mutationFn: () => API().console.toggleVote(entryId ?? ''),
    onSuccess: (updated) => {
      queryClient.setQueryData(['console', 'entries', entryId ?? ''], updated)
      void queryClient.invalidateQueries({ queryKey: ['console', 'entries'] })
    },
    onError: (e) => message(e),
  })

  return (
    <Dialog open={isOpen} onClose={onClose} className="relative z-50">
      <div className="fixed inset-0 bg-zinc-950/60 backdrop-blur-sm" aria-hidden="true" />
      <div className="fixed inset-0 flex items-center justify-center p-4">
        <DialogPanel className="flex max-h-[90vh] w-full max-w-2xl flex-col overflow-hidden rounded-xl bg-white dark:bg-zinc-900 text-zinc-900 dark:text-zinc-100 shadow-2xl">
          <div className="flex items-start gap-3 border-b border-zinc-200 dark:border-zinc-800 p-5">
            <DialogTitle className="flex-1 text-lg font-semibold leading-snug">
              {entry?.title ?? (isLoading ? 'Loading…' : 'Feedback')}
            </DialogTitle>
            {entry && (
              <a
                href={`/console/entry/${entry.id}`}
                title="Open full view"
                className="rounded-md p-1.5 text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800"
              >
                <ExternalLink className="size-4" />
              </a>
            )}
            <button
              type="button"
              onClick={onClose}
              aria-label="Close"
              className="rounded-md p-1.5 text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800"
            >
              <X className="size-4" />
            </button>
          </div>

          <div className="flex-1 overflow-y-auto p-5">
            {isLoading && <p className="text-zinc-500">Loading…</p>}
            {error && (
              <p className="text-rose-600">
                {(error as Error).message || 'Could not load feedback.'}
              </p>
            )}
            {entry && (
              <div className="space-y-5">
                <div className="flex items-start gap-4">
                  <button
                    type="button"
                    aria-pressed={entry.isVoted}
                    disabled={voteMut.isPending}
                    onClick={() => voteMut.mutate()}
                    className={clsx(
                      'flex shrink-0 flex-col items-center justify-center rounded-md border w-14 h-14 transition-colors',
                      entry.isVoted
                        ? 'border-sky-500 bg-sky-50 text-sky-700 dark:border-sky-400 dark:bg-sky-500/10 dark:text-sky-300 hover:bg-sky-100 dark:hover:bg-sky-500/20'
                        : 'border-zinc-300 dark:border-zinc-700 text-zinc-700 dark:text-zinc-300 hover:bg-zinc-100 dark:hover:bg-zinc-800',
                      voteMut.isPending && 'opacity-50 cursor-not-allowed',
                    )}
                  >
                    <ChevronUp className="size-4" />
                    <span className="text-sm font-semibold">{entry.voteCount}</span>
                  </button>
                  <div className="min-w-0 flex-1">
                    <div className="flex flex-wrap items-center gap-2">
                      {entry.status && <StatusBadge status={entry.status} />}
                      {entry.entryType && (
                        <EntryTypeBadge
                          entryType={entry.entryType}
                          appliedByAI={entry.entryTypeAppliedByAI}
                        />
                      )}
                      <span className="text-xs text-zinc-500">
                        by{' '}
                        <span className="text-zinc-700 dark:text-zinc-300">
                          {creatorLabel(entry.creator)}
                        </span>{' '}
                        · {dayjs(entry.createdAt).format('MMM D, YYYY')}
                      </span>
                    </div>
                    {entry.description ? (
                      <div className="mt-3">
                        <Markdown>{entry.description}</Markdown>
                      </div>
                    ) : (
                      <p className="mt-3 text-sm italic text-zinc-500">No description.</p>
                    )}
                  </div>
                </div>

                <Comments entryId={entry.id} count={entry.commentCount} />
              </div>
            )}
          </div>
        </DialogPanel>
      </div>
    </Dialog>
  )
}
