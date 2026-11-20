import type { WorkflowNode } from '../lib/api'
import { Button } from './ui/button'
import { Input } from './ui/input'

export function WorkflowNodeBuilder({ nodes, onChange }: { nodes: WorkflowNode[]; onChange: (nodes: WorkflowNode[]) => void }) {
  const update = (index: number, patch: Partial<WorkflowNode>) => {
    onChange(nodes.map((node, i) => (i === index ? { ...node, ...patch } : node)))
  }
  const remove = (index: number) => onChange(nodes.filter((_, i) => i !== index))
  const add = () => onChange([...nodes, { id: `step${nodes.length + 1}`, kind: 'script', name: 'Step' }])

  return (
    <div className="rounded-md border border-line2 p-2">
      <div className="flex items-center justify-between">
        <span className="text-[10.5px] uppercase tracking-wider text-muted">Node builder</span>
        <Button type="button" size="sm" variant="secondary" onClick={add}>Add node</Button>
      </div>
      <div className="mt-2 space-y-2">
        {nodes.map((node, index) => (
          <div key={`${node.id}-${index}`} className="rounded border border-line2/70 p-2">
            <div className="flex flex-wrap items-end gap-1.5">
              <label className="text-[10px] text-muted">ID<Input className="mt-0.5 h-7 w-24" value={node.id} onChange={(e) => update(index, { id: e.target.value })} /></label>
              <label className="text-[10px] text-muted">Kind
                <select className="mt-0.5 h-7 rounded border border-line2 bg-panel px-1 text-[11px]" value={node.kind} onChange={(e) => update(index, { kind: e.target.value as WorkflowNode['kind'] })}>
                  <option value="script">script</option>
                  <option value="parallel">parallel</option>
                  <option value="fanout">fanout</option>
                  <option value="agent">agent</option>
                  <option value="gate">gate</option>
                  <option value="condition">condition</option>
                  <option value="workflow">workflow</option>
                </select>
              </label>
              <label className="text-[10px] text-muted">Name<Input className="mt-0.5 h-7 w-32" value={node.name} onChange={(e) => update(index, { name: e.target.value })} /></label>
              {node.kind === 'script' && <label className="text-[10px] text-muted">Command<Input className="mt-0.5 h-7 w-64" value={node.command ?? ''} onChange={(e) => update(index, { command: e.target.value })} /></label>}
              {node.kind === 'parallel' && <label className="text-[10px] text-muted">Commands (comma separated)<Input className="mt-0.5 h-7 w-80" value={(node.commands ?? []).join(', ')} onChange={(e) => update(index, { commands: e.target.value.split(',').map((s) => s.trim()).filter(Boolean) })} /></label>}
              {node.kind === 'fanout' && <label className="text-[10px] text-muted">Agent IDs (comma separated)<Input className="mt-0.5 h-7 w-80" value={(node.agentIds ?? []).join(', ')} onChange={(e) => update(index, { agentIds: e.target.value.split(',').map((s) => s.trim()).filter(Boolean) })} /></label>}
              {node.kind === 'agent' && <>
                <label className="text-[10px] text-muted">Agent<Input className="mt-0.5 h-7 w-44" value={node.agentId ?? ''} onChange={(e) => update(index, { agentId: e.target.value })} /></label>
                <label className="text-[10px] text-muted">Role<Input className="mt-0.5 h-7 w-24" value={node.role ?? ''} onChange={(e) => update(index, { role: e.target.value })} /></label>
                <label className="text-[10px] text-muted">Model<Input className="mt-0.5 h-7 w-36" value={node.model ?? ''} onChange={(e) => update(index, { model: e.target.value })} /></label>
              </>}
              {node.kind === 'workflow' && <label className="text-[10px] text-muted">Workflow<Input className="mt-0.5 h-7 w-48" value={node.workflowId ?? ''} onChange={(e) => update(index, { workflowId: e.target.value })} /></label>}
              {node.kind === 'condition' && <label className="text-[10px] text-muted">When<Input className="mt-0.5 h-7 w-56" value={node.when ?? ''} onChange={(e) => update(index, { when: e.target.value })} /></label>}
              <label className="text-[10px] text-muted">Next<Input className="mt-0.5 h-7 w-24" value={node.next ?? ''} onChange={(e) => update(index, { next: e.target.value })} /></label>
              <label className="text-[10px] text-muted">On fail<Input className="mt-0.5 h-7 w-24" value={node.onFail ?? ''} onChange={(e) => update(index, { onFail: e.target.value })} /></label>
              <label className="text-[10px] text-muted">Retries<Input className="mt-0.5 h-7 w-16" type="number" value={node.retries ?? 0} onChange={(e) => update(index, { retries: Number(e.target.value) })} /></label>
              <label className="text-[10px] text-muted">Timeout (s)<Input className="mt-0.5 h-7 w-16" type="number" value={node.timeoutSeconds ?? 0} onChange={(e) => update(index, { timeoutSeconds: Number(e.target.value) })} /></label>
              <button type="button" className="ml-auto text-muted hover:text-err" onClick={() => remove(index)}>Remove</button>
            </div>
            {node.kind === 'agent' && <Input className="mt-1" placeholder="Prompt" value={node.prompt ?? ''} onChange={(e) => update(index, { prompt: e.target.value })} />}
          </div>
        ))}
        {nodes.length === 0 && <p className="text-[11px] text-muted">No nodes. Add one or paste JSON.</p>}
      </div>
    </div>
  )
}
