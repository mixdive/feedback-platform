import { useEffect, useRef, useState } from 'react'
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react'
import {
  CheckCircle2,
  Circle,
  FileText,
  Paperclip,
  X,
} from 'lucide-react'
import clsx from 'clsx'

import Button from '@/components/button'
import MarkdownEditor from '@/components/markdown-editor'
import { API, type ApiReleaseState } from '@/services/api'
import { message } from '@/utils/helpers'

export type FormPdf = { url: string; name: string; size: number }

export type FormState = {
  versionName: string
  title: string
  description: string
  releaseDate: string
  state: ApiReleaseState
  pdf: FormPdf | null
}

export const emptyForm: FormState = {
  versionName: '',
  title: '',
  description: '',
  releaseDate: '',
  state: 'planned',
  pdf: null,
}

// ReleaseStateChip is the round planned/completed indicator shared by
// the releases list and the release detail page.
export function ReleaseStateChip({ state }: { state: ApiReleaseState }) {
  if (state === 'completed') {
    return (
      <span
        title="Completed"
        className="inline-flex size-7 shrink-0 items-center justify-center rounded-full bg-emerald-100 text-emerald-600 dark:bg-emerald-500/15"
      >
        <CheckCircle2 className="size-4" />
      </span>
    )
  }
  return (
    <span
      title="Planned"
      className="inline-flex size-7 shrink-0 items-center justify-center rounded-full bg-sky-100 text-sky-600 dark:bg-sky-500/15"
    >
      <Circle className="size-4" />
    </span>
  )
}

// formatDate renders a YYYY-MM-DD calendar date as a locale-friendly
// label without timezone drift (the stored date is UTC-midnight).
export function formatDate(isoDate: string): string {
  const [y, m, d] = isoDate.split('-').map(Number)
  if (!y || !m || !d) return isoDate
  const date = new Date(Date.UTC(y, m - 1, d))
  return date.toLocaleDateString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  })
}

