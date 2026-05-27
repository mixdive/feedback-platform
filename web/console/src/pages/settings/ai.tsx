import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'

import Button from '@/components/button'
import ReadOnlyBanner from '@/components/read-only-banner'
import { API, type ApiAISettingsPatch } from '@/services/api'
import { useAppSelector } from '@/store/hooks'
import { message } from '@/utils/helpers'

dayjs.extend(relativeTime)

// Static list of selectable Claude models. Empty string means "use the
// backend's DefaultModel" — the backend resolves it to claude-haiku-4-5
// at call time. Admins picking an explicit model store its exact ID.
const MODEL_OPTIONS: { value: string; label: string; hint: string }[] = [
  { value: 'claude-haiku-4-5', label: 'Haiku 4.5', hint: 'Fastest and cheapest. Recommended for category classification.' },
  { value: 'claude-sonnet-4-6', label: 'Sonnet 4.6', hint: 'Higher accuracy on ambiguous text. Slower and more expensive.' },
  { value: 'claude-opus-4-7', label: 'Opus 4.7', hint: 'Most accurate. Reserve for difficult tasks; overkill for category classification.' },
]

export default function AISettingsPage() {
  const queryClient = useQueryClient()
  const isAdmin = !!useAppSelector((s) => s.auth.user?.roles?.includes('admin'))

  const [enabledDraft, setEnabledDraft] = useState<boolean | null>(null)
  const [modelDraft, setModelDraft] = useState<string | null>(null)
  const [apiKeyDraft, setApiKeyDraft] = useState('')

  const { data, isLoading, error } = useQuery({
    queryKey: ['console', 'settings'],
    queryFn: () => API().console.getSettings(),
  })

  // Live-poll the queue stats every 5 seconds. Cheap on the backend
  // (one Mongo count per analyzer) and gives the admin near-real-time
  // visibility into "what's happening right now."
  const { data: queue } = useQuery({
    queryKey: ['console', 'ai-queue'],
    queryFn: () => API().console.getAIQueue(),
    refetchInterval: 5_000,
    enabled: isAdmin,
  })

  useEffect(() => {
    if (!data) return
    if (enabledDraft === null) setEnabledDraft(data.ai.enabled)
    if (modelDraft === null) setModelDraft(data.ai.model || MODEL_OPTIONS[0].value)
  }, [data, enabledDraft, modelDraft])

  const invalidate = () => {
    void queryClient.invalidateQueries({ queryKey: ['console', 'settings'] })
    void queryClient.invalidateQueries({ queryKey: ['console', 'ai-queue'] })
  }

  const enableMut = useMutation({
    mutationFn: (enabled: boolean) => API().console.updateAISettings({ enabled }),
    onSuccess: () => {
      message('AI settings saved', 'success')
      invalidate()
    },
    onError: (e, attempted) => {
      message(e)
      // Backend rejected the toggle — revert the optimistic UI flip so
      // the visible state matches the saved state.
      setEnabledDraft(!attempted)
    },
  })

  const handleToggleEnabled = (next: boolean) => {
    if (!isAdmin) return
    setEnabledDraft(next)
    // First-time enable with no key on file: defer the backend write
    // until the admin saves the AI settings card below. The bottom
    // card activates immediately so they can supply model + key.
    if (next && !data?.ai.hasApiKey) return
    enableMut.mutate(next)
  }

  const settingsMut = useMutation({
    mutationFn: (body: ApiAISettingsPatch) => API().console.updateAISettings(body),
    onSuccess: () => {
      message('AI settings saved', 'success')
      setApiKeyDraft('')
      invalidate()
    },
    onError: (e) => message(e),
  })

  const clearKeyMut = useMutation({
    mutationFn: () => API().console.updateAISettings({ apiKey: '' }),
    onSuccess: () => {
      message('API key cleared', 'success')
      invalidate()
    },
    onError: (e) => message(e),
  })

  if (isLoading) return <p className="text-zinc-500">Loading…</p>
  if (error) return <p className="text-rose-600">{(error as Error).message}</p>
  if (enabledDraft === null || modelDraft === null || !data) return null

  // The bottom card's interactive state is gated by the local toggle
  // value, not the saved one, so the admin can flip the top toggle and
  // immediately fill in model + key on first-time setup.
  const settingsActive = enabledDraft && isAdmin

  const modelChanged = modelDraft !== (data.ai.model || MODEL_OPTIONS[0].value)
  const hasKeyDraft = apiKeyDraft.trim().length > 0
  const pendingEnable = enabledDraft && !data.ai.enabled
  const settingsDirty = modelChanged || hasKeyDraft || pendingEnable
  const settingsRequiresKey =
    enabledDraft && !data.ai.hasApiKey && !hasKeyDraft

  return (
    <div className="space-y-6">
      {!isAdmin && <ReadOnlyBanner />}

      {/* Enable toggle */}
      <div className="space-y-3 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-6">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-zinc-500">
          AI Analysis
        </h2>
        <fieldset disabled={!isAdmin} className="space-y-3 disabled:opacity-70">
          <label className="flex items-start gap-3">
            <input
              type="checkbox"
              checked={enabledDraft}
              onChange={(e) => handleToggleEnabled(e.target.checked)}
              disabled={enableMut.isPending}
              className="mt-0.5"
            />
            <span>
              <span className="block text-sm font-medium">AI analysis enabled</span>
              <span className="block text-xs text-zinc-500 mt-0.5">
                When on, Claude analyzes every new and edited entry. The category
                analyzer suggests one of your configured categories; results appear
                on each entry as advisory metadata (admins decide whether to apply).
              </span>
            </span>
          </label>

          {pendingEnable && (
            <p className="text-xs text-amber-600 dark:text-amber-400">
              Configure the model and API key below and save them to start analyzing.
            </p>
          )}
        </fieldset>
      </div>

      {/* AI settings (model + API key) */}
      <form
        className="space-y-4 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-6"
        onSubmit={(e) => {
          e.preventDefault()
          if (!settingsActive) return
          if (settingsRequiresKey) {
            message('Enter an API key first.')
            return
          }
          const body: ApiAISettingsPatch = { model: modelDraft }
          if (hasKeyDraft) body.apiKey = apiKeyDraft.trim()
          // Commit the deferred enable=true alongside the first-time
          // model + key save, so the admin doesn't have to re-toggle.
          if (pendingEnable) body.enabled = true
          settingsMut.mutate(body)
        }}
      >
        <h2 className="text-sm font-semibold uppercase tracking-wide text-zinc-500">
          AI Settings
        </h2>
        <fieldset disabled={!settingsActive} className="space-y-4 disabled:opacity-60">
          <div>
            <label className="block text-sm font-medium mb-1">Model</label>
            <select
              value={modelDraft}
              onChange={(e) => setModelDraft(e.target.value)}
              className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 text-sm focus:border-sky-500 focus:outline-none disabled:cursor-not-allowed"
            >
              {MODEL_OPTIONS.map((m) => (
                <option key={m.value} value={m.value}>
                  {m.label}
                </option>
              ))}
            </select>
            <p className="text-xs text-zinc-500 mt-1">
              {MODEL_OPTIONS.find((m) => m.value === modelDraft)?.hint}
            </p>
          </div>

          <div>
            <label className="block text-sm font-medium mb-1">Anthropic API key</label>
            {data.ai.hasApiKey ? (
              <p className="text-xs text-zinc-500 mb-2">
                Current key:{' '}
                <span className="font-mono text-zinc-700 dark:text-zinc-300">
                  {data.ai.apiKeyPreview}
                </span>{' '}
                · Enter a new value below to rotate.
              </p>
            ) : (
              <p className="text-xs text-zinc-500 mb-2">
                No key set. AI analysis cannot run without one.
              </p>
            )}
            <input
              type="password"
              autoComplete="off"
              value={apiKeyDraft}
              onChange={(e) => setApiKeyDraft(e.target.value)}
              placeholder="sk-ant-..."
              className="block w-full h-10 rounded-md border border-zinc-300 dark:border-zinc-700 bg-white dark:bg-zinc-950 px-3 font-mono text-xs focus:border-sky-500 focus:outline-none disabled:cursor-not-allowed"
            />
            <p className="text-xs text-zinc-500 mt-1">
              Stored on the singleton settings document in MongoDB. Never returned
              to the browser in cleartext — only the masked preview above.
            </p>
          </div>
        </fieldset>
        {settingsActive && (
          <div className="flex justify-end gap-2">
            {data.ai.hasApiKey && (
              <Button
                type="button"
                variant="ghost"
                onClick={() => clearKeyMut.mutate()}
                isLoading={clearKeyMut.isPending}
              >
                Clear key
              </Button>
            )}
            <Button
              type="submit"
              isLoading={settingsMut.isPending}
              disabled={!settingsDirty || settingsRequiresKey}
            >
              Save
            </Button>
          </div>
        )}
      </form>

      {/* Queue + last status */}
      <div className="space-y-4 rounded-lg border border-zinc-200 dark:border-zinc-800 bg-white dark:bg-zinc-900 p-6">
        <h2 className="text-sm font-semibold uppercase tracking-wide text-zinc-500">
          Queue
        </h2>
        {!isAdmin && (
          <p className="text-xs text-zinc-500">
            Queue stats are visible to admins only.
          </p>
        )}
        {isAdmin && !queue && <p className="text-xs text-zinc-500">Loading…</p>}
        {isAdmin && queue && (
          <div className="space-y-2">
            <div className="flex flex-wrap items-center gap-x-6 gap-y-2 text-sm">
              <Stat label="Status" value={queue.enabled ? 'Active' : 'Disabled'} />
              <Stat label="Waiting" value={String(queue.totalPending)} />
              <Stat label="In flight" value={String(queue.totalInFlight)} />
              <Stat label="Analyzed" value={queue.totalAnalyzed.toLocaleString()} />
              <Stat label="Polling every" value={queue.pollInterval} />
              <Stat label="Claim TTL" value={queue.claimTtl} />
            </div>
            <LastOutcome
              lastAnalyzedAt={data.ai.lastAnalyzedAt}
              lastErrorAt={data.ai.lastErrorAt}
              lastErrorMessage={data.ai.lastErrorMessage}
            />
          </div>
        )}
      </div>
    </div>
  )
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-baseline gap-1.5">
      <span className="text-xs uppercase tracking-wide text-zinc-500">{label}</span>
      <span className="font-medium">{value}</span>
    </div>
  )
}

