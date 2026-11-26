import type { PropsWithChildren } from 'react'
import { Platform, View, StyleSheet } from 'react-native'
import HarmonyApp from './src/HarmonyApp'
import { I18nProvider } from './src/lib/i18n'
import { ThemedStatusBar } from './src/components/ThemedStatusBar'
import { ThemeProvider, useTheme } from './src/lib/theme'

function ThemedRoot({ children }: PropsWithChildren) {
  const { colors } = useTheme()
  return (
    <View style={[styles.root, { backgroundColor: colors.background }]}>
      <ThemedStatusBar />
      {children}
    </View>
  )
}

export default function App() {
  if ((Platform.OS as string) === 'harmony') {
    return (
      <ThemeProvider>
        <I18nProvider>
          <HarmonyApp />
        </I18nProvider>
      </ThemeProvider>
    )
  }
  // Load the native-stack app lazily so HarmonyOS never initializes
  // react-native-screens or safe-area-context native modules.
  // eslint-disable-next-line @typescript-eslint/no-var-requires
  const NativeApp = require('./src/NativeApp').default
  return (
    <ThemeProvider>
      <I18nProvider>
        <ThemedRoot>
          <NativeApp />
        </ThemedRoot>
      </I18nProvider>
    </ThemeProvider>
  )
}

const styles = StyleSheet.create({
  root: { flex: 1 },
})
