import { StatusBar } from 'react-native'
import { useTheme } from '../lib/theme'

export function ThemedStatusBar() {
  const { isDark, colors } = useTheme()
  return (
    <StatusBar
      barStyle={isDark ? 'light-content' : 'dark-content'}
      backgroundColor={colors.background}
    />
  )
}
