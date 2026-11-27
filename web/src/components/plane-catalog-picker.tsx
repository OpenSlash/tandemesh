import { useEffect, useMemo, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Loader2 } from 'lucide-react'
import { api } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { useWorkspace } from '../lib/workspace'
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

/** True when one Plane scope contains the other ("acme" covers "acme/p1"). */
function overlaps(a: string, b: string) {
  return a === b || a.startsWith(b + '/') || b.startsWith(a + '/')
}

export interface PlaneSelection {
  /** "workspace" for the whole workspace, or "workspace/<project id>" each. */
  instances: string[]
  names: Record<string, string>
}

/**
 * Picks a Plane workspace and the projects to bind to the current Tandemesh
 * workspace. Nothing is selected by default. A project bound to another
 * Tandemesh workspace cannot be picked: one project belongs to one workspace.
 */
export function PlaneCatalogPicker({
  token,
  baseUrl,
  connectionId,
  initialWorkspace,
  workspaceId,
  onChange,
}: {
  token: string
  baseUrl: string
  connectionId?: string
  initialWorkspace?: string
  /** The Tandemesh workspace the projects will be bound to. */
  workspaceId: string
  onChange: (selection: PlaneSelection) => void
}) {
  const { t } = useI18n()
  const { workspaces: tandemeshWorkspaces } = useWorkspace()
  const [workspaceInput, setWorkspaceInput] = useState(initialWorkspace ?? '')
  const [selected, setSelected] = useState<string[]>([])
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
  const { data: allConnections } = useQuery({ queryKey: ['connections', 'all'], queryFn: () => api.connections('all') })
  const bound = useMemo(
    () =>
      (allConnections?.items ?? [])
        .filter((c) => c.provider === 'plane')
        .flatMap((c) => c.allBindings ?? [])
        .filter((b) => b.enabled),
    [allConnections],
  )
  const projectList = projects.data?.projects ?? []
  const workspaceList = workspaces.data?.workspaces ?? []
  const workspaceName = (id: string) => tandemeshWorkspaces.find((w) => w.id === id)?.name ?? id

  /** Where an instance is already bound, if anywhere. */
  const boundTo = (instance: string) => {
    const here = bound.find((b) => b.workspaceId === workspaceId && b.externalId === instance)
    if (here) return { here: true, name: workspaceName(workspaceId) }
    const elsewhere = bound.find((b) => b.workspaceId !== workspaceId && overlaps(b.externalId, instance))
    return elsewhere ? { here: false, name: workspaceName(elsewhere.workspaceId) } : null
  }

  useEffect(() => {
    setSelected([])
  }, [workspace])
  useEffect(() => {
    const names: Record<string, string> = {}
    for (const item of projectList) names[`${workspace}/${item.id}`] = item.name
    names[workspace] = workspace
    onChange({ instances: selected, names })
    // onChange is a fresh closure on every parent render.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selected, projects.data])

  const toggle = (instance: string) =>
    setSelected((prev) => (prev.includes(instance) ? prev.filter((x) => x !== instance) : [...prev, instance]))
  const wholeWorkspace = selected.includes(workspace)

  const row = (instance: string, label: string, hint?: string, forceDisabled = false) => {
    const status = boundTo(instance)
    const disabled = forceDisabled || Boolean(status)
    return (
      <label key={instance} className={`flex items-start gap-2 rounded px-1.5 py-1 text-[12px] ${disabled ? 'opacity-50' : 'hover:bg-panel2/60'}`}>
        <input
          type="checkbox"
          className="mt-0.5"
          checked={status?.here || selected.includes(instance) || (forceDisabled && wholeWorkspace)}
          disabled={disabled}
          onChange={() => toggle(instance)}
        />
        <span className="min-w-0 flex-1">
          <span className="text-fg">{label}</span>
          {hint && <span className="block text-[10.5px] text-muted/70">{hint}</span>}
        </span>
        {status && (
          <span className="shrink-0 text-[10.5px] text-muted">
            {status.here ? t('plane.boundHere') : t('plane.boundTo', { workspace: status.name })}
          </span>
        )}
      </label>
    )
  }

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
          <Input className="mt-1" value={workspaceInput} onChange={(e) => setWorkspaceInput(e.target.value)} placeholder="my-workspace" />
        )}
        <span className="mt-1 block text-[10.5px] text-muted/70">{t('plane.workspaceHint')}</span>
      </label>
      {workspace && (
        <div className="text-[12px] text-muted">
          <span className="flex items-center gap-1.5">
            {t('plane.projects')} *
            {projects.isFetching && <Loader2 className="h-3 w-3 animate-spin" />}
          </span>
          {!canQuery && <span className="mt-1 block text-[10.5px] text-muted/70">{t('plane.needCredentials')}</span>}
          {projects.error && <span className="mt-1 block text-[10.5px] text-err">{t('plane.projectsFailed', { error: projects.error.message })}</span>}
          {projectList.length > 0 && (
            <div className="mt-1 max-h-56 overflow-y-auto rounded-md border border-line2 p-1">
              {row(workspace, t('plane.wholeWorkspace'), t('plane.wholeWorkspaceHint'))}
              <div className="my-1 border-t border-line2" />
              {projectList.map((item) => row(`${workspace}/${item.id}`, item.name, undefined, wholeWorkspace))}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
