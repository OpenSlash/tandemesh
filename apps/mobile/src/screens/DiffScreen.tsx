import { useEffect, useState } from 'react'
import { ActivityIndicator, FlatList, Platform, StyleSheet, Text, View } from 'react-native'
import type { NativeStackScreenProps } from '@react-navigation/native-stack'
import { TandemeshClient } from '../lib/api'
import { useSession } from '../lib/SessionContext'
import type { RootStackParamList } from '../navigation/types'

type Props = NativeStackScreenProps<RootStackParamList, 'Diff'>

function lineStyle(line: string) {
  if (line.startsWith('@@')) return styles.hunk
  if (line.startsWith('+')) return styles.added
  if (line.startsWith('-')) return styles.removed
  return styles.context
}

export function DiffScreen({ route }: Props) {
  const { id } = route.params
  const { host, token } = useSession()
  const [client] = useState(() => new TandemeshClient(host, token))
  const [lines, setLines] = useState<string[]>([])
  const [truncated, setTruncated] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    const load = async () => {
      try {
        const result = await client.runDiff(id)
        if (!cancelled) {
          setLines(result.diff ? result.diff.split('\n') : [])
          setTruncated(result.truncated)
        }
      } catch (nextError) {
        if (!cancelled) setError(nextError instanceof Error ? nextError.message : 'Failed to load diff')
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
  if (error) {
    return (
      <View style={styles.center}>
        <Text style={styles.error}>{error}</Text>
      </View>
    )
  }

  return (
    <View style={styles.screen}>
      {truncated ? <Text style={styles.truncated}>Diff truncated by the server.</Text> : null}
      {lines.length === 0 ? (
        <View style={styles.center}>
          <Text style={styles.empty}>No changes.</Text>
        </View>
      ) : (
        <FlatList
          data={lines}
          keyExtractor={(_, index) => String(index)}
          initialNumToRender={80}
          renderItem={({ item }) => <Text style={[styles.line, lineStyle(item)]}>{item || ' '}</Text>}
        />
      )}
    </View>
  )
}

const mono = Platform.select({ ios: 'Menlo', android: 'monospace', default: 'monospace' })

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#07080f' },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: '#07080f', padding: 24 },
  error: { color: '#f87171', fontSize: 13, textAlign: 'center' },
  empty: { color: '#8b94a7', fontSize: 13 },
  truncated: { color: '#d8b46a', backgroundColor: '#1b160d', paddingHorizontal: 14, paddingVertical: 8, fontSize: 11 },
  line: { fontFamily: mono, fontSize: 11, lineHeight: 17, paddingHorizontal: 12 },
  added: { color: '#7bc98b', backgroundColor: '#0f1a12' },
  removed: { color: '#e08b8b', backgroundColor: '#1a0f12' },
  hunk: { color: '#8fb7e0', backgroundColor: '#0f141c' },
  context: { color: '#c7ccd8' },
})
