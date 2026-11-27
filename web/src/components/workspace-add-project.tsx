import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api, type Connection } from '../lib/api'
import { Dialog, DialogContent } from './ui/dialog'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { PlaneCatalogPicker } from './plane-catalog-picker'
import { useI18n } from '../lib/i18n'

export function WorkspaceAddProject({
  workspaceId,
  connections,
  open,
  onOpenChange,
}: {
  workspaceId: string
  connections: Connection[]
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const { t } = useI18n()
  const qc = useQueryClient()
  const [connectionId, setConnectionId] = useState('')
  const [instance, setInstance] = useState('')
  const [name, setName] = useState('')
  const [projectName, setProjectName] = useState('')
  const [message, setMessage] = useState('')
  const connection = connections.find((item) => item.id === connectionId)
  const save = useMutation({
    mutationFn: () =>
      api.connectProvider({
        provider: connection?.provider ?? '',
        token: '',
        workspaceId,
        reuseConnectionId: connectionId,
        baseUrl: connection?.baseUrl,
        instance,
        name: name || projectName || undefined,
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['connections', workspaceId] })
      setInstance('')
      setName('')
      setMessage(t('addProject.bound'))
    },
    onError: (err) => setMessage(String(err)),
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent title={t('addProject.title')} description={t('addProject.description')}>
        <div className="flex flex-col gap-3">
          <label className="text-[12px] text-muted">
            {t('addProject.connection')}
            <select
              className="mt-1 h-8 w-full rounded-md border border-line2 bg-panel px-2 text-[12px]"
              value={connectionId}
              onChange={(e) => {
                setConnectionId(e.target.value)
                setInstance('')
                setMessage('')
              }}
            >
              <option value="">{t('addProject.selectConnection')}</option>
              {connections.map((item) => (
                <option key={item.id} value={item.id}>
                  {item.name || item.provider} · {item.provider}
                </option>
              ))}
            </select>
          </label>

          {connection?.provider === 'plane' && (
            <PlaneCatalogPicker
              token=""
              baseUrl={connection.baseUrl}
              connectionId={connection.id}
              initialWorkspace={(connection.allBindings ?? connection.bindings)?.[0]?.externalId.split('/')[0]}
              onSelect={(value, project) => {
                setInstance(value)
                setProjectName(project ?? '')
              }}
            />
          )}

          <label className="text-[12px] text-muted">
            {t('addProject.instance')}
            <Input
              className="mt-1"
              value={instance}
              placeholder={connection?.provider === 'plane' ? 'workspace/project-uuid' : 'owner/repo'}
              onChange={(e) => setInstance(e.target.value)}
            />
          </label>
          <label className="text-[12px] text-muted">
            {t('addProject.name')}
            <Input
              className="mt-1"
              value={name}
              placeholder={t('addProject.namePlaceholder')}
              onChange={(e) => setName(e.target.value)}
            />
          </label>

          {message && <p className="text-[11px] text-muted">{message}</p>}
          <div className="flex justify-end">
            <Button
              type="button"
              variant="primary"
              disabled={!connection || !instance.trim() || save.isPending}
              onClick={() => save.mutate()}
            >
              {t('addProject.bind')}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
