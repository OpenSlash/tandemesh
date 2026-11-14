import { useState } from 'react'
import { useI18n } from '../lib/i18n'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { Check, ShieldCheck, X } from 'lucide-react'
import { api, type Approval } from '../lib/api'
import { useWorkspace } from '../lib/workspace'
import { age, cn, compactId } from '../lib/utils'
import { Button } from '../components/ui/button'

const RISK_KEYS: Record<string, string> = {
  low: 'risk.low',
  medium: 'risk.medium',
  high: 'risk.high',
}

export default function ApprovalsPage() {
  const { t } = useI18n()
  const qc = useQueryClient()
  const { workspaces } = useWorkspace()
  const [scope, setScope] = useState('all')
  const { data, isLoading, error } = useQuery({
    queryKey: ['approvals', scope],
    queryFn: () => api.approvals(scope),
    refetchInterval: 3000,
  })

  const decide = useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: 'approved' | 'denied' }) =>
      api.decide(id, decision),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['approvals'] }),
  })

  const items = data?.items ?? []

  return (
    <div className="mx-auto max-w-[1180px] px-6 py-5">
      <div className="flex items-end justify-between">
        <div>
          <h1 className="text-[18px] font-semibold tracking-tight">{t('page.approvals')}</h1>
          <p className="mt-0.5 text-[12px] text-muted">{t('approvals.subtitle')}</p>
        </div>
        <div className="flex items-center gap-3">
          <select
            value={scope}
            onChange={(event) => setScope(event.target.value)}
            className="h-7 rounded-md border border-line2 bg-panel px-2 text-[11.5px] text-fg"
          >
            <option value="all">{t('shell.allWorkspaces')}</option>
            {workspaces.map((workspace) => (
              <option key={workspace.id} value={workspace.id}>
                {workspace.name}
              </option>
            ))}
          </select>
          <span className="text-[12px] text-muted">{t('approvals.pending', { count: items.length })}</span>
        </div>
      </div>

      {error && <p className="mt-4 text-[12px] text-err">{String(error.message)}</p>}
      {isLoading && <p className="mt-6 text-[12px] text-muted">{t('common.loading')}</p>}

      <div className="mt-4 space-y-2">
        {items.map((item) => (
          <ApprovalRow
            key={item.id}
            item={item}
            busy={decide.isPending}
            onDecide={(decision) => decide.mutate({ id: item.id, decision })}
          />
        ))}
      </div>

      {!isLoading && items.length === 0 && (
        <div className="mt-8 flex flex-col items-center justify-center gap-2 rounded-lg border border-line py-16 text-muted">
          <ShieldCheck className="h-6 w-6 opacity-40" />
          <p className="text-[13px]">{t('approvals.empty')}</p>
          <p className="text-[11px]">{t('approvals.emptyHint')}</p>
        </div>
      )}
    </div>
  )
}

function ApprovalRow({
  item,
  busy,
  onDecide,
}: {
  item: Approval
  busy: boolean
  onDecide: (decision: 'approved' | 'denied') => void
}) {
  const { t } = useI18n()
  const risk = item.risk?.toLowerCase() ?? 'medium'
  return (
    <div className="flex items-start gap-3 rounded-lg border border-line bg-panel/40 px-3 py-3">
      <div
        className={cn(
          'mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-md border',
          risk === 'high'
            ? 'border-err/40 bg-err/10 text-err'
            : 'border-warn/40 bg-warn/10 text-warn',
        )}
      >
        <ShieldCheck className="h-3.5 w-3.5" />
      </div>

      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <span className="text-[13px] font-medium text-fg">{item.tool}</span>
          <span
            className={cn(
              'rounded border px-1 text-[10px] uppercase tracking-wide',
              risk === 'high' ? 'border-err/40 text-err' : 'border-warn/40 text-warn',
            )}
          >
            {RISK_KEYS[risk] ? t(RISK_KEYS[risk]) : risk}
          </span>
          <span className="text-[11px] text-muted">
            {t('approvals.needsApproval', { age: age(item.requestedAt) })}
          </span>
        </div>
        {item.command && (
          <pre className="mt-1.5 overflow-x-auto rounded border border-line bg-bg px-2 py-1.5 font-mono text-[11.5px] text-fg/80">
            {item.command}
          </pre>
        )}
        <div className="mt-1.5 flex items-center gap-3 text-[11px] text-muted">
          <Link to={`/runs/${item.runId}`} className="text-accent2 hover:underline">
            {compactId(item.runId, 10)}
          </Link>
          <span>{t('approvals.expires', { time: new Date(item.expiresAt).toLocaleTimeString() })}</span>
        </div>
      </div>

      <div className="flex shrink-0 gap-2">
        <Button size="sm" variant="secondary" disabled={busy} onClick={() => onDecide('denied')}>
          <X className="h-3.5 w-3.5" /> {t('approvals.deny')}
        </Button>
        <Button size="sm" variant="primary" disabled={busy} onClick={() => onDecide('approved')}>
          <Check className="h-3.5 w-3.5" /> {t('approvals.approve')}
        </Button>
      </div>
    </div>
  )
}
