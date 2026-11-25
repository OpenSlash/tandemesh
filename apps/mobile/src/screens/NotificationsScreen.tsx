import { useEffect, useState } from 'react'
import { ActivityIndicator, FlatList, Pressable, RefreshControl,  StyleSheet, Text, View } from 'react-native'
import { Screen } from '../components/Screen'
import { TandemeshClient } from '../lib/api'
import { useSession } from '../lib/SessionContext'
import { useWorkspace } from '../lib/WorkspaceContext'

interface Notification {
  id: string
  title: string
  body?: string
  readAt?: string
}

export function NotificationsScreen() {
  const { host, token } = useSession()
  const { workspace } = useWorkspace()
  const [client] = useState(() => new TandemeshClient(host, token))
  const [items, setItems] = useState<Notification[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)

  useEffect(() => {
    let cancelled = false
    const load = async () => {
      try {
        const result = await client.notifications(workspace.id)
        if (!cancelled) setItems(result.items)
      } catch (nextError) {
        if (!cancelled) setError(nextError instanceof Error ? nextError.message : 'Failed to load notifications')
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    void load()
    return () => {
      cancelled = true
    }
  }, [client, workspace.id])

  const refresh = async () => {
    setRefreshing(true)
    try {
      const result = await client.notifications(workspace.id)
      setItems(result.items)
      setError('')
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : 'Failed to load notifications')
    } finally {
      setRefreshing(false)
    }
  }

  const markRead = async (id: string) => {
    try {
      await client.markNotificationRead(id)
      setItems((current) =>
        current.map((item) =>
          item.id === id ? { ...item, readAt: item.readAt ?? new Date().toISOString() } : item,
        ),
      )
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : 'Failed to mark notification read')
    }
  }

  return (
    <Screen>
      <View style={styles.header}>
        <Text style={styles.brand}>Tandemesh</Text>
        <Text style={styles.title}>Notifications</Text>
      </View>
      {loading ? (
        <View style={styles.center}>
          <ActivityIndicator color="#d6d9e0" />
        </View>
      ) : error ? (
        <View style={styles.center}>
          <Text style={styles.error}>{error}</Text>
        </View>
      ) : (
        <FlatList
          data={items}
          keyExtractor={(item) => item.id}
          contentContainerStyle={styles.list}
          refreshControl={<RefreshControl refreshing={refreshing} onRefresh={() => void refresh()} tintColor="#d6d9e0" />}
          ListEmptyComponent={
            <View style={styles.center}>
              <Text style={styles.empty}>Nothing new.</Text>
            </View>
          }
          renderItem={({ item }) => (
            <Pressable
              style={[styles.card, item.readAt ? styles.read : null]}
              onPress={() => void markRead(item.id)}
            >
              <Text style={styles.cardTitle}>{item.title}</Text>
              {item.body ? <Text style={styles.cardBody}>{item.body}</Text> : null}
              {error ? <Text style={styles.error}>{error}</Text> : null}
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
  error: { color: '#f87171', fontSize: 12, marginTop: 6 },
  empty: { color: '#8b94a7', fontSize: 13 },
  list: { padding: 16, gap: 8 },
  card: { borderRadius: 10, borderWidth: 1, borderColor: '#202433', backgroundColor: '#10131c', padding: 14 },
  read: { opacity: 0.55 },
  cardTitle: { color: '#f3f4f6', fontSize: 14 },
  cardBody: { color: '#8b94a7', fontSize: 12, marginTop: 5 },
})
