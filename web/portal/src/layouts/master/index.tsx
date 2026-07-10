import { useEffect, useRef, useState } from 'react'
import { Outlet } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { Toaster } from 'react-hot-toast'
import { useTranslation } from 'react-i18next'

import Splash from '@/components/splash'
import { API, ApiError } from '@/services/api'
import { setConfig } from '@/store/site'
import { useSiteConfig } from '@/store/site/hooks'
import { store } from '@/store'
import { authCleared, authLoading, authResolved } from '@/store'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { message } from '@/utils/helpers'

// MasterLayout owns three bootstrap responsibilities:
//   1. Load the public portal config so MainLayout can render branding.
//   2. Hydrate the auth slice from /api/me (or surface 401 → unauthenticated).
//   3. Detect a `?token=` callback from the admin's external auth provider,
//      exchange it for a session via /api/portal/auth/custom, then strip
//      the token from the URL.
export default function MasterLayout() {
  const { t } = useTranslation()
  const dispatch = useAppDispatch()
  const config = useSiteConfig()
  const authStatus = useAppSelector((s) => s.auth.status)
  const [exchanging, setExchanging] = useState(false)
  const exchangedRef = useRef(false)

  const { data: configData, error: configError } = useQuery({
    queryKey: ['portal-config'],
    queryFn: () => API().portal.config(),
    retry: false,
  })

  useEffect(() => {
    if (configData) store.dispatch(setConfig(configData))
  }, [configData])

  useEffect(() => {
    document.title = config?.projectName?.trim() || 'Mixdive'
  }, [config?.projectName])

  // Drive the browser-tab icon from the uploaded org logo. index.html ships
  // several bundled Mixdive icon links (svg + png + apple-touch); when a custom
  // logo exists we drop them and install a single logo-backed icon, otherwise
  // the browser may keep preferring the bundled SVG. No logo → defaults stay.
  useEffect(() => {
    const logoUrl = config?.logoUrl?.trim()
    if (!logoUrl) return
    const head = document.head
    head
      .querySelectorAll(
        'link[rel~="icon"]:not([data-portal-favicon]), link[rel="apple-touch-icon"]:not([data-portal-favicon])',
      )
      .forEach((el) => el.remove())
    const ensure = (rel: string) => {
      let el = head.querySelector<HTMLLinkElement>(
        `link[data-portal-favicon][rel="${rel}"]`,
      )
      if (!el) {
        el = document.createElement('link')
        el.setAttribute('rel', rel)
        el.setAttribute('data-portal-favicon', '')
        head.appendChild(el)
      }
      el.href = logoUrl
    }
    ensure('icon')
    ensure('apple-touch-icon')
  }, [config?.logoUrl])

  // Exchange the `?auth.custom=<jwt>` callback for a session before doing
  // anything else. Runs exactly once — exchangedRef guards re-entry from
  // React strict mode and from the URL-cleanup useEffect.
  useEffect(() => {
    if (exchangedRef.current) return
    const url = new URL(window.location.href)
    const token = url.searchParams.get('auth.custom')
    if (!token) return
    exchangedRef.current = true
    setExchanging(true)
    dispatch(authLoading())
    API()
      .auth.customLogin(token)
      .then((res) => {
        dispatch(authResolved(res.user))
        url.searchParams.delete('auth.custom')
        window.history.replaceState({}, '', url.toString())
      })
      .catch((err) => {
        message(err)
        dispatch(authCleared())
        url.searchParams.delete('auth.custom')
        window.history.replaceState({}, '', url.toString())
      })
      .finally(() => setExchanging(false))
  }, [dispatch])

  // /me bootstrap. Skipped until the token exchange has had a chance to run.
  useEffect(() => {
    if (exchanging) return
    if (authStatus !== 'idle') return
    dispatch(authLoading())
    API()
      .auth.me()
      .then((res) => dispatch(authResolved(res.user)))
      .catch((err) => {
        if (err instanceof ApiError && err.status === 401) {
          dispatch(authCleared())
        } else {
          dispatch(authCleared())
        }
      })
  }, [authStatus, exchanging, dispatch])

  if (configError) {
    return (
      <div className="flex h-full w-full items-center justify-center bg-zinc-50 p-6 text-center">
        <p className="max-w-sm text-zinc-600">
          {(configError as Error).message || t('portalUnavailable')}
        </p>
      </div>
    )
  }

  if (!config || authStatus === 'idle' || authStatus === 'loading' || exchanging) {
    return <Splash />
  }

  return (
    <>
      <Outlet />
      <Toaster position="bottom-right" />
    </>
  )
}
