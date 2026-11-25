import { useEffect, useState } from 'react'
import { ActivityIndicator, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native'
import { useNavigation } from '@react-navigation/native'
import type { NativeStackNavigationProp, NativeStackScreenProps } from '@react-navigation/native-stack'
import { TandemeshClient, type Run } from '../lib/api'
import { useSession } from '../lib/SessionContext'
import type { RootStackParamList } from '../navigation/types'

type Props = NativeStackScreenProps<RootStackParamList, 'RunDetail'>

interface TranscriptEntry {
  kind: string
  text: string
  tsMs: number
  seq: number
}

export function RunDetailScreen({ route }: Props) {
  const { id } = route.params
  const { host, token } = useSession()
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>()
  const [client] = useState(() => new TandemeshClient(host, token))
  const [run, setRun] = useState<Run | null>(null)
  const [entries, setEntries] = useState<TranscriptEntry[]>([])
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    const load = async () => {
      try {
        const [runResult, transcript] = await Promise.all([
          client.run(id),
          client.runTranscript(id),
        ])
        if (!cancelled) {
          setRun(runResult)
          setEntries(transcript.items ?? [])
        }
      } catch (nextError) {
        if (!cancelled) setError(nextError instanceof Error ? nextError.message : 'Failed to load run')
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
  if (error || !run) {
    return (
      <View style={styles.center}>
        <Text style={styles.error}>{error || 'Run not found'}</Text>
      </View>
    )
  }

  return (
    <ScrollView style={styles.screen} contentContainerStyle={styles.content}>
      <Text style={styles.title}>{run.status}</Text>
      <Text style={styles.meta}>{run.agentId || run.id}</Text>
      <Pressable style={styles.diffButton} onPress={() => navigation.navigate('Diff', { id: run.id })}>\n        <Text style={styles.diffButtonText}>View diff</Text>
      </Pressable>
      {run.errorMessage ? <Text style={styles.error}>{run.errorMessage}</Text> : null}

      {run.prompt ? (
        <>
          <Text style={styles.section}>Prompt</Text>
          <Text style={styles.prompt}>{run.prompt}</Text>
        </>
      ) : null}

      <Text style={styles.section}>Transcript</Text>
      {entries.length === 0 ? (
        <Text style={styles.empty}>No transcript yet.</Text>
      ) : (
        entries.map((entry) => (
          <View key={`${entry.seq}-${entry.kind}`} style={styles.entry}>
            <Text style={styles.kind}>{entry.kind}</Text>
            <Text style={styles.text}>{entry.text}</Text>
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
  title: { color: '#f3f4f6', fontSize: 22, fontWeight: '600', textTransform: 'capitalize' },
  meta: { color: '#8b94a7', fontSize: 11, marginTop: 6 },
  diffButton: { alignSelf: 'flex-start', marginTop: 14, borderRadius: 7, borderWidth: 1, borderColor: '#2a3040', paddingHorizontal: 12, paddingVertical: 7 },
  diffButtonText: { color: '#d7dbe4', fontSize: 12 },
  error: { color: '#f87171', fontSize: 12, marginTop: 12 },
  section: { color: '#f3f4f6', fontSize: 15, fontWeight: '600', marginTop: 26, marginBottom: 8 },
  prompt: { color: '#c7ccd8', fontSize: 12, lineHeight: 18 },
  empty: { color: '#8b94a7', fontSize: 12 },
  entry: { borderRadius: 8, borderWidth: 1, borderColor: '#202433', backgroundColor: '#10131c', padding: 10, marginBottom: 8 },
  kind: { color: '#6e7787', fontSize: 10, textTransform: 'uppercase', letterSpacing: 1 },
  text: { color: '#d7dbe4', fontSize: 12, lineHeight: 18, marginTop: 4 },
})
