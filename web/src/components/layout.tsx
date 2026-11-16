import { Link, NavLink, Outlet, useLocation } from 'react-router-dom'
import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  Activity,
  Inbox,
  ListTodo,
  ShieldCheck,
  CalendarClock,
  Bot,
  UsersRound,
  Plug,
  KeyRound,
  Search,
  Settings2,
  Monitor,
  Command,
  ChevronDown,
  ChevronRight,
  ListTree,
  ChevronsUpDown,
  Sparkles,
  Bell,
  FileText,
  MessageSquare,
  LayoutDashboard,
  History,
  ScrollText,
  MoreHorizontal,
  Menu,
  Plus,
} from 'lucide-react'
import { cn } from '../lib/utils'
import { api } from '../lib/api'
import { useWorkspace } from '../lib/workspace'
import { useI18n } from '../lib/i18n'
import CommandPalette from './command-palette'
import { Dialog, DialogContent } from './ui/dialog'
import { Button } from './ui/button'
import { Input } from './ui/input'

interface NavEntry {
  to: string
  label: string
  icon: typeof Activity
  badge?: number
  also?: string[]
  onClick?: () => void
}

function NavItem({ to, label, icon: Icon, badge, onClick }: NavEntry) {
  const location = useLocation()
  const globalScope = location.search.includes('scope=all')
  return (
    <NavLink
      to={to}
      onClick={onClick}
      className={({ isActive }) =>
        cn(
          'flex h-8 items-center gap-2 rounded-md px-2 text-[13px] text-muted transition-colors',
          isActive && !globalScope ? 'bg-panel2 text-fg' : 'hover:bg-panel2/60 hover:text-fg',
        )
      }
    >
      <Icon className="h-4 w-4" />
      {label}
      {badge !== undefined && badge > 0 && (
        <span className="ml-auto rounded-full bg-fg px-1.5 text-[9.5px] font-medium leading-4 text-bg">
          {badge > 99 ? '99+' : badge}
        </span>
      )}
    </NavLink>
  )
}

function GlobalItem({ to, label, icon: Icon, badge, also }: NavEntry) {
  const location = useLocation()
  const [path, query] = to.split('?')
  const alias = also?.includes(location.pathname) ?? false
  const active =
    (location.pathname === path || alias) &&
    (alias || (query ? location.search === `?${query}` : location.search === ''))
  return (
    <NavLink
      to={to}
      className={cn(
        'flex h-8 items-center gap-2 rounded-md px-2 text-[13px] text-muted transition-colors',
        active ? 'bg-panel2 text-fg' : 'hover:bg-panel2/60 hover:text-fg',
      )}
    >
      <Icon className="h-4 w-4" />
      {label}
      {badge !== undefined && badge > 0 && (
        <span className="ml-auto rounded-full bg-fg px-1.5 text-[9.5px] font-medium leading-4 text-bg">
          {badge > 99 ? '99+' : badge}
        </span>
      )}
    </NavLink>
  )
}

function WorkspaceMenu({ id }: { id: string }) {
  const { t } = useI18n()
  const { setWorkspaceId } = useWorkspace()
  const [open, setOpen] = useState(false)
  const ref = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (!open) return
    const onDown = (event: MouseEvent) => {
      if (ref.current && !ref.current.contains(event.target as Node)) setOpen(false)
    }
    document.addEventListener('mousedown', onDown)
    return () => document.removeEventListener('mousedown', onDown)
  }, [open])

  return (
    <div ref={ref} className="relative shrink-0">
      <button
        title={t('workspaceSettings.menu')}
        aria-label={t('workspaceSettings.menu')}
        onClick={() => setOpen((value) => !value)}
        className="flex h-7 w-6 items-center justify-center rounded text-muted hover:bg-panel2 hover:text-fg"
      >
        <MoreHorizontal className="h-3.5 w-3.5" />
      </button>
      {open && (
        <div className="absolute right-0 top-7 z-50 w-44 rounded-md border border-line bg-panel p-1 shadow-lg">
          <Link
            to={`/workspaces/${id}/settings`}
            onClick={() => {
              setOpen(false)
              setWorkspaceId(id)
            }}
            className="flex h-8 items-center gap-2 rounded px-2 text-[12px] text-muted hover:bg-panel2 hover:text-fg"
          >
            <Settings2 className="h-3.5 w-3.5" />
            {t('workspaceSettings.menu')}
          </Link>
        </div>
      )}
    </div>
  )
}

