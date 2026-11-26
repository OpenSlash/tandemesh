import { useEffect, useState } from 'react'
import { ActivityIndicator, Linking, StyleSheet, View } from 'react-native'
import { SafeAreaProvider } from 'react-native-safe-area-context'
import { NavigationContainer, DarkTheme } from '@react-navigation/native'
import { createNativeStackNavigator } from '@react-navigation/native-stack'
import { PairScreen } from './screens/PairScreen'
import { MainTabs } from './screens/MainTabs'
import { TaskDetailScreen } from './screens/TaskDetailScreen'
import { RunDetailScreen } from './screens/RunDetailScreen'
import { DiffScreen } from './screens/DiffScreen'
import { SessionDetailScreen } from './screens/SessionDetailScreen'
import { withNativeRouter } from './navigation/native-router'
import { parsePairingLink } from './lib/api'
import { SessionContext } from './lib/SessionContext'
import { clearSession, loadSession, saveSession, type MobileSession } from './lib/session'
import type { RootStackParamList } from './navigation/types'

interface DeepLinkPair {
  host?: string
  code?: string
}

const Stack = createNativeStackNavigator<RootStackParamList>()

function App() {
  const [pairLink, setPairLink] = useState<DeepLinkPair>({})
  const [paired, setPaired] = useState<MobileSession | null | undefined>(undefined)

  useEffect(() => {
    void loadSession().then(setPaired)
  }, [])

  useEffect(() => {
    const apply = (url: string | null) => {
      if (!url) return
      const parsed = parsePairingLink(url)
      if (parsed.code || parsed.host) setPairLink(parsed)
    }
    void Linking.getInitialURL().then(apply)
    const subscription = Linking.addEventListener('url', (event) => apply(event.url))
    return () => subscription.remove()
  }, [])

  const signOut = () => {
    void clearSession()
    setPaired(null)
  }

  if (paired === undefined) {
    return (
      <SafeAreaProvider>
      <View style={styles.root}>
        <View style={styles.loading}>
          <ActivityIndicator color="#d6d9e0" />
        </View>
      </View>
      </SafeAreaProvider>
    )
  }

  if (!paired) {
    return (
      <SafeAreaProvider>
      <View style={styles.root}>
        <PairScreen
          // Remount when a pairing link arrives while this screen is open.
          key={`${pairLink.host ?? ''}|${pairLink.code ?? ''}`}
          initialHost={pairLink.host}
          initialCode={pairLink.code}
          onPaired={async (host, token) => {
            await saveSession(host, token)
            setPaired({ host, token })
          }}
        />
      </View>
      </SafeAreaProvider>
    )
  }

  return (
    <SafeAreaProvider>
    <SessionContext.Provider value={{ host: paired.host, token: paired.token, onSignOut: signOut }}>
      <NavigationContainer theme={DarkTheme}>
        <Stack.Navigator
          screenOptions={{
            headerStyle: { backgroundColor: '#0b0d14' },
            headerTintColor: '#f3f4f6',
            contentStyle: { backgroundColor: '#07080f' },
          }}
        >
          <Stack.Screen name="Main" component={MainTabs} options={{ headerShown: false }} />
          <Stack.Screen name="TaskDetail" component={withNativeRouter(TaskDetailScreen)} options={{ title: 'Task' }} />
          <Stack.Screen name="RunDetail" component={withNativeRouter(RunDetailScreen)} options={{ title: 'Run' }} />
          <Stack.Screen name="Diff" component={withNativeRouter(DiffScreen)} options={{ title: 'Diff' }} />
          <Stack.Screen name="SessionDetail" component={withNativeRouter(SessionDetailScreen)} options={{ title: 'Session' }} />
        </Stack.Navigator>
      </NavigationContainer>
    </SessionContext.Provider>
    </SafeAreaProvider>
  )
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: '#07080f' },
  loading: { flex: 1, alignItems: 'center', justifyContent: 'center' },
})

export default App
