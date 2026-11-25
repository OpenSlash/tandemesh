import { Pressable, SafeAreaView, StyleSheet, Text, View } from 'react-native'
import { useSession } from '../lib/SessionContext'

export function SettingsScreen() {
  const { host, onSignOut } = useSession()
  return (
    <SafeAreaView style={styles.screen}>
      <View style={styles.header}>
        <Text style={styles.brand}>Tandemesh</Text>
        <Text style={styles.title}>Settings</Text>
      </View>
      <View style={styles.card}>
        <Text style={styles.label}>Connected plane</Text>
        <Text style={styles.value}>{host}</Text>
      </View>
      <Pressable style={styles.button} onPress={onSignOut}>
        <Text style={styles.buttonText}>Unpair this device</Text>
      </Pressable>
    </SafeAreaView>
  )
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#07080f' },
  header: { padding: 20, borderBottomWidth: 1, borderBottomColor: '#202433' },
  brand: { color: '#9aa4b2', fontSize: 11, letterSpacing: 2, textTransform: 'uppercase' },
  title: { color: '#f3f4f6', fontSize: 22, fontWeight: '600', marginTop: 4 },
  card: { margin: 16, borderRadius: 10, borderWidth: 1, borderColor: '#202433', backgroundColor: '#10131c', padding: 14 },
  label: { color: '#8b94a7', fontSize: 11, textTransform: 'uppercase', letterSpacing: 1 },
  value: { color: '#f3f4f6', fontSize: 13, marginTop: 6 },
  button: { margin: 16, borderRadius: 8, borderWidth: 1, borderColor: '#5b2330', backgroundColor: '#1b0f13', alignItems: 'center', paddingVertical: 12 },
  buttonText: { color: '#f87171', fontSize: 13, fontWeight: '600' },
})
