import { createContext, useContext } from 'react'

export type RouteName = 'TaskDetail' | 'RunDetail' | 'Diff' | 'SessionDetail'

export interface RouterValue {
  params: Record<string, string> | undefined
  navigate: (name: RouteName, params?: Record<string, string>) => void
  back: () => void
}

export const RouterContext = createContext<RouterValue | null>(null)

export function useRouter(): RouterValue {
  const value = useContext(RouterContext)
  if (!value) throw new Error('RouterContext is missing')
  return value
}
