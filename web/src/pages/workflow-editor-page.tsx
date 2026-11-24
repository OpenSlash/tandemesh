import { useEffect, useRef, useState } from 'react'
import { useNavigate, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api, type WorkflowNode } from '../lib/api'
import { useWorkspace } from '../lib/workspace'
import { useI18n } from '../lib/i18n'
import { WorkflowEditor } from '../components/workflow-editor'

const DEFAULT_NODES = `[
  {"id":"review","kind":"gate","name":"Review","next":"run"},
  {"id":"run","kind":"script","name":"Run","command":"echo workflow ok"}
]`

export default function WorkflowEditorPage() {
  const { id } = useParams()
  const navigate = useNavigate()
  const { workspaceId } = useWorkspace()
  const { t } = useI18n()
  const qc = useQueryClient()
  const mode = id ? 'edit' : 'create'
  const loaded = useRef(false)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [nodesText, setNodesText] = useState(DEFAULT_NODES)
  const [message, setMessage] = useState('')

  const { data: workflowsData } = useQuery({ queryKey: ['workflows', workspaceId], queryFn: () => api.workflows(workspaceId) })
  const { data: templatesData } = useQuery({ queryKey: ['workflow-templates'], queryFn: api.workflowTemplates })
  const workflow = id ? (workflowsData?.items ?? []).find((item) => item.id === id) : undefined

  useEffect(() => {
    if (mode !== 'edit' || !workflow || loaded.current) return
    loaded.current = true
    setName(workflow.name)
    setDescription(workflow.description ?? '')
    setNodesText(JSON.stringify(workflow.nodes, null, 2))
  }, [mode, workflow])

  const save = useMutation({
    mutationFn: () => {
      const nodes = JSON.parse(nodesText) as WorkflowNode[]
      if (mode === 'edit' && id) {
        return api.updateWorkflow(id, { workspaceId, name, description, nodes })
      }
      return api.createWorkflow({ workspaceId, name, description, nodes })
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['workflows', workspaceId] })
      void qc.invalidateQueries({ queryKey: ['workflow-templates'] })
      navigate('/workflows')
    },
    onError: (err) => setMessage(String(err)),
  })

  if (mode === 'edit' && !workflow) {
    return <div className="p-6 text-[12px] text-muted">{t('workflows.loading')}</div>
  }

  return (
    <div className="flex h-[calc(100vh-1px)] flex-col p-3">
      <WorkflowEditor
        embedded
        mode={mode}
        name={name}
        description={description}
        nodesText={nodesText}
        templates={templatesData?.items ?? []}
        saving={save.isPending}
        error={message}
        onName={setName}
        onDescription={setDescription}
        onNodes={setNodesText}
        onSave={() => { setMessage(''); save.mutate() }}
        onClose={() => navigate('/workflows')}
      />
    </div>
  )
}
