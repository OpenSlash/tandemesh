import { useEffect, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { ListTodo, Search, UsersRound } from 'lucide-react'
import { api } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { useWorkspace } from '../lib/workspace'
import { Dialog, DialogContent } from './ui/dialog'
import { Input } from './ui/input'

export default function CommandPalette({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const { t } = useI18n()
  const navigate = useNavigate()
  const { workspaceId } = useWorkspace()
  const [query, setQuery] = useState('')
  const [debounced, setDebounced] = useState('')

  useEffect(() => {
    const timer = setTimeout(() => setDebounced(query.trim()), 200)
    return () => clearTimeout(timer)
  }, [query])

  const { data, isFetching } = useQuery({
    queryKey: ['search', workspaceId, debounced],
    queryFn: () => api.search(debounced, workspaceId),
    enabled: open && debounced.length > 1,
  })

  const go = (path: string) => {
    onOpenChange(false)
    setQuery('')
    navigate(path)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent title={t('shell.search')} description={t('palette.subtitle')}>
        <div className="flex items-center gap-2">
          <Search className="h-3.5 w-3.5 text-muted" />
          <Input
            autoFocus
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            placeholder={t('palette.placeholder')}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && data?.tasks?.[0]) go(`/tasks/${data.tasks[0].id}`)
            }}
          />
        </div>

        {isFetching && <p className="mt-3 text-[11.5px] text-muted">{t('palette.searching')}</p>}

        <div className="mt-3 max-h-[360px] space-y-3 overflow-y-auto">
          {(data?.tasks ?? []).length > 0 && (
            <section>
              <h3 className="text-[10.5px] uppercase tracking-wider text-muted/70">
                {t('page.tasks')}
              </h3>
              {(data?.tasks ?? []).map((task) => (
                <button
                  key={task.id}
                  onClick={() => go(`/tasks/${task.id}`)}
                  className="mt-1 flex w-full items-center gap-2 rounded px-2 py-1.5 text-left hover:bg-panel2"
                >
                  <ListTodo className="h-3.5 w-3.5 text-muted" />
                  <span className="min-w-0 flex-1 truncate text-[12.5px]">{task.title}</span>
                  <span className="font-mono text-[10px] text-muted">{task.workId}</span>
                </button>
              ))}
            </section>
          )}
          {(data?.groups ?? []).length > 0 && (
            <section>
              <h3 className="text-[10.5px] uppercase tracking-wider text-muted/70">
                {t('page.groups')}
              </h3>
              {(data?.groups ?? []).map((group) => (
                <button
                  key={group.id}
                  onClick={() => go('/groups')}
                  className="mt-1 flex w-full items-center gap-2 rounded px-2 py-1.5 text-left hover:bg-panel2"
                >
                  <UsersRound className="h-3.5 w-3.5 text-muted" />
                  <span className="min-w-0 flex-1 truncate text-[12.5px]">{group.name}</span>
                </button>
              ))}
            </section>
          )}
          {debounced.length > 1 && !isFetching && (data?.tasks ?? []).length === 0 && (data?.groups ?? []).length === 0 && (
            <p className="py-6 text-center text-[12px] text-muted">{t('palette.noMatches')}</p>
          )}
        </div>
      </DialogContent>
    </Dialog>
  )
}
