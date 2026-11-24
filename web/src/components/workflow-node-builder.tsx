import type { WorkflowNode } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { Button } from './ui/button'
import { Input } from './ui/input'

export function WorkflowNodeBuilder({ nodes, onChange }: { nodes: WorkflowNode[]; onChange: (nodes: WorkflowNode[]) => void }) {
  const { t } = useI18n()
  const update = (index: number, patch: Partial<WorkflowNode>) => {
    onChange(nodes.map((node, i) => (i === index ? { ...node, ...patch } : node)))
  }
  const remove = (index: number) => onChange(nodes.filter((_, i) => i !== index))
  const add = () => onChange([...nodes, { id: `step${nodes.length + 1}`, kind: 'script', name: 'Step' }])

  return (
    <div className="rounded-md border border-line2 p-2">
      <div className="flex items-center justify-between">
        <span className="text-[10.5px] uppercase tracking-wider text-muted">{t('workflows.nodeBuilder')}</span>
        <Button type="button" size="sm" variant="secondary" onClick={add}>{t('workflows.addNode')}</Button>
      </div>
      <div className="mt-2 space-y-2">
        {nodes.map((node, index) => (
          <div key={`${node.id}-${index}`} className="rounded border border-line2/70 p-2">
            <div className="flex flex-wrap items-end gap-1.5">
              <label className="text-[10px] text-muted">{t('workflows.id')}<Input className="mt-0.5 h-7 w-24" value={node.id} onChange={(e) => update(index, { id: e.target.value })} /></label>
              <label className="text-[10px] text-muted">{t('workflows.kind')}
                <select className="mt-0.5 h-7 rounded border border-line2 bg-panel px-1 text-[11px]" value={node.kind} onChange={(e) => update(index, { kind: e.target.value as WorkflowNode['kind'] })}>
                  <option value="script">script</option>
                  <option value="parallel">parallel</option>
                  <option value="fanout">fanout</option>
                  <option value="merge">merge</option>
                  <option value="agent">agent</option>
                  <option value="gate">gate</option>
                  <option value="condition">condition</option>
                  <option value="workflow">workflow</option>
                </select>
              </label>
              <label className="text-[10px] text-muted">{t('workflows.nodeName')}<Input className="mt-0.5 h-7 w-32" value={node.name} onChange={(e) => update(index, { name: e.target.value })} /></label>
              {node.kind === 'script' && <label className="text-[10px] text-muted">{t('workflows.command')}<Input className="mt-0.5 h-7 w-64" value={node.command ?? ''} onChange={(e) => update(index, { command: e.target.value })} /></label>}
              {node.kind === 'parallel' && <label className="text-[10px] text-muted">{t('workflows.commands')}<Input className="mt-0.5 h-7 w-80" value={(node.commands ?? []).join(', ')} onChange={(e) => update(index, { commands: e.target.value.split(',').map((s) => s.trim()).filter(Boolean) })} /></label>}
              {node.kind === 'fanout' && <label className="text-[10px] text-muted">{t('workflows.agentIds')}<Input className="mt-0.5 h-7 w-80" value={(node.agentIds ?? []).join(', ')} onChange={(e) => update(index, { agentIds: e.target.value.split(',').map((s) => s.trim()).filter(Boolean) })} /></label>}
              {node.kind === 'agent' && <>
                <label className="text-[10px] text-muted">{t('workflows.agent')}<Input className="mt-0.5 h-7 w-44" value={node.agentId ?? ''} onChange={(e) => update(index, { agentId: e.target.value })} /></label>
                <label className="text-[10px] text-muted">{t('workflows.role')}<Input className="mt-0.5 h-7 w-24" value={node.role ?? ''} onChange={(e) => update(index, { role: e.target.value })} /></label>
                <label className="text-[10px] text-muted">{t('workflows.model')}<Input className="mt-0.5 h-7 w-36" value={node.model ?? ''} onChange={(e) => update(index, { model: e.target.value })} /></label>
              </>}
              {node.kind === 'workflow' && <label className="text-[10px] text-muted">{t('workflows.workflow')}<Input className="mt-0.5 h-7 w-48" value={node.workflowId ?? ''} onChange={(e) => update(index, { workflowId: e.target.value })} /></label>}
              {node.kind === 'condition' && <label className="text-[10px] text-muted">{t('workflows.when')}<Input className="mt-0.5 h-7 w-56" value={node.when ?? ''} onChange={(e) => update(index, { when: e.target.value })} /></label>}
              {node.kind === 'merge' && <label className="text-[10px] text-muted">{t('workflows.mergeAgent')}<Input className="mt-0.5 h-7 w-44" value={node.agentId ?? ''} onChange={(e) => update(index, { agentId: e.target.value })} /></label>}
              <label className="text-[10px] text-muted">{t('workflows.needs')}<Input className="mt-0.5 h-7 w-40" value={(node.needs ?? []).join(', ')} onChange={(e) => update(index, { needs: e.target.value.split(',').map((s) => s.trim()).filter(Boolean) })} /></label>
              <label className="text-[10px] text-muted">{t('workflows.next')}<Input className="mt-0.5 h-7 w-24" value={node.next ?? ''} onChange={(e) => update(index, { next: e.target.value })} /></label>
              <label className="text-[10px] text-muted">{t('workflows.onFail')}<Input className="mt-0.5 h-7 w-24" value={node.onFail ?? ''} onChange={(e) => update(index, { onFail: e.target.value })} /></label>
              <label className="text-[10px] text-muted">{t('workflows.retries')}<Input className="mt-0.5 h-7 w-16" type="number" value={node.retries ?? 0} onChange={(e) => update(index, { retries: Number(e.target.value) })} /></label>
              <label className="text-[10px] text-muted">{t('workflows.timeout')}<Input className="mt-0.5 h-7 w-16" type="number" value={node.timeoutSeconds ?? 0} onChange={(e) => update(index, { timeoutSeconds: Number(e.target.value) })} /></label>
              <button type="button" className="ml-auto text-muted hover:text-err" onClick={() => remove(index)}>{t('workflows.remove')}</button>
            </div>
            {node.kind === 'agent' && <Input className="mt-1" placeholder={t('workflows.prompt')} value={node.prompt ?? ''} onChange={(e) => update(index, { prompt: e.target.value })} />}
          </div>
        ))}
        {nodes.length === 0 && <p className="text-[11px] text-muted">{t('workflows.noNodes')}</p>}
      </div>
    </div>
  )
}
