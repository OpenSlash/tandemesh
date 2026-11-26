import { useEffect, useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ExternalLink, History, MessageSquare, Plus, Search, X } from 'lucide-react'
import { api } from '../lib/api'
import { useWorkspace } from '../lib/workspace'
import { useI18n } from '../lib/i18n'
import { cn } from '../lib/utils'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Dialog, DialogContent, DialogTrigger } from '../components/ui/dialog'
import { SessionDetail } from './session-detail'

function sessionState(status: string, lastRunStatus?: string) {
  if (status === 'closed') return 'closed'
  if (status === 'archived') return 'archived'
  if (lastRunStatus === 'waiting_approval') return 'waiting'
  if (['queued', 'leased', 'preparing', 'running', 'finalizing'].includes(lastRunStatus ?? '')) return 'working'
  if (['failed', 'cancelled'].includes(lastRunStatus ?? '')) return 'blocked'
  if (lastRunStatus === 'succeeded') return 'done'
  return 'idle'
}

export default function SessionsPage() {
  const { t } = useI18n()
  const { workspaceId } = useWorkspace()
  const [searchParams, setSearchParams] = useSearchParams()
  const scope = searchParams.get('scope') === 'all' ? 'all' : workspaceId
  const qc = useQueryClient()
  const [showArchived, setShowArchived] = useState(false)
  const { data, isLoading } = useQuery({
    queryKey: ['sessions', scope, showArchived ? 'archived' : ''],
    queryFn: () => api.sessions(scope, undefined, showArchived ? 'archived' : undefined),
    refetchInterval: 5000,
  })
  const items = data?.sessions ?? []
  const [selectedId, setSelectedId] = useState(searchParams.get('session') ?? '')
  const [openTabs, setOpenTabs] = useState<string[]>([])
  const [draggedTab, setDraggedTab] = useState<string | null>(null)
  const [query, setQuery] = useState('')
  const [open, setOpen] = useState(false)
  const [title, setTitle] = useState('')
  const [repo, setRepo] = useState('')
  const [base, setBase] = useState('main')
  const [repoChoice, setRepoChoice] = useState('')
  const [worktreeId, setWorktreeId] = useState('')
  const [model, setModel] = useState('')
  const [reasoning, setReasoning] = useState('')
  const { data: agentsData } = useQuery({
    queryKey: ['agents', workspaceId],
    queryFn: () => api.agents(workspaceId),
  })
  const agents = agentsData?.items ?? []
  const { data: reposData } = useQuery({
    queryKey: ['repositories', workspaceId],
    queryFn: () => api.repositories(workspaceId),
  })
  const repositories = reposData?.items ?? []
  const { data: worktreesData } = useQuery({
    queryKey: ['worktrees', workspaceId, repo],
    queryFn: () => api.worktrees({ workspaceId, repo }),
    enabled: Boolean(repo),
  })
  const worktrees = worktreesData?.worktrees ?? []
  const [agentId, setAgentId] = useState('')
  const selectedAgentId = agentId || agents[0]?.id || 'claude'
  const [mode, setMode] = useState('chat')

  useEffect(() => {
    const fromUrl = searchParams.get('session')
    if (fromUrl && fromUrl !== selectedId) setSelectedId(fromUrl)
  }, [searchParams, selectedId])

  useEffect(() => {
    if (!selectedId && items.length > 0) setSelectedId(items[0].id)
  }, [items, selectedId])

  useEffect(() => {
    try {
      const raw = window.localStorage.getItem(`orca.session.tabs.${workspaceId}`)
      const parsed = raw ? (JSON.parse(raw) as string[]) : []
      setOpenTabs(Array.isArray(parsed) ? parsed : [])
    } catch {
      setOpenTabs([])
    }
  }, [workspaceId])

  useEffect(() => {
    if (!selectedId) return
    setOpenTabs((tabs) => (tabs.includes(selectedId) ? tabs : [...tabs, selectedId].slice(-10)))
  }, [selectedId])

  useEffect(() => {
    window.localStorage.setItem(`orca.session.tabs.${workspaceId}`, JSON.stringify(openTabs))
  }, [openTabs, workspaceId])

  const selectSession = (id: string) => {
    setSelectedId(id)
    setSearchParams(scope === 'all' ? { scope: 'all', session: id } : { session: id })
  }
  const closeTab = (id: string) => {
    setOpenTabs((tabs) => {
      const next = tabs.filter((tab) => tab !== id)
      if (selectedId === id) {
        const replacement = next[next.length - 1] ?? ''
        setSelectedId(replacement)
        setSearchParams(scope === 'all' ? { scope: 'all', ...(replacement ? { session: replacement } : {}) } : replacement ? { session: replacement } : {})
      }
      return next
    })
  }
  const reorderTab = (targetId: string) => {
    if (!draggedTab || draggedTab === targetId) return
    setOpenTabs((tabs) => {
      const from = tabs.indexOf(draggedTab)
      const to = tabs.indexOf(targetId)
      if (from < 0 || to < 0) return tabs
      const next = [...tabs]
      next.splice(from, 1)
      next.splice(to, 0, draggedTab)
      return next
    })
    setDraggedTab(null)
  }

  const filtered = items.filter((item) => {
    const text = `${item.title} ${item.repo ?? ''} ${item.agentId}`.toLowerCase()
    return text.includes(query.trim().toLowerCase())
  })
  const groups = useMemo(() => {
    const map = new Map<string, typeof filtered>()
    for (const item of filtered) {
      const key = item.repo || 'No repository'
      map.set(key, [...(map.get(key) ?? []), item])
    }
    return [...map.entries()]
  }, [filtered])

  const create = useMutation({
    mutationFn: () => {
      const agent = agents.find((item) => item.id === selectedAgentId)
      return api.createSession({
        workspaceId,
        title,
        agentId: selectedAgentId,
        backend: agent?.backend,
        model: model || undefined,
        reasoning: reasoning || undefined,
        mode,
        repo: repo || undefined,
        base: base || undefined,
        worktreeId: worktreeId || undefined,
      })
    },
    onSuccess: (session) => {
      setOpen(false)
      setTitle('')
      setRepo('')
      setRepoChoice('')
      setWorktreeId('')
      setModel('')
      setReasoning('')
      selectSession(session.id)
      void qc.invalidateQueries({ queryKey: ['sessions'] })
    },
  })

  return (
    <div className="flex h-full min-h-0">
      <aside className={cn('w-[300px] shrink-0 flex-col border-r border-line', selectedId ? 'hidden md:flex' : 'flex')}>
        <div className="flex items-start justify-between gap-2 px-4 pb-2 pt-4">
          <div>
            <h1 className="text-[16px] font-semibold tracking-tight">
              {scope === 'all' ? t('nav.recentSessions') : t('page.sessions')}
            </h1>
            <p className="mt-0.5 text-[11px] text-muted">{items.length} sessions</p>
          </div>
          <div className="flex items-center gap-1">
            <Link
              to="/session-history"
              title={t('nav.sessionHistory')}
              aria-label={t('nav.sessionHistory')}
              className="flex h-7 w-7 items-center justify-center rounded-md text-muted hover:bg-panel2 hover:text-fg"
            >
              <History className="h-3.5 w-3.5" />
            </Link>
            <Dialog
              open={open}
              onOpenChange={(value) => {
                setOpen(value)
                if (value && !agentId) setAgentId(agents[0]?.id ?? 'claude')
              }}
            >
            <DialogTrigger asChild>
              <Button variant="primary" size="sm">
                <Plus className="h-3.5 w-3.5" />
              </Button>
            </DialogTrigger>
            <DialogContent title={t('sessions.new')} description={t('sessions.newHint')}>
              <form
                className="flex flex-col gap-3"
                onSubmit={(e) => {
                  e.preventDefault()
                  create.mutate()
                }}
              >
                <Input autoFocus value={title} onChange={(e) => setTitle(e.target.value)} placeholder={t('sessions.title')} />
                <select
                  value={selectedAgentId}
                  onChange={(e) => setAgentId(e.target.value)}
                  className="rounded-md border border-line2 bg-panel px-2 py-1.5 text-[12px] text-fg"
                >
                  {agents.length === 0 && <option value="claude">claude</option>}
                  {agents.map((agent) => (
                    <option key={agent.id} value={agent.id}>
                      {agent.name} · {agent.backend}
                    </option>
                  ))}
                </select>
                <div className="flex flex-col gap-2">
                  <select
                    value={repoChoice}
                    onChange={(event) => {
                      const value = event.target.value
                      setRepoChoice(value)
                      setWorktreeId('')
                      const found = repositories.find((item) => item.repo === value)
                      if (found) {
                        setRepo(found.repo)
                        setBase(found.baseRef || 'main')
                      } else {
                        setRepo('')
                        setBase('main')
                      }
                    }}
                    className="rounded-md border border-line2 bg-panel px-2 py-1.5 text-[12px] text-fg"
                  >
                    <option value="">Custom repository…</option>
                    {repositories.map((item) => (
                      <option key={item.repo} value={item.repo}>
                        {item.repo} · {item.worktreeCount} worktree{item.worktreeCount === 1 ? '' : 's'}
                      </option>
                    ))}
                  </select>
                  {!repoChoice && (
                    <Input value={repo} onChange={(e) => setRepo(e.target.value)} placeholder={t('sessions.repoPlaceholder')} />
                  )}
                  {repo && (
                    <select
                      value={worktreeId}
                      onChange={(event) => {
                        const value = event.target.value
                        setWorktreeId(value)
                        const wt = worktrees.find((item) => item.id === value)
                        if (wt) setBase(wt.baseRef || 'main')
                      }}
                      className="rounded-md border border-line2 bg-panel px-2 py-1.5 text-[12px] text-fg"
                    >
                      <option value="">{t('sessions.newWorktree')}</option>
                      {worktrees.map((wt) => (
                        <option key={wt.id} value={wt.id}>
                          {wt.branch || wt.id} · {wt.status}
                        </option>
                      ))}
                    </select>
                  )}
                  <Input value={base} onChange={(e) => setBase(e.target.value)} placeholder={t('sessions.baseRef')} disabled={Boolean(worktreeId)} />
                </div>
                <Input value={model} onChange={(e) => setModel(e.target.value)} placeholder={t('sessions.modelOverride')} />
                <select value={reasoning} onChange={(e) => setReasoning(e.target.value)} className="rounded-md border border-line2 bg-panel px-2 py-1.5 text-[12px] text-fg">
                  <option value="">{t('sessions.reasoningDefault')}</option>
                  <option value="low">{t('sessions.reasoningLow')}</option>
                  <option value="medium">{t('sessions.reasoningMedium')}</option>
                  <option value="high">{t('sessions.reasoningHigh')}</option>
                </select>
                <select value={mode} onChange={(e) => setMode(e.target.value)} className="rounded-md border border-line2 bg-panel px-2 py-1.5 text-[12px] text-fg">
                  <option value="chat">{t('sessions.chat')}</option>
                  <option value="terminal">{t('sessions.terminal')}</option>
                </select>
                {create.error && <p className="text-[12px] text-err">{String(create.error.message)}</p>}
                <div className="flex justify-end gap-2">
                  <Button type="button" variant="ghost" onClick={() => setOpen(false)}>{t('common.cancel')}</Button>
                  <Button type="submit" variant="primary" disabled={!title || create.isPending}>{t('common.create')}</Button>
                </div>
              </form>
            </DialogContent>
          </Dialog>
          </div>
        </div>

        <div className="relative px-3 pb-2">
          <Search className="pointer-events-none absolute left-5 top-2 h-3.5 w-3.5 text-muted" />
          <Input className="pl-8" value={query} onChange={(event) => setQuery(event.target.value)} placeholder={t('sessions.search')} />
        </div>
        <div className="flex items-center gap-2 px-3 pb-2">
          <button
            onClick={() => setShowArchived((value) => !value)}
            className={cn(
              'rounded border px-1.5 py-0.5 text-[10px]',
              showArchived ? 'border-fg/40 bg-panel2 text-fg' : 'border-line text-muted hover:text-fg',
            )}
          >
            {t('sessions.archived')}
          </button>
          {showArchived && <span className="text-[10px] text-muted">{t('sessions.archivedHint')}</span>}
        </div>

        <div className="min-h-0 flex-1 overflow-y-auto px-2 pb-3">
          {isLoading && <p className="px-2 py-3 text-[12px] text-muted">{t('common.loading')}</p>}
          {groups.map(([repoName, sessions]) => (
            <div key={repoName} className="mb-2">
              <div className="px-2 py-1 text-[10px] uppercase tracking-wider text-muted/70">{repoName}</div>
              {sessions.map((session) => {
                const state = sessionState(session.status, session.lastRunStatus)
                return (
                  <button
                    key={session.id}
                    onClick={() => selectSession(session.id)}
                    className={cn(
                      'mb-0.5 flex w-full items-start gap-2 rounded-md px-2 py-2 text-left transition-colors',
                      session.id === selectedId ? 'bg-panel2 text-fg' : 'text-muted hover:bg-panel2/50 hover:text-fg',
                    )}
                  >
                    <span
                      className={cn(
                        'mt-1 h-1.5 w-1.5 shrink-0 rounded-full',
                        state === 'working' && 'bg-warn animate-pulse',
                        state === 'waiting' && 'bg-warn',
                        state === 'done' && 'bg-ok',
                        state === 'blocked' && 'bg-err',
                        (state === 'closed' || state === 'archived') && 'bg-line2',
                        state === 'idle' && 'bg-muted/40',
                      )}
                    />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-[12px]">{session.title}</span>
                      <span className="mt-0.5 block truncate text-[10.5px] text-muted">
                        {agents.find((agent) => agent.id === session.agentId)?.name ?? session.agentId} · {session.mode}
                      </span>
                    </span>
                    {state === 'waiting' && <span className="text-[10px] text-warn">!</span>}
                  </button>
                )
              })}
            </div>
          ))}
          {!isLoading && items.length === 0 && <p className="px-2 py-4 text-[12px] text-muted">{t('sessions.empty')}</p>}
        </div>
      </aside>

      <section className={cn('min-w-0 flex-1 flex-col', selectedId ? 'flex' : 'hidden md:flex')}>
        {selectedId && (
          <button
            onClick={() => {
              setSelectedId('')
              setSearchParams(scope === 'all' ? { scope: 'all' } : {})
            }}
            className="flex h-9 shrink-0 items-center gap-1 border-b border-line px-3 text-[11.5px] text-muted md:hidden"
          >
            ← Sessions
          </button>
        )}
        {openTabs.length > 0 && (
          <div className="flex h-9 shrink-0 items-center gap-1 overflow-x-auto border-b border-line px-2">
            {openTabs.map((tabId) => {
              const title = items.find((item) => item.id === tabId)?.title ?? 'Session'
              return (
                <div
                  key={tabId}
                  draggable
                  onDragStart={(event) => {
                    setDraggedTab(tabId)
                    event.dataTransfer.setData('text/orca-session', tabId)
                    event.dataTransfer.effectAllowed = 'move'
                  }}
                  onDragOver={(event) => event.preventDefault()}
                  onDrop={() => reorderTab(tabId)}
                  onDragEnd={() => setDraggedTab(null)}
                  className={
                    tabId === selectedId
                      ? 'flex cursor-grab items-center gap-1 rounded-t border-x border-t border-line bg-panel px-2 py-1 text-[11.5px] text-fg active:cursor-grabbing'
                      : 'flex cursor-grab items-center gap-1 rounded-t px-2 py-1 text-[11.5px] text-muted hover:text-fg active:cursor-grabbing'
                  }
                >
                  <button onClick={() => selectSession(tabId)} className="max-w-[160px] truncate">{title}</button>
                  <button
                    onClick={() => window.open(`/sessions/${tabId}?popout=1`, `orca-session-${tabId}`, 'width=1100,height=800')}
                    className="text-muted hover:text-fg"
                    title={t('sessions.popOutTab')}
                  >
                    <ExternalLink className="h-3 w-3" />
                  </button>
                  <button onClick={() => closeTab(tabId)} className="text-muted hover:text-fg" title={t('sessions.closeTab')}>
                    <X className="h-3 w-3" />
                  </button>
                </div>
              )
            })}
          </div>
        )}
        <div className="min-h-0 flex-1">
          {selectedId ? (
            <SessionDetail id={selectedId} embedded />
          ) : (
            <div className="flex h-full flex-col items-center justify-center gap-2 text-muted">
              <MessageSquare className="h-7 w-7 opacity-30" />
              <p className="text-[13px]">{t('sessions.select')}</p>
            </div>
          )}
        </div>
      </section>
    </div>
  )
}
