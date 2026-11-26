import { useEffect, useState } from 'react'
import { ActivityIndicator, Pressable, StatusBar, StyleSheet, Text, View, useWindowDimensions } from 'react-native'
import { AppTabBarView, type AppTabItem } from './components/AppTabBar'
import { PairScreen } from './screens/PairScreen'
import { TasksScreen } from './screens/TasksScreen'
import { SessionsScreen } from './screens/SessionsScreen'
import { ApprovalsScreen } from './screens/ApprovalsScreen'
import { NotificationsScreen } from './screens/NotificationsScreen'
import { SettingsScreen } from './screens/SettingsScreen'
import { TaskDetailScreen } from './screens/TaskDetailScreen'
import { RunDetailScreen } from './screens/RunDetailScreen'
import { DiffScreen } from './screens/DiffScreen'
import { SessionDetailScreen } from './screens/SessionDetailScreen'
import { TandemeshClient, type Workspace } from './lib/api'
import { SessionContext } from './lib/SessionContext'
import { WorkspaceContext } from './lib/WorkspaceContext'
import { RouterContext, type RouteName } from './lib/RouterContext'
import { clearSession, loadSession, saveSession, type MobileSession } from './lib/session'

type Tab = 'tasks' | 'sessions' | 'approvals' | 'notifications' | 'settings'

interface RouteState {
  name: RouteName
  params?: Record<string, string>
}

export default function HarmonyApp() {
  const [session, setSession] = useState<MobileSession | null | undefined>(undefined)

  useEffect(() => {
    void loadSession().then(setSession)
  }, [])

  if (session === undefined) {
    return (
      <View style={styles.center}>
        <StatusBar barStyle="light-content" backgroundColor="#07080f" />
        <ActivityIndicator color="#d6d9e0" />
      </View>
    )
  }
  if (!session) {
    return (
      <PairScreen
        onPaired={async (host, token) => {
          await saveSession(host, token)
          setSession({ host, token })
        }}
      />
    )
  }
  return <Home session={session} onSignOut={() => { void clearSession(); setSession(null) }} />
}

function Home({ session, onSignOut }: { session: MobileSession; onSignOut: () => void }) {
  const [client] = useState(() => new TandemeshClient(session.host, session.token))
  const [workspaces, setWorkspaces] = useState<Workspace[]>([])
  const [workspaceId, setWorkspaceId] = useState('')
  const [tab, setTab] = useState<Tab>('tasks')
  const [route, setRoute] = useState<RouteState | null>(null)
  const [error, setError] = useState('')
  const [unread, setUnread] = useState(0)

  useEffect(() => {
    client
      .workspaces()
      .then((result) => {
        setWorkspaces(result.items)
        setWorkspaceId(result.items[0]?.id ?? '')
      })
      .catch((nextError) => setError(nextError instanceof Error ? nextError.message : 'Failed to load workspace'))
  }, [client])

  useEffect(() => {
    const id = workspaces.find((item) => item.id === workspaceId)?.id ?? workspaces[0]?.id
    if (!id) return
    let cancelled = false
    const load = async () => {
      try {
        const result = await client.notifications(id)
        if (!cancelled) setUnread(result.items.filter((item) => !item.readAt).length)
      } catch {
        // ignore polling errors
      }
    }
    void load()
    const timer = setInterval(() => void load(), 15000)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [client, workspaces, workspaceId])

  const workspace = workspaces.find((item) => item.id === workspaceId) ?? workspaces[0]
  if (error) {
    return (
      <View style={styles.center}>
        <Text style={styles.error}>{error}</Text>
      </View>
    )
  }
  if (!workspace) {
    return (
      <View style={styles.center}>
        <ActivityIndicator color="#d6d9e0" />
      </View>
    )
  }

  const router = {
    params: route?.params,
    navigate: (name: RouteName, params?: Record<string, string>) => setRoute({ name, params }),
    back: () => setRoute(null),
  }
  const { width, height } = useWindowDimensions()
  const wide = width >= 720 && width > height
  const tabs: AppTabItem[] = [
    { key: 'tasks', label: 'Tasks', icon: 'tasks' },
    { key: 'sessions', label: 'Sessions', icon: 'sessions' },
    { key: 'approvals', label: 'Approvals', icon: 'approvals' },
    { key: 'notifications', label: 'Alerts', icon: 'alerts', badge: unread },
    { key: 'settings', label: 'Settings', icon: 'settings' },
  ]
  const tabBar = (
    <AppTabBarView items={tabs} active={tab} onChange={(key) => setTab(key as Tab)} bottomInset={18} />
  )
  const content = route ? (
    <>
      <Pressable style={styles.back} onPress={() => setRoute(null)}>
        <Text style={styles.backText}>‹ Back</Text>
      </Pressable>
      {route.name === 'TaskDetail' ? <TaskDetailScreen /> : null}
      {route.name === 'SessionDetail' ? <SessionDetailScreen /> : null}
      {route.name === 'RunDetail' ? <RunDetailScreen /> : null}
      {route.name === 'Diff' ? <DiffScreen /> : null}
    </>
  ) : (
    <>
      {tab === 'tasks' ? <TasksScreen /> : null}
      {tab === 'sessions' ? <SessionsScreen /> : null}
      {tab === 'approvals' ? <ApprovalsScreen /> : null}
      {tab === 'notifications' ? <NotificationsScreen /> : null}
      {tab === 'settings' ? <SettingsScreen /> : null}
    </>
  )

  return (
    <SessionContext.Provider value={{ host: session.host, token: session.token, onSignOut }}>
      <WorkspaceContext.Provider value={{ workspace, workspaces, selectWorkspace: setWorkspaceId }}>
        <RouterContext.Provider value={router}>
          <View style={styles.screen}>
            <StatusBar barStyle="light-content" backgroundColor="#07080f" />
            {wide && !route ? (
              <View style={styles.wide}>
                {tabBar}
                <View style={styles.body}>{content}</View>
              </View>
            ) : (
              <>
                <View style={styles.body}>{content}</View>
                {route ? null : tabBar}
              </>
            )}
          </View>
        </RouterContext.Provider>
      </WorkspaceContext.Provider>
    </SessionContext.Provider>
  )
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#07080f', paddingTop: 32 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: '#07080f', paddingTop: 32 },
  body: { flex: 1 },
  wide: { flex: 1, flexDirection: 'row' },
  error: { color: '#f87171', fontSize: 12 },
  back: { paddingHorizontal: 16, paddingVertical: 10 },
  backText: { color: '#8fb7e0', fontSize: 13 },
})
