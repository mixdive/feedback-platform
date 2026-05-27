import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import clsx from 'clsx'

export function markdownToPlainText(md: string): string {
  return md
    .replace(/```[\w-]*\n?([\s\S]*?)```/g, '$1')
    .replace(/`([^`]+)`/g, '$1')
    .replace(/!\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
    .replace(/^\s{0,3}#{1,6}\s+/gm, '')
    .replace(/^\s{0,3}>\s?/gm, '')
    .replace(/^\s{0,3}\d+\.\s+/gm, '')
    .replace(/^\s{0,3}[-*+]\s+/gm, '')
    .replace(/^\s{0,3}([-*_])\1{2,}\s*$/gm, '')
    .replace(/(\*\*|__)(.+?)\1/g, '$2')
    .replace(/(\*|_)(.+?)\1/g, '$2')
    .replace(/~~(.+?)~~/g, '$1')
    .replace(/[ \t]+/g, ' ')
    .replace(/\n{2,}/g, '\n')
    .trim()
}

interface Props {
  children: string
  compact?: boolean
  className?: string
}

export default function Markdown({ children, compact = false, className }: Props) {
  return (
    <div
      className={clsx(
        'text-sm text-zinc-600 dark:text-zinc-400',
        compact
          ? 'line-clamp-3 [&>*]:my-0 [&_p]:my-0 [&_ul]:my-0 [&_ol]:my-0 [&_pre]:my-0 [&_blockquote]:my-0 [&_h1]:my-0 [&_h2]:my-0 [&_h3]:my-0'
          : 'space-y-2',
        className,
      )}
    >
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          a: (props) => (
            <a
              {...props}
              target="_blank"
              rel="noopener noreferrer"
              className="text-sky-600 hover:underline dark:text-sky-400"
            />
          ),
          p: (props) => <p {...props} className="leading-relaxed" />,
          strong: (props) => (
            <strong {...props} className="font-semibold text-zinc-900 dark:text-zinc-100" />
          ),
          em: (props) => <em {...props} className="italic" />,
          code: ({ className: cls, children: cc, ...rest }) => {
            const isBlock = /language-/.test(cls || '')
            return isBlock ? (
              <code {...rest} className={clsx(cls, 'block whitespace-pre-wrap')}>
                {cc}
              </code>
            ) : (
              <code
                {...rest}
                className="rounded bg-zinc-100 px-1 py-0.5 text-xs dark:bg-zinc-800"
              >
                {cc}
              </code>
            )
          },
          pre: (props) => (
            <pre
              {...props}
              className="overflow-x-auto rounded bg-zinc-100 p-3 text-xs dark:bg-zinc-800"
            />
          ),
          ul: (props) => <ul {...props} className="list-disc pl-5 space-y-1" />,
          ol: (props) => <ol {...props} className="list-decimal pl-5 space-y-1" />,
          li: (props) => <li {...props} className="leading-relaxed" />,
          h1: (props) => (
            <h1
              {...props}
              className="text-base font-semibold text-zinc-900 dark:text-zinc-100"
            />
          ),
          h2: (props) => (
            <h2
              {...props}
              className="text-base font-semibold text-zinc-900 dark:text-zinc-100"
            />
          ),
          h3: (props) => (
            <h3
              {...props}
              className="text-sm font-semibold text-zinc-900 dark:text-zinc-100"
            />
          ),
          blockquote: (props) => (
            <blockquote
              {...props}
              className="border-l-2 border-zinc-300 pl-3 italic dark:border-zinc-700"
            />
          ),
          hr: () => <hr className="my-2 border-zinc-200 dark:border-zinc-800" />,
        }}
      >
        {children}
      </ReactMarkdown>
    </div>
  )
}
