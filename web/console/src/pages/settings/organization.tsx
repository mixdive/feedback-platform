import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Upload, X } from 'lucide-react'

import Button from '@/components/button'
import ReadOnlyBanner from '@/components/read-only-banner'
import { API, type ApiSettings } from '@/services/api'
import { useAppSelector } from '@/store/hooks'
import { message } from '@/utils/helpers'

type Form = Pick<ApiSettings, 'projectName' | 'logoUrl' | 'primaryColor'>

const DEFAULT_COLOR = '#0EA5E9'

export default function ProjectSettingsPage() {
  const queryClient = useQueryClient()
  const [form, setForm] = useState<Form | null>(null)
  const [uploading, setUploading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  const isAdmin = !!useAppSelector((s) => s.auth.user?.roles?.includes('admin'))

  const { data, isLoading, error } = useQuery({
    queryKey: ['console', 'settings'],
    queryFn: () => API().console.getSettings(),
  })

  useEffect(() => {
    if (!data || form) return
    setForm({
      projectName: data.projectName,
      logoUrl: data.logoUrl ?? '',
      primaryColor: data.primaryColor ?? '',
    })
  }, [data, form])

  const updateMut = useMutation({
    mutationFn: (body: Partial<ApiSettings>) => API().console.updateSettings(body),
    onSuccess: () => {
      message('Project settings saved', 'success')
      void queryClient.invalidateQueries({ queryKey: ['console', 'settings'] })
    },
    onError: (e) => message(e),
  })

  const handleLogoSelect = async (file: File) => {
    if (!form) return
    if (!file.type.startsWith('image/')) {
      message('Logo must be an image.')
      return
    }
    setUploading(true)
    try {
      const f = await API().console.uploadFile(file)
      setForm({ ...form, logoUrl: f.url })
    } catch (e) {
      message(e)
    } finally {
      setUploading(false)
      if (fileInputRef.current) fileInputRef.current.value = ''
    }
  }

  const clearLogo = () => {
    if (!form) return
    setForm({ ...form, logoUrl: '' })
  }

  if (isLoading) return <p className="text-zinc-500">Loading…</p>
  if (error) return <p className="text-rose-600">{(error as Error).message}</p>
  if (!form) return null

  const swatch = form.primaryColor?.trim() || DEFAULT_COLOR

  return (
    <form
      className="space-y-4 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-6"
      onSubmit={(e) => {
        e.preventDefault()
        if (!isAdmin) return
        updateMut.mutate({
          projectName: form.projectName,
          logoUrl: form.logoUrl ?? '',
          primaryColor: form.primaryColor ?? '',
        })
      }}
    >
      {!isAdmin && <ReadOnlyBanner />}
      <fieldset disabled={!isAdmin} className="space-y-4 disabled:opacity-70">
      <div>
        <label className="block text-sm font-medium mb-1">Project name</label>
        <input
          value={form.projectName}
          onChange={(e) => setForm({ ...form, projectName: e.target.value })}
          className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none disabled:cursor-not-allowed"
        />
      </div>

      <div>
        <label className="block text-sm font-medium mb-1">Logo</label>
        <div className="flex items-center gap-3">
          <div className="grid size-14 shrink-0 place-items-center overflow-hidden rounded-md border border-zinc-200 dark:border-zinc-800 bg-zinc-50 dark:bg-zinc-950">
            {form.logoUrl ? (
              <img src={form.logoUrl} alt="" className="h-full w-full object-contain" />
            ) : (
              <span className="text-[10px] uppercase tracking-wide text-zinc-400">
                No logo
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
              {form.logoUrl ? 'Replace' : 'Upload'}
            </Button>
            {form.logoUrl && (
              <Button type="button" variant="ghost" onClick={clearLogo}>
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
                if (f) void handleLogoSelect(f)
              }}
            />
          </div>
        </div>
        <p className="mt-1 text-xs text-zinc-500">
          PNG, JPG, GIF, or WebP. Shown next to the project name on the console and the portal.
        </p>
      </div>

      <div>
        <label className="block text-sm font-medium mb-1">Primary color</label>
        <div className="flex items-center gap-2">
          <input
            type="color"
            value={swatch}
            onChange={(e) => setForm({ ...form, primaryColor: e.target.value })}
            className="h-10 w-12 cursor-pointer rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 p-1"
            aria-label="Pick primary color"
          />
          <input
            value={form.primaryColor ?? ''}
            onChange={(e) => setForm({ ...form, primaryColor: e.target.value })}
            placeholder={DEFAULT_COLOR}
            className="block w-32 h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 font-mono text-sm focus:border-sky-500 focus:outline-none"
          />
        </div>
      </div>

      </fieldset>
      {isAdmin && (
        <div className="flex justify-end">
          <Button type="submit" isLoading={updateMut.isPending}>
            Save
          </Button>
        </div>
      )}
    </form>
  )
}
