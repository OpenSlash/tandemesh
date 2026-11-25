import { useEffect, useState } from 'react'
import { ActivityIndicator, Pressable, StatusBar, StyleSheet, Text, View } from 'react-native'
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

  useEffect(() => {
    client
      .workspaces()
      .then((result) => {
        setWorkspaces(result.items)
        setWorkspaceId(result.items[0]?.id ?? '')
      })
      .catch((nextError) => setError(nextError instanceof Error ? nextError.message : 'Failed to load workspace'))
  }, [client])

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

  return (
    <SessionContext.Provider value={{ host: session.host, token: session.token, onSignOut }}>
      <WorkspaceContext.Provider value={{ workspace, workspaces, selectWorkspace: setWorkspaceId }}>
        <RouterContext.Provider value={router}>
          <View style={styles.screen}>
            <StatusBar barStyle="light-content" backgroundColor="#07080f" />
            <View style={styles.body}>
              {route ? (
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
              )}
            </View>
            {route ? null : (
              <View style={styles.tabs}>
                {([
                  ['tasks', 'Tasks'],
                  ['sessions', 'Sessions'],
                  ['approvals', 'Approvals'],
                  ['notifications', 'Notifications'],
                  ['settings', 'Settings'],
                ] as [Tab, string][]).map(([value, label]) => (
                  <Pressable key={value} style={styles.tab} onPress={() => setTab(value)}>
                    <Text style={[styles.tabText, tab === value && styles.tabActive]}>{label}</Text>
                  </Pressable>
                ))}
              </View>
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
  error: { color: '#f87171', fontSize: 12 },
  back: { paddingHorizontal: 16, paddingVertical: 10 },
  backText: { color: '#8fb7e0', fontSize: 13 },
  tabs: { flexDirection: 'row', borderTopWidth: 1, borderTopColor: '#202433', backgroundColor: '#0b0d14', paddingTop: 10, paddingBottom: 18 },
  tab: { flex: 1, alignItems: 'center' },
  tabText: { color: '#6e7787', fontSize: 10.5 },
  tabActive: { color: '#f3f4f6', fontWeight: '600' },
})
