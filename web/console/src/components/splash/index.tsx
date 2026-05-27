export default function Splash() {
  return (
    <div className="flex h-full w-full flex-col items-center justify-center gap-4 bg-zinc-50 dark:bg-zinc-950">
      <img
        src="/console/mixdive-icon.svg"
        alt="Mixdive"
        className="size-12 animate-pulse"
      />
      <div className="text-sm text-zinc-500 dark:text-zinc-400">Loading…</div>
    </div>
  )
}
