import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Check, Play, Plus, RotateCcw, Workflow as WorkflowIcon, X } from 'lucide-react'
import { api, type Workflow, type WorkflowNode } from '../lib/api'
import { useWorkspace } from '../lib/workspace'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Dialog, DialogContent } from '../components/ui/dialog'
import { cn } from '../lib/utils'

const DEFAULT_NODES = `[
  {"id":"review","kind":"gate","name":"Review","next":"run"},
  {"id":"run","kind":"script","name":"Run","command":"echo workflow ok"}
]`

export default function WorkflowsPage() {
  const { workspaceId } = useWorkspace()
  const qc = useQueryClient()
  const [createOpen, setCreateOpen] = useState(false)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [nodesText, setNodesText] = useState(DEFAULT_NODES)
  const [runFor, setRunFor] = useState<Workflow | null>(null)
  const [taskId, setTaskId] = useState('')
  const [selected, setSelected] = useState(() => new URLSearchParams(window.location.search).get('run') ?? '')
  const [message, setMessage] = useState('')

  const { data: workflowsData } = useQuery({ queryKey: ['workflows', workspaceId], queryFn: () => api.workflows(workspaceId) })
  const { data: runsData } = useQuery({ queryKey: ['workflow-runs', workspaceId], queryFn: () => api.workflowRuns(workspaceId), refetchInterval: 3000 })
  const { data: tasksData } = useQuery({ queryKey: ['tasks', workspaceId, 'workflow'], queryFn: () => api.tasks({ workspace: workspaceId }) })
  const { data: run } = useQuery({
    queryKey: ['workflow-run', selected],
    queryFn: () => api.workflowRun(selected),
    enabled: Boolean(selected),
    refetchInterval: 2000,
  })
  const workflows = workflowsData?.items ?? []
  const runs = runsData?.items ?? []
  const tasks = tasksData?.items ?? []

  const invalidate = () => {
    void qc.invalidateQueries({ queryKey: ['workflows', workspaceId] })
    void qc.invalidateQueries({ queryKey: ['workflow-runs', workspaceId] })
  }
  const create = useMutation({
    mutationFn: () => {
      const nodes = JSON.parse(nodesText) as WorkflowNode[]
      return api.createWorkflow({ workspaceId, name, description, nodes })
    },
    onSuccess: () => { invalidate(); setCreateOpen(false); setName(''); setDescription('') },
    onError: (err) => setMessage(String(err)),
  })
  const start = useMutation({
    mutationFn: () => api.startWorkflow(runFor!.id, { taskId: taskId || undefined }),
    onSuccess: (result) => { invalidate(); setSelected(result.id); setRunFor(null); setTaskId('') },
    onError: (err) => setMessage(String(err)),
  })
  const retry = useMutation({
    mutationFn: (nodeId: string) => api.retryWorkflowRun(selected, nodeId),
    onSuccess: () => { void qc.invalidateQueries({ queryKey: ['workflow-run', selected] }); invalidate() },
    onError: (err) => setMessage(String(err)),
  })
  const cancel = useMutation({
    mutationFn: () => api.cancelWorkflowRun(selected),
    onSuccess: () => { void qc.invalidateQueries({ queryKey: ['workflow-run', selected] }); invalidate() },
    onError: (err) => setMessage(String(err)),
  })
  const gate = useMutation({
    mutationFn: (body: { nodeId: string; action: 'approve' | 'reject' | 'retry'; target?: string }) => api.gateWorkflowRun(selected, body),
    onSuccess: () => { void qc.invalidateQueries({ queryKey: ['workflow-run', selected] }); invalidate() },
    onError: (err) => setMessage(String(err)),
  })

  return (
    <div className="mx-auto max-w-[1180px] px-6 py-5">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-[18px] font-semibold tracking-tight">Workflows</h1>
          <p className="mt-0.5 text-[12px] text-muted">Script, agent and approval steps with replayable state.</p>
        </div>
        <Button size="sm" variant="primary" onClick={() => setCreateOpen(true)}><Plus className="h-3.5 w-3.5" /> New workflow</Button>
      </div>

      {message && <p className="mt-3 text-[11.5px] text-err">{message}</p>}

      <div className="mt-5 grid grid-cols-1 gap-4 lg:grid-cols-[360px_1fr]">
        <div className="space-y-3">
          <section className="rounded-lg border border-line bg-panel/40 p-3">
            <h2 className="text-[12px] uppercase tracking-wider text-muted">Definitions</h2>
            <div className="mt-2 space-y-2">
              {workflows.map((wf) => (
                <div key={wf.id} className="rounded border border-line2 p-2">
                  <div className="flex items-center gap-2">
                    <WorkflowIcon className="h-3.5 w-3.5 text-muted" />
                    <span className="text-[12px] font-medium">{wf.name}</span>
                    <Button className="ml-auto" size="sm" variant="secondary" onClick={() => setRunFor(wf)}>Run</Button>
                  </div>
                  <p className="mt-1 text-[10.5px] text-muted">{wf.nodes.map((n) => n.name || n.id).join(' -> ')}</p>
                </div>
              ))}
              {workflows.length === 0 && <p className="text-[11px] text-muted">No workflows yet.</p>}
            </div>
          </section>

          <section className="rounded-lg border border-line bg-panel/40 p-3">
            <h2 className="text-[12px] uppercase tracking-wider text-muted">Runs</h2>
            <div className="mt-2 space-y-1">
              {runs.map((item) => (
                <button key={item.id} onClick={() => setSelected(item.id)} className={cn('w-full rounded border px-2 py-1 text-left text-[11px]', selected === item.id ? 'border-accent2/60 bg-panel2' : 'border-line2 hover:bg-panel2/50')}>
                  <span className="font-mono">{item.id.slice(0, 14)}</span>
                  <span className="ml-2 text-muted">{item.status}</span>
                </button>
              ))}
              {runs.length === 0 && <p className="text-[11px] text-muted">No runs yet.</p>}
            </div>
          </section>
        </div>

        <section className="min-h-[300px] rounded-lg border border-line bg-panel/20 p-3">
          {!run ? (
            <p className="text-[12px] text-muted">Select a run to inspect nodes.</p>
          ) : (
            <>
              <div className="flex items-center gap-2">
                <span className="text-[13px] font-medium">{run.id}</span>
                <span className={cn('rounded px-1.5 py-0.5 text-[10px]', run.status === 'succeeded' ? 'bg-ok/10 text-ok' : run.status === 'failed' ? 'bg-err/10 text-err' : 'bg-warn/10 text-warn')}>{run.status}</span>
                {run.taskId && <span className="font-mono text-[10px] text-muted">{run.taskId}</span>}
                {(run.status === 'running' || run.status === 'waiting') && (
                  <Button className="ml-auto" size="sm" variant="secondary" onClick={() => cancel.mutate()}><X className="h-3 w-3" /> Cancel</Button>
                )}
              </div>
              <div className="mt-3 space-y-2">
                {run.nodes.map((node) => (
                  <div key={node.nodeId} className="rounded border border-line2 p-2">
                    <div className="flex items-center gap-2">
                      <span className="text-[11.5px] font-medium">{node.name || node.nodeId}</span>
                      <span className="text-[10px] uppercase tracking-wide text-muted">{node.kind}</span>
                      <span className={cn('ml-auto text-[10px]', node.status === 'succeeded' ? 'text-ok' : node.status === 'failed' ? 'text-err' : node.status === 'waiting' ? 'text-warn' : 'text-muted')}>{node.status}</span>
                    </div>
                    {node.output && <pre className="mt-1 max-h-32 overflow-auto whitespace-pre-wrap rounded bg-black/20 p-1.5 text-[10px] text-muted">{node.output}</pre>}
                    {node.error && <p className="mt-1 text-[10.5px] text-err">{node.error}</p>}
                    {(node.artifacts ?? []).map((artifact) => (
                      <details key={artifact.path} className="mt-1 rounded border border-line2/60 px-1.5 py-1 text-[10px]">
                        <summary className="cursor-pointer text-muted">{artifact.name} · {artifact.size} bytes</summary>
                        <pre className="mt-1 max-h-40 overflow-auto whitespace-pre-wrap text-[10px] text-muted">{artifact.content}</pre>
                      </details>
                    ))}
                    {node.kind === 'agent' && node.runId && <p className="mt-1 font-mono text-[10px] text-muted">run {node.runId}</p>}
                    {node.status === 'failed' && (
                      <div className="mt-1 flex gap-1">
                        <Button size="sm" variant="secondary" onClick={() => retry.mutate(node.nodeId)}><RotateCcw className="h-3 w-3" /> Retry</Button>
                      </div>
                    )}
                    {node.status === 'waiting' && (
                      <div className="mt-1 flex gap-1">
                        <Button size="sm" variant="primary" onClick={() => gate.mutate({ nodeId: node.nodeId, action: 'approve' })}><Check className="h-3 w-3" /> Approve</Button>
                        <Button size="sm" variant="secondary" onClick={() => gate.mutate({ nodeId: node.nodeId, action: 'retry' })}><RotateCcw className="h-3 w-3" /> Retry</Button>
                      </div>
                    )}
                  </div>
                ))}
              </div>
              {run.error && <p className="mt-2 text-[11px] text-err">{run.error}</p>}
            </>
          )}
        </section>
      </div>

      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent title="New workflow" description="Nodes are a linear list connected by next.">
          <form className="flex flex-col gap-3" onSubmit={(e) => { e.preventDefault(); create.mutate() }}>
            <Input autoFocus placeholder="Name" value={name} onChange={(e) => setName(e.target.value)} />
            <Input placeholder="Description" value={description} onChange={(e) => setDescription(e.target.value)} />
            <textarea className="min-h-48 rounded-md border border-line2 bg-panel p-2 font-mono text-[11px]" value={nodesText} onChange={(e) => setNodesText(e.target.value)} />
            <div className="flex justify-end"><Button type="submit" variant="primary" disabled={!name || create.isPending}>Create</Button></div>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={!!runFor} onOpenChange={(open) => !open && setRunFor(null)}>
        <DialogContent title={`Run ${runFor?.name ?? ''}`} description="Agent nodes need a task; script/gate workflows can run without one.">
          <form className="flex flex-col gap-3" onSubmit={(e) => { e.preventDefault(); start.mutate() }}>
            <label className="text-[11px] text-muted">Task (optional)
              <select className="mt-1 h-8 w-full rounded-md border border-line2 bg-panel px-2 text-[12px]" value={taskId} onChange={(e) => setTaskId(e.target.value)}>
                <option value="">No task</option>
                {tasks.map((task) => <option key={task.id} value={task.id}>{task.title}</option>)}
              </select>
            </label>
            <div className="flex justify-end"><Button type="submit" variant="primary" disabled={start.isPending}><Play className="h-3 w-3" /> Start</Button></div>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
