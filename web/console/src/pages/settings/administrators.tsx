import { Fragment, useEffect, useMemo, useState } from 'react'
import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react'
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'
import { Plus, Search, ShieldCheck, Trash2, X } from 'lucide-react'

import Button from '@/components/button'
import {
  API,
  type ApiAccountType,
  type ApiAdministrator,
  type ApiUserRow,
} from '@/services/api'

const accountLabels: Record<ApiAccountType, string> = {
  email: 'Email',
  custom: 'Custom',
  google: 'Google',
}

type EditableRole = 'admin' | 'editor'
const editableRoles: EditableRole[] = ['admin', 'editor']

function displayLabel(u: { email: string; name?: string; username?: string }): string {
  return u.username || u.name || u.email || '—'
}

export default function AdministratorsSettingsPage() {
  const [editing, setEditing] = useState<ApiAdministrator | null>(null)
  const [adding, setAdding] = useState(false)

  const { data, isLoading, error } = useQuery({
    queryKey: ['console', 'administrators'],
    queryFn: () => API().console.listAdministrators(),
  })

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-lg font-semibold">Team</h2>
          <p className="text-sm text-zinc-500">
            Admins manage other team members; editors can use everything else in the Console.
          </p>
        </div>
        <Button size="sm" onClick={() => setAdding(true)}>
          <Plus className="size-4" />
          Add team member
        </Button>
      </div>

      {isLoading && <p className="text-zinc-500">Loading…</p>}
      {error && <p className="text-rose-600">{(error as Error).message}</p>}

      {data && data.data.length === 0 && (
        <div className="rounded-lg border border-dashed border-zinc-300 dark:border-zinc-700 p-12 text-center text-zinc-500">
          No team members yet.
        </div>
      )}

      {data && data.data.length > 0 && (
        <div className="overflow-hidden rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900">
          <table className="w-full text-sm">
            <thead className="border-b border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950 text-left text-xs uppercase tracking-wide text-zinc-500">
              <tr>
                <th className="px-4 py-2 font-medium">Member</th>
                <th className="px-4 py-2 font-medium">Accounts</th>
                <th className="px-4 py-2 font-medium">Roles</th>
                <th className="px-4 py-2 font-medium">Status</th>
                <th className="px-4 py-2 font-medium">Created</th>
                <th className="px-4 py-2 font-medium" />
              </tr>
            </thead>
            <tbody>
              {data.data.map((u) => (
                <tr
                  key={u.id}
                  className="border-t border-zinc-200 dark:border-zinc-800 first:border-t-0"
                >
                  <td className="px-4 py-3">
                    <div className="flex items-center gap-3">
                      <Avatar user={u} />
                      <div className="min-w-0">
                        <div className="truncate font-medium">{displayLabel(u)}</div>
                        {u.email && u.email !== displayLabel(u) && (
                          <div className="truncate text-xs text-zinc-500">{u.email}</div>
                        )}
                      </div>
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <div className="flex flex-wrap gap-1.5">
                      {u.accounts.map((a) => (
                        <span
                          key={a}
                          className="inline-flex items-center rounded border border-zinc-300 dark:border-zinc-700 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-zinc-600 dark:text-zinc-400"
                        >
                          {accountLabels[a] ?? a}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="px-4 py-3">
                    <RoleBadges roles={u.roles} />
                  </td>
                  <td className="px-4 py-3">
                    {u.isBlocked ? (
                      <span className="text-rose-600">Blocked</span>
                    ) : (
                      <span className="text-emerald-600">Active</span>
                    )}
                  </td>
                  <td className="px-4 py-3 text-zinc-500">
                    {dayjs(u.createdAt).format('MMM D, YYYY')}
                  </td>
                  <td className="px-4 py-3 text-right">
                    <Button variant="ghost" size="sm" onClick={() => setEditing(u)}>
                      Edit
                    </Button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {editing && (
        <EditMemberDialog
          member={editing}
          onClose={() => setEditing(null)}
        />
      )}
      {adding && <AddMemberDialog onClose={() => setAdding(false)} />}
    </div>
  )
}



function RoleBadges({ roles }: { roles: string[] }) {
  if (roles.length === 0) {
    return <span className="text-xs text-zinc-500">—</span>
  }
  return (
    <div className="flex flex-wrap gap-1.5">
      {roles.map((r) => (
        <span
          key={r}
          className="inline-flex items-center rounded-full bg-sky-100 dark:bg-sky-900/40 px-2 py-0.5 text-xs font-medium text-sky-800 dark:text-sky-200"
        >
          {r}
        </span>
      ))}
    </div>
  )
}

function Avatar({ user }: { user: { imageUrl?: string; name?: string; username?: string; email?: string } }) {
  if (user.imageUrl) {
    return (
      <img
        src={user.imageUrl}
        alt=""
        className="size-8 shrink-0 rounded-full object-cover"
      />
    )
  }
  const initial = (
    user.name?.[0] ??
    user.username?.[0] ??
    user.email?.[0] ??
    '?'
  ).toUpperCase()
  return (
    <div className="grid size-8 shrink-0 place-items-center rounded-full bg-sky-600 text-xs font-semibold text-white">
      {initial}
    </div>
  )
}

function EditMemberDialog({
  member,
  onClose,
}: {
  member: ApiAdministrator
  onClose: () => void
}) {
  const queryClient = useQueryClient()
  const [roles, setRoles] = useState<EditableRole[]>(
    () => member.roles.filter((r): r is EditableRole => editableRoles.includes(r as EditableRole)),
  )
  const [serverError, setServerError] = useState<string | null>(null)

  const rolesMut = useMutation({
    mutationFn: (next: EditableRole[]) =>
      API().console.updateUserRoles(member.id, { roles: next }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['console', 'administrators'] })
      onClose()
    },
    onError: (e) => setServerError((e as Error).message),
  })

  const blockMut = useMutation({
    mutationFn: (next: boolean) =>
      API().console.updateUserBlock(member.id, { isBlocked: next }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['console', 'administrators'] })
      onClose()
    },
    onError: (e) => setServerError((e as Error).message),
  })

  function toggleRole(r: EditableRole) {
    setServerError(null)
    setRoles((prev) =>
      prev.includes(r) ? prev.filter((x) => x !== r) : [...prev, r],
    )
  }

  const willRemoveFromTeam = roles.length === 0
  const dirty =
    roles.slice().sort().join(',') !==
    member.roles
      .filter((r): r is EditableRole => editableRoles.includes(r as EditableRole))
      .slice()
      .sort()
      .join(',')

  return (
    <Dialog open onClose={onClose} className="relative z-50">
      <div className="fixed inset-0 bg-zinc-950/60 backdrop-blur-sm" aria-hidden="true" />
      <div className="fixed inset-0 flex items-center justify-center p-4">
        <DialogPanel className="w-full max-w-md rounded-xl bg-white dark:bg-zinc-900 text-zinc-900 dark:text-zinc-100 shadow-2xl">
          <div className="flex items-start gap-3 border-b border-zinc-200 dark:border-zinc-800 p-5">
            <Avatar user={member} />
            <div className="min-w-0 flex-1">
              <DialogTitle className="text-base font-semibold leading-snug">
                {displayLabel(member)}
              </DialogTitle>
              {member.email && member.email !== displayLabel(member) && (
                <div className="truncate text-xs text-zinc-500">{member.email}</div>
              )}
            </div>
            <button
              type="button"
              onClick={onClose}
              aria-label="Close"
              className="rounded-md p-1.5 text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800"
            >
              <X className="size-4" />
            </button>
          </div>

          <div className="space-y-5 p-5">
            <div>
              <div className="mb-2 text-sm font-medium">Roles</div>
              <div className="space-y-2">
                {editableRoles.map((r) => (
                  <label
                    key={r}
                    className="flex cursor-pointer items-start gap-3 rounded-md border border-zinc-200 dark:border-zinc-800 p-3 hover:border-zinc-300 dark:hover:border-zinc-700"
                  >
                    <input
                      type="checkbox"
                      checked={roles.includes(r)}
                      onChange={() => toggleRole(r)}
                      className="mt-1"
                    />
                    <div className="text-sm">
                      <div className="font-medium capitalize">{r}</div>
                      <div className="text-xs text-zinc-500">{roleDescription(r)}</div>
                    </div>
                  </label>
                ))}
              </div>
              {willRemoveFromTeam && (
                <p className="mt-2 text-xs text-amber-600">
                  Saving with no roles selected removes this user from the team.
                </p>
              )}
            </div>

            <div>
              <div className="mb-2 text-sm font-medium">Status</div>
              <div className="flex items-center justify-between rounded-md border border-zinc-200 dark:border-zinc-800 p-3">
                <div className="text-sm">
                  <div className="font-medium">
                    {member.isBlocked ? 'Blocked' : 'Active'}
                  </div>
                  <div className="text-xs text-zinc-500">
                    {member.isBlocked
                      ? 'This member cannot sign in. Unblock to restore access.'
                      : 'Block to suspend access without removing roles.'}
                  </div>
                </div>
                <Button
                  variant={member.isBlocked ? 'primary' : 'danger'}
                  size="sm"
                  isLoading={blockMut.isPending}
                  onClick={() => blockMut.mutate(!member.isBlocked)}
                >
                  {member.isBlocked ? 'Unblock' : 'Block'}
                </Button>
              </div>
            </div>

            {serverError && (
              <div className="rounded-md border border-rose-300 bg-rose-50 dark:bg-rose-900/20 dark:border-rose-800 p-3 text-sm text-rose-700 dark:text-rose-300">
                {serverError}
              </div>
            )}
          </div>

          <div className="flex items-center justify-end gap-2 border-t border-zinc-200 dark:border-zinc-800 p-4">
            <Button variant="ghost" size="sm" onClick={onClose}>
              Cancel
            </Button>
            <Button
              size="sm"
              isLoading={rolesMut.isPending}
              disabled={!dirty}
              onClick={() => rolesMut.mutate(roles)}
            >
              {willRemoveFromTeam ? (
                <>
                  <Trash2 className="size-4" />
                  Remove from team
                </>
              ) : (
                <>
                  <ShieldCheck className="size-4" />
                  Save roles
                </>
              )}
            </Button>
          </div>
        </DialogPanel>
      </div>
    </Dialog>
  )
}

function roleDescription(r: EditableRole): string {
  switch (r) {
    case 'admin':
      return 'Full Console access, including managing other team members.'
    case 'editor':
      return 'Manage settings and feedback, but cannot change the team.'
  }
}

function AddMemberDialog({ onClose }: { onClose: () => void }) {
  const queryClient = useQueryClient()
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [picked, setPicked] = useState<ApiUserRow | null>(null)
  const [roles, setRoles] = useState<EditableRole[]>(['editor'])
  const [serverError, setServerError] = useState<string | null>(null)

  useEffect(() => {
    const t = setTimeout(() => setSearch(searchInput.trim()), 250)
    return () => clearTimeout(t)
  }, [searchInput])

  // Search across the full users collection — matches by name, email, or
  // username. Only surface users that don't already carry a role; once they
  // have one, the parent table is the place to manage them.
  const { data, isFetching } = useQuery({
    queryKey: ['console', 'users', 'add-member', { search }],
    queryFn: () =>
      API().console.listUsers({
        search: search || undefined,
        role: 'none',
        status: 'active',
        limit: 25,
      }),
    placeholderData: keepPreviousData,
    enabled: search.length > 0,
  })

  const assignMut = useMutation({
    mutationFn: ({ id, next }: { id: string; next: EditableRole[] }) =>
      API().console.updateUserRoles(id, { roles: next }),
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ['console', 'administrators'] })
      onClose()
    },
    onError: (e) => setServerError((e as Error).message),
  })

  function toggleRole(r: EditableRole) {
    setServerError(null)
    setRoles((prev) =>
      prev.includes(r) ? prev.filter((x) => x !== r) : [...prev, r],
    )
  }

  const results = useMemo(() => data?.data ?? [], [data])

  return (
    <Dialog open onClose={onClose} className="relative z-50">
      <div className="fixed inset-0 bg-zinc-950/60 backdrop-blur-sm" aria-hidden="true" />
      <div className="fixed inset-0 flex items-center justify-center p-4">
        <DialogPanel className="w-full max-w-lg rounded-xl bg-white dark:bg-zinc-900 text-zinc-900 dark:text-zinc-100 shadow-2xl">
          <div className="flex items-start gap-3 border-b border-zinc-200 dark:border-zinc-800 p-5">
            <DialogTitle className="flex-1 text-lg font-semibold leading-snug">
              Add team member
            </DialogTitle>
            <button
              type="button"
              onClick={onClose}
              aria-label="Close"
              className="rounded-md p-1.5 text-zinc-500 hover:bg-zinc-100 dark:hover:bg-zinc-800"
            >
              <X className="size-4" />
            </button>
          </div>

          {!picked ? (
            <Fragment>
              <div className="p-5 pb-3">
                <p className="mb-3 text-sm text-zinc-500">
                  Find an existing user by name, email, or username, then assign one or more
                  roles.
                </p>
                <div className="relative">
                  <Search className="pointer-events-none absolute left-2.5 top-1/2 size-4 -translate-y-1/2 text-zinc-400" />
                  <input
                    autoFocus
                    value={searchInput}
                    onChange={(e) => setSearchInput(e.target.value)}
                    placeholder="Search users…"
                    className="block h-10 w-full rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 pl-9 pr-9 text-sm focus:border-sky-500 focus:outline-none"
                  />
                  {searchInput && (
                    <button
                      type="button"
                      onClick={() => setSearchInput('')}
                      aria-label="Clear search"
                      className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-zinc-400 hover:text-zinc-700 dark:hover:text-zinc-200"
                    >
                      <X className="size-4" />
                    </button>
                  )}
                </div>
              </div>

              <div className="max-h-80 overflow-y-auto px-2 pb-3">
                {search === '' && (
                  <p className="px-3 py-8 text-center text-sm text-zinc-500">
                    Start typing to search.
                  </p>
                )}
                {search !== '' && !isFetching && results.length === 0 && (
                  <p className="px-3 py-8 text-center text-sm text-zinc-500">
                    No matching users without a role.
                  </p>
                )}
                {results.map((u) => (
                  <button
                    key={u.id}
                    type="button"
                    onClick={() => setPicked(u)}
                    className="flex w-full items-center gap-3 rounded-md p-2 text-left hover:bg-zinc-100 dark:hover:bg-zinc-800"
                  >
                    <Avatar user={u} />
                    <div className="min-w-0 flex-1">
                      <div className="truncate text-sm font-medium">{displayLabel(u)}</div>
                      {u.email && u.email !== displayLabel(u) && (
                        <div className="truncate text-xs text-zinc-500">{u.email}</div>
                      )}
                    </div>
                  </button>
                ))}
              </div>

              <div className="flex items-center justify-end gap-2 border-t border-zinc-200 dark:border-zinc-800 p-4">
                <Button variant="ghost" size="sm" onClick={onClose}>
                  Cancel
                </Button>
              </div>
            </Fragment>
          ) : (
            <Fragment>
              <div className="space-y-5 p-5">
                <div className="flex items-center gap-3 rounded-md border border-zinc-200 dark:border-zinc-800 p-3">
                  <Avatar user={picked} />
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-sm font-medium">{displayLabel(picked)}</div>
                    {picked.email && picked.email !== displayLabel(picked) && (
                      <div className="truncate text-xs text-zinc-500">{picked.email}</div>
                    )}
                  </div>
                  <button
                    type="button"
                    className="text-xs text-zinc-500 hover:text-zinc-700 dark:hover:text-zinc-200"
                    onClick={() => setPicked(null)}
                  >
                    Change
                  </button>
                </div>

                <div>
                  <div className="mb-2 text-sm font-medium">Roles</div>
                  <div className="space-y-2">
                    {editableRoles.map((r) => (
                      <label
                        key={r}
                        className="flex cursor-pointer items-start gap-3 rounded-md border border-zinc-200 dark:border-zinc-800 p-3 hover:border-zinc-300 dark:hover:border-zinc-700"
                      >
                        <input
                          type="checkbox"
                          checked={roles.includes(r)}
                          onChange={() => toggleRole(r)}
                          className="mt-1"
                        />
                        <div className="text-sm">
                          <div className="font-medium capitalize">{r}</div>
                          <div className="text-xs text-zinc-500">{roleDescription(r)}</div>
                        </div>
                      </label>
                    ))}
                  </div>
                </div>

                {serverError && (
                  <div className="rounded-md border border-rose-300 bg-rose-50 dark:bg-rose-900/20 dark:border-rose-800 p-3 text-sm text-rose-700 dark:text-rose-300">
                    {serverError}
                  </div>
                )}
              </div>

              <div className="flex items-center justify-end gap-2 border-t border-zinc-200 dark:border-zinc-800 p-4">
                <Button variant="ghost" size="sm" onClick={onClose}>
                  Cancel
                </Button>
                <Button
                  size="sm"
                  isLoading={assignMut.isPending}
                  disabled={roles.length === 0}
                  onClick={() => assignMut.mutate({ id: picked.id, next: roles })}
                >
                  <ShieldCheck className="size-4" />
                  Add to team
                </Button>
              </div>
            </Fragment>
          )}
        </DialogPanel>
      </div>
    </Dialog>
  )
}

