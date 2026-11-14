import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import { useQuery } from '@tanstack/react-query'
import { api, type Workspace } from './api'

export type WorkspaceMode = 'select' | 'list'

interface WorkspaceContextValue {
  workspaceId: string
  setWorkspaceId: (id: string) => void
  workspaces: Workspace[]
  workspaceMode: WorkspaceMode
  setWorkspaceMode: (mode: WorkspaceMode) => void
}

const WorkspaceContext = createContext<WorkspaceContextValue>({
  workspaceId: 'ws_default',
  setWorkspaceId: () => {},
  workspaces: [],
  workspaceMode: 'select',
  setWorkspaceMode: () => {},
})

export function WorkspaceProvider({ children }: { children: ReactNode }) {
  const { data: ws } = useQuery({ queryKey: ['workspaces'], queryFn: () => api.workspaces() })
  const workspaces = useMemo(() => ws?.items ?? [], [ws])

  const [workspaceMode, setWorkspaceMode] = useState<WorkspaceMode>(
    () => (localStorage.getItem('orca.workspaceMode') as WorkspaceMode) || 'select',
  )
  useEffect(() => {
    localStorage.setItem('orca.workspaceMode', workspaceMode)
  }, [workspaceMode])

  const [workspaceId, setWorkspaceId] = useState(
    () => localStorage.getItem('orca.workspace') || 'ws_default',
  )
  useEffect(() => {
    if (workspaces.length > 0 && !workspaces.some((w) => w.id === workspaceId)) {
      setWorkspaceId(workspaces[0].id)
    }
  }, [workspaces, workspaceId])
  useEffect(() => {
    localStorage.setItem('orca.workspace', workspaceId)
  }, [workspaceId])

  return (
    <WorkspaceContext.Provider
      value={{
        workspaceId,
        setWorkspaceId,
        workspaces,
        workspaceMode,
        setWorkspaceMode,
      }}
    >
      {children}
    </WorkspaceContext.Provider>
  )
}

export function useWorkspace() {
  return useContext(WorkspaceContext)
}
