import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api, type Connection } from '../lib/api'
import { Dialog, DialogContent } from './ui/dialog'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { PlaneCatalogPicker, type PlaneSelection } from './plane-catalog-picker'
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
  const [plane, setPlane] = useState<PlaneSelection>({ instances: [], names: {} })
  const [message, setMessage] = useState('')
  const connection = connections.find((item) => item.id === connectionId)
  const isPlane = connection?.provider === 'plane'
  const targets = isPlane ? plane.instances : instance.trim() ? [instance.trim()] : []
  const save = useMutation({
    mutationFn: async () => {
      for (const target of targets) {
        await api.connectProvider({
          provider: connection?.provider ?? '',
          token: '',
          workspaceId,
          reuseConnectionId: connectionId,
          baseUrl: connection?.baseUrl,
          instance: target,
          name: (isPlane ? plane.names[target] : name) || undefined,
        })
      }
    },
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['connections'] })
      setInstance('')
      setName('')
      setMessage('')
      onOpenChange(false)
    },
    onError: (err) => setMessage(err instanceof Error ? err.message : String(err)),
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

          {isPlane && connection && (
            <PlaneCatalogPicker
              token=""
              baseUrl={connection.baseUrl}
              connectionId={connection.id}
              workspaceId={workspaceId}
              initialWorkspace={(connection.allBindings ?? connection.bindings)?.[0]?.externalId.split('/')[0]}
              onChange={setPlane}
            />
          )}

          {!isPlane && (
            <>
              <label className="text-[12px] text-muted">
                {t('addProject.instance')}
                <Input
                  className="mt-1"
                  value={instance}
                  placeholder="owner/repo"
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
            </>
          )}

          {message && <p className="text-[11px] text-err">{message}</p>}
          <div className="flex justify-end">
            <Button
              type="button"
              variant="primary"
              disabled={!connection || targets.length === 0 || save.isPending}
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
