import { useEffect, useState } from 'react'
import { Route, Routes, Navigate } from 'react-router-dom'
import { WorkspaceProvider } from './lib/workspace'
import { I18nProvider, useI18n } from './lib/i18n'
import Layout from './components/layout'
import InboxPage from './pages/inbox'
import DiscoverPage from './pages/discover'
import MyTasksPage from './pages/my-tasks'
import TasksPage from './pages/tasks'
import TaskDetailPage from './pages/task-detail'
import RunsPage from './pages/runs'
import RunDetailPage from './pages/run-detail'
import ApprovalsPage from './pages/approvals'
import AutomationsPage from './pages/automations'
import WorkflowsPage from './pages/workflows'
import WorkflowEditorPage from './pages/workflow-editor-page'
import UsagePage from './pages/usage'
import AgentsPage from './pages/agents'
import GroupsPage from './pages/groups'
import WorkspacePage from './pages/workspace'
import WorkspaceSettingsPage from './pages/workspace-settings'
import IterationsPage from './pages/iterations'
import ProvidersPage from './pages/providers'
import CredentialsPage from './pages/credentials'
import SettingsLayout from './pages/settings'
import SettingsAppearancePage from './pages/settings-appearance'
import SettingsAccountPage from './pages/settings-account'
import SettingsMobilePage from './pages/settings-mobile'
import ACPRegistryPage from './pages/acp-registry'
import SkillsPage from './pages/skills'
import MachinesPage from './pages/machines'
import MachineDetailPage from './pages/machine-detail'
import NotificationsPage from './pages/notifications'
import DocumentsPage from './pages/documents'
import SessionsPage from './pages/sessions'
import AgentDashboardPage from './pages/agent-dashboard'
import SessionHistoryPage from './pages/session-history'
import SessionDetailPage from './pages/session-detail'
import AuditLogPage from './pages/audit-log'
import SharedSessionPage from './pages/shared-session'

export default function App() {
  const [auth, setAuth] = useState<'loading' | 'ok' | 'login'>('loading')

  useEffect(() => {
    fetch('/auth/me', { credentials: 'same-origin' })
      .then((resp) => setAuth(resp.ok ? 'ok' : 'login'))
      .catch(() => setAuth('login'))
  }, [])

  if (window.location.pathname.startsWith('/shared/')) {
    return (
      <I18nProvider>
        <Routes>
          <Route path="/shared/:token" element={<SharedSessionPage />} />
        </Routes>
      </I18nProvider>
    )
  }

  if (auth === 'loading') {
    return (
      <I18nProvider>
        <LoadingScreen />
      </I18nProvider>
    )
  }
  if (auth === 'login') {
    return (
      <I18nProvider>
        <LoginScreen />
      </I18nProvider>
    )
  }

  return (
    <I18nProvider>
      <WorkspaceProvider>
        <Routes>
          <Route element={<Layout />}>
            <Route index element={<Navigate to="/tasks" replace />} />
            <Route path="inbox" element={<InboxPage />} />
            <Route path="discover" element={<DiscoverPage />} />
            <Route path="my-tasks" element={<MyTasksPage />} />
            <Route path="tasks" element={<TasksPage />} />
            <Route path="tasks/:id" element={<TaskDetailPage />} />
            <Route path="runs" element={<RunsPage />} />
            <Route path="runs/:id" element={<RunDetailPage />} />
            <Route path="dashboard" element={<AgentDashboardPage />} />
            <Route path="session-history" element={<SessionHistoryPage />} />
            <Route path="sessions" element={<SessionsPage />} />
            <Route path="sessions/:id" element={<SessionDetailPage />} />
            <Route path="audit" element={<AuditLogPage />} />
            <Route path="approvals" element={<ApprovalsPage />} />
            <Route path="automations" element={<AutomationsPage />} />
            <Route path="workflows" element={<WorkflowsPage />} />
            <Route path="workflows/new" element={<WorkflowEditorPage />} />
            <Route path="workflows/:id/edit" element={<WorkflowEditorPage />} />
            <Route path="usage" element={<UsagePage />} />
            <Route path="agents" element={<AgentsPage />} />
            <Route path="groups" element={<GroupsPage />} />
            <Route path="workspace" element={<WorkspacePage />} />
            <Route path="workspaces/:id/settings" element={<WorkspaceSettingsPage />} />
            <Route path="iterations" element={<IterationsPage />} />
            <Route path="providers" element={<ProvidersPage />} />
            <Route path="credentials" element={<CredentialsPage />} />
            <Route path="skills" element={<SkillsPage />} />
            <Route path="machines" element={<MachinesPage />} />
            <Route path="machines/registry" element={<ACPRegistryPage />} />
            <Route path="machines/:id" element={<MachineDetailPage />} />
            <Route path="notifications" element={<NotificationsPage />} />
            <Route path="documents" element={<DocumentsPage />} />
            <Route path="*" element={<Navigate to="/tasks" replace />} />
          </Route>
          <Route path="settings" element={<SettingsLayout />}>
            <Route index element={<Navigate to="appearance" replace />} />
            <Route path="appearance" element={<SettingsAppearancePage />} />
            <Route path="account" element={<SettingsAccountPage />} />
            <Route path="mobile" element={<SettingsMobilePage />} />
            <Route path="acp" element={<Navigate to="/machines/registry" replace />} />
          </Route>
        </Routes>
      </WorkspaceProvider>
    </I18nProvider>
  )
}

function LoadingScreen() {
  const { t } = useI18n()
  return (
    <div className="flex h-full items-center justify-center text-[12px] text-muted">
      {t('common.loading')}
    </div>
  )
}

function LoginScreen() {
  const { t } = useI18n()
  return (
    <div className="flex h-full items-center justify-center bg-bg">
      <div className="w-[320px] rounded-lg border border-line bg-panel/60 p-6 text-center">
        <div className="mx-auto flex h-8 w-8 items-center justify-center rounded bg-accent text-[13px] font-bold text-white">
          O
        </div>
        <h1 className="mt-3 text-[15px] font-semibold">{t('login.title')}</h1>
        <p className="mt-1 text-[12px] text-muted">{t('login.subtitle')}</p>
        <a
          href="/auth/login"
          className="mt-4 inline-flex h-8 w-full items-center justify-center rounded-md bg-accent text-[13px] font-medium text-white hover:opacity-90"
        >
          {t('login.sso')}
        </a>
      </div>
    </div>
  )
}
