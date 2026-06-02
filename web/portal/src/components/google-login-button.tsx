import { useEffect, useRef } from 'react'

import { API } from '@/services/api'
import { authCleared, authLoading, authResolved } from '@/store'
import { useAppDispatch } from '@/store/hooks'
import { message } from '@/utils/helpers'

// Minimal typing of the slice of Google Identity Services we use. The full
// library is loaded at runtime from Google's CDN, so we declare only the
// `google.accounts.id` surface the button touches.
interface GoogleCredentialResponse {
  credential: string
}

interface GoogleIdApi {
  initialize: (config: {
    client_id: string
    callback: (response: GoogleCredentialResponse) => void
  }) => void
  renderButton: (parent: HTMLElement, options: Record<string, unknown>) => void
}

declare global {
  interface Window {
    google?: { accounts?: { id?: GoogleIdApi } }
  }
}

const GIS_SRC = 'https://accounts.google.com/gsi/client'

// loadGis injects the Google Identity Services script once and resolves when
// the `google.accounts.id` API is available. Concurrent callers share one
// promise so the script is never added twice.
let gisPromise: Promise<void> | null = null
function loadGis(): Promise<void> {
  if (window.google?.accounts?.id) return Promise.resolve()
  if (gisPromise) return gisPromise
  gisPromise = new Promise<void>((resolve, reject) => {
    const fail = () => reject(new Error('Failed to load Google sign-in.'))
    const existing = document.querySelector<HTMLScriptElement>(`script[src="${GIS_SRC}"]`)
    if (existing) {
      existing.addEventListener('load', () => resolve())
      existing.addEventListener('error', fail)
      return
    }
    const s = document.createElement('script')
    s.src = GIS_SRC
    s.async = true
    s.defer = true
    s.onload = () => resolve()
    s.onerror = fail
    document.head.appendChild(s)
  })
  return gisPromise
}

// GoogleLoginButton renders the official Google Identity Services button. On
// a successful sign-in GIS hands us a signed ID token (the `credential`),
// which we exchange for a Mixdive session via POST /api/portal/auth/google.
export default function GoogleLoginButton({ clientId }: { clientId: string }) {
  const dispatch = useAppDispatch()
  const containerRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    let cancelled = false
    loadGis()
      .then(() => {
        if (cancelled) return
        const id = window.google?.accounts?.id
        const container = containerRef.current
        if (!id || !container) return
        id.initialize({
          client_id: clientId,
          callback: (response) => {
            dispatch(authLoading())
            API()
              .auth.googleLogin(response.credential)
              .then((res) => dispatch(authResolved(res.user)))
              .catch((err) => {
                message(err)
                dispatch(authCleared())
              })
          },
        })
        container.innerHTML = ''
        id.renderButton(container, {
          type: 'standard',
          theme: 'outline',
          size: 'large',
          text: 'signin_with',
          shape: 'rectangular',
        })
      })
      .catch((err) => message(err))
    return () => {
      cancelled = true
    }
  }, [clientId, dispatch])

  return <div ref={containerRef} className="flex justify-center" />
}
