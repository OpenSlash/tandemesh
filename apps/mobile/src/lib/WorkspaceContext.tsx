import { createContext, useContext } from 'react'
import type { Workspace } from './api'

export const WorkspaceContext = createContext<Workspace | null>(null)

export function useWorkspace(): Workspace {
  const workspace = useContext(WorkspaceContext)
  if (!workspace) throw new Error('WorkspaceContext is missing')
  return workspace
}