// LastOutcome shows whichever of (last success, last error) is the most
// recent. Showing both is misleading — once a retry succeeds, the
// historical error from before the fix becomes noise. Backend still
// keeps both timestamps in the settings doc for debugging.
function LastOutcome({
  lastAnalyzedAt,
  lastErrorAt,
  lastErrorMessage,
}: {
  lastAnalyzedAt?: string
  lastErrorAt?: string
  lastErrorMessage?: string
}) {
  const hasError = !!(lastErrorMessage && lastErrorAt)
  const hasSuccess = !!lastAnalyzedAt
  const errorIsCurrent =
    hasError && (!hasSuccess || dayjs(lastErrorAt).isAfter(dayjs(lastAnalyzedAt)))

  if (errorIsCurrent) {
    return (
      <p className="text-xs text-rose-600 dark:text-rose-400">
        Last error{' '}
        <span title={lastErrorAt}>{dayjs(lastErrorAt).fromNow()}</span>
        : {lastErrorMessage}
      </p>
    )
  }
  if (hasSuccess) {
    return (
      <p className="text-xs text-zinc-500">
        Last analyzed{' '}
        <span title={lastAnalyzedAt}>{dayjs(lastAnalyzedAt).fromNow()}</span>.
      </p>
    )
  }
  return null
}
