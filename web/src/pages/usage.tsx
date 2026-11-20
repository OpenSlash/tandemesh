import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Activity } from 'lucide-react'
import { api } from '../lib/api'
import { useWorkspace } from '../lib/workspace'
import { cn } from '../lib/utils'

function formatTokens(value: number) {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}k`
  return String(value)
}

export default function UsagePage() {
  const { workspaceId } = useWorkspace()
  const [days, setDays] = useState(30)
  const { data, isLoading } = useQuery({ queryKey: ['usage', workspaceId, days], queryFn: () => api.usage(workspaceId, days) })
  const items = data?.items ?? []

  return (
    <div className="mx-auto max-w-[1180px] px-6 py-5">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-[18px] font-semibold tracking-tight">Usage</h1>
          <p className="mt-0.5 text-[12px] text-muted">Token usage by agent and model.</p>
        </div>
        <select className="h-8 rounded-md border border-line2 bg-panel px-2 text-[11.5px] text-muted" value={days} onChange={(e) => setDays(Number(e.target.value))}>
          <option value={7}>7 days</option>
          <option value={30}>30 days</option>
          <option value={90}>90 days</option>
        </select>
      </div>

      <div className="mt-5 grid grid-cols-2 gap-3 md:grid-cols-4">
        {[
          ['Runs', data?.runs ?? 0],
          ['Input tokens', data?.inputTokens ?? 0],
          ['Output tokens', data?.outputTokens ?? 0],
          ['Cache read', data?.cacheReadTokens ?? 0],
        ].map(([label, value]) => (
          <div key={String(label)} className="rounded-lg border border-line bg-panel/40 p-3">
            <p className="text-[10px] uppercase tracking-wider text-muted">{label}</p>
            <p className="mt-1 text-[20px] font-semibold tabular-nums">{formatTokens(Number(value))}</p>
          </div>
        ))}
      </div>

      <div className="mt-5 overflow-hidden rounded-lg border border-line">
        <table className="w-full text-left text-[11.5px]">
          <thead className="bg-panel/80 text-[10px] uppercase tracking-wider text-muted">
            <tr>
              <th className="px-3 py-2 font-medium">Agent</th>
              <th className="px-3 py-2 font-medium">Model</th>
              <th className="px-3 py-2 font-medium">Runs</th>
              <th className="px-3 py-2 font-medium">Input</th>
              <th className="px-3 py-2 font-medium">Output</th>
              <th className="px-3 py-2 font-medium">Cache read</th>
            </tr>
          </thead>
          <tbody>
            {items.map((row) => (
              <tr key={`${row.agentId}-${row.model}`} className="border-t border-line/60">
                <td className="px-3 py-2 font-mono text-[10.5px]">{row.agentId || '-'}</td>
                <td className="px-3 py-2 text-muted">{row.model || '-'}</td>
                <td className="px-3 py-2">{row.runs}</td>
                <td className={cn('px-3 py-2', row.inputTokens > 0 && 'text-fg')}>{formatTokens(row.inputTokens)}</td>
                <td className="px-3 py-2">{formatTokens(row.outputTokens)}</td>
                <td className="px-3 py-2 text-muted">{formatTokens(row.cacheReadTokens)}</td>
              </tr>
            ))}
          </tbody>
        </table>
        {isLoading && <p className="p-4 text-[11.5px] text-muted">Loading usage...</p>}
        {!isLoading && items.length === 0 && <p className="p-4 text-[11.5px] text-muted">No usage in this period.</p>}
      </div>
    </div>
  )
}
