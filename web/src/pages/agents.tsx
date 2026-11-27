import { useEffect, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Bot, Check, ChevronDown, Plus, Trash2, AlertTriangle } from 'lucide-react'
import { useSearchParams } from 'react-router-dom'
import { api, type Agent, type Machine } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { useWorkspace } from '../lib/workspace'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Dialog, DialogContent, DialogTrigger } from '../components/ui/dialog'
import { AgentIcon, registryIdsByBackend } from '../components/agent-icon'
import { isLocalHost } from '../lib/hosts'

const MODEL_OPTIONS: Record<string, string[]> = {
  claude: ['claude-sonnet-4-5', 'claude-opus-4-1', 'claude-haiku-4-5'],
  codex: ['gpt-5-codex', 'gpt-5', 'o4-mini'],
  opencode: ['claude-sonnet-4-5', 'gpt-5', 'qwen3-coder'],
  deepseek: ['deepseek-chat', 'deepseek-reasoner'],
  gemini: ['gemini-2.5-pro', 'gemini-2.5-flash'],
  cursor: ['auto', 'claude-sonnet-4-5', 'gpt-5'],
  qwen: ['qwen3-coder-plus', 'qwen3-coder-flash'],
}
export default function AgentsPage() {
  const { t } = useI18n()
  const qc = useQueryClient()
  const { workspaceId, workspaces } = useWorkspace()
  const [searchParams] = useSearchParams()
  const scope = searchParams.get('scope') === 'all' ? 'all' : workspaceId
  const { data, isLoading, error } = useQuery({
    queryKey: ['agents', scope],
    queryFn: () => api.agents(scope),
  })
  const items = data?.items ?? []
  const { data: machinesData } = useQuery({
    queryKey: ['machines'],
    queryFn: api.machines,
    refetchInterval: 15000,
  })
  const machines = machinesData?.items ?? []
  const { data: runtimeCatalog } = useQuery({ queryKey: ['runtimes'], queryFn: api.runtimes })
  const registryIds = registryIdsByBackend(runtimeCatalog?.items ?? [], machines)
  const available = new Set(
    machines.flatMap((machine) => [...machine.agents, ...Object.keys(machine.runtimes ?? {})]),
  )
  const missing = [...new Set(items.map((agent) => agent.backend))].filter((b) => !available.has(b))

  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [instructions, setInstructions] = useState('')
  const [backend, setBackend] = useState('claude')
  const [machineId, setMachineId] = useState('')
  const [model, setModel] = useState('')
  const [customModel, setCustomModel] = useState('')
  const [skills, setSkills] = useState<string[]>([])
  const [askTools, setAskTools] = useState<string[]>([])
  const [repos, setRepos] = useState('')
  const [sessionMode, setSessionMode] = useState<'off' | 'task' | 'agent'>('off')
  const [createWorkspaceIds, setCreateWorkspaceIds] = useState<string[]>([workspaceId])
  const [editWorkspaceIds, setEditWorkspaceIds] = useState<string[]>([])
  const [editFor, setEditFor] = useState<Agent | null>(null)
  const [editSessionMode, setEditSessionMode] = useState<'off' | 'task' | 'agent'>('off')
  const [editName, setEditName] = useState('')
  const [editInstructions, setEditInstructions] = useState('')
  const [editRole, setEditRole] = useState('')
  const [editCaps, setEditCaps] = useState('')
  const [editEnv, setEditEnv] = useState('')
  const [editBackend, setEditBackend] = useState('claude')
  const [editMachineId, setEditMachineId] = useState('')
  const [editModel, setEditModel] = useState('')
  const [editCustomModel, setEditCustomModel] = useState('')
  const [editSkills, setEditSkills] = useState<string[]>([])
  const [editAskTools, setEditAskTools] = useState<string[]>([])

  const create = useMutation({
    mutationFn: () =>
      api.createAgent({
        name,
        instructions,
        backend,
        model: model === '__custom' ? customModel : model,
        workspaceId: createWorkspaceIds[0] ?? workspaceId,
        workspaceIds: createWorkspaceIds,
        machineId: machineId || undefined,
        skills,
        tools: askTools.length > 0 ? { approval: { ask: askTools } } : undefined,
        repos: repos ? repos.split(',').map((s) => s.trim()).filter(Boolean) : [],
        sessionMode,
      }),
    onSuccess: () => {
      setOpen(false)
      setName('')
      setInstructions('')
      setModel('')
      setCustomModel('')
      setSkills([])
      setAskTools([])
      setRepos('')
      setSessionMode('off')
      qc.invalidateQueries({ queryKey: ['agents'] })
    },
  })
  const update = useMutation({
    mutationFn: async () => {
      if (!editFor) throw new Error('no agent')
      await api.updateAgent(editFor.id, {
        name: editName,
        instructions: editInstructions,
        role: editRole,
        capabilities: editCaps ? editCaps.split(',').map((s) => s.trim()).filter(Boolean) : [],
        backend: editBackend,
        model: editModel === '__custom' ? editCustomModel : editModel,
        machineId: editMachineId,
        skills: editSkills,
        tools: { ...(editFor.tools ?? {}), approval: { ...((editFor.tools?.approval as Record<string, unknown>) ?? {}), ask: editAskTools } },
        repos: editFor.repos ?? [],
        sessionMode: editSessionMode,
        workspaceId: editWorkspaceIds[0] ?? editFor.workspaceId,
        workspaceIds: editWorkspaceIds,
      })
      const vars: Record<string, string> = {}
      editEnv.split('\\n').forEach((line) => {
        const i = line.indexOf('=')
        if (i > 0) vars[line.slice(0, i).trim()] = line.slice(i + 1)
      })
      if (Object.keys(vars).length > 0) await api.putAgentEnv(editFor.id, vars)
    },
    onSuccess: () => {
      setEditFor(null)
      qc.invalidateQueries({ queryKey: ['agents'] })
    },
  })
  const remove = useMutation({
    mutationFn: (id: string) => api.deleteAgent(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['agents'] }),
  })

  return (
    <div className="mx-auto max-w-[1180px] px-6 py-5">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-[18px] font-semibold tracking-tight">{t('page.agents')}</h1>
          <p className="mt-0.5 text-[12px] text-muted">{t('agents.subtitle')}</p>
        </div>
        <Dialog
          open={open}
          onOpenChange={(value) => {
            setOpen(value)
            if (value) setCreateWorkspaceIds([workspaceId])
          }}
        >
          <DialogTrigger asChild>
            <Button variant="primary">
              <Plus className="h-3.5 w-3.5" /> {t('agents.new')}
            </Button>
          </DialogTrigger>
          <DialogContent title={t('agents.new')} description={t('agents.newHint')} className="max-h-[85vh] overflow-y-auto">
            <form
              className="flex flex-col gap-3"
              onSubmit={(e) => {
                e.preventDefault()
                create.mutate()
              }}
            >
              <label className="text-[12px] text-muted">
                {t('common.name')}
                <Input className="mt-1" value={name} onChange={(e) => setName(e.target.value)} />
              </label>
              <div className="text-[12px] text-muted">
                {t('agents.workspaces')}
                <WorkspacePicker value={createWorkspaceIds} onChange={setCreateWorkspaceIds} />
                <p className="mt-1 text-[10.5px] text-muted/60">{t('agents.workspacesHint')}</p>
              </div>
              <label className="text-[12px] text-muted">
                {t('agents.backend')}
                <BackendHostSelect
                  machines={machines}
                  backend={backend}
                  machineId={machineId}
                  registryIds={registryIds}
                  onChange={(nextBackend, nextMachine) => {
                    setBackend(nextBackend)
                    setMachineId(nextMachine)
                    setModel('')
                    setCustomModel('')
                  }}
                />
                <p className="mt-1 text-[10.5px] text-muted/60">{t('agents.backendHint')}</p>
              </label>
              <label className="text-[12px] text-muted">
                {t('agents.model')}
                <ModelSelect
                  backend={backend}
                  value={model}
                  customValue={customModel}
                  onChange={setModel}
                  onCustomChange={setCustomModel}
                />
              </label>
              <label className="text-[12px] text-muted">
                {t('agents.instructions')}
                <textarea
                  className="mt-1 min-h-24 w-full rounded-md border border-line2 bg-panel px-2 py-1 text-[12px] leading-5"
                  value={instructions}
                  onChange={(e) => setInstructions(e.target.value)}
                  placeholder={t('agents.instructionsPlaceholder')}
                />
                <p className="mt-1 text-[10.5px] text-muted/60">{t('agents.instructionsHint')}</p>
              </label>
              <div className="text-[12px] text-muted">
                {t('agents.skills')}
                <SkillsPicker value={skills} onChange={setSkills} />
                <p className="mt-1 text-[10.5px] text-muted/60">{t('agents.skillsHint')}</p>
              </div>
              <div className="text-[12px] text-muted">
                Approval policy
                <ApprovalPolicy value={askTools} onChange={setAskTools} />
                <p className="mt-1 text-[10.5px] text-muted/60">Tools listed here pause the run until you approve or deny the call.</p>
              </div>
              <label className="text-[12px] text-muted">
                {t('agents.reposCommaSeparated')}
                <Input className="mt-1" value={repos} onChange={(e) => setRepos(e.target.value)} />
              </label>
              <label className="text-[12px] text-muted">
                Context
                <select
                  className="mt-1 w-full rounded-md border border-line2 bg-panel px-2 py-1.5 text-[12px] text-fg"
                  value={sessionMode}
                  onChange={(e) => setSessionMode(e.target.value as 'off' | 'task' | 'agent')}
                >
                  <option value="off">Off — fresh context per run</option>
                  <option value="task">Per task — reuse context for the same task</option>
                  <option value="agent">Agent — reuse one context across tasks</option>
                </select>
                <p className="mt-1 text-[10.5px] text-muted/60">Reuse an agent session so later runs remember earlier work.</p>
              </label>
              {create.error && <p className="text-[12px] text-err">{String(create.error.message)}</p>}
              <div className="mt-1 flex justify-end gap-2">
                <Button type="button" variant="ghost" onClick={() => setOpen(false)}>
                  {t('common.cancel')}
                </Button>
                <Button type="submit" variant="primary" disabled={!name || createWorkspaceIds.length === 0 || create.isPending}>
                  {t('common.create')}
                </Button>
              </div>
            </form>
          </DialogContent>
        </Dialog>
      </div>

      {machines.length === 0 && (
        <div className="mt-3 flex items-start gap-2 rounded-md border border-warn/30 bg-warn/10 px-3 py-2 text-[11.5px] text-warn">
          <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
          {t('agents.noMachine')}
        </div>
      )}
      {machines.length > 0 && missing.length > 0 && (
        <div className="mt-3 flex items-start gap-2 rounded-md border border-warn/30 bg-warn/10 px-3 py-2 text-[11.5px] text-warn">
          <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
          {t('agents.missingRuntime', { backends: missing.join(', ') })}
        </div>
      )}

      {error && <p className="mt-4 text-[12px] text-err">{String(error.message)}</p>}
      {isLoading && <p className="mt-6 text-[12px] text-muted">{t('common.loading')}</p>}

      <div className="mt-4 grid grid-cols-2 gap-3">
        {items.map((a) => (
          <div key={a.id} className="rounded-lg border border-line bg-panel/40 p-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Bot className="h-3.5 w-3.5 text-accent2" />
                <span className="text-[13px] font-medium">{a.name}</span>
              </div>
              <div className="flex items-center gap-2">
                <button
                  className="text-[10px] text-muted hover:text-fg"
                  onClick={() => {
                    setEditFor(a)
                    setEditSessionMode((a.sessionMode as 'off' | 'task' | 'agent') ?? 'off')
                    setEditName(a.name)
                    setEditWorkspaceIds(a.workspaceIds?.length ? a.workspaceIds : [a.workspaceId])
                    setEditInstructions(a.instructions ?? '')
                    setEditRole(a.role ?? '')
                    setEditCaps((a.capabilities ?? []).join(','))
                    setEditEnv('')
                    setEditBackend(a.backend)
                    setEditMachineId(a.machineId ?? '')
                    setEditModel(
                      a.model && MODEL_OPTIONS[a.backend]?.includes(a.model) ? a.model : a.model ? '__custom' : '',
                    )
                    setEditCustomModel(
                      a.model && MODEL_OPTIONS[a.backend]?.includes(a.model) ? '' : a.model ?? '',
                    )
                    setEditSkills(a.skills ?? [])
                    setEditAskTools(((a.tools?.approval as { ask?: string[] } | undefined)?.ask) ?? [])
                    api.agentEnv(a.id).then((r) => setEditEnv(r.keys.map((k) => k + '=').join('\\n'))).catch(() => {})
                  }}
                >
                  {t('common.edit')}
                </button>
                <button
                  className="text-muted hover:text-err"
                  onClick={() => remove.mutate(a.id)}
                  title={t('common.delete')}
                >
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
            {a.machineId && (
              <p className="mt-1 text-[10.5px] text-muted">
                {t('agents.machine')}:{' '}
                {machines.find((machine) => machine.id === a.machineId)?.name ?? a.machineId}
              </p>
            )}
            <p className="mt-1 text-[10.5px] text-muted">
              {t('agents.workspaces')}:{' '}
              {(a.workspaceIds?.length ? a.workspaceIds : [a.workspaceId])
                .map((id) => workspaces.find((workspace) => workspace.id === id)?.name ?? id)
                .join(', ')}
            </p>
            <div className="mt-2 flex flex-wrap gap-1 text-[10px]">
              <span className="inline-flex items-center gap-1 rounded border border-line2 px-1.5 py-0.5 text-muted">
                <AgentIcon registryId={registryIds.get(a.backend)} name={a.backend} className="h-3 w-3" />
                {a.backend}
                {a.model ? ` · ${a.model}` : ''}
              </span>
              {a.role && (
                <span className="rounded bg-panel2 px-1.5 py-0.5 text-muted">{a.role}</span>
              )}
              {(a.capabilities ?? []).map((c) => (
                <span key={c} className="rounded bg-panel2 px-1.5 py-0.5 text-muted">{c}</span>
              ))}
              {(a.skills ?? []).map((s) => (
                <span key={s} className="rounded bg-accent/10 px-1.5 py-0.5 text-accent2">
                  {s}
                </span>
              ))}
            </div>
            {a.repos && a.repos.length > 0 && (
              <p className="mt-2 truncate text-[11px] text-muted">{a.repos.join(', ')}</p>
            )}
          </div>
        ))}
      </div>
      {!isLoading && items.length === 0 && (
        <p className="mt-8 text-[12px] text-muted">{t('agents.empty')}</p>
      )}

      <Dialog open={!!editFor} onOpenChange={(open) => !open && setEditFor(null)}>
        <DialogContent title={editFor?.name ?? ''} description={t('agents.editHint')} className="max-h-[85vh] overflow-y-auto">
          <form
            className="flex flex-col gap-3"
            onSubmit={(e) => {
              e.preventDefault()
              update.mutate()
            }}
          >
            <label className="text-[12px] text-muted">
              {t('common.name')}
              <Input className="mt-1" value={editName} onChange={(e) => setEditName(e.target.value)} />
            </label>
            <div className="text-[12px] text-muted">
              {t('agents.workspaces')}
              <WorkspacePicker value={editWorkspaceIds} onChange={setEditWorkspaceIds} />
              <p className="mt-1 text-[10.5px] text-muted/60">{t('agents.workspacesHint')}</p>
            </div>
            <label className="text-[12px] text-muted">
              {t('agents.backend')}
              <BackendHostSelect
                machines={machines}
                backend={editBackend}
                machineId={editMachineId}
                registryIds={registryIds}
                onChange={(nextBackend, nextMachine) => {
                  setEditBackend(nextBackend)
                  setEditMachineId(nextMachine)
                  setEditModel('')
                  setEditCustomModel('')
                }}
              />
              <p className="mt-1 text-[10.5px] text-muted/60">{t('agents.backendHint')}</p>
            </label>
            <label className="text-[12px] text-muted">
              {t('agents.model')}
              <ModelSelect
                backend={editBackend}
                value={editModel}
                customValue={editCustomModel}
                onChange={setEditModel}
                onCustomChange={setEditCustomModel}
              />
            </label>
            <label className="text-[12px] text-muted">
              {t('agents.instructions')}
              <textarea
                className="mt-1 min-h-24 w-full rounded-md border border-line2 bg-panel px-2 py-1 text-[12px] leading-5"
                value={editInstructions}
                onChange={(e) => setEditInstructions(e.target.value)}
                placeholder={t('agents.instructionsPlaceholder')}
              />
              <p className="mt-1 text-[10.5px] text-muted/60">{t('agents.instructionsHint')}</p>
            </label>
            <div className="text-[12px] text-muted">
              {t('agents.skills')}
              <SkillsPicker value={editSkills} onChange={setEditSkills} />
              <p className="mt-1 text-[10.5px] text-muted/60">{t('agents.skillsHint')}</p>
            </div>
            <div className="text-[12px] text-muted">
              Approval policy
              <ApprovalPolicy value={editAskTools} onChange={setEditAskTools} />
              <p className="mt-1 text-[10.5px] text-muted/60">Tools listed here pause the run until you approve or deny the call.</p>
            </div>
            <label className="text-[12px] text-muted">
              {t('agents.role')}
              <Input className="mt-1" value={editRole} onChange={(e) => setEditRole(e.target.value)} />
            </label>
            <label className="text-[12px] text-muted">
              {t('agents.capabilities')}
              <Input
                className="mt-1"
                value={editCaps}
                onChange={(e) => setEditCaps(e.target.value)}
                placeholder="code, review, deploy"
              />
            </label>
            <label className="text-[12px] text-muted">
              Context
              <select
                className="mt-1 w-full rounded-md border border-line2 bg-panel px-2 py-1.5 text-[12px] text-fg"
                value={editSessionMode}
                onChange={(e) => setEditSessionMode(e.target.value as 'off' | 'task' | 'agent')}
              >
                <option value="off">Off — fresh context per run</option>
                <option value="task">Per task — reuse context for the same task</option>
                <option value="agent">Agent — reuse one context across tasks</option>
              </select>
            </label>
            <label className="text-[12px] text-muted">
              {t('agents.env')}
              <textarea
                className="mt-1 min-h-24 w-full rounded-md border border-line2 bg-panel px-2 py-1 font-mono text-[11px]"
                value={editEnv}
                onChange={(e) => setEditEnv(e.target.value)}
                placeholder={'SERVICE_API_KEY=...\nANOTHER_KEY=...'}
              />
            </label>
            {update.error && <p className="text-[12px] text-err">{String(update.error.message)}</p>}
            <div className="flex justify-end gap-2">
              <Button type="button" variant="ghost" onClick={() => setEditFor(null)}>
                {t('common.cancel')}
              </Button>
              <Button type="submit" variant="primary" disabled={update.isPending}>
                {t('common.save')}
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
function WorkspacePicker({
  value,
  onChange,
}: {
  value: string[]
  onChange: (value: string[]) => void
}) {
  const { workspaces } = useWorkspace()
  return (
    <div className="mt-1 flex max-h-28 flex-wrap gap-1 overflow-y-auto rounded-md border border-line2 bg-panel p-2">
      {workspaces.map((workspace) => {
        const checked = value.includes(workspace.id)
        return (
          <label
            key={workspace.id}
            className="flex cursor-pointer items-center gap-1.5 rounded px-1.5 py-1 text-[11.5px] text-muted hover:text-fg"
          >
            <input
              type="checkbox"
              checked={checked}
              onChange={() =>
                onChange(
                  checked ? value.filter((id) => id !== workspace.id) : [...value, workspace.id],
                )
              }
              className="accent-fg"
            />
            {workspace.name}
          </label>
        )
      })}
    </div>
  )
}

function BackendHostSelect({
  machines,
  backend,
  machineId,
  registryIds,
  onChange,
}: {
  machines: Machine[]
  backend: string
  machineId: string
  registryIds: Map<string, string>
  onChange: (backend: string, machineId: string) => void
}) {
  const { t } = useI18n()
  const [open, setOpen] = useState(false)
  const rootRef = useRef<HTMLDivElement>(null)
  useEffect(() => {
    if (!open) return
    const close = (event: MouseEvent) => {
      if (!rootRef.current?.contains(event.target as Node)) setOpen(false)
    }
    // Escape is left to the surrounding dialog, which closes on it.
    document.addEventListener('mousedown', close)
    return () => document.removeEventListener('mousedown', close)
  }, [open])

  const groups = machines
    .filter((machine) => machine.status === 'online' || machine.id === machineId)
    .map((machine) => ({
      id: machine.id,
      label: isLocalHost(machine) ? t('acp.localHost', { name: machine.name }) : machine.name,
      backends: [...new Set([...machine.agents, ...Object.keys(machine.runtimes ?? {})])],
    }))
    .filter((group) => group.backends.length > 0)
  const any = [...new Set(groups.flatMap((group) => group.backends))].sort()
  const fallback = any.length > 0 ? any : ['claude', 'codex', 'opencode', 'deepseek', 'gemini', 'cursor', 'qwen']
  const sections = [{ id: '', label: t('agents.anyCompatibleHost'), backends: fallback }, ...groups]
  const selectedHost = groups.find((group) => group.id === machineId)
  const icon = (value: string) => <AgentIcon registryId={registryIds.get(value)} name={value} className="h-3.5 w-3.5" />

  return (
    <div ref={rootRef} className="relative mt-1">
      <button
        type="button"
        aria-haspopup="listbox"
        aria-expanded={open}
        onClick={() => setOpen((value) => !value)}
        className="flex h-8 w-full items-center gap-2 rounded-md border border-line2 bg-panel px-2 text-left text-[13px] text-fg"
      >
        {icon(backend)}
        <span className="truncate">{backend}</span>
        <span className="truncate text-[11.5px] text-muted">· {selectedHost?.label ?? t('agents.anyHost')}</span>
        <ChevronDown className="ml-auto h-3.5 w-3.5 shrink-0 text-muted" />
      </button>
      {open && (
        <div role="listbox" className="absolute z-50 mt-1 max-h-72 w-full overflow-y-auto rounded-md border border-line2 bg-panel py-1 shadow-lg">
          {sections.map((section) => (
            <div key={section.id || 'any'}>
              <p className="px-2 pb-0.5 pt-1.5 text-[10px] uppercase tracking-wider text-muted/70">{section.label}</p>
              {section.backends.map((value) => {
                const selected = value === backend && section.id === (selectedHost ? machineId : '')
                return (
                  <button
                    key={`${section.id}::${value}`}
                    type="button"
                    role="option"
                    aria-selected={selected}
                    onClick={() => {
                      onChange(value, section.id)
                      setOpen(false)
                    }}
                    className={`flex w-full items-center gap-2 px-2 py-1.5 text-left text-[12.5px] hover:bg-panel2 ${selected ? 'bg-panel2 text-fg' : 'text-fg/90'}`}
                  >
                    {icon(value)}
                    {value}
                    {selected && <Check className="ml-auto h-3.5 w-3.5 text-accent2" />}
                  </button>
                )
              })}
            </div>
          ))}
        </div>
      )}
    </div>
  )
}

function ModelSelect({
  backend,
  value,
  customValue,
  onChange,
  onCustomChange,
}: {
  backend: string
  value: string
  customValue: string
  onChange: (value: string) => void
  onCustomChange: (value: string) => void
}) {
  const { t } = useI18n()
  const options = MODEL_OPTIONS[backend] ?? []
  return (
    <>
      <select
        className="mt-1 h-8 w-full rounded-md border border-line2 bg-panel px-2 text-[13px]"
        value={value}
        onChange={(event) => onChange(event.target.value)}
      >
        <option value="">{t('agents.defaultModel')} · {backend}</option>
        {options.map((option) => (
          <option key={option} value={option}>
            {option}
          </option>
        ))}
        <option value="__custom">{t('agents.customModel')}</option>
      </select>
      {value === '__custom' && (
        <Input
          className="mt-1"
          value={customValue}
          onChange={(event) => onCustomChange(event.target.value)}
          placeholder={t('agents.modelId')}
        />
      )}
    </>
  )
}

const APPROVAL_PRESETS = ['Bash', 'Edit', 'Write', 'Read', 'WebFetch', 'MCP']

function ApprovalPolicy({ value, onChange }: { value: string[]; onChange: (value: string[]) => void }) {
  const [draft, setDraft] = useState('')
  const toggle = (tool: string) => {
    onChange(value.includes(tool) ? value.filter((item) => item !== tool) : [...value, tool])
  }
  const add = () => {
    const tool = draft.trim()
    if (tool && !value.includes(tool)) onChange([...value, tool])
    setDraft('')
  }
  return (
    <div className="mt-1 flex flex-col gap-2">
      <div className="flex flex-wrap gap-1">
        {APPROVAL_PRESETS.map((tool) => (
          <button
            key={tool}
            type="button"
            onClick={() => toggle(tool)}
            className={
              value.includes(tool)
                ? 'rounded border border-warn/40 bg-warn/10 px-1.5 py-0.5 text-[10.5px] text-warn'
                : 'rounded border border-line px-1.5 py-0.5 text-[10.5px] text-muted hover:text-fg'
            }
          >
            {tool}
          </button>
        ))}
      </div>
      <div className="flex gap-1">
        <Input
          value={draft}
          onChange={(event) => setDraft(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter') {
              event.preventDefault()
              add()
            }
          }}
          placeholder="Custom tool name"
          className="h-7 text-[11px]"
        />
        <Button type="button" variant="secondary" size="sm" onClick={add}>Add</Button>
      </div>
      {value.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {value.map((tool) => (
            <span key={tool} className="flex items-center gap-1 rounded bg-panel2 px-1.5 py-0.5 text-[10.5px] text-muted">
              {tool}
              <button type="button" onClick={() => toggle(tool)} className="hover:text-fg">×</button>
            </span>
          ))}
        </div>
      )}
    </div>
  )
}

