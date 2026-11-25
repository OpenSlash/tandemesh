import { useEffect, useState } from 'react'
import {
  ActivityIndicator,
  FlatList,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  View,
} from 'react-native'
import { Screen } from '../components/Screen'
import { TandemeshClient, type AgentSession } from '../lib/api'
import { useSession } from '../lib/SessionContext'
import { useWorkspace } from '../lib/WorkspaceContext'
import { useRouter } from '../lib/RouterContext'

export function SessionsScreen() {
  const { host, token } = useSession()
  const { workspace } = useWorkspace()
  const router = useRouter()
  const [client] = useState(() => new TandemeshClient(host, token))
  const [sessions, setSessions] = useState<AgentSession[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)

  const load = async (reset: boolean) => {
    try {
      const result = await client.sessions(workspace.id)
      setSessions(result.sessions)
      setError('')
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : 'Failed to load sessions')
    } finally {
      if (reset) setLoading(false)
      setRefreshing(false)
    }
  }

  useEffect(() => {
    setLoading(true)
    void load(true)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [client, workspace.id])

  return (
    <Screen>
      <View style={styles.header}>
        <Text style={styles.brand}>Tandemesh</Text>
        <Text style={styles.title}>Sessions</Text>
      </View>
      {loading ? (
        <View style={styles.center}><ActivityIndicator color="#d6d9e0" /></View>
      ) : error ? (
        <View style={styles.center}><Text style={styles.error}>{error}</Text></View>
      ) : (
        <FlatList
          data={sessions}
          keyExtractor={(session) => session.id}
          contentContainerStyle={styles.list}
          refreshControl={
            <RefreshControl
              refreshing={refreshing}
              onRefresh={() => {
                setRefreshing(true)
                void load(false)
              }}
              tintColor="#d6d9e0"
            />
          }
          ListEmptyComponent={
            <View style={styles.center}>
              <Text style={styles.empty}>No agent sessions yet.</Text>
            </View>
          }
          renderItem={({ item }) => (
            <Pressable
              style={styles.card}
              onPress={() => router.navigate('SessionDetail', { id: item.id })}
            >
              <Text style={styles.cardTitle}>{item.title || item.id}</Text>
              <Text style={styles.cardMeta}>
                {item.backend || item.agentId} · {item.lastRunStatus || item.status}
              </Text>
            </Pressable>
          )}
        />
      )}
    </Screen>
  )
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#07080f' },
  header: { padding: 20, borderBottomWidth: 1, borderBottomColor: '#202433' },
  brand: { color: '#9aa4b2', fontSize: 11, letterSpacing: 2, textTransform: 'uppercase' },
  title: { color: '#f3f4f6', fontSize: 22, fontWeight: '600', marginTop: 4 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 },
  error: { color: '#f87171', fontSize: 13, textAlign: 'center' },
  empty: { color: '#8b94a7', fontSize: 13 },
  list: { padding: 16, gap: 8 },
  card: { borderRadius: 10, borderWidth: 1, borderColor: '#202433', backgroundColor: '#10131c', padding: 14 },
  cardTitle: { color: '#f3f4f6', fontSize: 14 },
  cardMeta: { color: '#8b94a7', fontSize: 11, marginTop: 6 },
})