export default function Layout() {
  const location = useLocation()
  const popout = new URLSearchParams(location.search).get('popout') === '1'
  const {
    workspaceId,
    setWorkspaceId,
    workspaces,
    workspaceMode,
    setWorkspaceMode,
  } = useWorkspace()
  const { t } = useI18n()
  const qc = useQueryClient()
  const [searchOpen, setSearchOpen] = useState(false)
  const [createOpen, setCreateOpen] = useState(false)
  const [newWorkspaceName, setNewWorkspaceName] = useState('')
  const [openGroups, setOpenGroups] = useState<Record<string, boolean>>({})
  const createWorkspace = useMutation({
    mutationFn: () => api.createWorkspace({ name: newWorkspaceName.trim() }),
    onSuccess: (workspace) => {
      void qc.invalidateQueries({ queryKey: ['workspaces'] })
      setWorkspaceId(workspace.id)
      setCreateOpen(false)
      setNewWorkspaceName('')
    },
  })
  const [mobileNav, setMobileNav] = useState(false)
  useEffect(() => {
    setOpenGroups((prev) => ({ ...prev, [workspaceId]: true }))
  }, [workspaceId])
  useEffect(() => {
    setMobileNav(false)
  }, [location.pathname])
  const toggleGroup = (id: string) => {
    setOpenGroups((prev) => ({ ...prev, [id]: !(prev[id] ?? id === workspaceId) }))
  }

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault()
        setSearchOpen((open) => !open)
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  const { data: approvals } = useQuery({
    queryKey: ['approvals', 'all'],
    queryFn: () => api.approvals('all'),
    refetchInterval: 30000,
  })
  const { data: inboxAll } = useQuery({
    queryKey: ['inbox-tasks', 'all'],
    queryFn: () => api.tasks({ workspace: 'all' }),
    refetchInterval: 30000,
  })
  const unreadAll = (inboxAll?.items ?? []).filter(
    (t) => t.inboxState === 'inbox' && !t.inboxReadAt,
  ).length
  const pendingApprovals = approvals?.items.length ?? 0
  const { data: notifications } = useQuery({
    queryKey: ['notifications', 'all'],
    queryFn: () => api.notifications('all'),
    refetchInterval: 30000,
  })
  const unreadNotifications = (notifications?.notifications ?? []).filter((n) => !n.readAt).length

  const globalNav: NavEntry[] = [
    { to: '/machines', label: t('nav.machines'), icon: Monitor },
    { to: '/skills', label: t('nav.skills'), icon: Sparkles },
    { to: '/providers', label: t('nav.providers'), icon: Plug },
    { to: '/credentials', label: t('nav.credentials'), icon: KeyRound },
    { to: '/audit', label: 'Audit log', icon: ScrollText },
    { to: '/settings', label: t('nav.settings'), icon: Settings2 },
  ]

  if (popout) {
    return (
      <div className="h-screen overflow-hidden bg-bg">
        <Outlet />
      </div>
    )
  }

  return (
    <div className="flex h-screen overflow-hidden">
      <div
        onClick={() => setMobileNav(false)}
        className={cn('fixed inset-0 z-40 bg-black/50 md:hidden', mobileNav ? 'block' : 'hidden')}
      />
      <aside
        className={cn(
          'fixed inset-y-0 left-0 z-50 flex w-[236px] shrink-0 flex-col overflow-y-auto border-r border-line bg-panel transition-transform md:static md:z-auto md:translate-x-0',
          mobileNav ? 'translate-x-0' : '-translate-x-full',
        )}
      >
        <div className="flex h-12 items-center gap-2 px-4">
          <div className="flex h-5 w-5 items-center justify-center rounded bg-fg text-[11px] font-bold text-bg">
            O
          </div>
          <span className="truncate text-[13px] font-semibold tracking-tight">Tandemesh</span>
          <span className="ml-auto rounded border border-line2 px-1 text-[10px] text-muted">
            {t('shell.dev')}
          </span>
        </div>

        <div className="flex flex-col gap-1.5 px-3 pb-1">
          <div className="mt-1 flex flex-col gap-0.5">
            <div className="px-2 text-[10.5px] uppercase tracking-wider text-muted/70">
              {t('shell.personal')}
            </div>
            <GlobalItem
              to="/inbox"
              label={t('nav.myWork')}
              icon={Inbox}
              badge={unreadAll}
              also={['/my-tasks']}
            />
            <GlobalItem
              to="/approvals"
              label={t('nav.approvals')}
              icon={ShieldCheck}
              badge={pendingApprovals}
            />
            <GlobalItem
              to="/notifications"
              label={t('nav.notifications')}
              icon={Bell}
              badge={unreadNotifications}
            />
          </div>

          <div className="mt-2 flex flex-col gap-0.5">
            <div className="px-2 text-[10.5px] uppercase tracking-wider text-muted/70">
              {t('shell.allWorkspaces')}
            </div>
            <GlobalItem to="/tasks?scope=all" label={t('nav.tasks')} icon={ListTodo} />
            <GlobalItem to="/sessions?scope=all" label={t('nav.recentSessions')} icon={MessageSquare} />
            <GlobalItem to="/dashboard" label="Agent dashboard" icon={LayoutDashboard} />
            <GlobalItem to="/session-history" label="Session history" icon={History} />
            <GlobalItem to="/runs?scope=all" label={t('nav.runs')} icon={Activity} />
            <GlobalItem to="/agents?scope=all" label={t('nav.agents')} icon={Bot} />
          </div>

          <div className="flex items-center justify-between px-0.5">
            <span className="text-[10.5px] uppercase tracking-wider text-muted/70">
              {t('shell.workspaces')}
            </span>
            <div className="flex items-center gap-1">
              <button
                title="New workspace"
                aria-label="New workspace"
                onClick={() => setCreateOpen(true)}
                className="text-muted hover:text-fg"
              >
                <Plus className="h-3.5 w-3.5" />
              </button>
              <button
                title={workspaceMode === 'select' ? t('settings.list') : t('settings.dropdown')}
                onClick={() => setWorkspaceMode(workspaceMode === 'select' ? 'list' : 'select')}
                className="text-muted hover:text-fg"
              >
                {workspaceMode === 'select' ? (
                  <ListTree className="h-3.5 w-3.5" />
                ) : (
                  <ChevronsUpDown className="h-3.5 w-3.5" />
                )}
              </button>
            </div>
          </div>

          {workspaceMode === 'select' ? (
            <div className="flex items-center gap-1">
              <select
                value={workspaceId}
                onChange={(e) => setWorkspaceId(e.target.value)}
                title={t('shell.workspaces')}
                className="h-7 min-w-0 flex-1 rounded-md border border-line2 bg-panel px-2 text-[11.5px] text-fg"
              >
                {workspaces.map((w) => (
                  <option key={w.id} value={w.id}>
                    {w.name}
                  </option>
                ))}
              </select>
              <WorkspaceMenu id={workspaceId} />
            </div>
          ) : (
            <div className="flex flex-col gap-0.5">
              {workspaces.map((w) => {
                const active = w.id === workspaceId
                const open = openGroups[w.id] ?? active
                return (
                  <div key={w.id}>
                    <div className="flex items-center gap-0.5">
                      <button
                        onClick={() => {
                          toggleGroup(w.id)
                          if (!open) setWorkspaceId(w.id)
                        }}
                        className={cn(
                          'flex h-7 min-w-0 flex-1 items-center gap-1.5 rounded-md px-1 text-[12.5px]',
                          active ? 'bg-panel2 text-fg' : 'text-muted hover:bg-panel2/50 hover:text-fg',
                        )}
                      >
                        {open ? (
                          <ChevronDown className="h-3.5 w-3.5 shrink-0" />
                        ) : (
                          <ChevronRight className="h-3.5 w-3.5 shrink-0" />
                        )}
                        <span className="text-muted/70">◐</span>
                        <span className="truncate">{w.name}</span>
                      </button>
                      <WorkspaceMenu id={w.id} />
                    </div>
                    {open && (
                      <div className="ml-3 border-l border-line2 pl-1">
                        <WorkspaceNav onNavigate={() => setWorkspaceId(w.id)} />
                      </div>
                    )}
                  </div>
                )
              })}
            </div>
          )}

          <button
            onClick={() => setSearchOpen(true)}
            className="mt-0.5 flex h-7 w-full items-center gap-2 rounded-md border border-line2 bg-panel px-2 text-[11.5px] text-muted hover:text-fg"
          >
            <Search className="h-3.5 w-3.5" />
            {t('shell.search')}
            <span className="ml-auto text-[10px] text-muted/60">⌘K</span>
          </button>
        </div>

        {workspaceMode === 'select' && (
          <WorkspaceNav />
        )}

        <div className="mt-auto border-t border-line pt-2">
          <div className="px-2">
            {globalNav.map((item) => (
              <NavItem key={item.to} {...item} />
            ))}
          </div>
          <div className="flex items-center gap-2 px-4 py-3 text-[11px] text-muted">
            <Command className="h-3.5 w-3.5" />
            <span>orca 0.1.0-dev</span>
          </div>
        </div>
      </aside>

      <main className="min-h-0 min-w-0 flex-1 overflow-y-auto">
        <div className="sticky top-0 z-30 flex h-11 items-center gap-2 border-b border-line bg-bg px-3 md:hidden">
          <button onClick={() => setMobileNav(true)} className="text-muted hover:text-fg" aria-label="Open navigation">
            <Menu className="h-5 w-5" />
          </button>
          <span className="text-[13px] font-semibold tracking-tight">Tandemesh</span>
        </div>
        <Outlet />
      </main>
      <CommandPalette open={searchOpen} onOpenChange={setSearchOpen} />
      <Dialog open={createOpen} onOpenChange={setCreateOpen}>
        <DialogContent title="New workspace" description="Create a workspace for a separate project or team.">
          <form
            className="flex flex-col gap-3"
            onSubmit={(e) => {
              e.preventDefault()
              if (newWorkspaceName.trim()) createWorkspace.mutate()
            }}
          >
            <Input
              autoFocus
              value={newWorkspaceName}
              maxLength={80}
              placeholder="Workspace name"
              onChange={(e) => setNewWorkspaceName(e.target.value)}
            />
            {createWorkspace.isError && (
              <p className="text-[11px] text-err">{createWorkspace.error.message}</p>
            )}
            <div className="flex justify-end">
              <Button
                type="submit"
                variant="primary"
                disabled={!newWorkspaceName.trim() || createWorkspace.isPending}
              >
                Create
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}

function WorkspaceNav({ onNavigate }: { onNavigate?: () => void }) {
  const { t } = useI18n()
  return (
    <div className="mt-1 flex flex-col gap-0.5 px-2">
      <NavItem to="/tasks" label={t('nav.tasks')} icon={ListTodo} onClick={onNavigate} />
      <NavItem to="/sessions" label={t('nav.sessions')} icon={MessageSquare} onClick={onNavigate} />
      <NavItem to="/iterations" label={t('nav.iterations')} icon={CalendarClock} onClick={onNavigate} />
      <NavItem to="/groups" label={t('nav.groups')} icon={UsersRound} onClick={onNavigate} />
      <NavItem to="/agents" label={t('nav.agents')} icon={Bot} onClick={onNavigate} />
      <NavItem to="/automations" label={t('nav.automations')} icon={Activity} onClick={onNavigate} />
      <NavItem to="/runs" label={t('nav.runs')} icon={Activity} onClick={onNavigate} />
      <NavItem to="/documents" label={t('nav.documents')} icon={FileText} onClick={onNavigate} />
    </div>
  )
}
