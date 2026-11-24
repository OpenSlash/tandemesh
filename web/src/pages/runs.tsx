import { useI18n } from '../lib/i18n'
import { useState } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ArrowUpRight, Plus, TerminalSquare } from 'lucide-react'
import { api, type Run } from '../lib/api'
import { useWorkspace } from '../lib/workspace'
import { age, cn, refId } from '../lib/utils'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Dialog, DialogContent, DialogTrigger } from '../components/ui/dialog'
import { StatusBadge } from '../components/status'

const FILTERS: { key: string; labelKey: string }[] = [
  { key: '', labelKey: 'common.all' },
  { key: 'queued', labelKey: 'status.queued' },
  { key: 'running', labelKey: 'status.running' },
  { key: 'succeeded', labelKey: 'status.succeeded' },
  { key: 'failed', labelKey: 'status.failed' },
]

export default function RunsPage() {
  const { t } = useI18n()
  const [filter, setFilter] = useState('')
  const navigate = useNavigate()
  const qc = useQueryClient()
  const { workspaceId } = useWorkspace()
  const [searchParams] = useSearchParams()
  const scope = searchParams.get('scope') === 'all' ? 'all' : workspaceId

  const { data, isLoading, error } = useQuery({
    queryKey: ['runs', filter, scope],
    queryFn: () => api.runs(scope, filter || undefined),
    refetchInterval: 2000,
  })

  const [open, setOpen] = useState(false)
  const [repo, setRepo] = useState('acme/api')
  const [prompt, setPrompt] = useState('')
  const [agent, setAgent] = useState('claude')

  const create = useMutation({
    mutationFn: () => api.createRun({ repo, prompt, agentId: agent, workspaceId }),
    onSuccess: (run) => {
      setOpen(false)
      setPrompt('')
      qc.invalidateQueries({ queryKey: ['runs'] })
      navigate(`/runs/${run.id}`)
    },
  })

  const runs = data?.items ?? []

  return (
    <div className="mx-auto max-w-[1180px] px-6 py-5">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-[18px] font-semibold tracking-tight">{t('page.runs')}</h1>
          <p className="mt-0.5 text-[12px] text-muted">{t('runs.subtitle')}</p>
        </div>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button variant="primary">
              <Plus className="h-3.5 w-3.5" /> {t('runs.newRun')}
            </Button>
          </DialogTrigger>
          <DialogContent title={t('runs.newRun')} description={t('runs.newRunHint')}>
            <form
              className="flex flex-col gap-3"
              onSubmit={(e) => {
                e.preventDefault()
                create.mutate()
              }}
            >
              <label className="text-[12px] text-muted">
                {t('common.repository')}
                <Input className="mt-1" value={repo} onChange={(e) => setRepo(e.target.value)} />
              </label>
              <label className="text-[12px] text-muted">
                {t('common.agent')}
                <Input className="mt-1" value={agent} onChange={(e) => setAgent(e.target.value)} />
              </label>
              <label className="text-[12px] text-muted">
                {t('runs.taskLabel')}
                <Input
                  className="mt-1"
                  value={prompt}
                  onChange={(e) => setPrompt(e.target.value)}
                  placeholder={t('runs.taskPlaceholder')}
                />
              </label>
              {create.error && (
                <p className="text-[12px] text-err">{String(create.error.message)}</p>
              )}
              <div className="mt-1 flex justify-end gap-2">
                <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
                  {t('common.cancel')}
                </Button>
                <Button type="submit" variant="primary" disabled={!prompt || create.isPending}>
                  {create.isPending ? t('runs.starting') : t('runs.startRun')}
                </Button>
              </div>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <div className="mt-4 flex items-center gap-1 border-b border-line pb-2">
        {FILTERS.map((f) => (
          <button
            key={f.key}
            onClick={() => setFilter(f.key)}
            className={cn(
              'rounded-md px-2.5 py-1 text-[12px] transition-colors',
              filter === f.key ? 'bg-panel2 text-fg' : 'text-muted hover:text-fg',
            )}
          >
            {t(f.labelKey)}
          </button>
        ))}
        <span className="ml-auto text-[12px] text-muted">{t('runs.count', { count: runs.length })}</span>
      </div>

      {error && <p className="mt-4 text-[12px] text-err">{String(error.message)}</p>}
      {isLoading && <p className="mt-6 text-[12px] text-muted">{t('common.loading')}</p>}

      <div className="mt-2 overflow-hidden rounded-lg border border-line">
        <table className="w-full table-fixed border-collapse text-left">
          <thead>
            <tr className="bg-panel/80 text-[11px] uppercase tracking-wider text-muted/80">
              <th className="w-[110px] px-3 py-2 font-medium">{t('common.status')}</th>
              <th className="w-[120px] px-3 py-2 font-medium">{t('runs.colRun')}</th>
              <th className="px-3 py-2 font-medium">{t('common.source')}</th>
              <th className="w-[150px] px-3 py-2 font-medium">{t('runs.colRepo')}</th>
              <th className="w-[110px] px-3 py-2 font-medium">{t('common.agent')}</th>
              <th className="w-[120px] px-3 py-2 font-medium">{t('runs.colMachine')}</th>
              <th className="w-[70px] px-3 py-2 text-right font-medium">{t('runs.colAge')}</th>
            </tr>
          </thead>
          <tbody>
            {runs.map((run) => (
              <Row key={run.id} run={run} onOpen={() => navigate(`/runs/${run.id}`)} />
            ))}
          </tbody>
        </table>
        {!isLoading && runs.length === 0 && (
          <div className="flex flex-col items-center justify-center gap-2 py-16 text-muted">
            <TerminalSquare className="h-6 w-6 opacity-40" />
            <p className="text-[13px]">{t('runs.empty')}</p>
            <p className="text-[11px]">{t('runs.emptyHint')}</p>
          </div>
        )}
      </div>
    </div>
  )
}

function Row({ run, onOpen }: { run: Run; onOpen: () => void }) {
  const source = `${run.work.provider}:${run.work.id}`
  return (
    <tr
      onClick={onOpen}
      className="cursor-pointer border-t border-line text-[12.5px] transition-colors hover:bg-panel2/50"
    >
      <td className="px-3 py-2">
        <StatusBadge status={run.status} />
      </td>
      <td className="px-3 py-2 font-mono text-[11.5px] text-muted">{refId(run.id)}</td>
      <td className="truncate px-3 py-2">
        {run.work.url ? (
          <a
            href={run.work.url}
            target="_blank"
            rel="noreferrer"
            onClick={(e) => e.stopPropagation()}
            className="inline-flex items-center gap-1 text-fg/90 hover:text-accent2"
          >
            {source}
            <ArrowUpRight className="h-3 w-3 opacity-50" />
          </a>
        ) : (
          <span className="text-muted">{source}</span>
        )}
      </td>
      <td className="truncate px-3 py-2 text-muted">{run.repo || '—'}</td>
      <td className="px-3 py-2 text-muted">{run.agentId || '—'}</td>
      <td className="truncate px-3 py-2 font-mono text-[11.5px] text-muted">
        {run.machineId ? refId(run.machineId, 8) : '—'}
      </td>
      <td className="px-3 py-2 text-right text-[11.5px] text-muted">{age(run.createdAt)}</td>
    </tr>
  )
}
