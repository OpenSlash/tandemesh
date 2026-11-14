import { useState } from 'react'
import { useQuery, useQueryClient } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import { Bell } from 'lucide-react'
import { api } from '../lib/api'
import { useWorkspace } from '../lib/workspace'
import { useI18n } from '../lib/i18n'
import { Button } from '../components/ui/button'

export default function NotificationsPage() {
  const { t } = useI18n()
  const qc = useQueryClient()
  const { workspaces } = useWorkspace()
  const [scope, setScope] = useState('all')
  const { data } = useQuery({
    queryKey: ['notifications', scope],
    queryFn: () => api.notifications(scope),
    refetchInterval: 15000,
  })
  const items = data?.notifications ?? []
  const unread = items.filter((item) => !item.readAt).length

  const refresh = () => qc.invalidateQueries({ queryKey: ['notifications', scope] })

  return (
    <div className="mx-auto max-w-[900px] px-6 py-5">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-[16px] font-semibold">{t('nav.notifications')}</h1>
          <p className="mt-0.5 text-[12px] text-muted">
            {unread > 0 ? `${unread} unread` : t('notifications.empty')}
          </p>
        </div>
        <div className="flex items-center gap-2">
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
          <Button
            size="sm"
            variant="ghost"
            disabled={unread === 0}
            onClick={async () => {
              await api.markAllNotificationsRead(scope)
              refresh()
            }}
          >
            {t('notifications.markAll')}
          </Button>
        </div>
      </div>

      <div className="mt-4 flex flex-col gap-2">
        {items.length === 0 && (
          <div className="rounded-lg border border-line bg-panel px-4 py-10 text-center text-[13px] text-muted">
            <Bell className="mx-auto mb-2 h-5 w-5 opacity-50" />
            {t('notifications.empty')}
          </div>
        )}
        {items.map((item) => (
          <div
            key={item.id}
            className={`flex items-start gap-3 rounded-lg border px-4 py-3 ${
              item.readAt ? 'border-line bg-panel/50' : 'border-accent/40 bg-panel'
            }`}
          >
            <span
              className={`mt-1.5 h-1.5 w-1.5 shrink-0 rounded-full ${
                item.readAt ? 'bg-transparent' : 'bg-fg'
              }`}
            />
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <span className="text-[13px] font-medium">{item.title}</span>
                <span className="text-[10.5px] uppercase text-muted/60">{item.type}</span>
              </div>
              {item.body && <p className="mt-0.5 whitespace-pre-wrap text-[12px] text-muted">{item.body}</p>}
              <div className="mt-1 flex items-center gap-3 text-[11px] text-muted/60">
                <span>{new Date(item.createdAt).toLocaleString()}</span>
                {item.link && (
                  <Link to={item.link} className="text-fg hover:underline">
                    {t('notifications.open')}
                  </Link>
                )}
              </div>
            </div>
            {!item.readAt && (
              <Button
                size="sm"
                variant="ghost"
                onClick={async () => {
                  await api.markNotificationRead(item.id)
                  refresh()
                }}
              >
                {t('notifications.markRead')}
              </Button>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
