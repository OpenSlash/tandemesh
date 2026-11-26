import { useEffect, useMemo, useState } from 'react'
import {
  ActivityIndicator,
  FlatList,
  Modal,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  TextInput,
  View,
  useWindowDimensions,
} from 'react-native'
import { Screen } from '../components/Screen'
import { useTheme, type ThemeColors } from '../lib/theme'
import { TandemeshClient, type AgentSession } from '../lib/api'
import { useSession } from '../lib/SessionContext'
import { useWorkspace } from '../lib/WorkspaceContext'
import { useRouter } from '../lib/RouterContext'
import { useI18n } from '../lib/i18n'

export function SessionsScreen() {
  const { host, token } = useSession()
  const { workspace } = useWorkspace()
  const router = useRouter()
  const { t } = useI18n()
  const { colors } = useTheme()
  const styles = useMemo(() => makeStyles(colors), [colors])

  const [client] = useState(() => new TandemeshClient(host, token))
  const [sessions, setSessions] = useState<AgentSession[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [newOpen, setNewOpen] = useState(false)
  const [newTitle, setNewTitle] = useState('')
  const [creating, setCreating] = useState(false)
  const { width } = useWindowDimensions()
  const columns = width >= 700 ? 2 : 1

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

  const create = async () => {
    const title = newTitle.trim() || 'New session'
    setCreating(true)
    try {
      const created = await client.createSession(workspace.id, title)
      setSessions((current) => [created, ...current])
      setNewOpen(false)
      setNewTitle('')
      router.navigate('SessionDetail', { id: created.id })
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : 'Failed to create session')
    } finally {
      setCreating(false)
    }
  }

  return (
    <Screen>
      <View style={styles.header}>
        <View style={styles.headerText}>
          <Text style={styles.brand}>Tandemesh</Text>
          <Text style={styles.title}>{t('sessions.title')}</Text>
        </View>
        <Pressable style={styles.newButton} onPress={() => setNewOpen(true)}>
          <Text style={styles.newButtonText}>{t('sessions.new')}</Text>
        </Pressable>
      </View>

      <Modal visible={newOpen} transparent animationType="fade" onRequestClose={() => setNewOpen(false)}>
        <Pressable style={styles.backdrop} onPress={() => setNewOpen(false)}>
          <View style={styles.dialog}>
            <Text style={styles.dialogTitle}>{t('sessions.newTitle')}</Text>
            <TextInput
              value={newTitle}
              onChangeText={setNewTitle}
              placeholder={t('sessions.titlePlaceholder')}
              placeholderTextColor="#6b7280"
              style={styles.input}
            />
            <View style={styles.dialogActions}>
              <Pressable style={styles.cancel} onPress={() => setNewOpen(false)}>
                <Text style={styles.cancelText}>{t('common.cancel')}</Text>
              </Pressable>
              <Pressable style={styles.create} disabled={creating} onPress={() => void create()}>
                <Text style={styles.createText}>{creating ? t('common.creating') : t('common.create')}</Text>
              </Pressable>
            </View>
          </View>
        </Pressable>
      </Modal>
      {loading ? (
        <View style={styles.center}><ActivityIndicator color="#d6d9e0" /></View>
      ) : error ? (
        <View style={styles.center}><Text style={styles.error}>{error}</Text></View>
      ) : (
        <FlatList
          key={columns}
          data={sessions}
          numColumns={columns}
          columnWrapperStyle={columns > 1 ? styles.columns : undefined}
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
              <Text style={styles.empty}>{t('sessions.empty')}</Text>
            </View>
          }
          renderItem={({ item }) => (
            <Pressable
              style={[styles.card, columns > 1 && styles.gridItem]}
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

const makeStyles = (c: ThemeColors) => StyleSheet.create({
  screen: { flex: 1, backgroundColor: c.background },
  header: { flexDirection: 'row', alignItems: 'center', paddingHorizontal: 20, paddingVertical: 12, borderBottomWidth: 1, borderBottomColor: c.border },
  headerText: { flex: 1 },
  newButton: { borderRadius: 7, backgroundColor: c.accent, paddingHorizontal: 12, paddingVertical: 8 },
  newButtonText: { color: c.accentText, fontSize: 11, fontWeight: '600' },
  backdrop: { flex: 1, backgroundColor: 'rgba(0,0,0,0.55)', justifyContent: 'center', padding: 24 },
  dialog: { borderRadius: 12, borderWidth: 1, borderColor: c.border, backgroundColor: c.surface, padding: 18 },
  dialogTitle: { color: c.text, fontSize: 16, fontWeight: '600', marginBottom: 12 },
  input: { borderRadius: 8, borderWidth: 1, borderColor: c.borderStrong, backgroundColor: c.surfaceAlt, color: c.text, paddingHorizontal: 12, paddingVertical: 10, fontSize: 13 },
  dialogActions: { flexDirection: 'row', justifyContent: 'flex-end', gap: 10, marginTop: 16 },
  cancel: { borderRadius: 7, paddingHorizontal: 12, paddingVertical: 9 },
  cancelText: { color: c.textMuted, fontSize: 12 },
  create: { borderRadius: 7, backgroundColor: c.accent, paddingHorizontal: 14, paddingVertical: 9 },
  createText: { color: c.accentText, fontSize: 12, fontWeight: '600' },
  brand: { color: c.textMuted, fontSize: 10, letterSpacing: 2, textTransform: 'uppercase' },
  title: { color: c.text, fontSize: 20, fontWeight: '600', marginTop: 2 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', padding: 24 },
  error: { color: c.danger, fontSize: 13, textAlign: 'center' },
  empty: { color: c.textMuted, fontSize: 13 },
  list: { padding: 16, gap: 8, paddingBottom: 110 },
  columns: { gap: 10 },
  gridItem: { flex: 1 },
  card: {
    borderRadius: 10, borderWidth: 1, borderColor: c.border, backgroundColor: c.surface, padding: 14 },
  cardTitle: { color: c.text, fontSize: 14 },
  cardMeta: { color: c.textMuted, fontSize: 11, marginTop: 6 },
})
