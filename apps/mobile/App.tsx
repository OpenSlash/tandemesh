import { Platform, View, StyleSheet } from 'react-native'
import HarmonyApp from './src/HarmonyApp'

export default function App() {
  if ((Platform.OS as string) === 'harmony') {
    return <HarmonyApp />
  }
  // Load the native-stack app lazily so HarmonyOS never initializes
  // react-native-screens or safe-area-context native modules.
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  const NativeApp = require('./src/NativeApp').default
  return (
    <View style={styles.root}>
      <NativeApp />
    </View>
  )
}

const styles = StyleSheet.create({
  root: { flex: 1, backgroundColor: '#07080f' },
})
