import { useEffect, useState } from 'react'
import { ActivityIndicator, Linking, StatusBar, StyleSheet, View } from 'react-native'
import { PairScreen } from './src/screens/PairScreen'
import { TasksScreen } from './src/screens/TasksScreen'
import { parsePairingLink } from './src/lib/api'
import { clearSession, loadSession, saveSession, type MobileSession } from './src/lib/session'

interface DeepLinkPair {
  host?: string
  code?: string
}

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

  return (
    <View style={styles.root}>
      <StatusBar barStyle="light-content" backgroundColor="#07080f" />
      {paired === undefined ? (
        <View style={styles.loading}>
          <ActivityIndicator color="#d6d9e0" />
        </View>
      ) : paired ? (
        <TasksScreen
          host={paired.host}
          token={paired.token}
          onSignOut={() => {
            void clearSession()
            setPaired(null)
          }}
        />
      ) : (
        <PairScreen
          initialHost={pairLink.host}
          initialCode={pairLink.code}
          onPaired={async (host, token) => {
            await saveSession(host, token)
            setPaired({ host, token })
          }}
        />
      )}
    </View>
  )
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: '#07080f' },
  loading: { flex: 1, alignItems: 'center', justifyContent: 'center' },
})

export default App
