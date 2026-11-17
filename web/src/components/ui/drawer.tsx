import type { ReactNode } from 'react'

export function Drawer({
  open,
  onOpenChange,
  title,
  description,
  children,
}: {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description?: string
  children: ReactNode
}) {
  if (!open) return null
  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black/40" onClick={() => onOpenChange(false)}>
      <aside
        className="flex h-full w-full max-w-[640px] flex-col border-l border-line bg-bg shadow-2xl"
        onClick={(event) => event.stopPropagation()}
      >
        <header className="flex items-start justify-between gap-3 border-b border-line px-4 py-3">
          <div>
            <h2 className="text-[14px] font-semibold tracking-tight">{title}</h2>
            {description && <p className="mt-0.5 text-[11.5px] text-muted">{description}</p>}
          </div>
          <button
            className="rounded border border-line2 px-2 py-1 text-[11px] text-muted hover:text-fg"
            onClick={() => onOpenChange(false)}
          >
            Close
          </button>
        </header>
        <div className="min-h-0 flex-1 overflow-y-auto px-4 py-3">{children}</div>
      </aside>
    </div>
  )
}
