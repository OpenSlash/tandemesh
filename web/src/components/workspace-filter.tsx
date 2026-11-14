import { useSearchParams } from 'react-router-dom'
import { useWorkspace } from '../lib/workspace'
import { useI18n } from '../lib/i18n'

export function WorkspaceFilter() {
  const { t } = useI18n()
  const { workspaces } = useWorkspace()
  const [searchParams, setSearchParams] = useSearchParams()
  const selected = searchParams.get('workspace') || 'all'

  return (
    <select
      value={selected}
      onChange={(event) => {
        const next = new URLSearchParams(searchParams)
        next.delete('scope')
        if (event.target.value === 'all') next.delete('workspace')
        else next.set('workspace', event.target.value)
        setSearchParams(next)
      }}
      title={t('common.allWorkspaces')}
      className="h-7 rounded-md border border-line2 bg-panel px-2 text-[11.5px] text-fg"
    >
      <option value="all">{t('common.allWorkspaces')}</option>
      {workspaces.map((workspace) => (
        <option key={workspace.id} value={workspace.id}>
          {workspace.name}
        </option>
      ))}
    </select>
  )
}
