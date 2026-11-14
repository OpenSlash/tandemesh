import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { ArrowLeft, Settings2 } from 'lucide-react'
import { api } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { useWorkspace } from '../lib/workspace'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'

export default function WorkspaceSettingsPage() {
  const { t } = useI18n()
  const { id = '' } = useParams()
  const qc = useQueryClient()
  const { workspaceId, setWorkspaceId, workspaces } = useWorkspace()
  const workspace = workspaces.find((w) => w.id === id)
  const [name, setName] = useState(workspace?.name ?? '')

  useEffect(() => {
    if (workspace) setName(workspace.name)
  }, [workspace])

  useEffect(() => {
    if (id && id !== workspaceId) setWorkspaceId(id)
  }, [id, workspaceId, setWorkspaceId])

  const rename = useMutation({
    mutationFn: () => api.updateWorkspace(id, name.trim()),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['workspaces'] })
    },
  })

  return (
    <div className="mx-auto max-w-[900px] px-6 py-5">
      <Link to="/tasks" className="inline-flex items-center gap-1 text-[12px] text-muted hover:text-fg">
        <ArrowLeft className="h-3.5 w-3.5" /> {t('workspaceSettings.back')}
      </Link>

      <div className="mt-3 flex items-start justify-between gap-3">
        <div>
          <h1 className="text-[18px] font-semibold tracking-tight">{t('workspaceSettings.title')}</h1>
          <p className="mt-0.5 text-[12px] text-muted">{workspace?.name ?? id}</p>
        </div>
        <Link
          to="/workspace"
          className="inline-flex h-8 items-center gap-1.5 rounded-md border border-line2 px-2.5 text-[11.5px] text-muted hover:text-fg"
        >
          <Settings2 className="h-3.5 w-3.5" />
          {t('workspaceSettings.manage')}
        </Link>
      </div>

      <section className="mt-5 rounded-lg border border-line bg-panel/40 p-4">
        <h2 className="text-[13px] font-medium">{t('workspaceSettings.name')}</h2>
        <p className="mt-1 text-[11.5px] text-muted">{t('workspaceSettings.nameHint')}</p>
        <form
          className="mt-3 flex gap-2"
          onSubmit={(e) => {
            e.preventDefault()
            if (name.trim() && name.trim() !== workspace?.name) rename.mutate()
          }}
        >
          <Input
            value={name}
            maxLength={80}
            onChange={(e) => {
              setName(e.target.value)
              if (rename.isSuccess || rename.isError) rename.reset()
            }}
            placeholder={t('workspaceSettings.namePlaceholder')}
          />
          <Button
            type="submit"
            variant="primary"
            disabled={!name.trim() || name.trim() === workspace?.name || rename.isPending}
          >
            {t('common.save')}
          </Button>
        </form>
        <p className="mt-1 text-[10.5px] text-muted/60">
          {id}
          {workspace?.slug ? ` · ${workspace.slug}` : ''}
        </p>
        {rename.isSuccess && <p className="mt-1 text-[10.5px] text-ok">{t('workspaceSettings.saved')}</p>}
        {rename.isError && <p className="mt-1 text-[10.5px] text-err">{rename.error.message}</p>}
      </section>
    </div>
  )
}
