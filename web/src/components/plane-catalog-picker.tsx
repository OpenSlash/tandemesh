import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { api } from '../lib/api'
import { Button } from './ui/button'
import { Input } from './ui/input'

type Item = { id: string; name: string; slug?: string }

export function PlaneCatalogPicker({
  token,
  baseUrl,
  onSelect,
}: {
  token: string
  baseUrl: string
  onSelect: (instance: string) => void
}) {
  const [workspace, setWorkspace] = useState('')
  const [projects, setProjects] = useState<Item[]>([])
  const [message, setMessage] = useState('')

  const loadProjects = useMutation({
    mutationFn: () => api.catalog('plane', { token, baseUrl, workspace }),
    onSuccess: (data) => {
      setProjects(data.projects ?? [])
      setMessage('')
    },
    onError: (err) => setMessage(String(err)),
  })

  return (
    <div className="rounded-md border border-line2 p-2">
      <span className="text-[11px] uppercase tracking-wider text-muted">Plane workspace / project</span>
      <div className="mt-2 flex gap-2">
        <Input
          value={workspace}
          onChange={(e) => setWorkspace(e.target.value)}
          placeholder="workspace slug"
        />
        <Button
          type="button"
          size="sm"
          variant="secondary"
          disabled={!token || !workspace || loadProjects.isPending}
          onClick={() => loadProjects.mutate()}
        >
          Load projects
        </Button>
      </div>
      {projects.length > 0 && (
        <select
          className="mt-2 h-8 w-full rounded-md border border-line2 bg-panel px-2 text-[12px]"
          value=""
          onChange={(e) => {
            const project = projects.find((item) => item.id === e.target.value)
            if (project) onSelect(`${workspace}/${project.id}`)
          }}
        >
          <option value="">Select project</option>
          {projects.map((item) => (
            <option key={item.id} value={item.id}>{item.name}</option>
          ))}
        </select>
      )}
      {message && <p className="mt-1 text-[10.5px] text-err">{message}</p>}
    </div>
  )
}
