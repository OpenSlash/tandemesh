import { useEffect, useState } from 'react'
import {
  ActivityIndicator,
  FlatList,
  Pressable,
  SafeAreaView,
  StyleSheet,
  Text,
  View,
} from 'react-native'
import { useNavigation } from '@react-navigation/native'
import type { NativeStackNavigationProp } from '@react-navigation/native-stack'
import { TandemeshClient, type Task, type Workspace } from '../lib/api'
import { useSession } from '../lib/SessionContext'
import type { RootStackParamList } from '../navigation/types'

export function TasksScreen() {
  const { host, token } = useSession()
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>()
  const [client] = useState(() => new TandemeshClient(host, token))
  const [workspace, setWorkspace] = useState<Workspace | null>(null)
  const [tasks, setTasks] = useState<Task[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    const load = async () => {
      try {
        const workspaces = await client.workspaces()
        const first = workspaces.items[0]
        if (!first) throw new Error('No workspace available')
        const result = await client.tasks(first.id)
        if (cancelled) return
        setWorkspace(first)
        setTasks(result.items)
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
  }, [client])

  return (
    <SafeAreaView style={styles.screen}>
      <View style={styles.header}>
        <Text style={styles.brand}>Tandemesh</Text>
        <Text style={styles.title}>{workspace?.name ?? 'Tasks'}</Text>
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
})
