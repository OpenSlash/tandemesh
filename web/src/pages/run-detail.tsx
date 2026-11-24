import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useNavigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  AlertTriangle,
  ArrowLeft,
  CheckCircle2,
  Circle,
  GitCompare,
  MessageSquare,
  ShieldCheck,
  SquareTerminal,
} from 'lucide-react'
import { api, type RunEvent } from '../lib/api'
import { useI18n, type TFunction } from '../lib/i18n'
import { cn, compactId, refId } from '../lib/utils'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Dialog, DialogContent, DialogTrigger } from '../components/ui/dialog'
import { StatusBadge, statusLabel } from '../components/status'

export default function RunDetailPage() {
  const { t } = useI18n()
  const { id = '' } = useParams()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const [events, setEvents] = useState<RunEvent[]>([])
  const [agents, setAgents] = useState('claude,codex')
  const [compareOpen, setCompareOpen] = useState(false)

  const { data: run, error } = useQuery({
    queryKey: ['run', id],
    queryFn: () => api.run(id),
    refetchInterval: 2000,
  })
  const { data: variants } = useQuery({
    queryKey: ['variants', id],
    queryFn: () => api.runVariants(id),
    refetchInterval: 3000,
  })

  const cancel = useMutation({
    mutationFn: () => api.cancelRun(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['run', id] }),
  })
  const retry = useMutation({
    mutationFn: () => api.retryRun(id),
    onSuccess: (run) => {
      qc.invalidateQueries({ queryKey: ['run', id] })
      navigate(`/runs/${run.id}`)
    },
  })
  const createVariants = useMutation({
    mutationFn: () =>
      api.createVariants(
        id,
        agents
          .split(',')
          .map((a) => a.trim())
          .filter(Boolean),
      ),
    onSuccess: () => {
      setCompareOpen(false)
      qc.invalidateQueries({ queryKey: ['variants', id] })
    },
  })

  useEffect(() => {
    setEvents([])
    const es = new EventSource(`/api/v1/runs/${id}/events?after=0`)
    es.onmessage = (e) => {
      try {
        const ev = JSON.parse(e.data) as RunEvent
        setEvents((prev) => (prev.some((x) => x.seq === ev.seq) ? prev : [...prev, ev].slice(-500)))
      } catch {
        /* ignore malformed frames */
      }
    }
    es.addEventListener('end', () => es.close())
    return () => es.close()
  }, [id])

  const terminal = run ? ['succeeded', 'failed', 'cancelled'].includes(run.status) : false

  return (
    <div className="mx-auto max-w-[1180px] px-6 py-5">
      <Link to="/runs" className="inline-flex items-center gap-1 text-[12px] text-muted hover:text-fg">
        <ArrowLeft className="h-3.5 w-3.5" /> {t('page.runs')}
      </Link>

      <div className="mt-3 flex items-start justify-between">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="font-mono text-[15px] text-fg">{run ? compactId(run.id, 10) : '…'}</h1>
            {run && <StatusBadge status={run.status} />}
          </div>
          <p className="mt-1 text-[12px] text-muted">
            {run?.repo || t('runs.noRepository')} · {run?.branch || t('runs.noBranch')}
          </p>
          {run?.taskId && (
            <Link
              to={`/tasks/${run.taskId}`}
              className="mt-1 inline-block text-[11.5px] text-accent2 hover:underline"
            >
              {t('runs.taskLink')}
            </Link>
          )}
        </div>
        <div className="flex gap-2">
          <Dialog open={compareOpen} onOpenChange={setCompareOpen}>
            <DialogTrigger asChild>
              <Button variant="ghost" size="sm">
                <GitCompare className="h-3.5 w-3.5" /> {t('runs.compareAgents')}
              </Button>
            </DialogTrigger>
            <DialogContent title={t('runs.variantsTitle')} description={t('runs.variantsHint')}>
              <form
                className="flex flex-col gap-3"
                onSubmit={(e) => {
                  e.preventDefault()
                  createVariants.mutate()
                }}
              >
                <label className="text-[12px] text-muted">
                  {t('runs.agentsCommaSeparated')}
                  <Input className="mt-1" value={agents} onChange={(e) => setAgents(e.target.value)} />
                </label>
                {createVariants.error && (
                  <p className="text-[12px] text-err">{String(createVariants.error.message)}</p>
                )}
                <div className="flex justify-end gap-2">
                  <Button type="button" variant="ghost" onClick={() => setCompareOpen(false)}>
                    {t('common.cancel')}
                  </Button>
                  <Button type="submit" variant="primary" disabled={createVariants.isPending}>
                    {t('runs.startVariants')}
                  </Button>
                </div>
              </form>
            </DialogContent>
          </Dialog>
          {!terminal && (
            <Button variant="danger" size="sm" onClick={() => cancel.mutate()} disabled={cancel.isPending}>
              {t('runs.cancelRun')}
            </Button>
          )}
          {run && (run.status === 'failed' || run.status === 'cancelled') && (
            <Button size="sm" onClick={() => retry.mutate()} disabled={retry.isPending}>
              {t('runs.retryRun')}
            </Button>
          )}
        </div>
      </div>

      {error && <p className="mt-4 text-[12px] text-err">{String(error.message)}</p>}

      <div className="mt-5 grid grid-cols-[minmax(0,1fr)_280px] gap-5">
        <section className="min-w-0 overflow-hidden rounded-lg border border-line">
          <header className="flex items-center justify-between border-b border-line bg-panel/60 px-3 py-2">
            <span className="text-[12px] font-medium text-fg">{t('runs.timeline')}</span>
            <span className="text-[11px] text-muted">
              {t('common.events', { count: events.length })}
            </span>
          </header>
          <Timeline events={events} />
        </section>

        <aside className="space-y-4">
          <div className="rounded-lg border border-line bg-panel/40 p-3">
            <h2 className="text-[11px] font-medium uppercase tracking-wider text-muted">
              {t('runs.metaRun')}
            </h2>
            <dl className="mt-2 space-y-2 text-[12px]">
              <Meta label={t('common.status')} value={run ? statusLabel(t, run.status) : '—'} />
              <Meta label={t('common.agent')} value={run?.agentId ?? '—'} />
              <Meta
                label={t('runs.colMachine')}
                value={run?.machineId ? refId(run.machineId, 10) : '—'}
                mono
              />
              <Meta label={t('runs.attempt')} value={String(run?.attempt ?? 0)} />
              <Meta label={t('runs.fence')} value={String(run?.fence ?? 0)} mono />
              <Meta label={t('runs.priority')} value={String(run?.priority ?? 0)} />
            </dl>
          </div>

          {(run?.inputTokens ?? 0) + (run?.outputTokens ?? 0) > 0 && (
            <div className="rounded-lg border border-line bg-panel/40 p-3">
              <h2 className="text-[11px] font-medium uppercase tracking-wider text-muted">
                {t('runs.usage')}
              </h2>
              <dl className="mt-2 space-y-2 text-[12px]">
                <Meta label={t('runs.model')} value={run?.model ?? '—'} />
                <Meta label={t('runs.tokensIn')} value={String(run?.inputTokens ?? 0)} />
                <Meta
                  label={t('runs.tokensOut')}
                  value={`${run?.usageEstimated ? '~' : ''}${run?.outputTokens ?? 0}`}
                />
                <Meta
                  label={t('runs.cache')}
                  value={`${run?.cacheReadTokens ?? 0} / ${run?.cacheWriteTokens ?? 0}${
                    run?.usageEstimated ? ` (${t('runs.estimated')})` : ''
                  }`}
                />
              </dl>
            </div>
          )}

          <div className="rounded-lg border border-line bg-panel/40 p-3">
            <h2 className="text-[11px] font-medium uppercase tracking-wider text-muted">
              {t('common.source')}
            </h2>
            <dl className="mt-2 space-y-2 text-[12px]">
              <Meta label={t('meta.provider')} value={run?.work.provider ?? '—'} />
              <Meta label={t('meta.id')} value={run?.work.id ?? '—'} mono />
              {run?.work.url && (
                <div className="flex justify-between gap-3">
                  <dt className="text-muted">{t('meta.link')}</dt>
                  <dd className="truncate text-right">
                    <a
                      href={run.work.url}
                      target="_blank"
                      rel="noreferrer"
                      className="text-accent2 hover:underline"
                    >
                      {t('meta.openLink')}
                    </a>
                  </dd>
                </div>
              )}
            </dl>
          </div>

          {run?.errorCode && (
            <div className="rounded-lg border border-err/30 bg-err/10 p-3">
              <h2 className="text-[11px] font-medium uppercase tracking-wider text-err">
                {t('common.error')}
              </h2>
              <p className="mt-1 font-mono text-[12px] text-err">{run.errorCode}</p>
              {run.errorMessage && <p className="mt-1 text-[11px] text-err/80">{run.errorMessage}</p>}
            </div>
          )}

          <div className="rounded-lg border border-line bg-panel/40 p-3">
            <h2 className="text-[11px] font-medium uppercase tracking-wider text-muted">
              {t('runs.variants')}
            </h2>
            {run?.variantOf && (
              <Link
                to={`/runs/${run.variantOf}`}
                className="mt-1 block text-[11px] text-accent2 hover:underline"
              >
                {t('runs.originRun')}
              </Link>
            )}
            <div className="mt-2 space-y-1">
              {(variants?.items ?? [])
                .filter((v) => v.id !== run?.id)
                .map((v) => (
                  <Link
                    key={v.id}
                    to={`/runs/${v.id}`}
                    className="flex items-center justify-between gap-2 rounded px-1 py-0.5 text-[12px] hover:bg-panel2"
                  >
                    <span className="truncate text-fg/90">{v.variantLabel || v.agentId}</span>
                    <StatusBadge status={v.status} />
                  </Link>
                ))}
              {(!variants || variants.items.length <= 1) && (
                <p className="text-[11.5px] text-muted">{t('runs.noVariants')}</p>
              )}
            </div>
          </div>
        </aside>
      </div>
    </div>
  )
}

