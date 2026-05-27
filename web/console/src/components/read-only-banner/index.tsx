import { Lock } from 'lucide-react'

export default function ReadOnlyBanner() {
  return (
    <div className="flex items-center gap-2 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-800 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-200">
      <Lock className="size-3.5 shrink-0" />
      <span>Read-only — admin role required to make changes.</span>
    </div>
  )
}
