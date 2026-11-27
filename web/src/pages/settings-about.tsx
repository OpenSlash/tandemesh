import { useMutation, useQuery } from '@tanstack/react-query'
import { CheckCircle2, Download, Loader2, RefreshCw } from 'lucide-react'
import { useI18n } from '../lib/i18n'
import { desktopApi } from '../lib/desktop'
import { isDesktopRuntime } from '../components/app-topbar'
import { Button } from '../components/ui/button'

export default function SettingsAboutPage() {
  const { t } = useI18n()
  const desktop = isDesktopRuntime()

  return (
    <div className="mx-auto max-w-[760px] px-8 py-7">
      <h1 className="text-[18px] font-semibold tracking-tight">{t('settings.about')}</h1>
      <p className="mt-0.5 text-[12px] text-muted">{t('settings.aboutHint')}</p>

      <section className="mt-5 rounded-lg border border-line bg-panel/40 p-4">
        <h2 className="text-[13px] font-medium">{t('about.version')}</h2>
        <p className="mt-1 font-mono text-[12px]">{__APP_VERSION__}</p>
      </section>

      <section className="mt-4 rounded-lg border border-line bg-panel/40 p-4">
        <h2 className="text-[13px] font-medium">{t('about.updates')}</h2>
        {desktop ? <DesktopUpdates /> : <p className="mt-1 text-[11.5px] text-muted">{t('about.webHint')}</p>}
      </section>
    </div>
  )
}

function DesktopUpdates() {
  const { t } = useI18n()
  // Shares its cache with the sidebar badge, so a check here updates both.
  const check = useQuery({ queryKey: ['desktop-update'], queryFn: desktopApi.checkUpdate, retry: false })
  const install = useMutation({ mutationFn: desktopApi.installUpdate })
  const status = check.data

  if (status && !status.supported) {
    return <p className="mt-1 text-[11.5px] text-muted">{t('about.unsupported')}</p>
  }
  return (
    <div className="mt-2 flex flex-col gap-3">
      <div className="flex items-center gap-3">
        <Button variant="secondary" size="sm" disabled={check.isFetching || install.isPending} onClick={() => void check.refetch()}>
          {check.isFetching ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
          {check.isFetching ? t('about.checking') : t('about.check')}
        </Button>
        {!check.isFetching && status && !status.available && (
          <span className="inline-flex items-center gap-1 text-[11.5px] text-ok">
            <CheckCircle2 className="h-3.5 w-3.5" /> {t('about.upToDate')}
          </span>
        )}
        {check.error && <span className="text-[11.5px] text-err">{t('about.failed', { error: check.error.message })}</span>}
      </div>
      {status?.available && (
        <div className="rounded-md border border-accent2/40 bg-accent/5 p-3">
          <p className="text-[12.5px] font-medium">{t('about.available', { version: status.available })}</p>
          {status.notes && <p className="mt-1 whitespace-pre-wrap break-words text-[11.5px] text-muted">{status.notes}</p>}
          <Button className="mt-2" variant="primary" size="sm" disabled={install.isPending || install.isSuccess} onClick={() => install.mutate()}>
            {install.isPending ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Download className="h-3.5 w-3.5" />}
            {install.isPending ? t('about.installing') : install.isSuccess ? t('about.restarting') : t('about.install')}
          </Button>
          {install.error && <p className="mt-2 text-[11.5px] text-err">{install.error.message}</p>}
        </div>
      )}
    </div>
  )
}
