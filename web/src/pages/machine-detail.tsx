import { Link, useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import {
  AlertCircle,
  ArrowLeft,
  CheckCircle2,
  Clock3,
  Cpu,
  ExternalLink,
  Monitor,
  Server,
  Terminal,
} from 'lucide-react'
import { api } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { cn } from '../lib/utils'

export default function MachineDetailPage() {
  const { t } = useI18n()
  const { id = '' } = useParams()
  const { data: machine, isLoading } = useQuery({
    queryKey: ['machine', id],
    queryFn: () => api.machine(id),
    enabled: !!id,
    refetchInterval: 10000,
  })
  const { data: catalogData } = useQuery({ queryKey: ['runtimes'], queryFn: api.runtimes })
  const catalog = catalogData?.items ?? []
  const detected = machine?.runtimes ?? {}
  const knownIds = new Set(catalog.map((runtime) => runtime.id))
  const extras = Object.entries(detected).filter(([name]) => !knownIds.has(name))
  const online =
    machine?.status === 'online' && Date.now() - new Date(machine.lastSeen).getTime() < 45000

  return (
    <div className="mx-auto max-w-[980px] px-6 py-5">
      <Link to="/machines" className="inline-flex items-center gap-1 text-[12px] text-muted hover:text-fg">
        <ArrowLeft className="h-3.5 w-3.5" /> {t('machines.back')}
      </Link>

      {isLoading && <p className="mt-6 text-[12px] text-muted">{t('common.loading')}</p>}
      {!isLoading && !machine && (
        <p className="mt-6 text-[12px] text-err">{t('machines.notFound')}</p>
      )}

      {machine && (
        <>
          <div className="mt-3 flex flex-wrap items-start justify-between gap-3">
            <div className="flex items-start gap-3">
              <Monitor className="mt-1 h-5 w-5 text-muted" />
              <div>
                <div className="flex items-center gap-2">
                  <h1 className="text-[18px] font-semibold tracking-tight">{machine.name}</h1>
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
              </div>
            </div>
            <div className="flex flex-wrap gap-1">
              {(machine.labels ?? []).map((label) => (
                <span key={label} className="rounded bg-panel2 px-1.5 py-0.5 font-mono text-[10px] text-muted">
                  {label}
                </span>
              ))}
            </div>
          </div>

          <div className="mt-4 grid gap-2 sm:grid-cols-4">
            <Metric icon={Cpu} label={t('machines.slots')} value={String(machine.concurrency)} />
            <Metric icon={Server} label={t('machines.version')} value={machine.version || '—'} />
            <Metric
              icon={Terminal}
              label={t('machines.runtimes')}
              value={String(Object.keys(detected).length)}
            />
            <Metric
              icon={Clock3}
              label={t('machines.lastSeen')}
              value={new Date(machine.lastSeen).toLocaleString()}
            />
          </div>

          {machine.metrics ? (
            <div className="mt-2 grid gap-2 sm:grid-cols-3">
              <Usage
                label={t('machines.cpu')}
                percent={machine.metrics.cpuPercent}
                value={`${machine.metrics.cpuPercent.toFixed(0)}%`}
              />
              <Usage
                label={t('machines.memory')}
                percent={
                  machine.metrics.memoryTotalBytes
                    ? (machine.metrics.memoryUsedBytes / machine.metrics.memoryTotalBytes) * 100
                    : 0
                }
                value={`${formatBytes(machine.metrics.memoryUsedBytes)} / ${formatBytes(machine.metrics.memoryTotalBytes)}`}
              />
              <Usage
                label={t('machines.disk')}
                percent={
                  machine.metrics.diskTotalBytes
                    ? (machine.metrics.diskUsedBytes / machine.metrics.diskTotalBytes) * 100
                    : 0
                }
                value={`${formatBytes(machine.metrics.diskUsedBytes)} / ${formatBytes(machine.metrics.diskTotalBytes)}`}
              />
            </div>
          ) : (
            <p className="mt-2 text-[10.5px] text-muted/60">{t('machines.noMetrics')}</p>
          )}

          <section className="mt-6">
            <div className="flex items-end justify-between gap-3">
              <div>
                <h2 className="text-[13px] font-medium">{t('machines.runtimeMatrix')}</h2>
                <p className="mt-0.5 text-[11.5px] text-muted">{t('machines.runtimeMatrixHint')}</p>
              </div>
            </div>
            <div className="mt-3 overflow-hidden rounded-lg border border-line">
              {catalog.map((runtime) => {
                const version = detected[runtime.id]
                return (
                  <div
                    key={runtime.id}
                    className="flex flex-wrap items-center gap-2 border-b border-line/60 px-3 py-2.5 last:border-0"
                  >
                    <Terminal className="h-3.5 w-3.5 text-muted" />
                    <span className="text-[12.5px] font-medium">{runtime.name}</span>
                    <span className="font-mono text-[10.5px] text-muted">{runtime.id}</span>
                    {runtime.acp && (
                      <span className="rounded bg-accent/10 px-1 py-0.5 text-[9.5px] text-accent2">ACP</span>
                    )}
                    <span className="ml-auto flex items-center gap-1.5 text-[10.5px]">
                      {version ? (
                        <>
                          <CheckCircle2 className="h-3.5 w-3.5 text-ok" />
                          <span className="text-ok">{t('machines.installed')}</span>
                          <span className="max-w-[180px] truncate font-mono text-muted" title={version}>
                            {version}
                          </span>
                        </>
                      ) : (
                        <>
                          <AlertCircle className="h-3.5 w-3.5 text-muted/60" />
                          <span className="text-muted">{t('machines.notInstalled')}</span>
                        </>
                      )}
                    </span>
                    {!version && (
                      <div className="flex w-full items-center gap-2 pl-5">
                        <code className="rounded bg-panel2 px-2 py-1 font-mono text-[10.5px] text-muted">
                          {runtime.install}
                        </code>
                        <a
                          href={runtime.docs}
                          target="_blank"
                          rel="noreferrer"
                          className="inline-flex items-center gap-1 text-[10.5px] text-accent2 hover:underline"
                        >
                          {t('machines.docs')} <ExternalLink className="h-3 w-3" />
                        </a>
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          </section>

          {extras.length > 0 && (
            <section className="mt-5">
              <h2 className="text-[13px] font-medium">{t('machines.detectedExtras')}</h2>
              <p className="mt-0.5 text-[11.5px] text-muted">{t('machines.detectedExtrasHint')}</p>
              <div className="mt-2 flex flex-wrap gap-1.5">
                {extras.map(([name, version]) => (
                  <span key={name} className="rounded bg-panel2 px-2 py-1 font-mono text-[10.5px] text-muted">
                    {name} <span className="text-fg/60">{version}</span>
                  </span>
                ))}
              </div>
            </section>
          )}

          <section className="mt-5">
            <h2 className="text-[13px] font-medium">{t('machines.configuredAgents')}</h2>
            <p className="mt-0.5 text-[11.5px] text-muted">{t('machines.configuredAgentsHint')}</p>
            <div className="mt-2 flex flex-wrap gap-1.5">
              {machine.agents.map((agent) => (
                <span
                  key={agent}
                  className="rounded bg-accent/10 px-2 py-1 text-[11px] text-accent2"
                >
                  {agent}
                </span>
              ))}
              {machine.agents.length === 0 && (
                <span className="text-[11px] text-muted/70">{t('common.none')}</span>
              )}
            </div>
          </section>

          <p className="mt-6 text-[10.5px] text-muted/60">{t('machines.detectionNote')}</p>
        </>
      )}
    </div>
  )
}

function Metric({
  icon: Icon,
  label,
  value,
}: {
  icon: typeof Cpu
  label: string
  value: string
}) {
  return (
    <div className="rounded-lg border border-line bg-panel/40 p-3">
      <div className="flex items-center gap-1.5 text-[10.5px] uppercase tracking-wider text-muted/70">
        <Icon className="h-3 w-3" />
        {label}
      </div>
      <p className="mt-1 truncate text-[12px] text-fg" title={value}>
        {value}
      </p>
    </div>
  )
}
function Usage({ label, percent, value }: { label: string; percent: number; value: string }) {
  const bounded = Math.max(0, Math.min(100, percent))
  return (
    <div className="rounded-lg border border-line bg-panel/40 p-3">
      <div className="flex items-center justify-between text-[10px] uppercase tracking-wider text-muted/70">
        <span>{label}</span>
        <span className="text-muted">{bounded.toFixed(0)}%</span>
      </div>
      <div className="mt-1.5 h-1.5 overflow-hidden rounded-full bg-panel2">
        <div
          className={cn(
            'h-full rounded-full',
            bounded >= 90 ? 'bg-err' : bounded >= 70 ? 'bg-warn' : 'bg-ok',
          )}
          style={{ width: `${bounded}%` }}
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