import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { api } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { Button } from './ui/button'

const LOCAL_STATUSES = ['open', 'in_progress', 'done', 'cancelled'] as const
type LocalStatus = (typeof LOCAL_STATUSES)[number]

/** Mirrors the plane's default group mapping, to label "default" choices. */
const GROUP_DEFAULT: Record<string, LocalStatus> = {
  backlog: 'open',
  unstarted: 'open',
  started: 'in_progress',
  completed: 'done',
  cancelled: 'cancelled',
  canceled: 'cancelled',
}
const STATUS_GROUPS: Record<LocalStatus, string[]> = {
  open: ['unstarted', 'backlog'],
  in_progress: ['started'],
  done: ['completed'],
  cancelled: ['cancelled', 'canceled'],
}
const GROUP_ORDER = ['backlog', 'unstarted', 'started', 'completed', 'cancelled', 'canceled', '']

const IN = 'state.in.'
const OUT = 'state.out.'
const WRITEBACK = 'writebackStatus'

/**
 * Edits how one binding's platform states map to local statuses and back.
 * Mappings key on state ids, which survive renames; a binding covering a
 * whole Plane workspace keys on names because ids differ per project.
 */
export function BindingStateMapping({
  binding,
}: {
  binding: { id: string; externalId: string; settings?: Record<string, string> }
}) {
  const { t } = useI18n()
  const qc = useQueryClient()
  const byName = binding.externalId.split('/').length < 2
  const { data, isLoading, error } = useQuery({
    queryKey: ['binding-states', binding.id],
    queryFn: () => api.bindingStates(binding.id),
  })
  const [settings, setSettings] = useState<Record<string, string>>(binding.settings ?? {})
  useEffect(() => setSettings(binding.settings ?? {}), [binding.settings])
  const save = useMutation({
    mutationFn: () => api.patchBinding(binding.id, { settings }),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['connections'] }),
  })

  const states = [...(data?.items ?? [])].sort(
    (a, b) => GROUP_ORDER.indexOf(a.group ?? '') - GROUP_ORDER.indexOf(b.group ?? ''),
  )
  const keyOf = (state: { id: string; name: string }) => (byName ? state.name : state.id)
  const set = (key: string, value: string) =>
    setSettings((prev) => {
      const next = { ...prev }
      if (value) next[key] = value
      else delete next[key]
      return next
    })
  // Same order as the plane: the first group listed wins, then API order.
  const defaultTarget = (status: LocalStatus) => {
    for (const group of STATUS_GROUPS[status]) {
      const hit = (data?.items ?? []).find((state) => (state.group ?? '').toLowerCase() === group)
      if (hit) return hit
    }
    return undefined
  }
  const groupLabel = (group?: string) => (group ? t(`stateGroup.${group.toLowerCase()}`) : '—')

  if (isLoading) return <p className="py-2 text-[11px] text-muted"><Loader2 className="inline h-3 w-3 animate-spin" /> {t('common.loading')}</p>
  if (error) return <p className="py-2 text-[11px] text-err">{error.message}</p>

  return (
    <div className="mt-2 flex flex-col gap-4 rounded-md border border-line2 bg-panel2/40 p-3 text-[11.5px]">
      <div>
        <p className="font-medium text-fg">{t('stateMap.inbound')}</p>
        <p className="mt-0.5 text-[10.5px] text-muted">{t(byName ? 'stateMap.inboundHintByName' : 'stateMap.inboundHint')}</p>
        <div className="mt-2 grid grid-cols-[1fr_auto_auto] items-center gap-x-3 gap-y-1">
          {states.map((state) => {
            const fallback = GROUP_DEFAULT[(state.group ?? '').toLowerCase()]
            return (
              <div key={state.id} className="contents">
                <span className="truncate text-fg">{state.name}</span>
                <span className="text-[10px] text-muted">{groupLabel(state.group)}</span>
                <select
                  className="h-7 rounded-md border border-line2 bg-panel px-1.5 text-[11px]"
                  value={settings[IN + keyOf(state)] ?? ''}
                  onChange={(e) => set(IN + keyOf(state), e.target.value)}
                >
                  <option value="">{fallback ? t('stateMap.default', { status: t(`status.${fallback}`) }) : t('stateMap.unchanged')}</option>
                  {LOCAL_STATUSES.map((status) => (
                    <option key={status} value={status}>{t(`status.${status}`)}</option>
                  ))}
                </select>
              </div>
            )
          })}
        </div>
      </div>

      <div>
        <label className="flex items-center gap-2 font-medium text-fg">
          <input
            type="checkbox"
            checked={settings[WRITEBACK] === 'true'}
            onChange={(e) => set(WRITEBACK, e.target.checked ? 'true' : '')}
          />
          {t('stateMap.outbound')}
        </label>
        <p className="mt-0.5 text-[10.5px] text-muted">{t('stateMap.outboundHint')}</p>
        {settings[WRITEBACK] === 'true' && (
          <div className="mt-2 grid grid-cols-[auto_1fr] items-center gap-x-3 gap-y-1">
            {LOCAL_STATUSES.map((status) => {
              const auto = defaultTarget(status)
              return (
                <div key={status} className="contents">
                  <span className="text-fg">{t(`status.${status}`)}</span>
                  <select
                    className="h-7 rounded-md border border-line2 bg-panel px-1.5 text-[11px]"
                    value={settings[OUT + status] ?? ''}
                    onChange={(e) => set(OUT + status, e.target.value)}
                  >
                    <option value="">{auto ? t('stateMap.auto', { state: auto.name }) : t('stateMap.noWriteback')}</option>
                    {states.map((state) => (
                      <option key={state.id} value={keyOf(state)}>{state.name}</option>
                    ))}
                  </select>
                </div>
              )
            })}
          </div>
        )}
      </div>

      <div className="flex items-center justify-end gap-2">
        {save.isSuccess && <span className="text-[10.5px] text-ok">{t('common.saved')}</span>}
        {save.error && <span className="text-[10.5px] text-err">{save.error.message}</span>}
        <Button size="sm" variant="primary" disabled={save.isPending} onClick={() => save.mutate()}>
          {t('common.save')}
        </Button>
      </div>
    </div>
  )
}
