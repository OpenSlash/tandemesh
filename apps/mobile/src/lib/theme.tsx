import { createContext, useContext, useEffect, useMemo, useState, type PropsWithChildren } from 'react'
import { useColorScheme } from 'react-native'
import { loadPreference, savePreference } from './preferences'

export type ThemeMode = 'system' | 'light' | 'dark'

export interface ThemeColors {
  background: string
  surface: string
  surfaceAlt: string
  border: string
  borderStrong: string
  text: string
  textSecondary: string
  textMuted: string
  textSoft: string
  accent: string
  accentText: string
  userBubble: string
  danger: string
  dangerBg: string
  dangerBorder: string
  codeBg: string
  codeBorder: string
  tabBar: string
  tabBarBorder: string
  tabActive: string
  tabInactive: string
}

const dark: ThemeColors = {
  background: '#07080f',
  surface: '#10131c',
  surfaceAlt: '#0b0d14',
  border: '#202433',
  borderStrong: '#262b3a',
  text: '#f3f4f6',
  textSecondary: '#d7dbe4',
  textMuted: '#8b94a7',
  textSoft: '#6e7787',
  accent: '#d6d9e0',
  accentText: '#0b0d12',
  userBubble: '#1d2331',
  danger: '#f87171',
  dangerBg: '#1b0f13',
  dangerBorder: '#5b2330',
  codeBg: '#0b0f16',
  codeBorder: '#1a202b',
  tabBar: 'rgba(15,18,27,0.96)',
  tabBarBorder: '#262c3b',
  tabActive: '#f3f4f6',
  tabInactive: '#6e7787',
}

const light: ThemeColors = {
  background: '#f4f5f7',
  surface: '#ffffff',
  surfaceAlt: '#eef0f4',
  border: '#dfe3ea',
  borderStrong: '#cbd2df',
  text: '#111827',
  textSecondary: '#374151',
  textMuted: '#6b7280',
  textSoft: '#9ca3af',
  accent: '#111827',
  accentText: '#ffffff',
  userBubble: '#e5e9f0',
  danger: '#dc2626',
  dangerBg: '#fee2e2',
  dangerBorder: '#fecaca',
  codeBg: '#f3f4f6',
  codeBorder: '#e5e7eb',
  tabBar: 'rgba(255,255,255,0.96)',
  tabBarBorder: '#dfe3ea',
  tabActive: '#111827',
  tabInactive: '#9ca3af',
}

interface ThemeValue {
  mode: ThemeMode
  setMode: (mode: ThemeMode) => void
  isDark: boolean
  colors: ThemeColors
}

const ThemeContext = createContext<ThemeValue | null>(null)

export function ThemeProvider({ children }: PropsWithChildren) {
  const scheme = useColorScheme()
  const [mode, setModeState] = useState<ThemeMode>('system')

  useEffect(() => {
    void loadPreference('theme').then((value) => {
      if (value === 'system' || value === 'light' || value === 'dark') setModeState(value)
    })
  }, [])

  const setMode = (next: ThemeMode) => {
    setModeState(next)
    void savePreference('theme', next)
  }
  const isDark = mode === 'system' ? scheme !== 'light' : mode === 'dark'
  const colors = isDark ? dark : light
  const value = useMemo(() => ({ mode, setMode, isDark, colors }), [mode, isDark, colors])
  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>
}

export function useTheme(): ThemeValue {
  const value = useContext(ThemeContext)
  if (!value) throw new Error('ThemeProvider is missing')
  return value
}
