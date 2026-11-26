import { useMemo, useState } from 'react'
import {
  ActivityIndicator,
  Pressable,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native'
import { Screen } from '../components/Screen'
import { useTheme, type ThemeColors } from '../lib/theme'
import { Platform } from 'react-native'
import { exchangePairing } from '../lib/api'

interface Props {
  initialHost?: string
  initialCode?: string
  onPaired: (host: string, token: string) => void
}

export function PairScreen({ initialHost, initialCode, onPaired }: Props) {
  const defaultHost = 'http://127.0.0.1:8080'
  const [host, setHost] = useState(initialHost ?? defaultHost)
  const [code, setCode] = useState(initialCode ?? '')
  const [deviceName, setDeviceName] = useState('Tandemesh Mobile')
  const [busy, setBusy] = useState(false)
  const { colors } = useTheme()
  const styles = useMemo(() => makeStyles(colors), [colors])

  const [error, setError] = useState('')

  const pair = async () => {
    setBusy(true)
    setError('')
    try {
      const result = await exchangePairing(host, code, deviceName, Platform.OS)
      onPaired(host, result.token)
    } catch (nextError) {
      setError(nextError instanceof Error ? nextError.message : 'Pairing failed')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Screen><View style={styles.content}>
      <View style={styles.card}>
        <Text style={styles.brand}>Tandemesh</Text>
        <Text style={styles.title}>Pair this device</Text>
        <Text style={styles.hint}>
          On the desktop open Settings → Mobile devices, generate a code, then enter it below.
        </Text>

        <Text style={styles.label}>Host</Text>
        <TextInput
          value={host}
          onChangeText={setHost}
          autoCapitalize="none"
          autoCorrect={false}
          placeholder="http://192.168.1.10:8080"
          placeholderTextColor="#6b7280"
          style={styles.input}
        />

        <Text style={styles.label}>Pairing code</Text>
        <TextInput
          value={code}
          onChangeText={setCode}
          autoCapitalize="characters"
          autoCorrect={false}
          placeholder="ABCD-EFGH"
          placeholderTextColor="#6b7280"
          style={styles.input}
        />

        <Text style={styles.label}>Device name</Text>
        <TextInput
          value={deviceName}
          onChangeText={setDeviceName}
          style={styles.input}
        />

        {error ? <Text style={styles.error}>{error}</Text> : null}

        <Pressable
          onPress={pair}
          disabled={busy || !host.trim() || !code.trim()}
          style={[styles.button, (busy || !host.trim() || !code.trim()) && styles.buttonDisabled]}
        >
          {busy ? <ActivityIndicator color="#0b0d12" /> : <Text style={styles.buttonText}>Pair</Text>}
        </Pressable>
      </View>
    </View></Screen>
  )
}

const makeStyles = (c: ThemeColors) => StyleSheet.create({
  content: { flex: 1, justifyContent: 'center', padding: 20 },
  card: { borderRadius: 16, borderWidth: 1, borderColor: c.border, backgroundColor: c.surface, padding: 20 },
  brand: { color: c.textMuted, fontSize: 13, letterSpacing: 2, textTransform: 'uppercase' },
  title: { color: c.text, fontSize: 26, fontWeight: '600', marginTop: 8 },
  hint: { color: c.textMuted, fontSize: 13, lineHeight: 19, marginTop: 8, marginBottom: 16 },
  label: { color: c.textMuted, fontSize: 11, textTransform: 'uppercase', letterSpacing: 1, marginTop: 10, marginBottom: 5 },
  input: {
    borderRadius: 8,
    borderWidth: 1,
    borderColor: c.borderStrong,
    backgroundColor: c.surfaceAlt,
    color: c.text,
    paddingHorizontal: 12,
    paddingVertical: 10,
    fontSize: 14,
  },
  error: { color: c.danger, fontSize: 12, marginTop: 12 },
  button: { marginTop: 20, alignItems: 'center', borderRadius: 8, backgroundColor: c.accent, paddingVertical: 12 },
  buttonDisabled: { opacity: 0.4 },
  buttonText: { color: c.accentText, fontSize: 14, fontWeight: '600' },
})
