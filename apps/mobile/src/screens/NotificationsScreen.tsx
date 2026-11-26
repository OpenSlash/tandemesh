import { useEffect, useMemo, useState } from 'react'
import { ActivityIndicator, FlatList, Pressable, RefreshControl,  StyleSheet, Text, View } from 'react-native'
import { Screen } from '../components/Screen'
import { useTheme, type ThemeColors } from '../lib/theme'
import { TandemeshClient } from '../lib/api'
import { useI18n } from '../lib/i18n'
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
  const { t } = useI18n()
  const { colors } = useTheme()
  const styles = useMemo(() => makeStyles(colors), [colors])

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
        <Text style={styles.title}>{t('notifications.title')}</Text>
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
              <Text style={styles.empty}>{t('notifications.empty')}</Text>
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

const makeStyles = (c: ThemeColors) => StyleSheet.create({
  screen: { flex: 1, backgroundColor: c.background },
  header: { paddingHorizontal: 20, paddingVertical: 12, borderBottomWidth: 1, borderBottomColor: c.border },
  brand: { color: c.textMuted, fontSize: 10, letterSpacing: 2, textTransform: 'uppercase' },
  title: { color: c.text, fontSize: 20, fontWeight: '600', marginTop: 2 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 },
  error: { color: c.danger, fontSize: 12, marginTop: 6 },
  empty: { color: c.textMuted, fontSize: 13 },
  list: { padding: 16, gap: 8, paddingBottom: 110 },
  card: { borderRadius: 10, borderWidth: 1, borderColor: c.border, backgroundColor: c.surface, padding: 14 },
  read: { opacity: 0.55 },
  cardTitle: { color: c.text, fontSize: 14 },
  cardBody: { color: c.textMuted, fontSize: 12, marginTop: 5 },
})
