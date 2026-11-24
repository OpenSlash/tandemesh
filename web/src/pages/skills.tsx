import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Download, Plus, Search, Sparkles, Store, Trash2, UserPlus } from 'lucide-react'
import { api, type MarketSkill, type Skill } from '../lib/api'
import { useWorkspace } from '../lib/workspace'
import { useI18n } from '../lib/i18n'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { Dialog, DialogContent, DialogTrigger } from '../components/ui/dialog'

export default function SkillsPage() {
  const { t } = useI18n()
  const { workspaceId } = useWorkspace()
  const qc = useQueryClient()
  const { data, isLoading } = useQuery({
    queryKey: ['skills', workspaceId],
    queryFn: () => api.skills(workspaceId),
  })
  const items = data?.items ?? []
  const [query, setQuery] = useState('')
  const [attachSkill, setAttachSkill] = useState<Skill | null>(null)
  const [attachIds, setAttachIds] = useState<string[]>([])
  const { data: agentsData } = useQuery({
    queryKey: ['agents', 'all'],
    queryFn: () => api.agents('all'),
  })
  const agents = (agentsData?.items ?? []).filter((agent) =>
    (agent.workspaceIds?.length ? agent.workspaceIds : [agent.workspaceId]).includes(workspaceId),
  )
  const [tab, setTab] = useState<'library' | 'market'>('library')
  const [marketQuery, setMarketQuery] = useState('')
  const [marketSkill, setMarketSkill] = useState<MarketSkill | null>(null)
  const [marketAgentIds, setMarketAgentIds] = useState<string[]>([])
  const { data: marketData } = useQuery({
    queryKey: ['skill-market', workspaceId],
    queryFn: () => api.skillMarket(workspaceId),
    enabled: tab === 'market',
  })
  const marketItems = (marketData?.items ?? []).filter((skill) => {
    const text = `${skill.name} ${skill.description} ${skill.author} ${(skill.tags ?? []).join(' ')}`.toLowerCase()
    return text.includes(marketQuery.trim().toLowerCase())
  })
  const filtered = items.filter((skill) => {
    const text = `${skill.name} ${skill.description} ${(skill.tags ?? []).join(' ')}`.toLowerCase()
    return text.includes(query.trim().toLowerCase())
  })

  const [open, setOpen] = useState(false)
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [version, setVersion] = useState('')
  const [tags, setTags] = useState('')
  const [body, setBody] = useState('')

  const [installOpen, setInstallOpen] = useState(false)
  const [sourceUrl, setSourceUrl] = useState('')

  const refresh = () => qc.invalidateQueries({ queryKey: ['skills', workspaceId] })
  const create = useMutation({
    mutationFn: () =>
      api.createSkill({
        workspaceId, name, description, body, version: version || undefined,
        source: 'manual', tags: tags ? tags.split(',').map((x) => x.trim()).filter(Boolean) : undefined,
      }),
    onSuccess: () => {
      setOpen(false)
      setName('')
      setDescription('')
      setVersion('')
      setTags('')
      setBody('')
      refresh()
    },
  })
  const install = useMutation({
    mutationFn: () => api.installSkill({ workspaceId, sourceUrl }),
    onSuccess: () => {
      setInstallOpen(false)
      setSourceUrl('')
      refresh()
    },
  })
  const remove = useMutation({
    mutationFn: (id: string) => api.deleteSkill(id),
    onSuccess: refresh,
  })
  const toggle = useMutation({
    mutationFn: (args: { id: string; enabled: boolean }) => api.setSkillEnabled(args.id, args.enabled),
    onSuccess: refresh,
  })
  const attach = useMutation({
    mutationFn: async () => {
      if (!attachSkill) return
      await Promise.all(
        agents.map(async (agent) => {
          const current = agent.skills ?? []
          const has = current.includes(attachSkill.name)
          const should = attachIds.includes(agent.id)
          if (has === should) return
          const next = should ? [...current, attachSkill.name] : current.filter((name) => name !== attachSkill.name)
          await api.setAgentSkills(agent.id, next)
        }),
      )
    },
    onSuccess: () => {
      setAttachSkill(null)
      qc.invalidateQueries({ queryKey: ['agents'] })
    },
  })
  const installMarket = useMutation({
    mutationFn: async () => {
      if (!marketSkill) return
      await api.installMarketSkill(marketSkill.id, { workspaceId, agentIds: marketAgentIds })
    },
    onSuccess: () => {
      setMarketSkill(null)
      setMarketAgentIds([])
      refresh()
      qc.invalidateQueries({ queryKey: ['skill-market', workspaceId] })
      qc.invalidateQueries({ queryKey: ['agents'] })
    },
  })

  return (
    <div className="mx-auto max-w-[980px] px-6 py-5">
      <div className="flex items-start justify-between gap-2">
        <div>
          <h1 className="text-[18px] font-semibold tracking-tight">{t('page.skills')}</h1>
          <p className="mt-0.5 text-[12px] text-muted">{t('skills.subtitle')}</p>
        </div>
        <div className="flex gap-2">
          <Dialog open={installOpen} onOpenChange={setInstallOpen}>
            <DialogTrigger asChild>
              <Button variant="ghost" size="sm">
                <Download className="h-3.5 w-3.5" /> {t('skills.install')}
              </Button>
            </DialogTrigger>
            <DialogContent title={t('skills.install')} description={t('skills.installHint')}>
              <form
                className="flex flex-col gap-3"
                onSubmit={(e) => {
                  e.preventDefault()
                  install.mutate()
                }}
              >
                <Input autoFocus value={sourceUrl} onChange={(e) => setSourceUrl(e.target.value)} placeholder="https://example.com/SKILL.md" />
                {install.error && <p className="text-[12px] text-err">{String(install.error.message)}</p>}
                <div className="flex justify-end gap-2">
                  <Button type="button" variant="ghost" onClick={() => setInstallOpen(false)}>{t('common.cancel')}</Button>
                  <Button type="submit" variant="primary" disabled={!sourceUrl || install.isPending}>{t('skills.install')}</Button>
                </div>
              </form>
            </DialogContent>
          </Dialog>
          <Dialog open={open} onOpenChange={setOpen}>
            <DialogTrigger asChild>
              <Button variant="primary" size="sm">
                <Plus className="h-3.5 w-3.5" /> {t('common.create')}
              </Button>
            </DialogTrigger>
            <DialogContent title={t('skills.new')} description={t('skills.newHint')}>
              <form
                className="flex flex-col gap-3"
                onSubmit={(e) => {
                  e.preventDefault()
                  create.mutate()
                }}
              >
                <Input autoFocus value={name} onChange={(e) => setName(e.target.value)} placeholder={t('skills.name')} />
                <Input value={description} onChange={(e) => setDescription(e.target.value)} placeholder={t('skills.description')} />
                <div className="flex gap-2">
                  <Input value={version} onChange={(e) => setVersion(e.target.value)} placeholder={t('skills.version')} />
                  <Input value={tags} onChange={(e) => setTags(e.target.value)} placeholder={t('skills.tags')} />
                </div>
                <textarea
                  value={body}
                  onChange={(e) => setBody(e.target.value)}
                  rows={8}
                  placeholder={t('skills.body')}
                  className="w-full rounded-md border border-line2 bg-panel px-2 py-1.5 font-mono text-[12px] text-fg placeholder:text-muted/60"
                />
                {create.error && <p className="text-[12px] text-err">{String(create.error.message)}</p>}
                <div className="flex justify-end gap-2">
                  <Button type="button" variant="ghost" onClick={() => setOpen(false)}>{t('common.cancel')}</Button>
                  <Button type="submit" variant="primary" disabled={!name || create.isPending}>{t('common.create')}</Button>
                </div>
              </form>
            </DialogContent>
          </Dialog>
        </div>
      </div>

      <div className="mt-3 flex gap-1 border-b border-line">
        {(['library', 'market'] as const).map((value) => (
          <button
            key={value}
            onClick={() => setTab(value)}
            className={
              tab === value
                ? 'border-b-2 border-fg px-2 py-1.5 text-[12px] font-medium text-fg'
                : 'px-2 py-1.5 text-[12px] text-muted hover:text-fg'
            }
          >
            {value === 'library' ? t('skills.tabLibrary') : t('skills.tabMarket')}
          </button>
        ))}
      </div>

      <div className="relative mt-3">
        <Search className="pointer-events-none absolute left-2.5 top-2 h-3.5 w-3.5 text-muted" />
        <Input
          className="pl-8"
          value={tab === 'library' ? query : marketQuery}
          onChange={(event) =>
            tab === 'library' ? setQuery(event.target.value) : setMarketQuery(event.target.value)
          }
          placeholder={tab === 'library' ? t('skills.searchPlaceholder') : t('skills.marketSearch')}
        />
      </div>

      <Dialog open={!!attachSkill} onOpenChange={(open) => !open && setAttachSkill(null)}>
        <DialogContent title={t('skills.attachTitle')} description={t('skills.attachHint')}>
          <div className="flex flex-col gap-3">
            <p className="font-mono text-[12px] text-fg">{attachSkill?.name}</p>
            {agents.length === 0 && <p className="text-[12px] text-muted">{t('skills.noAgents')}</p>}
            <div className="flex max-h-64 flex-col gap-1 overflow-y-auto">
              {agents.map((agent) => {
                const checked = attachIds.includes(agent.id)
                return (
                  <label
                    key={agent.id}
                    className="flex cursor-pointer items-center gap-2 rounded px-1.5 py-1 text-[12px] text-muted hover:bg-panel2 hover:text-fg"
                  >
                    <input
                      type="checkbox"
                      checked={checked}
                      onChange={() =>
                        setAttachIds(checked ? attachIds.filter((id) => id !== agent.id) : [...attachIds, agent.id])
                      }
                      className="accent-fg"
                    />
                    {agent.name}
                    <span className="ml-auto font-mono text-[10px] text-muted/60">{agent.backend}</span>
                  </label>
                )
              })}
            </div>
            {attach.error && <p className="text-[12px] text-err">{String(attach.error.message)}</p>}
            <div className="flex justify-end gap-2">
              <Button type="button" variant="ghost" onClick={() => setAttachSkill(null)}>
                {t('common.cancel')}
              </Button>
              <Button type="button" variant="primary" disabled={attach.isPending} onClick={() => attach.mutate()}>
                {t('skills.attachSave')}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      {isLoading && <p className="mt-4 text-[12px] text-muted">{t('common.loading')}</p>}

      {tab === 'library' && (
        <div className="mt-4 space-y-2">
          {filtered.map((skill) => (
            <div key={skill.id} className="rounded-lg border border-line bg-panel/40 p-3">
              <div className="flex items-start gap-2">
                <Sparkles className="mt-0.5 h-4 w-4 text-accent2" />
                <div className="min-w-0 flex-1">
                  <div className="flex flex-wrap items-center gap-2">
                    <span className="font-mono text-[12.5px] font-medium">{skill.name}</span>
                    {skill.version && <span className="rounded bg-panel px-1 text-[10px] text-muted">v{skill.version}</span>}
                    <span className="text-[10px] text-muted">{skill.source}</span>
                    {(skill.tags ?? []).map((tag) => (
                      <span key={tag} className="rounded bg-panel px-1 text-[10px] text-muted">#{tag}</span>
                    ))}
                    <span className={skill.enabled ? 'text-[10px] text-ok' : 'text-[10px] text-muted'}>
                      {skill.enabled ? t('skills.enabled') : t('skills.disabled')}
                    </span>
                    <span className="text-[10px] text-muted">{skill.description}</span>
                  </div>
                  <pre className="mt-1 whitespace-pre-wrap break-words text-[11.5px] text-muted/90">{skill.body}</pre>
                </div>
                <button
                  title={t('skills.addToAgent')}
                  onClick={() => {
                    setAttachSkill(skill)
                    setAttachIds(agents.filter((agent) => (agent.skills ?? []).includes(skill.name)).map((agent) => agent.id))
                  }}
                  className="text-muted hover:text-fg"
                >
                  <UserPlus className="h-3.5 w-3.5" />
                </button>
                <button
                  onClick={() => toggle.mutate({ id: skill.id, enabled: !skill.enabled })}
                  className="text-[10px] text-muted hover:text-fg"
                >
                  {skill.enabled ? t('skills.disable') : t('skills.enable')}
                </button>
                <button title={t('common.delete')} onClick={() => remove.mutate(skill.id)} className="text-muted hover:text-err">
                  <Trash2 className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>
          ))}
          {!isLoading && filtered.length === 0 && (
            <p className="rounded-lg border border-dashed border-line2 px-4 py-10 text-center text-[12px] text-muted">
              {items.length === 0 ? t('skills.empty') : t('skills.noResults')}
            </p>
          )}
        </div>
      )}

      {tab === 'market' && (
        <div className="mt-4 grid gap-2 md:grid-cols-2">
          {marketItems.map((skill) => (
            <div key={skill.id} className="rounded-lg border border-line bg-panel/40 p-3">
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <div className="flex items-center gap-2">
                    <Store className="h-3.5 w-3.5 text-muted" />
                    <span className="font-mono text-[12.5px] font-medium">{skill.name}</span>
                    <span className="text-[10px] text-muted">v{skill.version}</span>
                  </div>
                  <p className="mt-1 text-[11.5px] text-muted">{skill.description}</p>
                  <div className="mt-1 flex flex-wrap gap-1">
                    {(skill.tags ?? []).map((tag) => (
                      <span key={tag} className="rounded bg-panel2 px-1 text-[10px] text-muted">#{tag}</span>
                    ))}
                    <span className="text-[10px] text-muted">by {skill.author}</span>
                    {skill.source && <span className="rounded bg-panel2 px-1 text-[10px] text-muted">{skill.source}</span>}
                  </div>
                </div>
                {skill.installed ? (
                  <span className="shrink-0 rounded bg-ok/10 px-2 py-1 text-[10px] text-ok">{t('skills.installed')}</span>
                ) : (
                  <Button
                    size="sm"
                    variant="primary"
                    onClick={() => {
                      setMarketSkill(skill)
                      setMarketAgentIds([])
                    }}
                  >
                    {t('skills.marketInstall')}
                  </Button>
                )}
              </div>
            </div>
          ))}
          {marketItems.length === 0 && (
            <p className="rounded-lg border border-dashed border-line2 px-4 py-10 text-center text-[12px] text-muted md:col-span-2">
              {t('skills.marketEmpty')}
            </p>
          )}
        </div>
      )}

      <Dialog open={!!marketSkill} onOpenChange={(open) => !open && setMarketSkill(null)}>
        <DialogContent title={t('skills.marketInstallTitle')} description={t('skills.marketInstallHint')}>
          <div className="flex flex-col gap-3">
            <p className="font-mono text-[12px] text-fg">{marketSkill?.name}</p>
            <p className="text-[11.5px] text-muted">{marketSkill?.description}</p>
            <div>
              <p className="text-[11.5px] text-muted">{t('skills.marketAttachHint')}</p>
              <div className="mt-1 flex max-h-52 flex-col gap-1 overflow-y-auto rounded-md border border-line2 bg-panel p-2">
                {agents.map((agent) => {
                  const checked = marketAgentIds.includes(agent.id)
                  return (
                    <label key={agent.id} className="flex items-center gap-2 px-1 py-0.5 text-[12px] text-muted hover:text-fg">
                      <input
                        type="checkbox"
                        checked={checked}
                        onChange={() =>
                          setMarketAgentIds(checked ? marketAgentIds.filter((id) => id !== agent.id) : [...marketAgentIds, agent.id])
                        }
                        className="accent-fg"
                      />
                      {agent.name}
                    </label>
                  )
                })}
                {agents.length === 0 && <p className="text-[11px] text-muted">{t('skills.noAgents')}</p>}
              </div>
            </div>
            {installMarket.error && <p className="text-[12px] text-err">{String(installMarket.error.message)}</p>}
            <div className="flex justify-end gap-2">
              <Button type="button" variant="ghost" onClick={() => setMarketSkill(null)}>
                {t('common.cancel')}
              </Button>
              <Button type="button" variant="primary" disabled={installMarket.isPending} onClick={() => installMarket.mutate()}>
                {t('skills.marketInstall')}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>
    </div>
  )
}
