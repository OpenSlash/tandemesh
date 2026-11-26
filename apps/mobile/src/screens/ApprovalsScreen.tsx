import { useEffect, useMemo, useState } from 'react'
import { ActivityIndicator, FlatList, Pressable, RefreshControl,  StyleSheet, Text, View } from 'react-native'
import { Screen } from '../components/Screen'
import { useTheme, type ThemeColors } from '../lib/theme'
import { TandemeshClient } from '../lib/api'
import { useI18n } from '../lib/i18n'
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
  const { t } = useI18n()
  const { colors } = useTheme()
  const styles = useMemo(() => makeStyles(colors), [colors])

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
    <Screen>
      <View style={styles.header}>
        <Text style={styles.brand}>Tandemesh</Text>
        <Text style={styles.title}>{t('approvals.title')}</Text>
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
              <Text style={styles.empty}>{t('approvals.empty')}</Text>
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
    </Screen>
  )
}

const makeStyles = (c: ThemeColors) => StyleSheet.create({
  screen: { flex: 1, backgroundColor: c.background },
  header: { paddingHorizontal: 20, paddingVertical: 12, borderBottomWidth: 1, borderBottomColor: c.border },
  brand: { color: c.textMuted, fontSize: 10, letterSpacing: 2, textTransform: 'uppercase' },
  title: { color: c.text, fontSize: 20, fontWeight: '600', marginTop: 2 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 },
  error: { color: c.danger, fontSize: 12, marginTop: 8 },
  empty: { color: c.textMuted, fontSize: 13 },
  list: { padding: 16, gap: 8, paddingBottom: 110 },
  card: { borderRadius: 10, borderWidth: 1, borderColor: c.border, backgroundColor: c.surface, padding: 14 },
  cardTitle: { color: c.text, fontSize: 14 },
  cardMeta: { color: c.textMuted, fontSize: 11, marginTop: 6 },
  actions: { flexDirection: 'row', gap: 8, marginTop: 12 },
  action: { borderRadius: 6, paddingHorizontal: 12, paddingVertical: 7 },
  approve: { backgroundColor: c.surfaceAlt },
  deny: { backgroundColor: c.dangerBg },
  actionText: { color: c.surfaceAlt, fontSize: 12 },
})
