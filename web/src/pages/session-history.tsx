import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { History, Loader2, Play, RefreshCw, Search } from 'lucide-react'
import { api, type HistoryEntry } from '../lib/api'
import { useWorkspace } from '../lib/workspace'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'

export default function SessionHistoryPage() {
  const { workspaceId } = useWorkspace()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const [query, setQuery] = useState('')
  const [agent, setAgent] = useState<'all' | 'claude' | 'codex' | 'opencode' | 'gemini' | 'cursor'>('all')
  const { data, isLoading, error } = useQuery({
    queryKey: ['session-history', query],
    queryFn: () => api.sessionHistory(query),
    refetchInterval: 30000,
  })
  const items = (data?.items ?? []).filter((entry) => agent === 'all' || entry.agent === agent)
  const scan = useMutation({
    mutationFn: () => api.scanHistory(),
    onSuccess: () => {
      window.setTimeout(() => void qc.invalidateQueries({ queryKey: ['session-history'] }), 2500)
    },
  })
  const resume = useMutation({
    mutationFn: (entry: HistoryEntry) =>
      api.resumeHistory({
        workspaceId,
        sessionId: entry.id,
        title: entry.title,
        repo: entry.cwd,
        model: entry.model,
        machineId: entry.machineId,
      }),
    onSuccess: (session) => {
      void qc.invalidateQueries({ queryKey: ['sessions'] })
      navigate(`/sessions?session=${session.id}`)
    },
  })

  return (
    <div className="flex h-full min-h-0 flex-col p-4">
      <div className="flex items-end justify-between gap-3">
        <div>
          <h1 className="text-[18px] font-semibold tracking-tight">Session history</h1>
          <p className="mt-0.5 text-[12px] text-muted">
            Agent sessions discovered on {data?.host || 'this host'} and connected machines.
          </p>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="secondary" disabled={scan.isPending} onClick={() => scan.mutate()}>
            {scan.isPending ? <Loader2 className="h-3 w-3 animate-spin" /> : <RefreshCw className="h-3 w-3" />}
            Scan hosts
          </Button>
          <div className="relative w-[300px]">
            <Search className="pointer-events-none absolute left-2.5 top-2 h-3.5 w-3.5 text-muted" />
            <Input className="pl-8" value={query} onChange={(event) => setQuery(event.target.value)} placeholder="Search title, project, preview..." />
          </div>
        </div>
      </div>

      <div className="mt-3 flex gap-1">
        {(['all', 'claude', 'codex', 'opencode', 'gemini', 'cursor'] as const).map((value) => (
          <button
            key={value}
            onClick={() => setAgent(value)}
            className={
              agent === value
                ? 'rounded-md bg-panel2 px-2 py-1 text-[11px] font-medium text-fg'
                : 'rounded-md px-2 py-1 text-[11px] text-muted hover:text-fg'
            }
          >
            {value === 'all' ? 'All' : value === 'claude' ? 'Claude Code' : value === 'codex' ? 'Codex' : value === 'opencode' ? 'OpenCode' : value === 'gemini' ? 'Gemini' : 'Cursor'}
          </button>
        ))}
      </div>

      {resume.error && <p className="mt-3 text-[12px] text-err">{String(resume.error.message)}</p>}
      {error && <p className="mt-3 text-[12px] text-err">{String(error.message)}</p>}

      <div className="mt-4 min-h-0 flex-1 overflow-y-auto">
        {isLoading && (
          <div className="flex items-center gap-2 text-[12px] text-muted">
            <Loader2 className="h-3.5 w-3.5 animate-spin" /> Scanning local sessions…
          </div>
        )}
        {!isLoading && items.length === 0 && (
          <div className="flex flex-col items-center gap-2 py-16 text-muted">
            <History className="h-7 w-7 opacity-30" />
            <p className="text-[13px]">No sessions found</p>
            <p className="text-[11px]">Start an agent session and it will appear here.</p>
          </div>
        )}
        <div className="space-y-2">
          {items.map((entry) => (
            <div key={entry.id} className="rounded-lg border border-line bg-panel/30 p-3">
              <div className="flex items-start gap-3">
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2">
                    <span className="truncate text-[12.5px] font-medium">{entry.title}</span>
                    <span className="rounded bg-panel2 px-1 text-[9.5px] uppercase text-muted">{entry.agent}</span>
                    {entry.machineName && (
                      <span className="rounded border border-line px-1 text-[9.5px] text-muted" title={entry.machineId}>
                        {entry.machineName}
                      </span>
                    )}
                  </div>
                  <div className="mt-1 truncate text-[10.5px] text-muted">
                    {entry.cwd}
                    {entry.branch ? ` · ${entry.branch}` : ''}
                    {entry.model ? ` · ${entry.model}` : ''}
                  </div>
                  {entry.preview && <p className="mt-1.5 line-clamp-2 text-[11.5px] text-muted">{entry.preview}</p>}
                </div>
                <div className="flex shrink-0 flex-col items-end gap-2">
                  <span className="text-[10px] text-muted">{new Date(entry.updatedAt).toLocaleString()}</span>
                  <Button
                    size="sm"
                    variant="primary"
                    disabled={resume.isPending || Boolean(entry.path?.endsWith('state.vscdb'))}
                    title={entry.path?.endsWith('state.vscdb') ? 'Cursor IDE sessions open in the editor' : entry.machineId ? 'Resume on ' + (entry.machineName || 'remote host') : 'Resume session'}
                    onClick={() => resume.mutate(entry)}
                  >
                    <Play className="h-3 w-3" /> {entry.path?.endsWith('state.vscdb') ? 'Read-only' : 'Resume'}
                  </Button>
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  )
}
