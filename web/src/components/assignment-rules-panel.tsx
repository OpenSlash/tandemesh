import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Trash2 } from 'lucide-react'
import { api } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { useWorkspace } from '../lib/workspace'
import { Button } from './ui/button'
import { Input } from './ui/input'

export function AssignmentRulesPanel() {
  const { t } = useI18n()
  const { workspaceId } = useWorkspace()
  const qc = useQueryClient()
  const [name, setName] = useState('')
  const [provider, setProvider] = useState('')
  const [instance, setInstance] = useState('')
  const [label, setLabel] = useState('')
  const [agentId, setAgentId] = useState('')
  const [groupId, setGroupId] = useState('')
  const [autoDispatch, setAutoDispatch] = useState(true)

  const { data } = useQuery({ queryKey: ['automations', workspaceId], queryFn: () => api.automations(workspaceId) })
  const { data: agentsData } = useQuery({ queryKey: ['agents', workspaceId], queryFn: () => api.agents(workspaceId) })
  const { data: groupsData } = useQuery({ queryKey: ['groups', workspaceId], queryFn: () => api.groups(workspaceId) })
  const rules = (data?.items ?? []).filter((item) => item.triggerKind === 'task.synced')
  const agents = agentsData?.items ?? []
  const groups = groupsData?.items ?? []

  const invalidate = () => qc.invalidateQueries({ queryKey: ['automations', workspaceId] })
  const add = useMutation({
    mutationFn: () =>
      api.createAutomation({
        workspaceId,
        kind: 'event',
        name: name || 'Task routing',
        provider: provider || undefined,
        instance: instance || undefined,
        triggerKind: 'task.synced',
        triggerValue: label || undefined,
        agentId,
        writeback: { groupId: groupId || undefined, autoDispatch, policy: 'conditional' },
        enabled: true,
      }),
    onSuccess: () => {
      invalidate()
      setName('')
      setProvider('')
      setInstance('')
      setLabel('')
      setAgentId('')
      setGroupId('')
    },
  })
  const remove = useMutation({ mutationFn: (id: string) => api.deleteAutomation(id), onSuccess: invalidate })

  return (
    <section className="mt-8">
      <h2 className="text-[13px] font-medium">{t('rules.title')}</h2>
      <p className="mt-0.5 text-[11.5px] text-muted">{t('providers.rulesHint')}</p>
      <div className="mt-2 space-y-2">
        {rules.map((rule) => (
          <div key={rule.id} className="flex flex-wrap items-center gap-2 rounded-lg border border-line bg-panel/40 p-2.5">
            <span className="text-[12px] font-medium">{rule.name || rule.id}</span>
            <span className="font-mono text-[10.5px] text-muted">{rule.provider || '*'}</span>
            <span className="font-mono text-[10.5px] text-muted">{rule.instance || '*'}</span>
            {rule.triggerValue && <span className="rounded bg-panel2 px-1.5 py-0.5 text-[10px]">{rule.triggerValue}</span>}
            <span className="ml-auto text-[10.5px] text-muted">{rule.agentId}</span>
            <button className="text-muted hover:text-err" onClick={() => remove.mutate(rule.id)}>
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </div>
        ))}
        {rules.length === 0 && <p className="text-[11.5px] text-muted">{t('providers.noRules')}</p>}
      </div>
      <form
        className="mt-3 flex flex-wrap items-end gap-2 rounded-lg border border-line bg-panel/40 p-2.5"
        onSubmit={(e) => {
          e.preventDefault()
          if (agentId) add.mutate()
        }}
      >
        <label className="text-[11px] text-muted">
          {t('providers.ruleName')}
          <Input className="mt-1 h-7 w-40" value={name} onChange={(e) => setName(e.target.value)} />
        </label>
        <label className="text-[11px] text-muted">
          {t('meta.provider')}
          <Input className="mt-1 h-7 w-28" value={provider} onChange={(e) => setProvider(e.target.value)} />
        </label>
        <label className="text-[11px] text-muted">
          {t('providers.instance')}
          <Input className="mt-1 h-7 w-56" value={instance} onChange={(e) => setInstance(e.target.value)} />
        </label>
        <label className="text-[11px] text-muted">
          {t('common.labels')}
          <Input className="mt-1 h-7 w-28" value={label} onChange={(e) => setLabel(e.target.value)} />
        </label>
        <label className="text-[11px] text-muted">
          {t('providers.agent')}
          <select className="mt-1 h-7 rounded-md border border-line2 bg-panel px-2 text-[11px]" value={agentId} onChange={(e) => setAgentId(e.target.value)}>
            <option value="">-</option>
            {agents.map((agent) => <option key={agent.id} value={agent.id}>{agent.name}</option>)}
          </select>
        </label>
        <label className="text-[11px] text-muted">
          {t('providers.group')}
          <select className="mt-1 h-7 rounded-md border border-line2 bg-panel px-2 text-[11px]" value={groupId} onChange={(e) => setGroupId(e.target.value)}>
            <option value="">-</option>
            {groups.map((group) => <option key={group.id} value={group.id}>{group.name}</option>)}
          </select>
        </label>
        <label className="flex items-center gap-1.5 pb-1.5 text-[11px] text-muted">
          <input type="checkbox" checked={autoDispatch} onChange={(e) => setAutoDispatch(e.target.checked)} />
          {t('providers.autoDispatch')}
        </label>
        <Button type="submit" size="sm" variant="primary" disabled={(!agentId && !groupId) || add.isPending}>
          {t('common.add')}
        </Button>
      </form>
    </section>
  )
}
