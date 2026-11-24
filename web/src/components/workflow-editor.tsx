import { useEffect } from 'react'
import { Save, X } from 'lucide-react'
import type { WorkflowNode, WorkflowTemplate } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { Button } from './ui/button'
import { Input } from './ui/input'
import { WorkflowCanvas } from './workflow-canvas'
import { WorkflowNodeBuilder } from './workflow-node-builder'

function parseNodes(text: string): WorkflowNode[] {
  try {
    const parsed = JSON.parse(text)
    return Array.isArray(parsed) ? parsed : []
  } catch {
    return []
  }
}

export function WorkflowEditor({
  mode,
  name,
  description,
  nodesText,
  templates = [],
  saving,
  error,
  onName,
  onDescription,
  onNodes,
  onSave,
  onClose,
  embedded = false,
}: {
  mode: 'create' | 'edit'
  name: string
  description: string
  nodesText: string
  templates?: WorkflowTemplate[]
  saving: boolean
  error?: string
  onName: (value: string) => void
  onDescription: (value: string) => void
  onNodes: (value: string) => void
  onSave: () => void
  onClose: () => void
  embedded?: boolean
}) {
  const { t } = useI18n()
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape') onClose()
      if ((event.metaKey || event.ctrlKey) && event.key === 'Enter') onSave()
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [onClose, onSave])

  return (
    <div className={embedded ? 'h-full' : 'fixed inset-0 z-50 bg-black/70 p-3 backdrop-blur-sm'}>
      <div className="mx-auto flex h-full max-w-[1560px] flex-col overflow-hidden rounded-lg border border-line2 bg-bg shadow-2xl">
        <header className="flex shrink-0 flex-wrap items-center gap-2 border-b border-line2 px-4 py-2.5">
          <div className="mr-2 min-w-[120px]">
            <p className="text-[13px] font-semibold">{mode === 'create' ? t('workflows.new') : t('workflows.edit')}</p>
            <p className="text-[10px] text-muted">{t('workflows.dragHint')}</p>
          </div>
          <Input autoFocus className="h-8 w-[220px]" placeholder={t('workflows.name')} value={name} onChange={(e) => onName(e.target.value)} />
          <Input className="h-8 min-w-[180px] flex-1" placeholder={t('workflows.description')} value={description} onChange={(e) => onDescription(e.target.value)} />
          <Button size="sm" variant="primary" disabled={!name || saving} onClick={onSave}>
            <Save className="h-3.5 w-3.5" /> {saving ? t('workflows.saving') : t('workflows.save')}
          </Button>
          <Button size="sm" variant="secondary" onClick={onClose}><X className="h-3.5 w-3.5" /> {t('workflows.close')}</Button>
        </header>
        {error && <p className="shrink-0 border-b border-line2 px-4 py-2 text-[11px] text-err">{error}</p>}
        <div className="grid min-h-0 flex-1 grid-cols-1 lg:grid-cols-[minmax(0,1fr)_380px]">
          <div className="min-h-[320px] p-3">
            <WorkflowCanvas
              nodes={parseNodes(nodesText)}
              onChange={(next) => onNodes(JSON.stringify(next, null, 2))}
              className="flex h-full flex-col"
              bodyClassName="min-h-0 flex-1"
            />
          </div>
          <aside className="min-h-0 overflow-y-auto border-t border-line2 p-4 lg:border-l lg:border-t-0">
            {mode === 'create' && templates.length > 0 && (
              <section className="mb-4">
                <p className="mb-2 text-[10.5px] uppercase tracking-wider text-muted">{t('workflows.templates')}</p>
                <div className="flex flex-wrap gap-1">
                  {templates.map((template) => (
                    <button
                      key={template.id}
                      type="button"
                      className="rounded border border-line2 px-2 py-1 text-[10.5px] text-muted hover:text-fg"
                      onClick={() => {
                        onName(template.name)
                        onDescription(template.description)
                        onNodes(JSON.stringify(template.nodes, null, 2))
                      }}
                    >
                      {template.name}
                    </button>
                  ))}
                </div>
              </section>
            )}
            <WorkflowNodeBuilder nodes={parseNodes(nodesText)} onChange={(next) => onNodes(JSON.stringify(next, null, 2))} />
            <details className="mt-4">
              <summary className="cursor-pointer text-[10.5px] uppercase tracking-wider text-muted">{t('workflows.json')}</summary>
              <textarea
                className="mt-2 min-h-52 w-full rounded-md border border-line2 bg-panel p-2 font-mono text-[11px]"
                value={nodesText}
                onChange={(e) => onNodes(e.target.value)}
              />
            </details>
          </aside>
        </div>
      </div>
    </div>
  )
}
