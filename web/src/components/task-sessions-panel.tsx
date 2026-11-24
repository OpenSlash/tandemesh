import { useQuery } from '@tanstack/react-query'
import { MessageSquare } from 'lucide-react'
import { api } from '../lib/api'
import { useI18n } from '../lib/i18n'

export function TaskSessionsPanel({ taskId, workspaceId }: { taskId: string; workspaceId: string }) {
  const { t } = useI18n()
  const { data } = useQuery({
    queryKey: ['task-sessions', taskId],
    queryFn: () => api.sessions(workspaceId, taskId),
    refetchInterval: 5000,
  })
  const sessions = data?.sessions ?? []
  if (sessions.length === 0) return null

  return (
    <section className="rounded-lg border border-line bg-panel/20 p-3">
      <div className="flex items-center gap-2">
        <MessageSquare className="h-3.5 w-3.5 text-muted" />
        <h2 className="text-[12px] uppercase tracking-wider text-muted">{t('nav.sessions')}</h2>
        <span className="ml-auto text-[10px] text-muted">{sessions.length}</span>
      </div>
      <div className="mt-2 space-y-1">
        {sessions.map((session) => (
          <a key={session.id} href={`/sessions?session=${session.id}`} className="flex items-center gap-2 rounded border border-line2 px-2 py-1 text-[11px] hover:bg-panel2/50">
            <span className="truncate">{session.title || session.id}</span>
            <span className="ml-auto text-[10px] text-muted">{session.lastRunStatus ?? session.status}</span>
          </a>
        ))}
      </div>
    </section>
  )
}