function SkillsPicker({ value, onChange }: { value: string[]; onChange: (value: string[]) => void }) {
  const { t } = useI18n()
  const { workspaceId } = useWorkspace()
  const [query, setQuery] = useState('')
  const { data } = useQuery({ queryKey: ['skills', workspaceId], queryFn: () => api.skills(workspaceId) })
  const items = data?.items ?? []
  const names = new Set(items.map((skill) => skill.name))
  const extras = value.filter((name) => !names.has(name))
  const all = [...items.map((skill) => skill.name), ...extras]
  const visible = all.filter((name) => name.toLowerCase().includes(query.trim().toLowerCase()))

  if (all.length === 0) {
    return (
      <p className="mt-1 rounded-md border border-line2 bg-panel px-2 py-2 text-[11px] text-muted">
        {t('agents.noSkills')}
      </p>
    )
  }

  return (
    <div className="mt-1 rounded-md border border-line2 bg-panel p-2">
      <Input
        className="mb-1 h-7"
        value={query}
        onChange={(event) => setQuery(event.target.value)}
        placeholder={t('agents.searchSkills')}
      />
      <div className="flex max-h-24 flex-wrap gap-1 overflow-y-auto">
        {visible.map((name) => {
          const checked = value.includes(name)
          return (
            <label
              key={name}
              className="flex cursor-pointer items-center gap-1.5 rounded px-1.5 py-1 font-mono text-[11px] text-muted hover:text-fg"
            >
              <input
                type="checkbox"
                checked={checked}
                onChange={() => onChange(checked ? value.filter((item) => item !== name) : [...value, name])}
                className="accent-fg"
              />
              {name}
            </label>
          )
        })}
        {visible.length === 0 && <p className="text-[11px] text-muted">{t('skills.noResults')}</p>}
      </div>
    </div>
  )
}