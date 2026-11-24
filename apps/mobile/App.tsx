import { useEffect, useState } from 'react'
import { Linking, StatusBar, StyleSheet, View } from 'react-native'
import { PairScreen } from './src/screens/PairScreen'
import { TasksScreen } from './src/screens/TasksScreen'
import { parsePairingLink } from './src/lib/api'
import { clearSession, saveSession, session } from './src/lib/session'

interface DeepLinkPair {
  host?: string
  code?: string
}

function App() {
  const [pairLink, setPairLink] = useState<DeepLinkPair>({})
  const [paired, setPaired] = useState(session())

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
      {paired ? (
        <TasksScreen
          host={paired.host}
          token={paired.token}
          onSignOut={() => {
            clearSession()
            setPaired(null)
          }}
        />
      ) : (
        <PairScreen
          initialHost={pairLink.host}
          initialCode={pairLink.code}
          onPaired={(host, token) => {
            saveSession(host, token)
            setPaired({ host, token })
          }}
        />
      )}
    </View>
  )
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: '#07080f' },
})

export default App
