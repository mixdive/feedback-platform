import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import { ExternalLink, Github } from 'lucide-react'

import Button from '@/components/button'
import ReadOnlyBanner from '@/components/read-only-banner'
import { API } from '@/services/api'
import { useAppSelector } from '@/store/hooks'
import { message } from '@/utils/helpers'

dayjs.extend(relativeTime)

// Integrations live on the singleton settings document. v0.1 ships
// only the GitHub integration; the page is laid out as a card grid so
// future integrations (Slack, Linear, …) drop in as siblings without
// restructuring.
//
// Connecting GitHub verifies the (owner, repo, token) triple against
// GitHub's API server-side before persisting — a typo or
// scope-light PAT surfaces inline as a 400 rather than failing later
// when the admin tries to publish an issue.
export default function IntegrationsSettingsPage() {
  const isAdmin = !!useAppSelector((s) => s.auth.user?.roles?.includes('admin'))

  return (
    <div className="space-y-6">
      {!isAdmin && <ReadOnlyBanner />}
      <GitHubCard isAdmin={isAdmin} />
    </div>
  )
}

function GitHubCard({ isAdmin }: { isAdmin: boolean }) {
  const queryClient = useQueryClient()
  const { data, isLoading, error } = useQuery({
    queryKey: ['console', 'integrations'],
    queryFn: () => API().console.getIntegrations(),
  })

  const [owner, setOwner] = useState('')
  const [repo, setRepo] = useState('')
  const [token, setToken] = useState('')

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['console', 'integrations'] })
    // The masked github block on the main settings response also
    // changes; refresh that cache so any consumer that polls it
    // stays in sync.
    void queryClient.invalidateQueries({ queryKey: ['console', 'settings'] })
  }

  const connectMut = useMutation({
    mutationFn: () =>
      API().console.updateGitHubIntegration({
        owner: owner.trim(),
        repo: repo.trim(),
        token: token.trim(),
      }),
    onSuccess: () => {
      message('GitHub connected', 'success')
      setOwner('')
      setRepo('')
      setToken('')
      invalidate()
    },
    onError: (e) => message(e),
  })

  const disconnectMut = useMutation({
    mutationFn: () => API().console.deleteGitHubIntegration(),
    onSuccess: () => {
      message('GitHub disconnected', 'success')
      invalidate()
    },
    onError: (e) => message(e),
  })

  if (isLoading) return <CardShell title="GitHub"><p className="text-sm text-zinc-500">Loading…</p></CardShell>
  if (error) return <CardShell title="GitHub"><p className="text-sm text-rose-600">{(error as Error).message}</p></CardShell>
  if (!data) return null

  const gh = data.github
  const connected = gh.active

  return (
    <CardShell
      title="GitHub"
      icon={<Github className="h-5 w-5" />}
      subtitle={
        connected
          ? `Connected to ${gh.owner}/${gh.repo}.`
          : 'Publish feature requests and bugs as issues in a GitHub repository.'
      }
      headerRight={
        connected && (
          <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-100 px-2 py-0.5 text-xs font-medium text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300">
            <span className="h-1.5 w-1.5 rounded-full bg-emerald-500" />
            Connected
          </span>
        )
      }
    >
      {connected ? (
        <div className="space-y-4">
          <dl className="grid grid-cols-[120px_1fr] gap-x-4 gap-y-2 text-sm">
            <dt className="text-zinc-500">Repository</dt>
            <dd className="font-mono text-zinc-900 dark:text-zinc-100">
              <a
                href={`https://github.com/${gh.owner}/${gh.repo}`}
                target="_blank"
                rel="noopener noreferrer"
                className="inline-flex items-center gap-1 hover:underline"
              >
                {gh.owner}/{gh.repo}
                <ExternalLink className="h-3 w-3" />
              </a>
            </dd>
            <dt className="text-zinc-500">Token</dt>
            <dd className="font-mono text-zinc-700 dark:text-zinc-300">{gh.tokenPreview}</dd>
            {gh.connectedAt && (
              <>
                <dt className="text-zinc-500">Connected</dt>
                <dd title={gh.connectedAt}>{dayjs(gh.connectedAt).fromNow()}</dd>
              </>
            )}
          </dl>
          <div className="border-t border-zinc-200 dark:border-zinc-800 pt-4">
            <p className="text-xs text-zinc-500">
              Feature requests and bugs gain a "Create GitHub issue" button on
              their detail page. Clicking it summarises the entry with AI and
              opens a new issue under {gh.owner}/{gh.repo}.
            </p>
            {isAdmin && (
              <div className="flex justify-end mt-3">
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => {
                    if (confirm('Disconnect the GitHub integration? Already-linked issues stay intact, but no new ones can be created until you reconnect.')) {
                      disconnectMut.mutate()
                    }
                  }}
                  isLoading={disconnectMut.isPending}
                >
                  Disconnect
                </Button>
              </div>
            )}
          </div>
        </div>
      ) : (
        <form
          className="space-y-4"
          onSubmit={(e) => {
            e.preventDefault()
            if (!isAdmin) return
            if (!owner.trim() || !repo.trim() || !token.trim()) {
              message('Owner, repo, and token are all required.')
              return
            }
            connectMut.mutate()
          }}
        >
          <fieldset disabled={!isAdmin} className="space-y-4 disabled:opacity-60">
            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm font-medium mb-1">Owner</label>
                <input
                  type="text"
                  value={owner}
                  onChange={(e) => setOwner(e.target.value)}
                  placeholder="acme-corp"
                  className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none disabled:cursor-not-allowed"
                />
              </div>
              <div>
                <label className="block text-sm font-medium mb-1">Repository</label>
                <input
                  type="text"
                  value={repo}
                  onChange={(e) => setRepo(e.target.value)}
                  placeholder="webapp"
                  className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none disabled:cursor-not-allowed"
                />
              </div>
            </div>
            <div>
              <label className="block text-sm font-medium mb-1">Personal access token</label>
              <input
                type="password"
                autoComplete="off"
                value={token}
                onChange={(e) => setToken(e.target.value)}
                placeholder="ghp_..."
                className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 font-mono text-xs focus:border-sky-500 focus:outline-none disabled:cursor-not-allowed"
              />
              <p className="text-xs text-zinc-500 mt-1">
                Needs <code>repo</code> scope (classic) or <em>Issues: Read &amp; Write</em>{' '}
                (fine-grained). Stored on the settings document in MongoDB; never returned
                to the browser in cleartext.{' '}
                <a
                  href="https://github.com/settings/tokens"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-sky-600 dark:text-sky-400 hover:underline inline-flex items-center gap-1"
                >
                  Create a token
                  <ExternalLink className="h-3 w-3" />
                </a>
              </p>
            </div>
          </fieldset>
          {isAdmin && (
            <div className="flex justify-end">
              <Button
                type="submit"
                isLoading={connectMut.isPending}
                disabled={!owner.trim() || !repo.trim() || !token.trim()}
              >
                Connect
              </Button>
            </div>
          )}
        </form>
      )}
    </CardShell>
  )
}

function CardShell({
  title,
  subtitle,
  icon,
  headerRight,
  children,
}: {
  title: string
  subtitle?: string
  icon?: React.ReactNode
  headerRight?: React.ReactNode
  children: React.ReactNode
}) {
  return (
    <div className="rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-6">
      <div className="flex items-start justify-between mb-4">
        <div className="flex items-start gap-3">
          {icon && (
            <div className="rounded-md bg-zinc-100 dark:bg-zinc-800 p-2 text-zinc-700 dark:text-zinc-300">
              {icon}
            </div>
          )}
          <div>
            <h2 className="text-base font-semibold">{title}</h2>
            {subtitle && <p className="text-sm text-zinc-500 mt-0.5">{subtitle}</p>}
          </div>
        </div>
        {headerRight}
      </div>
      {children}
    </div>
  )
}
