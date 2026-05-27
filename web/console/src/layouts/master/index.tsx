import { useEffect } from 'react'
import { Outlet, useLocation, useNavigate } from 'react-router-dom'
import { Toaster } from 'react-hot-toast'

import Splash from '@/components/splash'
import { API, ApiError } from '@/services/api'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { authCleared, authLoading, authResolved } from '@/store'

// MasterLayout owns the auth bootstrap. On load it calls /api/me; the
// result hydrates the auth slice. Routes that need a session use
// <AuthGuard>; the login route is accessible regardless of auth state.
// Authenticated users without the admin role are bounced out to the
// portal (one origin, so a hard navigation is the simplest way to leave
// the /console basename).
export default function MasterLayout() {
  const dispatch = useAppDispatch()
  const status = useAppSelector((s) => s.auth.status)
  const user = useAppSelector((s) => s.auth.user)
  const navigate = useNavigate()
  const location = useLocation()

  useEffect(() => {
    if (status !== 'idle') return
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
  }, [status, dispatch])

  useEffect(() => {
    if (status !== 'unauthenticated') return
    if (location.pathname === '/login') return
    navigate('/login', { replace: true, state: { from: location.pathname } })
  }, [status, location.pathname, navigate])

  useEffect(() => {
    if (status !== 'authenticated') return
    if (user?.roles?.includes('admin')) return
    window.location.replace('/')
  }, [status, user])

  if (status === 'idle' || status === 'loading') {
    return <Splash />
  }

  return (
    <>
      <Outlet />
      <Toaster position="bottom-right" />
    </>
  )
}
