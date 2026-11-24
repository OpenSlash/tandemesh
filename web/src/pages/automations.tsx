import { useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { CalendarClock, Play, Plus, Trash2, Zap } from 'lucide-react'
import { api, type UnifiedAutomation } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { useWorkspace } from '../lib/workspace'
import { cn } from '../lib/utils'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Dialog, DialogContent, DialogTrigger } from '../components/ui/dialog'
import { AssignmentRulesPanel } from '../components/assignment-rules-panel'

const TRIGGERS = [
  { value: 'any', labelKey: 'automations.triggerAny' },
  { value: 'created', labelKey: 'automations.triggerCreated' },
  { value: 'updated', labelKey: 'automations.triggerUpdated' },
  { value: 'label', labelKey: 'automations.triggerLabel' },
  { value: 'assign', labelKey: 'automations.triggerAssign' },
  { value: 'comment', labelKey: 'automations.triggerComment' },
  { value: 'pipeline', labelKey: 'automations.triggerPipeline' },
  { value: 'rule', labelKey: 'automations.triggerRule' },
]

type Tab = 'all' | 'scheduled' | 'event'
type TriggerType = 'scheduled' | 'event'

export default function AutomationsPage() {
  const { t } = useI18n()
  const qc = useQueryClient()
  const { workspaceId } = useWorkspace()
  const [searchParams, setSearchParams] = useSearchParams()
  const tab: Tab =
    searchParams.get('type') === 'event'
      ? 'event'
      : searchParams.get('type') === 'scheduled'
        ? 'scheduled'
        : 'all'

  const { data, isLoading, error } = useQuery({
    queryKey: ['automations', workspaceId],
    queryFn: () => api.automations(workspaceId),
  })
  const { data: agents } = useQuery({
    queryKey: ['agents', workspaceId],
    queryFn: () => api.agents(workspaceId),
  })
  const { data: providers } = useQuery({ queryKey: ['providers'], queryFn: api.providers })
  const taskProviders = (providers?.items ?? []).filter((p) => p.kind === 'task')

  const [open, setOpen] = useState(false)
  const [triggerType, setTriggerType] = useState<TriggerType>('scheduled')
  const [name, setName] = useState('')
  const [agentId, setAgentId] = useState('claude')
  const [repo, setRepo] = useState('acme/api')
  const [schedule, setSchedule] = useState('0 9 * * 1-5')
  const [prompt, setPrompt] = useState('')
  const [precheck, setPrecheck] = useState('')
  const [grace, setGrace] = useState(0)
  const [reuseSession, setReuseSession] = useState(false)
  const [provider, setProvider] = useState('github')
  const [instance, setInstance] = useState('acme/api')
  const [triggerKind, setTriggerKind] = useState('label')
  const [triggerValue, setTriggerValue] = useState('agent:fix')
  const [statusWriteback, setStatusWriteback] = useState(false)
  const [pipelineRetries, setPipelineRetries] = useState(3)
  const [pipelineBranches, setPipelineBranches] = useState('')
  const [pipelineLabels, setPipelineLabels] = useState('')
  const [pipelineCooldown, setPipelineCooldown] = useState(0)
  const [customFields, setCustomFields] = useState('')

  const invalidate = () => {
    qc.invalidateQueries({ queryKey: ['automations', workspaceId] })
  }
  const createScheduled = useMutation({
    mutationFn: () =>
      api.createAutomation({
        kind: 'schedule',
        workspaceId,
        name,
        schedule,
        agentId,
        repo,
        prompt,
        precheck: precheck || undefined,
        missedGraceSeconds: grace,
        reuseSession,
        enabled: true,
      }),
    onSuccess: resetAndClose,
  })
  const createEvent = useMutation({
    mutationFn: () =>
      api.createAutomation({
        kind: 'event',
        workspaceId,
        name,
        provider,
        instance,
        agentId,
        triggerKind,
        triggerValue,
        writeback: {
          status: statusWriteback,
          pipelineMaxRetries: pipelineRetries,
          pipelineBranches,
          pipelineLabels,
          pipelineCooldownSeconds: pipelineCooldown,
          customFields,
        },
        enabled: true,
      }),
    onSuccess: resetAndClose,
  })
  function resetAndClose() {
    setOpen(false)
    setName('')
    setPrompt('')
    setPrecheck('')
    setGrace(0)
    setReuseSession(false)
    invalidate()
  }
  const toggle = useMutation({
    mutationFn: ({ kind, id, enabled }: { kind: 'schedule' | 'event'; id: string; enabled: boolean }) =>
      api.updateAutomation(id, { kind, enabled }),
    onSuccess: invalidate,
  })
  const runNow = useMutation({ mutationFn: (id: string) => api.runAutomation(id), onSuccess: invalidate })
  const removeSchedule = useMutation({ mutationFn: (id: string) => api.deleteAutomation(id), onSuccess: invalidate })

  const rows = (data?.items ?? [])
    .filter((row: UnifiedAutomation) => row.triggerKind !== 'task.synced')
    .filter((row: UnifiedAutomation) =>
      tab === 'all' ? true : tab === 'scheduled' ? row.kind === 'schedule' : row.kind === 'event',
    )
    .map((row: UnifiedAutomation) => ({
      ...row,
      trigger:
        row.kind === 'schedule'
          ? row.schedule ?? ''
          : `${row.provider} · ${row.triggerKind}${row.triggerValue ? `: ${row.triggerValue}` : ''}`,
      timing:
        row.kind === 'schedule'
          ? row.enabled && row.nextRun
            ? new Date(row.nextRun).toLocaleString()
            : t('common.paused')
          : row.instance || t('automations.anyInstance'),
      detail: row.kind === 'schedule' ? row.prompt ?? '' : row.triggerValue ?? '',
      status: row.lastStatus ?? '',
    }))

  const createError = triggerType === 'scheduled' ? createScheduled.error : createEvent.error
  const creating = triggerType === 'scheduled' ? createScheduled.isPending : createEvent.isPending
  const canCreate =
    name.trim() !== '' && (triggerType === 'scheduled' ? prompt.trim() !== '' : instance.trim() !== '')

  return (
    <div className="mx-auto max-w-[1180px] px-6 py-5">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-[18px] font-semibold tracking-tight">{t('page.automations')}</h1>
          <p className="mt-0.5 text-[12px] text-muted">{t('automations.unifiedSubtitle')}</p>
        </div>
        <Dialog
          open={open}
          onOpenChange={(value) => {
            setOpen(value)
            if (value) setTriggerType(tab === 'event' ? 'event' : 'scheduled')
          }}
        >
          <DialogTrigger asChild>
            <Button variant="primary">
              <Plus className="h-3.5 w-3.5" /> {t('automations.new')}
            </Button>
          </DialogTrigger>
          <DialogContent
            title={t('automations.new')}
            description={t('automations.unifiedNewHint')}
            className="max-h-[85vh] overflow-y-auto"
          >
            <form
              className="flex flex-col gap-3"
              onSubmit={(e) => {
                e.preventDefault()
                if (triggerType === 'scheduled') createScheduled.mutate()
                else createEvent.mutate()
              }}
            >
              <div className="flex gap-1 rounded-md bg-panel2 p-1">
                {(['scheduled', 'event'] as TriggerType[]).map((value) => (
                  <button
                    key={value}
                    type="button"
                    onClick={() => setTriggerType(value)}
                    className={cn(
                      'flex h-7 flex-1 items-center justify-center gap-1 rounded text-[11.5px]',
                      triggerType === value ? 'bg-panel text-fg shadow-sm' : 'text-muted hover:text-fg',
                    )}
                  >
                    {value === 'scheduled' ? <CalendarClock className="h-3.5 w-3.5" /> : <Zap className="h-3.5 w-3.5" />}
                    {value === 'scheduled' ? t('automations.typeSchedule') : t('automations.typeEvent')}
                  </button>
                ))}
              </div>
              <label className="text-[12px] text-muted">
                {t('common.name')}
                <Input className="mt-1" value={name} onChange={(e) => setName(e.target.value)} />
              </label>
              <label className="text-[12px] text-muted">
                {t('common.agent')}
                <select
                  className="mt-1 h-8 w-full rounded-md border border-line2 bg-panel px-2 text-[13px]"
                  value={agentId}
                  onChange={(e) => setAgentId(e.target.value)}
                >
                  <option value="claude">claude</option>
                  {(agents?.items ?? []).map((a) => (
                    <option key={a.id} value={a.id}>
                      {a.name}
                    </option>
                  ))}
                </select>
              </label>

              {triggerType === 'scheduled' ? (
                <>
                  <label className="text-[12px] text-muted">
                    {t('automations.schedule')}
                    <Input className="mt-1 font-mono" value={schedule} onChange={(e) => setSchedule(e.target.value)} />
                  </label>
                  <label className="text-[12px] text-muted">
                    {t('common.repository')}
                    <Input className="mt-1" value={repo} onChange={(e) => setRepo(e.target.value)} />
                  </label>
                  <label className="text-[12px] text-muted">
                    {t('common.prompt')}
                    <textarea
                      className="mt-1 min-h-20 w-full rounded-md border border-line2 bg-panel px-2 py-1 text-[12px]"
                      value={prompt}
                      onChange={(e) => setPrompt(e.target.value)}
                    />
                  </label>
                  <label className="text-[12px] text-muted">
                    {t('automations.precheck')}
                    <Input className="mt-1 font-mono" value={precheck} onChange={(e) => setPrecheck(e.target.value)} placeholder="npm run precheck" />
                  </label>
                  <label className="text-[12px] text-muted">
                    {t('automations.missedGrace')}
                    <Input className="mt-1" type="number" value={grace} onChange={(e) => setGrace(Number(e.target.value))} />
                  </label>
                  <label className="flex items-center gap-2 text-[12px] text-muted">
                    <input type="checkbox" checked={reuseSession} onChange={(e) => setReuseSession(e.target.checked)} />
                    {t('automations.reuseSession')}
                  </label>
                </>
              ) : (
                <>
                  <label className="text-[12px] text-muted">
                    {t('meta.provider')}
                    <select
                      className="mt-1 h-8 w-full rounded-md border border-line2 bg-panel px-2 text-[13px]"
                      value={provider}
                      onChange={(e) => setProvider(e.target.value)}
                    >
                      {taskProviders.map((p) => (
                        <option key={p.id} value={p.id}>
                          {p.name}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label className="text-[12px] text-muted">
                    {t('meta.instance')}
                    <Input className="mt-1" value={instance} onChange={(e) => setInstance(e.target.value)} />
                  </label>
                  <div className="grid grid-cols-2 gap-2">
                    <label className="text-[12px] text-muted">
                      {t('automations.trigger')}
                      <select
                        className="mt-1 h-8 w-full rounded-md border border-line2 bg-panel px-2 text-[13px]"
                        value={triggerKind}
                        onChange={(e) => setTriggerKind(e.target.value)}
                      >
                        {TRIGGERS.map((trigger) => (
                          <option key={trigger.value} value={trigger.value}>
                            {t(trigger.labelKey)}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label className="text-[12px] text-muted">
                      {t('automations.value')}
                      <Input className="mt-1" value={triggerValue} onChange={(e) => setTriggerValue(e.target.value)} />
                    </label>
                  </div>
                  <label className="flex items-center gap-2 text-[12px] text-muted">
                    <input type="checkbox" checked={statusWriteback} onChange={(e) => setStatusWriteback(e.target.checked)} />
                    {t('automations.writebackStatus')}
                  </label>
                  <label className="text-[12px] text-muted">
                    {t('automations.customFields')}
                    <Input className="mt-1" value={customFields} onChange={(e) => setCustomFields(e.target.value)} placeholder="customfield_10010=Story Points" />
                  </label>
                  {triggerKind === 'pipeline' && (
                    <div className="grid grid-cols-2 gap-2 rounded-md border border-line bg-panel/40 p-2">
                      <label className="text-[12px] text-muted">
                        {t('automations.pipelineRetries')}
                        <Input
                          className="mt-1"
                          type="number"
                          value={pipelineRetries}
                          onChange={(e) => setPipelineRetries(Number(e.target.value))}
                        />
                      </label>
                      <label className="text-[12px] text-muted">
                        {t('automations.pipelineCooldown')}
                        <Input
                          className="mt-1"
                          type="number"
                          value={pipelineCooldown}
                          onChange={(e) => setPipelineCooldown(Number(e.target.value))}
                        />
                      </label>
                      <label className="text-[12px] text-muted">
                        {t('automations.pipelineBranches')}
                        <Input className="mt-1" value={pipelineBranches} onChange={(e) => setPipelineBranches(e.target.value)} placeholder="main,release/*" />
                      </label>
                      <label className="text-[12px] text-muted">
                        {t('automations.pipelineLabels')}
                        <Input className="mt-1" value={pipelineLabels} onChange={(e) => setPipelineLabels(e.target.value)} placeholder="bug,critical" />
                      </label>
                    </div>
                  )}
                </>
              )}

              {createError && <p className="text-[12px] text-err">{String(createError.message)}</p>}
              <div className="mt-1 flex justify-end gap-2">
                <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
                  {t('common.cancel')}
                </Button>
                <Button type="submit" variant="primary" disabled={!canCreate || creating}>
                  {t('common.create')}
                </Button>
              </div>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      <div className="mt-3 flex gap-1 border-b border-line">
        {(['all', 'scheduled', 'event'] as Tab[]).map((value) => (
          <button
            key={value}
            onClick={() => setSearchParams(value === 'all' ? {} : { type: value })}
            className={cn(
              'px-2 py-1.5 text-[12px]',
              tab === value ? 'border-b-2 border-fg font-medium text-fg' : 'text-muted hover:text-fg',
            )}
          >
            {value === 'all'
              ? t('automations.tabAll')
              : value === 'scheduled'
                ? t('automations.tabScheduled')
                : t('automations.tabEvent')}
          </button>
        ))}
      </div>

      {error && <p className="mt-4 text-[12px] text-err">{String(error.message)}</p>}
      {isLoading && <p className="mt-6 text-[12px] text-muted">{t('common.loading')}</p>}

      <div className="mt-4 overflow-hidden rounded-lg border border-line">
        <table className="w-full text-left">
          <thead>
            <tr className="bg-panel/80 text-[11px] uppercase tracking-wider text-muted/80">
              <th className="px-3 py-2 font-medium">{t('common.name')}</th>
              <th className="px-3 py-2 font-medium">{t('automations.colTrigger')}</th>
              <th className="px-3 py-2 font-medium">{t('common.agent')}</th>
              <th className="px-3 py-2 font-medium">{t('automations.colTiming')}</th>
              <th className="w-32 px-3 py-2 font-medium">{t('common.actions')}</th>
            </tr>
          </thead>
          <tbody>
            {rows.map((row) => (
              <tr key={`${row.kind}-${row.id}`} className="border-t border-line text-[12.5px]">
                <td className="px-3 py-2">
                  <div className="flex items-center gap-2">
                    {row.kind === 'schedule' ? (
                      <CalendarClock className={cn('h-3.5 w-3.5', row.enabled ? 'text-accent2' : 'text-muted')} />
                    ) : (
                      <Zap className={cn('h-3.5 w-3.5', row.enabled ? 'text-accent2' : 'text-muted')} />
                    )}
                    <span>{row.name}</span>
                    <span className="rounded bg-panel2 px-1 text-[10px] text-muted">
                      {row.kind === 'schedule' ? t('automations.typeSchedule') : t('automations.typeEvent')}
                    </span>
                  </div>
                  <div className="mt-0.5 truncate text-[11px] text-muted">{row.detail}</div>
                  {row.status && <div className="mt-0.5 text-[10px] text-muted">{row.status}</div>}
                </td>
                <td className="px-3 py-2 font-mono text-[11.5px] text-muted">{row.trigger}</td>
                <td className="px-3 py-2 text-muted">{row.agentId}</td>
                <td className="px-3 py-2 text-[11.5px] text-muted">{row.timing}</td>
                <td className="px-3 py-2">
                  <div className="flex items-center gap-2">
                    {row.kind === 'schedule' ? (
                      <>
                        <button className="text-muted hover:text-fg" onClick={() => runNow.mutate(row.id)} title={t('automations.runNow')}>
                          <Play className="h-3.5 w-3.5" />
                        </button>
                        <button
                          className={cn('text-[11px]', row.enabled ? 'text-ok' : 'text-muted')}
                          onClick={() => toggle.mutate({ kind: 'schedule', id: row.id, enabled: !row.enabled })}
                        >
                          {row.enabled ? t('common.enabled') : t('common.paused')}
                        </button>
                        <button className="text-muted hover:text-err" onClick={() => removeSchedule.mutate(row.id)} title={t('common.delete')}>
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </>
                    ) : (
                      <>
                        <button
                          className={cn('text-[11px]', row.enabled ? 'text-ok' : 'text-muted')}
                          onClick={() => toggle.mutate({ kind: 'event', id: row.id, enabled: !row.enabled })}
                        >
                          {row.enabled ? t('common.enabled') : t('common.paused')}
                        </button>
                        <button className="text-muted hover:text-err" onClick={() => removeSchedule.mutate(row.id)} title={t('common.delete')}>
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </>
                    )}
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
        {!isLoading && rows.length === 0 && (
          <p className="px-3 py-10 text-center text-[12px] text-muted">{t('automations.emptyUnified')}</p>
        )}
      </div>
      <AssignmentRulesPanel />
    </div>
  )
}
