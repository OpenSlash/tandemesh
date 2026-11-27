import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { api } from '../lib/api'
import { Dialog, DialogContent } from './ui/dialog'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { useI18n } from '../lib/i18n'

const PROVIDERS = ['github', 'gitlab', 'jira', 'linear', 'outlook', 'notion', 'slack']

export function OAuthAppsDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const { t } = useI18n()
  const qc = useQueryClient()
  const [provider, setProvider] = useState('github')
  const [clientId, setClientId] = useState('')
  const [clientSecret, setClientSecret] = useState('')
  const [message, setMessage] = useState('')
  const { data, isLoading } = useQuery({
    queryKey: ['oauth-app', provider],
    queryFn: () => api.oauthApp(provider),
    enabled: open,
  })

  useEffect(() => {
    setClientId(data?.clientId ?? '')
    setClientSecret('')
    setMessage('')
  }, [provider, data?.configured, data?.clientId])

  const save = useMutation({
    mutationFn: () => api.saveOAuthApp(provider, { clientId, clientSecret }),
    onSuccess: () => {
      setClientSecret('')
      onOpenChange(false)
      qc.invalidateQueries({ queryKey: ['oauth-app', provider] })
      qc.invalidateQueries({ queryKey: ['github-oauth'] })
      qc.invalidateQueries({ queryKey: ['oauth-status'] })
    },
    onError: (err) => setMessage(String(err)),
  })

  const remove = useMutation({
    mutationFn: () => api.deleteOAuthApp(provider),
    onSuccess: () => {
      setMessage(t('common.removed'))
      setClientId('')
      setClientSecret('')
      qc.invalidateQueries({ queryKey: ['oauth-app', provider] })
      qc.invalidateQueries({ queryKey: ['github-oauth'] })
      qc.invalidateQueries({ queryKey: ['oauth-status'] })
    },
    onError: (err) => setMessage(String(err)),
  })

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent title={t('oauthApps.title')} description={t('oauthApps.description')}>
        <div className="flex flex-col gap-3">
          <label className="text-[11px] uppercase tracking-wider text-muted">{t('oauthApps.provider')}</label>
          <select
            className="h-9 rounded-md border border-line bg-panel px-3 text-[12px]"
            value={provider}
            onChange={(e) => setProvider(e.target.value)}
          >
            {PROVIDERS.map((id) => (
              <option key={id} value={id}>{id}</option>
            ))}
          </select>

          <label className="text-[11px] uppercase tracking-wider text-muted">{t('oauthApps.clientId')}</label>
          <Input value={clientId} onChange={(e) => setClientId(e.target.value)} placeholder={t('oauthApps.clientId')} />

          <label className="text-[11px] uppercase tracking-wider text-muted">{t('oauthApps.clientSecret')}</label>
          <Input
            type="password"
            value={clientSecret}
            onChange={(e) => setClientSecret(e.target.value)}
            placeholder={data?.configured ? t('oauthApps.keepSecret') : t('oauthApps.clientSecret')}
          />

          <label className="text-[11px] uppercase tracking-wider text-muted">{t('oauthApps.callbackUrl')}</label>
          <div className="flex gap-2">
            <Input readOnly value={data?.callbackUrl ?? ''} />
            <Button
              type="button"
              variant="secondary"
              onClick={() => navigator.clipboard?.writeText(data?.callbackUrl ?? '')}
            >
              {t('common.copy')}
            </Button>
          </div>

          <p className="text-[11px] text-muted">
            {isLoading ? t('common.loading') : data?.configured ? t('oauthApps.configured') : t('oauthApps.notConfigured')}
          </p>
          {message && <p className="text-[11px] text-muted">{message}</p>}

          <div className="mt-1 flex justify-end gap-2">
            {data?.configured && (
              <Button type="button" variant="secondary" onClick={() => remove.mutate()} disabled={remove.isPending}>
                {t('common.remove')}
              </Button>
            )}
            <Button
              type="button"
              onClick={() => save.mutate()}
              disabled={save.isPending || !clientId || (!data?.configured && !clientSecret)}
            >
              {t('common.save')}
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  )
}
