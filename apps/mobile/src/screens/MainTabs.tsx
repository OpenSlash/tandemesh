import { useEffect, useState } from 'react'
import { ActivityIndicator, StyleSheet, Text, View } from 'react-native'
import { createBottomTabNavigator } from '@react-navigation/bottom-tabs'
import { TandemeshClient, type Workspace } from '../lib/api'
import { useSession } from '../lib/SessionContext'
import { WorkspaceContext } from '../lib/WorkspaceContext'
import { TasksScreen } from './TasksScreen'
import { SessionsScreen } from './SessionsScreen'
import { ApprovalsScreen } from './ApprovalsScreen'
import { NotificationsScreen } from './NotificationsScreen'
import { SettingsScreen } from './SettingsScreen'
import { withNativeRouter } from '../navigation/native-router'

const Tab = createBottomTabNavigator()

export function MainTabs() {
  const { host, token } = useSession()
  const [client] = useState(() => new TandemeshClient(host, token))
  const [workspaces, setWorkspaces] = useState<Workspace[]>([])
  const [selectedId, setSelectedId] = useState('')
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    const load = async () => {
      try {
        const result = await client.workspaces()
        if (!cancelled) {
          setWorkspaces(result.items)
          setSelectedId((current) => current || result.items[0]?.id || '')
        }
      } catch (nextError) {
        if (!cancelled) setError(nextError instanceof Error ? nextError.message : 'Failed to load workspace')
      }
    }
    void load()
    return () => {
      cancelled = true
    }
  }, [client])

  if (error) {
    return (
      <View style={styles.center}>
        <Text style={styles.error}>{error}</Text>
      </View>
    )
  }
  const workspace = workspaces.find((item) => item.id === selectedId) ?? workspaces[0]
  if (!workspace) {
    return (
      <View style={styles.center}>
        <ActivityIndicator color="#d6d9e0" />
      </View>
    )
  }

  return (
    <WorkspaceContext.Provider value={{ workspace, workspaces, selectWorkspace: setSelectedId }}>
      <Tab.Navigator
        screenOptions={{
          headerShown: false,
          tabBarStyle: { backgroundColor: '#0b0d14', borderTopColor: '#202433' },
          tabBarActiveTintColor: '#f3f4f6',
          tabBarInactiveTintColor: '#6e7787',
        }}
      >
        <Tab.Screen name="Tasks" component={withNativeRouter(TasksScreen)} />
        <Tab.Screen name="Sessions" component={withNativeRouter(SessionsScreen)} />
        <Tab.Screen name="Approvals" component={withNativeRouter(ApprovalsScreen)} />
        <Tab.Screen name="Notifications" component={withNativeRouter(NotificationsScreen)} />
        <Tab.Screen name="Settings" component={withNativeRouter(SettingsScreen)} />
      </Tab.Navigator>
    </WorkspaceContext.Provider>
  )
}

const styles = StyleSheet.create({
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: '#07080f', padding: 24 },
  error: { color: '#f87171', fontSize: 13, textAlign: 'center' },
})
