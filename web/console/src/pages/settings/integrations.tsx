import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useSearchParams } from 'react-router-dom'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import { AlertTriangle, Check, ChevronDown, Copy, ExternalLink, Github, Slack } from 'lucide-react'

import Button from '@/components/button'
import ReadOnlyBanner from '@/components/read-only-banner'
import { API } from '@/services/api'
import { useAppSelector } from '@/store/hooks'
import { message } from '@/utils/helpers'

dayjs.extend(relativeTime)

// Where the browser is sent to start the Slack OAuth flow. A full-page
// navigation (not XHR) so the browser follows the 302 to slack.com; the
// callback lands back here with a ?slack= query the page toasts.
const SLACK_AUTHORIZE_PATH = '/api/console/integrations/slack/authorize'

// Integrations live on the singleton settings document under the
// Integrations sub-block. Each connection renders as its own card;
// future integrations (Linear, …) drop in as siblings without
// restructuring.
//
// Connecting GitHub verifies the (owner, repo, token) triple against
// GitHub's API server-side before persisting; connecting Slack runs a
// bring-your-own-app OAuth flow (the admin picks a channel on Slack).
export default function IntegrationsSettingsPage() {
  const isAdmin = !!useAppSelector((s) => s.auth.user?.roles?.includes('admin'))
  const queryClient = useQueryClient()
  const [searchParams, setSearchParams] = useSearchParams()

  // The Slack OAuth callback redirects back here with ?slack=connected
  // or ?slack=error&message=…. Surface it as a toast, refresh the
  // integrations cache, then strip the query so a refresh doesn't repeat.
  useEffect(() => {
    const status = searchParams.get('slack')
    if (!status) return
    if (status === 'connected') {
      message('Slack connected', 'success')
    } else {
      message(searchParams.get('message') || 'Slack connection failed.')
    }
    void queryClient.invalidateQueries({ queryKey: ['console', 'integrations'] })
    const next = new URLSearchParams(searchParams)
    next.delete('slack')
    next.delete('message')
    setSearchParams(next, { replace: true })
  }, [searchParams, setSearchParams, queryClient])

  return (
    <div className="space-y-6">
      {!isAdmin && <ReadOnlyBanner />}
      <GitHubCard isAdmin={isAdmin} />
      <SlackCard isAdmin={isAdmin} />
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

// The three Portal events an admin can route to Slack. Order matches
// their volume/signal ranking (feedback strongest, votes noisiest).
const SLACK_EVENTS = [
  {
    key: 'notifyOnEntry' as const,
    label: 'New feedback submitted',
    hint: 'Someone posts new feedback on the portal.',
  },
  {
    key: 'notifyOnComment' as const,
    label: 'New comment',
    hint: 'Someone comments on an entry.',
  },
  {
    key: 'notifyOnVote' as const,
    label: 'New vote',
    hint: 'Someone votes on an entry. Highest volume — can get chatty on a hot entry.',
  },
]

function SlackCard({ isAdmin }: { isAdmin: boolean }) {
  const queryClient = useQueryClient()
  const { data, isLoading, error } = useQuery({
    queryKey: ['console', 'integrations'],
    queryFn: () => API().console.getIntegrations(),
  })

  const slack = data?.slack
  const appConfigured = !!slack?.appConfigured
  const connected = !!slack?.connected

  // Client ID is not a secret and round-trips, so it pre-fills. Client
  // secret is write-only: blank on load, and blank-means-keep on save.
  // Toggles are seeded from and kept in sync with server state.
  const [clientId, setClientId] = useState('')
  const [clientSecret, setClientSecret] = useState('')
  const [events, setEvents] = useState({
    notifyOnEntry: false,
    notifyOnComment: false,
    notifyOnVote: false,
  })
  const [copied, setCopied] = useState(false)
  const [showAdvanced, setShowAdvanced] = useState(false)
  useEffect(() => {
    if (slack) {
      setClientId(slack.clientId ?? '')
      setEvents({
        notifyOnEntry: slack.notifyOnEntry,
        notifyOnComment: slack.notifyOnComment,
        notifyOnVote: slack.notifyOnVote,
      })
    }
  }, [slack?.clientId, slack?.notifyOnEntry, slack?.notifyOnComment, slack?.notifyOnVote])

  // The OAuth popup posts its result back here (server-side finishSlack)
  // and closes itself, so the settings page never navigates away.
  useEffect(() => {
    function onMessage(e: MessageEvent) {
      if (e.origin !== window.location.origin) return
      const d = e.data as { source?: string; status?: string; message?: string } | null
      if (!d || d.source !== 'mixdive-slack') return
      if (d.status === 'connected') message('Slack connected', 'success')
      else message(d.message || 'Slack connection failed.')
      void queryClient.invalidateQueries({ queryKey: ['console', 'integrations'] })
    }
    window.addEventListener('message', onMessage)
    return () => window.removeEventListener('message', onMessage)
  }, [queryClient])

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['console', 'integrations'] })
    void queryClient.invalidateQueries({ queryKey: ['console', 'settings'] })
  }

  const saveMut = useMutation({
    mutationFn: () =>
      API().console.updateSlackIntegration({
        clientId: clientId.trim(),
        clientSecret: clientSecret.trim(),
        ...events,
      }),
    onSuccess: () => {
      message('Slack settings saved', 'success')
      setClientSecret('')
      invalidate()
    },
    onError: (e) => message(e),
  })

  const disconnectMut = useMutation({
    mutationFn: () => API().console.deleteSlackIntegration(),
    onSuccess: () => {
      message('Slack removed', 'success')
      setClientId('')
      setClientSecret('')
      invalidate()
    },
    onError: (e) => message(e),
  })

  if (isLoading)
    return (
      <CardShell title="Slack">
        <p className="text-sm text-zinc-500">Loading…</p>
      </CardShell>
    )
  if (error)
    return (
      <CardShell title="Slack">
        <p className="text-sm text-rose-600">{(error as Error).message}</p>
      </CardShell>
    )
  if (!slack) return null

  // Unsaved credential edits must be saved before OAuth, or the flow
  // would run against stale server-side credentials.
  const credsDirty =
    clientSecret.trim() !== '' || clientId.trim() !== (slack.clientId ?? '')

  // Credentials live behind the "Connect your Slack App" disclosure at
  // all times (even before first setup), so the card reads as a single
  // row + the Add to Slack button once configured.
  const showCreds = showAdvanced

  const openSlackPopup = () => {
    const w = window.open(
      `${SLACK_AUTHORIZE_PATH}?popup=1`,
      'mixdive-slack-oauth',
      'popup,width=600,height=760',
    )
    // Popup blocked → fall back to a full-page redirect (page mode; the
    // page-level ?slack= handler picks up the result).
    if (!w) window.location.href = SLACK_AUTHORIZE_PATH
  }

  const copyRedirect = () => {
    if (!slack.redirectUri) return
    try {
      void navigator.clipboard?.writeText(slack.redirectUri)
      setCopied(true)
      setTimeout(() => setCopied(false), 1500)
    } catch {
      /* clipboard unavailable (insecure context) — the field is selectable */
    }
  }

  const inputClass =
    'block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none disabled:cursor-not-allowed'

  return (
    <CardShell
      title="Slack"
      icon={<Slack className="h-5 w-5" />}
      subtitle={
        connected
          ? 'Portal activity is posted to your Slack channel.'
          : 'Post new portal feedback, comments, and votes to a Slack channel.'
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
      <div className="space-y-5">
        {connected && slack.lastErrorMessage && (
          <div className="flex items-start gap-2 rounded-md border border-amber-300 bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:border-amber-900/50 dark:bg-amber-950/30 dark:text-amber-300">
            <AlertTriangle className="h-4 w-4 shrink-0 mt-0.5" />
            <span>
              Last delivery failed
              {slack.lastErrorAt ? ` ${dayjs(slack.lastErrorAt).fromNow()}` : ''}:{' '}
              <span className="font-mono">{slack.lastErrorMessage}</span>
            </span>
          </div>
        )}

        {connected && (
          <dl className="grid grid-cols-[120px_1fr] gap-x-4 gap-y-2 text-sm">
            <dt className="text-zinc-500">Channel</dt>
            <dd className="font-medium text-zinc-900 dark:text-zinc-100">
              {slack.channelName || 'connected'}
              {slack.teamName && (
                <span className="font-normal text-zinc-500"> · {slack.teamName}</span>
              )}
            </dd>
            {slack.connectedAt && (
              <>
                <dt className="text-zinc-500">Connected</dt>
                <dd title={slack.connectedAt}>{dayjs(slack.connectedAt).fromNow()}</dd>
              </>
            )}
          </dl>
        )}

        <form
          className="space-y-5"
          onSubmit={(e) => {
            e.preventDefault()
            if (!isAdmin) return
            saveMut.mutate()
          }}
        >
          <fieldset disabled={!isAdmin} className="space-y-5 disabled:opacity-60">
            {/* App credentials — only during setup (before a channel is
                connected). Once the app is configured they collapse behind
                the disclosure; once connected, Disconnect + re-add rotates
                them. */}
            {!connected && (
              <div className="space-y-3">
                <button
                  type="button"
                  onClick={() => setShowAdvanced((v) => !v)}
                  className="flex items-center gap-1.5 text-sm font-medium text-zinc-600 dark:text-zinc-300 hover:text-zinc-900 dark:hover:text-zinc-100"
                >
                  <ChevronDown
                    className={`h-4 w-4 transition-transform ${showCreds ? '' : '-rotate-90'}`}
                  />
                  {appConfigured ? 'Slack App credentials' : 'Connect your Slack App'}
                  {appConfigured && (
                    <span className="text-xs font-normal text-zinc-400">· configured</span>
                  )}
                </button>
                {showCreds && (
                  <>
                <div className="grid grid-cols-2 gap-3">
                  <div>
                    <label className="block text-sm font-medium mb-1">Client ID</label>
                    <input
                      type="text"
                      autoComplete="off"
                      value={clientId}
                      onChange={(e) => setClientId(e.target.value)}
                      placeholder="1234567890.1234567890"
                      className={inputClass}
                    />
                  </div>
                  <div>
                    <label className="block text-sm font-medium mb-1">Client secret</label>
                    <input
                      type="password"
                      autoComplete="off"
                      value={clientSecret}
                      onChange={(e) => setClientSecret(e.target.value)}
                      placeholder={slack.hasClientSecret ? 'Leave blank to keep' : '••••••••'}
                      className={inputClass}
                    />
                  </div>
                </div>
                <div>
                  <label className="block text-sm font-medium mb-1">Redirect URL</label>
                  <div className="flex gap-2">
                    <input
                      type="text"
                      readOnly
                      value={slack.redirectUri ?? ''}
                      onFocus={(e) => e.currentTarget.select()}
                      className={`${inputClass} font-mono text-xs`}
                    />
                    <Button type="button" variant="ghost" onClick={copyRedirect}>
                      {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                    </Button>
                  </div>
                  <p className="text-xs text-zinc-500 mt-1">
                    From your own{' '}
                    <a
                      href="https://api.slack.com/apps"
                      target="_blank"
                      rel="noopener noreferrer"
                      className="text-sky-600 dark:text-sky-400 hover:underline inline-flex items-center gap-1"
                    >
                      Slack App
                      <ExternalLink className="h-3 w-3" />
                    </a>{' '}
                    (<em>Basic Information</em> → App Credentials). Add the redirect URL above
                    under <em>OAuth &amp; Permissions</em>, then Save and click{' '}
                    <em>Add to Slack</em> to pick a channel. Requires HTTPS. The secret is
                    stored in MongoDB and never returned to the browser.
                  </p>
                </div>
                  </>
                )}
              </div>
            )}

            <div>
              <p className="text-sm font-medium mb-2">Notify on</p>
              <div className="space-y-2">
                {SLACK_EVENTS.map((ev) => (
                  <label key={ev.key} className="flex items-start gap-2.5 cursor-pointer">
                    <input
                      type="checkbox"
                      checked={events[ev.key]}
                      onChange={(e) =>
                        setEvents((prev) => ({ ...prev, [ev.key]: e.target.checked }))
                      }
                      className="mt-0.5 h-4 w-4 rounded border-zinc-300 dark:border-zinc-700 text-sky-600 focus:ring-sky-500"
                    />
                    <span className="text-sm">
                      <span className="text-zinc-900 dark:text-zinc-100">{ev.label}</span>
                      <span className="block text-xs text-zinc-500">{ev.hint}</span>
                    </span>
                  </label>
                ))}
              </div>
            </div>
          </fieldset>

          {isAdmin && (
            <div className="flex flex-wrap justify-end gap-2 border-t border-zinc-200 dark:border-zinc-800 pt-4">
              {(appConfigured || connected) && (
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => {
                    if (
                      confirm(
                        'Remove the Slack integration? The channel will stop receiving portal activity and the app credentials will be cleared.',
                      )
                    ) {
                      disconnectMut.mutate()
                    }
                  }}
                  isLoading={disconnectMut.isPending}
                >
                  {connected ? 'Disconnect' : 'Remove'}
                </Button>
              )}
              <Button type="submit" isLoading={saveMut.isPending}>
                {connected ? 'Save changes' : 'Save credentials'}
              </Button>
              {appConfigured && (
                <Button
                  type="button"
                  onClick={openSlackPopup}
                  disabled={credsDirty}
                  title={credsDirty ? 'Save your changes first' : undefined}
                >
                  {connected ? 'Change channel' : 'Add to Slack'}
                </Button>
              )}
            </div>
          )}
        </form>
      </div>
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
