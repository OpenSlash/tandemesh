import { useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import { ListTodo } from 'lucide-react'
import { api } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { useWorkspace } from '../lib/workspace'
import { cn, compactId } from '../lib/utils'
import { statusLabel } from '../components/status'
import { WorkspaceFilter } from '../components/workspace-filter'
import { MyWorkTabs } from '../components/my-work-tabs'

export default function MyTasksPage() {
  const { t } = useI18n()
  const { workspaces } = useWorkspace()
  const [searchParams] = useSearchParams()
  const scope = searchParams.get('workspace') || 'all'
  const [view, setView] = useState<'active' | 'all'>('active')
  const { data: me } = useQuery({ queryKey: ['me'], queryFn: api.me })
  const name = me?.name ?? ''

  const { data, isLoading } = useQuery({
    queryKey: ['my-tasks', scope, name],
    queryFn: () => api.tasks({ workspace: scope, member: name }),
    enabled: !!name,
  })
  const items = data?.items ?? []
  const activeCount = items.filter((task) => task.status !== 'done').length
  const visible = view === 'active' ? items.filter((task) => task.status !== 'done') : items
  const workspaceName = workspaces.find((workspace) => workspace.id === scope)?.name

  return (
    <div className="mx-auto max-w-[1180px] px-6 py-5">
      <MyWorkTabs current="assigned" />
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-[18px] font-semibold tracking-tight">{t('page.myTasks')}</h1>
          <p className="mt-0.5 text-[12px] text-muted">
            {t('myTasks.assignedTo')} <span className="text-fg">{name || t('common.you')}</span>{' '}
            {scope === 'all'
              ? t('myTasks.acrossWorkspaces')
              : t('myTasks.inWorkspace', { workspace: workspaceName ?? t('page.workspace') })}
          </p>
        </div>
        <WorkspaceFilter />
      </div>

      <div className="mt-3 flex items-center gap-1 border-b border-line">
        {(
          [
            { value: 'active', label: t('myTasks.active'), count: activeCount },
            { value: 'all', label: t('myTasks.all'), count: items.length },
          ] as const
        ).map((tab) => (
          <button
            key={tab.value}
            onClick={() => setView(tab.value)}
            className={cn(
              '-mb-px flex items-center gap-1.5 border-b-2 px-2 pb-2 text-[12px]',
              view === tab.value
                ? 'border-accent2 text-fg'
                : 'border-transparent text-muted hover:text-fg',
            )}
          >
            {tab.label}
            <span className="rounded bg-panel2 px-1 text-[10px] text-muted">{tab.count}</span>
          </button>
        ))}
      </div>

      {isLoading && <p className="mt-6 text-[12px] text-muted">{t('common.loading')}</p>}

      <div className="mt-4 overflow-hidden rounded-lg border border-line">
        <table className="w-full text-left">
          <thead>
            <tr className="bg-panel/80 text-[11px] uppercase tracking-wider text-muted/80">
              <th className="px-3 py-2 font-medium">{t('tasks.colTask')}</th>
              <th className="px-3 py-2 font-medium">{t('tasks.colSource')}</th>
              <th className="px-3 py-2 font-medium">{t('common.status')}</th>
              <th className="px-3 py-2 font-medium">{t('tasks.colAgents')}</th>
              <th className="px-3 py-2 font-medium">{t('common.updated')}</th>
            </tr>
          </thead>
          <tbody>
            {visible.map((task) => (
              <tr key={task.id} className="border-t border-line text-[12.5px] hover:bg-panel/40">
                <td className="px-3 py-2">
                  <Link to={`/tasks/${task.id}`} className="flex items-center gap-2 hover:text-accent2">
                    <ListTodo className="h-3.5 w-3.5 text-muted" />
                    <span className="truncate">{task.title || compactId(task.id, 10)}</span>
                  </Link>
                </td>
                <td className="px-3 py-2 text-muted">{task.provider}</td>
                <td className="px-3 py-2">
                  <span className="rounded border border-line2 px-1.5 py-0.5 text-[10.5px] text-muted">
                    {statusLabel(t, task.status)}
                  </span>
                </td>
                <td className="px-3 py-2 text-muted">
                  {t('tasks.runsSummaryShort', {
                    succeeded: task.succeededRuns,
                    active: task.activeRuns,
                  })}
                </td>
                <td className="px-3 py-2 text-[11.5px] text-muted">
                  {new Date(task.updatedAt).toLocaleString()}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!isLoading && visible.length === 0 && (
          <p className="px-3 py-12 text-center text-[12px] text-muted">
            {view === 'active' ? t('myTasks.emptyActive') : t('myTasks.empty')}
          </p>
        )}
      </div>
    </div>
  )
}
