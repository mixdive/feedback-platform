import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Upload, X } from 'lucide-react'

import Button from '@/components/button'
import { API } from '@/services/api'
import { message } from '@/utils/helpers'

type ProfileForm = { name: string; imageUrl: string }
type PasswordForm = { current: string; next: string; repeat: string }

const emptyPassword: PasswordForm = { current: '', next: '', repeat: '' }

export default function MyProfileSettingsPage() {
  const queryClient = useQueryClient()
  const fileInputRef = useRef<HTMLInputElement>(null)

  const [form, setForm] = useState<ProfileForm | null>(null)
  const [password, setPassword] = useState<PasswordForm>(emptyPassword)
  const [uploading, setUploading] = useState(false)

  const { data, isLoading, error } = useQuery({
    queryKey: ['console', 'me'],
    queryFn: () => API().console.getMyProfile(),
  })

  useEffect(() => {
    if (!data || form) return
    setForm({ name: data.name ?? '', imageUrl: data.imageUrl ?? '' })
  }, [data, form])

  const profileMut = useMutation({
    mutationFn: (body: { name: string; imageUrl: string }) =>
      API().console.updateMyProfile(body),
    onSuccess: () => {
      message('Profile saved', 'success')
      void queryClient.invalidateQueries({ queryKey: ['console', 'me'] })
    },
    onError: (e) => message(e),
  })

  const passwordMut = useMutation({
    mutationFn: (body: { currentPassword: string; newPassword: string }) =>
      API().console.updateMyPassword(body),
    onSuccess: () => {
      message('Password updated', 'success')
      setPassword(emptyPassword)
    },
    onError: (e) => message(e),
  })

  const handleAvatarSelect = async (file: File) => {
    if (!form) return
    if (!file.type.startsWith('image/')) {
      message('Avatar must be an image.')
      return
    }
    setUploading(true)
    try {
      const f = await API().console.uploadFile(file)
      setForm({ ...form, imageUrl: f.url })
    } catch (e) {
      message(e)
    } finally {
      setUploading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  const clearAvatar = () => {
    if (!form) return
    setForm({ ...form, imageUrl: '' })
  }

  if (isLoading) return <p className="text-zinc-500">Loading…</p>
  if (error) return <p className="text-rose-600">{(error as Error).message}</p>
  if (!data || !form) return null

  const passwordError = (() => {
    if (!password.current && !password.next && !password.repeat) return null
    if (password.next.length < 10) return 'New password must be at least 10 characters.'
    if (password.next !== password.repeat) return 'Passwords do not match.'
    return null
  })()

  return (
    <div className="space-y-6">
      <form
        className="space-y-4 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-6"
        onSubmit={(e) => {
          e.preventDefault()
          profileMut.mutate({ name: form.name.trim(), imageUrl: form.imageUrl.trim() })
        }}
      >
        <div>
          <h2 className="text-sm font-semibold uppercase tracking-wide text-zinc-500">
            Change Profile
          </h2>
          {data.email && (
            <p className="mt-1 text-xs text-zinc-500">{data.email}</p>
          )}
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">Full name</label>
          <input
            value={form.name}
            onChange={(e) => setForm({ ...form, name: e.target.value })}
            className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none"
          />
        </div>

        <div>
          <label className="block text-sm font-medium mb-1">Avatar</label>
          <div className="flex items-center gap-3">
            <div className="grid size-14 shrink-0 place-items-center overflow-hidden rounded-full border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950">
              {form.imageUrl ? (
                <img src={form.imageUrl} alt="" className="h-full w-full object-cover" />
              ) : (
                <span className="text-[10px] uppercase tracking-wide text-zinc-400">
                  None
                </span>
              )}
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <Button
                type="button"
                variant="ghost"
                isLoading={uploading}
                onClick={() => fileInputRef.current?.click()}
              >
                <Upload className="size-4" />
                {form.imageUrl ? 'Replace' : 'Upload'}
              </Button>
              {form.imageUrl && (
                <Button type="button" variant="ghost" onClick={clearAvatar}>
                  <X className="size-4" />
                  Remove
                </Button>
              )}
              <input
                ref={fileInputRef}
                type="file"
                accept="image/*"
                className="hidden"
                onChange={(e) => {
                  const f = e.target.files?.[0]
                  if (f) void handleAvatarSelect(f)
                }}
              />
            </div>
          </div>
          <p className="mt-1 text-xs text-zinc-500">
            PNG, JPG, GIF, or WebP.
          </p>
        </div>

        <div className="flex justify-end">
          <Button type="submit" isLoading={profileMut.isPending}>
            Update profile
          </Button>
        </div>
      </form>

      {data.hasPasswordAccount && (
        <form
          className="space-y-4 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-6"
          onSubmit={(e) => {
            e.preventDefault()
            if (passwordError) {
              message(passwordError)
              return
            }
            if (!password.current || !password.next) {
              message('Fill in all password fields.')
              return
            }
            passwordMut.mutate({
              currentPassword: password.current,
              newPassword: password.next,
            })
          }}
        >
          <h2 className="text-sm font-semibold uppercase tracking-wide text-zinc-500">
            Change Password
          </h2>

          <div>
            <label className="block text-sm font-medium mb-1">Current password</label>
            <input
              type="password"
              autoComplete="current-password"
              value={password.current}
              onChange={(e) => setPassword({ ...password, current: e.target.value })}
              className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none"
            />
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">New password</label>
            <input
              type="password"
              autoComplete="new-password"
              value={password.next}
              onChange={(e) => setPassword({ ...password, next: e.target.value })}
              className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none"
            />
            <p className="mt-1 text-xs text-zinc-500">Minimum 10 characters.</p>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">Repeat new password</label>
            <input
              type="password"
              autoComplete="new-password"
              value={password.repeat}
              onChange={(e) => setPassword({ ...password, repeat: e.target.value })}
              className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none"
            />
            {passwordError && (
              <p className="mt-1 text-xs text-rose-600">{passwordError}</p>
            )}
          </div>

          <div className="flex justify-end">
            <Button
              type="submit"
              isLoading={passwordMut.isPending}
              disabled={!!passwordError}
            >
              Update password
            </Button>
          </div>
        </form>
      )}
    </div>
  )
}

