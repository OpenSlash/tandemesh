import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { KeyRound, Plus, Trash2 } from 'lucide-react'
import { api } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Dialog, DialogContent, DialogTrigger } from '../components/ui/dialog'

export default function CredentialsPage() {
  const { t } = useI18n()
  const qc = useQueryClient()
  const { data, isLoading, error } = useQuery({ queryKey: ['credentials'], queryFn: api.credentials })
  const { data: providers } = useQuery({ queryKey: ['providers'], queryFn: api.providers })
  const items = data?.items ?? []

  const [open, setOpen] = useState(false)
  const [provider, setProvider] = useState('github')
  const [subject, setSubject] = useState('')
  const [token, setToken] = useState('')

  const add = useMutation({
    mutationFn: () => api.putCredential({ provider, subject: subject || undefined, accessToken: token }),
    onSuccess: () => {
      setOpen(false)
      setToken('')
      qc.invalidateQueries({ queryKey: ['credentials'] })
    },
  })
  const remove = useMutation({
    mutationFn: ({ provider, subject }: { provider: string; subject: string }) =>
      api.deleteCredential(provider, subject),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['credentials'] }),
  })

  return (
    <div className="mx-auto max-w-[1180px] px-6 py-5">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-[18px] font-semibold tracking-tight">{t('page.credentials')}</h1>
          <p className="mt-0.5 text-[12px] text-muted">{t('credentials.subtitle')}</p>
        </div>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button variant="primary">
              <Plus className="h-3.5 w-3.5" /> {t('credentials.addToken')}
            </Button>
          </DialogTrigger>
          <DialogContent title={t('credentials.addTitle')} description={t('credentials.addHint')}>
            <form
              className="flex flex-col gap-3"
              onSubmit={(e) => {
                e.preventDefault()
                add.mutate()
              }}
            >
              <label className="text-[12px] text-muted">
                {t('meta.provider')}
                <select
                  className="mt-1 h-8 w-full rounded-md border border-line2 bg-panel px-2 text-[13px]"
                  value={provider}
                  onChange={(e) => setProvider(e.target.value)}
                >
                  {(providers?.items ?? []).map((p) => (
                    <option key={p.id} value={p.id}>
                      {p.name}
                    </option>
                  ))}
                </select>
              </label>
              <label className="text-[12px] text-muted">
                {t('credentials.subjectOptional')}
                <Input
                  className="mt-1"
                  value={subject}
                  onChange={(e) => setSubject(e.target.value)}
                  placeholder="default"
                />
              </label>
              <label className="text-[12px] text-muted">
                {t('credentials.token')}
                <Input
                  className="mt-1"
                  type="password"
                  value={token}
                  onChange={(e) => setToken(e.target.value)}
                  placeholder="ghp_… / xoxb-… / secret_…"
                />
              </label>
              {add.error && <p className="text-[12px] text-err">{String(add.error.message)}</p>}
              <div className="mt-1 flex justify-end gap-2">
                <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
                  {t('common.cancel')}
                </Button>
                <Button type="submit" variant="primary" disabled={!token || add.isPending}>
                  {t('common.save')}
                </Button>
              </div>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      {error && <p className="mt-4 text-[12px] text-err">{String(error.message)}</p>}
      {isLoading && <p className="mt-6 text-[12px] text-muted">{t('common.loading')}</p>}

      <div className="mt-4 overflow-hidden rounded-lg border border-line">
        <table className="w-full text-left">
          <thead>
            <tr className="bg-panel/80 text-[11px] uppercase tracking-wider text-muted/80">
              <th className="px-3 py-2 font-medium">{t('meta.provider')}</th>
              <th className="px-3 py-2 font-medium">{t('credentials.colSubject')}</th>
              <th className="px-3 py-2 font-medium">{t('credentials.colKind')}</th>
              <th className="px-3 py-2 font-medium">{t('common.updated')}</th>
              <th className="w-10 px-3 py-2" />
            </tr>
          </thead>
          <tbody>
            {items.map((c) => (
              <tr key={`${c.provider}/${c.subject}`} className="border-t border-line text-[12.5px]">
                <td className="px-3 py-2">
                  <span className="inline-flex items-center gap-1.5">
                    <KeyRound className="h-3.5 w-3.5 text-muted" />
                    {c.provider}
                  </span>
                </td>
                <td className="px-3 py-2 font-mono text-[11.5px] text-muted">{c.subject}</td>
                <td className="px-3 py-2 text-muted">{c.kind}</td>
                <td className="px-3 py-2 text-[11.5px] text-muted">
                  {new Date(c.updatedAt).toLocaleString()}
                </td>
                <td className="px-3 py-2 text-right">
                  <button
                    className="text-muted hover:text-err"
                    onClick={() => remove.mutate({ provider: c.provider, subject: c.subject })}
                    title={t('common.delete')}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!isLoading && items.length === 0 && (
          <p className="px-3 py-10 text-center text-[12px] text-muted">{t('credentials.empty')}</p>
        )}
      </div>
    </div>
  )
}
