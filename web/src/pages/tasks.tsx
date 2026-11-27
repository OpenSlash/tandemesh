import { useI18n, type TFunction } from '../lib/i18n'
import { useEffect, useState } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Bot, Columns3, LayoutList, ListTodo, Pencil, Trash2 } from 'lucide-react'
import { api, type Task, type TaskActivityItem } from '../lib/api'
import { useWorkspace } from '../lib/workspace'
import { cn, compactId } from '../lib/utils'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Dialog, DialogContent, DialogTrigger } from '../components/ui/dialog'
import { statusLabel } from '../components/status'
import { Markdown } from '../components/markdown'

const TABS = [
  { value: '', labelKey: 'common.all' },
  { value: 'open', labelKey: 'status.open' },
  { value: 'in_progress', labelKey: 'status.in_progress' },
  { value: 'done', labelKey: 'status.done' },
]

const COLUMNS = [
  { status: 'open', labelKey: 'status.open' },
  { status: 'in_progress', labelKey: 'status.in_progress' },
  { status: 'done', labelKey: 'status.done' },
]

export default function TasksPage() {
  const { t } = useI18n()
  const qc = useQueryClient()
  const navigate = useNavigate()
  const { workspaceId, workspaces } = useWorkspace()
  const [searchParams] = useSearchParams()
  const scope = searchParams.get('scope') === 'all' ? 'all' : workspaceId
  const allScope = scope === 'all'
  const [status, setStatus] = useState('')
  const [view, setView] = useState<'list' | 'board'>('list')
  const [dragId, setDragId] = useState('')
  const [label, setLabel] = useState('')
  const [agent, setAgent] = useState('')
  const [iteration, setIteration] = useState('')
  const [unassigned, setUnassigned] = useState(false)
  const [provider, setProvider] = useState('')
  const [drawerId, setDrawerId] = useState('')
  const [activeView, setActiveView] = useState('')
  const [saveOpen, setSaveOpen] = useState(false)
  const [viewName, setViewName] = useState('')
  const [renameOpen, setRenameOpen] = useState(false)
  const [renameName, setRenameName] = useState('')
  const [createOpen, setCreateOpen] = useState(false)
  const [newTitle, setNewTitle] = useState('')
  const [newBody, setNewBody] = useState('')
  const [newLabels, setNewLabels] = useState('')
  const [newAgent, setNewAgent] = useState('')
  const [newMember, setNewMember] = useState('')
  const createTask = useMutation({
    mutationFn: () => api.createTask({ title: newTitle, body: newBody, workspaceId, agentId: newAgent || undefined, memberId: newMember || undefined, labels: newLabels.split(',').map((v) => v.trim()).filter(Boolean) }),
    onSuccess: (task) => {
      setCreateOpen(false)
      setNewTitle('')
      setNewBody('')
      setNewLabels('')
      setNewAgent('')
      setNewMember('')
      qc.invalidateQueries({ queryKey: ['tasks'] })
      navigate(`/tasks/${task.id}`)
    },
  })

  const { data: viewsData } = useQuery({
    queryKey: ['views', workspaceId],
    queryFn: () => api.savedViews(workspaceId),
  })
  const views = viewsData?.items ?? []
  const saveView = useMutation({
    mutationFn: () =>
      api.createSavedView(workspaceId, {
        name: viewName,
        filters: { status, label, agent, iteration },
      }),
    onSuccess: (v) => {
      setSaveOpen(false)
      setViewName('')
      setActiveView(v.id)
      qc.invalidateQueries({ queryKey: ['views', workspaceId] })
    },
  })
  const renameView = useMutation({
    mutationFn: () => api.renameSavedView(workspaceId, activeView, renameName),
    onSuccess: () => {
      setRenameOpen(false)
      qc.invalidateQueries({ queryKey: ['views', workspaceId] })
    },
  })
  const deleteView = useMutation({
    mutationFn: () => api.deleteSavedView(workspaceId, activeView),
    onSuccess: () => {
      setActiveView('')
      qc.invalidateQueries({ queryKey: ['views', workspaceId] })
    },
  })

  const applyView = (id: string) => {
    setActiveView(id)
    const view = views.find((v) => v.id === id)
    if (!view) return
    setStatus(view.filters.status ?? '')
    setLabel(view.filters.label ?? '')
    setAgent(view.filters.agent ?? '')
    setIteration(view.filters.iteration ?? '')
  }

  const { data: iterationsData } = useQuery({
    queryKey: ['iterations', workspaceId],
    queryFn: () => api.iterations(workspaceId),
  })
  const iterations = iterationsData?.items ?? []

  const effectiveStatus = view === 'board' ? '' : status
  const { data, isLoading, error } = useQuery({
    queryKey: ['tasks', scope, effectiveStatus, label, agent, iteration, unassigned, provider],
    queryFn: () =>
      api.tasks({
        workspace: scope,
        status: effectiveStatus || undefined,
        label: label || undefined,
        agent: agent || undefined,
        iteration: iteration || undefined,
        unassigned: unassigned || undefined,
        provider: provider || undefined,
      }),
    refetchInterval: 5000,
  })
  const items = data?.items ?? []
  const { data: connectionsData } = useQuery({
    queryKey: ['connections', scope],
    queryFn: () => api.connections(scope === 'all' ? workspaceId : scope),
  })
  const connectedSources = new Set((connectionsData?.items ?? []).map((c) => c.provider))
  const sourceSet = new Set<string>(connectedSources)
  sourceSet.add('local')
  for (const item of items) {
    if (item.provider === 'manual' || item.provider === 'automation') sourceSet.add(item.provider)
  }
  const sources = Array.from(sourceSet)
  const { data: agentsData } = useQuery({
    queryKey: ['agents', scope],
    queryFn: () => api.agents(scope),
  })
  const { data: groupsData } = useQuery({
    queryKey: ['groups', scope],
    queryFn: () => api.groups(scope),
  })
  const agentNames: Record<string, string> = {}
  for (const a of agentsData?.items ?? []) agentNames[a.id] = a.name
  const groupNames: Record<string, string> = {}
  for (const g of groupsData?.items ?? []) groupNames[g.id] = g.name
  const actorName = (id?: string) => (id ? agentNames[id] || groupNames[id] || compactId(id, 8) : '')

  const patch = useMutation({
    mutationFn: ({ id, status }: { id: string; status: string }) => api.patchTask(id, status),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['tasks'] }),
  })

  return (
    <div className="mx-auto max-w-[1180px] px-6 py-5">
      <div className="flex flex-col gap-3">
        <div>
          <div className="flex items-center justify-between">
            <h1 className="text-[18px] font-semibold tracking-tight">{t('page.tasks')}</h1>
            <Button size="sm" variant="primary" onClick={() => setCreateOpen(true)}>{t('tasks.newTask')}</Button>
          </div>
          <p className="mt-0.5 text-[12px] text-muted">{t('tasks.subtitle')}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <select
            value={activeView}
            onChange={(e) => applyView(e.target.value)}
            className="h-7 rounded-md border border-line2 bg-panel px-1.5 text-[11.5px] text-muted"
          >
            <option value="">{t('tasks.views')}</option>
            {views.map((v) => (
              <option key={v.id} value={v.id}>
                {v.name}
              </option>
            ))}
          </select>
          {activeView && (
            <>
              <button
                title={t('tasks.renameView')}
                onClick={() => {
                  const current = views.find((v) => v.id === activeView)
                  setRenameName(current?.name ?? '')
                  setRenameOpen(true)
                }}
                className="text-muted hover:text-fg"
              >
                <Pencil className="h-3.5 w-3.5" />
              </button>
              <button
                title={t('tasks.deleteView')}
                onClick={() => deleteView.mutate()}
                className="text-muted hover:text-err"
              >
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </>
          )}
          <Dialog open={renameOpen} onOpenChange={setRenameOpen}>
            <DialogContent title={t('tasks.renameView')}>
              <form
                className="flex flex-col gap-3"
                onSubmit={(e) => {
                  e.preventDefault()
                  if (renameName) renameView.mutate()
                }}
              >
                <Input value={renameName} onChange={(e) => setRenameName(e.target.value)} />
                <div className="flex justify-end gap-2">
                  <Button type="button" variant="ghost" onClick={() => setRenameOpen(false)}>
                    {t('common.cancel')}
                  </Button>
                  <Button type="submit" variant="primary" disabled={!renameName || renameView.isPending}>
                    {t('common.save')}
                  </Button>
                </div>
              </form>
            </DialogContent>
          </Dialog>
          <Dialog open={saveOpen} onOpenChange={setSaveOpen}>
            <DialogTrigger asChild>
              <Button variant="ghost" size="sm">
                {t('tasks.saveView')}
              </Button>
            </DialogTrigger>
            <DialogContent title={t('tasks.saveView')} description={t('tasks.saveViewHint')}>
              <form
                className="flex flex-col gap-3"
                onSubmit={(e) => {
                  e.preventDefault()
                  if (viewName) saveView.mutate()
                }}
              >
                <Input
                  value={viewName}
                  onChange={(e) => setViewName(e.target.value)}
                  placeholder={t('tasks.viewName')}
                />
                {saveView.error && (
                  <p className="text-[12px] text-err">{String(saveView.error.message)}</p>
                )}
                <div className="flex justify-end gap-2">
                  <Button type="button" variant="ghost" onClick={() => setSaveOpen(false)}>
                    {t('common.cancel')}
                  </Button>
                  <Button type="submit" variant="primary" disabled={!viewName || saveView.isPending}>
                    {t('common.save')}
                  </Button>
                </div>
              </form>
            </DialogContent>
          </Dialog>
          <select
            value={provider}
            onChange={(e) => setProvider(e.target.value)}
            className="h-7 rounded-md border border-line2 bg-panel px-1.5 text-[11.5px] text-muted"
          >
            <option value="">{t('tasks.filterSource')}</option>
            {sources.map((source) => (
              <option key={source} value={source}>
                {source === 'automation' ? t('tasks.sourceAutomation') : source === 'manual' ? t('tasks.sourceManual') : source}
              </option>
            ))}
          </select>
          <select
            value={iteration}
            onChange={(e) => setIteration(e.target.value)}
            className="h-7 rounded-md border border-line2 bg-panel px-1.5 text-[11.5px] text-muted"
          >
            <option value="">{t('nav.iterations')}</option>
            {iterations.map((it) => (
              <option key={it.id} value={it.id}>
                {it.name}
              </option>
            ))}
          </select>
          <input
            value={label}
            onChange={(e) => setLabel(e.target.value)}
            placeholder={t('tasks.filterLabel')}
            className="h-7 w-[120px] rounded-md border border-line2 bg-panel px-2 text-[11.5px] text-fg placeholder:text-muted/60"
          />
          <input
            value={agent}
            onChange={(e) => setAgent(e.target.value)}
            placeholder={t('tasks.filterAgent')}
            className="h-7 w-[110px] rounded-md border border-line2 bg-panel px-2 text-[11.5px] text-fg placeholder:text-muted/60"
          />
          <label className="flex items-center gap-1.5 text-[11px] text-muted">
            <input type="checkbox" checked={unassigned} onChange={(e) => setUnassigned(e.target.checked)} />
            {t('tasks.unassigned')}
          </label>
          {view === 'list' && (
            <div className="flex items-center gap-1 rounded-md border border-line bg-panel/60 p-0.5">
              {TABS.map((tab) => (
                <button
                  key={tab.value}
                  onClick={() => setStatus(tab.value)}
                  className={cn(
                    'rounded px-2 py-1 text-[11.5px]',
                    status === tab.value ? 'bg-panel2 text-fg' : 'text-muted hover:text-fg',
                  )}
                >
                  {t(tab.labelKey)}
                </button>
              ))}
            </div>
          )}
          <div className="flex items-center gap-1 rounded-md border border-line bg-panel/60 p-0.5">
            <button
              onClick={() => setView('list')}
              className={cn(
                'rounded px-1.5 py-1',
                view === 'list' ? 'bg-panel2 text-fg' : 'text-muted hover:text-fg',
              )}
              title={t('tasks.tabList')}
            >
              <LayoutList className="h-3.5 w-3.5" />
            </button>
            <button
              onClick={() => setView('board')}
              className={cn(
                'rounded px-1.5 py-1',
                view === 'board' ? 'bg-panel2 text-fg' : 'text-muted hover:text-fg',
              )}
              title={t('tasks.tabBoard')}
            >
              <Columns3 className="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
      </div>

      {error && <p className="mt-4 text-[12px] text-err">{String(error.message)}</p>}
      {isLoading && <p className="mt-6 text-[12px] text-muted">{t('common.loading')}</p>}

      {view === 'board' ? (
        <div className="mt-4 grid grid-cols-3 gap-3">
          {COLUMNS.map((column) => {
            const cards = items.filter((task) => task.status === column.status)
            return (
              <div
                key={column.status}
                onDragOver={(e) => e.preventDefault()}
                onDrop={() => {
                  if (dragId) {
                    patch.mutate({ id: dragId, status: column.status })
                    setDragId('')
                  }
                }}
                className="min-h-[360px] rounded-lg border border-line bg-panel/20 p-2"
              >
                <div className="flex items-center justify-between px-1 pb-2">
                  <span className="text-[11px] font-medium uppercase tracking-wider text-muted">
                    {t(column.labelKey)}
                  </span>
                  <span className="text-[10.5px] text-muted/70">{cards.length}</span>
                </div>
                <div className="space-y-2">
                  {cards.map((task) => (
                    <TaskCard key={task.id} task={task} onDragStart={() => setDragId(task.id)} onOpen={() => setDrawerId(task.id)} agentNames={agentNames} groupNames={groupNames} />
                  ))}
                  {cards.length === 0 && (
                    <p className="px-1 py-6 text-center text-[11px] text-muted/60">
                      {t('tasks.dropHere')}
                    </p>
                  )}
                </div>
              </div>
            )
          })}
        </div>
      ) : (
        <div className="mt-4 overflow-hidden rounded-lg border border-line">
          <table className="w-full text-left">
            <thead>
              <tr className="bg-panel/80 text-[11px] uppercase tracking-wider text-muted/80">
                <th className="px-3 py-2 font-medium">{t('tasks.colTask')}</th>
                <th className="px-3 py-2 font-medium">{t('tasks.colSource')}</th>
                <th className="px-3 py-2 font-medium">{t('common.labels')}</th>
                <th className="px-3 py-2 font-medium">{t('tasks.colLatestComment')}</th>
                <th className="px-3 py-2 font-medium">{t('tasks.colExecution')}</th>
                <th className="px-3 py-2 font-medium">{t('common.updated')}</th>
                <th className="w-24 px-3 py-2 font-medium">{t('common.status')}</th>
              </tr>
            </thead>
            <tbody>
              {items.map((task) => (
                <tr key={task.id} className="border-t border-line text-[12.5px] hover:bg-panel/40">
                  <td className="px-3 py-2">
                    <Link
                      to={`/tasks/${task.id}`}
                      className="flex items-center gap-2 text-fg hover:text-accent2"
                      onClick={(e) => { e.preventDefault(); setDrawerId(task.id) }}
                    >
                      <ListTodo className="h-3.5 w-3.5 text-muted" />
                      <span className="truncate">{task.title || compactId(task.id, 10)}</span>
                    </Link>
                    <div className="mt-0.5 truncate font-mono text-[10.5px] text-muted">
                      {task.workType} · {task.workId}
                      {allScope && (
                        <span className="ml-2 rounded bg-panel2 px-1 text-[10px] text-muted">
                          {workspaces.find((w) => w.id === task.workspaceId)?.name ??
                            compactId(task.workspaceId, 8)}
                        </span>
                      )}
                    </div>
                  </td>
                  <td className="px-3 py-2 text-muted">{task.origin === 'local' ? 'local' : task.provider}</td>
                  <td className="px-3 py-2">
                    <div className="flex flex-wrap gap-1">
                      {(task.labels ?? []).slice(0, 3).map((label) => (
                        <span
                          key={label}
                          className="rounded bg-accent/10 px-1.5 py-0.5 text-[10px] text-accent2"
                        >
                          {label}
                        </span>
                      ))}
                    </div>
                  </td>
                  <td className="px-3 py-2 text-[11.5px] text-muted">
                    {task.lastComment ? (
                      <span className="block max-w-[220px] truncate" title={task.lastComment}>
                        <span className="text-fg/70">{task.lastCommentBy}: </span>
                        {task.lastComment}
                      </span>
                    ) : (
                      <span className="text-muted/50">—</span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-[11.5px] text-muted">
                    {task.activeAgentId ? (
                      <span className="text-ok">执行中 · {actorName(task.activeAgentId)}</span>
                    ) : task.assigneeAgentId ? (
                      <span>Agent · {actorName(task.assigneeAgentId)}</span>
                    ) : task.assigneeGroupId ? (
                      <span>小组 · {actorName(task.assigneeGroupId)}</span>
                    ) : (
                      <span className="text-muted/50">{t('tasks.unassigned')}</span>
                    )}
                  </td>
                  <td className="px-3 py-2 text-[11.5px] text-muted">
                    {new Date(task.updatedAt).toLocaleString()}
                  </td>
                  <td className="px-3 py-2">
                    <span className="rounded border border-line2 px-1.5 py-0.5 text-[10.5px] text-muted">
                      {statusLabel(t, task.status)}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {!isLoading && items.length === 0 && (
            <div className="px-3 py-12 text-center">
              <p className="text-[12.5px] text-muted">{t('tasks.empty')}</p>
              <p className="mt-1 text-[11.5px] text-muted/70">{t('tasks.emptyHint')}</p>
              <Link to="/automations?type=event" className="mt-3 inline-block text-[12px] text-accent2 hover:underline">
                {t('tasks.configureAutomations')}
              </Link>
            </div>
          )}
        </div>
      )}
      <TaskDrawer id={drawerId} onClose={() => setDrawerId('')} />
    </div>
  )
}

function TaskCard({ task, onDragStart, onOpen, agentNames, groupNames }: { task: Task; onDragStart: () => void; onOpen: () => void; agentNames: Record<string, string>; groupNames: Record<string, string> }) {
  const { t } = useI18n()
  const executor = task.activeAgentId || task.assigneeAgentId || task.assigneeGroupId
  const executorName = executor ? agentNames[executor] || groupNames[executor] || compactId(executor, 8) : ''
  return (
    <Link
      to={`/tasks/${task.id}`}
      draggable
      onDragStart={onDragStart}
      onClick={(e) => { e.preventDefault(); onOpen() }}
      className="block cursor-grab rounded-md border border-line bg-panel/70 p-2.5 hover:border-line2"
    >
      <div className="flex items-start justify-between gap-2">
        <p className="line-clamp-2 text-[12.5px] text-fg">{task.title || compactId(task.id, 10)}</p>
        <span
          title={executorName || t('tasks.unassigned')}
          className={cn(
            'shrink-0 rounded-full border px-1.5 py-0.5 text-[9.5px] font-medium',
            task.activeAgentId
              ? 'border-ok/40 bg-ok/10 text-ok'
              : executor
                ? 'border-accent/40 bg-accent/10 text-accent2'
                : 'border-line2 bg-panel2 text-muted/60',
          )}
        >
          {executorName ? executorName.slice(0, 1) : '—'}
        </span>
      </div>
      <div className="mt-1.5 flex items-center gap-2 text-[10.5px] text-muted">
        <span>{task.origin === 'local' ? 'local' : task.provider}</span>
        <span>·</span>
        <span className="font-mono">{task.workId}</span>
        {task.readyToClose && (
          <span className="rounded bg-ok/15 px-1.5 py-0.5 text-[9.5px] text-ok">
            {t('common.ready')}
          </span>
        )}
        <span className="ml-auto">{t('runs.count', { count: task.runCount ?? 0 })}</span>
      </div>
      {(task.labels ?? []).length > 0 && (
        <div className="mt-1.5 flex flex-wrap gap-1">
          {(task.labels ?? []).slice(0, 2).map((label) => (
            <span key={label} className="rounded bg-accent/10 px-1.5 py-0.5 text-[10px] text-accent2">
              {label}
            </span>
          ))}
        </div>
      )}
    </Link>
  )
}

function TaskDrawer({ id, onClose }: { id: string; onClose: () => void }) {
  const { t } = useI18n()
  const qc = useQueryClient()
  const { data } = useQuery({ queryKey: ['drawer-task', id], queryFn: () => api.task(id), enabled: !!id, refetchInterval: 3000 })
  const { data: agentsData } = useQuery({ queryKey: ['agents', data?.task.workspaceId], queryFn: () => api.agents(data!.task.workspaceId), enabled: !!data?.task.workspaceId })
  const { data: groupsData } = useQuery({ queryKey: ['groups', data?.task.workspaceId], queryFn: () => api.groups(data!.task.workspaceId), enabled: !!data?.task.workspaceId })
  const [agent, setAgent] = useState('')
  const [group, setGroup] = useState('')
  const task = data?.task
  const runs = data?.runs ?? []
  const tokens = runs.reduce((sum, run) => sum + (run.inputTokens ?? 0) + (run.outputTokens ?? 0), 0)
  const latestRun = runs[runs.length - 1]
  const { data: transcript } = useQuery({
    queryKey: ['drawer-transcript', latestRun?.id],
    queryFn: () => api.runTranscript(latestRun!.id),
    enabled: !!latestRun,
    refetchInterval: 3000,
  })
  const { data: platformComments } = useQuery({
    queryKey: ['drawer-platform-comments', id],
    queryFn: () => api.listPlatformComments(id),
    enabled: !!id,
    refetchInterval: 30000,
  })
  const assign = useMutation({
    mutationFn: () => api.assignTask(id, group ? { groupId: group } : { agentId: agent }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['drawer-task', id] }),
  })
  const unassign = useMutation({
    mutationFn: () => api.assignTask(id, {}),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['drawer-task', id] }),
  })
  const start = useMutation({
    mutationFn: async () => {
      if (agent) await api.assignTask(id, { agentId: agent })
      return api.startTaskRun(id, { agentId: agent, repo: '', role: 'lead' })
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: ['drawer-task', id] }),
  })
  const summon = useMutation({
    mutationFn: () => api.summonTask(id),
    onSuccess: (result) => {
      setAgent(result.agent.id)
      qc.invalidateQueries({ queryKey: ['drawer-task', id] })
      qc.invalidateQueries({ queryKey: ['drawer-transcript', result.run.id] })
    },
  })
  useEffect(() => {
    if (!task) return
    setAgent(task.assigneeAgentId ?? '')
    setGroup(task.assigneeGroupId ?? '')
  }, [task?.id, task?.assigneeAgentId, task?.assigneeGroupId])
  const agentName = (agentId?: string) =>
    agentId ? (agentsData?.items ?? []).find((a) => a.id === agentId)?.name ?? agentId : ''
  const selectedGroup = (groupsData?.items ?? []).find((g) => g.id === group)
  const selectedAgent = (agentsData?.items ?? []).find((a) => a.id === agent)
  const selectedName = selectedGroup?.name ?? selectedAgent?.name ?? ''
  if (!id) return null
  return (
    <div className="fixed inset-0 z-50 flex justify-end bg-black/40" onClick={onClose}>
      <aside
        className="h-full w-[760px] max-w-[95vw] overflow-y-auto border-l border-line bg-panel p-5"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="flex items-center gap-2">
              <h2 className="text-[18px] font-semibold">{task?.title ?? t('common.loading')}</h2>
              <span className="rounded border border-line2 bg-panel2 px-1.5 py-0.5 text-[10.5px] text-muted">
                {statusLabel(t, task?.status ?? 'open')}
              </span>
            </div>
            <p className="mt-1 text-[11px] text-muted">
              {task?.provider} - {task?.workId} - 作者 {task?.author || '-'} - 平台负责人 {task?.assignee || '-'}
            </p>
            <p className="mt-1 text-[11px] text-accent2">
              本地指派：{task?.assigneeGroupId
                ? (groupsData?.items ?? []).find((g) => g.id === task.assigneeGroupId)?.name ?? task.assigneeGroupId
                : task?.assigneeAgentId
                  ? (agentsData?.items ?? []).find((a) => a.id === task.assigneeAgentId)?.name ?? task.assigneeAgentId
                  : t('tasks.unassigned')}
            </p>
          </div>
          <Button size="sm" variant="ghost" onClick={onClose}>{t('common.close')}</Button>
        </div>
        <div className="mt-3 flex flex-wrap items-center gap-2">
          <span className={group ? "flex h-7 w-7 items-center justify-center rounded-full bg-accent2/15 text-[12px] font-medium text-accent2" : "flex h-7 w-7 items-center justify-center rounded-full bg-fg/10 text-[12px] font-medium text-fg"}>
            {selectedName ? selectedName.slice(0, 1) : "?"}
          </span>
          <select className="h-7 rounded border border-line2 bg-panel px-2 text-[11px]" value={agent} onChange={(e) => { setAgent(e.target.value); setGroup('') }}>
            <option value="">{t('tasks.agent')}</option>
            {agent && !(agentsData?.items ?? []).some((a) => a.id === agent) && <option value={agent}>{agent}</option>}
            {(agentsData?.items ?? []).map((a) => <option key={a.id} value={a.id}>{a.name}</option>)}
          </select>
          <select className="h-7 rounded border border-line2 bg-panel px-2 text-[11px]" value={group} onChange={(e) => { setGroup(e.target.value); setAgent('') }}>
            <option value="">{t('tasks.group')}</option>
            {group && !(groupsData?.items ?? []).some((g) => g.id === group) && <option value={group}>{group}</option>}
            {(groupsData?.items ?? []).map((g) => <option key={g.id} value={g.id}>{g.name}</option>)}
          </select>
          <Button size="sm" variant="ghost" disabled={!agent && !group} onClick={() => assign.mutate()}>{t('task.assign')}</Button>
          {(task?.assigneeAgentId || task?.assigneeGroupId) && (
            <Button size="sm" variant="ghost" onClick={() => unassign.mutate()}>取消指派</Button>
          )}
          <Button size="sm" variant="primary" disabled={(task?.activeRuns ?? 0) > 0 || summon.isPending || start.isPending} onClick={() => summon.mutate()}>
            {(task?.activeRuns ?? 0) > 0 ? t('tasks.summonRunning') : t('tasks.summonAgent')}
          </Button>
          <Button size="sm" variant="ghost" disabled={(task?.activeRuns ?? 0) > 0 || !agent || start.isPending} onClick={() => start.mutate()}>
            {(task?.activeRuns ?? 0) > 0 || start.isPending ? 'Running...' : t('task.start')}
          </Button>
          <Link to={`/tasks/${id}`} className="ml-auto text-[11px] text-accent2 hover:underline">{t('meta.openLink')}</Link>
        </div>
        <div className="mt-4">
          <Markdown>{task?.body || t('task.noDescription')}</Markdown>
        </div>
        <div className="mt-4 rounded-lg border border-line p-3">
          <div className="flex items-center justify-between text-[11px] text-muted">
            <span>{t('task.agentsOnTask')} · {runs.length}</span>
            <span>tokens: {tokens}</span>
          </div>
          {[...runs].reverse().map((run) => (
            <Link key={run.id} to={`/runs/${run.id}`} className="mt-2 flex items-center gap-2 text-[12px] hover:text-accent2">
              <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-fg/10 text-[10px] font-medium text-fg">
                {(agentName(run.agentId) || "?").slice(0, 1)}
              </span>
              <span>{agentName(run.agentId)}</span>
              <span className="text-[10.5px] text-muted">{run.role}</span>
              <span className="ml-auto text-[10.5px] text-muted">{run.status}</span>
            </Link>
          ))}
        </div>
        <div className="mt-4 rounded-lg border border-line p-3">
          <p className="text-[11px] text-muted">{t('task.activity')}</p>
          {(transcript?.items ?? []).length === 0 && (
            <p className="mt-2 text-[11.5px] text-muted/60">{t('task.activityEmpty')}</p>
          )}
          {(transcript?.items ?? []).map((block, i) => (
            <div key={i} className="mt-2 flex items-start gap-2 text-[11.5px]">
              <span className="shrink-0 font-mono text-[10px] text-muted/70">
                {new Date(block.tsMs).toLocaleTimeString()}
              </span>
              <div className="min-w-0 flex-1">
                {block.kind === 'thought' ? (
                  <div className="text-[11px] italic text-muted">
                  思考：<Markdown className="text-[11px] italic text-muted">{block.text}</Markdown>
                </div>
                ) : block.kind === 'tool' ? (
                  <p className="truncate font-mono text-[10.5px] text-accent2">🔧 {block.text}</p>
                ) : block.kind === 'status' ? (
                  <p className="text-[10.5px] text-muted">{block.text}</p>
                ) : (
                  <Markdown>{block.text}</Markdown>
                )}
              </div>
            </div>
          ))}
        {!!(platformComments?.items ?? []).length && (
          <div className="mt-3 rounded-lg border border-line p-3">
            <p className="text-[11px] text-muted">平台评论</p>
            {(platformComments?.items ?? []).map((comment) => (
              <div key={comment.id} className="mt-2 text-[11.5px]">
                <p className="text-[10.5px] text-muted/70">{comment.author || comment.provider}</p>
                <Markdown>{comment.body}</Markdown>
              </div>
            ))}
          </div>
        )}
        </div>
      </aside>
    </div>
  )
}
function drawerActivityText(item: TaskActivityItem, _t: TFunction): string {
  const p = (item.payload ?? {}) as Record<string, unknown>
  switch (item.type) {
    case 'agent.message':
      return String(p.text ?? '')
    case 'run.finished':
      return String(p.status ?? '')
    case 'run.error':
      return `${String(p.code ?? '')} ${String(p.message ?? '')}`
    case 'tool.call':
      return [p.name, p.args].filter(Boolean).join(' · ')
    case 'task.status':
      return `${String(p.from ?? '')} → ${String(p.to ?? '')}`
    default:
      return String(p.text ?? p.message ?? p.status ?? '')
  }
}

