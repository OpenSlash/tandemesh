import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { AlertCircle, Check, Clock, Loader2, Search } from 'lucide-react'
import { api, type AgentSession } from '../lib/api'
import { Input } from '../components/ui/input'
import { cn } from '../lib/utils'
import { useWorkspace } from '../lib/workspace'

type Column = 'needs' | 'working' | 'done' | 'idle'

const columns: { id: Column; title: string; icon: typeof AlertCircle; tone: string }[] = [
  { id: 'needs', title: 'Needs You', icon: AlertCircle, tone: 'text-warn' },
  { id: 'working', title: 'Working', icon: Loader2, tone: 'text-accent2' },
  { id: 'done', title: 'Done', icon: Check, tone: 'text-ok' },
  { id: 'idle', title: 'Idle', icon: Clock, tone: 'text-muted' },
]

function stateOf(session: AgentSession): Column {
  if (session.lastRunStatus === 'waiting_approval') return 'needs'
  if (['queued', 'leased', 'preparing', 'running', 'finalizing'].includes(session.lastRunStatus ?? '')) return 'working'
  if (session.lastRunStatus === 'succeeded') return 'done'
  return 'idle'
}

function age(iso: string) {
  const minutes = Math.max(0, Math.floor((Date.now() - new Date(iso).getTime()) / 60000))
  if (minutes < 1) return 'now'
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h`
  return `${Math.floor(hours / 24)}d`
}

export default function AgentDashboardPage() {
  const { workspaceId } = useWorkspace()
  const [query, setQuery] = useState('')
  const [repoFilter, setRepoFilter] = useState('all')
  const [agentFilter, setAgentFilter] = useState('all')
  const [timeFilter, setTimeFilter] = useState<'all' | '24h' | '7d'>('all')
  const { data: sessionsData, isLoading } = useQuery({
    queryKey: ['sessions', 'all'],
    queryFn: () => api.sessions('all'),
    refetchInterval: 4000,
  })
  const { data: approvalsData } = useQuery({
    queryKey: ['approvals', 'all'],
    queryFn: () => api.approvals('all'),
    refetchInterval: 4000,
  })
  const { data: usage } = useQuery({
    queryKey: ['usage-trend', workspaceId],
    queryFn: () => api.usageTrend(workspaceId, 14),
    refetchInterval: 30000,
  })
  const sessions = sessionsData?.sessions ?? []
  const pendingRuns = useMemo(
    () => new Set((approvalsData?.items ?? []).filter((a) => a.status === 'pending').map((a) => a.runId)),
    [approvalsData],
  )
  const repos = useMemo(() => [...new Set(sessions.map((session) => session.repo).filter(Boolean))].sort(), [sessions])
  const agents = useMemo(() => [...new Set(sessions.map((session) => session.agentId).filter(Boolean))].sort(), [sessions])
  const filtered = sessions.filter((session) => {
    if (session.status === 'closed' && session.lastRunStatus !== 'succeeded') return false
    if (repoFilter !== 'all' && session.repo !== repoFilter) return false
    if (agentFilter !== 'all' && session.agentId !== agentFilter) return false
    if (timeFilter !== 'all') {
      const cutoff = Date.now() - (timeFilter === '24h' ? 24 : 24 * 7) * 3600_000
      if (new Date(session.updatedAt).getTime() < cutoff) return false
    }
    const text = `${session.title} ${session.repo ?? ''} ${session.agentId}`.toLowerCase()
    return text.includes(query.trim().toLowerCase())
  })
  const usagePoints = usage?.points ?? []
  const maxUsage = Math.max(1, ...usagePoints.map((point) => point.tokens))
  const grouped = (column: Column) =>
    filtered.filter((session) => {
      if (column === 'needs' && session.lastRunId && pendingRuns.has(session.lastRunId)) return true
      return stateOf(session) === column
    })

  return (
    <div className="flex h-full min-h-0 flex-col p-4">
      <div className="flex items-end justify-between gap-3">
        <div>
          <h1 className="text-[18px] font-semibold tracking-tight">Agent Dashboard</h1>
          <p className="mt-0.5 text-[12px] text-muted">Every agent session across permitted workspaces.</p>
        </div>
        <div className="relative w-[260px]">
          <Search className="pointer-events-none absolute left-2.5 top-2 h-3.5 w-3.5 text-muted" />
          <Input className="pl-8" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search agents, repos…" />
        </div>
      </div>

      <div className="mt-3 flex flex-wrap items-center gap-2">
        <select
          value={repoFilter}
          onChange={(event) => setRepoFilter(event.target.value)}
          className="rounded-md border border-line bg-panel px-2 py-1 text-[11px] text-muted"
        >
          <option value="all">All projects</option>
          {repos.map((repo) => <option key={repo} value={repo}>{repo}</option>)}
        </select>
        <select
          value={agentFilter}
          onChange={(event) => setAgentFilter(event.target.value)}
          className="rounded-md border border-line bg-panel px-2 py-1 text-[11px] text-muted"
        >
          <option value="all">All agents</option>
          {agents.map((agent) => <option key={agent} value={agent}>{agent}</option>)}
        </select>
        <select
          value={timeFilter}
          onChange={(event) => setTimeFilter(event.target.value as 'all' | '24h' | '7d')}
          className="rounded-md border border-line bg-panel px-2 py-1 text-[11px] text-muted"
        >
          <option value="all">Any time</option>
          <option value="24h">Last 24 hours</option>
          <option value="7d">Last 7 days</option>
        </select>
        <span className="text-[10.5px] text-muted">{filtered.length} sessions</span>
      </div>

      {usage && (
        <div className="mt-3 flex items-center gap-6 rounded-lg border border-line bg-panel/30 px-3 py-2">
          <div>
            <p className="text-[9.5px] uppercase tracking-wider text-muted">Tokens · 14d</p>
            <p className="text-[14px] font-medium">{usage.totalTokens >= 1_000_000 ? `${(usage.totalTokens / 1_000_000).toFixed(1)}M` : usage.totalTokens >= 1000 ? `${(usage.totalTokens / 1000).toFixed(1)}K` : usage.totalTokens}</p>
          </div>
          <div>
            <p className="text-[9.5px] uppercase tracking-wider text-muted">Cost · 14d</p>
            <p className="text-[14px] font-medium">${usage.totalCost.toFixed(2)}</p>
          </div>
          <div>
            <p className="text-[9.5px] uppercase tracking-wider text-muted">Runs</p>
            <p className="text-[14px] font-medium">{usage.totalRuns}</p>
          </div>
          <div className="ml-auto flex h-12 items-end gap-1">
            {usagePoints.map((point) => (
              <div
                key={point.date}
                title={`${point.date}: ${point.tokens} tokens · $${point.cost.toFixed(2)}`}
                className="w-1.5 rounded-t bg-fg/35 hover:bg-fg"
                style={{ height: `${Math.max(2, Math.round((point.tokens / maxUsage) * 44))}px` }}
              />
            ))}
          </div>
        </div>
      )}

      {isLoading ? (
        <p className="mt-6 text-[12px] text-muted">Loading sessions…</p>
      ) : (
        <div className="mt-4 grid min-h-0 flex-1 grid-cols-4 gap-3">
          {columns.map((column) => {
            const items = grouped(column.id)
            const Icon = column.icon
            return (
              <section key={column.id} className="flex min-h-0 flex-col rounded-lg border border-line bg-panel/30">
                <div className="flex items-center gap-2 border-b border-line px-3 py-2">
                  <Icon className={cn('h-3.5 w-3.5', column.tone, column.id === 'working' && 'animate-spin')} />
                  <span className="text-[11.5px] font-medium">{column.title}</span>
                  <span className="ml-auto text-[10.5px] text-muted">{items.length}</span>
                </div>
                <div className="min-h-0 flex-1 overflow-y-auto p-2">
                  {items.map((session) => (
                    <Link
                      key={session.id}
                      to={`/sessions?scope=all&session=${session.id}`}
                      className={cn(
                        'mb-2 block rounded-md border border-line bg-panel px-2.5 py-2 hover:border-line2',
                        column.id === 'needs' && 'border-warn/30 bg-warn/5',
                        column.id === 'done' && 'border-ok/20 bg-ok/5',
                      )}
                    >
                      <div className="flex items-start gap-2">
                        <span
                          className={cn(
                            'mt-1 h-1.5 w-1.5 shrink-0 rounded-full',
                            column.id === 'needs' && 'bg-warn',
                            column.id === 'working' && 'animate-pulse bg-accent2',
                            column.id === 'done' && 'bg-ok',
                            column.id === 'idle' && 'bg-muted/40',
                          )}
                        />
                        <span className="min-w-0 flex-1">
                          <span className="block truncate text-[12px] font-medium">{session.title}</span>
                          <span className="mt-0.5 block truncate text-[10.5px] text-muted">
                            {session.repo || 'No repository'}
                          </span>
                        </span>
                        <span className="shrink-0 text-[10px] text-muted">{age(session.updatedAt)}</span>
                      </div>
                      <div className="mt-1.5 flex items-center gap-2 text-[10px] text-muted">
                        <span className="truncate">{session.agentId}</span>
                        <span className="ml-auto shrink-0 uppercase">{session.lastRunStatus ?? session.status}</span>
                      </div>
                    </Link>
                  ))}
                  {items.length === 0 && <p className="px-1 py-3 text-[11px] text-muted/60">None</p>}
                </div>
              </section>
            )
          })}
        </div>
      )}
    </div>
  )
}
