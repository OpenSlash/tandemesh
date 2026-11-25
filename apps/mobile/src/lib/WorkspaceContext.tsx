import { createContext, useContext } from 'react'
import type { Workspace } from './api'

export interface WorkspaceState {
  workspace: Workspace
  workspaces: Workspace[]
  selectWorkspace: (id: string) => void
}

export const WorkspaceContext = createContext<WorkspaceState | null>(null)

export function useWorkspace(): WorkspaceState {
  const value = useContext(WorkspaceContext)
  if (!value) throw new Error('WorkspaceContext is missing')
  return value
}
