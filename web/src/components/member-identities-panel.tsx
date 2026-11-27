import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Trash2 } from 'lucide-react'
import { api } from '../lib/api'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { useI18n } from '../lib/i18n'

export function MemberIdentitiesPanel({ workspaceId }: { workspaceId: string }) {
  const { t } = useI18n()
  const qc = useQueryClient()
  const [provider, setProvider] = useState('github')
  const [externalId, setExternalId] = useState('')
  const [login, setLogin] = useState('')
  const [email, setEmail] = useState('')
  const [memberId, setMemberId] = useState('me')
  const [token, setToken] = useState('')
  const { data } = useQuery({ queryKey: ['member-identities', workspaceId], queryFn: () => api.memberIdentities(workspaceId) })
  const items = data?.items ?? []
  const invalidate = () => qc.invalidateQueries({ queryKey: ['member-identities', workspaceId] })
  const add = useMutation({
    mutationFn: () => api.createMemberIdentity(workspaceId, { memberId, provider, externalId, login: login || undefined, email: email || undefined, token: token || undefined }),
    onSuccess: () => { invalidate(); setExternalId(''); setLogin(''); setEmail(''); setToken('') },
  })
  const remove = useMutation({ mutationFn: (id: string) => api.deleteMemberIdentity(id), onSuccess: invalidate })

  return (
    <section className="mt-5 rounded-lg border border-line bg-panel/40 p-4">
      <h2 className="text-[13px] font-medium">{t('identities.linkedTitle')}</h2>
      <p className="mt-1 text-[11.5px] text-muted">{t('identities.linkedHint')}</p>
      <div className="mt-3 space-y-2">
        {items.map((item) => (
          <div key={item.id} className="flex flex-wrap items-center gap-2 rounded border border-line2 px-2 py-1 text-[11px]">
            <span className="font-medium">{item.memberId}</span>
            <span className="font-mono text-muted">{item.provider}</span>
            <span className="font-mono text-muted">{item.externalId}</span>
            <span className="text-muted">{item.login}</span>
            <button className="ml-auto text-muted hover:text-err" onClick={() => remove.mutate(item.id)}>
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </div>
        ))}
        {items.length === 0 && <p className="text-[11px] text-muted">{t('identities.linkedEmpty')}</p>}
      </div>
      <form className="mt-3 flex flex-wrap items-end gap-2" onSubmit={(e) => { e.preventDefault(); if (memberId && externalId) add.mutate() }}>
        <label className="text-[11px] text-muted">{t('identities.member')}<Input className="mt-1 h-7 w-28" value={memberId} onChange={(e) => setMemberId(e.target.value)} /></label>
        <label className="text-[11px] text-muted">{t('identities.provider')}<Input className="mt-1 h-7 w-24" value={provider} onChange={(e) => setProvider(e.target.value)} /></label>
        <label className="text-[11px] text-muted">{t('identities.externalId')}<Input className="mt-1 h-7 w-40" value={externalId} onChange={(e) => setExternalId(e.target.value)} /></label>
        <label className="text-[11px] text-muted">{t('identities.login')}<Input className="mt-1 h-7 w-32" value={login} onChange={(e) => setLogin(e.target.value)} /></label>
        <label className="text-[11px] text-muted">{t('identities.token')}<Input className="mt-1 h-7 w-40" type="password" value={token} onChange={(e) => setToken(e.target.value)} /></label>
        <Button type="submit" size="sm" variant="primary" disabled={!memberId || !externalId || add.isPending}>{t('common.add')}</Button>
      </form>
    </section>
  )
}
