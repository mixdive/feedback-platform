import { NavLink, Outlet } from 'react-router-dom'
import clsx from 'clsx'

const sections = [
  { to: 'profile', label: 'My Profile' },
  { to: 'project', label: 'Project' },
  { to: 'portal', label: 'Authentication' },
  { to: 'feedback', label: 'Feedback' },
  { to: 'ai', label: 'AI' },
  { to: 'integrations', label: 'Integrations' },
  { to: 'administrators', label: 'Team' },
]

export default function SettingsLayout() {
  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-semibold">Settings</h1>
      <div className="grid grid-cols-[200px_1fr] gap-8">
        <nav className="space-y-1">
          {sections.map(({ to, label }) => (
            <NavLink
              key={to}
              to={to}
              className={({ isActive }) =>
                clsx(
                  'block rounded-md px-3 py-2 text-sm transition-colors',
                  isActive
                    ? 'bg-zinc-200 dark:bg-zinc-800'
                    : 'text-zinc-600 dark:text-zinc-400 hover:bg-zinc-100 dark:hover:bg-zinc-900',
                )
              }
            >
              {label}
            </NavLink>
          ))}
        </nav>
        <div className="min-w-0">
          <Outlet />
        </div>
      </div>
    </div>
  )
}