interface TranscriptBlock {
  kind: 'message' | 'thought' | 'tool' | 'status'
  text: string
  ts: string
}

function transcriptBlocks(items: TaskActivityItem[]): TranscriptBlock[] {
  const sorted = [...items].sort((a, b) => Date.parse(a.ts) - Date.parse(b.ts))
  const out: TranscriptBlock[] = []
  for (const item of sorted) {
    const p = (item.payload ?? {}) as Record<string, unknown>
    if (item.type === 'agent.message') {
      const text = String(p.text ?? '')
      if (!text) continue
      const kind = p.role === 'thought' ? 'thought' : 'message'
      const last = out[out.length - 1]
      if (kind === 'message' && last?.kind === 'message' && last.ts.slice(0, 16) === item.ts.slice(0, 16)) {
        last.text += text
      } else {
        out.push({ kind, text, ts: item.ts })
      }
    } else if (item.type === 'tool.call') {
      out.push({ kind: 'tool', text: [p.name, p.args].filter(Boolean).join(' '), ts: item.ts })
    } else if (item.type === 'tool.result') {
      out.push({ kind: 'status', text: `${String(p.toolCallId ?? 'tool')} · ${String(p.status ?? '')}`, ts: item.ts })
    } else if (item.type === 'run.finished') {
      out.push({ kind: 'status', text: `run ${String(p.status ?? '')}`, ts: item.ts })
    }
  }
  return out
}