import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Monitor, Cpu, Terminal, Power, PauseCircle } from 'lucide-react'
import { api, type Machine } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { cn } from '../lib/utils'
import { Button } from '../components/ui/button'
import { ConnectMachineDialog } from '../components/connect-machine-dialog'

export default function MachinesPage() {
  const { t } = useI18n()
  const { data, isLoading } = useQuery({
    queryKey: ['machines'],
    queryFn: api.machines,
    refetchInterval: 10000,
  })
  const machines = data?.items ?? []
  const { data: catalogData } = useQuery({ queryKey: ['runtimes'], queryFn: api.runtimes })
  const catalog = catalogData?.items ?? []
  const installed = new Set(machines.flatMap((machine) => Object.keys(machine.runtimes ?? {})))
  const now = Date.now()

  return (
    <div className="mx-auto max-w-[1100px] px-6 py-5">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-[18px] font-semibold tracking-tight">{t('page.machines')}</h1>
          <p className="mt-0.5 text-[12px] text-muted">{t('machines.subtitle')}</p>
        </div>
        <ConnectMachineDialog />
      </div>

      {isLoading && <p className="mt-4 text-[12px] text-muted">{t('common.loading')}</p>}

      <div className="mt-4 space-y-3">
        {machines.map((machine) => {
          const online = machine.status === 'online' && now - new Date(machine.lastSeen).getTime() < 45000
          const runtimes = Object.entries(machine.runtimes ?? {})
          return (
            <div key={machine.id} className="rounded-lg border border-line bg-panel/40 p-4">
              <div className="flex flex-wrap items-start justify-between gap-3">
                <div className="flex items-start gap-3">
                  <Monitor className="mt-0.5 h-4 w-4 text-muted" />
                  <div>
                    <div className="flex items-center gap-2">
                      <Link to={`/machines/${machine.id}`} className="text-[13px] font-medium hover:text-accent2">
                        {machine.name}
                      </Link>
                      <span
                        className={cn(
                          'rounded px-1.5 py-0.5 text-[10px]',
                          online ? 'bg-ok/15 text-ok' : 'bg-panel2 text-muted',
                        )}
                      >
                        {online ? t('machines.online') : t('machines.offline')}
                      </span>
                    </div>
                    <p className="mt-0.5 font-mono text-[10.5px] text-muted">{machine.id}</p>
                    <div className="mt-1 flex flex-wrap gap-1">
                      {(machine.labels ?? []).map((label) => (
                        <span key={label} className="rounded bg-panel2 px-1.5 py-0.5 font-mono text-[10px] text-muted">{label}</span>
                      ))}
                    </div>
                  </div>
                </div>
                <div className="flex items-center gap-3 text-[11px] text-muted">
                  <span>{t('machines.slots')}: {machine.concurrency}</span>
                  <span>{new Date(machine.lastSeen).toLocaleString()}</span>
                  <MachineStatusControl machine={machine} />
                </div>
              </div>

              <div className="mt-3 grid gap-3 sm:grid-cols-2">
                <div>
                  <div className="flex items-center gap-1.5 text-[10.5px] uppercase tracking-wider text-muted/70">
                    <Terminal className="h-3 w-3" />
                    {t('machines.runtimes')}
                  </div>
                  <div className="mt-1 flex flex-wrap gap-1">
                    {runtimes.map(([name, version]) => (
                      <span
                        key={name}
                        className="rounded bg-panel2 px-1.5 py-0.5 font-mono text-[10.5px] text-fg/80"
                        title={version}
                      >
                        {name} <span className="text-muted">{version.slice(0, 18)}</span>
                      </span>
                    ))}
                    {runtimes.length === 0 && (
                      <span className="text-[11px] text-warn">{t('machines.noRuntimes')}</span>
                    )}
                  </div>
                </div>
                <div>
                  <div className="flex items-center gap-1.5 text-[10.5px] uppercase tracking-wider text-muted/70">
                    <Cpu className="h-3 w-3" />
                    {t('machines.agents')}
                  </div>
                  <div className="mt-1 flex flex-wrap gap-1">
                    {machine.agents.map((agent) => (
                      <span key={agent} className="rounded bg-accent/10 px-1.5 py-0.5 text-[10.5px] text-accent2">
                        {agent}
                      </span>
                    ))}
                    {machine.agents.length === 0 && (
                      <span className="text-[11px] text-muted/70">{t('common.none')}</span>
                    )}
                  </div>
                </div>
              </div>

              {machine.metrics ? (
                <div className="mt-3 grid gap-3 border-t border-line/60 pt-3 sm:grid-cols-3">
                  <ResourceBar
                    label={t('machines.cpu')}
                    value={`${machine.metrics.cpuPercent.toFixed(0)}%`}
                    ratio={machine.metrics.cpuPercent / 100}
                  />
                  <ResourceBar
                    label={t('machines.memory')}
                    value={`${formatBytes(machine.metrics.memoryUsedBytes)} / ${formatBytes(machine.metrics.memoryTotalBytes)}`}
                    ratio={
                      machine.metrics.memoryTotalBytes
                        ? machine.metrics.memoryUsedBytes / machine.metrics.memoryTotalBytes
                        : 0
                    }
                  />
                  <ResourceBar
                    label={t('machines.disk')}
                    value={`${formatBytes(machine.metrics.diskUsedBytes)} / ${formatBytes(machine.metrics.diskTotalBytes)}`}
                    ratio={
                      machine.metrics.diskTotalBytes
                        ? machine.metrics.diskUsedBytes / machine.metrics.diskTotalBytes
                        : 0
                    }
                  />
                </div>
              ) : (
                <p className="mt-3 border-t border-line/60 pt-3 text-[10.5px] text-muted/60">
                  {t('machines.noMetrics')}
                </p>
              )}
              {!online && <TakeoverControl machine={machine} machines={machines} />}
            </div>
          )
        })}

        {!isLoading && machines.length === 0 && (
          <div className="rounded-lg border border-dashed border-line2 px-4 py-12 text-center">
            <p className="text-[12.5px] font-medium">{t('machines.empty')}</p>
            <p className="mt-1 text-[11.5px] text-muted">{t('machines.emptyHint')}</p>
            <code className="mt-3 inline-block rounded bg-panel2 px-2 py-1 font-mono text-[11px] text-muted">
              tandemesh-agentd -plane http://127.0.0.1:8080
            </code>
            <div className="mt-3 flex justify-center">
              <ConnectMachineDialog />
            </div>
          </div>
        )}
      </div>

      <section className="mt-6">
        <h2 className="text-[13px] font-medium">{t('machines.guide')}</h2>
        <p className="mt-0.5 text-[11.5px] text-muted">{t('machines.guideHint')}</p>
        <div className="mt-3 space-y-2">
          {catalog.map((runtime) => (
            <div key={runtime.id} className="rounded-lg border border-line bg-panel/40 p-3">
              <div className="flex flex-wrap items-center gap-2">
                <span className="font-mono text-[12px]">{runtime.id}</span>
                <span className="text-[11px] text-muted">{runtime.name}</span>
                <span className={cn(
                  'ml-auto rounded px-1.5 py-0.5 text-[10px]',
                  installed.has(runtime.id) ? 'bg-ok/15 text-ok' : 'bg-panel2 text-muted',
                )}>
                  {installed.has(runtime.id) ? t('machines.installed') : t('machines.notInstalled')}
                </span>
              </div>
              <div className="mt-2 flex flex-wrap items-center gap-2">
                <code className="rounded bg-panel2 px-2 py-1 font-mono text-[11px] text-muted">{runtime.install}</code>
                <a href={runtime.docs} target="_blank" rel="noreferrer" className="text-[11px] text-accent2 hover:underline">
                  {t('machines.docs')}
                </a>
              </div>
            </div>
          ))}
        </div>
      </section>
    </div>
  )
}

