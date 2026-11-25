import { useEffect, useState } from 'react'
import {
  ActivityIndicator,
  FlatList,
  Modal,
  Pressable,
  RefreshControl,
  StyleSheet,
  Text,
  View,
} from 'react-native'
import { SafeAreaView } from 'react-native-safe-area-context'
import { useNavigation } from '@react-navigation/native'
import type { NativeStackNavigationProp } from '@react-navigation/native-stack'
import { TandemeshClient, type Task } from '../lib/api'
import { useSession } from '../lib/SessionContext'
import { useWorkspace } from '../lib/WorkspaceContext'
import type { RootStackParamList } from '../navigation/types'

export function TasksScreen() {
  const { host, token } = useSession()
  const { workspace, workspaces, selectWorkspace } = useWorkspace()
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>()
  const [client] = useState(() => new TandemeshClient(host, token))
  const [tasks, setTasks] = useState<Task[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
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
    <SafeAreaView style={styles.screen}>
      <View style={styles.header}>
        <Text style={styles.brand}>Tandemesh</Text>
        <Pressable onPress={() => setSwitcher(true)}>
          <Text style={styles.title}>{workspace.name} ▾</Text>
        </Pressable>
      </View>

      <View style={styles.filters}>
        {[
          ['', 'All'],
          ['open', 'Open'],
          ['in_progress', 'In progress'],
          ['done', 'Done'],
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
          data={tasks}
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
              <Text style={styles.empty}>No tasks in this workspace.</Text>
            </View>
          }
          renderItem={({ item }) => (
            <Pressable
              style={styles.task}
              onPress={() => navigation.navigate('TaskDetail', { id: item.id })}
            >
              <Text style={styles.taskTitle}>{item.title || item.id}</Text>
              <Text style={styles.taskMeta}>{item.status}</Text>
            </Pressable>
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
  error: { color: '#f87171', fontSize: 13, textAlign: 'center' },
  empty: { color: '#8b94a7', fontSize: 13 },
  list: { padding: 16, gap: 8 },
  task: { borderRadius: 10, borderWidth: 1, borderColor: '#202433', backgroundColor: '#10131c', padding: 14 },
  taskTitle: { color: '#f3f4f6', fontSize: 14 },
  taskMeta: { color: '#8b94a7', fontSize: 11, marginTop: 6, textTransform: 'lowercase' },
  filters: { flexDirection: 'row', gap: 6, paddingHorizontal: 16, paddingTop: 12 },
  filter: { borderRadius: 6, borderWidth: 1, borderColor: '#262b3a', paddingHorizontal: 10, paddingVertical: 6 },
  filterActive: { borderColor: '#6b7a96', backgroundColor: '#1a2130' },
  filterText: { color: '#c7ccd8', fontSize: 11 },
  backdrop: { flex: 1, backgroundColor: 'rgba(0,0,0,0.55)', justifyContent: 'flex-end' },
  sheet: { backgroundColor: '#10131c', borderTopLeftRadius: 14, borderTopRightRadius: 14, padding: 16, paddingBottom: 28 },
  option: { borderRadius: 8, paddingHorizontal: 12, paddingVertical: 12 },
  optionActive: { backgroundColor: '#1d2331' },
  optionText: { color: '#f3f4f6', fontSize: 14 },
})