function Meta({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex justify-between gap-3">
      <dt className="text-muted">{label}</dt>
      <dd className={cn('truncate text-right text-fg/90', mono && 'font-mono text-[11.5px]')}>{value}</dd>
    </div>
  )
}

function Timeline({ events }: { events: RunEvent[] }) {
  const { t } = useI18n()
  const ref = useRef<HTMLDivElement>(null)
  const items = useMemo(() => events.map((ev) => ({ ev, text: describe(ev, t) })), [events, t])

  useEffect(() => {
    const el = ref.current
    if (el) el.scrollTop = el.scrollHeight
  }, [items.length])

  if (items.length === 0) {
    return (
      <div className="flex h-[320px] items-center justify-center text-[12px] text-muted">
        {t('runs.waitingEvents')}
      </div>
    )
  }

  return (
    <div ref={ref} className="max-h-[560px] overflow-y-auto">
      {items.map(({ ev, text }) => {
        const Icon = iconFor(ev.type)
        return (
          <div key={ev.seq} className="flex gap-3 border-b border-line/60 px-3 py-2 last:border-0">
            <Icon className={cn('mt-0.5 h-3.5 w-3.5 shrink-0', colorFor(ev.type))} />
            <div className="min-w-0 flex-1">
              <div className="flex items-baseline gap-2">
                <span className="font-mono text-[11px] text-muted">{ev.type}</span>
                <span className="text-[10px] text-muted/60">
                  {new Date(ev.ts).toLocaleTimeString()}
                </span>
              </div>
              {text && <p className="mt-0.5 break-words text-[12.5px] text-fg/90">{text}</p>}
            </div>
          </div>
        )
      })}
    </div>
  )
}

