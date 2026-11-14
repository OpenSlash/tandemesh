import { useQuery } from '@tanstack/react-query'
import { api } from '../lib/api'
import { useI18n } from '../lib/i18n'

export default function SettingsAccountPage() {
  const { t } = useI18n()
  const { data: me } = useQuery({ queryKey: ['me'], queryFn: api.me })

  return (
    <div className="mx-auto max-w-[760px] px-8 py-7">
      <h1 className="text-[18px] font-semibold tracking-tight">{t('settings.account')}</h1>
      <p className="mt-0.5 text-[12px] text-muted">{t('settings.accountHint')}</p>

      <section className="mt-5 rounded-lg border border-line bg-panel/40 p-4">
        <p className="text-[12px]">
          {t('settings.signedInAs')}: <span className="font-medium">{me?.name ?? '—'}</span>
        </p>
      </section>
    </div>
  )
}
