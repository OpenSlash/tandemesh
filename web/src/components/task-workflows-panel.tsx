import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Play, Workflow } from 'lucide-react'
import { api } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { Button } from './ui/button'
import { cn } from '../lib/utils'

export function TaskWorkflowsPanel({ taskId, workspaceId }: { taskId: string; workspaceId: string }) {
  const { t } = useI18n()
  const qc = useQueryClient()
  const [workflowId, setWorkflowId] = useState('')
  const { data: workflowsData } = useQuery({ queryKey: ['workflows', workspaceId], queryFn: () => api.workflows(workspaceId) })
  const { data: runsData } = useQuery({
    queryKey: ['workflow-runs', workspaceId, taskId],
    queryFn: () => api.workflowRuns(workspaceId, taskId),
    refetchInterval: 3000,
  })
  const workflows = workflowsData?.items ?? []
  const runs = runsData?.items ?? []
  const start = useMutation({
    mutationFn: () => api.startWorkflow(workflowId, { taskId }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['workflow-runs', workspaceId, taskId] })
      setWorkflowId('')
    },
  })

  return (
    <section className="rounded-lg border border-line bg-panel/20 p-3">
      <div className="flex items-center gap-2">
        <Workflow className="h-3.5 w-3.5 text-muted" />
        <h2 className="text-[12px] uppercase tracking-wider text-muted">{t('nav.workflows')}</h2>
        <span className="ml-auto text-[10px] text-muted">{t('runs.count', { count: runs.length })}</span>
      </div>
      <div className="mt-2 flex flex-wrap items-end gap-2">
        <select
          className="h-7 min-w-52 rounded-md border border-line2 bg-panel px-2 text-[11px]"
          value={workflowId}
          onChange={(e) => setWorkflowId(e.target.value)}
        >
          <option value="">{t('taskWorkflows.select')}</option>
          {workflows.map((wf) => <option key={wf.id} value={wf.id}>{wf.name}</option>)}
        </select>
        <Button size="sm" variant="primary" disabled={!workflowId || start.isPending} onClick={() => start.mutate()}>
          <Play className="h-3 w-3" /> {t('common.start')}
        </Button>
      </div>
      {start.error && <p className="mt-1 text-[10.5px] text-err">{String(start.error.message)}</p>}
      <div className="mt-2 space-y-1">
        {runs.slice(0, 5).map((run) => (
          <a key={run.id} href={`/workflows?run=${run.id}`} className="flex items-center gap-2 rounded border border-line2 px-2 py-1 text-[11px] hover:bg-panel2/50">
            <span className="font-mono text-[10px] text-muted">{run.id.slice(0, 14)}</span>
            <span className={cn('ml-auto', run.status === 'succeeded' ? 'text-ok' : run.status === 'failed' ? 'text-err' : 'text-warn')}>{run.status}</span>
          </a>
        ))}
        {runs.length === 0 && <p className="text-[10.5px] text-muted">{t('taskWorkflows.empty')}</p>}
      </div>
    </section>
  )
}
