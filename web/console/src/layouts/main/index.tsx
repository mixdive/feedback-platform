import { Link, NavLink, Outlet, useNavigate } from 'react-router-dom'
import {
  Bug,
  ExternalLink,
  Inbox,
  LayoutDashboard,
  LifeBuoy,
  Lightbulb,
  LogOut,
  MessageSquare,
  Rocket,
  Settings as SettingsIcon,
  Tag,
  Users,
} from 'lucide-react'
import clsx from 'clsx'

import { API } from '@/services/api'
import { useAppDispatch, useAppSelector } from '@/store/hooks'
import { authCleared } from '@/store'
import { message } from '@/utils/helpers'

type NavItem = { to: string; label: string; icon: typeof MessageSquare }

const rootItems: NavItem[] = [
  { to: '/inbox', label: 'Inbox', icon: Inbox },
  { to: '/dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { to: '/entry', label: 'All Feedback', icon: MessageSquare },
  { to: '/feature-requests', label: 'Feature Requests', icon: Lightbulb },
  { to: '/bugs', label: 'Bugs', icon: Bug },
  { to: '/support', label: 'Support', icon: LifeBuoy },
  { to: '/topics', label: 'Topics', icon: Tag },
  { to: '/releases', label: 'Releases', icon: Rocket },
  { to: '/users', label: 'Users', icon: Users },
]

export default function MainLayout() {
  const dispatch = useAppDispatch()
  const navigate = useNavigate()
  const user = useAppSelector((s) => s.auth.user)

  const handleLogout = async () => {
    try {
      await API().auth.logout()
    } catch (err) {
      message(err)
    } finally {
      dispatch(authCleared())
      navigate('/login', { replace: true })
    }
  }

  const initial = user?.email?.[0]?.toUpperCase() ?? '?'

  return (
    <div className="grid h-full grid-cols-[240px_1fr] bg-canvas dark:bg-zinc-950 text-zinc-900 dark:text-zinc-100">
      <aside className="flex flex-col border-r border-zinc-200/80 dark:border-zinc-800 bg-white/60 dark:bg-zinc-950/40 p-4">
        <Link
          to="/entry"
          className="mb-6 flex items-center gap-2 px-2 text-lg font-semibold"
        >
          <img
            src="/console/mixdive-icon.svg"
            alt=""
            className="size-6 rounded object-contain"
          />
          <span className="truncate">Mixdive</span>
        </Link>
        <nav className="flex-1 space-y-4">
          <div className="space-y-1">
            {rootItems.map(({ to, label, icon: Icon }) => (
              <NavLink
                key={to}
                to={to}
                className={({ isActive }) =>
                  clsx(
                    'flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-all duration-150',
                    isActive
                      ? 'bg-brand-50 text-brand-700 font-medium dark:bg-brand-500/15 dark:text-brand-300'
                      : 'text-zinc-600 dark:text-zinc-400 hover:bg-zinc-100/70 dark:hover:bg-zinc-900',
                  )
                }
              >
                {({ isActive }) => (
                  <>
                    <Icon className={clsx('size-4', isActive && 'text-brand-600 dark:text-brand-300')} />
                    {label}
                  </>
                )}
              </NavLink>
            ))}
          </div>

        </nav>

        <div className="mt-4 space-y-1 border-t border-zinc-200/80 dark:border-zinc-800 pt-4">
          <a
            href="/"
            className="flex items-center gap-3 rounded-lg px-3 py-2 text-sm text-zinc-600 dark:text-zinc-400 hover:bg-zinc-100/70 dark:hover:bg-zinc-900 transition-colors"
          >
            <ExternalLink className="size-4" />
            Go to Portal
          </a>

          <NavLink
            to="/settings"
            className={({ isActive }) =>
              clsx(
                'flex items-center gap-3 rounded-lg px-3 py-2 text-sm transition-all duration-150',
                isActive
                  ? 'bg-brand-50 text-brand-700 font-medium dark:bg-brand-500/15 dark:text-brand-300'
                  : 'text-zinc-600 dark:text-zinc-400 hover:bg-zinc-100/70 dark:hover:bg-zinc-900',
              )
            }
          >
            {({ isActive }) => (
              <>
                <SettingsIcon className={clsx('size-4', isActive && 'text-brand-600 dark:text-brand-300')} />
                Settings
              </>
            )}
          </NavLink>

          {user && (
            <div className="mt-2 rounded-xl border border-zinc-200/80 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-3 shadow-soft">
              <div className="flex items-center gap-3">
                <div className="grid size-8 shrink-0 place-items-center rounded-full bg-gradient-to-br from-brand-500 to-brand-700 text-xs font-semibold text-white shadow-soft">
                  {initial}
                </div>
                <div className="min-w-0 flex-1">
                  <div className="truncate text-sm font-medium" title={user.email}>
                    {user.email}
                  </div>
                  <div className="truncate text-xs text-zinc-500">
                    {user.roles.join(', ') || 'user'}
                  </div>
                </div>
              </div>
              <button
                type="button"
                onClick={handleLogout}
                className="mt-3 flex w-full items-center justify-center gap-2 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 py-1.5 text-sm text-zinc-700 dark:text-zinc-200 hover:bg-zinc-50 dark:hover:bg-zinc-800 transition-colors"
              >
                <LogOut className="size-4" />
                Log out
              </button>
            </div>
          )}
        </div>
      </aside>
      <div className="flex min-h-0 flex-col overflow-hidden">
        <main className="flex-1 overflow-auto p-8">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
