import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CalendarRange, Plus, Trash2 } from 'lucide-react'
import { api } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { useWorkspace } from '../lib/workspace'
import { cn } from '../lib/utils'
import { statusLabel } from '../components/status'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Dialog, DialogContent, DialogTrigger } from '../components/ui/dialog'

const STATUS = ['planned', 'active', 'done']

export default function IterationsPage() {
  const { t } = useI18n()
  const { workspaceId } = useWorkspace()
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [startDate, setStartDate] = useState('')
  const [endDate, setEndDate] = useState('')
  const [selected, setSelected] = useState('')

  const { data, isLoading } = useQuery({
    queryKey: ['iterations', workspaceId],
    queryFn: () => api.iterations(workspaceId),
  })
  const items = data?.items ?? []
  const active = selected || items[0]?.id || ''
  const { data: burndown } = useQuery({
    queryKey: ['burndown', active],
    queryFn: () => api.iterationBurndown(active),
    enabled: !!active,
  })

  const create = useMutation({
    mutationFn: () =>
      api.createIteration(workspaceId, { name, startDate, endDate, status: 'planned' }),
    onSuccess: () => {
      setOpen(false)
      setName('')
      qc.invalidateQueries({ queryKey: ['iterations', workspaceId] })
    },
  })
  const setStatus = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) => api.patchIteration(id, status),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['iterations', workspaceId] }),
  })
  const remove = useMutation({
    mutationFn: (id: string) => api.deleteIteration(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['iterations', workspaceId] }),
  })

  return (
    <div className="mx-auto max-w-[1000px] px-6 py-5">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-[18px] font-semibold tracking-tight">{t('page.iterations')}</h1>
          <p className="mt-0.5 text-[12px] text-muted">{t('iterations.subtitle')}</p>
        </div>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button variant="primary" size="sm">
              <Plus className="h-3.5 w-3.5" /> {t('iterations.new')}
            </Button>
          </DialogTrigger>
          <DialogContent title={t('iterations.new')} description={t('iterations.newHint')}>
            <form
              className="flex flex-col gap-3"
              onSubmit={(e) => {
                e.preventDefault()
                create.mutate()
              }}
            >
              <label className="text-[12px] text-muted">
                {t('common.name')}
                <Input className="mt-1" value={name} onChange={(e) => setName(e.target.value)} />
              </label>
              <div className="grid grid-cols-2 gap-2">
                <label className="text-[12px] text-muted">
                  {t('iterations.start')}
                  <Input
                    className="mt-1"
                    value={startDate}
                    onChange={(e) => setStartDate(e.target.value)}
                    placeholder="2026-11-01"
                  />
                </label>
                <label className="text-[12px] text-muted">
                  {t('iterations.end')}
                  <Input
                    className="mt-1"
                    value={endDate}
                    onChange={(e) => setEndDate(e.target.value)}
                    placeholder="2026-11-14"
                  />
                </label>
              </div>
              {create.error && <p className="text-[12px] text-err">{String(create.error.message)}</p>}
              <div className="flex justify-end gap-2">
                <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
                  {t('common.cancel')}
                </Button>
                <Button type="submit" variant="primary" disabled={!name || create.isPending}>
                  {t('common.create')}
                </Button>
              </div>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      {isLoading && <p className="mt-6 text-[12px] text-muted">{t('common.loading')}</p>}

      <div className="mt-4 overflow-hidden rounded-lg border border-line">
        {items.map((it) => (
          <div
            key={it.id}
            onClick={() => setSelected(it.id)}
            className={cn(
              'flex cursor-pointer items-center justify-between border-b border-line px-3 py-2.5 last:border-0',
              active === it.id ? 'bg-panel2/60' : 'hover:bg-panel/40',
            )}
          >
            <div className="flex min-w-0 items-center gap-2">
              <CalendarRange className="h-3.5 w-3.5 text-muted" />
              <span className="text-[12.5px]">{it.name}</span>
              <span className="text-[10.5px] text-muted">
                {it.startDate} → {it.endDate}
              </span>
              <span className="text-[10.5px] text-muted">
                {t('iterations.doneOf', { done: it.doneCount, total: it.taskCount })}
              </span>
            </div>
            <div className="flex items-center gap-1">
              {STATUS.map((s) => (
                <button
                  key={s}
                  onClick={() => setStatus.mutate({ id: it.id, status: s })}
                  className={`rounded px-1.5 py-0.5 text-[10.5px] ${
                    it.status === s ? 'bg-accent/15 text-accent2' : 'text-muted hover:text-fg'
                  }`}
                >
                  {statusLabel(t, s)}
                </button>
              ))}
              <button
                onClick={() => remove.mutate(it.id)}
                className="ml-1 text-muted hover:text-err"
                title={t('common.delete')}
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </div>
          </div>
        ))}
        {!isLoading && items.length === 0 && (
          <p className="px-3 py-12 text-center text-[12px] text-muted">{t('iterations.empty')}</p>
        )}
      </div>

      {burndown && (
        <div className="mt-4 rounded-lg border border-line bg-panel/30 p-4">
          <div className="flex items-center justify-between">
            <div>
              <h2 className="text-[13px] font-medium">{burndown.iteration.name}</h2>
              <p className="text-[11px] text-muted">
                {burndown.iteration.startDate} → {burndown.iteration.endDate}
              </p>
            </div>
            <div className="flex gap-3 text-[11px] text-muted">
              <span>{t('iterations.total', { count: burndown.stats.total })}</span>
              <span className="text-ok">{t('iterations.done', { count: burndown.stats.done })}</span>
              <span>{t('iterations.open', { count: burndown.stats.open })}</span>
              <span>{t('iterations.inProgress', { count: burndown.stats.inProgress })}</span>
            </div>
          </div>
          <BurndownChart points={burndown.points} total={burndown.stats.total} />
        </div>
      )}
    </div>
  )
}

function BurndownChart({ points, total }: { points: { date: string; remaining: number }[]; total: number }) {
  const { t } = useI18n()
  if (points.length < 2 || total === 0) {
    return <p className="mt-4 text-[11.5px] text-muted">{t('iterations.noBurndown')}</p>
  }
  const width = 400
  const height = 90
  const max = Math.max(total, ...points.map((p) => p.remaining))
  const coords = points.map((p, i) => {
    const x = 10 + (i / (points.length - 1)) * (width - 20)
    const y = 10 + (1 - p.remaining / max) * (height - 20)
    return `${x},${y}`
  })
  return (
    <div className="mt-3">
      <svg viewBox={`0 0 ${width} ${height}`} className="h-[90px] w-full">
        <line x1="10" y1={height - 10} x2={width - 10} y2={height - 10} stroke="var(--color-line2)" />
        <polyline
          points={coords.join(' ')}
          fill="none"
          stroke="var(--color-accent2)"
          strokeWidth="1.5"
        />
      </svg>
      <div className="flex justify-between text-[10px] text-muted">
        <span>{points[0].date}</span>
        <span>{points[points.length - 1].date}</span>
      </div>
    </div>
  )
}
