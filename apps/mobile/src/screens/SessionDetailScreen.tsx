import { useEffect, useMemo, useRef, useState } from 'react'
import {
  ActivityIndicator,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native'
import { TandemeshClient, type AgentSession, type SessionMessage } from '../lib/api'
import { useRouter } from '../lib/RouterContext'
import { groupSessionMessages } from '../lib/messages'
import { useTheme, type ThemeColors } from '../lib/theme'
import { useSession } from '../lib/SessionContext'

export function SessionDetailScreen() {
  const { params } = useRouter()
  const id = params?.id ?? ''
  const { host, token } = useSession()
  const { colors } = useTheme()
  const styles = useMemo(() => makeStyles(colors), [colors])

  const [client] = useState(() => new TandemeshClient(host, token))
  const [session, setSession] = useState<AgentSession | null>(null)
  const [messages, setMessages] = useState<SessionMessage[]>([])
  const [draft, setDraft] = useState('')
  const [sending, setSending] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const [expandedThinking, setExpandedThinking] = useState<Record<string, boolean>>({})
  const [runningRunId, setRunningRunId] = useState<string | null>(null)
  const scrollRef = useRef<ScrollView>(null)
  const displayMessages = groupSessionMessages(messages)
  const assistantStarted = runningRunId
    ? messages.some((message) => message.runId === runningRunId && message.role === 'assistant')
    : false

  const loadMessages = async () => {
    try {
      const result = await client.sessionMessages(id)
      setMessages(result.messages ?? [])
      setError('')
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : 'Failed to load messages')
    }
  }

  useEffect(() => {
    let cancelled = false
    const load = async () => {
      try {
        const [detail] = await Promise.all([client.session(id), loadMessages()])
        if (!cancelled) setSession(detail.session)
      } catch (nextError) {
        if (!cancelled) setError(nextError instanceof Error ? nextError.message : 'Failed to load session')
      } finally {
        if (!cancelled) setLoading(false)
      }
    }
    void load()
    const timer = setInterval(() => void loadMessages(), 4000)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [client, id])

  useEffect(() => {
    scrollRef.current?.scrollToEnd({ animated: true })
  }, [messages.length])

  useEffect(() => {
    if (!runningRunId) return
    let cancelled = false
    const tick = async () => {
      try {
        const run = await client.run(runningRunId)
        if (cancelled) return
        await loadMessages()
        if (run.status !== 'running' && run.status !== 'queued' && run.status !== 'leased') {
          setRunningRunId(null)
        }
      } catch {
        // keep polling; transient network failures are expected
      }
    }
    void tick()
    const timer = setInterval(() => void tick(), 1200)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [client, runningRunId])

  const send = async () => {
    const prompt = draft.trim()
    if (!prompt) return
    setSending(true)
    try {
      const run = await client.createSessionRun(id, prompt)
      setDraft('')
      await loadMessages()
      setRunningRunId(run.id)
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : 'Failed to send message')
    } finally {
      setSending(false)
    }
  }

  if (loading) {
    return (
      <View style={styles.center}>
        <ActivityIndicator color="#d6d9e0" />
      </View>
    )
  }

  return (
    <View style={styles.screen}>
      <View style={styles.header}>
        <Text style={styles.title}>{session?.title || id}</Text>
        <Text style={styles.meta}>
          {session?.backend || session?.agentId} · {session?.lastRunStatus || session?.status}
        </Text>
      </View>

      <ScrollView ref={scrollRef} style={styles.messages} contentContainerStyle={styles.messagesContent}>
        {displayMessages.length === 0 ? <Text style={styles.empty}>No messages yet.</Text> : null}
        {displayMessages.map((message) =>
          message.kind === 'reasoning' ? (
            <View key={message.id} style={styles.thinking}>
              <Pressable
                style={styles.thinkingHeader}
                onPress={() =>
                  setExpandedThinking((current) => ({ ...current, [message.id]: !current[message.id] }))
                }
              >
                <Text style={styles.thinkingLabel}>
                  {expandedThinking[message.id] ? 'Hide thinking' : 'Show thinking'}
                </Text>
                <Text style={styles.thinkingChevron}>{expandedThinking[message.id] ? '▾' : '▸'}</Text>
              </Pressable>
              {expandedThinking[message.id] ? (
                <Text style={styles.thinkingText}>{message.text}</Text>
              ) : null}
            </View>
          ) : (
            <View
              key={message.id}
              style={[
                styles.message,
                message.role === 'user' ? styles.userMessage : styles.agentMessage,
                message.kind === 'tool' ? styles.toolMessage : null,
              ]}
            >
              <Text style={styles.role}>{message.role || message.kind}</Text>
              <Text style={styles.text}>{message.text}</Text>
            </View>
          ),
        )}
        {runningRunId ? (
          <View style={styles.statusRow}>
            <ActivityIndicator size="small" color="#7f8ba0" />
            <Text style={styles.statusText}>{assistantStarted ? 'Responding…' : 'Thinking…'}</Text>
          </View>
        ) : null}
        {error ? <Text style={styles.error}>{error}</Text> : null}
      </ScrollView>

      <View style={styles.composer}>
        <TextInput
          value={draft}
          onChangeText={setDraft}
          placeholder="Message the agent…"
          placeholderTextColor="#6b7280"
          multiline
          style={styles.input}
        />
        <Pressable
          style={[styles.send, (!draft.trim() || sending || Boolean(runningRunId)) && styles.sendDisabled]}
          disabled={!draft.trim() || sending || Boolean(runningRunId)}
          onPress={() => void send()}
        >
          <Text style={styles.sendText}>{sending ? 'Sending…' : 'Send'}</Text>
        </Pressable>
      </View>
    </View>
  )
}

const makeStyles = (c: ThemeColors) => StyleSheet.create({
  screen: { flex: 1, backgroundColor: c.background },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: c.background },
  header: { paddingHorizontal: 16, paddingVertical: 12, borderBottomWidth: 1, borderBottomColor: c.border },
  title: { color: c.text, fontSize: 16, fontWeight: '600' },
  meta: { color: c.textMuted, fontSize: 11, marginTop: 3 },
  messages: { flex: 1 },
  messagesContent: { padding: 14, gap: 8 },
  empty: { color: c.textMuted, fontSize: 12, textAlign: 'center', marginTop: 20 },
  statusRow: { flexDirection: 'row', alignItems: 'center', gap: 8, paddingVertical: 8 },
  statusText: { color: c.textMuted, fontSize: 11.5 },
  message: { maxWidth: '92%', borderRadius: 10, paddingHorizontal: 12, paddingVertical: 9 },
  userMessage: { alignSelf: 'flex-end', backgroundColor: c.userBubble },
  agentMessage: { alignSelf: 'stretch', maxWidth: '100%', backgroundColor: 'transparent', borderWidth: 0, borderRadius: 0, paddingHorizontal: 0, paddingVertical: 2 },
  toolMessage: { alignSelf: 'stretch', backgroundColor: c.codeBg, borderWidth: 1, borderColor: c.codeBorder },
  thinking: { alignSelf: 'stretch', paddingVertical: 2 },
  thinkingHeader: { flexDirection: 'row', alignItems: 'center', gap: 6, paddingVertical: 4 },
  thinkingLabel: { color: c.textSoft, fontSize: 11.5 },
  thinkingChevron: { color: c.textSoft, fontSize: 11 },
  thinkingText: { color: c.textMuted, fontSize: 11.5, lineHeight: 17, paddingVertical: 4 },
  role: { color: c.textSoft, fontSize: 9.5, textTransform: 'uppercase', letterSpacing: 1, marginBottom: 4 },
  text: { color: c.textSecondary, fontSize: 12.5, lineHeight: 18 },
  error: { color: c.danger, fontSize: 11, marginTop: 6 },
  composer: { flexDirection: 'row', alignItems: 'flex-end', gap: 8, padding: 12, borderTopWidth: 1, borderTopColor: c.border },
  input: {
    flex: 1,
    minHeight: 42,
    maxHeight: 120,
    borderRadius: 9,
    borderWidth: 1,
    borderColor: c.borderStrong,
    backgroundColor: c.surfaceAlt,
    color: c.text,
    paddingHorizontal: 12,
    paddingVertical: 9,
    fontSize: 13,
  },
  send: { borderRadius: 8, backgroundColor: c.accent, paddingHorizontal: 16, paddingVertical: 11 },
  sendDisabled: { opacity: 0.4 },
  sendText: { color: c.accentText, fontSize: 12, fontWeight: '600' },
})
