import { useEffect, useState } from 'react'
import {
  ActivityIndicator,
  FlatList,
  Pressable,
  ScrollView,
  StatusBar,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native'
import {
  TandemeshClient,
  exchangePairing,
  type AgentSession,
  type SessionMessage,
  type Task,
  type Workspace,
} from './lib/api'
import { clearSession, loadSession, saveSession, type MobileSession } from './lib/session'

type Tab = 'tasks' | 'sessions' | 'settings'

export default function HarmonyApp() {
  const [session, setSession] = useState<MobileSession | null | undefined>(undefined)

  useEffect(() => {
    void loadSession().then(setSession)
  }, [])

  if (session === undefined) {
    return (
      <View style={styles.center}>
        <StatusBar barStyle="light-content" backgroundColor="#07080f" />
        <ActivityIndicator color="#d6d9e0" />
      </View>
    )
  }
  if (!session) {
    return (
      <Pair
        onPaired={async (host, token) => {
          await saveSession(host, token)
          setSession({ host, token })
        }}
      />
    )
  }
  return (
    <Home
      session={session}
      onSignOut={() => {
        void clearSession()
        setSession(null)
      }}
    />
  )
}

function Pair({ onPaired }: { onPaired: (host: string, token: string) => void }) {
  const [host, setHost] = useState('http://192.168.111.58:8080')
  const [code, setCode] = useState('')
  const [device, setDevice] = useState('HarmonyOS')
  const [error, setError] = useState('')
  const [busy, setBusy] = useState(false)

  const pair = async () => {
    setBusy(true)
    setError('')
    try {
      const result = await exchangePairing(host, code, device, 'harmony')
      onPaired(host, result.token)
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : 'Pairing failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <View style={styles.screen}>
      <StatusBar barStyle="light-content" backgroundColor="#07080f" />
      <View style={styles.card}>
        <Text style={styles.brand}>TANDEMESH</Text>
        <Text style={styles.title}>Pair this device</Text>
        <Text style={styles.hint}>Generate a code on the desktop and enter it below.</Text>
        <Text style={styles.label}>Host</Text>
        <TextInput value={host} onChangeText={setHost} autoCapitalize="none" style={styles.input} />
        <Text style={styles.label}>Pairing code</Text>
        <TextInput
          value={code}
          onChangeText={setCode}
          autoCapitalize="characters"
          placeholder="ABCD-EFGH"
          placeholderTextColor="#6b7280"
          style={styles.input}
        />
        <Text style={styles.label}>Device name</Text>
        <TextInput value={device} onChangeText={setDevice} style={styles.input} />
        {error ? <Text style={styles.error}>{error}</Text> : null}
        <Pressable
          style={[styles.primary, (busy || !code.trim()) && styles.disabled]}
          disabled={busy || !code.trim()}
          onPress={() => void pair()}
        >
          <Text style={styles.primaryText}>{busy ? 'Pairing…' : 'Pair'}</Text>
        </Pressable>
      </View>
    </View>
  )
}

function Home({ session, onSignOut }: { session: MobileSession; onSignOut: () => void }) {
  const [client] = useState(() => new TandemeshClient(session.host, session.token))
  const [workspace, setWorkspace] = useState<Workspace | null>(null)
  const [tab, setTab] = useState<Tab>('tasks')
  const [error, setError] = useState('')

  useEffect(() => {
    client
      .workspaces()
      .then((result) => setWorkspace(result.items[0] ?? null))
      .catch((nextError) => setError(nextError instanceof Error ? nextError.message : 'Failed to load workspace'))
  }, [client])

  if (error) {
    return (
      <View style={styles.center}>
        <Text style={styles.error}>{error}</Text>
      </View>
    )
  }
  if (!workspace) {
    return (
      <View style={styles.center}>
        <ActivityIndicator color="#d6d9e0" />
      </View>
    )
  }

  return (
    <View style={styles.screen}>
      <StatusBar barStyle="light-content" backgroundColor="#07080f" />
      <View style={styles.header}>
        <Text style={styles.brand}>TANDEMESH</Text>
        <Text style={styles.title}>{workspace.name}</Text>
      </View>

      <View style={styles.body}>
        {tab === 'tasks' ? <Tasks client={client} workspace={workspace} /> : null}
        {tab === 'sessions' ? <Sessions client={client} workspace={workspace} /> : null}
        {tab === 'settings' ? (
          <ScrollView contentContainerStyle={styles.settings}>
            <Text style={styles.label}>Connected plane</Text>
            <Text style={styles.value}>{session.host}</Text>
            <Text style={styles.label}>Workspace</Text>
            <Text style={styles.value}>{workspace.name}</Text>
            <Pressable style={styles.danger} onPress={onSignOut}>
              <Text style={styles.dangerText}>Unpair this device</Text>
            </Pressable>
          </ScrollView>
        ) : null}
      </View>

      <View style={styles.tabs}>
        {([
          ['tasks', 'Tasks'],
          ['sessions', 'Sessions'],
          ['settings', 'Settings'],
        ] as [Tab, string][]).map(([value, label]) => (
          <Pressable key={value} style={styles.tab} onPress={() => setTab(value)}>
            <Text style={[styles.tabText, tab === value && styles.tabActive]}>{label}</Text>
          </Pressable>
        ))}
      </View>
    </View>
  )
}

function Tasks({ client, workspace }: { client: TandemeshClient; workspace: Workspace }) {
  const [tasks, setTasks] = useState<Task[]>([])
  const [error, setError] = useState('')
  useEffect(() => {
    client
      .tasks(workspace.id)
      .then((result) => setTasks(result.items))
      .catch((nextError) => setError(nextError instanceof Error ? nextError.message : 'Failed to load tasks'))
  }, [client, workspace.id])

  if (error) return <Text style={styles.error}>{error}</Text>
  return (
    <FlatList
      data={tasks}
      keyExtractor={(task) => task.id}
      contentContainerStyle={styles.list}
      ListEmptyComponent={<Text style={styles.empty}>No tasks.</Text>}
      renderItem={({ item }) => (
        <View style={styles.row}>
          <Text style={styles.rowTitle}>{item.title || item.id}</Text>
          <Text style={styles.rowMeta}>{item.status}</Text>
        </View>
      )}
    />
  )
}

function Sessions({ client, workspace }: { client: TandemeshClient; workspace: Workspace }) {
  const [sessions, setSessions] = useState<AgentSession[]>([])
  const [selected, setSelected] = useState<AgentSession | null>(null)

  useEffect(() => {
    client
      .sessions(workspace.id)
      .then((result) => setSessions(result.sessions))
      .catch(() => setSessions([]))
  }, [client, workspace.id])

  if (selected) {
    return <Chat client={client} session={selected} onBack={() => setSelected(null)} />
  }

  return (
    <FlatList
      data={sessions}
      keyExtractor={(session) => session.id}
      contentContainerStyle={styles.list}
      ListEmptyComponent={<Text style={styles.empty}>No sessions.</Text>}
      renderItem={({ item }) => (
        <Pressable style={styles.row} onPress={() => setSelected(item)}>
          <Text style={styles.rowTitle}>{item.title || item.id}</Text>
          <Text style={styles.rowMeta}>
            {item.backend || item.agentId} · {item.lastRunStatus || item.status}
          </Text>
        </Pressable>
      )}
    />
  )
}

function Chat({
  client,
  session,
  onBack,
}: {
  client: TandemeshClient
  session: AgentSession
  onBack: () => void
}) {
  const [messages, setMessages] = useState<SessionMessage[]>([])
  const [draft, setDraft] = useState('')
  const [sending, setSending] = useState(false)

  const load = async () => {
    try {
      const result = await client.sessionMessages(session.id)
      setMessages(result.messages ?? [])
    } catch {
      // ignore polling errors
    }
  }

  useEffect(() => {
    void load()
    const timer = setInterval(() => void load(), 4000)
    return () => clearInterval(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [client, session.id])

  const send = async () => {
    if (!draft.trim()) return
    setSending(true)
    try {
      await client.createSessionRun(session.id, draft.trim())
      setDraft('')
      await load()
    } finally {
      setSending(false)
    }
  }

  return (
    <View style={styles.chat}>
      <Pressable onPress={onBack} style={styles.back}>
        <Text style={styles.backText}>‹ Sessions</Text>
      </Pressable>
      <Text style={styles.chatTitle}>{session.title || session.id}</Text>
      <ScrollView style={styles.messages} contentContainerStyle={styles.messagesContent}>
        {messages.map((message, index) => (
          <View
            key={message.id ?? `${message.runId}-${message.ts}-${index}`}
            style={[styles.message, message.role === 'user' ? styles.userMessage : styles.agentMessage]}
          >
            <Text style={styles.role}>{message.role || message.kind}</Text>
            <Text style={styles.messageText}>{message.text}</Text>
          </View>
        ))}
      </ScrollView>
      <View style={styles.composer}>
        <TextInput
          value={draft}
          onChangeText={setDraft}
          placeholder="Message the agent…"
          placeholderTextColor="#6b7280"
          style={styles.composerInput}
        />
        <Pressable style={styles.primarySmall} disabled={sending} onPress={() => void send()}>
          <Text style={styles.primaryText}>{sending ? '…' : 'Send'}</Text>
        </Pressable>
      </View>
    </View>
  )
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#07080f', paddingTop: 32 },
  center: { flex: 1, alignItems: 'center', justifyContent: 'center', backgroundColor: '#07080f', paddingTop: 32 },
  header: { paddingHorizontal: 18, paddingVertical: 14, borderBottomWidth: 1, borderBottomColor: '#202433' },
  brand: { color: '#9aa4b2', fontSize: 11, letterSpacing: 2 },
  title: { color: '#f3f4f6', fontSize: 22, fontWeight: '600', marginTop: 5 },
  body: { flex: 1 },
  tabs: { flexDirection: 'row', borderTopWidth: 1, borderTopColor: '#202433', backgroundColor: '#0b0d14', paddingBottom: 18, paddingTop: 10 },
  tab: { flex: 1, alignItems: 'center' },
  tabText: { color: '#6e7787', fontSize: 12 },
  tabActive: { color: '#f3f4f6', fontWeight: '600' },
  card: { margin: 20, marginTop: 80, borderRadius: 14, borderWidth: 1, borderColor: '#202433', backgroundColor: '#10131c', padding: 20 },
  hint: { color: '#8b94a7', fontSize: 13, marginTop: 8, marginBottom: 14 },
  label: { color: '#8b94a7', fontSize: 10, letterSpacing: 1, marginTop: 12, marginBottom: 5 },
  input: { borderRadius: 8, borderWidth: 1, borderColor: '#262b3a', backgroundColor: '#0b0d14', color: '#f3f4f6', paddingHorizontal: 12, paddingVertical: 9, fontSize: 13 },
  error: { color: '#f87171', fontSize: 12, margin: 16 },
  primary: { marginTop: 20, borderRadius: 8, backgroundColor: '#d6d9e0', alignItems: 'center', paddingVertical: 11 },
  primaryText: { color: '#0b0d12', fontSize: 13, fontWeight: '600' },
  disabled: { opacity: 0.4 },
  list: { padding: 14, gap: 8 },
  row: { borderRadius: 9, borderWidth: 1, borderColor: '#202433', backgroundColor: '#10131c', padding: 13 },
  rowTitle: { color: '#f3f4f6', fontSize: 14 },
  rowMeta: { color: '#8b94a7', fontSize: 11, marginTop: 5 },
  empty: { color: '#8b94a7', fontSize: 12, textAlign: 'center', marginTop: 24 },
  settings: { padding: 18 },
  value: { color: '#f3f4f6', fontSize: 13 },
  danger: { marginTop: 24, borderRadius: 8, borderWidth: 1, borderColor: '#5b2330', backgroundColor: '#1b0f13', alignItems: 'center', paddingVertical: 11 },
  dangerText: { color: '#f87171', fontSize: 12 },
  chat: { flex: 1 },
  back: { paddingHorizontal: 14, paddingVertical: 10 },
  backText: { color: '#8fb7e0', fontSize: 13 },
  chatTitle: { color: '#f3f4f6', fontSize: 16, fontWeight: '600', paddingHorizontal: 14, paddingBottom: 8 },
  messages: { flex: 1 },
  messagesContent: { padding: 12, gap: 8 },
  message: { maxWidth: '92%', borderRadius: 9, paddingHorizontal: 11, paddingVertical: 8 },
  userMessage: { alignSelf: 'flex-end', backgroundColor: '#1d2331' },
  agentMessage: { alignSelf: 'flex-start', backgroundColor: '#10131c', borderWidth: 1, borderColor: '#202433' },
  role: { color: '#6e7787', fontSize: 9, letterSpacing: 1, marginBottom: 3 },
  messageText: { color: '#d7dbe4', fontSize: 12, lineHeight: 17 },
  composer: { flexDirection: 'row', gap: 8, padding: 10, borderTopWidth: 1, borderTopColor: '#202433', paddingBottom: 18 },
  composerInput: { flex: 1, borderRadius: 8, borderWidth: 1, borderColor: '#262b3a', backgroundColor: '#0b0d14', color: '#f3f4f6', paddingHorizontal: 10, paddingVertical: 9, fontSize: 13 },
  primarySmall: { borderRadius: 8, backgroundColor: '#d6d9e0', paddingHorizontal: 14, justifyContent: 'center' },
})
