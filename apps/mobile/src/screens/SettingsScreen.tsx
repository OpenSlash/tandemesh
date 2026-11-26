import { Alert, Platform, Pressable,  ScrollView, StyleSheet, Text, View } from 'react-native'
import { Screen } from '../components/Screen'
import { useSession } from '../lib/SessionContext'
import { useWorkspace } from '../lib/WorkspaceContext'

const APP_VERSION = '0.1.0'

export function SettingsScreen() {
  const { host, onSignOut } = useSession()
  const { workspace, workspaces } = useWorkspace()

  const confirmSignOut = () => {
    Alert.alert('Unpair device?', 'The bearer token will be deleted from this device.', [
      { text: 'Cancel', style: 'cancel' },
      { text: 'Unpair', style: 'destructive', onPress: onSignOut },
    ])
  }

  return (
    <Screen>
      <ScrollView contentContainerStyle={styles.content}>
        <View style={styles.header}>
          <Text style={styles.brand}>Tandemesh</Text>
          <Text style={styles.title}>Settings</Text>
        </View>

        <View style={styles.card}>
          <Text style={styles.label}>Connected plane</Text>
          <Text style={styles.value}>{host}</Text>
          <Text style={styles.label}>Workspace</Text>
          <Text style={styles.value}>{workspace.name}</Text>
          <Text style={styles.label}>Workspaces</Text>
          <Text style={styles.value}>{workspaces.map((item) => item.name).join(', ')}</Text>
        </View>

        <View style={styles.card}>
          <Text style={styles.label}>App version</Text>
          <Text style={styles.value}>{APP_VERSION}</Text>
          <Text style={styles.label}>Platform</Text>
          <Text style={styles.value}>{Platform.OS}</Text>
        </View>

        <Pressable style={styles.button} onPress={confirmSignOut}>
          <Text style={styles.buttonText}>Unpair this device</Text>
        </Pressable>
      </ScrollView>
    </Screen>
  )
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#07080f' },
  content: { paddingBottom: 130 },
  header: { paddingHorizontal: 20, paddingVertical: 12, borderBottomWidth: 1, borderBottomColor: '#202433' },
  brand: { color: '#9aa4b2', fontSize: 10, letterSpacing: 2, textTransform: 'uppercase' },
  title: { color: '#f3f4f6', fontSize: 20, fontWeight: '600', marginTop: 2 },
  card: { marginHorizontal: 16, marginTop: 16, borderRadius: 10, borderWidth: 1, borderColor: '#202433', backgroundColor: '#10131c', padding: 14, gap: 4 },
  label: { color: '#8b94a7', fontSize: 10, textTransform: 'uppercase', letterSpacing: 1, marginTop: 6 },
  value: { color: '#f3f4f6', fontSize: 13 },
  button: { marginHorizontal: 16, marginTop: 20, borderRadius: 8, borderWidth: 1, borderColor: '#5b2330', backgroundColor: '#1b0f13', alignItems: 'center', paddingVertical: 12 },
  buttonText: { color: '#f87171', fontSize: 13, fontWeight: '600' },
})
