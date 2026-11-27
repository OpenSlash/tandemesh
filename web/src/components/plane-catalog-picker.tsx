import { useEffect, useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { api } from '../lib/api'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { useI18n } from '../lib/i18n'

type Item = { id: string; name: string; slug?: string }

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
  const [workspace, setWorkspace] = useState('')
  const [projects, setProjects] = useState<Item[]>([])
  const [message, setMessage] = useState('')

  const loadProjects = useMutation({
    mutationFn: (slug: string) => api.catalog('plane', { token, baseUrl, workspace: slug, connectionId }),
    onSuccess: (data) => {
      setProjects(data.projects ?? [])
      setMessage('')
    },
    onError: (err) => setMessage(String(err)),
  })

  useEffect(() => {
    if (!initialWorkspace) return
    setWorkspace(initialWorkspace)
    loadProjects.mutate(initialWorkspace)
  }, [initialWorkspace])

  return (
    <div className="rounded-md border border-line2 p-2">
      <span className="text-[11px] uppercase tracking-wider text-muted">{t('plane.workspaceProject')}</span>
      <div className="mt-2 flex gap-2">
        <Input
          value={workspace}
          onChange={(e) => setWorkspace(e.target.value)}
          placeholder={t('plane.workspaceSlug')}
        />
        <Button
          type="button"
          size="sm"
          variant="secondary"
          disabled={(!token && !connectionId) || !workspace || loadProjects.isPending}
          onClick={() => loadProjects.mutate(workspace)}
        >
          {t('plane.loadProjects')}
        </Button>
      </div>
      {projects.length > 0 && (
        <select
          className="mt-2 h-8 w-full rounded-md border border-line2 bg-panel px-2 text-[12px]"
          value=""
          onChange={(e) => {
            const project = projects.find((item) => item.id === e.target.value)
            if (project) onSelect(`${workspace}/${project.id}`, project.name)
          }}
        >
          <option value="">{t('plane.selectProject')}</option>
          {projects.map((item) => (
            <option key={item.id} value={item.id}>{item.name}</option>
          ))}
        </select>
      )}
      {message && <p className="mt-1 text-[10.5px] text-err">{message}</p>}
    </div>
  )
}
