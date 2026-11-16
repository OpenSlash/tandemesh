import { useState } from 'react'
import { useMutation, useQueryClient } from '@tanstack/react-query'
import { api, type Connection } from '../lib/api'
import { Dialog, DialogContent } from './ui/dialog'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { PlaneCatalogPicker } from './plane-catalog-picker'

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
  const qc = useQueryClient()
  const [connectionId, setConnectionId] = useState('')
  const [instance, setInstance] = useState('')
  const [name, setName] = useState('')
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
        name: name || undefined,
      }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['connections', workspaceId] })
      setInstance('')
      setName('')
      setMessage('Project bound')
    },
    onError: (err) => setMessage(String(err)),
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent title="Add project" description="Bind an external project to this workspace using an existing account.">
        <div className="flex flex-col gap-3">
          <label className="text-[12px] text-muted">
            Connection
            <select
              className="mt-1 h-8 w-full rounded-md border border-line2 bg-panel px-2 text-[12px]"
              value={connectionId}
              onChange={(e) => {
                setConnectionId(e.target.value)
                setInstance('')
                setMessage('')
              }}
            >
              <option value="">Select connection</option>
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
              onSelect={(value) => setInstance(value)}
            />
          )}

          <label className="text-[12px] text-muted">
            Instance
            <Input
              className="mt-1"
              value={instance}
              placeholder={connection?.provider === 'plane' ? 'workspace/project-uuid' : 'owner/repo'}
              onChange={(e) => setInstance(e.target.value)}
            />
          </label>
          <label className="text-[12px] text-muted">
            Name
            <Input
              className="mt-1"
              value={name}
              placeholder="Optional display name"
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
              Bind project
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
