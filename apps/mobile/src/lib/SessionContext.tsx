import { createContext, useContext } from 'react'

export interface SessionValue {
  host: string
  token: string
  onSignOut: () => void
}

export const SessionContext = createContext<SessionValue | null>(null)

export function useSession(): SessionValue {
  const value = useContext(SessionContext)
  if (!value) throw new Error('SessionContext is missing')
  return value
}
