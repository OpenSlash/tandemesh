import { createContext, useContext, useEffect, useState, type ReactNode } from 'react'

export type Theme = 'dark' | 'light' | 'system'

interface ThemeValue {
  theme: Theme
  setTheme: (theme: Theme) => void
  resolved: 'dark' | 'light'
}

const ThemeContext = createContext<ThemeValue>({ theme: 'dark', setTheme: () => {}, resolved: 'dark' })

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setTheme] = useState<Theme>(() => {
    const saved = localStorage.getItem('orca.theme')
    if (saved === 'dark' || saved === 'light' || saved === 'system') return saved
    return 'dark'
  })
  const [systemDark, setSystemDark] = useState(() =>
    typeof window !== 'undefined' ? window.matchMedia('(prefers-color-scheme: dark)').matches : true,
  )

  useEffect(() => {
    const query = window.matchMedia('(prefers-color-scheme: dark)')
    const listener = (event: MediaQueryListEvent) => setSystemDark(event.matches)
    query.addEventListener('change', listener)
    return () => query.removeEventListener('change', listener)
  }, [])

  const resolved = theme === 'system' ? (systemDark ? 'dark' : 'light') : theme

  useEffect(() => {
    document.documentElement.dataset.theme = resolved
    localStorage.setItem('orca.theme', theme)
  }, [theme, resolved])

  return <ThemeContext.Provider value={{ theme, setTheme, resolved }}>{children}</ThemeContext.Provider>
}

export function useTheme() {
  return useContext(ThemeContext)
}
