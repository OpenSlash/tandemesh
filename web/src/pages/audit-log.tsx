import { useMemo, useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Loader2, RefreshCw, ScrollText, Search } from 'lucide-react'
import { api, type AuditLog } from '../lib/api'
import { useWorkspace } from '../lib/workspace'
import { useI18n } from '../lib/i18n'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { cn } from '../lib/utils'

const ACTION_TONE: Record<string, string> = {
  GET: 'text-muted',
  POST: 'text-ok',
  PATCH: 'text-warn',
  PUT: 'text-warn',
  DELETE: 'text-err',
}

function actionTone(action: string) {
  return ACTION_TONE[action] ?? 'text-muted'
}

export default function AuditLogPage() {
  const { workspaceId, workspaces } = useWorkspace()
  const { t } = useI18n()
  const qc = useQueryClient()
  const [query, setQuery] = useState('')
  const [actor, setActor] = useState('all')
  const [action, setAction] = useState('all')
  const { data, isLoading, error, isFetching } = useQuery({
    queryKey: ['audit-logs', workspaceId],
    queryFn: () => api.auditLogs(workspaceId),
    refetchInterval: 30000,
  })
  const logs = data?.auditLogs ?? []
  const actors = useMemo(() => [...new Set(logs.map((entry) => entry.actor).filter(Boolean))].sort(), [logs])
  const actions = useMemo(() => [...new Set(logs.map((entry) => entry.action).filter(Boolean))].sort(), [logs])
  const filtered = logs.filter((entry) => {
    if (actor !== 'all' && entry.actor !== actor) return false
    if (action !== 'all' && entry.action !== action) return false
    const haystack = `${entry.actor} ${entry.action} ${entry.resource} ${entry.detail ?? ''}`.toLowerCase()
    return haystack.includes(query.trim().toLowerCase())
  })
  const workspaceName = workspaces.find((workspace) => workspace.id === workspaceId)?.name ?? workspaceId

  return (
    <div className="flex h-full min-h-0 flex-col p-4">
      <div className="flex items-end justify-between gap-3">
        <div>
          <h1 className="text-[18px] font-semibold tracking-tight">{t('audit.title')}</h1>
          <p className="mt-0.5 text-[12px] text-muted">{t('audit.subtitle', { workspace: workspaceName })}</p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="secondary" disabled={isFetching} onClick={() => qc.invalidateQueries({ queryKey: ['audit-logs', workspaceId] })}>
            {isFetching ? <Loader2 className="h-3 w-3 animate-spin" /> : <RefreshCw className="h-3 w-3" />}
            {t('audit.refresh')}
          </Button>
          <div className="relative w-[300px]">
            <Search className="pointer-events-none absolute left-2.5 top-2 h-3.5 w-3.5 text-muted" />
            <Input className="pl-8" value={query} onChange={(event) => setQuery(event.target.value)} placeholder={t('audit.search')} />
          </div>
        </div>
      </div>

      <div className="mt-3 flex flex-wrap items-center gap-2">
        <select value={actor} onChange={(event) => setActor(event.target.value)} className="rounded-md border border-line bg-panel px-2 py-1 text-[11px] text-muted">
          <option value="all">{t('audit.allActors')}</option>
          {actors.map((value) => <option key={value} value={value}>{value}</option>)}
        </select>
        <select value={action} onChange={(event) => setAction(event.target.value)} className="rounded-md border border-line bg-panel px-2 py-1 text-[11px] text-muted">
          <option value="all">{t('audit.allMethods')}</option>
          {actions.map((value) => <option key={value} value={value}>{value}</option>)}
        </select>
        <span className="text-[10.5px] text-muted">{t('audit.entries', { filtered: filtered.length, total: logs.length })}</span>
      </div>

      {error && <p className="mt-3 text-[12px] text-err">{String(error.message)}</p>}

      <div className="mt-4 min-h-0 flex-1 overflow-y-auto rounded-lg border border-line bg-panel/20">
        {isLoading ? (
          <div className="flex items-center gap-2 p-4 text-[12px] text-muted">
            <Loader2 className="h-3.5 w-3.5 animate-spin" /> {t('audit.loading')}
          </div>
        ) : filtered.length === 0 ? (
          <div className="flex flex-col items-center gap-2 py-20 text-muted">
            <ScrollText className="h-7 w-7 opacity-30" />
            <p className="text-[13px]">{t('audit.empty')}</p>
            <p className="text-[11px]">{t('audit.emptyHint')}</p>
          </div>
        ) : (
          <table className="w-full table-fixed border-collapse text-left">
            <thead className="sticky top-0 bg-panel text-[9.5px] uppercase tracking-wider text-muted">
              <tr>
                <th className="w-[82px] border-b border-line px-3 py-2">{t('audit.time')}</th>
                <th className="w-[140px] border-b border-line px-3 py-2">{t('audit.actor')}</th>
                <th className="w-[80px] border-b border-line px-3 py-2">{t('audit.method')}</th>
                <th className="border-b border-line px-3 py-2">{t('audit.resource')}</th>
              </tr>
            </thead>
            <tbody>
              {filtered.map((entry: AuditLog) => (
                <tr key={entry.id} className="align-top text-[11.5px] hover:bg-panel2/40">
                  <td className="whitespace-nowrap border-b border-line/60 px-3 py-2 text-[10px] text-muted">
                    {new Date(entry.createdAt).toLocaleTimeString()}
                  </td>
                  <td className="truncate border-b border-line/60 px-3 py-2 text-muted">{entry.actor || t('audit.anonymous')}</td>
                  <td className={cn('border-b border-line/60 px-3 py-2 font-mono text-[10px] font-medium', actionTone(entry.action))}>
                    {entry.action}
                  </td>
                  <td className="border-b border-line/60 px-3 py-2">
                    <p className="truncate font-mono text-[10.5px] text-fg/80">{entry.resource}</p>
                    {entry.detail && <p className="mt-0.5 truncate text-[10px] text-muted">{entry.detail}</p>}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}
