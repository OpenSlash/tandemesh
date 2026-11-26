import { NavLink } from 'react-router-dom'
import { useI18n } from '../lib/i18n'
import { cn } from '../lib/utils'

/** Switches between the host list and the agent registry that installs onto hosts. */
export function MachinesTabs() {
  const { t } = useI18n()
  const tabs = [
    { to: '/machines', label: t('machines.tabHosts'), end: true },
    { to: '/machines/registry', label: t('machines.tabRegistry'), end: false },
  ]
  return (
    <nav className="mb-4 flex gap-1 border-b border-line">
      {tabs.map((tab) => (
        <NavLink
          key={tab.to}
          to={tab.to}
          end={tab.end}
          className={({ isActive }) =>
            cn(
              '-mb-px border-b-2 px-3 py-2 text-[12.5px] transition-colors',
              isActive ? 'border-fg text-fg' : 'border-transparent text-muted hover:text-fg',
            )
          }
        >
          {tab.label}
        </NavLink>
      ))}
    </nav>
  )
}
