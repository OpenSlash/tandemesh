import type { PropsWithChildren } from 'react'
import { Platform, StyleSheet, View } from 'react-native'

const SafeAreaView =
  (Platform.OS as string) === 'harmony'
    ? View
    : // Loaded lazily so HarmonyOS never initializes the native module.
      // eslint-disable-next-line @typescript-eslint/no-var-requires
      (require('react-native-safe-area-context').SafeAreaView as typeof View)

export function Screen({ children }: PropsWithChildren) {
  return <SafeAreaView style={styles.screen}>{children}</SafeAreaView>
}

const styles = StyleSheet.create({
  screen: { flex: 1, backgroundColor: '#07080f', paddingTop: (Platform.OS as string) === 'harmony' ? 32 : 0 },
})
