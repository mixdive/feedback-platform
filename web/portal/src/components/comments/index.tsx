import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { MessageSquare } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import dayjs from 'dayjs'

import Button from '@/components/button'
import Markdown from '@/components/markdown'
import MarkdownEditor from '@/components/markdown-editor'
import { API, type ApiComment, type ApiEntryCreator } from '@/services/api'
import { useAppSelector } from '@/store/hooks'
import { message } from '@/utils/helpers'

function authorLabel(a: ApiEntryCreator | undefined, fallback: string): string {
  if (!a) return fallback
  return a.name || a.username || fallback
}

interface Props {
  entryId: string
  count: number
}

export default function Comments({ entryId, count }: Props) {
  const { t } = useTranslation()
  const queryClient = useQueryClient()
  const me = useAppSelector((s) => s.auth.user)
  const [body, setBody] = useState('')

  const { data, isLoading, error } = useQuery({
    queryKey: ['portal', 'comments', entryId],
    queryFn: () => API().portal.listComments(entryId),
    enabled: !!entryId,
  })

  const createMut = useMutation({
    mutationFn: (b: { body: string }) => API().portal.createComment(entryId, b),
    onSuccess: () => {
      setBody('')
      void queryClient.invalidateQueries({ queryKey: ['portal', 'comments', entryId] })
      void queryClient.invalidateQueries({ queryKey: ['portal', 'entries', entryId] })
      void queryClient.invalidateQueries({ queryKey: ['portal', 'entries'] })
    },
    onError: (e) => message(e),
  })

  const total = data?.data.length ?? count

  return (
    <section className="space-y-4">
      <div className="flex items-center gap-2 border-b border-zinc-200 dark:border-zinc-800 pb-2">
        <MessageSquare className="size-4 text-zinc-500" />
        <h2 className="text-sm font-semibold uppercase tracking-wide text-zinc-600 dark:text-zinc-400">
          {t('comments.heading', { count: total })}
        </h2>
      </div>

      {isLoading && <p className="text-sm text-zinc-500">{t('comments.loading')}</p>}
      {error && <p className="text-sm text-rose-600">{(error as Error).message}</p>}

      {data && data.data.length === 0 && (
        <p className="text-sm text-zinc-500 italic">{t('comments.empty')}</p>
      )}

      {data && data.data.length > 0 && (
        <ul className="space-y-3">
          {data.data.map((c) => (
            <CommentItem key={c.id} comment={c} />
          ))}
        </ul>
      )}

      {me ? (
        <form
          onSubmit={(e) => {
            e.preventDefault()
            if (!body.trim()) return
            createMut.mutate({ body: body.trim() })
          }}
          className="space-y-3 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-4"
        >
          <MarkdownEditor
            value={body}
            onChange={setBody}
            placeholder={t('comments.placeholder')}
            rows={4}
            upload={(file) => API().portal.uploadFile(file)}
          />
          <div className="flex justify-end">
            <Button type="submit" isLoading={createMut.isPending} disabled={!body.trim()}>
              {t('comments.post')}
            </Button>
          </div>
        </form>
      ) : (
        <div className="rounded-lg border border-dashed border-zinc-300 dark:border-zinc-700 p-4 text-center text-sm text-zinc-500">
          {t('comments.signInToComment')}
        </div>
      )}
    </section>
  )
}

function CommentItem({ comment }: { comment: ApiComment }) {
  const { t } = useTranslation()
  const author = comment.author
  const initial = (author?.name || author?.username || '?')[0]?.toUpperCase() ?? '?'
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
              {authorLabel(author, t('common.unknown'))}
            </span>
            <span className="text-xs text-zinc-500" title={comment.createdAt}>
              {dayjs(comment.createdAt).format('LL · LT')}
            </span>
          </div>
          <div className="mt-2">
            <Markdown>{comment.body}</Markdown>
          </div>
        </div>
      </div>
    </li>
  )
}
