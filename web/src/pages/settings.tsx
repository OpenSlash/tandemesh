import { Link, NavLink, Outlet } from 'react-router-dom'
import { ArrowLeft, Monitor, Palette, Settings2, Smartphone, UserRound } from 'lucide-react'
import { useI18n } from '../lib/i18n'
import { cn } from '../lib/utils'
import { DesktopTitleBar } from '../components/app-topbar'

const groups = [
  {
    labelKey: 'settings.group.preferences',
    items: [
      { to: '/settings/appearance', labelKey: 'settings.appearance', icon: Palette },
      { to: '/settings/account', labelKey: 'settings.account', icon: UserRound },
      { to: '/settings/mobile', labelKey: 'settings.mobile', icon: Smartphone },
    ],
  },
] as const

export default function SettingsLayout() {
  const { t } = useI18n()

  return (
    <div className="flex h-screen flex-col overflow-hidden bg-bg">
      <DesktopTitleBar />
      <div className="flex min-h-0 flex-1 overflow-hidden">
        <aside className="flex w-[236px] shrink-0 flex-col overflow-y-auto border-r border-line bg-panel/60">
          <div className="border-b border-line px-4 py-4">
            <Link
              to="/tasks"
              className="inline-flex items-center gap-1.5 text-[11px] text-muted hover:text-fg"
            >
              <ArrowLeft className="h-3.5 w-3.5" />
              {t('settings.back')}
            </Link>
            <div className="mt-3 flex items-center gap-2">
              <Settings2 className="h-4 w-4 text-muted" />
              <h1 className="text-[14px] font-semibold tracking-tight">{t('settings.title')}</h1>
            </div>
          </div>

          <nav className="flex flex-1 flex-col gap-5 px-3 py-4">
            {groups.map((group) => (
              <div key={group.labelKey} className="flex flex-col gap-1">
                <p className="px-2 text-[10px] uppercase tracking-wider text-muted/60">
                  {t(group.labelKey)}
                </p>
                {group.items.map((item) => (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    className={({ isActive }) =>
                      cn(
                        'flex h-8 items-center gap-2 rounded-md px-2 text-[12.5px] text-muted transition-colors',
                        isActive ? 'bg-panel2 text-fg' : 'hover:bg-panel2/60 hover:text-fg',
                      )
                    }
                  >
                    <item.icon className="h-4 w-4" />
                    {t(item.labelKey)}
                  </NavLink>
                ))}
              </div>
            ))}

            <div className="mt-auto border-t border-line pt-3">
              <Link
                to="/workspace"
                className="flex h-8 items-center gap-2 rounded-md px-2 text-[12.5px] text-muted hover:bg-panel2/60 hover:text-fg"
              >
                <Monitor className="h-4 w-4" />
                {t('workspaceSettings.manage')}
              </Link>
            </div>
          </nav>
        </aside>

        <main className="min-h-0 min-w-0 flex-1 overflow-y-auto">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
