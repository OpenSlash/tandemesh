import { useEffect, useMemo, useState } from 'react'
import { ActivityIndicator, Pressable, ScrollView, StyleSheet, Text, View } from 'react-native'
import { TandemeshClient, type Run } from '../lib/api'
import { useRouter } from '../lib/RouterContext'
import { useTheme, type ThemeColors } from '../lib/theme'
import { useSession } from '../lib/SessionContext'

interface TranscriptEntry {
  kind: string
  text: string
  tsMs: number
  seq: number
}

export function RunDetailScreen() {
  const { params } = useRouter()
  const { colors } = useTheme()
  const styles = useMemo(() => makeStyles(colors), [colors])

  const id = params?.id ?? ''
  const { host, token } = useSession()
  const router = useRouter()
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
      <Pressable style={styles.diffButton} onPress={() => router.navigate('Diff', { id: run.id })}>\n        <Text style={styles.diffButtonText}>View diff</Text>
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

const makeStyles = (c: ThemeColors) => StyleSheet.create({
  screen: { flex: 1, backgroundColor: c.background },
  content: { padding: 20 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: c.background, padding: 24 },
  title: { color: c.text, fontSize: 22, fontWeight: '600', textTransform: 'capitalize' },
  meta: { color: c.textMuted, fontSize: 11, marginTop: 6 },
  diffButton: { alignSelf: 'flex-start', marginTop: 14, borderRadius: 7, borderWidth: 1, borderColor: '#2a3040', paddingHorizontal: 12, paddingVertical: 7 },
  diffButtonText: { color: c.textSecondary, fontSize: 12 },
  error: { color: c.danger, fontSize: 12, marginTop: 12 },
  section: { color: c.text, fontSize: 15, fontWeight: '600', marginTop: 26, marginBottom: 8 },
  prompt: { color: c.textSecondary, fontSize: 12, lineHeight: 18 },
  empty: { color: c.textMuted, fontSize: 12 },
  entry: { borderRadius: 8, borderWidth: 1, borderColor: c.border, backgroundColor: c.surface, padding: 10, marginBottom: 8 },
  kind: { color: c.textSoft, fontSize: 10, textTransform: 'uppercase', letterSpacing: 1 },
  text: { color: c.textSecondary, fontSize: 12, lineHeight: 18, marginTop: 4 },
})
