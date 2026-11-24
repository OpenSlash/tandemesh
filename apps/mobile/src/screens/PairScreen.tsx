import { useState } from 'react'
import {
  ActivityIndicator,
  Pressable,
  SafeAreaView,
  StyleSheet,
  Text,
  TextInput,
  View,
} from 'react-native'
import { Platform } from 'react-native'
import { exchangePairing } from '../lib/api'

interface Props {
  initialHost?: string
  initialCode?: string
  onPaired: (host: string, token: string) => void
}

export function PairScreen({ initialHost, initialCode, onPaired }: Props) {
  const [host, setHost] = useState(initialHost ?? `http://10.0.2.2:8080`)
  const [code, setCode] = useState(initialCode ?? '')
  const [deviceName, setDeviceName] = useState('Tandemesh Mobile')
  const [busy, setBusy] = useState(false)
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
    <SafeAreaView style={styles.screen}>
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
    </SafeAreaView>
  )
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#07080f', justifyContent: 'center', padding: 20 },
  card: { borderRadius: 16, borderWidth: 1, borderColor: '#202433', backgroundColor: '#10131c', padding: 20 },
  brand: { color: '#9aa4b2', fontSize: 13, letterSpacing: 2, textTransform: 'uppercase' },
  title: { color: '#f3f4f6', fontSize: 26, fontWeight: '600', marginTop: 8 },
  hint: { color: '#8b94a7', fontSize: 13, lineHeight: 19, marginTop: 8, marginBottom: 16 },
  label: { color: '#8b94a7', fontSize: 11, textTransform: 'uppercase', letterSpacing: 1, marginTop: 10, marginBottom: 5 },
  input: {
    borderRadius: 8,
    borderWidth: 1,
    borderColor: '#262b3a',
    backgroundColor: '#0b0d14',
    color: '#f3f4f6',
    paddingHorizontal: 12,
    paddingVertical: 10,
    fontSize: 14,
  },
  error: { color: '#f87171', fontSize: 12, marginTop: 12 },
  button: { marginTop: 20, alignItems: 'center', borderRadius: 8, backgroundColor: '#d6d9e0', paddingVertical: 12 },
  buttonDisabled: { opacity: 0.4 },
  buttonText: { color: '#0b0d12', fontSize: 14, fontWeight: '600' },
})
