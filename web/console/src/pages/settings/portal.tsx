import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import Button from '@/components/button'
import ReadOnlyBanner from '@/components/read-only-banner'
import { API, type ApiSettingsPatch } from '@/services/api'
import { useAppSelector } from '@/store/hooks'
import { message } from '@/utils/helpers'

type AuthForm = {
  customAuthEnabled: boolean
  authUrl: string
  customAuthButtonText: string
  googleAuthEnabled: boolean
  googleClientId: string
}

// Render a JWT private key like "abcdABCD••••••••WXYZwxyz" — enough head/tail
// for the admin to recognize the value at a glance, but with the bulk hidden
// so a screenshare / over-the-shoulder view doesn't leak the full secret.
// The full key is still copied verbatim by the Copy button.
function maskKey(key: string): string {
  if (!key) return ''
  if (key.length <= 12) return '•'.repeat(key.length)
  const head = key.slice(0, 6)
  const tail = key.slice(-6)
  return `${head}${'•'.repeat(12)}${tail}`
}

export default function PortalSettingsPage() {
  const queryClient = useQueryClient()
  const isAdmin = !!useAppSelector((s) => s.auth.user?.roles?.includes('admin'))

  const [authForm, setAuthForm] = useState<AuthForm | null>(null)

  // The deployment origin the admin must register under Google's
  // "Authorized JavaScript origins". The Console and Portal are served
  // from the same origin (single binary), so this is exactly the value
  // Google needs.
  const origin = typeof window !== 'undefined' ? window.location.origin : ''

  const { data, isLoading, error } = useQuery({
    queryKey: ['console', 'settings'],
    queryFn: () => API().console.getSettings(),
  })

  useEffect(() => {
    if (!data) return
    if (!authForm) {
      setAuthForm({
        customAuthEnabled: data.portal.customAuthEnabled,
        authUrl: data.portal.authUrl,
        customAuthButtonText: data.portal.customAuthButtonText,
        googleAuthEnabled: data.portal.googleAuthEnabled,
        googleClientId: data.portal.googleClientId,
      })
    }
  }, [data, authForm])

  const authMut = useMutation({
    mutationFn: (body: ApiSettingsPatch) => API().console.updateSettings(body),
    onSuccess: () => {
      message('Authentication saved', 'success')
      void queryClient.invalidateQueries({ queryKey: ['console', 'settings'] })
    },
    onError: (e) => message(e),
  })

  // Auth URL is only required when custom auth is on — mirrors the server
  // invariant (customAuthEnabled=true requires a usable absolute URL).
  const authUrlError = (() => {
    if (!authForm) return null
    if (!authForm.customAuthEnabled) return null
    const v = authForm.authUrl.trim()
    if (!v) return 'Auth URL is required.'
    try {
      const u = new URL(v)
      if (u.protocol !== 'http:' && u.protocol !== 'https:') {
        return 'Auth URL must use http or https.'
      }
    } catch {
      return 'Auth URL must be a valid absolute URL.'
    }
    return null
  })()

  // Client ID is only required when Google auth is on — mirrors the server
  // invariant (googleAuthEnabled=true requires a non-empty Client ID).
  const googleClientIdError = (() => {
    if (!authForm) return null
    if (!authForm.googleAuthEnabled) return null
    if (!authForm.googleClientId.trim()) return 'Google Client ID is required.'
    return null
  })()

  const maskedKey = useMemo(() => maskKey(data?.portal.jwtPrivateKey ?? ''), [data?.portal.jwtPrivateKey])

  const copyKey = async () => {
    if (!data?.portal.jwtPrivateKey) return
    try {
      await navigator.clipboard.writeText(data.portal.jwtPrivateKey)
      message('Key copied to clipboard', 'success')
    } catch (err) {
      message(err)
    }
  }

  if (isLoading) return <p className="text-zinc-500">Loading…</p>
  if (error) return <p className="text-rose-600">{(error as Error).message}</p>
  if (!authForm || !data) return null

  const formInvalid = !!authUrlError || !!googleClientIdError

  return (
    <div className="space-y-6">
      {!isAdmin && <ReadOnlyBanner />}
      <form
        className="space-y-6"
        onSubmit={(e) => {
          e.preventDefault()
          if (!isAdmin) return
          if (formInvalid) {
            message(authUrlError || googleClientIdError || 'Please fix the errors above.')
            return
          }
          authMut.mutate({
            portal: {
              customAuthEnabled: authForm.customAuthEnabled,
              authUrl: authForm.authUrl.trim(),
              customAuthButtonText: authForm.customAuthButtonText.trim(),
              googleAuthEnabled: authForm.googleAuthEnabled,
              googleClientId: authForm.googleClientId.trim(),
            },
          })
        }}
      >
        <p className="text-sm text-zinc-500">
          Enable one or more sign-in methods for the portal. When several are on, every
          enabled method appears on the portal login screen.
        </p>

        <fieldset disabled={!isAdmin} className="space-y-6 disabled:opacity-70">
          {/* Custom authentication */}
          <div className="space-y-4 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-6">
            <label className="flex items-start gap-3">
              <input
                type="checkbox"
                checked={authForm.customAuthEnabled}
                onChange={(e) =>
                  setAuthForm({ ...authForm, customAuthEnabled: e.target.checked })
                }
                className="mt-0.5"
              />
              <span>
                <span className="block text-sm font-medium">Custom authentication</span>
                <span className="block text-xs text-zinc-500 mt-0.5">
                  Your users sign in to the portal via your own authentication system, which
                  redirects back with a signed JWT.
                </span>
              </span>
            </label>

            <div className={authForm.customAuthEnabled ? '' : 'opacity-50'}>
              <label className="block text-sm font-medium mb-1">Auth URL</label>
              <input
                type="url"
                value={authForm.authUrl}
                disabled={!authForm.customAuthEnabled}
                onChange={(e) => setAuthForm({ ...authForm, authUrl: e.target.value })}
                placeholder="https://your-app.example.com/portal-auth"
                className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none disabled:cursor-not-allowed"
              />
              {authUrlError ? (
                <p className="text-xs text-rose-600 mt-1">{authUrlError}</p>
              ) : (
                <p className="text-xs text-zinc-500 mt-1">
                  The portal redirects unauthenticated visitors here to obtain a signed JWT.
                </p>
              )}
            </div>

            <div className={authForm.customAuthEnabled ? '' : 'opacity-50'}>
              <label className="block text-sm font-medium mb-1">Login button text</label>
              <input
                type="text"
                value={authForm.customAuthButtonText}
                disabled={!authForm.customAuthEnabled}
                onChange={(e) =>
                  setAuthForm({ ...authForm, customAuthButtonText: e.target.value })
                }
                placeholder="Custom Login"
                className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none disabled:cursor-not-allowed"
              />
              <p className="text-xs text-zinc-500 mt-1">
                Optional. Shown on the portal login button. Leave empty to use the default.
              </p>
            </div>

            {isAdmin && data.portal.jwtPrivateKey && (
              <div className={authForm.customAuthEnabled ? '' : 'opacity-50'}>
                <label className="block text-sm font-medium mb-1">Private key</label>
                <div className="flex items-stretch gap-2">
                  <input
                    value={maskedKey}
                    readOnly
                    className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-950 px-3 font-mono text-xs text-zinc-700 dark:text-zinc-300"
                  />
                  <Button type="button" variant="ghost" onClick={copyKey}>
                    Copy
                  </Button>
                </div>
                <p className="text-xs text-zinc-500 mt-1">
                  Generated at setup. Use it on your auth page to sign portal JWTs (HS256). The
                  value is partially hidden here; click Copy to copy the full key. Treat as a
                  secret.
                </p>
              </div>
            )}
          </div>

          {/* Sign in with Google */}
          <div className="space-y-4 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-6">
            <label className="flex items-start gap-3">
              <input
                type="checkbox"
                checked={authForm.googleAuthEnabled}
                onChange={(e) =>
                  setAuthForm({ ...authForm, googleAuthEnabled: e.target.checked })
                }
                className="mt-0.5"
              />
              <span>
                <span className="block text-sm font-medium">Sign in with Google</span>
                <span className="block text-xs text-zinc-500 mt-0.5">
                  Let users sign in to the portal with their Google account.
                </span>
              </span>
            </label>

            <div className={authForm.googleAuthEnabled ? '' : 'opacity-50'}>
              <label className="block text-sm font-medium mb-1">Google Client ID</label>
              <input
                type="text"
                value={authForm.googleClientId}
                disabled={!authForm.googleAuthEnabled}
                onChange={(e) => setAuthForm({ ...authForm, googleClientId: e.target.value })}
                placeholder="1234567890-abcdef.apps.googleusercontent.com"
                className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 font-mono text-xs focus:border-sky-500 focus:outline-none disabled:cursor-not-allowed"
              />
              {googleClientIdError ? (
                <p className="text-xs text-rose-600 mt-1">{googleClientIdError}</p>
              ) : (
                <p className="text-xs text-zinc-500 mt-1">
                  Not a secret — the portal embeds it in the browser. No client secret is
                  required for this flow.
                </p>
              )}
            </div>

            {/* How to obtain a Google Client ID */}
            <div className="rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950/40 p-4">
              <p className="text-xs font-medium text-zinc-700 dark:text-zinc-300">
                How to get a Client ID
              </p>
              <ol className="mt-2 list-decimal space-y-1.5 pl-4 text-xs text-zinc-500">
                <li>
                  Open the{' '}
                  <a
                    href="https://console.cloud.google.com/apis/credentials"
                    target="_blank"
                    rel="noreferrer"
                    className="text-sky-600 hover:underline dark:text-sky-400"
                  >
                    Google Cloud Console → Credentials
                  </a>{' '}
                  page (create or pick a project first).
                </li>
                <li>
                  If prompted, configure the{' '}
                  <a
                    href="https://console.cloud.google.com/apis/credentials/consent"
                    target="_blank"
                    rel="noreferrer"
                    className="text-sky-600 hover:underline dark:text-sky-400"
                  >
                    OAuth consent screen
                  </a>{' '}
                  once (User type “External”, add your app name and support email).
                </li>
                <li>
                  Click <span className="font-medium">Create credentials → OAuth client ID</span>,
                  and choose application type <span className="font-medium">Web application</span>.
                </li>
                <li>
                  Under <span className="font-medium">Authorized JavaScript origins</span>, add this
                  portal&apos;s origin:
                  <code className="ml-1 rounded bg-zinc-200 px-1 py-0.5 font-mono text-[11px] text-zinc-800 dark:bg-zinc-800 dark:text-zinc-200">
                    {origin}
                  </code>
                </li>
                <li>
                  Click <span className="font-medium">Create</span>, then copy the
                  generated <span className="font-medium">Client ID</span> into the field above.
                  The client secret is not needed.
                </li>
              </ol>
            </div>
          </div>
        </fieldset>

        {isAdmin && (
          <div className="flex justify-end">
            <Button type="submit" isLoading={authMut.isPending} disabled={formInvalid}>
              Save
            </Button>
          </div>
        )}
      </form>
    </div>
  )
}
