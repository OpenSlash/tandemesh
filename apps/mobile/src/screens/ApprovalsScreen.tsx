import { useEffect, useState } from 'react'
import { ActivityIndicator, FlatList, Pressable, RefreshControl,  StyleSheet, Text, View } from 'react-native'
import { SafeAreaView } from 'react-native-safe-area-context'
import { TandemeshClient } from '../lib/api'
import { useSession } from '../lib/SessionContext'
import { useWorkspace } from '../lib/WorkspaceContext'

interface Approval {
  id: string
  runId: string
  tool: string
  risk: string
  status: string
}

export function ApprovalsScreen() {
  const { host, token } = useSession()
  const { workspace } = useWorkspace()
  const [client] = useState(() => new TandemeshClient(host, token))
  const [items, setItems] = useState<Approval[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [deciding, setDeciding] = useState('')

  useEffect(() => {
    let cancelled = false
    const load = async () => {
      try {
        const result = await client.approvals(workspace.id)
        if (!cancelled) setItems(result.items)
      } catch (nextError) {
        if (!cancelled) setError(nextError instanceof Error ? nextError.message : 'Failed to load approvals')
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
      const result = await client.approvals(workspace.id)
      setItems(result.items)
      setError('')
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : 'Failed to load approvals')
    } finally {
      setRefreshing(false)
    }
  }

  const decide = async (id: string, decision: 'approved' | 'denied') => {
    setDeciding(id)
    try {
      await client.decideApproval(id, decision)
      setItems((current) => current.map((item) => (item.id === id ? { ...item, status: decision } : item)))
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : 'Failed to decide')
    } finally {
      setDeciding('')
    }
  }

  return (
    <SafeAreaView style={styles.screen}>
      <View style={styles.header}>
        <Text style={styles.brand}>Tandemesh</Text>
        <Text style={styles.title}>Approvals</Text>
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
              <Text style={styles.empty}>No pending approvals.</Text>
            </View>
          }
          renderItem={({ item }) => (
            <View style={styles.card}>
              <Text style={styles.cardTitle}>{item.tool || item.runId}</Text>
              <Text style={styles.cardMeta}>
                {item.risk} · {item.status}
              </Text>
              {error ? <Text style={styles.error}>{error}</Text> : null}
              {item.status === 'pending' ? (
                <View style={styles.actions}>
                  <Pressable
                    style={[styles.action, styles.approve]}
                    disabled={deciding === item.id}
                    onPress={() => void decide(item.id, 'approved')}
                  >
                    <Text style={styles.actionText}>Approve</Text>
                  </Pressable>
                  <Pressable
                    style={[styles.action, styles.deny]}
                    disabled={deciding === item.id}
                    onPress={() => void decide(item.id, 'denied')}
                  >
                    <Text style={styles.actionText}>Deny</Text>
                  </Pressable>
                </View>
              ) : null}
            </View>
          )}
        />
      )}
    </SafeAreaView>
  )
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#07080f' },
  header: { padding: 20, borderBottomWidth: 1, borderBottomColor: '#202433' },
  brand: { color: '#9aa4b2', fontSize: 11, letterSpacing: 2, textTransform: 'uppercase' },
  title: { color: '#f3f4f6', fontSize: 22, fontWeight: '600', marginTop: 4 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 },
  error: { color: '#f87171', fontSize: 12, marginTop: 8 },
  empty: { color: '#8b94a7', fontSize: 13 },
  list: { padding: 16, gap: 8 },
  card: { borderRadius: 10, borderWidth: 1, borderColor: '#202433', backgroundColor: '#10131c', padding: 14 },
  cardTitle: { color: '#f3f4f6', fontSize: 14 },
  cardMeta: { color: '#8b94a7', fontSize: 11, marginTop: 6 },
  actions: { flexDirection: 'row', gap: 8, marginTop: 12 },
  action: { borderRadius: 6, paddingHorizontal: 12, paddingVertical: 7 },
  approve: { backgroundColor: '#17351f' },
  deny: { backgroundColor: '#3a171a' },
  actionText: { color: '#eef0f4', fontSize: 12 },
})
