import { useEffect, useMemo, useState } from 'react'
import {
  ActivityIndicator,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native'
import { TandemeshClient, type Run, type Task, type TaskComment } from '../lib/api'
import { useRouter } from '../lib/RouterContext'
import { useTheme, type ThemeColors } from '../lib/theme'
import { useSession } from '../lib/SessionContext'

const STATUSES: { value: string; label: string }[] = [
  { value: 'open', label: 'Open' },
  { value: 'in_progress', label: 'In progress' },
  { value: 'done', label: 'Done' },
]

export function TaskDetailScreen() {
  const { params } = useRouter()
  const { colors } = useTheme()
  const styles = useMemo(() => makeStyles(colors), [colors])

  const id = params?.id ?? ''
  const { host, token } = useSession()
  const router = useRouter()
  const [client] = useState(() => new TandemeshClient(host, token))
  const [task, setTask] = useState<Task | null>(null)
  const [runs, setRuns] = useState<Run[]>([])
  const [comments, setComments] = useState<TaskComment[]>([])
  const [commentBody, setCommentBody] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let cancelled = false
    const load = async () => {
      try {
        const [detail, commentList] = await Promise.all([client.task(id), client.comments(id)])
        if (!cancelled) {
          setTask(detail.task)
          setRuns(detail.runs ?? [])
          setComments(commentList.items)
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

  const updateStatus = async (status: string) => {
    if (!task) return
    setSaving(true)
    try {
      const updated = await client.patchTask(task.id, status)
      setTask(updated)
      setError('')
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : 'Failed to update status')
    } finally {
      setSaving(false)
    }
  }

  const sendComment = async () => {
    const body = commentBody.trim()
    if (!body) return
    setSaving(true)
    try {
      const comment = await client.addComment(id, body)
      setComments((current) => [...current, comment])
      setCommentBody('')
      setError('')
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : 'Failed to send comment')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <View style={styles.center}>
        <ActivityIndicator color="#d6d9e0" />
      </View>
    )
  }
  if (!task) {
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

      <View style={styles.statusRow}>
        {STATUSES.map((status) => (
          <Pressable
            key={status.value}
            style={[styles.statusButton, task.status === status.value && styles.statusActive]}
            disabled={saving}
            onPress={() => void updateStatus(status.value)}
          >
            <Text style={styles.statusText}>{status.label}</Text>
          </Pressable>
        ))}
      </View>

      {error ? <Text style={styles.error}>{error}</Text> : null}

      <Text style={styles.section}>Runs</Text>
      {runs.length === 0 ? (
        <Text style={styles.empty}>No runs yet.</Text>
      ) : (
        runs.map((run) => (
          <Pressable
            key={run.id}
            style={styles.run}
            onPress={() => router.navigate('RunDetail', { id: run.id })}
          >
            <Text style={styles.runStatus}>{run.status}</Text>
            <Text style={styles.runMeta}>{run.agentId || run.id}</Text>
          </Pressable>
        ))
      )}

      <Text style={styles.section}>Comments</Text>
      {comments.length === 0 ? <Text style={styles.empty}>No comments yet.</Text> : null}
      {comments.map((comment) => (
        <View key={comment.id} style={styles.comment}>
          <Text style={styles.commentMeta}>{comment.author || 'unknown'}</Text>
          <Text style={styles.commentBody}>{comment.body}</Text>
        </View>
      ))}
      <TextInput
        value={commentBody}
        onChangeText={setCommentBody}
        placeholder="Write a comment…"
        placeholderTextColor="#6b7280"
        multiline
        style={styles.input}
      />
      <Pressable
        style={[styles.send, (!commentBody.trim() || saving) && styles.sendDisabled]}
        disabled={!commentBody.trim() || saving}
        onPress={() => void sendComment()}
      >
        <Text style={styles.sendText}>{saving ? 'Sending…' : 'Send comment'}</Text>
      </Pressable>
    </ScrollView>
  )
}

const makeStyles = (c: ThemeColors) => StyleSheet.create({
  screen: { flex: 1, backgroundColor: c.background },
  content: { padding: 20, paddingBottom: 40 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: c.background, padding: 24 },
  error: { color: c.danger, fontSize: 12, marginTop: 12 },
  title: { color: c.text, fontSize: 22, fontWeight: '600', lineHeight: 30 },
  metaRow: { flexDirection: 'row', alignItems: 'center', gap: 8, marginTop: 12 },
  badge: { overflow: 'hidden', borderRadius: 5, backgroundColor: '#1a1f2b', color: c.textSecondary, paddingHorizontal: 7, paddingVertical: 3, fontSize: 11 },
  id: { color: c.textSoft, fontSize: 11 },
  statusRow: { flexDirection: 'row', flexWrap: 'wrap', gap: 8, marginTop: 16 },
  statusButton: { borderRadius: 7, borderWidth: 1, borderColor: '#2a3040', paddingHorizontal: 11, paddingVertical: 7 },
  statusActive: { borderColor: c.borderStrong, backgroundColor: c.surfaceAlt },
  statusText: { color: c.textSecondary, fontSize: 12 },
  section: { color: c.text, fontSize: 15, fontWeight: '600', marginTop: 28, marginBottom: 8 },
  empty: { color: c.textMuted, fontSize: 12 },
  run: { borderRadius: 8, borderWidth: 1, borderColor: c.border, backgroundColor: c.surface, padding: 12, marginBottom: 8 },
  runStatus: { color: c.text, fontSize: 13 },
  runMeta: { color: c.textMuted, fontSize: 11, marginTop: 4 },
  comment: { borderRadius: 8, backgroundColor: c.surface, padding: 12, marginBottom: 8 },
  commentMeta: { color: c.textSoft, fontSize: 10.5, marginBottom: 4 },
  commentBody: { color: c.textSecondary, fontSize: 12, lineHeight: 18 },
  input: {
    marginTop: 8,
    minHeight: 76,
    borderRadius: 8,
    borderWidth: 1,
    borderColor: c.borderStrong,
    backgroundColor: c.surfaceAlt,
    color: c.text,
    paddingHorizontal: 12,
    paddingVertical: 10,
    fontSize: 13,
    textAlignVertical: 'top',
  },
  send: { marginTop: 10, alignSelf: 'flex-end', borderRadius: 7, backgroundColor: c.accent, paddingHorizontal: 14, paddingVertical: 9 },
  sendDisabled: { opacity: 0.4 },
  sendText: { color: c.accentText, fontSize: 12, fontWeight: '600' },
})
