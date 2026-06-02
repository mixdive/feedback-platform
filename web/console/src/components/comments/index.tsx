import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Lock, MessageSquare, Globe, ShieldCheck } from 'lucide-react'
import dayjs from 'dayjs'

import Markdown from '@/components/markdown'
import { API, type ApiComment, type ApiEntryCreator } from '@/services/api'
import { message } from '@/utils/helpers'

function authorLabel(a?: ApiEntryCreator): string {
  if (!a) return 'Unknown'
  return a.name || a.username || 'Unknown'
}

interface Props {
  entryId: string
  count: number
}

export default function Comments({ entryId, count }: Props) {
  const queryClient = useQueryClient()

  const { data, isLoading, error } = useQuery({
    queryKey: ['console', 'comments', entryId],
    queryFn: () => API().console.listComments(entryId),
    enabled: !!entryId,
  })

  const updateMut = useMutation({
    mutationFn: ({ id, isInternal }: { id: string; isInternal: boolean }) =>
      API().console.updateComment(entryId, id, { isInternal }),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['console', 'comments', entryId] })
    },
    onError: (e) => message(e),
  })

  const total = data?.data.length ?? count

  return (
    <section className="space-y-4">
      <div className="flex items-center gap-2 border-b border-zinc-200 dark:border-zinc-800 pb-2">
        <MessageSquare className="size-4 text-zinc-500" />
        <h2 className="text-sm font-semibold uppercase tracking-wide text-zinc-600 dark:text-zinc-400">
          Comments ({total})
        </h2>
      </div>

      {isLoading && <p className="text-sm text-zinc-500">Loading comments…</p>}
      {error && <p className="text-sm text-rose-600">{(error as Error).message}</p>}

      {data && data.data.length === 0 && (
        <p className="text-sm text-zinc-500 italic">No comments yet.</p>
      )}

      {data && data.data.length > 0 && (
        <ul className="space-y-3">
          {data.data.map((c) => (
            <CommentItem
              key={c.id}
              comment={c}
              onToggleInternal={(isInternal) =>
                updateMut.mutate({ id: c.id, isInternal })
              }
              isUpdating={updateMut.isPending && updateMut.variables?.id === c.id}
            />
          ))}
        </ul>
      )}
    </section>
  )
}

interface CommentItemProps {
  comment: ApiComment
  onToggleInternal: (isInternal: boolean) => void
  isUpdating: boolean
}

function CommentItem({ comment, onToggleInternal, isUpdating }: CommentItemProps) {
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
            {comment.authorIsTeam && (
              <span className="inline-flex items-center gap-1 rounded bg-indigo-100 px-1.5 py-0.5 text-[10px] font-medium uppercase tracking-wide text-indigo-800 dark:bg-indigo-900/40 dark:text-indigo-200">
                <ShieldCheck className="size-3" />
                Team
              </span>
            )}
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
