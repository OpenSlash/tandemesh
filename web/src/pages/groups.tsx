import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Plus, UserPlus, UsersRound } from 'lucide-react'
import { api } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { useWorkspace } from '../lib/workspace'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Dialog, DialogContent, DialogTrigger } from '../components/ui/dialog'

export default function GroupsPage() {
  const { t } = useI18n()
  const { workspaceId } = useWorkspace()
  const qc = useQueryClient()
  const [selected, setSelected] = useState('')
  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [agentId, setAgentId] = useState('claude')
  const [role, setRole] = useState('lead')

  const { data, isLoading } = useQuery({
    queryKey: ['groups', workspaceId],
    queryFn: () => api.groups(workspaceId),
  })
  const groups = data?.items ?? []
  const active = selected || groups[0]?.id || ''

  const { data: detail } = useQuery({
    queryKey: ['group', active],
    queryFn: () => api.group(active),
    enabled: !!active,
  })

  const create = useMutation({
    mutationFn: () => api.createGroup({ workspaceId, name, description }),
    onSuccess: (group) => {
      setOpen(false)
      setName('')
      setDescription('')
      setSelected(group.id)
      qc.invalidateQueries({ queryKey: ['groups', workspaceId] })
    },
  })
  const addMember = useMutation({
    mutationFn: () => api.addGroupMember(active, { agentId, role }),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['group', active] })
      qc.invalidateQueries({ queryKey: ['groups', workspaceId] })
    },
  })

  return (
    <div className="mx-auto max-w-[1180px] px-6 py-5">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-[18px] font-semibold tracking-tight">{t('page.groups')}</h1>
          <p className="mt-0.5 text-[12px] text-muted">{t('groups.subtitle')}</p>
        </div>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button variant="primary" size="sm">
              <Plus className="h-3.5 w-3.5" /> {t('groups.new')}
            </Button>
          </DialogTrigger>
          <DialogContent title={t('groups.new')} description={t('groups.newHint')}>
            <form
              className="flex flex-col gap-3"
              onSubmit={(e) => {
                e.preventDefault()
                create.mutate()
              }}
            >
              <label className="text-[12px] text-muted">
                {t('common.name')}
                <Input className="mt-1" value={name} onChange={(e) => setName(e.target.value)} />
              </label>
              <label className="text-[12px] text-muted">
                {t('common.description')}
                <Input
                  className="mt-1"
                  value={description}
                  onChange={(e) => setDescription(e.target.value)}
                />
              </label>
              {create.error && <p className="text-[12px] text-err">{String(create.error.message)}</p>}
              <div className="flex justify-end gap-2">
                <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
                  {t('common.cancel')}
                </Button>
                <Button type="submit" variant="primary" disabled={!name || create.isPending}>
                  {t('common.create')}
                </Button>
              </div>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      {isLoading && <p className="mt-6 text-[12px] text-muted">{t('common.loading')}</p>}

      <div className="mt-4 grid grid-cols-[280px_minmax(0,1fr)] gap-4">
        <div className="space-y-2">
          {groups.map((group) => (
            <button
              key={group.id}
              onClick={() => setSelected(group.id)}
              className={`w-full rounded-lg border px-3 py-2 text-left ${
                active === group.id ? 'border-accent2/50 bg-panel2' : 'border-line bg-panel/40 hover:border-line2'
              }`}
            >
              <div className="flex items-center gap-2">
                <UsersRound className="h-3.5 w-3.5 text-muted" />
                <span className="truncate text-[12.5px]">{group.name}</span>
              </div>
              {group.description && (
                <p className="mt-0.5 truncate text-[11px] text-muted">{group.description}</p>
              )}
            </button>
          ))}
          {!isLoading && groups.length === 0 && (
            <p className="text-[12px] text-muted">{t('groups.empty')}</p>
          )}
        </div>

        <div className="rounded-lg border border-line bg-panel/30 p-4">
          {detail ? (
            <>
              <h2 className="text-[14px] font-medium">{detail.group.name}</h2>
              <p className="mt-1 text-[11.5px] text-muted">
                {detail.group.description || t('groups.noDescription')}
              </p>
              <div className="mt-3 space-y-1.5">
                {(detail.members ?? []).map((m) => (
                  <div
                    key={m.agentId}
                    className="flex items-center justify-between rounded border border-line2/60 px-2 py-1.5"
                  >
                    <span className="text-[12px]">{m.agentId}</span>
                    <span className="rounded bg-panel2 px-1.5 py-0.5 text-[10px] text-muted">
                      {m.role || t('roles.member')}
                    </span>
                  </div>
                ))}
                {(detail.members ?? []).length === 0 && (
                  <p className="text-[11.5px] text-muted">{t('groups.noMembers')}</p>
                )}
              </div>
              <form
                className="mt-3 flex items-end gap-2"
                onSubmit={(e) => {
                  e.preventDefault()
                  addMember.mutate()
                }}
              >
                <label className="flex-1 text-[11px] text-muted">
                  {t('common.agent')}
                  <Input className="mt-1" value={agentId} onChange={(e) => setAgentId(e.target.value)} />
                </label>
                <label className="w-[120px] text-[11px] text-muted">
                  {t('common.role')}
                  <Input className="mt-1" value={role} onChange={(e) => setRole(e.target.value)} />
                </label>
                <Button type="submit" variant="ghost" disabled={!agentId || addMember.isPending}>
                  <UserPlus className="h-3.5 w-3.5" /> {t('common.add')}
                </Button>
              </form>
              {addMember.error && (
                <p className="mt-1 text-[11.5px] text-err">{String(addMember.error.message)}</p>
              )}
            </>
          ) : (
            <p className="text-[12px] text-muted">{t('groups.selectPrompt')}</p>
          )}
        </div>
      </div>
    </div>
  )
}
