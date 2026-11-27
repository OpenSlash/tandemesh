import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ArrowLeft, ExternalLink, Play, Bot, Users } from 'lucide-react'
import { api, type TaskActivityItem } from '../lib/api'
import { useI18n, type TFunction } from '../lib/i18n'
import { cn, compactId } from '../lib/utils'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Dialog, DialogContent, DialogTrigger } from '../components/ui/dialog'
import { StatusBadge, statusLabel } from '../components/status'
import { TaskExecutionPanel } from '../components/task-execution-panel'
import { TaskDiffPanel } from '../components/task-diff-panel'
import { TaskWorkflowsPanel } from '../components/task-workflows-panel'
import { TaskSessionsPanel } from '../components/task-sessions-panel'

export default function TaskDetailPage() {
  const { t } = useI18n()
  const { id = '' } = useParams()
  const qc = useQueryClient()
  const [agentId, setAgentId] = useState('claude')
  const [assignGroup, setAssignGroup] = useState('')
  const [repo, setRepo] = useState('')
  const [prompt, setPrompt] = useState('')
  const [role, setRole] = useState('implementer')
  const [noteBody, setNoteBody] = useState('')
  const [commentBody, setCommentBody] = useState('')
  const [platformBody, setPlatformBody] = useState('')
  const [depTarget, setDepTarget] = useState('')
  const [command, setCommand] = useState('')
  const [open, setOpen] = useState(false)
  const [teamOpen, setTeamOpen] = useState(false)
  const [teamRepo, setTeamRepo] = useState('')
  const [teamGroup, setTeamGroup] = useState('')
  const [teamAgents, setTeamAgents] = useState({ lead: 'claude', implementer: 'codex', reviewer: 'claude' })

  const { data, isLoading, error } = useQuery({
    queryKey: ['task', id],
    queryFn: () => api.task(id),
    refetchInterval: 3000,
  })
  const task = data?.task
  const runs = data?.runs ?? []
  const { data: agentsData } = useQuery({
    queryKey: ['agents', task?.workspaceId],
    queryFn: () => api.agents(task!.workspaceId),
    enabled: !!task?.workspaceId,
  })
  const agents = agentsData?.items ?? []
  const { data: me } = useQuery({
    queryKey: ['me'],
    queryFn: api.me,
  })
  const [assignMember, setAssignMember] = useState('')
  const [assignRun, setAssignRun] = useState(false)
  const assignTaskMember = useMutation({
    mutationFn: () => task?.origin === 'local'
      ? api.assignLocalMember(id, assignMember, assignMember, assignRun)
      : api.assignTaskMember(id, assignMember),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['task', id] }),
  })
  const [fieldPriority, setFieldPriority] = useState('')
  const [fieldComponent, setFieldComponent] = useState('')
  const [fieldVersion, setFieldVersion] = useState('')
  const [customDrafts, setCustomDrafts] = useState<Record<string, string>>({})
  const splitValues = (value: string) => value.split(',').map((entry) => entry.trim()).filter(Boolean)
  const componentValues = splitValues(fieldComponent)
  const versionValues = splitValues(fieldVersion)
  const saveCustomFields = useMutation({
    mutationFn: () => api.updateTaskFields(id, { customFields: customDrafts }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['task', id] }),
  })
  const updateTaskFields = useMutation({
    mutationFn: () =>
      api.updateTaskFields(id, {
        priority: fieldPriority || undefined,
        components: fieldComponent ? fieldComponent.split(',').map((value) => value.trim()).filter(Boolean) : undefined,
        fixVersions: fieldVersion ? fieldVersion.split(',').map((value) => value.trim()).filter(Boolean) : undefined,
      }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['task', id] }),
  })
  useEffect(() => {
    if (task?.assigneeAgentId) setAgentId(task.assigneeAgentId)
  }, [task?.assigneeAgentId])
  useEffect(() => {
    const value = (key: string, prefix: string) =>
      task?.fields?.[key]?.[0] ??
      task?.labels?.find((label) => label.toLowerCase().startsWith(prefix))?.slice(prefix.length) ??
      ''
    if (!task) return
    setFieldPriority(value('priority', 'priority:'))
    setFieldComponent((task.fields?.components ?? []).join(', ') || value('components', 'component:'))
    setFieldVersion((task.fields?.fixVersions ?? []).join(', ') || value('fixVersions', 'fixversion:'))
    const known = new Set(['priority', 'components', 'fixVersions'])
    setCustomDrafts(Object.fromEntries(
      Object.entries(task.fields ?? {})
        .filter(([key]) => !known.has(key))
        .map(([key, values]) => [key, values.join(', ')]),
    ))
  }, [task?.id, task?.labels, task?.fields])

  const { data: iterationsData } = useQuery({
    queryKey: ['iterations', task?.workspaceId],
    queryFn: () => api.iterations(task!.workspaceId),
    enabled: !!task?.workspaceId,
  })
  const iterations = iterationsData?.items ?? []
  const { data: groupsData } = useQuery({
    queryKey: ['groups', task?.workspaceId],
    queryFn: () => api.groups(task!.workspaceId),
    enabled: !!task?.workspaceId,
  })
  const groups = groupsData?.items ?? []
  const { data: workspaceTasks } = useQuery({
    queryKey: ['tasks', task?.workspaceId],
    queryFn: () => api.tasks({ workspace: task!.workspaceId }),
    enabled: !!task?.workspaceId,
  })
  const { data: children } = useQuery({
    queryKey: ['task-children', id],
    queryFn: () => api.taskChildren(id),
    enabled: !!task,
  })
  const { data: dependencies } = useQuery({
    queryKey: ['task-dependencies', id],
    queryFn: () => api.taskDependencies(id),
    enabled: !!task,
  })
  const setIteration = useMutation({
    mutationFn: (iterationId: string) => api.setTaskIteration(id, iterationId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['task', id] }),
  })

  useEffect(() => {
    api.markTaskRead(id).catch(() => {})
  }, [id])
  const [activityOffset, setActivityOffset] = useState<number | undefined>(undefined)
  const { data: activity } = useQuery({
    queryKey: ['task-activity', id, activityOffset],
    queryFn: () => api.taskActivity(id, 100, activityOffset),
    refetchInterval: 4000,
  })
  const pipelineFollowups = (activity?.items ?? []).filter((item) => item.type === 'task.pipeline_followup')
  const pipeline = (activity?.items ?? []).find((item) => item.type === 'task.pipeline')?.payload as
    | { status?: string; ref?: string; sha?: string; url?: string }
    | undefined
  const { data: notes } = useQuery({
    queryKey: ['task-notes', id],
    queryFn: () => api.taskNotes(id),
  })
  const { data: comments } = useQuery({
    queryKey: ['task-comments', id],
    queryFn: () => api.taskComments(id),
  })
  const { data: pullRequests } = useQuery({
    queryKey: ['task-pull-requests', id],
    queryFn: () => api.taskPullRequests(id),
  })
  const syncPullRequests = useMutation({
    mutationFn: () => api.syncTaskPullRequests(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['task-pull-requests', id] }),
  })
  const { data: taskCommits } = useQuery({
    queryKey: ['task-commits', id],
    queryFn: () => api.taskCommits(id),
  })
  const syncTaskCommits = useMutation({
    mutationFn: () => api.syncTaskCommits(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['task-commits', id] }),
  })
  const { data: fieldOptions } = useQuery({
    queryKey: ['task-field-options', id],
    queryFn: () => api.taskFieldOptions(id),
    enabled: task?.provider === 'jira',
  })
  const { data: fieldMetadata } = useQuery({
    queryKey: ['task-field-metadata', id],
    queryFn: () => api.taskFieldMetadata(id),
    enabled: task?.provider === 'jira',
  })
  const { data: taskAttachments } = useQuery({
    queryKey: ['task-attachments', id],
    queryFn: () => api.taskAttachments(id),
  })
  const { data: commentThreads } = useQuery({
    queryKey: ['task-comment-threads', id],
    queryFn: () => api.commentThreads(id),
  })
  const addPlatformComment = useMutation({
    mutationFn: ({ threadId, body }: { threadId?: string; body: string }) => api.addPlatformComment(id, body, threadId),
    onSuccess: () => {
      setPlatformBody('')
      qc.invalidateQueries({ queryKey: ['task-comment-threads', id] })
    },
  })
  const resolveThread = useMutation({
    mutationFn: ({ threadId, resolved }: { threadId: string; resolved: boolean }) =>
      api.resolveCommentThread(id, threadId, resolved),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['task-comment-threads', id] }),
  })

  const setStatus = useMutation({
    mutationFn: (status: string) => api.patchTask(id, status),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['task', id] }),
  })
  const addNote = useMutation({
    mutationFn: () => api.addTaskNote(id, noteBody),
    onSuccess: () => {
      setNoteBody('')
      qc.invalidateQueries({ queryKey: ['task-notes', id] })
    },
  })
  const addComment = useMutation({
    mutationFn: () => api.addTaskComment(id, commentBody),
    onSuccess: () => {
      setCommentBody('')
      qc.invalidateQueries({ queryKey: ['task-comments', id] })
      qc.invalidateQueries({ queryKey: ['task-activity', id] })
    },
  })
  const invalidateRelations = () => {
    qc.invalidateQueries({ queryKey: ['task', id] })
    qc.invalidateQueries({ queryKey: ['task-children', id] })
    qc.invalidateQueries({ queryKey: ['task-dependencies', id] })
  }
  const setParent = useMutation({
    mutationFn: (parentId: string) => api.setTaskParent(id, parentId),
    onSuccess: invalidateRelations,
  })
  const addDependency = useMutation({
    mutationFn: (dependsOn: string) => api.addTaskDependency(id, dependsOn),
    onSuccess: () => {
      setDepTarget('')
      invalidateRelations()
    },
  })
  const removeDependency = useMutation({
    mutationFn: (dependsOn: string) => api.removeTaskDependency(id, dependsOn),
    onSuccess: invalidateRelations,
  })
  const sendCommand = useMutation({
    mutationFn: () => api.taskCommand(id, { text: command, repo }),
    onSuccess: () => {
      setCommand('')
      qc.invalidateQueries({ queryKey: ['task', id] })
      qc.invalidateQueries({ queryKey: ['task-notes', id] })
      qc.invalidateQueries({ queryKey: ['task-activity', id] })
    },
  })

  const start = useMutation({
    mutationFn: async () => {
      if (agentId) await api.assignTask(id, { agentId })
      return api.startTaskRun(id, { agentId, repo, role, prompt: prompt || undefined })
    },
    onSuccess: () => {
      setOpen(false)
      setPrompt('')
      qc.invalidateQueries({ queryKey: ['task', id] })
    },
  })
  const assign = useMutation({
    mutationFn: () => api.assignTask(id, assignGroup ? { groupId: assignGroup } : { agentId }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['task', id] }),
  })
  const startTeam = useMutation({
    mutationFn: async () => {
      if (teamGroup) await api.assignTask(id, { groupId: teamGroup })
      return api.startTaskTeam(id, {
        repo: teamRepo,
        groupId: teamGroup || undefined,
        roles: teamGroup
          ? []
          : [
              { role: 'lead', agentId: teamAgents.lead },
              { role: 'implementer', agentId: teamAgents.implementer },
              { role: 'reviewer', agentId: teamAgents.reviewer },
            ],
      })
    },
    onSuccess: () => {
      setTeamOpen(false)
      qc.invalidateQueries({ queryKey: ['task', id] })
    },
  })

  return (
    <div className="mx-auto max-w-[1180px] px-6 py-5">
      <Link to="/tasks" className="inline-flex items-center gap-1 text-[12px] text-muted hover:text-fg">
        <ArrowLeft className="h-3.5 w-3.5" /> {t('page.tasks')}
      </Link>

      {error && <p className="mt-4 text-[12px] text-err">{String(error.message)}</p>}
      {isLoading && <p className="mt-6 text-[12px] text-muted">{t('common.loading')}</p>}

      {task && (
        <>
          <div className="mt-3 flex items-start justify-between gap-4">
            <div className="min-w-0">
              <h1 className="truncate text-[18px] font-semibold tracking-tight">{task.title}</h1>
              <div className="mt-1 flex items-center gap-2 text-[11.5px] text-muted">
                <span>{task.provider}</span>
                <span>·</span>
                <span className="font-mono">{task.workId}</span>
                <span className="rounded bg-accent/10 px-1.5 py-0.5 text-[10.5px] text-accent2">
                  {t('tasks.localAssignShort')}{task.assigneeGroupId
                    ? groups.find((g) => g.id === task.assigneeGroupId)?.name ?? compactId(task.assigneeGroupId, 8)
                    : task.assigneeAgentId
                      ? agents.find((a) => a.id === task.assigneeAgentId)?.name ?? compactId(task.assigneeAgentId, 8)
                      : t('tasks.unassigned')}
                </span>
                <select
                  value={task.status}
                  onChange={(e) => setStatus.mutate(e.target.value)}
                  className="rounded border border-line2 bg-panel px-1.5 py-0.5 text-[10.5px] text-muted"
                >
                  <option value="open">{t('status.open')}</option>
                  <option value="in_progress">{t('status.in_progress')}</option>
                  <option value="done">{t('status.done')}</option>
                </select>
                {task.url && (
                  <a
                    href={task.url}
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex items-center gap-1 text-accent2 hover:underline"
                  >
                    {t('common.source')} <ExternalLink className="h-3 w-3" />
                  </a>
                )}
              </div>
            </div>
            <div className="flex items-center gap-2">
              {task.readyToClose && task.status !== 'done' && (
                <Button variant="ghost" size="sm" onClick={() => setStatus.mutate('done')}>
                  {t('task.markDone')}
                </Button>
              )}
              <Dialog open={teamOpen} onOpenChange={setTeamOpen}>
                <DialogTrigger asChild>
                  <Button variant="ghost" size="sm">
                    <Users className="h-3.5 w-3.5" /> {t('task.startTeam')}
                  </Button>
                </DialogTrigger>
                <DialogContent title={t('task.startTeam')} description={t('task.startTeamHint')}>
                  <form
                    className="flex flex-col gap-3"
                    onSubmit={(e) => {
                      e.preventDefault()
                      startTeam.mutate()
                    }}
                  >
                    <label className="text-[12px] text-muted">
                      {t('common.repository')}
                      <Input
                        className="mt-1"
                        value={teamRepo}
                        onChange={(e) => setTeamRepo(e.target.value)}
                        placeholder="acme/api"
                      />
                    </label>
                    <label className="text-[12px] text-muted">
                      {t('task.groupOptional')}
                      <select
                        className="mt-1 h-8 w-full rounded-md border border-line2 bg-panel px-2 text-[13px]"
                        value={teamGroup}
                        onChange={(e) => setTeamGroup(e.target.value)}
                      >
                        <option value="">{t('task.manualAgents')}</option>
                        {groups.map((group) => (
                          <option key={group.id} value={group.id}>
                            {group.name}
                          </option>
                        ))}
                      </select>
                    </label>
                    {!teamGroup &&
                      (['lead', 'implementer', 'reviewer'] as const).map((r) => (
                        <label key={r} className="text-[12px] text-muted">
                          {t(`roles.${r}`)}
                          <Input
                            className="mt-1"
                            value={teamAgents[r]}
                            onChange={(e) => setTeamAgents({ ...teamAgents, [r]: e.target.value })}
                          />
                        </label>
                      ))}
                    {startTeam.error && (
                      <p className="text-[12px] text-err">{String(startTeam.error.message)}</p>
                    )}
                    <div className="flex justify-end gap-2">
                      <Button type="button" variant="ghost" onClick={() => setTeamOpen(false)}>
                        {t('common.cancel')}
                      </Button>
                      <Button type="submit" variant="primary" disabled={startTeam.isPending}>
                        {t('task.startTeam')}
                      </Button>
                    </div>
                  </form>
                </DialogContent>
              </Dialog>
              <Dialog open={open} onOpenChange={setOpen}>
              <DialogTrigger asChild>
                <Button variant="primary" size="sm">
                  <Play className="h-3.5 w-3.5" /> {t('task.startAgent')}
                </Button>
              </DialogTrigger>
              <DialogContent title={t('task.startAgent')} description={t('task.startAgentHint')}>
                <form
                  className="flex flex-col gap-3"
                  onSubmit={(e) => {
                    e.preventDefault()
                    start.mutate()
                  }}
                >
                  <label className="text-[12px] text-muted">
                    {t('providers.group')}
                    <select
                      className="mt-1 h-8 w-full rounded-md border border-line2 bg-panel px-2 text-[13px]"
                      value={assignGroup}
                      onChange={(e) => setAssignGroup(e.target.value)}
                    >
                      <option value="">—</option>
                      {groups.map((group) => (
                        <option key={group.id} value={group.id}>
                          {group.name}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label className="text-[12px] text-muted">
                    {t('common.agent')}
                    <select
                      className="mt-1 h-8 w-full rounded-md border border-line2 bg-panel px-2 text-[13px]"
                      value={agentId}
                      onChange={(e) => setAgentId(e.target.value)}
                    >
                      {agents.map((agent) => (
                        <option key={agent.id} value={agent.id}>
                          {agent.name} · {agent.backend}
                        </option>
                      ))}
                      {agents.length === 0 && <option value="claude">claude</option>}
                    </select>
                  </label>
                  <label className="text-[12px] text-muted">
                    {t('common.role')}
                    <select
                      className="mt-1 h-8 w-full rounded-md border border-line2 bg-panel px-2 text-[13px]"
                      value={role}
                      onChange={(e) => setRole(e.target.value)}
                    >
                      <option value="lead">{t('roles.lead')}</option>
                      <option value="implementer">{t('roles.implementer')}</option>
                      <option value="reviewer">{t('roles.reviewer')}</option>
                    </select>
                  </label>
                  <label className="text-[12px] text-muted">
                    {t('common.repository')}
                    <Input
                      className="mt-1"
                      value={repo}
                      onChange={(e) => setRepo(e.target.value)}
                      placeholder="acme/api"
                    />
                  </label>
                  <label className="text-[12px] text-muted">
                    {t('task.promptOptional')}
                    <Input
                      className="mt-1"
                      value={prompt}
                      onChange={(e) => setPrompt(e.target.value)}
                      placeholder={t('task.promptPlaceholder')}
                    />
                  </label>
                  {start.error && <p className="text-[12px] text-err">{String(start.error.message)}</p>}
                  <div className="flex justify-end gap-2">
                    <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
                      {t('common.cancel')}
                    </Button>
                    <Button
                      type="button"
                      variant="ghost"
                      disabled={!agentId || assign.isPending}
                      onClick={() => assign.mutate()}
                    >
                      {t('task.assign')}
                    </Button>
                    <Button type="submit" variant="primary" disabled={start.isPending}>
                      {t('task.start')}
                    </Button>
                  </div>
                </form>
              </DialogContent>
              </Dialog>
            </div>
          </div>

          <div className="mt-5 grid grid-cols-[minmax(0,1fr)_280px] gap-5">
            <section className="space-y-4">
              <div className="rounded-lg border border-line bg-panel/40 p-4">
                <h2 className="text-[11px] font-medium uppercase tracking-wider text-muted">
                  {t('common.description')}
                </h2>
                <p className="mt-2 whitespace-pre-wrap text-[12.5px] leading-relaxed text-fg/90">
                  {task.body || t('task.noDescription')}
                </p>
              </div>

              <div className="rounded-lg border border-line bg-panel/40 p-4">
                <h2 className="text-[11px] font-medium uppercase tracking-wider text-muted">
                  {t('task.teamNotes')}
                </h2>
                <p className="mt-1 text-[11px] text-muted/70">{t('task.teamNotesHint')}</p>
                <div className="mt-2 space-y-2">
                  {(notes?.items ?? []).map((note) => (
                    <div key={note.id} className="rounded border border-line2/60 px-2 py-1.5">
                      <div className="flex gap-2 text-[10.5px] text-muted">
                        <span>{note.author}</span>
                        <span>{new Date(note.createdAt).toLocaleString()}</span>
                      </div>
                      <p className="mt-0.5 whitespace-pre-wrap text-[12px] text-fg/90">{note.body}</p>
                    </div>
                  ))}
                  {(notes?.items ?? []).length === 0 && (
                    <p className="text-[11.5px] text-muted">{t('task.noNotes')}</p>
                  )}
                </div>
                <form
                  className="mt-2 flex gap-2"
                  onSubmit={(e) => {
                    e.preventDefault()
                    if (noteBody.trim()) addNote.mutate()
                  }}
                >
                  <Input
                    value={noteBody}
                    onChange={(e) => setNoteBody(e.target.value)}
                    placeholder={t('task.notePlaceholder')}
                  />
                  <Button type="submit" variant="ghost" disabled={!noteBody.trim() || addNote.isPending}>
                    {t('common.add')}
                  </Button>
                </form>
              </div>

              <div className="rounded-lg border border-line bg-panel/40 p-4">
                <div className="flex items-center justify-between gap-2">
                  <h2 className="text-[11px] font-medium uppercase tracking-wider text-muted">
                    {t('task.pullRequests')} <span className="text-muted/60">{pullRequests?.items.length ?? 0}</span>
                  </h2>
                  <Button
                    size="sm"
                    variant="ghost"
                    disabled={syncPullRequests.isPending || (task.provider !== 'github' && task.provider !== 'gitlab')}
                    onClick={() => syncPullRequests.mutate()}
                  >
                    {t('task.refreshPullRequests')}
                  </Button>
                </div>
                <div className="mt-2 space-y-2">
                  {(pullRequests?.items ?? []).map((pr) => (
                    <a
                      key={pr.id}
                      href={pr.url}
                      target="_blank"
                      rel="noreferrer"
                      className="flex items-center gap-2 rounded border border-line2/60 px-2 py-1.5 hover:text-fg"
                    >
                      <span className="font-mono text-[11px] text-muted">#{pr.number}</span>
                      <span className="min-w-0 flex-1 truncate text-[12px]">{pr.title}</span>
                      <span className="rounded bg-panel2 px-1.5 py-0.5 text-[10px] text-muted">{pr.state}</span>
                      <ExternalLink className="h-3 w-3 text-muted" />
                    </a>
                  ))}
                  {(pullRequests?.items ?? []).length === 0 && (
                    <p className="text-[11.5px] text-muted">{t('task.noPullRequests')}</p>
                  )}
                </div>
              </div>

              <div className="rounded-lg border border-line bg-panel/40 p-4">
                <div className="flex items-center justify-between gap-2">
                  <h2 className="text-[11px] font-medium uppercase tracking-wider text-muted">
                    {t('task.commits')} <span className="text-muted/60">{taskCommits?.items.length ?? 0}</span>
                  </h2>
                  <Button
                    size="sm"
                    variant="ghost"
                    disabled={syncTaskCommits.isPending || (task.provider !== 'github' && task.provider !== 'gitlab')}
                    onClick={() => syncTaskCommits.mutate()}
                  >
                    {t('task.refreshCommits')}
                  </Button>
                </div>
                <div className="mt-2 space-y-2">
                  {(taskCommits?.items ?? []).map((commit) => (
                    <a
                      key={commit.id}
                      href={commit.url}
                      target="_blank"
                      rel="noreferrer"
                      className="flex items-center gap-2 rounded border border-line2/60 px-2 py-1.5 hover:text-fg"
                    >
                      <span className="font-mono text-[11px] text-muted">{commit.sha.slice(0, 8)}</span>
                      <span className="min-w-0 flex-1 truncate text-[12px]">{commit.message.split('\n')[0]}</span>
                      <ExternalLink className="h-3 w-3 text-muted" />
                    </a>
                  ))}
                  {(taskCommits?.items ?? []).length === 0 && (
                    <p className="text-[11.5px] text-muted">{t('task.noCommits')}</p>
                  )}
                </div>
              </div>

              <div className="rounded-lg border border-line bg-panel/40 p-4">
                <h2 className="text-[11px] font-medium uppercase tracking-wider text-muted">
                  {t('task.attachments')} <span className="text-muted/60">{taskAttachments?.items.length ?? 0}</span>
                </h2>
                <div className="mt-2 space-y-2">
                  {(taskAttachments?.items ?? []).map((attachment) => (
                    <a
                      key={attachment.id}
                      href={`/api/v1/tasks/${id}/attachments/${attachment.id}/content`}
                      target="_blank"
                      rel="noreferrer"
                      className="flex items-center gap-2 rounded border border-line2/60 px-2 py-1.5 hover:text-fg"
                    >
                      <span className="min-w-0 flex-1 truncate text-[12px]">{attachment.name}</span>
                      <span className="text-[10px] text-muted">{attachment.author}</span>
                      <ExternalLink className="h-3 w-3 text-muted" />
                    </a>
                  ))}
                  {(taskAttachments?.items ?? []).length === 0 && (
                    <p className="text-[11.5px] text-muted">{t('task.noAttachments')}</p>
                  )}
                </div>
              </div>

              <div className="rounded-lg border border-line bg-panel/40 p-4">
                <h2 className="text-[11px] font-medium uppercase tracking-wider text-muted">
                  {t('task.comments')} <span className="text-muted/60">{comments?.items.length ?? 0}</span>
                </h2>
                <div className="mt-2 space-y-2">
                  {(comments?.items ?? []).map((comment) => (
                    <div key={comment.id} className="rounded border border-line2/60 px-2 py-1.5">
                      <div className="flex gap-2 text-[10.5px] text-muted">
                        <span>{comment.author}</span>
                        <span>{new Date(comment.createdAt).toLocaleString()}</span>
                      </div>
                      <p className="mt-0.5 whitespace-pre-wrap text-[12px] text-fg/90">{comment.body}</p>
                    </div>
                  ))}
                  {(comments?.items ?? []).length === 0 && (
                    <p className="text-[11.5px] text-muted">{t('task.noComments')}</p>
                  )}
                </div>
                <form
                  className="mt-2 flex gap-2"
                  onSubmit={(e) => {
                    e.preventDefault()
                    if (commentBody.trim()) addComment.mutate()
                  }}
                >
                  <Input
                    value={commentBody}
                    onChange={(e) => setCommentBody(e.target.value)}
                    placeholder={t('task.commentPlaceholder')}
                  />
                  <Button
                    type="submit"
                    variant="ghost"
                    disabled={!commentBody.trim() || addComment.isPending}
                  >
                    {t('task.comment')}
                  </Button>
                </form>
              </div>

              <div className="rounded-lg border border-line bg-panel/40 p-4">
                <h2 className="text-[11px] font-medium uppercase tracking-wider text-muted">
                  {t('task.platformComments')} <span className="text-muted/60">{commentThreads?.items.length ?? 0}</span>
                </h2>
                <div className="mt-2 space-y-3">
                  {(commentThreads?.items ?? []).map((thread) => (
                    <div key={thread.id} className="rounded border border-line2/60 p-2">
                      <div className="flex items-center gap-2 text-[10.5px] text-muted">
                        <span className="font-mono">{thread.source}</span>
                        <span className="truncate">{thread.externalId}</span>
                        <button
                          className="ml-auto rounded border border-line2 px-1.5 py-0.5 hover:text-fg"
                          onClick={() => resolveThread.mutate({ threadId: thread.id, resolved: !thread.resolved })}
                        >
                          {thread.resolved ? t('task.reopenThread') : t('task.resolveThread')}
                        </button>
                      </div>
                      <div className="mt-1 space-y-1">
                        {(commentThreads?.comments[thread.id] ?? []).map((comment) => (
                          <div key={comment.id} className={comment.direction === 'outbound' ? 'pl-4' : ''}>
                            <div className="flex gap-2 text-[10.5px] text-muted">
                              <span>{comment.author}</span>
                              <span>{new Date(comment.createdAt).toLocaleString()}</span>
                            </div>
                            <p className="mt-0.5 whitespace-pre-wrap text-[12px] text-fg/90">{comment.body}</p>
                          </div>
                        ))}
                      </div>
                      {!thread.resolved && (
                        <form
                          className="mt-2 flex gap-2"
                          onSubmit={(e) => {
                            e.preventDefault()
                            if (platformBody.trim()) addPlatformComment.mutate({ threadId: thread.id, body: platformBody })
                          }}
                        >
                          <Input
                            value={platformBody}
                            onChange={(e) => setPlatformBody(e.target.value)}
                            placeholder={t('task.replyPlatform')}
                          />
                          <Button type="submit" variant="ghost" disabled={!platformBody.trim() || addPlatformComment.isPending}>
                            {t('task.comment')}
                          </Button>
                        </form>
                      )}
                    </div>
                  ))}
                  {(commentThreads?.items ?? []).length === 0 && (
                    <p className="text-[11.5px] text-muted">{t('task.noPlatformComments')}</p>
                  )}
                </div>
              </div>

              <div className="rounded-lg border border-line bg-panel/40 p-4">
                <h2 className="text-[11px] font-medium uppercase tracking-wider text-muted">
                  {t('task.relations')}
                </h2>
                <div className="mt-2 space-y-2 text-[12px]">
                  <div className="flex items-center justify-between gap-3">
                    <span className="text-muted">{t('task.parent')}</span>
                    <select
                      value={task.parentId ?? ''}
                      onChange={(e) => setParent.mutate(e.target.value)}
                      className="max-w-[220px] rounded border border-line2 bg-panel px-1.5 py-0.5 text-[11px] text-muted"
                    >
                      <option value="">{t('common.none')}</option>
                      {(workspaceTasks?.items ?? [])
                        .filter((item) => item.id !== task.id)
                        .map((item) => (
                          <option key={item.id} value={item.id}>
                            {item.title}
                          </option>
                        ))}
                    </select>
                  </div>

                  <div>
                    <p className="text-[11px] text-muted">{t('task.subtasks')}</p>
                    {(children?.items ?? []).map((child) => (
                      <Link
                        key={child.id}
                        to={`/tasks/${child.id}`}
                        className="mt-1 block truncate text-[11.5px] text-accent2 hover:underline"
                      >
                        {child.title}
                      </Link>
                    ))}
                    {(children?.items ?? []).length === 0 && (
                      <p className="text-[11px] text-muted/60">{t('common.none')}</p>
                    )}
                  </div>

                  <div>
                    <p className="text-[11px] text-muted">{t('task.dependsOn')}</p>
                    {(dependencies?.items ?? []).map((dep) => (
                      <div key={dep.id} className="mt-1 flex items-center justify-between gap-2">
                        <Link
                          to={`/tasks/${dep.id}`}
                          className="min-w-0 flex-1 truncate text-[11.5px] text-accent2 hover:underline"
                        >
                          {dep.title}
                        </Link>
                        <button
                          onClick={() => removeDependency.mutate(dep.id)}
                          className="text-[10px] text-muted hover:text-err"
                        >
                          {t('common.remove')}
                        </button>
                      </div>
                    ))}
                    <div className="mt-1.5 flex gap-2">
                      <select
                        value={depTarget}
                        onChange={(e) => setDepTarget(e.target.value)}
                        className="min-w-0 flex-1 rounded border border-line2 bg-panel px-1.5 py-0.5 text-[11px] text-muted"
                      >
                        <option value="">{t('task.addDependency')}</option>
                        {(workspaceTasks?.items ?? [])
                          .filter((item) => item.id !== task.id)
                          .map((item) => (
                            <option key={item.id} value={item.id}>
                              {item.title}
                            </option>
                          ))}
                      </select>
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={!depTarget || addDependency.isPending}
                        onClick={() => addDependency.mutate(depTarget)}
                      >
                        {t('common.add')}
                      </Button>
                    </div>
                  </div>
                </div>
              </div>

              <TaskDiffPanel taskId={id} runs={runs} />

              <TaskExecutionPanel taskId={id} runs={runs} executionState={task.executionState} />

              <TaskWorkflowsPanel taskId={id} workspaceId={task.workspaceId} />

              <TaskSessionsPanel taskId={id} workspaceId={task.workspaceId} />

              <div className="rounded-lg border border-line bg-panel/40 p-4">
                <h2 className="text-[11px] font-medium uppercase tracking-wider text-muted">
                  {t('task.command')}
                </h2>
                <p className="mt-1 font-mono text-[11px] text-muted/70">
                  /orca team · /orca run codex · /orca note &lt;text&gt;
                </p>
                <form
                  className="mt-2 flex gap-2"
                  onSubmit={(e) => {
                    e.preventDefault()
                    if (command.trim()) sendCommand.mutate()
                  }}
                >
                  <Input
                    value={command}
                    onChange={(e) => setCommand(e.target.value)}
                    placeholder="/orca team"
                  />
                  <Button type="submit" variant="primary" disabled={!command.trim() || sendCommand.isPending}>
                    {t('task.send')}
                  </Button>
                </form>
                {sendCommand.error && (
                  <p className="mt-1 text-[11.5px] text-err">{String(sendCommand.error.message)}</p>
                )}
              </div>

              <div className="overflow-hidden rounded-lg border border-line">
                <header className="flex items-center justify-between border-b border-line bg-panel/60 px-3 py-2">
                  <span className="text-[12px] font-medium">{t('task.agentsOnTask')}</span>
                  <span className="text-[11px] text-muted">{t('runs.count', { count: runs.length })}</span>
                </header>
                {runs.length === 0 && (
                  <p className="px-3 py-8 text-center text-[12px] text-muted">{t('task.noRuns')}</p>
                )}
                {runs.map((run) => (
                  <Link
                    key={run.id}
                    to={`/runs/${run.id}`}
                    className="flex items-center justify-between gap-3 border-b border-line/60 px-3 py-2 last:border-0 hover:bg-panel/40"
                  >
                    <div className="flex min-w-0 items-center gap-2">
                      <Bot className="h-3.5 w-3.5 text-muted" />
                      <span className="text-[12.5px]">{run.agentId}</span>
                      {run.role && (
                        <span className="rounded bg-panel2 px-1.5 py-0.5 text-[10px] text-muted">
                          {run.role}
                        </span>
                      )}
                      <span className="font-mono text-[10.5px] text-muted">{compactId(run.id, 8)}</span>
                      {run.variantLabel && (
                        <span className="rounded bg-accent/10 px-1.5 py-0.5 text-[10px] text-accent2">
                          {run.variantLabel}
                        </span>
                      )}
                    </div>
                    <div className="flex items-center gap-2">
                      <span className="text-[10.5px] text-muted">
                        {new Date(run.createdAt).toLocaleTimeString()}
                      </span>
                      <StatusBadge status={run.status} />
                    </div>
                  </Link>
                ))}
              </div>

              {pipeline && (
                <div className="rounded-lg border border-line bg-panel/40 p-3">
                  <div className="flex items-center gap-2">
                    <span className="text-[11px] font-medium uppercase tracking-wider text-muted">{t('task.pipeline')}</span>
                    <span className="rounded bg-panel2 px-1.5 py-0.5 text-[10px]">{pipeline.status || '—'}</span>
                    {pipeline.ref && <span className="font-mono text-[10.5px] text-muted">{pipeline.ref}</span>}
                    {pipeline.url && (
                      <a href={pipeline.url} target="_blank" rel="noreferrer" className="ml-auto text-muted hover:text-fg">
                        <ExternalLink className="h-3.5 w-3.5" />
                      </a>
                    )}
                  </div>
                  {pipeline.sha && <p className="mt-1 font-mono text-[10.5px] text-muted">{pipeline.sha.slice(0, 8)}</p>}
                  {pipelineFollowups.length > 0 && (
                    <div className="mt-1 flex flex-wrap items-center gap-2 text-[10.5px] text-muted">
                      <span>{t('task.pipelineRepairs', { count: pipelineFollowups.length })}</span>
                      {pipelineFollowups.slice(0, 3).map((item, index) => {
                        const runId = typeof item.payload?.runId === 'string' ? item.payload.runId : ''
                        return runId ? (
                          <Link key={`${runId}-${index}`} to={`/runs/${runId}`} className="text-accent2 hover:underline">
                            {compactId(runId, 8)}
                          </Link>
                        ) : null
                      })}
                    </div>
                  )}
                </div>
              )}

              <div className="overflow-hidden rounded-lg border border-line">
                <header className="flex items-center justify-between border-b border-line bg-panel/60 px-3 py-2">
                  <span className="text-[12px] font-medium">{t('task.activity')}</span>
                  <div className="flex items-center gap-2">
                    {activity?.hasMore && (
                      <button
                        onClick={() => setActivityOffset(Math.max(0, (activity?.offset ?? 0) - 100))}
                        className="text-[10.5px] text-accent2 hover:underline"
                      >
                        {t('task.loadOlder')}
                      </button>
                    )}
                    <span className="text-[11px] text-muted">
                      {t('common.events', { count: activity?.total ?? 0 })}
                    </span>
                  </div>
                </header>
                {(activity?.items ?? []).length === 0 && (
                  <p className="px-3 py-8 text-center text-[12px] text-muted">{t('task.activityEmpty')}</p>
                )}
                {(activity?.items ?? []).map((item, index) => (
                  <div
                    key={`${item.source}-${item.type}-${item.ts}-${index}`}
                    className="flex gap-3 border-b border-line/60 px-3 py-2 last:border-0"
                  >
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-baseline gap-2">
                        <span className="font-mono text-[11px] text-muted">{item.type}</span>
                        {item.actor && <span className="text-[10.5px] text-fg/70">{item.actor}</span>}
                        {item.agentId && <span className="text-[10.5px] text-muted/80">{item.agentId}</span>}
                        <span className="text-[10px] text-muted/60">
                          {new Date(item.ts).toLocaleTimeString()}
                        </span>
                      </div>
                      {activityText(item, t) && (
                        <p className="mt-0.5 break-words text-[12px] text-fg/90">{activityText(item, t)}</p>
                      )}
                    </div>
                  </div>
                ))}
              </div>
            </section>

            <aside className="space-y-4">
              <div className="rounded-lg border border-line bg-panel/40 p-3">
                <h2 className="text-[11px] font-medium uppercase tracking-wider text-muted">
                  {t('common.source')}
                </h2>
                <dl className="mt-2 space-y-2 text-[12px]">
                  <Meta label={t('meta.provider')} value={task.provider} />
                  <Meta label={t('meta.instance')} value={task.instance} />
                  <Meta label={t('meta.type')} value={task.workType} />
                  <Meta label={t('meta.id')} value={task.workId} mono />
                  <Meta label={t('meta.author')} value={task.author || '—'} />
                  <Meta label={t('meta.assignee')} value={task.assignee || '—'} />
                  {task.assigneeMemberId && <Meta label={t('meta.member')} value={task.assigneeMemberId} />}
                  <div className="flex items-center gap-1.5">
                    <select
                      className="h-7 min-w-0 flex-1 rounded-md border border-line2 bg-panel px-1 text-[11px]"
                      value={assignMember}
                      onChange={(e) => setAssignMember(e.target.value)}
                    >
                      <option value="">{t('task.assignMember')}</option>
                      {me?.name && <option value={me.name}>{me.name}</option>}
                    </select>
                    {task.origin === 'local' && (
                      <label className="flex items-center gap-1 text-[10.5px] text-muted">
                        <input type="checkbox" checked={assignRun} onChange={(e) => setAssignRun(e.target.checked)} />
                        {t('task.assignAndStart')}
                      </label>
                    )}
                    <Button
                      size="sm"
                      variant="ghost"
                      disabled={!assignMember || assignTaskMember.isPending}
                      onClick={() => assignTaskMember.mutate()}
                    >
                      {t('task.assign')}
                    </Button>
                  </div>
                  {assignTaskMember.error && <p className="text-[10.5px] text-err">{String(assignTaskMember.error.message)}</p>}
                  {task.provider === 'jira' && (
                    <div className="flex flex-wrap items-center gap-1.5">
                      <Input list="jira-priorities" className="h-7 w-24" value={fieldPriority} onChange={(e) => setFieldPriority(e.target.value)} placeholder={t('task.priority')} />
                      <Input list="jira-components" className="h-7 w-28" value={fieldComponent} onChange={(e) => setFieldComponent(e.target.value)} placeholder={t('task.component')} />
                      <Input list="jira-versions" className="h-7 w-28" value={fieldVersion} onChange={(e) => setFieldVersion(e.target.value)} placeholder={t('task.fixVersion')} />
                      <datalist id="jira-priorities">
                        {(fieldOptions?.options.priority ?? []).map((option) => <option key={option} value={option} />)}
                      </datalist>
                      <datalist id="jira-components">
                        {(fieldOptions?.options.components ?? []).map((option) => <option key={option} value={option} />)}
                      </datalist>
                      <datalist id="jira-versions">
                        {(fieldOptions?.options.fixVersions ?? []).map((option) => <option key={option} value={option} />)}
                      </datalist>
                      <div className="flex flex-wrap items-center gap-1">
                        {componentValues.map((value) => (
                          <span key={value} className="flex items-center gap-1 rounded bg-panel2 px-1.5 py-0.5 text-[10px]">
                            {value}
                            <button onClick={() => setFieldComponent(componentValues.filter((entry) => entry !== value).join(', '))}>×</button>
                          </span>
                        ))}
                        <select
                          className="h-7 rounded-md border border-line2 bg-panel px-1 text-[10px]"
                          value=""
                          onChange={(e) => {
                            if (!e.target.value || componentValues.includes(e.target.value)) return
                            setFieldComponent([...componentValues, e.target.value].join(', '))
                          }}
                        >
                          <option value="">+ component</option>
                          {(fieldOptions?.options.components ?? []).map((option) => <option key={option} value={option}>{option}</option>)}
                        </select>
                      </div>
                      <div className="flex flex-wrap items-center gap-1">
                        {versionValues.map((value) => (
                          <span key={value} className="flex items-center gap-1 rounded bg-panel2 px-1.5 py-0.5 text-[10px]">
                            {value}
                            <button onClick={() => setFieldVersion(versionValues.filter((entry) => entry !== value).join(', '))}>×</button>
                          </span>
                        ))}
                        <select
                          className="h-7 rounded-md border border-line2 bg-panel px-1 text-[10px]"
                          value=""
                          onChange={(e) => {
                            if (!e.target.value || versionValues.includes(e.target.value)) return
                            setFieldVersion([...versionValues, e.target.value].join(', '))
                          }}
                        >
                          <option value="">+ version</option>
                          {(fieldOptions?.options.fixVersions ?? []).map((option) => <option key={option} value={option}>{option}</option>)}
                        </select>
                      </div>
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={updateTaskFields.isPending || (!fieldPriority && !fieldComponent && !fieldVersion)}
                        onClick={() => updateTaskFields.mutate()}
                      >
                        {t('task.updateFields')}
                      </Button>
                    </div>
                  )}
                  {task.provider === 'jira' && (() => {
                    const projected = Object.entries(task.fields ?? {}).flatMap(([key, values]) => values.map((value) => `${key}:${value}`))
                    const items = projected.length > 0
                      ? projected
                      : task.labels.filter((label) => /^(priority|component|fixversion):/i.test(label))
                    return items.length > 0 ? (
                      <div className="flex flex-wrap gap-1.5">
                        {items.map((label) => (
                          <span key={label} className="rounded bg-panel2 px-1.5 py-0.5 text-[10px] text-muted">{label}</span>
                        ))}
                      </div>
                    ) : null
                  })()}
                  {task.provider === 'jira' && Object.keys(customDrafts).length > 0 && (
                    <div className="space-y-1">
                      {Object.entries(customDrafts).map(([name, value]) => {
                        const meta = fieldMetadata?.fields[name]
                        return (
                          <label key={name} className="flex items-center gap-1.5 text-[10.5px] text-muted">
                            <span className="w-24 truncate">{name}</span>
                            {meta?.allowed && meta.allowed.length > 0 ? (
                              <select
                                className="h-7 flex-1 rounded-md border border-line2 bg-panel px-1 text-[11px]"
                                value={value}
                                onChange={(e) => setCustomDrafts((prev) => ({ ...prev, [name]: e.target.value }))}
                              >
                                <option value="">—</option>
                                {meta.allowed.map((option) => <option key={option} value={option}>{option}</option>)}
                              </select>
                            ) : (
                              <Input
                                className="h-7 flex-1"
                                type={meta?.type === 'number' ? 'number' : 'text'}
                                value={value}
                                onChange={(e) => setCustomDrafts((prev) => ({ ...prev, [name]: e.target.value }))}
                              />
                            )}
                          </label>
                        )
                      })}
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={saveCustomFields.isPending}
                        onClick={() => saveCustomFields.mutate()}
                      >
                        {t('task.updateCustomFields')}
                      </Button>
                    </div>
                  )}
                  <Meta
                    label={t('meta.revision')}
                    value={task.revision ? compactId(task.revision, 12) : '—'}
                    mono
                  />
                  <Meta
                    label={t('meta.runs')}
                    value={t('tasks.runsSummary', {
                      succeeded: task.succeededRuns,
                      active: task.activeRuns,
                      failed: task.failedRuns,
                    })}
                  />
                  <div className="flex items-center justify-between gap-3">
                    <dt className="text-muted">{t('meta.iteration')}</dt>
                    <dd>
                      <select
                        value={task.iterationId ?? ''}
                        onChange={(e) => setIteration.mutate(e.target.value)}
                        className="max-w-[140px] rounded border border-line2 bg-panel px-1.5 py-0.5 text-[11px] text-muted"
                      >
                        <option value="">{t('common.none')}</option>
                        {iterations.map((it) => (
                          <option key={it.id} value={it.id}>
                            {it.name}
                          </option>
                        ))}
                      </select>
                    </dd>
                  </div>
                  <Meta label={t('meta.synced')} value={new Date(task.syncedAt).toLocaleString()} />
                </dl>
              </div>

              <div className="rounded-lg border border-line bg-panel/40 p-3">
                <h2 className="text-[11px] font-medium uppercase tracking-wider text-muted">
                  {t('common.labels')}
                </h2>
                <div className="mt-2 flex flex-wrap gap-1">
                  {(task.labels ?? []).length === 0 && (
                    <span className="text-[11.5px] text-muted">{t('common.none')}</span>
                  )}
                  {(task.labels ?? []).map((label) => (
                    <span
                      key={label}
                      className={cn('rounded px-1.5 py-0.5 text-[10.5px]', 'bg-accent/10 text-accent2')}
                    >
                      {label}
                    </span>
                  ))}
                </div>
              </div>
            </aside>
          </div>
        </>
      )}
    </div>
  )
}

function activityText(item: TaskActivityItem, t: TFunction): string {
  const p = item.payload ?? {}
  switch (item.type) {
    case 'task.status':
      return `${String(p.from ?? '')} → ${String(p.to ?? '')}`
    case 'task.inbox':
      return `${String(p.from ?? '')} → ${String(p.to ?? '')}`
    case 'task.note':
      return String(p.body ?? '')
    case 'task.dispatch':
      return p.role
        ? t('task.actDispatchedRole', { agent: String(p.agent ?? 'agent'), role: String(p.role) })
        : t('task.actDispatched', { agent: String(p.agent ?? 'agent') })
    case 'task.imported':
      return t('task.actImported', { provider: String(p.provider ?? 'source') })
    case 'agent.message':
      return String(p.text ?? '')
    case 'tool.call':
      return [p.name, p.args].filter(Boolean).join(' · ')
    case 'tool.result':
      return statusLabel(t, String(p.status ?? 'done'))
    case 'approval.requested':
      return t('task.actNeedsApproval', { tool: String(p.tool ?? 'tool') })
    case 'run.finished':
      return statusLabel(t, String(p.status ?? ''))
    case 'run.error':
      return `${String(p.code ?? '')} ${String(p.message ?? '')}`
    case 'run.created':
      return t('task.actRunStarted')
    case 'workspace.ready':
      return t('task.actWorktree', { branch: String(p.branch ?? '') })
    case 'skills.loaded':
      return t('task.actSkills', { skills: (p.skills as string[] | undefined)?.join(', ') ?? '' })
    default:
      return ''
  }
}

function Meta({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex justify-between gap-3">
      <dt className="text-muted">{label}</dt>
      <dd className={cn('truncate text-right text-fg/90', mono && 'font-mono text-[11px]')}>{value}</dd>
    </div>
  )
}
