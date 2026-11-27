import { useI18n } from '../lib/i18n'
import { useEffect, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CheckCheck, Inbox, Search, Trash2, UserRound } from 'lucide-react'
import { api, type Task } from '../lib/api'
import { useWorkspace } from '../lib/workspace'
import { cn, compactId } from '../lib/utils'
import { WorkspaceFilter } from '../components/workspace-filter'
import { MyWorkTabs } from '../components/my-work-tabs'

const TABS = [
  { value: 'inbox', labelKey: 'inbox.tabInbox' },
  { value: 'later', labelKey: 'inbox.tabLater' },
  { value: 'cleared', labelKey: 'inbox.tabCleared' },
]

export default function InboxPage() {
  const { t } = useI18n()
  const { workspaces } = useWorkspace()
  const [searchParams] = useSearchParams()
  const scope = searchParams.get('workspace') || 'all'
  const qc = useQueryClient()
  const [tab, setTab] = useState('inbox')
  const [onlyMine, setOnlyMine] = useState(false)

  const { data: me } = useQuery({ queryKey: ['me'], queryFn: api.me })
  const { data, isLoading } = useQuery({
    queryKey: ['inbox-tasks', scope],
    queryFn: () => api.tasks({ workspace: scope }),
    refetchInterval: 30000,
  })
  const all = data?.items ?? []
  // assigneeMemberId resolves the platform assignee through my linked accounts.
  const base = onlyMine ? all.filter((task) => task.assigneeMemberId === me?.name) : all
  const items = base.filter((task) => task.inboxState === tab)
  const count = (state: string) => base.filter((task) => task.inboxState === state).length
  const currentTab = TABS.find((entry) => entry.value === tab)
  const workspaceName = workspaces.find((workspace) => workspace.id === scope)?.name

  const move = useMutation({
    mutationFn: ({ id, state }: { id: string; state: 'inbox' | 'later' | 'cleared' }) =>
      api.setTaskInbox(id, state),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['inbox-tasks', scope] }),
  })
  const read = useMutation({ mutationFn: (id: string) => api.markTaskRead(id) })
  const [selected, setSelected] = useState<Set<string>>(new Set())
  useEffect(() => setSelected(new Set()), [tab, scope, onlyMine])
  const batch = useMutation({
    mutationFn: (body: Parameters<typeof api.batchTaskInbox>[0]) => api.batchTaskInbox(body),
    onSuccess: () => {
      setSelected(new Set())
      void qc.invalidateQueries({ queryKey: ['inbox-tasks'] })
    },
  })
  const selectedIds = items.filter((task) => selected.has(task.id)).map((task) => task.id)
  const allSelected = items.length > 0 && selectedIds.length === items.length
  const toggle = (id: string) =>
    setSelected((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  const unreadIds = items.filter((task) => !task.inboxReadAt).map((task) => task.id)
  const closedCount = items.filter((task) => task.status === 'done' || task.status === 'cancelled').length
  const batchButton = 'inline-flex h-7 items-center gap-1 rounded-md border border-line px-2 text-[11px] text-muted hover:text-fg disabled:opacity-40'

  return (
    <div className="mx-auto max-w-[1180px] px-6 py-5">
      <MyWorkTabs current="inbox" />
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-[18px] font-semibold tracking-tight">{t('page.inbox')}</h1>
          <p className="mt-0.5 text-[12px] text-muted">
            {scope === 'all'
              ? t('inbox.subtitle')
              : t('inbox.subtitleWorkspace', { workspace: workspaceName ?? t('page.workspace') })}
          </p>
        </div>
        <div className="flex items-center gap-1.5">
          <WorkspaceFilter />
          <button
            onClick={() => setOnlyMine((value) => !value)}
            className={cn(
              'inline-flex h-7 items-center gap-1.5 rounded-md border px-2 text-[11.5px]',
              onlyMine
                ? 'border-accent2/60 bg-panel2 text-fg'
                : 'border-line text-muted hover:text-fg',
            )}
          >
            <UserRound className="h-3.5 w-3.5" />
            {t('inbox.onlyMine')}
          </button>
          <Link
            to="/discover"
            className="inline-flex items-center gap-1.5 rounded-md border border-line px-2 py-1 text-[11.5px] text-muted hover:text-fg"
          >
            <Search className="h-3.5 w-3.5" /> {t('inbox.discoverAssigned')}
          </Link>
        </div>
      </div>

      <div className="mt-4 flex items-center gap-4 border-b border-line">
        {TABS.map((entry) => (
          <button
            key={entry.value}
            onClick={() => setTab(entry.value)}
            className={cn(
              '-mb-px flex items-center gap-1.5 border-b-2 px-0.5 pb-2 text-[12px]',
              tab === entry.value
                ? 'border-accent2 text-fg'
                : 'border-transparent text-muted hover:text-fg',
            )}
          >
            {t(entry.labelKey)}
            <span className="rounded bg-panel2 px-1 text-[10px] text-muted">{count(entry.value)}</span>
          </button>
        ))}
      </div>

      {items.length > 0 && (
        <div className="mt-2 flex flex-wrap items-center gap-2 px-1 py-1.5">
          <input
            type="checkbox"
            aria-label={t('inbox.selectAll')}
            checked={allSelected}
            onChange={() => setSelected(allSelected ? new Set() : new Set(items.map((task) => task.id)))}
          />
          {selectedIds.length > 0 ? (
            <>
              <span className="text-[11.5px] text-fg">{t('inbox.selected', { count: selectedIds.length })}</span>
              <button className={batchButton} disabled={batch.isPending} onClick={() => batch.mutate({ ids: selectedIds, read: true })}>
                <CheckCheck className="h-3.5 w-3.5" /> {t('inbox.markRead')}
              </button>
              {tab !== 'later' && (
                <button className={batchButton} disabled={batch.isPending} onClick={() => batch.mutate({ ids: selectedIds, state: 'later' })}>
                  {t('inbox.actionLater')}
                </button>
              )}
              {tab !== 'cleared' && (
                <button className={batchButton} disabled={batch.isPending} onClick={() => batch.mutate({ ids: selectedIds, state: 'cleared' })}>
                  {t('inbox.actionClear')}
                </button>
              )}
              {tab !== 'inbox' && (
                <button className={batchButton} disabled={batch.isPending} onClick={() => batch.mutate({ ids: selectedIds, state: 'inbox' })}>
                  {t('inbox.actionRestore')}
                </button>
              )}
              <button className="text-[11px] text-muted hover:text-fg" onClick={() => setSelected(new Set())}>
                {t('inbox.clearSelection')}
              </button>
            </>
          ) : (
            <>
              <span className="text-[11.5px] text-muted">{t('inbox.selectHint')}</span>
              <div className="ml-auto flex items-center gap-2">
                {unreadIds.length > 0 && (
                  <button className={batchButton} disabled={batch.isPending} onClick={() => batch.mutate({ ids: unreadIds, read: true })}>
                    <CheckCheck className="h-3.5 w-3.5" /> {t('inbox.markAllRead')}
                  </button>
                )}
                {tab !== 'cleared' && closedCount > 0 && (
                  <button
                    className={batchButton}
                    disabled={batch.isPending}
                    title={t('inbox.clearClosedHint')}
                    onClick={() => batch.mutate({ closed: true, workspace: scope, from: tab as 'inbox' | 'later', state: 'cleared' })}
                  >
                    <Trash2 className="h-3.5 w-3.5" /> {t('inbox.clearClosed')}
                  </button>
                )}
              </div>
            </>
          )}
          {batch.error && <span className="text-[11px] text-err">{batch.error.message}</span>}
        </div>
      )}

      {isLoading && <p className="mt-6 text-[12px] text-muted">{t('common.loading')}</p>}

      <div className="mt-2">
        {items.map((task) => (
          <InboxRow
            key={task.id}
            task={task}
            selected={selected.has(task.id)}
            selecting={selected.size > 0}
            onToggle={() => toggle(task.id)}
            onOpen={() => {
              if (!task.inboxReadAt) read.mutate(task.id)
            }}
            onMove={(state) => move.mutate({ id: task.id, state })}
          />
        ))}
        {!isLoading && items.length === 0 && (
          <p className="py-12 text-center text-[12px] text-muted">
            {onlyMine
              ? t('inbox.emptyMine')
              : t('inbox.empty', {
                  tab: currentTab ? t(currentTab.labelKey).toLowerCase() : '',
                })}
          </p>
        )}
      </div>
    </div>
  )
}

function InboxRow({
  task,
  selected,
  selecting,
  onToggle,
  onOpen,
  onMove,
}: {
  task: Task
  selected: boolean
  selecting: boolean
  onToggle: () => void
  onOpen: () => void
  onMove: (state: 'inbox' | 'later' | 'cleared') => void
}) {
  const { t } = useI18n()
  const unread = !task.inboxReadAt && task.inboxState === 'inbox'
  return (
    <div className={cn('group flex items-center gap-3 border-b border-line px-1 py-2.5', selected && 'bg-panel2/60')}>
      <input
        type="checkbox"
        aria-label={task.title}
        checked={selected}
        onChange={onToggle}
        className={cn('shrink-0', selecting || selected ? 'opacity-100' : 'opacity-0 group-hover:opacity-100')}
      />
      <span
        className={cn(
          'h-1.5 w-1.5 shrink-0 rounded-full',
          unread ? 'bg-fg' : 'bg-transparent',
        )}
      />
      <Link
        to={`/tasks/${task.id}`}
        onClick={onOpen}
        className="flex min-w-0 flex-1 items-center gap-2"
      >
        <Inbox className="h-3.5 w-3.5 shrink-0 text-muted" />
        <span className="truncate text-[12.5px]">{task.title || compactId(task.id, 10)}</span>
        <span className="shrink-0 font-mono text-[10.5px] text-muted">
          {task.provider} · {task.workId}
        </span>
        {task.readyToClose && (
          <span className="shrink-0 rounded bg-ok/15 px-1.5 py-0.5 text-[9.5px] text-ok">
            {t('common.ready')}
          </span>
        )}
      </Link>
      <span className="hidden shrink-0 text-[10.5px] text-muted sm:block">
        {new Date(task.updatedAt).toLocaleDateString()}
      </span>
      <div className="flex shrink-0 items-center gap-1 opacity-0 transition-opacity group-hover:opacity-100">
        {task.inboxState !== 'later' && (
          <button
            onClick={() => onMove('later')}
            className="rounded px-1.5 py-0.5 text-[10.5px] text-muted hover:text-fg"
          >
            {t('inbox.actionLater')}
          </button>
        )}
        {task.inboxState !== 'cleared' ? (
          <button
            onClick={() => onMove('cleared')}
            className="rounded px-1.5 py-0.5 text-[10.5px] text-muted hover:text-fg"
          >
            {t('inbox.actionClear')}
          </button>
        ) : (
          <button
            onClick={() => onMove('inbox')}
            className="rounded px-1.5 py-0.5 text-[10.5px] text-muted hover:text-fg"
          >
            {t('inbox.actionRestore')}
          </button>
        )}
      </div>
    </div>
  )
}
