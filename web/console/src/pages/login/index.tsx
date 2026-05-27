import { useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'

import Button from '@/components/button'
import { API } from '@/services/api'
import { useAppDispatch } from '@/store/hooks'
import { authResolved } from '@/store'
import { message } from '@/utils/helpers'

export default function LoginPage() {
  const dispatch = useAppDispatch()
  const navigate = useNavigate()
  const location = useLocation()
  const [email, setEmail] = useState('')
  const [password, setPassword] = useState('')
  const [submitting, setSubmitting] = useState(false)

  const submit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!email.trim() || !password) return
    setSubmitting(true)
    try {
      const res = await API().auth.login({ email: email.trim(), password })
      dispatch(authResolved(res.user))
      const next = (location.state as { from?: string } | null)?.from ?? '/entry'
      navigate(next, { replace: true })
    } catch (err) {
      message(err)
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="grid h-full place-items-center bg-zinc-50 dark:bg-zinc-950 text-zinc-900 dark:text-zinc-100 p-6">
      <form
        onSubmit={submit}
        className="w-full max-w-sm space-y-5 rounded-xl border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-8 shadow-sm"
      >
        <div className="flex justify-center">
          <img
            src="/console/mixdive-lockup-horizontal.svg"
            alt="Mixdive"
            className="h-9 w-auto dark:invert"
          />
        </div>
        <div className="space-y-1 text-center">
          <h1 className="text-xl font-semibold">Sign in</h1>
          <p className="text-sm text-zinc-500">Console access is admin-only.</p>
        </div>

        <div>
          <label htmlFor="email" className="block text-sm font-medium mb-1">
            Email
          </label>
          <input
            id="email"
            type="email"
            autoComplete="email"
            required
            value={email}
            onChange={(e) => setEmail(e.target.value)}
            className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none"
          />
        </div>

        <div>
          <label htmlFor="password" className="block text-sm font-medium mb-1">
            Password
          </label>
          <input
            id="password"
            type="password"
            autoComplete="current-password"
            required
            value={password}
            onChange={(e) => setPassword(e.target.value)}
            className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none"
          />
        </div>

        <Button type="submit" isLoading={submitting} className="w-full">
          Sign in
        </Button>
      </form>
    </div>
  )
}
