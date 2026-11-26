import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { AlertCircle, AlertTriangle, CheckCircle2, ExternalLink, Loader2, Search } from 'lucide-react'
import { api, waitForMachineJob, type MachineJob, type RuntimeStatus } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { cn } from '../lib/utils'
import { isLocalHost } from '../lib/hosts'
import { Input } from '../components/ui/input'
import { MachinesTabs } from '../components/machines-tabs'
import { AgentIcon } from '../components/agent-icon'

function isOnline(machine: { status: string; lastSeen: string }) {
  return machine.status === 'online' && Date.now() - new Date(machine.lastSeen).getTime() < 45000
}

/** One agent, whether it comes from the ACP Registry, the built-in catalog or both. */
interface AgentRow {
  /** Backend id runs use, e.g. "claude" for the registry's "claude-acp". */
  backend: string
  name: string
  description?: string
  link?: string
  registryId?: string
  registryVersion?: string
  /** Manual install command for agents the registry does not carry. */
  install?: string
  known: boolean
}

export default function AgentCatalogPage() {
  const { t } = useI18n()
  const queryClient = useQueryClient()
  const [searchParams] = useSearchParams()
  // Links from a host page preselect that host and agent.
  const [query, setQuery] = useState(searchParams.get('q') ?? '')
  const [hostId, setHostId] = useState(searchParams.get('host') ?? '')
  const { data: registry, isLoading, error } = useQuery({
    queryKey: ['acp-registry'],
    queryFn: () => api.acpRegistry(),
    staleTime: 5 * 60 * 1000,
  })
  const { data: catalogData } = useQuery({ queryKey: ['runtimes'], queryFn: api.runtimes })
  const { data: machinesData } = useQuery({ queryKey: ['machines'], queryFn: api.machines, refetchInterval: 15000 })
  const hosts = (machinesData?.items ?? []).filter(isOnline)
  const host = hosts.find((machine) => machine.id === hostId) ?? hosts[0]
  const statusOf = new Map((host?.runtimeDetails ?? []).map((status) => [status.id, status]))

  const catalog = catalogData?.items ?? []
  const backendFor = new Map(catalog.filter((r) => r.registryId).map((r) => [r.registryId!, r.id]))
  const rows: AgentRow[] = [
    ...(registry?.items ?? []).map((item) => ({
      backend: backendFor.get(item.id) ?? item.id,
      name: item.name,
      description: item.description,
      link: item.repository || item.website,
      registryId: item.id,
      registryVersion: item.version,
      known: backendFor.has(item.id),
    })),
    ...catalog
      .filter((runtime) => runtime.acp && !runtime.registryId)
      .map((runtime) => ({
        backend: runtime.id,
        name: runtime.name,
        link: runtime.docs,
        install: runtime.install,
        known: true,
      })),
  ]
  const rank = (row: AgentRow) => (statusOf.get(row.backend)?.status === 'ready' ? 0 : row.known ? 1 : 2)
  const needle = query.trim().toLowerCase()
  const visible = rows
    .filter((row) => !needle || `${row.backend} ${row.registryId ?? ''} ${row.name} ${row.description ?? ''}`.toLowerCase().includes(needle))
    .sort((a, b) => rank(a) - rank(b) || a.name.localeCompare(b.name))

  const job = useMutation({
    mutationFn: async ({ start }: { registryId: string; start: () => Promise<MachineJob> }) => {
      const finished = await waitForMachineJob((await start()).id)
      if (finished.status === 'failed') throw new Error(finished.error ?? 'failed')
      return finished
    },
    onSettled: () => {
      void queryClient.invalidateQueries({ queryKey: ['machines'] })
      if (host) void queryClient.invalidateQueries({ queryKey: ['machine', host.id] })
    },
  })
  const busy = job.isPending ? job.variables?.registryId : undefined
  const hostLabel = (machine: { name: string; labels: string[] }) =>
    isLocalHost(machine) ? t('acp.localHost', { name: machine.name }) : machine.name

  return (
    <div className="mx-auto max-w-[1100px] px-6 py-5">
      <MachinesTabs />
      <h1 className="text-[18px] font-semibold tracking-tight">{t('acp.title')}</h1>
      <p className="mt-1 text-[12px] text-muted">{t('acp.subtitle')}</p>
      <div className="mt-4 flex flex-wrap items-center gap-3">
        <div className="relative w-full max-w-[420px]">
          <Search className="pointer-events-none absolute left-2 top-2 h-3.5 w-3.5 text-muted" />
          <Input className="h-8 pl-7" placeholder={t('acp.search')} value={query} onChange={(event) => setQuery(event.target.value)} />
        </div>
        {hosts.length > 1 ? (
          <label className="flex items-center gap-2 text-[11.5px] text-muted">
            {t('acp.host')}
            <select
              value={host?.id ?? ''}
              onChange={(event) => setHostId(event.target.value)}
              className="h-8 rounded-md border border-line2 bg-panel px-2 text-[11.5px] text-fg"
            >
              {hosts.map((machine) => (
                <option key={machine.id} value={machine.id}>{hostLabel(machine)}</option>
              ))}
            </select>
          </label>
        ) : host ? (
          // With one host there is nothing to choose; just say where installs go.
          <span className="text-[11.5px] text-muted">
            {t('acp.host')} <span className="text-fg">{hostLabel(host)}</span>
          </span>
        ) : (
          <span className="text-[11.5px] text-warn">{t('acp.noHost')}</span>
        )}
      </div>
      {isLoading && <p className="mt-5 text-[12px] text-muted">{t('acp.loading')}</p>}
      {error && <p className="mt-5 text-[12px] text-err">{String(error)}</p>}
      {job.error && <p className="mt-3 text-[11.5px] text-err">{t('machines.jobFailed', { error: job.error.message })}</p>}
      <div className="mt-4 divide-y divide-line2 rounded-lg border border-line2">
        {visible.map((row) => {
          const status = statusOf.get(row.backend)
          const fromThisRegistry = !!row.registryId && status?.registryId === row.registryId
          return (
            <div key={row.registryId ?? row.backend} className="flex items-start gap-3 px-3 py-2.5">
              <AgentIcon registryId={row.registryId} name={row.name} className="mt-0.5 text-fg/80" />
              <div className="min-w-0 flex-1">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-[12.5px] font-medium">{row.name}</span>
                  <span className="font-mono text-[10px] text-muted">{row.registryId ?? row.backend}</span>
                  {row.registryVersion && <span className="text-[10px] text-muted">v{row.registryVersion}</span>}
                  {row.registryId && row.backend !== row.registryId && (
                    <span className="rounded bg-accent/10 px-1 py-0.5 text-[9.5px] text-accent2">{t('acp.serves', { backend: row.backend })}</span>
                  )}
                  {host && <HostStatus status={status} />}
                </div>
                {row.description && <p className="mt-0.5 line-clamp-2 text-[11px] text-muted">{row.description}</p>}
                {status?.status === 'adapter-missing' && (
                  <p className="mt-0.5 text-[10.5px] text-warn">{t('machines.adapterMissingHint', { launcher: status.launcher ?? '' })}</p>
                )}
                {row.install && status?.status !== 'ready' && (
                  <code className="mt-1 inline-block rounded bg-panel2 px-2 py-0.5 font-mono text-[10.5px] text-muted">{row.install}</code>
                )}
              </div>
              <div className="flex shrink-0 items-center gap-2">
                {host && row.registryId && (fromThisRegistry || status?.status !== 'ready') && (
                  <button
                    type="button"
                    className="inline-flex items-center gap-1 rounded border border-line2 px-2 py-0.5 text-[10.5px] text-muted hover:text-fg disabled:opacity-50"
                    disabled={job.isPending}
                    onClick={() =>
                      job.mutate({
                        registryId: row.registryId!,
                        start: () =>
                          fromThisRegistry
                            ? api.uninstallMachineACP(host.id, row.registryId!)
                            : api.installMachineACP(host.id, row.registryId!),
                      })
                    }
                  >
                    {busy === row.registryId && <Loader2 className="h-3 w-3 animate-spin" />}
                    {fromThisRegistry
                      ? busy === row.registryId ? t('acp.uninstalling') : t('acp.uninstall')
                      : busy === row.registryId ? t('acp.installing') : t('acp.install')}
                  </button>
                )}
                {row.link && (
                  <a className="inline-flex items-center gap-1 text-[10.5px] text-accent2 hover:underline" href={row.link} target="_blank" rel="noreferrer">
                    <ExternalLink className="h-3 w-3" /> {t('acp.open')}
                  </a>
                )}
              </div>
            </div>
          )
        })}
        {!isLoading && visible.length === 0 && <p className="px-3 py-5 text-[12px] text-muted">{t('acp.empty')}</p>}
      </div>
    </div>
  )
}

/** The agent's state on the selected host; nothing for agents the host has never seen. */
function HostStatus({ status }: { status?: RuntimeStatus }) {
  const { t } = useI18n()
  if (!status || status.status === 'missing') return null
  const ready = status.status === 'ready'
  const Icon = ready ? CheckCircle2 : status.status === 'adapter-missing' ? AlertTriangle : AlertCircle
  const label = ready
    ? status.source === 'registry'
      ? t('acp.installed', { version: status.registryVersion ?? '' })
      : t('machines.statusReady')
    : status.status === 'adapter-missing'
      ? t('machines.statusAdapterMissing')
      : t('machines.statusUnsupported')
  return (
    <span className={cn('inline-flex items-center gap-1 rounded px-1.5 py-0.5 text-[10px]', ready ? 'bg-ok/15 text-ok' : 'bg-warn/10 text-warn')}>
      <Icon className="h-3 w-3" />
      {label}
    </span>
  )
}
