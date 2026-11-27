import { useEffect, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { api } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { Input } from './ui/input'

function useDebounced<T>(value: T, ms = 400) {
  const [debounced, setDebounced] = useState(value)
  useEffect(() => {
    const timer = setTimeout(() => setDebounced(value), ms)
    return () => clearTimeout(timer)
  }, [value, ms])
  return debounced
}

/** Accepts a workspace slug or any Plane URL inside the workspace. */
export function planeWorkspaceSlug(input: string) {
  const value = input.trim()
  if (!value.includes('://')) return value.replace(/^\/+|\/+$/g, '')
  try {
    return new URL(value).pathname.split('/').filter(Boolean)[0] ?? ''
  } catch {
    return value
  }
}

/**
 * Picks a Plane workspace and, optionally, one project. The instance it
 * reports is "workspace" (every project, including ones created later) or
 * "workspace/<project id>".
 */
export function PlaneCatalogPicker({
  token,
  baseUrl,
  connectionId,
  initialWorkspace,
  onSelect,
}: {
  token: string
  baseUrl: string
  connectionId?: string
  initialWorkspace?: string
  onSelect: (instance: string, name?: string) => void
}) {
  const { t } = useI18n()
  const [workspaceInput, setWorkspaceInput] = useState(initialWorkspace ?? '')
  const [projectId, setProjectId] = useState('')
  const creds = { token: useDebounced(token), baseUrl: useDebounced(baseUrl), connectionId }
  const canQuery = Boolean((creds.token && creds.baseUrl) || connectionId)
  const workspace = planeWorkspaceSlug(useDebounced(workspaceInput))

  useEffect(() => {
    if (initialWorkspace) setWorkspaceInput(initialWorkspace)
  }, [initialWorkspace])

  // Plane's public API usually cannot list workspaces; when it can, offer a
  // picker instead of a text field.
  const workspaces = useQuery({
    queryKey: ['plane-catalog', creds.baseUrl, connectionId, creds.token ? 'token' : '', 'workspaces'],
    queryFn: () => api.catalog('plane', creds),
    enabled: canQuery,
    retry: false,
    staleTime: 5 * 60 * 1000,
  })
  const projects = useQuery({
    queryKey: ['plane-catalog', creds.baseUrl, connectionId, creds.token ? 'token' : '', workspace],
    queryFn: () => api.catalog('plane', { ...creds, workspace }),
    enabled: canQuery && Boolean(workspace),
    retry: false,
    staleTime: 60 * 1000,
  })
  const projectList = projects.data?.projects ?? []
  const workspaceList = workspaces.data?.workspaces ?? []

  useEffect(() => {
    setProjectId('')
  }, [workspace])
  useEffect(() => {
    if (!workspace) return
    const project = projectList.find((item) => item.id === projectId)
    onSelect(project ? `${workspace}/${project.id}` : workspace, project?.name)
    // onSelect is a fresh closure on every parent render; reporting on the
    // values that matter is enough.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [workspace, projectId, projects.data])

  return (
    <div className="flex flex-col gap-3">
      <label className="text-[12px] text-muted">
        {t('plane.workspace')} *
        {workspaceList.length > 0 ? (
          <select
            className="mt-1 h-8 w-full rounded-md border border-line2 bg-panel px-2 text-[12px]"
            value={workspace}
            onChange={(e) => setWorkspaceInput(e.target.value)}
          >
            <option value="">{t('plane.selectWorkspace')}</option>
            {workspaceList.map((item) => (
              <option key={item.id} value={item.slug ?? item.id}>{item.name}</option>
            ))}
          </select>
        ) : (
          <Input
            className="mt-1"
            value={workspaceInput}
            onChange={(e) => setWorkspaceInput(e.target.value)}
            placeholder="my-workspace"
          />
        )}
        <span className="mt-1 block text-[10.5px] text-muted/70">{t('plane.workspaceHint')}</span>
      </label>
      {workspace && (
        <label className="text-[12px] text-muted">
          <span className="flex items-center gap-1.5">
            {t('plane.project')}
            {projects.isFetching && <Loader2 className="h-3 w-3 animate-spin" />}
          </span>
          <select
            className="mt-1 h-8 w-full rounded-md border border-line2 bg-panel px-2 text-[12px]"
            value={projectId}
            onChange={(e) => setProjectId(e.target.value)}
            disabled={projects.isFetching}
          >
            <option value="">{t('plane.allProjects', { count: projectList.length })}</option>
            {projectList.map((item) => (
              <option key={item.id} value={item.id}>{item.name}</option>
            ))}
          </select>
          {projects.error && (
            <span className="mt-1 block text-[10.5px] text-err">{t('plane.projectsFailed', { error: projects.error.message })}</span>
          )}
          {!canQuery && <span className="mt-1 block text-[10.5px] text-muted/70">{t('plane.needCredentials')}</span>}
        </label>
      )}
    </div>
  )
}
