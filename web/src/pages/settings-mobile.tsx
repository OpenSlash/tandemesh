import { useEffect, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Check, Copy, Loader2, Smartphone, Trash2 } from 'lucide-react'
import { api, type MobileDevice } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { Button } from '../components/ui/button'

function formatWhen(value: number) {
  return value ? new Date(value).toLocaleString() : '—'
}

export default function SettingsMobilePage() {
  const { t } = useI18n()
  const qc = useQueryClient()
  const [pairing, setPairing] = useState<{ code: string; deepLink: string; expiresAt: number } | null>(null)
  const [copied, setCopied] = useState(false)
  const [remaining, setRemaining] = useState(0)

  const { data: devicesData, isLoading } = useQuery({
    queryKey: ['mobile-devices'],
    queryFn: api.mobileDevices,
  })
  const devices = devicesData?.items ?? []

  const createPairing = useMutation({
    mutationFn: () => api.createMobilePairing(),
    onSuccess: (result) => {
      const deepLink = `tandemesh://pair?host=${encodeURIComponent(window.location.origin)}&code=${encodeURIComponent(result.code)}`
      setPairing({ ...result, deepLink })
      setCopied(false)
    },
  })

  const revoke = useMutation({
    mutationFn: (id: string) => api.revokeMobileDevice(id),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['mobile-devices'] }),
  })

  useEffect(() => {
    if (!pairing) return
    const update = () => setRemaining(Math.max(0, Math.floor((pairing.expiresAt - Date.now()) / 1000)))
    update()
    const timer = window.setInterval(update, 1000)
    return () => window.clearInterval(timer)
  }, [pairing])

  const copy = async () => {
    if (!pairing) return
    try {
      await navigator.clipboard.writeText(pairing.deepLink)
      setCopied(true)
      window.setTimeout(() => setCopied(false), 1500)
    } catch {
      setCopied(false)
    }
  }

  return (
    <div className="mx-auto max-w-[760px] px-8 py-7">
      <h1 className="flex items-center gap-2 text-[18px] font-semibold tracking-tight">
        <Smartphone className="h-4 w-4 text-muted" />
        {t('mobile.title')}
      </h1>
      <p className="mt-0.5 text-[12px] text-muted">{t('mobile.subtitle')}</p>

      <section className="mt-5 rounded-lg border border-line bg-panel/40 p-4">
        <div className="flex items-start justify-between gap-3">
          <div>
            <h2 className="text-[13px] font-medium">{t('mobile.pairDevice')}</h2>
            <p className="mt-0.5 text-[11.5px] text-muted">{t('mobile.pairHint')}</p>
          </div>
          <Button variant="primary" size="sm" onClick={() => createPairing.mutate()} disabled={createPairing.isPending}>
            {createPairing.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : null}
            {t('mobile.generate')}
          </Button>
        </div>

        {createPairing.error && (
          <p className="mt-3 text-[11.5px] text-err">{String(createPairing.error.message)}</p>
        )}

        {pairing && (
          <div className="mt-4 rounded-md border border-line2 bg-panel2/40 p-4">
            <p className="text-[10.5px] uppercase tracking-wider text-muted">{t('mobile.code')}</p>
            <p className="mt-1 font-mono text-[24px] font-semibold tracking-[0.2em]">{pairing.code}</p>
            <p className="mt-1 text-[11px] text-muted">
              {remaining > 0 ? t('mobile.expiresIn', { seconds: remaining }) : t('mobile.expired')}
            </p>
            <div className="mt-3 flex items-center gap-2">
              <Button variant="secondary" size="sm" onClick={copy}>
                {copied ? <Check className="h-3.5 w-3.5" /> : <Copy className="h-3.5 w-3.5" />}
                {copied ? t('mobile.copied') : t('mobile.copyLink')}
              </Button>
              <span className="truncate font-mono text-[10.5px] text-muted">{pairing.deepLink}</span>
            </div>
          </div>
        )}
      </section>

      <section className="mt-5 rounded-lg border border-line bg-panel/40 p-4">
        <h2 className="text-[13px] font-medium">{t('mobile.devices')}</h2>
        <p className="mt-0.5 text-[11.5px] text-muted">{t('mobile.devicesHint')}</p>
        {isLoading ? (
          <p className="mt-3 text-[11.5px] text-muted">{t('mobile.loading')}</p>
        ) : devices.length === 0 ? (
          <p className="mt-3 text-[11.5px] text-muted">{t('mobile.noDevices')}</p>
        ) : (
          <div className="mt-3 divide-y divide-line rounded-md border border-line">
            {devices.map((device: MobileDevice) => (
              <div key={device.id} className="flex items-center gap-3 px-3 py-2">
                <Smartphone className="h-3.5 w-3.5 text-muted" />
                <div className="min-w-0 flex-1">
                  <p className="truncate text-[12px]">{device.name || device.id}</p>
                  <p className="text-[10.5px] text-muted">
                    {device.platform || 'unknown'} · {formatWhen(device.createdAt)} · {formatWhen(device.lastUsedAt)}
                  </p>
                </div>
                {device.revokedAt ? (
                  <span className="text-[10.5px] text-muted">{t('mobile.revoked')}</span>
                ) : (
                  <button
                    className="text-muted hover:text-err"
                    title={t('mobile.revoke')}
                    onClick={() => revoke.mutate(device.id)}
                    disabled={revoke.isPending}
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                )}
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  )
}
