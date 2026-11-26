import { useEffect, useMemo, useState } from 'react'
import {
  ActivityIndicator,
  FlatList,
  Modal,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  View,
  useWindowDimensions,
} from 'react-native'
import { Screen } from '../components/Screen'
import { useTheme, type ThemeColors } from '../lib/theme'
import { TandemeshClient, type Task } from '../lib/api'
import { useSession } from '../lib/SessionContext'
import { useWorkspace } from '../lib/WorkspaceContext'
import { useRouter } from '../lib/RouterContext'
import { useI18n } from '../lib/i18n'

export function TasksScreen() {
  const { host, token } = useSession()
  const { workspace, workspaces, selectWorkspace } = useWorkspace()
  const router = useRouter()
  const { t } = useI18n()
  const { colors } = useTheme()
  const styles = useMemo(() => makeStyles(colors), [colors])

  const [client] = useState(() => new TandemeshClient(host, token))
  const [tasks, setTasks] = useState<Task[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const { width } = useWindowDimensions()
  const columns = width >= 700 ? 2 : 1
  const [switcher, setSwitcher] = useState(false)
  const [filter, setFilter] = useState('')

  const load = async (reset: boolean) => {
    try {
      const result = await client.tasks(workspace.id, filter || undefined)
      setTasks(result.items)
      setError('')
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : 'Failed to load tasks')
    } finally {
      if (reset) setLoading(false)
      setRefreshing(false)
    }
  }

  useEffect(() => {
    setLoading(true)
    void load(true)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [client, workspace.id, filter])

  return (
    <Screen>
      <View style={styles.header}>
        <Text style={styles.brand}>Tandemesh</Text>
        <Pressable onPress={() => setSwitcher(true)}>
          <Text style={styles.title}>{workspace.name} ▾</Text>
        </Pressable>
      </View>

      <View style={styles.filters}>
        {[
          ['', t('tasks.filter.all')],
          ['open', t('tasks.filter.open')],
          ['in_progress', t('tasks.filter.progress')],
          ['done', t('tasks.filter.done')],
        ].map(([value, label]) => (
          <Pressable
            key={value}
            style={[styles.filter, filter === value && styles.filterActive]}
            onPress={() => setFilter(value)}
          >
            <Text style={styles.filterText}>{label}</Text>
          </Pressable>
        ))}
      </View>

      <Modal visible={switcher} transparent animationType="fade" onRequestClose={() => setSwitcher(false)}>
        <Pressable style={styles.backdrop} onPress={() => setSwitcher(false)}>
          <View style={styles.sheet}>
            {workspaces.map((item) => (
              <Pressable
                key={item.id}
                style={[styles.option, item.id === workspace.id && styles.optionActive]}
                onPress={() => {
                  selectWorkspace(item.id)
                  setSwitcher(false)
                }}
              >
                <Text style={styles.optionText}>{item.name}</Text>
              </Pressable>
            ))}
          </View>
        </Pressable>
      </Modal>

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
          key={columns}
          data={tasks}
          numColumns={columns}
          columnWrapperStyle={columns > 1 ? styles.columns : undefined}
          keyExtractor={(task) => task.id}
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
              <Text style={styles.empty}>{t('tasks.empty')}</Text>
            </View>
          }
          renderItem={({ item }) => (
            <Pressable
              style={[styles.task, columns > 1 && styles.gridItem]}
              onPress={() => router.navigate('TaskDetail', { id: item.id })}
            >
              <Text style={styles.taskTitle}>{item.title || item.id}</Text>
              <Text style={styles.taskMeta}>{item.status}</Text>
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
  error: { color: c.danger, fontSize: 13, textAlign: 'center' },
  empty: { color: c.textMuted, fontSize: 13 },
  list: { padding: 16, gap: 8, paddingBottom: 110 },
  columns: { gap: 10 },
  gridItem: { flex: 1 },
  task: {
    borderRadius: 10, borderWidth: 1, borderColor: c.border, backgroundColor: c.surface, padding: 14 },
  taskTitle: { color: c.text, fontSize: 14 },
  taskMeta: { color: c.textMuted, fontSize: 11, marginTop: 6, textTransform: 'lowercase' },
  filters: { flexDirection: 'row', gap: 6, paddingHorizontal: 16, paddingTop: 12 },
  filter: { borderRadius: 6, borderWidth: 1, borderColor: c.borderStrong, paddingHorizontal: 10, paddingVertical: 6 },
  filterActive: { borderColor: c.borderStrong, backgroundColor: c.surfaceAlt },
  filterText: { color: c.textSecondary, fontSize: 11 },
  backdrop: { flex: 1, backgroundColor: 'rgba(0,0,0,0.55)', justifyContent: 'flex-end' },
  sheet: { backgroundColor: c.surface, borderTopLeftRadius: 14, borderTopRightRadius: 14, padding: 16, paddingBottom: 28 },
  option: { borderRadius: 8, paddingHorizontal: 12, paddingVertical: 12 },
  optionActive: { backgroundColor: c.userBubble },
  optionText: { color: c.text, fontSize: 14 },
})
