import { useEffect, useState } from 'react'
import { ActivityIndicator, Linking, StatusBar, StyleSheet, View } from 'react-native'
import { NavigationContainer, DarkTheme } from '@react-navigation/native'
import { createNativeStackNavigator } from '@react-navigation/native-stack'
import { PairScreen } from './src/screens/PairScreen'
import { MainTabs } from './src/screens/MainTabs'
import { TaskDetailScreen } from './src/screens/TaskDetailScreen'
import { parsePairingLink } from './src/lib/api'
import { SessionContext } from './src/lib/SessionContext'
import { clearSession, loadSession, saveSession, type MobileSession } from './src/lib/session'
import type { RootStackParamList } from './src/navigation/types'

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
      <View style={styles.root}>
        <StatusBar barStyle="light-content" backgroundColor="#07080f" />
        <View style={styles.loading}>
          <ActivityIndicator color="#d6d9e0" />
        </View>
      </View>
    )
  }

  if (!paired) {
    return (
      <View style={styles.root}>
        <StatusBar barStyle="light-content" backgroundColor="#07080f" />
        <PairScreen
          initialHost={pairLink.host}
          initialCode={pairLink.code}
          onPaired={async (host, token) => {
            await saveSession(host, token)
            setPaired({ host, token })
          }}
        />
      </View>
    )
  }

  return (
    <SessionContext.Provider value={{ host: paired.host, token: paired.token, onSignOut: signOut }}>
      <StatusBar barStyle="light-content" backgroundColor="#07080f" />
      <NavigationContainer theme={DarkTheme}>
        <Stack.Navigator
          screenOptions={{
            headerStyle: { backgroundColor: '#0b0d14' },
            headerTintColor: '#f3f4f6',
            contentStyle: { backgroundColor: '#07080f' },
          }}
        >
          <Stack.Screen name="Main" component={MainTabs} options={{ headerShown: false }} />
          <Stack.Screen name="TaskDetail" component={TaskDetailScreen} options={{ title: 'Task' }} />
        </Stack.Navigator>
      </NavigationContainer>
    </SessionContext.Provider>
  )
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: '#07080f' },
  loading: { flex: 1, alignItems: 'center', justifyContent: 'center' },
})

export default App
