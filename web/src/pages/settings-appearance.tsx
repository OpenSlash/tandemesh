import { Check, ChevronsUpDown, Languages, LayoutList, Monitor, Moon, Sun } from 'lucide-react'
import { useI18n, type Lang } from '../lib/i18n'
import { useWorkspace } from '../lib/workspace'
import { useTheme, type Theme } from '../lib/theme'
import { cn } from '../lib/utils'

export default function SettingsAppearancePage() {
  const { lang, setLang, t } = useI18n()
  const { workspaceMode, setWorkspaceMode } = useWorkspace()
  const { theme, setTheme } = useTheme()

  return (
    <div className="mx-auto max-w-[760px] px-8 py-7">
      <h1 className="text-[18px] font-semibold tracking-tight">{t('settings.appearance')}</h1>
      <p className="mt-0.5 text-[12px] text-muted">{t('settings.appearanceHint')}</p>

      <section className="mt-5 rounded-lg border border-line bg-panel/40 p-4">
        <h2 className="text-[13px] font-medium">{t('settings.theme')}</h2>
        <p className="mt-1 text-[11.5px] text-muted">{t('settings.themeHint')}</p>
        <div className="mt-3 grid gap-2 sm:grid-cols-3">
          {(
            [
              { value: 'dark', label: t('settings.dark'), icon: Moon },
              { value: 'light', label: t('settings.light'), icon: Sun },
              { value: 'system', label: t('settings.system'), icon: Monitor },
            ] as { value: Theme; label: string; icon: typeof Moon }[]
          ).map((option) => (
            <button
              key={option.value}
              onClick={() => setTheme(option.value)}
              className={cn(
                'flex h-9 items-center gap-2 rounded-md border px-3 text-[12.5px]',
                theme === option.value
                  ? 'border-accent2/60 bg-panel2 text-fg'
                  : 'border-line2 text-muted hover:text-fg',
              )}
            >
              <option.icon className="h-3.5 w-3.5" />
              {option.label}
              {theme === option.value && <Check className="ml-auto h-3.5 w-3.5 text-accent2" />}
            </button>
          ))}
        </div>
      </section>

      <section className="mt-4 rounded-lg border border-line bg-panel/40 p-4">
        <div className="flex items-center gap-2">
          <Languages className="h-4 w-4 text-muted" />
          <h2 className="text-[13px] font-medium">{t('settings.language')}</h2>
        </div>
        <p className="mt-1 text-[11.5px] text-muted">{t('settings.languageHint')}</p>
        <div className="mt-3 flex gap-2">
          {(['en', 'zh'] as Lang[]).map((code) => (
            <button
              key={code}
              onClick={() => setLang(code)}
              className={cn(
                'flex h-8 items-center gap-2 rounded-md border px-3 text-[12.5px]',
                lang === code
                  ? 'border-accent2/60 bg-panel2 text-fg'
                  : 'border-line2 text-muted hover:text-fg',
              )}
            >
              {code === 'en' ? 'English' : '中文'}
              {lang === code && <Check className="h-3.5 w-3.5 text-accent2" />}
            </button>
          ))}
        </div>
      </section>

      <section className="mt-4 rounded-lg border border-line bg-panel/40 p-4">
        <div className="flex items-center gap-2">
          <LayoutList className="h-4 w-4 text-muted" />
          <h2 className="text-[13px] font-medium">{t('settings.display')}</h2>
        </div>
        <p className="mt-1 text-[11.5px] text-muted">{t('settings.displayHint')}</p>
        <div className="mt-3 grid gap-2 sm:grid-cols-2">
          <button
            onClick={() => setWorkspaceMode('select')}
            className={cn(
              'rounded-md border p-3 text-left',
              workspaceMode === 'select'
                ? 'border-accent2/60 bg-panel2'
                : 'border-line2 hover:border-line2/80',
            )}
          >
            <div className="flex items-center gap-2 text-[12.5px] font-medium">
              <ChevronsUpDown className="h-3.5 w-3.5" />
              {t('settings.dropdown')}
              {workspaceMode === 'select' && <Check className="ml-auto h-3.5 w-3.5 text-accent2" />}
            </div>
            <p className="mt-1 text-[11px] text-muted">{t('settings.dropdownDesc')}</p>
          </button>
          <button
            onClick={() => setWorkspaceMode('list')}
            className={cn(
              'rounded-md border p-3 text-left',
              workspaceMode === 'list'
                ? 'border-accent2/60 bg-panel2'
                : 'border-line2 hover:border-line2/80',
            )}
          >
            <div className="flex items-center gap-2 text-[12.5px] font-medium">
              <LayoutList className="h-3.5 w-3.5" />
              {t('settings.list')}
              {workspaceMode === 'list' && <Check className="ml-auto h-3.5 w-3.5 text-accent2" />}
            </div>
            <p className="mt-1 text-[11px] text-muted">{t('settings.listDesc')}</p>
          </button>
        </div>
      </section>
    </div>
  )
}
