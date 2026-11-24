import { useState } from 'react'
import { Link } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { Check, Plus, Settings2 } from 'lucide-react'
import { api } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { useWorkspace } from '../lib/workspace'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Dialog, DialogContent, DialogTrigger } from '../components/ui/dialog'

export default function WorkspacePage() {
  const { t } = useI18n()
  const { workspaceId, setWorkspaceId, workspaces } = useWorkspace()
  const qc = useQueryClient()
  const [wsNameInput, setWsNameInput] = useState('')
  const [wsOpen, setWsOpen] = useState(false)

  const createWorkspace = useMutation({
    mutationFn: () => api.createWorkspace({ name: wsNameInput }),
    onSuccess: (ws) => {
      setWsOpen(false)
      setWsNameInput('')
      setWorkspaceId(ws.id)
      qc.invalidateQueries({ queryKey: ['workspaces'] })
    },
  })

  return (
    <div className="mx-auto max-w-[900px] px-6 py-5">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-[18px] font-semibold tracking-tight">{t('page.workspace')}</h1>
          <p className="mt-0.5 text-[12px] text-muted">{t('shell.workspaces')}</p>
        </div>
        <Dialog open={wsOpen} onOpenChange={setWsOpen}>
          <DialogTrigger asChild>
            <Button variant="primary" size="sm">
              <Plus className="h-3.5 w-3.5" /> {t('workspace.workspaceButton')}
            </Button>
          </DialogTrigger>
          <DialogContent title={t('workspace.newWorkspace')}>
            <form
              className="flex flex-col gap-3"
              onSubmit={(e) => {
                e.preventDefault()
                createWorkspace.mutate()
              }}
            >
              <Input
                autoFocus
                value={wsNameInput}
                onChange={(e) => setWsNameInput(e.target.value)}
                placeholder={t('workspace.wsNamePlaceholder')}
              />
              {createWorkspace.error && (
                <p className="text-[12px] text-err">{String(createWorkspace.error.message)}</p>
              )}
              <div className="flex justify-end gap-2">
                <Button type="button" variant="ghost" onClick={() => setWsOpen(false)}>
                  {t('common.cancel')}
                </Button>
                <Button type="submit" variant="primary" disabled={!wsNameInput || createWorkspace.isPending}>
                  {t('common.create')}
                </Button>
              </div>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <div className="mt-4 overflow-hidden rounded-lg border border-line">
        {workspaces.map((w) => (
          <div
            key={w.id}
            className="flex items-center justify-between border-b border-line/60 px-3 py-2 last:border-0"
          >
            <button
              type="button"
              onClick={() => setWorkspaceId(w.id)}
              className="flex min-w-0 items-center gap-2 text-left"
            >
              <span className="flex h-5 w-5 items-center justify-center rounded-full bg-panel2 text-[10px] uppercase text-muted">
                {w.name.slice(0, 1)}
              </span>
              <span className="truncate text-[12.5px]">{w.name}</span>
              {w.id === workspaceId && <Check className="h-3.5 w-3.5 text-accent2" />}
            </button>
            <Link
              to={`/workspaces/${w.id}/settings`}
              className="inline-flex items-center gap-1 text-[11.5px] text-muted hover:text-fg"
            >
              <Settings2 className="h-3.5 w-3.5" />
              {t('workspaceSettings.menu')}
            </Link>
          </div>
        ))}
        {workspaces.length === 0 && (
          <p className="px-3 py-6 text-[12px] text-muted">{t('common.loading')}</p>
        )}
      </div>
    </div>
  )
}
