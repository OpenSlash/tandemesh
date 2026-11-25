import { useEffect, useRef, useState } from 'react'
import {
  ActivityIndicator,
  Pressable,
  ScrollView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native'
import type { NativeStackScreenProps } from '@react-navigation/native-stack'
import { TandemeshClient, type AgentSession, type SessionMessage } from '../lib/api'
import { useSession } from '../lib/SessionContext'
import type { RootStackParamList } from '../navigation/types'

type Props = NativeStackScreenProps<RootStackParamList, 'SessionDetail'>

export function SessionDetailScreen({ route }: Props) {
  const { id } = route.params
  const { host, token } = useSession()
  const [client] = useState(() => new TandemeshClient(host, token))
  const [session, setSession] = useState<AgentSession | null>(null)
  const [messages, setMessages] = useState<SessionMessage[]>([])
  const [draft, setDraft] = useState('')
  const [sending, setSending] = useState(false)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)
  const scrollRef = useRef<ScrollView>(null)

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

  const send = async () => {
    const prompt = draft.trim()
    if (!prompt) return
    setSending(true)
    try {
      await client.createSessionRun(id, prompt)
      setDraft('')
      await loadMessages()
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
        {messages.length === 0 ? <Text style={styles.empty}>No messages yet.</Text> : null}
        {messages.map((message, index) => (
          <View
            key={message.id ?? `${message.runId}-${message.ts}-${index}`}
            style={[
              styles.message,
              message.role === 'user' ? styles.userMessage : styles.agentMessage,
              message.kind === 'tool' ? styles.toolMessage : null,
            ]}
          >
            <Text style={styles.role}>{message.role || message.kind}</Text>
            <Text style={styles.text}>{message.text}</Text>
          </View>
        ))}
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
          style={[styles.send, (!draft.trim() || sending) && styles.sendDisabled]}
          disabled={!draft.trim() || sending}
          onPress={() => void send()}
        >
          <Text style={styles.sendText}>{sending ? 'Sending…' : 'Send'}</Text>
        </Pressable>
      </View>
    </View>
  )
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#07080f' },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: '#07080f' },
  header: { paddingHorizontal: 16, paddingVertical: 12, borderBottomWidth: 1, borderBottomColor: '#202433' },
  title: { color: '#f3f4f6', fontSize: 16, fontWeight: '600' },
  meta: { color: '#8b94a7', fontSize: 11, marginTop: 3 },
  messages: { flex: 1 },
  messagesContent: { padding: 14, gap: 8 },
  empty: { color: '#8b94a7', fontSize: 12, textAlign: 'center', marginTop: 20 },
  message: { maxWidth: '92%', borderRadius: 10, paddingHorizontal: 12, paddingVertical: 9 },
  userMessage: { alignSelf: 'flex-end', backgroundColor: '#1d2331' },
  agentMessage: { alignSelf: 'flex-start', backgroundColor: '#10131c', borderWidth: 1, borderColor: '#202433' },
  toolMessage: { alignSelf: 'stretch', backgroundColor: '#0b0f16', borderWidth: 1, borderColor: '#1a202b' },
  role: { color: '#6e7787', fontSize: 9.5, textTransform: 'uppercase', letterSpacing: 1, marginBottom: 4 },
  text: { color: '#d7dbe4', fontSize: 12.5, lineHeight: 18 },
  error: { color: '#f87171', fontSize: 11, marginTop: 6 },
  composer: { flexDirection: 'row', alignItems: 'flex-end', gap: 8, padding: 12, borderTopWidth: 1, borderTopColor: '#202433' },
  input: {
    flex: 1,
    minHeight: 42,
    maxHeight: 120,
    borderRadius: 9,
    borderWidth: 1,
    borderColor: '#262b3a',
    backgroundColor: '#0b0d14',
    color: '#f3f4f6',
    paddingHorizontal: 12,
    paddingVertical: 9,
    fontSize: 13,
  },
  send: { borderRadius: 8, backgroundColor: '#d6d9e0', paddingHorizontal: 16, paddingVertical: 11 },
  sendDisabled: { opacity: 0.4 },
  sendText: { color: '#0b0d12', fontSize: 12, fontWeight: '600' },
})
