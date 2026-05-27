import { useRef, useState } from 'react'
import clsx from 'clsx'
import {
  Bold,
  Code,
  Eye,
  Heading2,
  Italic,
  Link as LinkIcon,
  List,
  ListOrdered,
  Paperclip,
  Pencil,
  Quote,
} from 'lucide-react'
import toast from 'react-hot-toast'

import Markdown from '@/components/markdown'

type UploadResult = {
  url: string
  originalName: string
  isImage: boolean
}

interface Props {
  value: string
  onChange: (next: string) => void
  placeholder?: string
  rows?: number
  id?: string
  name?: string
  className?: string
  // upload, when provided, enables paste/drop/click file upload. Returning
  // the upload result causes the editor to insert markdown — `![](url)` for
  // images, `[name](url)` for everything else — at the current cursor.
  upload?: (file: File) => Promise<UploadResult>
}

type Mode = 'edit' | 'preview'

function fileNameStem(name: string): string {
  const dot = name.lastIndexOf('.')
  return dot > 0 ? name.slice(0, dot) : name
}

function markdownFor(result: UploadResult): string {
  const alt = fileNameStem(result.originalName) || 'file'
  return result.isImage ? `![${alt}](${result.url})` : `[${alt}](${result.url})`
}

export default function MarkdownEditor({
  value,
  onChange,
  placeholder,
  rows = 6,
  id,
  name,
  className,
  upload,
}: Props) {
  const ref = useRef<HTMLTextAreaElement>(null)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const [mode, setMode] = useState<Mode>('edit')
  // Track upload-in-flight count so we can disable mode-switching while bytes
  // are still flowing. Concurrent uploads share the counter.
  const [uploading, setUploading] = useState(0)
  // Latest committed value, in a ref, so async upload completions can splice
  // into the most recent text instead of a stale closure.
  const latestRef = useRef(value)
  latestRef.current = value

  function wrap(prefix: string, suffix: string, placeholderText: string) {
    const ta = ref.current
    if (!ta) return
    const start = ta.selectionStart
    const end = ta.selectionEnd
    const selected = value.slice(start, end) || placeholderText
    const next = value.slice(0, start) + prefix + selected + suffix + value.slice(end)
    onChange(next)
    requestAnimationFrame(() => {
      ta.focus()
      const cursorStart = start + prefix.length
      const cursorEnd = cursorStart + selected.length
      ta.setSelectionRange(cursorStart, cursorEnd)
    })
  }

  function prefixLines(prefix: string | ((i: number) => string)) {
    const ta = ref.current
    if (!ta) return
    const start = ta.selectionStart
    const end = ta.selectionEnd
    const lineStart = value.lastIndexOf('\n', Math.max(0, start - 1)) + 1
    const nextNl = value.indexOf('\n', end)
    const segEnd = nextNl === -1 ? value.length : nextNl
    const segment = value.slice(lineStart, segEnd)
    const lines = segment.length === 0 ? [''] : segment.split('\n')
    const updated = lines
      .map((l, i) => (typeof prefix === 'function' ? prefix(i) : prefix) + l)
      .join('\n')
    const next = value.slice(0, lineStart) + updated + value.slice(segEnd)
    onChange(next)
    requestAnimationFrame(() => {
      ta.focus()
      ta.setSelectionRange(lineStart, lineStart + updated.length)
    })
  }

  function insertLink() {
    const ta = ref.current
    if (!ta) return
    const start = ta.selectionStart
    const end = ta.selectionEnd
    const selected = value.slice(start, end) || 'link text'
    const inserted = `[${selected}](https://)`
    const next = value.slice(0, start) + inserted + value.slice(end)
    onChange(next)
    requestAnimationFrame(() => {
      ta.focus()
      const urlStart = start + selected.length + 3
      ta.setSelectionRange(urlStart, urlStart + 'https://'.length)
    })
  }

  // uploadFiles inserts a placeholder per file at the current selection, runs
  // the uploads, then swaps each placeholder for the final markdown. Failures
  // remove the placeholder and surface a toast.
  async function uploadFiles(files: File[]) {
    if (!upload || files.length === 0) return
    const ta = ref.current
    let cursor = ta ? ta.selectionStart : latestRef.current.length
    const tokens: string[] = []
    let buffer = latestRef.current
    for (let i = 0; i < files.length; i++) {
      const token = `[uploading-${Math.random().toString(36).slice(2, 10)}]`
      tokens.push(token)
      const placeholder = `${token}\n`
      buffer = buffer.slice(0, cursor) + placeholder + buffer.slice(cursor)
      cursor += placeholder.length
    }
    onChange(buffer)
    latestRef.current = buffer

    setUploading((n) => n + files.length)
    await Promise.all(
      files.map(async (file, i) => {
        const token = tokens[i]
        try {
          const result = await upload(file)
          const replacement = markdownFor(result)
          const current = latestRef.current
          const next = current.replace(token, replacement)
          latestRef.current = next
          onChange(next)
        } catch (err) {
          const msg = err instanceof Error ? err.message : 'Upload failed.'
          toast.error(msg)
          const current = latestRef.current
          // Strip the placeholder + the trailing newline we added with it.
          const next = current.replace(`${token}\n`, '').replace(token, '')
          latestRef.current = next
          onChange(next)
        } finally {
          setUploading((n) => n - 1)
        }
      }),
    )
  }

  function onPaste(e: React.ClipboardEvent<HTMLTextAreaElement>) {
    if (!upload) return
    const items = Array.from(e.clipboardData?.items ?? [])
    const files: File[] = []
    for (const it of items) {
      if (it.kind === 'file') {
        const f = it.getAsFile()
        if (f) files.push(f)
      }
    }
    if (files.length === 0) return
    e.preventDefault()
    void uploadFiles(files)
  }

  function onDrop(e: React.DragEvent<HTMLTextAreaElement>) {
    if (!upload) return
    const files = Array.from(e.dataTransfer?.files ?? [])
    if (files.length === 0) return
    e.preventDefault()
    void uploadFiles(files)
  }

  function onDragOver(e: React.DragEvent<HTMLTextAreaElement>) {
    if (!upload) return
    if (Array.from(e.dataTransfer?.items ?? []).some((i) => i.kind === 'file')) {
      e.preventDefault()
    }
  }

  function onFilePick(e: React.ChangeEvent<HTMLInputElement>) {
    const files = Array.from(e.target.files ?? [])
    if (files.length > 0) void uploadFiles(files)
    // Reset so picking the same file twice in a row still fires onChange.
    e.target.value = ''
  }

  const buttons = [
    { icon: Heading2, label: 'Heading', run: () => prefixLines('## ') },
    { icon: Bold, label: 'Bold', run: () => wrap('**', '**', 'bold') },
    { icon: Italic, label: 'Italic', run: () => wrap('_', '_', 'italic') },
    { icon: Code, label: 'Code', run: () => wrap('`', '`', 'code') },
    { icon: LinkIcon, label: 'Link', run: insertLink },
    { icon: List, label: 'Bullet list', run: () => prefixLines('- ') },
    { icon: ListOrdered, label: 'Numbered list', run: () => prefixLines((i) => `${i + 1}. `) },
    { icon: Quote, label: 'Quote', run: () => prefixLines('> ') },
  ]

  return (
    <div
      className={clsx(
        'rounded-md border border-zinc-300 bg-white focus-within:border-sky-500 dark:border-zinc-700 dark:bg-zinc-950',
        className,
      )}
    >
      <div className="flex items-center justify-between gap-1 border-b border-zinc-200 px-2 py-1 dark:border-zinc-800">
        <div className="flex items-center gap-0.5">
          {buttons.map((b) => (
            <button
              key={b.label}
              type="button"
              title={b.label}
              aria-label={b.label}
              disabled={mode === 'preview'}
              onClick={b.run}
              className="inline-flex size-7 items-center justify-center rounded text-zinc-600 hover:bg-zinc-100 disabled:cursor-not-allowed disabled:opacity-40 dark:text-zinc-400 dark:hover:bg-zinc-800"
            >
              <b.icon className="size-4" />
            </button>
          ))}
          {upload && (
            <button
              type="button"
              title="Attach file"
              aria-label="Attach file"
              disabled={mode === 'preview'}
              onClick={() => fileInputRef.current?.click()}
              className="inline-flex size-7 items-center justify-center rounded text-zinc-600 hover:bg-zinc-100 disabled:cursor-not-allowed disabled:opacity-40 dark:text-zinc-400 dark:hover:bg-zinc-800"
            >
              <Paperclip className="size-4" />
            </button>
          )}
          {upload && (
            <input
              ref={fileInputRef}
              type="file"
              multiple
              hidden
              onChange={onFilePick}
            />
          )}
        </div>
        <div className="flex items-center gap-0.5">
          {uploading > 0 && (
            <span className="mr-1 text-[11px] text-zinc-500 dark:text-zinc-400">
              Uploading…
            </span>
          )}
          <button
            type="button"
            onClick={() => setMode('edit')}
            className={clsx(
              'inline-flex h-7 items-center gap-1 rounded px-2 text-xs',
              mode === 'edit'
                ? 'bg-zinc-100 text-zinc-900 dark:bg-zinc-800 dark:text-zinc-100'
                : 'text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800',
            )}
          >
            <Pencil className="size-3.5" /> Write
          </button>
          <button
            type="button"
            onClick={() => setMode('preview')}
            className={clsx(
              'inline-flex h-7 items-center gap-1 rounded px-2 text-xs',
              mode === 'preview'
                ? 'bg-zinc-100 text-zinc-900 dark:bg-zinc-800 dark:text-zinc-100'
                : 'text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800',
            )}
          >
            <Eye className="size-3.5" /> Preview
          </button>
        </div>
      </div>

      {mode === 'edit' ? (
        <textarea
          ref={ref}
          id={id}
          name={name}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          onPaste={onPaste}
          onDrop={onDrop}
          onDragOver={onDragOver}
          placeholder={placeholder}
          rows={rows}
          className="block w-full resize-y bg-transparent p-3 text-sm focus:outline-none"
        />
      ) : (
        <div className="min-h-[120px] p-3">
          {value.trim() ? (
            <Markdown>{value}</Markdown>
          ) : (
            <p className="text-sm text-zinc-400">Nothing to preview</p>
          )}
        </div>
      )}
    </div>
  )
}
