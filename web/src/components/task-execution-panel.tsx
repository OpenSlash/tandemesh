import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { GitBranch, MessageSquare, ShieldQuestion } from 'lucide-react'
import { api, type Run } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { Button } from './ui/button'
import { Input } from './ui/input'

export function TaskExecutionPanel({ taskId, runs, executionState }: { taskId: string; runs: Run[]; executionState?: string }) {
  const { t } = useI18n()
  const qc = useQueryClient()
  const [gateQuestion, setGateQuestion] = useState('')
  const [annotationFile, setAnnotationFile] = useState('')
  const [annotationLine, setAnnotationLine] = useState(1)
  const [annotationBody, setAnnotationBody] = useState('')
  const [messageBody, setMessageBody] = useState('')
  const [targetRun, setTargetRun] = useState('')
  const [checkpoint, setCheckpoint] = useState('')
  const [workspaceStatus, setWorkspaceStatus] = useState('in-progress')

  const { data: dispatchesData } = useQuery({
    queryKey: ['task-dispatches', taskId],
    queryFn: () => api.listDispatches(taskId),
    refetchInterval: 4000,
  })
  const { data: gatesData } = useQuery({
    queryKey: ['task-gates', taskId],
    queryFn: () => api.listGates(taskId),
    refetchInterval: 4000,
  })
  const { data: annotationsData } = useQuery({
    queryKey: ['task-annotations', taskId],
    queryFn: () => api.listAnnotations(taskId),
    refetchInterval: 4000,
  })
  const { data: worktreesData } = useQuery({ 
    queryKey: ['task-worktrees', taskId],
    queryFn: () => api.worktrees({ taskId }),
    refetchInterval: 4000,
  })
  const { data: messagesData } = useQuery({
    queryKey: ['task-messages', taskId],
    queryFn: () => api.taskMessages(taskId),
    refetchInterval: 4000,
  })
  const refresh = () => {
    void qc.invalidateQueries({ queryKey: ['task-dispatches', taskId] })
    void qc.invalidateQueries({ queryKey: ['task-gates', taskId] })
    void qc.invalidateQueries({ queryKey: ['task-annotations', taskId] })
    void qc.invalidateQueries({ queryKey: ['task-messages', taskId] })
    void qc.invalidateQueries({ queryKey: ['task-worktrees', taskId] })
    void qc.invalidateQueries({ queryKey: ['task', taskId] })
  }
  const createGate = useMutation({
    mutationFn: () => api.createGate(taskId, { question: gateQuestion }),
    onSuccess: () => { setGateQuestion(''); refresh() },
  })
  const resolveGate = useMutation({
    mutationFn: (id: string) => api.resolveGate(id, 'resolved'),
    onSuccess: refresh,
  })
  const createAnnotation = useMutation({
    mutationFn: () =>
      api.createAnnotation(taskId, {
        runId: targetRun || undefined,
        filePath: annotationFile,
        line: annotationLine,
        body: annotationBody,
      }),
    onSuccess: () => { setAnnotationBody(''); refresh() },
  })
  const sendAnnotations = useMutation({
    mutationFn: () => api.sendAnnotations(taskId, { runId: targetRun || undefined }),
    onSuccess: refresh,
  })
  const saveCheckpoint = useMutation({
    mutationFn: (id: string) => api.setWorktreeCheckpoint(id, { comment: checkpoint, workspaceStatus }),
    onSuccess: refresh,
  })
  const archiveWorktree = useMutation({
    mutationFn: (id: string) => api.archiveWorktree(id),
    onSuccess: refresh,
  })
  const cleanupWorktree = useMutation({
    mutationFn: (id: string) => api.cleanupWorktree(id),
    onSuccess: refresh,
  })
  const sendMessage = useMutation({
    mutationFn: () =>
      api.sendTaskMessage(taskId, {
        to: targetRun || '@all',
        type: 'status',
        body: messageBody,
        subject: messageBody.slice(0, 40),
      }),
    onSuccess: () => { setMessageBody(''); refresh() },
  })

  const dispatches = dispatchesData?.dispatches ?? []
  const gates = gatesData?.gates ?? []
  const annotations = annotationsData?.annotations ?? []
  const messages = messagesData?.messages ?? []
  const worktree = worktreesData?.worktrees?.[0]

  return (
    <div className="space-y-4">
      <div className="rounded-lg border border-line bg-panel/40 p-4">
        <h2 className="flex items-center gap-1.5 text-[11px] font-medium uppercase tracking-wider text-muted">
          <GitBranch className="h-3.5 w-3.5" />
          {t('task.execution')}
          {executionState && <span className="ml-auto normal-case tracking-normal text-accent2">{executionState}</span>}
        </h2>
        <div className="mt-2 space-y-1.5">
          {dispatches.map((d) => (
            <div key={d.id} className="flex items-center justify-between gap-2 rounded border border-line2 bg-panel px-2 py-1 text-[11px]">
              <span className="font-mono text-muted/70">{d.id.slice(-10)}</span>
              <span className="text-muted">#{d.attempt}</span>
              <span className={d.status === 'succeeded' ? 'text-ok' : d.status === 'failed' ? 'text-err' : 'text-muted'}>{d.status}</span>
              <span className="truncate text-muted/60">{d.result}</span>
            </div>
          ))}
          {dispatches.length === 0 && <p className="text-[11px] text-muted/60">{t('task.noDispatches')}</p>}
        </div>
      </div>

      {worktree && (
        <div className="rounded-lg border border-line bg-panel/40 p-4">
          <h2 className="flex items-center gap-1.5 text-[11px] font-medium uppercase tracking-wider text-muted">
            <GitBranch className="h-3.5 w-3.5" />
            {t('task.worktree')}
            <span className="ml-auto normal-case tracking-normal text-accent2">{worktree.status}</span>
          </h2>
          <div className="mt-2 space-y-1 text-[11px]">
            <p className="truncate font-mono text-muted/70">{worktree.branch}</p>
            <p className="truncate font-mono text-muted/50">{worktree.path}</p>
            {worktree.checkpointComment && <p className="text-muted">{worktree.checkpointComment}</p>}
            <div className="mt-2 flex gap-1.5">
              <Input value={checkpoint} onChange={(e) => setCheckpoint(e.target.value)} placeholder={t('task.checkpointPlaceholder')} />
              <select value={workspaceStatus} onChange={(e) => setWorkspaceStatus(e.target.value)} className="rounded border border-line2 bg-panel px-1.5 py-0.5 text-[11px] text-muted">
                <option value="todo">todo</option>
                <option value="in-progress">in-progress</option>
                <option value="in-review">in-review</option>
                <option value="completed">completed</option>
              </select>
              <Button size="sm" variant="ghost" disabled={!checkpoint || saveCheckpoint.isPending} onClick={() => saveCheckpoint.mutate(worktree.id)}>{t('common.save')}</Button>
            </div>
            <div className="flex justify-end gap-1.5">
              <Button size="sm" variant="ghost" disabled={worktree.status === 'archived' || archiveWorktree.isPending} onClick={() => archiveWorktree.mutate(worktree.id)}>{t('task.archiveWorktree')}</Button>
              <Button size="sm" variant="ghost" disabled={worktree.status !== 'archived' || cleanupWorktree.isPending} onClick={() => cleanupWorktree.mutate(worktree.id)}>{t('task.cleanupWorktree')}</Button>
            </div>
          </div>
        </div>
      )}

      <div className="rounded-lg border border-line bg-panel/40 p-4">
        <h2 className="flex items-center gap-1.5 text-[11px] font-medium uppercase tracking-wider text-muted">
          <ShieldQuestion className="h-3.5 w-3.5" />
          {t('task.gates')}
        </h2>
        <div className="mt-2 space-y-1.5">
          {gates.map((g) => (
            <div key={g.id} className="flex items-center justify-between gap-2 text-[11px]">
              <span className={g.status === 'open' ? 'text-warn' : 'text-muted/60'}>{g.question}</span>
              {g.status === 'open' ? (
                <Button size="sm" variant="ghost" onClick={() => resolveGate.mutate(g.id)}>{t('task.resolve')}</Button>
              ) : (
                <span className="text-muted/60">{g.resolution}</span>
              )}
            </div>
          ))}
          <div className="mt-1.5 flex gap-2">
            <Input value={gateQuestion} onChange={(e) => setGateQuestion(e.target.value)} placeholder={t('task.gateQuestion')} />
            <Button size="sm" variant="ghost" disabled={!gateQuestion || createGate.isPending} onClick={() => createGate.mutate()}>
              {t('task.createGate')}
            </Button>
          </div>
        </div>
      </div>

      <div className="rounded-lg border border-line bg-panel/40 p-4">
        <h2 className="flex items-center gap-1.5 text-[11px] font-medium uppercase tracking-wider text-muted">
          <MessageSquare className="h-3.5 w-3.5" />
          {t('task.annotations')}
        </h2>
        <div className="mt-2 space-y-1.5">
          {annotations.map((a) => (
            <div key={a.id} className="flex items-center justify-between gap-2 text-[11px]">
              <span className="truncate font-mono text-muted/70">{a.filePath}:{a.line}</span>
              <span className={a.status === 'open' ? 'text-warn' : 'text-muted/50'}>{a.body}</span>
              {a.status === 'open' && (
                <Button size="sm" variant="ghost" onClick={() => api.resolveAnnotation(a.id).then(refresh)}>{t('task.resolve')}</Button>
              )}
            </div>
          ))}
          {annotations.length === 0 && <p className="text-[11px] text-muted/60">{t('task.noAnnotations')}</p>}
          <div className="mt-1.5 flex gap-1.5">
            <Input value={annotationFile} onChange={(e) => setAnnotationFile(e.target.value)} placeholder="path/file.tsx" />
            <Input type="number" value={annotationLine} onChange={(e) => setAnnotationLine(Number(e.target.value))} className="w-20" />
            <Input value={annotationBody} onChange={(e) => setAnnotationBody(e.target.value)} placeholder={t('task.annotationPlaceholder')} />
            <Button size="sm" variant="ghost" disabled={!annotationFile || !annotationBody} onClick={() => createAnnotation.mutate()}>{t('common.add')}</Button>
          </div>
          <div className="flex justify-end">
            <Button size="sm" variant="ghost" disabled={annotations.filter((a) => a.status === 'open').length === 0 || sendAnnotations.isPending} onClick={() => sendAnnotations.mutate()}>
              {t('task.sendReview')}
            </Button>
          </div>
        </div>
      </div>

      <div className="rounded-lg border border-line bg-panel/40 p-4">
        <div className="flex items-center justify-between">
          <h2 className="flex items-center gap-1.5 text-[11px] font-medium uppercase tracking-wider text-muted">
            <MessageSquare className="h-3.5 w-3.5" />
            {t('task.messages')}
          </h2>
          <select value={targetRun} onChange={(e) => setTargetRun(e.target.value)} className="rounded border border-line2 bg-panel px-1.5 py-0.5 text-[11px] text-muted">
            <option value="">{t('task.recipientAll')}</option>
            {runs.map((run) => (
              <option key={run.id} value={run.id}>{run.agentId || run.id.slice(-8)}</option>
            ))}
          </select>
        </div>
        <div className="mt-2 max-h-40 space-y-1 overflow-auto">
          {messages.map((m) => (
            <div key={m.id} className="rounded border border-line2 bg-panel px-2 py-1 text-[11px]">
              <span className="text-muted/60">{m.type} · {m.sender.slice(-8)} → {m.recipient.slice(-8)}</span>
              <p className="text-muted">{m.subject || m.body}</p>
            </div>
          ))}
        </div>
        <div className="mt-1.5 flex gap-2">
          <Input value={messageBody} onChange={(e) => setMessageBody(e.target.value)} placeholder={t('task.messagePlaceholder')} />
          <Button size="sm" variant="ghost" disabled={!messageBody || sendMessage.isPending} onClick={() => sendMessage.mutate()}>{t('task.sendMessage')}</Button>
        </div>
      </div>
    </div>
  )
}