function describe(ev: RunEvent, t: TFunction): string {
  const p = (ev.payload ?? {}) as Record<string, unknown>
  switch (ev.type) {
    case 'agent.message':
      return String(p.text ?? '')
    case 'tool.call':
      return [p.name, p.args].filter(Boolean).join(' · ')
    case 'tool.result':
      return statusLabel(t, String(p.status ?? 'done'))
    case 'run.leased':
      return t('runs.actLeased', {
        machine: String(p.machine ?? 'machine'),
        fence: String(p.fence ?? ''),
      })
    case 'workspace.ready':
      return t('runs.actWorktree', { branch: String(p.branch ?? '') })
    case 'approval.requested':
      return t('runs.actNeedsApproval', {
        tool: String(p.tool ?? 'tool'),
        risk: String(p.risk ?? ''),
      })
    case 'run.finished':
      return statusLabel(t, String(p.status ?? ''))
    case 'run.error':
      return `${String(p.code ?? '')} ${String(p.message ?? '')}`
    default:
      return ''
  }
}

function iconFor(type: string) {
  if (type.startsWith('tool')) return SquareTerminal
  if (type.startsWith('approval')) return ShieldCheck
  if (type.includes('error') || type === 'run.finished') return AlertTriangle
  if (type === 'agent.message') return MessageSquare
  if (type === 'run.created') return Circle
  if (type === 'workspace.ready') return CheckCircle2
  return Circle
}

function colorFor(type: string) {
  if (type.startsWith('tool')) return 'text-accent2'
  if (type.startsWith('approval')) return 'text-warn'
  if (type.includes('error')) return 'text-err'
  if (type === 'run.finished') return 'text-ok'
  return 'text-muted'
}
