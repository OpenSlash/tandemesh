import { Link, useSearchParams } from 'react-router-dom'
import { useI18n } from '../lib/i18n'
import { cn } from '../lib/utils'

export function MyWorkTabs({ current }: { current: 'inbox' | 'assigned' }) {
  const { t } = useI18n()
  const [searchParams] = useSearchParams()
  const workspace = searchParams.get('workspace')
  const query = workspace ? `?workspace=${encodeURIComponent(workspace)}` : ''

  const tabs = [
    { id: 'inbox', to: `/inbox${query}`, label: t('myWork.inbox') },
    { id: 'assigned', to: `/my-tasks${query}`, label: t('myWork.assigned') },
  ] as const

  return (
    <div className="mb-4 flex items-center gap-4 border-b border-line">
      {tabs.map((tab) => (
        <Link
          key={tab.id}
          to={tab.to}
          className={cn(
            '-mb-px border-b-2 px-0.5 pb-2 text-[12px]',
            current === tab.id
              ? 'border-accent2 text-fg'
              : 'border-transparent text-muted hover:text-fg',
          )}
        >
          {tab.label}
        </Link>
      ))}
    </div>
  )
}
