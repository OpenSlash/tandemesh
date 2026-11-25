import { useEffect, useState } from 'react'
import {
  ActivityIndicator,
  FlatList,
  Modal,
  Pressable,
  SafeAreaView,
  StyleSheet,
  Text,
  View,
} from 'react-native'
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
  const [switcher, setSwitcher] = useState(false)

  useEffect(() => {
    let cancelled = false
    const load = async () => {
      try {
        const result = await client.tasks(workspace.id)
        if (!cancelled) setTasks(result.items)
      } catch (nextError) {
        if (!cancelled) setError(nextError instanceof Error ? nextError.message : 'Failed to load tasks')
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    void load()
    return () => {
      cancelled = true
    }
  }, [client, workspace.id])

  return (
    <SafeAreaView style={styles.screen}>
      <View style={styles.header}>
        <Text style={styles.brand}>Tandemesh</Text>
        <Pressable onPress={() => setSwitcher(true)}>
          <Text style={styles.title}>{workspace.name} ▾</Text>
        </Pressable>
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
  backdrop: { flex: 1, backgroundColor: 'rgba(0,0,0,0.55)', justifyContent: 'flex-end' },
  sheet: { backgroundColor: '#10131c', borderTopLeftRadius: 14, borderTopRightRadius: 14, padding: 16, paddingBottom: 28 },
  option: { borderRadius: 8, paddingHorizontal: 12, paddingVertical: 12 },
  optionActive: { backgroundColor: '#1d2331' },
  optionText: { color: '#f3f4f6', fontSize: 14 },
})
