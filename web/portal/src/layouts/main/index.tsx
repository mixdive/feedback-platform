import { Link, NavLink, Outlet } from 'react-router-dom'
import { LogIn, LogOut, ShieldCheck } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import clsx from 'clsx'

import GoogleLoginButton from '@/components/google-login-button'
import LanguagePicker from '@/components/language-picker'
import { API } from '@/services/api'
import { useSiteConfig } from '@/store/site/hooks'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { authCleared } from '@/store'
import { message } from '@/utils/helpers'

export default function MainLayout() {
  const { t } = useTranslation()
  const config = useSiteConfig()
  const dispatch = useAppDispatch()
  const user = useAppSelector((s) => s.auth.user)
  const status = useAppSelector((s) => s.auth.status)

  const topNavItems = [
    { to: '/', label: t('nav.feedback'), end: true },
    { to: '/changelog', label: t('nav.changelog'), end: false },
  ]

  const handleCustomLogin = () => {
    if (!config?.authUrl) return
    const callback = window.location.href
    const sep = config.authUrl.includes('?') ? '&' : '?'
    window.location.href = `${config.authUrl}${sep}callback_url=${encodeURIComponent(callback)}`
  }

  const handleLogout = async () => {
    try {
      await API().auth.logout()
    } catch (err) {
      message(err)
    } finally {
      dispatch(authCleared())
    }
  }

  const initial =
    user?.username?.[0]?.toUpperCase() ??
    user?.name?.[0]?.toUpperCase() ??
    user?.email?.[0]?.toUpperCase() ??
    '?'

  const customAuthOn = !!config?.customAuthEnabled && !!config?.authUrl
  const googleAuthOn = !!config?.googleAuthEnabled && !!config?.googleClientId
  const requireLogin = status === 'unauthenticated' && (customAuthOn || googleAuthOn)
  const isAdmin = !!user?.roles?.includes('admin')
  const loginLabel = config?.customAuthButtonText?.trim() || t('auth.defaultLoginLabel')

  return (
    <div className="min-h-full bg-canvas dark:bg-zinc-950">
      <header className="border-b border-zinc-200/70 dark:border-zinc-800 bg-white/80 backdrop-blur-sm dark:bg-zinc-900 shadow-soft">
        <div className="mx-auto flex max-w-5xl items-center gap-2 sm:gap-3 px-4 sm:px-6 py-3 sm:py-4">
          <Link to="/" className="flex min-w-0 items-center gap-2 sm:gap-3 text-zinc-900 dark:text-zinc-100 hover:opacity-80 transition-opacity">
            {/* Brand rule (Portal):
                - projectName set + logoUrl set  → custom logo + project name
                - projectName set, no logoUrl    → project name only (text)
                - projectName empty + logoUrl    → custom logo + "Mixdive"
                - both empty                     → bundled horizontal lockup
                Console always renders the Mixdive lockup regardless of
                project settings. */}
            {(() => {
              const projectName = config?.projectName?.trim() || ''
              const logoUrl = config?.logoUrl || ''
              if (projectName && logoUrl) {
                return (
                  <>
                    <img
                      src={logoUrl}
                      alt=""
                      className="size-7 rounded-md object-contain"
                    />
                    <div className="truncate text-base font-semibold">{projectName}</div>
                  </>
                )
              }
              if (projectName) {
                return (
                  <div className="text-base font-semibold">{projectName}</div>
                )
              }
              if (logoUrl) {
                return (
                  <>
                    <img
                      src={logoUrl}
                      alt=""
                      className="size-7 rounded-md object-contain"
                    />
                    <div className="text-base font-semibold">Mixdive</div>
                  </>
                )
              }
              return (
                <img
                  src="/mixdive-lockup-horizontal.svg"
                  alt="Mixdive"
                  className="h-7 w-auto object-contain dark:invert"
                />
              )
            })()}
          </Link>

          {!requireLogin && (
            <nav className="flex items-center gap-1 text-sm">
              {topNavItems.map(({ to, label, end }) => (
                <NavLink
                  key={to}
                  to={to}
                  end={end}
                  className={({ isActive }) =>
                    clsx(
                      'rounded-lg px-3 py-1.5 font-medium transition-all duration-150',
                      isActive
                        ? 'bg-brand-50 text-brand-700 dark:bg-brand-500/15 dark:text-brand-300'
                        : 'text-zinc-600 dark:text-zinc-400 hover:bg-zinc-100/70 dark:hover:bg-zinc-800',
                    )
                  }
                >
                  {label}
                </NavLink>
              ))}
            </nav>
          )}

          <div className="flex-1" />

          <LanguagePicker />

          {user && isAdmin && (
            <a
              href="/console/"
              className="inline-flex items-center gap-2 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-1.5 text-sm text-zinc-700 dark:text-zinc-200 hover:bg-zinc-50 dark:hover:bg-zinc-800 transition-colors"
            >
              <ShieldCheck className="size-4" />
              {t('common.goToConsole')}
            </a>
          )}

          {user && (
            <div className="flex items-center gap-2">
              {user.imageUrl ? (
                <img
                  src={user.imageUrl}
                  alt=""
                  className="size-7 rounded-full object-cover"
                />
              ) : (
                <div className="grid size-7 shrink-0 place-items-center rounded-full bg-gradient-to-br from-brand-500 to-brand-700 text-xs font-semibold text-white shadow-soft">
                  {initial}
                </div>
              )}
              <div className="hidden sm:block text-sm text-zinc-700 dark:text-zinc-200">
                {user.username || user.name || user.email || t('common.signedIn')}
              </div>
              <button
                type="button"
                onClick={handleLogout}
                title={t('common.logOut')}
                className="inline-flex items-center justify-center rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 size-8 text-zinc-700 dark:text-zinc-200 hover:bg-zinc-50 dark:hover:bg-zinc-800 transition-colors"
              >
                <LogOut className="size-4" />
              </button>
            </div>
          )}
        </div>
      </header>
      <main className="mx-auto max-w-5xl px-4 sm:px-6 py-6 sm:py-8">
        {requireLogin ? (
          <div className="rounded-lg border border-dashed border-zinc-300 dark:border-zinc-700 p-12 text-center">
            <h2 className="text-lg font-semibold">{t('auth.gateTitle')}</h2>
            <p className="mt-2 text-sm text-zinc-500">
              {config?.projectName
                ? t('auth.gateDescriptionWithProvider', { provider: config.projectName })
                : t('auth.gateDescriptionDefault')}
            </p>
            <div className="mt-6 flex flex-col items-center gap-3">
              {customAuthOn && (
                <button
                  type="button"
                  onClick={handleCustomLogin}
                  className="inline-flex items-center gap-2 rounded-lg bg-brand-600 px-4 py-2 text-sm font-medium text-white hover:bg-brand-500 shadow-soft transition-all"
                >
                  <LogIn className="size-4" />
                  {loginLabel}
                </button>
              )}
              {googleAuthOn && config?.googleClientId && (
                <GoogleLoginButton clientId={config.googleClientId} />
              )}
            </div>
          </div>
        ) : (
          <Outlet />
        )}
      </main>
    </div>
  )
}
