import { useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { ExternalLink, Search } from 'lucide-react'
import { api } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { Input } from '../components/ui/input'

export default function ACPRegistryPage() {
  const { t } = useI18n()
  const [query, setQuery] = useState('')
  const [status, setStatus] = useState<Record<string, string>>({})
  const { data, isLoading, error } = useQuery({
    queryKey: ['acp-registry', query],
    queryFn: () => api.acpRegistry(query),
    staleTime: 5 * 60 * 1000,
  })
  const items = data?.items ?? []
  const install = async (id: string) => {
    setStatus((current) => ({ ...current, [id]: 'installing' }))
    try {
      const result = await api.installACP(id)
      setStatus((current) => ({ ...current, [id]: result.command }))
    } catch (err) {
      setStatus((current) => ({ ...current, [id]: String(err) }))
    }
  }

  return (
    <div className="mx-auto max-w-[980px] px-8 py-6">
      <h1 className="text-[18px] font-semibold tracking-tight">{t('acp.title')}</h1>
      <p className="mt-1 text-[12px] text-muted">{t('acp.subtitle')}</p>
      <div className="relative mt-4 max-w-[420px]">
        <Search className="pointer-events-none absolute left-2 top-2 h-3.5 w-3.5 text-muted" />
        <Input className="h-8 pl-7" placeholder={t('acp.search')} value={query} onChange={(event) => setQuery(event.target.value)} />
      </div>
      {isLoading && <p className="mt-5 text-[12px] text-muted">{t('acp.loading')}</p>}
      {error && <p className="mt-5 text-[12px] text-err">{String(error)}</p>}
      <div className="mt-4 divide-y divide-line2 rounded-lg border border-line2">
        {items.map((item) => (
          <div key={item.id} className="flex items-start gap-3 px-3 py-2.5">
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <span className="text-[12.5px] font-medium">{item.name}</span>
                <span className="font-mono text-[10px] text-muted">{item.id}</span>
                <span className="text-[10px] text-muted">v{item.version}</span>
              </div>
              <p className="mt-0.5 line-clamp-2 text-[11px] text-muted">{item.description}</p>
            </div>
            <div className="flex shrink-0 flex-col items-end gap-1">
              <button
                type="button"
                className="rounded border border-line2 px-2 py-0.5 text-[10.5px] text-muted hover:text-fg"
                disabled={status[item.id] === 'installing'}
                onClick={() => void install(item.id)}
              >
                {status[item.id] === 'installing' ? t('acp.installing') : t('acp.install')}
              </button>
              {status[item.id] && status[item.id] !== 'installing' && (
                <span className="max-w-[220px] truncate font-mono text-[9px] text-muted" title={status[item.id]}>{status[item.id]}</span>
              )}
            </div>
            {(item.repository || item.website) && (
              <a
                className="mt-0.5 inline-flex items-center gap-1 text-[10.5px] text-accent2 hover:underline"
                href={item.repository || item.website}
                target="_blank"
                rel="noreferrer"
              >
                <ExternalLink className="h-3 w-3" /> {t('acp.open')}
              </a>
            )}
          </div>
        ))}
        {!isLoading && items.length === 0 && <p className="px-3 py-5 text-[12px] text-muted">{t('acp.empty')}</p>}
      </div>
    </div>
  )
}
