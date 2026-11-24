import { useI18n } from '../lib/i18n'
import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { ExternalLink, Inbox } from 'lucide-react'
import { api, type WorkItem } from '../lib/api'
import { useWorkspace } from '../lib/workspace'
import { cn } from '../lib/utils'
import { Input } from '../components/ui/input'
import { Button } from '../components/ui/button'

const PROVIDERS = [
  { id: 'github', label: 'GitHub' },
  { id: 'linear', label: 'Linear' },
  { id: 'jira', label: 'Jira' },
]

export default function DiscoverPage() {
  const { t } = useI18n()
  const navigate = useNavigate()
  const qc = useQueryClient()
  const { workspaceId } = useWorkspace()
  const [provider, setProvider] = useState('github')
  const [instance, setInstance] = useState('')
  const [synced, setSynced] = useState<number | null>(null)

  const { data, isLoading, error } = useQuery({
    queryKey: ['work', provider, instance],
    queryFn: () => api.work(provider, provider === 'jira' ? instance : undefined),
    enabled: provider !== 'jira' || instance.length > 0,
    refetchInterval: 60000,
  })
  const items = data?.items ?? []

  const importItem = useMutation({
    mutationFn: (item: WorkItem) => api.importWork(item, workspaceId),
    onSuccess: (task) => navigate(`/tasks/${task.id}`),
  })
  const sync = useMutation({
    mutationFn: () => api.workSync(provider, provider === 'jira' ? instance : undefined, workspaceId),
    onSuccess: (result) => {
      setSynced(result.synced)
      qc.invalidateQueries({ queryKey: ['tasks'] })
    },
  })

  return (
    <div className="mx-auto max-w-[1180px] px-6 py-5">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-[18px] font-semibold tracking-tight">{t('page.discover')}</h1>
          <p className="mt-0.5 text-[12px] text-muted">{t('discover.subtitle')}</p>
        </div>
        <div className="flex items-center gap-2">
          {synced !== null && (
            <span className="text-[11px] text-ok">{t('discover.synced', { count: synced })}</span>
          )}
          <Button
            size="sm"
            variant="ghost"
            onClick={() => sync.mutate()}
            disabled={sync.isPending || (provider === 'jira' && !instance)}
          >
            {t('discover.syncToTasks')}
          </Button>
          <div className="flex items-center gap-1 rounded-md border border-line bg-panel/60 p-0.5">
            {PROVIDERS.map((p) => (
              <button
                key={p.id}
                onClick={() => {
                  setProvider(p.id)
                  setSynced(null)
                }}
                className={cn(
                  'rounded px-2 py-1 text-[11.5px]',
                  provider === p.id ? 'bg-panel2 text-fg' : 'text-muted hover:text-fg',
                )}
              >
                {p.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      {provider === 'jira' && (
        <div className="mt-3 flex max-w-md items-center gap-2">
          <Input
            value={instance}
            onChange={(e) => setInstance(e.target.value)}
            placeholder="https://acme.atlassian.net"
          />
        </div>
      )}

      {error && <p className="mt-4 text-[12px] text-err">{String(error.message)}</p>}
      {isLoading && <p className="mt-6 text-[12px] text-muted">{t('common.loading')}</p>}

      <div className="mt-4 overflow-hidden rounded-lg border border-line">
        {items.map((item) => (
          <div
            key={`${item.ref.provider}/${item.ref.instance}/${item.ref.id}`}
            className="flex items-center justify-between gap-3 border-b border-line px-3 py-2.5 last:border-0 hover:bg-panel/30"
          >
            <div className="min-w-0">
              <div className="flex items-center gap-2">
                <Inbox className="h-3.5 w-3.5 shrink-0 text-muted" />
                <span className="truncate text-[12.5px]">{item.title}</span>
                <span className="shrink-0 font-mono text-[10.5px] text-muted">{item.ref.id}</span>
              </div>
              <div className="mt-0.5 flex flex-wrap items-center gap-2 text-[10.5px] text-muted">
                <span>{item.ref.instance || item.ref.provider}</span>
                {item.assignee && <span>· {item.assignee}</span>}
                {item.ref.url && (
                  <a
                    href={item.ref.url}
                    target="_blank"
                    rel="noreferrer"
                    className="inline-flex items-center gap-1 text-accent2 hover:underline"
                  >
                    {t('common.source')} <ExternalLink className="h-2.5 w-2.5" />
                  </a>
                )}
              </div>
              {(item.labels ?? []).length > 0 && (
                <div className="mt-1 flex flex-wrap gap-1">
                  {(item.labels ?? []).slice(0, 3).map((label) => (
                    <span key={label} className="rounded bg-accent/10 px-1.5 py-0.5 text-[10px] text-accent2">
                      {label}
                    </span>
                  ))}
                </div>
              )}
            </div>
            <Button
              size="sm"
              variant="ghost"
              onClick={() => importItem.mutate(item)}
              disabled={importItem.isPending && importItem.variables?.ref.id === item.ref.id}
            >
              {t('discover.addToTasks')}
            </Button>
          </div>
        ))}
        {!isLoading && items.length === 0 && (
          <p className="px-3 py-12 text-center text-[12px] text-muted">{t('discover.empty')}</p>
        )}
      </div>
    </div>
  )
}
