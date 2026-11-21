import type { WorkflowNode, WorkflowRun } from '../lib/api'
import { cn } from '../lib/utils'

const NODE_W = 150
const NODE_H = 44
const GAP_X = 70
const GAP_Y = 24

function statusClass(status?: string) {
  if (status === 'succeeded') return 'fill-ok/10 stroke-ok'
  if (status === 'failed') return 'fill-err/10 stroke-err'
  if (status === 'running' || status === 'waiting') return 'fill-warn/10 stroke-warn'
  if (status === 'skipped') return 'fill-panel2 stroke-line2 opacity-50'
  return 'fill-panel stroke-line2'
}

export function WorkflowGraph({ nodes, run }: { nodes: WorkflowNode[]; run: WorkflowRun }) {
  if (nodes.length === 0) return null
  const byID = new Map(nodes.map((node) => [node.id, node]))
  const incoming = new Map<string, string[]>()
  for (const node of nodes) incoming.set(node.id, [])
  const hasNeeds = nodes.some((node) => (node.needs ?? []).length > 0)
  for (const node of nodes) {
    if (hasNeeds) {
      for (const need of node.needs ?? []) incoming.get(node.id)?.push(need)
    } else {
      for (const other of nodes) {
        if (other.next === node.id) incoming.get(node.id)?.push(other.id)
      }
    }
  }

  const level = new Map<string, number>()
  const visiting = new Set<string>()
  const depth = (id: string): number => {
    if (level.has(id)) return level.get(id)!
    if (visiting.has(id)) return 0
    visiting.add(id)
    const parents = incoming.get(id) ?? []
    const value = parents.length === 0 ? 0 : Math.max(...parents.map((parent) => depth(parent) + 1))
    visiting.delete(id)
    level.set(id, value)
    return value
  }
  nodes.forEach((node) => depth(node.id))

  const byLevel = new Map<number, WorkflowNode[]>()
  for (const node of nodes) {
    const value = level.get(node.id) ?? 0
    byLevel.set(value, [...(byLevel.get(value) ?? []), node])
  }
  const maxLevel = Math.max(...[...byLevel.keys()], 0)
  const maxRows = Math.max(...[...byLevel.values()].map((items) => items.length), 1)
  const width = (maxLevel + 1) * (NODE_W + GAP_X) + 20
  const height = maxRows * (NODE_H + GAP_Y) + 20
  const statuses = new Map(run.nodes.map((node) => [node.nodeId, node.status]))
  const positions = new Map<string, { x: number; y: number }>()
  for (const [value, items] of byLevel) {
    items.forEach((node, index) => {
      positions.set(node.id, { x: 10 + value * (NODE_W + GAP_X), y: 10 + index * (NODE_H + GAP_Y) })
    })
  }

  return (
    <div className="mt-3 overflow-auto rounded border border-line2 bg-panel/20 p-2">
      <svg width={width} height={height} className="min-w-full">
        {nodes.flatMap((node) =>
          (incoming.get(node.id) ?? []).map((parent) => {
            const from = positions.get(parent)
            const to = positions.get(node.id)
            if (!from || !to) return null
            return (
              <line
                key={`${parent}-${node.id}`}
                x1={from.x + NODE_W}
                y1={from.y + NODE_H / 2}
                x2={to.x}
                y2={to.y + NODE_H / 2}
                className="stroke-line2"
                strokeWidth={1}
              />
            )
          }),
        )}
        {nodes.map((node) => {
          const pos = positions.get(node.id)
          if (!pos) return null
          const status = statuses.get(node.id)
          return (
            <g key={node.id}>
              <rect x={pos.x} y={pos.y} width={NODE_W} height={NODE_H} rx={6} className={cn('stroke', statusClass(status))} strokeWidth={1} />
              <text x={pos.x + 8} y={pos.y + 18} className="fill-fg text-[10px]">{node.name || node.id}</text>
              <text x={pos.x + 8} y={pos.y + 33} className="fill-muted text-[9px]">{node.kind} · {status ?? 'pending'}</text>
            </g>
          )
        })}
      </svg>
    </div>
  )
}