function MachineStatusControl({ machine }: { machine: Machine }) {
  const { t } = useI18n()
  const qc = useQueryClient()
  const update = useMutation({
    mutationFn: (status: string) => api.setMachineStatus(machine.id, status),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['machines'] }),
  })
  if (machine.status === 'draining') {
    return (
      <Button size="sm" variant="ghost" disabled={update.isPending} onClick={() => update.mutate('online')}><Power className="h-3 w-3" /> {t('machines.resume')}</Button>
    )
  }
  if (machine.status === 'online') {
    return (
      <Button size="sm" variant="ghost" disabled={update.isPending} onClick={() => update.mutate('draining')}><PauseCircle className="h-3 w-3" /> {t('machines.drain')}</Button>
    )
  }
  return null
}

function TakeoverControl({ machine, machines }: { machine: Machine; machines: Machine[] }) {
  const { t } = useI18n()
  const qc = useQueryClient()
  const [target, setTarget] = useState('')
  const takeover = useMutation({
    mutationFn: () => api.takeoverMachine(machine.id, target),
    onSuccess: () => {
      setTarget('')
      qc.invalidateQueries({ queryKey: ['machines'] })
    },
  })
  const others = machines.filter(
    (m) =>
      m.id !== machine.id &&
      m.status === 'online' &&
      Date.now() - new Date(m.lastSeen).getTime() < 45000,
  )
  if (others.length === 0) {
    return (
      <div className="mt-3 border-t border-line pt-3">
        <p className="text-[11px] text-muted">{t('machines.takeoverNoTarget')}</p>
      </div>
    )
  }
  return (
    <div className="mt-3 border-t border-line pt-3">
      <p className="text-[11px] text-muted">{t('machines.takeoverHint')}</p>
      <div className="mt-2 flex flex-wrap items-center gap-2">
        <select
          value={target}
          onChange={(e) => setTarget(e.target.value)}
          className="h-7 rounded-md border border-line2 bg-panel px-2 text-[11.5px] text-fg"
        >
          <option value="">{t('machines.takeoverTarget')}</option>
          {others.map((m) => (
            <option key={m.id} value={m.id}>
              {m.name}
            </option>
          ))}
        </select>
        <Button size="sm" variant="primary" disabled={!target || takeover.isPending} onClick={() => takeover.mutate()}>
          {t('machines.takeover')}
        </Button>
      </div>
    </div>
  )
}
function ResourceBar({ label, value, ratio }: { label: string; value: string; ratio: number }) {
  const percent = Math.max(0, Math.min(100, Math.round(ratio * 100)))
  return (
    <div>
      <div className="flex items-center justify-between text-[10px] uppercase tracking-wider text-muted/70">
        <span>{label}</span>
        <span className="text-muted">{percent}%</span>
      </div>
      <div className="mt-1 h-1.5 overflow-hidden rounded-full bg-panel2">
        <div
          className={cn(
            'h-full rounded-full',
            percent >= 90 ? 'bg-err' : percent >= 70 ? 'bg-warn' : 'bg-ok',
          )}
          style={{ width: `${percent}%` }}
        />
      </div>
      <p className="mt-1 truncate font-mono text-[10px] text-muted" title={value}>
        {value}
      </p>
    </div>
  )
}

function formatBytes(value: number): string {
  if (!value) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let size = value
  let unit = 0
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024
    unit++
  }
  return `${size.toFixed(size >= 10 || unit === 0 ? 0 : 1)} ${units[unit]}`
}