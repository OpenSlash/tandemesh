import { useEffect, useState } from 'react'
import { ActivityIndicator, ScrollView, StyleSheet, Text, View } from 'react-native'
import type { NativeStackScreenProps } from '@react-navigation/native-stack'
import { TandemeshClient, type Run, type Task } from '../lib/api'
import { useSession } from '../lib/SessionContext'
import type { RootStackParamList } from '../navigation/types'

type Props = NativeStackScreenProps<RootStackParamList, 'TaskDetail'>

export function TaskDetailScreen({ route }: Props) {
  const { id } = route.params
  const { host, token } = useSession()
  const [client] = useState(() => new TandemeshClient(host, token))
  const [task, setTask] = useState<Task | null>(null)
  const [runs, setRuns] = useState<Run[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    const load = async () => {
      try {
        const result = await client.task(id)
        if (!cancelled) {
          setTask(result.task)
          setRuns(result.runs ?? [])
        }
      } catch (nextError) {
        if (!cancelled) setError(nextError instanceof Error ? nextError.message : 'Failed to load task')
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    void load()
    return () => {
      cancelled = true
    }
  }, [client, id])

  if (loading) {
    return (
      <View style={styles.center}>
        <ActivityIndicator color="#d6d9e0" />
      </View>
    )
  }
  if (error || !task) {
    return (
      <View style={styles.center}>
        <Text style={styles.error}>{error || 'Task not found'}</Text>
      </View>
    )
  }

  return (
    <ScrollView style={styles.screen} contentContainerStyle={styles.content}>
      <Text style={styles.title}>{task.title || task.id}</Text>
      <View style={styles.metaRow}>
        <Text style={styles.badge}>{task.status}</Text>
        <Text style={styles.id}>{task.id}</Text>
      </View>

      <Text style={styles.section}>Runs</Text>
      {runs.length === 0 ? (
        <Text style={styles.empty}>No runs yet.</Text>
      ) : (
        runs.map((run) => (
          <View key={run.id} style={styles.run}>
            <Text style={styles.runStatus}>{run.status}</Text>
            <Text style={styles.runMeta}>{run.agentId || run.id}</Text>
          </View>
        ))
      )}
    </ScrollView>
  )
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#07080f' },
  content: { padding: 20 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: '#07080f', padding: 24 },
  error: { color: '#f87171', fontSize: 13, textAlign: 'center' },
  title: { color: '#f3f4f6', fontSize: 22, fontWeight: '600', lineHeight: 30 },
  metaRow: { flexDirection: 'row', alignItems: 'center', gap: 8, marginTop: 12 },
  badge: { overflow: 'hidden', borderRadius: 5, backgroundColor: '#1a1f2b', color: '#c7ccd8', paddingHorizontal: 7, paddingVertical: 3, fontSize: 11 },
  id: { color: '#6e7787', fontSize: 11 },
  section: { color: '#f3f4f6', fontSize: 15, fontWeight: '600', marginTop: 28, marginBottom: 8 },
  empty: { color: '#8b94a7', fontSize: 12 },
  run: { borderRadius: 8, borderWidth: 1, borderColor: '#202433', backgroundColor: '#10131c', padding: 12, marginBottom: 8 },
  runStatus: { color: '#f3f4f6', fontSize: 13 },
  runMeta: { color: '#8b94a7', fontSize: 11, marginTop: 4 },
})
