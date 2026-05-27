import { useEffect, useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import Button from '@/components/button'
import ReadOnlyBanner from '@/components/read-only-banner'
import { API, type ApiSettingsPatch } from '@/services/api'
import { useAppSelector } from '@/store/hooks'
import { message } from '@/utils/helpers'

type AuthForm = {
  authUrl: string
  customAuthButtonText: string
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

  const { data, isLoading, error } = useQuery({
    queryKey: ['console', 'settings'],
    queryFn: () => API().console.getSettings(),
  })

  useEffect(() => {
    if (!data) return
    if (!authForm) {
      setAuthForm({
        authUrl: data.portal.authUrl,
        customAuthButtonText: data.portal.customAuthButtonText,
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

  const authUrlError = (() => {
    if (!authForm) return null
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

  return (
    <div className="space-y-6">
      {!isAdmin && <ReadOnlyBanner />}
      <form
        className="space-y-4 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-6"
        onSubmit={(e) => {
          e.preventDefault()
          if (!isAdmin) return
          if (authUrlError) {
            message(authUrlError)
            return
          }
          authMut.mutate({
            portal: {
              customAuthEnabled: true,
              authUrl: authForm.authUrl.trim(),
              customAuthButtonText: authForm.customAuthButtonText.trim(),
            },
          })
        }}
      >
        <h2 className="text-sm font-semibold uppercase tracking-wide text-zinc-500">
          Authentication
        </h2>
        <fieldset disabled={!isAdmin} className="space-y-4 disabled:opacity-70">
          <label className="flex items-start gap-3">
            <input
              type="checkbox"
              checked
              disabled
              readOnly
              className="mt-0.5 cursor-not-allowed opacity-60"
            />
            <span>
              <span className="block text-sm font-medium">Custom authentication</span>
              <span className="block text-xs text-zinc-500 mt-0.5">
                Custom authentication is the only authentication method available right now,
                so it is always on. Your users sign in to the portal via your own
                authentication system.
              </span>
            </span>
          </label>

          <div>
            <label className="block text-sm font-medium mb-1">Auth URL</label>
            <input
              type="url"
              value={authForm.authUrl}
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

          <div>
            <label className="block text-sm font-medium mb-1">Login button text</label>
            <input
              type="text"
              value={authForm.customAuthButtonText}
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
        </fieldset>

        {isAdmin && data.portal.jwtPrivateKey && (
          <div>
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
              value is partially hidden here; click Copy to copy the full key. Treat as a secret.
            </p>
          </div>
        )}

        {isAdmin && (
          <div className="flex justify-end">
            <Button type="submit" isLoading={authMut.isPending} disabled={!!authUrlError}>
              Save
            </Button>
          </div>
        )}
      </form>
    </div>
  )
}
