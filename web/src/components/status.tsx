import { cn } from '../lib/utils'
import { useI18n, type TFunction } from '../lib/i18n'
import type { RunStatus } from '../lib/api'

const STATUS: Record<RunStatus, { labelKey: string; className: string; dot: string }> = {
  queued: { labelKey: 'status.queued', className: 'text-muted', dot: 'bg-muted' },
  leased: { labelKey: 'status.leased', className: 'text-warn', dot: 'bg-warn' },
  preparing: { labelKey: 'status.preparing', className: 'text-warn', dot: 'bg-warn' },
  running: { labelKey: 'status.running', className: 'text-accent2', dot: 'bg-accent2 animate-pulse' },
  waiting_approval: { labelKey: 'status.waiting_approval', className: 'text-warn', dot: 'bg-warn' },
  finalizing: { labelKey: 'status.finalizing', className: 'text-accent2', dot: 'bg-accent2' },
  succeeded: { labelKey: 'status.succeeded', className: 'text-ok', dot: 'bg-ok' },
  failed: { labelKey: 'status.failed', className: 'text-err', dot: 'bg-err' },
  cancelled: { labelKey: 'status.cancelled', className: 'text-muted', dot: 'bg-muted' },
}

// statusLabel translates a dynamic status value, falling back to the raw value
// when the status is not part of the dictionary.
export function statusLabel(t: TFunction, status: string): string {
  const key = `status.${status}`
  const label = t(key)
  return label === key ? status : label
}

export function StatusBadge({ status }: { status: RunStatus }) {
  const { t } = useI18n()
  const s = STATUS[status] ?? STATUS.queued
  return (
    <span className={cn('inline-flex items-center gap-1.5 text-[12px] font-medium', s.className)}>
      <span className={cn('h-1.5 w-1.5 rounded-full', s.dot)} />
      {t(s.labelKey)}
    </span>
  )
}

export function StatusDot({ status }: { status: RunStatus }) {
  const s = STATUS[status] ?? STATUS.queued
  return <span className={cn('h-1.5 w-1.5 rounded-full', s.dot)} />
}
