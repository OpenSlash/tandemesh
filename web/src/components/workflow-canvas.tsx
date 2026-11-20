import { useMemo } from 'react'
import {
  Background,
  Controls,
  Handle,
  MiniMap,
  Position,
  ReactFlow,
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

  const savePosition = (_event: unknown, moved: Node) => {
    const index = nodes.findIndex((node) => node.id === moved.id)
    if (index < 0) return
    const next = nodes.map((node, i) => i === index ? { ...node, position: { x: Math.round(moved.position.x), y: Math.round(moved.position.y) } } : node)
    onChange(next)
  }

  return (
    <div className="h-[340px] overflow-hidden rounded-md border border-line2">
      <ReactFlow
        nodes={flowNodes}
        edges={edges}
        nodeTypes={nodeTypes}
        fitView
        onNodeDragStop={savePosition}
        nodesConnectable={false}
        proOptions={{ hideAttribution: true }}
      >
        <Background gap={18} size={1} />
        <MiniMap pannable zoomable className="!bg-panel" />
        <Controls />
      </ReactFlow>
    </div>
  )
}
