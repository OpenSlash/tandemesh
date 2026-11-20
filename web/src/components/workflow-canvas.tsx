import { useMemo, useState } from 'react'
import {
  Background,
  Controls,
  Handle,
  MiniMap,
  Position,
  ReactFlow,
  type Connection,
  type Edge,
  type Node,
  type NodeProps,
} from '@xyflow/react'
import '@xyflow/react/dist/style.css'
import type { WorkflowNode } from '../lib/api'

type CanvasData = { label: string; kind: string }
type CanvasNode = Node<CanvasData, 'workflow'>

function WorkflowCanvasNode({ data }: NodeProps<CanvasNode>) {
  return (
    <div className="min-w-[140px] rounded-md border border-line2 bg-panel px-2 py-1.5 shadow-sm">
      <Handle type="target" position={Position.Left} className="!h-2 !w-2 !border-line2 !bg-panel2" />
      <p className="truncate text-[11px] font-medium">{data.label}</p>
      <p className="text-[9px] uppercase tracking-wide text-muted">{data.kind}</p>
      <Handle type="source" position={Position.Right} className="!h-2 !w-2 !border-line2 !bg-panel2" />
    </div>
  )
}

const nodeTypes = { workflow: WorkflowCanvasNode }

function autoLayout(nodes: WorkflowNode[]): WorkflowNode[] {
  const byID = new Map(nodes.map((node) => [node.id, node]))
  const incoming = new Map<string, string[]>()
  for (const node of nodes) incoming.set(node.id, [])
  const hasNeeds = nodes.some((node) => (node.needs ?? []).length > 0)
  for (const node of nodes) {
    if (hasNeeds) {
      for (const need of node.needs ?? []) incoming.get(node.id)?.push(need)
    } else if (node.next) {
      incoming.get(node.next)?.push(node.id)
    }
  }
  const depth = new Map<string, number>()
  const visit = (id: string, stack: Set<string>): number => {
    if (depth.has(id)) return depth.get(id)!
    if (stack.has(id)) return 0
    stack.add(id)
    const parents = incoming.get(id) ?? []
    const value = parents.length === 0 ? 0 : Math.max(...parents.map((parent) => visit(parent, stack) + 1))
    stack.delete(id)
    depth.set(id, value)
    return value
  }
  nodes.forEach((node) => visit(node.id, new Set()))
  const rows = new Map<number, number>()
  return nodes.map((node) => {
    const column = depth.get(node.id) ?? 0
    const row = rows.get(column) ?? 0
    rows.set(column, row + 1)
    return { ...node, position: { x: 40 + column * 220, y: 40 + row * 110 } }
  })
}
export function WorkflowCanvas({ nodes, onChange }: { nodes: WorkflowNode[]; onChange: (nodes: WorkflowNode[]) => void }) {
  const flowNodes = useMemo<CanvasNode[]>(() => nodes.map((node, index) => ({
    id: node.id,
    type: 'workflow',
    position: node.position ?? { x: 40 + (index % 3) * 220, y: 40 + Math.floor(index / 3) * 110 },
    data: { label: node.name || node.id, kind: node.kind },
  })), [nodes])

  const edges = useMemo<Edge[]>(() => {
    const out: Edge[] = []
    for (const node of nodes) {
      if ((node.needs ?? []).length > 0) {
        for (const need of node.needs ?? []) out.push({ id: `${need}-${node.id}`, source: need, target: node.id })
      } else if (node.next) {
        out.push({ id: `${node.id}-${node.next}`, source: node.id, target: node.next })
      }
    }
    return out
  }, [nodes])

  const connect = (connection: Connection) => {
    if (!connection.source || !connection.target || connection.source === connection.target) return
    onChange(nodes.map((node) => {
      if (node.id !== connection.target) return node
      const needs = node.needs ?? []
      if (needs.includes(connection.source)) return node
      return { ...node, needs: [...needs, connection.source] }
    }))
  }
  const removeEdges = (removed: Edge[]) => {
    const pairs = new Set(removed.filter((edge) => !edge.id.startsWith('fail:')).map((edge) => `${edge.source}->${edge.target}`))
    const failPairs = new Set(removed.filter((edge) => edge.id.startsWith('fail:')).map((edge) => `${edge.source}->${edge.target}`))
    onChange(nodes.map((node) => ({
      ...node,
      next: pairs.has(`${node.id}->${node.next}`) ? '' : node.next,
      onFail: failPairs.has(`${node.id}->${node.onFail}`) ? '' : node.onFail,
      needs: (node.needs ?? []).filter((need) => !pairs.has(`${need}->${node.id}`)),
    })))
  }
  const savePosition = (_event: unknown, moved: Node) => {
    const index = nodes.findIndex((node) => node.id === moved.id)
    if (index < 0) return
    const next = nodes.map((node, i) => i === index ? { ...node, position: { x: Math.round(moved.position.x), y: Math.round(moved.position.y) } } : node)
    onChange(next)
  }

  return (
    <div className="overflow-hidden rounded-md border border-line2">
      <div className="flex items-center justify-between border-b border-line2 px-2 py-1">
        <span className="text-[10.5px] uppercase tracking-wider text-muted">Canvas</span>
        <button type="button" className="rounded border border-line2 px-2 py-0.5 text-[10.5px] text-muted hover:text-fg" onClick={() => onChange(autoLayout(nodes))}>
          Auto layout
        </button>
      </div>
      <div className="h-[310px]">
        <ReactFlow
          nodes={flowNodes}
          edges={edges}
          nodeTypes={nodeTypes}
          fitView
          onNodeDragStop={savePosition}
          onConnect={connect}
          onEdgesDelete={removeEdges}
          nodesConnectable
          proOptions={{ hideAttribution: true }}
        >
          <Background gap={18} size={1} />
          <MiniMap pannable zoomable className="!bg-panel" />
          <Controls />
        </ReactFlow>
      </div>
    </div>
  )
}