// ReleaseFormDialog is the create/edit modal for a release. Shared by
// the releases list page (create + edit) and the release detail page
// (edit) so the form stays in one place.
export default function ReleaseFormDialog({
  title,
  submitLabel,
  initial,
  isPending,
  onCancel,
  onSubmit,
}: {
  title: string
  submitLabel: string
  initial: FormState
  isPending: boolean
  onCancel: () => void
  onSubmit: (form: FormState) => void
}) {
  const [form, setForm] = useState<FormState>(initial)

  useEffect(() => {
    setForm(initial)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [JSON.stringify(initial)])

  return (
    <Dialog
      open
      onClose={() => (isPending ? undefined : onCancel())}
      className="relative z-40"
    >
      <div className="fixed inset-0 bg-black/40" aria-hidden />
      <div className="fixed inset-0 grid place-items-center p-4">
        <DialogPanel className="w-full max-w-lg rounded-xl bg-white dark:bg-zinc-900 border border-zinc-200 dark:border-zinc-800 p-6 space-y-4">
          <DialogTitle className="text-lg font-semibold">{title}</DialogTitle>
          <form
            onSubmit={(ev) => {
              ev.preventDefault()
              if (!form.versionName.trim()) return
              onSubmit(form)
            }}
            className="space-y-4"
          >
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-xs font-medium text-zinc-500 mb-1">
                  Version name
                </label>
                <input
                  value={form.versionName}
                  onChange={(e) => setForm({ ...form, versionName: e.target.value })}
                  placeholder="v1.2.0"
                  className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none"
                  autoFocus
                  maxLength={60}
                />
              </div>
              <div>
                <label className="block text-xs font-medium text-zinc-500 mb-1">
                  Release date
                </label>
                <input
                  type="date"
                  value={form.releaseDate}
                  onChange={(e) => setForm({ ...form, releaseDate: e.target.value })}
                  className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none"
                />
              </div>
            </div>
            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-1">
                Title
              </label>
              <input
                value={form.title}
                onChange={(e) => setForm({ ...form, title: e.target.value })}
                placeholder="The April release"
                className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none"
                maxLength={200}
              />
            </div>
            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-1">
                Description
              </label>
              <MarkdownEditor
                value={form.description}
                onChange={(next) =>
                  setForm({ ...form, description: next.slice(0, 10000) })
                }
                placeholder="Markdown supported. Paste, drop, or attach images."
                rows={6}
                upload={(file) => API().console.uploadFile(file)}
              />
              <p className="mt-1 text-xs text-zinc-500">
                Rendered above the linked entries on the Portal changelog.
              </p>
            </div>
            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-1">
                PDF attachment
              </label>
              <PdfField
                value={form.pdf}
                onChange={(next) => setForm({ ...form, pdf: next })}
                disabled={isPending}
              />
              <p className="mt-1 text-xs text-zinc-500">
                Optional. Surfaced as a download link on the Portal
                changelog page.
              </p>
            </div>
            <div>
              <label className="block text-xs font-medium text-zinc-500 mb-1">
                State
              </label>
              <div className="flex gap-2">
                {(['planned', 'completed'] as const).map((s) => (
                  <button
                    type="button"
                    key={s}
                    onClick={() => setForm({ ...form, state: s })}
                    className={clsx(
                      'rounded-md border px-3 py-1.5 text-sm capitalize transition-colors',
                      form.state === s
                        ? 'border-sky-500 bg-sky-50 text-sky-700 dark:bg-sky-500/10 dark:text-sky-300'
                        : 'border-zinc-300 dark:border-zinc-700 text-zinc-600 dark:text-zinc-300 hover:bg-zinc-50 dark:hover:bg-zinc-800',
                    )}
                  >
                    {s}
                  </button>
                ))}
              </div>
              <p className="mt-1 text-xs text-zinc-500">
                Only completed releases appear on the Portal changelog.
              </p>
            </div>
            <div className="flex justify-end gap-2">
              <Button
                type="button"
                variant="ghost"
                onClick={onCancel}
                disabled={isPending}
              >
                Cancel
              </Button>
              <Button type="submit" isLoading={isPending}>
                {submitLabel}
              </Button>
            </div>
          </form>
        </DialogPanel>
      </div>
    </Dialog>
  )
}

function PdfField({
  value,
  onChange,
  disabled,
}: {
  value: FormPdf | null
  onChange: (next: FormPdf | null) => void
  disabled: boolean
}) {
  const inputRef = useRef<HTMLInputElement | null>(null)
  const [uploading, setUploading] = useState(false)

  const handlePick = () => {
    if (disabled || uploading) return
    inputRef.current?.click()
  }

  const handleFile = async (file: File) => {
    if (file.type !== 'application/pdf') {
      message('Only PDF files are supported.')
      return
    }
    setUploading(true)
    try {
      const uploaded = await API().console.uploadFile(file)
      onChange({
        url: uploaded.url,
        name: uploaded.originalName || file.name,
        size: uploaded.size,
      })
    } catch (err) {
      message(err)
    } finally {
      setUploading(false)
      if (inputRef.current) inputRef.current.value = ''
    }
  }

  return (
    <div>
      <input
        ref={inputRef}
        type="file"
        accept="application/pdf,.pdf"
        className="hidden"
        onChange={(e) => {
          const file = e.target.files?.[0]
          if (file) void handleFile(file)
        }}
      />
      {value ? (
        <div className="flex items-center gap-3 rounded-md border border-zinc-300 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950/40 px-3 py-2">
          <FileText className="size-4 shrink-0 text-zinc-500" />
          <a
            href={value.url}
            target="_blank"
            rel="noopener noreferrer"
            className="min-w-0 flex-1 truncate text-sm text-sky-700 hover:underline dark:text-sky-300"
          >
            {value.name}
          </a>
          {value.size > 0 && (
            <span className="shrink-0 text-xs text-zinc-500 tabular-nums">
              {formatBytes(value.size)}
            </span>
          )}
          <button
            type="button"
            onClick={handlePick}
            disabled={disabled || uploading}
            className="shrink-0 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-900 px-2 py-1 text-xs text-zinc-600 dark:text-zinc-300 hover:bg-zinc-100 dark:hover:bg-zinc-800 disabled:opacity-50"
          >
            {uploading ? 'Uploading…' : 'Replace'}
          </button>
          <button
            type="button"
            onClick={() => onChange(null)}
            disabled={disabled || uploading}
            title="Remove"
            className="shrink-0 rounded p-1 text-zinc-500 hover:bg-rose-50 dark:hover:bg-rose-500/10 hover:text-rose-600 disabled:opacity-50"
          >
            <X className="size-4" />
          </button>
        </div>
      ) : (
        <button
          type="button"
          onClick={handlePick}
          disabled={disabled || uploading}
          className="inline-flex items-center gap-2 rounded-md border border-dashed border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-2 text-sm text-zinc-600 dark:text-zinc-300 hover:bg-zinc-50 dark:hover:bg-zinc-800 disabled:opacity-50"
        >
          <Paperclip className="size-4" />
          {uploading ? 'Uploading…' : 'Attach PDF'}
        </button>
      )}
    </div>
  )
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(0)} KB`
  return `${(bytes / 1024 / 1024).toFixed(1)} MB`
}
