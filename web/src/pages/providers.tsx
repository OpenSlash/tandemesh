import { useI18n } from '../lib/i18n'
import { useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Boxes, MessageSquare, FileText, Mail, Bot, Plug, Copy, RefreshCw, Trash2, KeyRound } from 'lucide-react'
import { api, type Connection, type Provider } from '../lib/api'
import { useWorkspace } from '../lib/workspace'
import { Dialog, DialogContent } from '../components/ui/dialog'
import { Button } from '../components/ui/button'
import { Input } from '../components/ui/input'
import { cn } from '../lib/utils'
import { OAuthAppsDialog } from '../components/oauth-apps-dialog'
import { PlaneCatalogPicker } from '../components/plane-catalog-picker'

const GROUPS: { kind: Provider['kind']; labelKey: string; icon: typeof Boxes }[] = [
  { kind: 'task', labelKey: 'providers.groupTask', icon: Boxes },
  { kind: 'chat', labelKey: 'providers.groupChat', icon: MessageSquare },
  { kind: 'doc', labelKey: 'providers.groupDocs', icon: FileText },
  { kind: 'email', labelKey: 'providers.groupEmail', icon: Mail },
  { kind: 'agent', labelKey: 'providers.groupAgent', icon: Bot },
]

export default function ProvidersPage() {
  const { t } = useI18n()
  const { workspaceId } = useWorkspace()
  const qc = useQueryClient()
  const { data, isLoading, error } = useQuery({ queryKey: ['providers'], queryFn: api.providers })
  const { data: connectionsData } = useQuery({
    queryKey: ['connections', workspaceId],
    queryFn: () => api.connections(workspaceId),
  })
  const { data: capabilities } = useQuery({ queryKey: ['capabilities'], queryFn: api.capabilities })
  const showWebhook = capabilities?.webhook ?? false
  const { data: importsData } = useQuery({
    queryKey: ['imports', workspaceId],
    queryFn: () => api.imports(workspaceId),
    refetchInterval: 5000,
  })
  const { data: importRuntime } = useQuery({
    queryKey: ['import-runtime'],
    queryFn: () => api.importRuntime(),
    refetchInterval: 5000,
  })
  const { data: identitiesData } = useQuery({
    queryKey: ['identities', workspaceId],
    queryFn: () => api.identities(workspaceId),
  })
  const { data: memberIdentitiesData } = useQuery({
    queryKey: ['member-identities', workspaceId],
    queryFn: () => api.memberIdentities(workspaceId),
  })
  const { data: rulesData } = useQuery({
    queryKey: ['assignment-rules', workspaceId],
    queryFn: () => api.assignmentRules(workspaceId),
  })
  const { data: agentsData } = useQuery({
    queryKey: ['agents', workspaceId],
    queryFn: () => api.agents(workspaceId),
  })
  const { data: groupsData } = useQuery({
    queryKey: ['groups', workspaceId],
    queryFn: () => api.groups(workspaceId),
  })
  const { data: githubOAuth } = useQuery({
    queryKey: ['github-oauth'],
    queryFn: api.githubOAuthStatus,
    retry: false,
  })
  const { data: githubApp } = useQuery({
    queryKey: ['github-app'],
    queryFn: api.githubAppStatus,
    retry: false,
  })
  const { data: oauthStatus } = useQuery({
    queryKey: ['oauth-status'],
    queryFn: api.oauthStatus,
    retry: false,
  })
  const oauthProviders = oauthStatus?.providers ?? []
  const connections = connectionsData?.items ?? []
  const jiraConnection = connections.find((connection) => connection.provider === 'jira')
  const gitlabConnection = connections.find((connection) => connection.provider === 'gitlab')
  const imports = importsData?.items ?? []
  const identities = identitiesData?.items ?? []
  const memberIdentities = memberIdentitiesData?.items ?? []
  const rules = rulesData?.items ?? []
  const agents = agentsData?.items ?? []
  const groups = groupsData?.items ?? []
  const items = data?.items ?? []
  const [ruleName, setRuleName] = useState('')
  const [ruleProvider, setRuleProvider] = useState('')
  const [ruleInstance, setRuleInstance] = useState('')
  const [ruleLabel, setRuleLabel] = useState('')
  const [ruleAgent, setRuleAgent] = useState('')
  const [ruleGroup, setRuleGroup] = useState('')
  const [ruleAuto, setRuleAuto] = useState(false)
  const [results, setResults] = useState<Record<string, string>>({})
  const [connectFor, setConnectFor] = useState('')
  const [oauthOpen, setOauthOpen] = useState(false)
  const [authKind, setAuthKind] = useState('token')
  const [fields, setFields] = useState<Record<string, string>>({})
  const [token, setToken] = useState('')
  const [baseUrl, setBaseUrl] = useState('')
  const [instance, setInstance] = useState('')
  const [transition, setTransition] = useState('')
  const [syncDirection, setSyncDirection] = useState('read_write')
  const [webhookSecret, setWebhookSecret] = useState('')
  const [commentTemplate, setCommentTemplate] = useState('')
  const [states, setStates] = useState<{ id: string; name: string; group?: string }[]>([])
  const [connectionName, setConnectionName] = useState('')
  const [syncState, setSyncState] = useState<Record<string, string>>({})
  const [reauthFor, setReauthFor] = useState<Connection | null>(null)
  const [reauthToken, setReauthToken] = useState('')
  const [editFor, setEditFor] = useState<Connection | null>(null)
  const [editName, setEditName] = useState('')
  const [editBaseUrl, setEditBaseUrl] = useState('')
  const [editInstance, setEditInstance] = useState('')
  const [editEnabled, setEditEnabled] = useState(true)
  const [connectMsg, setConnectMsg] = useState('')
  const [bindingsFor, setBindingsFor] = useState<Connection | null>(null)
  const [bindingExternal, setBindingExternal] = useState('')
  const [bindingName, setBindingName] = useState('')
  const [bindingDirection, setBindingDirection] = useState<'read' | 'read_write'>('read_write')
  const [bindingBacklink, setBindingBacklink] = useState(false)
  const [bindingTrusted, setBindingTrusted] = useState(false)
  const [bindingAccountID, setBindingAccountID] = useState('')
  const [identityProvider, setIdentityProvider] = useState('github')
  const [identityExternal, setIdentityExternal] = useState('')
  const [identityLogin, setIdentityLogin] = useState('')
  const [identityMember, setIdentityMember] = useState('')
  const [linkMembers, setLinkMembers] = useState<Record<string, string>>({})
  const [memberProvider, setMemberProvider] = useState('github')
  const memberOAuthEnabled = memberProvider === 'github'
    ? Boolean(githubOAuth?.enabled)
    : ['gitlab', 'jira', 'linear'].includes(memberProvider) && oauthProviders.includes(memberProvider)
  const [memberID, setMemberID] = useState('')
  const [memberExternal, setMemberExternal] = useState('')
  const [memberLogin, setMemberLogin] = useState('')
  const [memberEmail, setMemberEmail] = useState('')
  const [memberToken, setMemberToken] = useState('')
  const [memberUsers, setMemberUsers] = useState<{ id: string; login: string; name?: string; email?: string }[]>([])
  const [memberUserQuery, setMemberUserQuery] = useState('')
  const [accountsFor, setAccountsFor] = useState<Connection | null>(null)
  const [accountToken, setAccountToken] = useState('')
  const [accountKind, setAccountKind] = useState('user')
  const [accountLogin, setAccountLogin] = useState('')
  const [accountEmail, setAccountEmail] = useState('')
  const activeProvider = items.find((provider) => provider.id === connectFor)
  const authOptions = activeProvider?.authOptions ?? []
  const selectedAuth = authOptions.find((option) => option.kind === authKind)
  const configFields = selectedAuth?.fields?.length
    ? selectedAuth.fields.filter((field) => showWebhook || field.key !== 'webhookSecret')
    : (activeProvider?.configFields ?? []).filter(
        (field) => (showWebhook || field.key !== 'webhookSecret') && !(authKind !== 'token' && authKind !== 'bot' && field.key === 'token'),
      )
  const fieldValue = (key: string) => {
    if (fields[key] !== undefined) return fields[key]
    if (key === 'token') return token
    if (key === 'baseUrl') return baseUrl
    if (key === 'instance') return instance
    if (key === 'webhookSecret') return webhookSecret
    return ''
  }
  const setFieldValue = (key: string, value: string) => {
    setFields((prev) => ({ ...prev, [key]: value }))
    if (key === 'token') setToken(value)
    if (key === 'baseUrl') setBaseUrl(value)
    if (key === 'instance') setInstance(value)
    if (key === 'webhookSecret') setWebhookSecret(value)
  }
  const configReady = configFields.length > 0
    ? configFields.every((field) => !field.required || fieldValue(field.key).trim() !== '')
    : Boolean(token)
  const probe = useMutation({
    mutationFn: async (id: string) => ({ id, result: await api.probeProvider(id) }),
    onSuccess: ({ id, result }) =>
      setResults((prev) => ({ ...prev, [id]: result.ok ? 'ok' : result.error || 'failed' })),
    onError: (err, id) => setResults((prev) => ({ ...prev, [id]: String(err) })),
  })
  const connect = useMutation({
    mutationFn: async () => {
      await api.connectProvider({
        provider: connectFor,
        token,
        baseUrl: baseUrl || undefined,
        instance: instance || undefined,
        name: connectionName || undefined,
        workspaceId,
        transition: transition || undefined,
        webhookSecret: webhookSecret || undefined,
        commentTemplate: commentTemplate || undefined,
        fields: { ...fields, syncDirection: syncDirection },
      })
      if (activeProvider?.kind !== 'task') return { synced: 0 }
      return api.workSync(connectFor, instance || undefined, workspaceId)
    },
    onSuccess: (result) => {
      setConnectMsg(activeProvider?.kind === 'task'
        ? t('providers.connected', { count: result.synced })
        : t('providers.connectedChat'))
      qc.invalidateQueries({ queryKey: ['connections', workspaceId] })
      setConnectFor('')
      setToken('')
      setWebhookSecret('')
      setConnectionName('')
    },
    onError: (err) => setConnectMsg(String(err)),
  })
  const loadStates = useMutation({
    mutationFn: () =>
      api.integrationStates({
        provider: connectFor,
        token,
        baseUrl: baseUrl || undefined,
        instance,
      }),
    onSuccess: (result) => setStates(result.items),
  })
  const removeConnection = useMutation({
    mutationFn: (id: string) => api.deleteConnection(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['connections', workspaceId] }),
  })
  const registerWebhook = useMutation({
    mutationFn: (connection: Connection) => api.registerWebhook(connection.id),
    onSuccess: (result, connection) =>
      setSyncState((prev) => ({ ...prev, [connection.id]: `webhook ${result.webhookId}` })),
    onError: (err, connection) => setSyncState((prev) => ({ ...prev, [connection.id]: String(err) })),
  })
  const copyWebhook = useMutation({
    mutationFn: (connection: Connection) => api.connectionWebhook(connection.id),
    onSuccess: async (result, connection) => {
      try {
        await navigator.clipboard.writeText(result.url)
        setSyncState((prev) => ({ ...prev, [connection.id]: t('providers.webhookCopied') }))
      } catch {
        setSyncState((prev) => ({ ...prev, [connection.id]: result.url }))
      }
    },
    onError: (err, connection) => setSyncState((prev) => ({ ...prev, [connection.id]: String(err) })),
  })
  const { data: bindingData, isLoading: bindingsLoading } = useQuery({
    queryKey: ['bindings', bindingsFor?.id],
    queryFn: () => api.bindings(bindingsFor!.id),
    enabled: !!bindingsFor,
  })
  const { data: bindingAccountsData } = useQuery({
    queryKey: ['accounts', bindingsFor?.id],
    queryFn: () => api.accounts(bindingsFor!.id),
    enabled: !!bindingsFor,
  })
  const addBinding = useMutation({
    mutationFn: () =>
      api.createBinding(bindingsFor!.id, {
        externalId: bindingExternal,
        name: bindingName || bindingExternal,
        direction: bindingDirection,
        accountId: bindingAccountID || undefined,
        settings: {
          ...(bindingBacklink ? { backlink: 'true' } : {}),
          ...(bindingTrusted ? { trustedAuthorsOnly: 'true' } : {}),
        },
      }),
    onSuccess: () => {
      setBindingExternal('')
      setBindingName('')
      setBindingAccountID('')
      setBindingBacklink(false)
      setBindingTrusted(false)
      qc.invalidateQueries({ queryKey: ['bindings', bindingsFor?.id] })
    },
  })
  const removeBinding = useMutation({
    mutationFn: (id: string) => api.deleteBinding(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['bindings', bindingsFor?.id] }),
  })
  const addIdentity = useMutation({
    mutationFn: () =>
      api.createIdentity(workspaceId, {
        provider: identityProvider,
        externalId: identityExternal,
        login: identityLogin || identityExternal,
        memberId: identityMember,
      }),
    onSuccess: () => {
      setIdentityExternal('')
      setIdentityLogin('')
      setIdentityMember('')
      qc.invalidateQueries({ queryKey: ['identities', workspaceId] })
    },
  })
  const removeIdentity = useMutation({
    mutationFn: (id: string) => api.deleteIdentity(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['identities', workspaceId] }),
  })
  const linkIdentity = useMutation({
    mutationFn: ({ id, memberId }: { id: string; memberId: string }) => api.linkIdentity(id, memberId),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['identities', workspaceId] }),
  })
  const addMemberIdentity = useMutation({
    mutationFn: () =>
      api.createMemberIdentity(workspaceId, {
        memberId: memberID,
        provider: memberProvider,
        externalId: memberExternal,
        login: memberLogin || memberExternal,
        email: memberEmail || undefined,
        token: memberToken || undefined,
      }),
    onSuccess: () => {
      setMemberID('')
      setMemberExternal('')
      setMemberLogin('')
      setMemberEmail('')
      setMemberToken('')
      qc.invalidateQueries({ queryKey: ['member-identities', workspaceId] })
    },
  })
  const removeMemberIdentity = useMutation({
    mutationFn: (id: string) => api.deleteMemberIdentity(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['member-identities', workspaceId] }),
  })
  const searchMemberUsers = useMutation({
    mutationFn: () => api.connectionUsers(jiraConnection!.id, memberUserQuery),
    onSuccess: (data) => setMemberUsers(data.items ?? []),
  })
  const { data: accountsData } = useQuery({
    queryKey: ['accounts', accountsFor?.id],
    queryFn: () => api.accounts(accountsFor!.id),
    enabled: !!accountsFor,
  })
  const addAccount = useMutation({
    mutationFn: () =>
      api.createAccount(accountsFor!.id, {
        token: accountToken,
        kind: accountKind,
        login: accountLogin || accountEmail,
        email: accountEmail,
      }),
    onSuccess: () => {
      setAccountToken('')
      setAccountKind('user')
      setAccountLogin('')
      setAccountEmail('')
      qc.invalidateQueries({ queryKey: ['accounts', accountsFor?.id] })
    },
  })
  const makeDefaultAccount = useMutation({
    mutationFn: (id: string) => api.setDefaultAccount(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['accounts', accountsFor?.id] }),
  })
  const removeAccount = useMutation({
    mutationFn: (id: string) => api.deleteAccount(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['accounts', accountsFor?.id] }),
  })
  const syncConnection = useMutation({
    mutationFn: async (connection: Connection) => ({
      id: connection.id,
      result: await api.workSync(connection.provider, connection.instance, workspaceId, connection.id),
    }),
    onSuccess: ({ id, result }) =>
      setSyncState((prev) => ({ ...prev, [id]: `synced ${result.synced}` })),
    onError: (err, connection) => setSyncState((prev) => ({ ...prev, [connection.id]: String(err) })),
  })
  const reconnect = useMutation({
    mutationFn: async () => {
      if (!reauthFor) throw new Error('no connection selected')
      return api.reconnectConnection(reauthFor.id, { token: reauthToken })
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['connections', workspaceId] })
      setReauthFor(null)
      setReauthToken('')
    },
  })
  const [connStates, setConnStates] = useState<Record<string, { id: string; name: string; group?: string }[]>>({})
  const [jiraStateMap, setJiraStateMap] = useState<Record<string, Record<string, string>>>({})
  const loadConnStates = useMutation({
    mutationFn: async (connection: Connection) => ({
      id: connection.id,
      result: await api.connectionStates(connection.id),
    }),
    onSuccess: ({ id, result }) =>
      setConnStates((prev) => ({ ...prev, [id]: result.items })),
  })
  const patchConn = useMutation({
    mutationFn: ({ id, body }: { id: string; body: { enabled?: boolean; transition?: string; stateMap?: Record<string, string> } }) =>
      api.patchConnection(id, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['connections', workspaceId] }),
  })
  const rollbackImport = useMutation({
    mutationFn: (id: string) => api.rollbackImport(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['imports', workspaceId] }),
  })
  const cancelImport = useMutation({
    mutationFn: (id: string) => api.cancelImport(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['imports', workspaceId] }),
  })
  const resumeImport = useMutation({
    mutationFn: (id: string) => api.resumeImport(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['imports', workspaceId] }),
  })
  const createImport = useMutation({
    mutationFn: ({ jql, startAt }: { jql: string; startAt?: number }) =>
      api.createJiraImport(workspaceId, { connectionId: jiraConnection!.id, jql, limit: 100, startAt }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['imports', workspaceId] }),
  })
  const createGitlabImport = useMutation({
    mutationFn: (query: string) =>
      api.createJiraImport(workspaceId, { connectionId: gitlabConnection!.id, jql: query, limit: 100 }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['imports', workspaceId] }),
  })
  const saveEdit = useMutation({
    mutationFn: async () => {
      if (!editFor) throw new Error('no connection selected')
      return api.patchConnection(editFor.id, {
        name: editName,
        baseUrl: editBaseUrl,
        instance: editInstance,
        agentId: '',
        enabled: editEnabled,
      })
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['connections', workspaceId] })
      setEditFor(null)
    },
  })
  const openEdit = (connection: Connection) => {
    setEditFor(connection)
    setEditName(connection.name)
    setEditBaseUrl(connection.baseUrl ?? '')
    setEditInstance(connection.instance)
    setEditEnabled(connection.enabled)
  }
  const addRule = useMutation({
    mutationFn: () =>
      api.createAssignmentRule({
        workspaceId,
        name: ruleName,
        provider: ruleProvider || undefined,
        instance: ruleInstance || undefined,
        label: ruleLabel || undefined,
        agentId: ruleAgent || undefined,
        groupId: ruleGroup || undefined,
        autoDispatch: ruleAuto,
      }),
    onSuccess: () => {
      setRuleName('')
      setRuleLabel('')
      setRuleInstance('')
      qc.invalidateQueries({ queryKey: ['assignment-rules', workspaceId] })
    },
  })
  const deleteRule = useMutation({
    mutationFn: (id: string) => api.deleteAssignmentRule(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['assignment-rules', workspaceId] }),
  })

  return (
    <div className="mx-auto max-w-[1180px] px-6 py-5">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h1 className="text-[18px] font-semibold tracking-tight">{t('page.providers')}</h1>
          <p className="mt-0.5 text-[12px] text-muted">{t('providers.subtitle')}</p>
        </div>
        <Button variant="secondary" size="sm" onClick={() => setOauthOpen(true)}>
          <KeyRound className="mr-1.5 h-3.5 w-3.5" /> OAuth Apps
        </Button>
      </div>

      {error && <p className="mt-4 text-[12px] text-err">{String(error.message)}</p>}
      {isLoading && <p className="mt-6 text-[12px] text-muted">{t('common.loading')}</p>}

      <div className="mt-5 space-y-6">
        {GROUPS.map((group) => {
          const list = items.filter((p) => p.kind === group.kind)
          if (list.length === 0) return null
          return (
            <section key={group.kind}>
              <div className="mb-2 flex items-center gap-2 text-[11px] font-medium uppercase tracking-wider text-muted">
                <group.icon className="h-3.5 w-3.5" />
                {t(group.labelKey)}
                <span className="text-muted/60">({list.length})</span>
              </div>
              <div className="grid grid-cols-2 gap-3">
                {list.map((p) => (
                  <div key={p.id} className="rounded-lg border border-line bg-panel/40 p-3">
                    <div className="flex items-center justify-between">
                      <div className="flex items-center gap-2">
                        <Plug className="h-3.5 w-3.5 text-accent2" />
                        <span className="text-[13px] font-medium">{p.name}</span>
                      </div>
                      <div className="flex items-center gap-2">
                        <span
                          className={cn(
                            'rounded px-1.5 py-0.5 text-[10px]',
                            connections.some((c) => c.provider === p.id)
                              ? 'bg-ok/10 text-ok'
                              : 'bg-panel2 text-muted',
                          )}
                        >
                          {connections.some((c) => c.provider === p.id)
                            ? t('providers.connectedCount', {
                                count: connections.filter((c) => c.provider === p.id).length,
                              })
                            : t('providers.notConnected')}
                        </span>
                        <span className="font-mono text-[11px] text-muted">{p.id}</span>
                      </div>
                    </div>
                    {connections.filter((c) => c.provider === p.id).map((c) => (
                      <div key={c.id} className="mt-1 truncate font-mono text-[10px] text-muted">
                        {c.instance} · {c.baseUrl || 'default'}
                      </div>
                    ))}
                    <div className="mt-2 flex flex-wrap items-center gap-1">
                      <span className="rounded border border-line2 px-1.5 py-0.5 text-[10px] uppercase tracking-wide text-muted">
                        {t('providers.auth', { auth: p.auth })}
                      </span>
                      {p.capabilities.map((c) => (
                        <span
                          key={c}
                          className={cn(
                            'rounded px-1.5 py-0.5 text-[10px]',
                            'bg-accent/10 text-accent2',
                          )}
                        >
                          {c}
                        </span>
                      ))}
                      <button
                        className="ml-auto rounded border border-line2 px-1.5 py-0.5 text-[10px] text-muted hover:text-fg"
                        onClick={() => {
                          const existing = connections.find((c) => c.provider === p.id)
                          if (existing) {
                            openEdit(existing)
                            return
                          }
                          setConnectFor(p.id)
                          const preferredAuth = p.authOptions?.find((option) => option.default)?.kind ?? p.authOptions?.[0]?.kind ?? p.auth
                          setAuthKind(preferredAuth === 'oauth2' && !oauthProviders.includes(p.id) ? 'token' : preferredAuth)
                          setConnectMsg('')
                          setFields({})
                          setToken('')
                          setBaseUrl('')
                          setInstance('')
                          setStates([])
                          setTransition('')
                          setSyncDirection('read_write')
                          setConnectionName('')
                          setWebhookSecret('')
                        }}
                      >
                        {connections.some((c) => c.provider === p.id)
                          ? t('providers.manage')
                          : t('providers.connect')}
                      </button>
                      <button
                        className="rounded border border-line2 px-1.5 py-0.5 text-[10px] text-muted hover:text-fg"
                        onClick={() => probe.mutate(p.id)}
                        disabled={probe.isPending && probe.variables === p.id}
                      >
                        {probe.isPending && probe.variables === p.id
                          ? t('providers.testing')
                          : t('providers.test')}
                      </button>
                    </div>
                    {results[p.id] && (
                      <p
                        className={cn(
                          'mt-1.5 truncate text-[10.5px]',
                          results[p.id] === 'ok' ? 'text-ok' : 'text-err',
                        )}
                        title={results[p.id]}
                      >
                        {results[p.id] === 'ok' ? t('providers.credentialWorks') : results[p.id]}
                      </p>
                    )}
                  </div>
                ))}
              </div>
            </section>
          )
        })}
      </div>

      {!isLoading && items.length === 0 && (
        <p className="mt-8 text-[12px] text-muted">{t('providers.empty')}</p>
      )}

      {connectMsg && <p className="mt-4 text-[12px] text-muted">{connectMsg}</p>}

      <section className="mt-8">
        <h2 className="text-[13px] font-medium">{t('providers.connections')}</h2>
        <div className="mt-2 space-y-2">
          {connections.map((connection) => {
            const webhook = connection.automationId
              ? `${window.location.origin}/webhooks/${connection.provider}/${connection.automationId}`
              : ''
            return (
              <div key={connection.id} className="rounded-lg border border-line bg-panel/40 p-3">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-[12.5px] font-medium">{connection.name || connection.provider}</span>
                  <span className="rounded border border-line2 px-1.5 py-0.5 text-[10px] text-muted">
                    {connection.provider}
                  </span>
                  <span
                    className={cn(
                      'rounded px-1.5 py-0.5 text-[10px]',
                      connection.status === 'connected'
                        ? 'bg-ok/10 text-ok'
                        : connection.status === 'needs_reauth'
                          ? 'bg-warn/10 text-warn'
                          : 'bg-panel2 text-muted',
                    )}
                  >
                    {connection.status}
                  </span>
                  <span className="text-[10px] text-muted">{connection.authMethod}</span>
                  <span className="font-mono text-[10.5px] text-muted">{connection.instance}</span>
                  <div className="ml-auto flex items-center gap-2">
                    {connection.status === 'needs_reauth' && (
                      <Button
                        size="sm"
                        variant="primary"
                        onClick={() => {
                          if (connection.authMethod === 'oauth') {
                            const params = new URLSearchParams({
                              workspace: workspaceId,
                              instance: connection.instance,
                              name: connection.name,
                              connectionId: connection.id,
                            })
                            window.location.href = `/api/v1/integrations/github/oauth/start?${params}`
                          } else {
                            setReauthFor(connection)
                          }
                        }}
                      >
                        {t('providers.reconnect')}
                      </Button>
                    )}
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => {
                        setAccountsFor(connection)
                        setAccountToken('')
      setAccountKind('user')
                        setAccountLogin('')
                        setAccountEmail('')
                      }}
                    >
                      <Plug className="h-3 w-3" /> {t('providers.accounts')}
                    </Button>
                    <Button
                      size="sm"
                      variant="ghost"
                      onClick={() => {
                        setBindingsFor(connection)
                        setBindingExternal('')
                        setBindingName('')
                        setBindingDirection(connection.syncDirection === 'read' ? 'read' : 'read_write')
                        setBindingAccountID('')
                        setBindingBacklink(false)
                        setBindingTrusted(false)
                      }}
                    >
                      <Boxes className="h-3 w-3" /> {t('providers.bindings')}
                    </Button>
                    {showWebhook && connection.automationId && (
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={copyWebhook.isPending}
                        onClick={() => copyWebhook.mutate(connection)}
                      >
                        <Copy className="h-3 w-3" /> {t('providers.copyWebhook')}
                      </Button>
                    )}
                    {showWebhook && connection.automationId && ['github', 'plane', 'linear', 'jira'].includes(connection.provider) && (
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={registerWebhook.isPending}
                        onClick={() => registerWebhook.mutate(connection)}
                      >
                        {t('providers.registerWebhook')}
                      </Button>
                    )}
                    {items.find((provider) => provider.id === connection.provider)?.kind === 'task' && (
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={syncConnection.isPending}
                        onClick={() => syncConnection.mutate(connection)}
                      >
                        <RefreshCw className="h-3 w-3" /> {t('providers.sync')}
                      </Button>
                    )}
                    <button
                      className="rounded border border-line2 px-1.5 py-0.5 text-[10px] text-muted hover:text-fg"
                      onClick={() => {
                        setEditFor(connection)
                        setEditName(connection.name)
                        setEditBaseUrl(connection.baseUrl ?? '')
                        setEditInstance(connection.instance)
                                            setEditEnabled(connection.enabled)
                      }}
                    >
                      {t('providers.edit')}
                    </button>
                    <button
                      title={t('providers.delete')}
                      className="text-muted hover:text-err"
                      onClick={() => removeConnection.mutate(connection.id)}
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
                <div className="mt-1 flex flex-wrap items-center gap-2 text-[10.5px] text-muted">
                  <span>
                    {t('providers.config')}: {connection.baseUrl || 'default'}
                  </span>
                  <span>·</span>
                  <span>{connection.authMethod}</span>
                  <span>·</span>
                  <span>{new Date(connection.createdAt).toLocaleDateString()}</span>
                </div>
                {connection.status === 'connected' && <ConnectionSyncStatus connectionId={connection.id} />}
                {showWebhook && connection.automationId && <ConnectionDeliveries automationId={connection.automationId} />}
                {connection.provider === 'plane' && (
                  <div className="mt-2 flex items-center gap-2">
                    {connStates[connection.id] ? (
                      <select
                        className="h-7 min-w-0 flex-1 rounded-md border border-line2 bg-panel px-2 text-[11px]"
                        value={connection.transition ?? ''}
                        onChange={(e) =>
                          patchConn.mutate({ id: connection.id, body: { transition: e.target.value } })
                        }
                      >
                        <option value="">{t('providers.doneState')}</option>
                        {connStates[connection.id].map((state) => (
                          <option key={state.id} value={state.id}>
                            {state.name}
                          </option>
                        ))}
                      </select>
                    ) : (
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={loadConnStates.isPending}
                        onClick={() => loadConnStates.mutate(connection)}
                      >
                        {t('providers.states')}
                      </Button>
                    )}
                  </div>
                )}
                {connection.provider === 'jira' && (
                  <div className="mt-2 flex flex-wrap items-center gap-2">
                    {connStates[connection.id] ? (
                      (['open', 'in_progress', 'done'] as const).map((local) => (
                        <label key={local} className="text-[10px] text-muted">
                          {t(`status.${local}`)}
                          <select
                            className="ml-1 h-7 rounded-md border border-line2 bg-panel px-1 text-[10.5px]"
                            value={jiraStateMap[connection.id]?.[local] ?? ''}
                            onChange={(e) => {
                              const value = e.target.value
                              setJiraStateMap((prev) => ({
                                ...prev,
                                [connection.id]: { ...(prev[connection.id] ?? {}), [local]: value },
                              }))
                              patchConn.mutate({ id: connection.id, body: { stateMap: { [local]: value } } })
                            }}
                          >
                            <option value="">—</option>
                            {connStates[connection.id].map((state) => (
                              <option key={state.id} value={state.id}>{state.name}</option>
                            ))}
                          </select>
                        </label>
                      ))
                    ) : (
                      <Button
                        size="sm"
                        variant="ghost"
                        disabled={loadConnStates.isPending}
                        onClick={() => loadConnStates.mutate(connection)}
                      >
                        {t('providers.states')}
                      </Button>
                    )}
                  </div>
                )}
                {connection.provider === 'jira' && (
                  <div className="mt-2 space-y-1">
                    <Button
                      size="sm"
                      variant="ghost"
                      disabled={createImport.isPending}
                      onClick={() => {
                        const jql = window.prompt(t('providers.importJql'), 'project = ENG ORDER BY updated DESC')
                        if (jql?.trim()) createImport.mutate({ jql: jql.trim() })
                      }}
                    >
                      {t('providers.importJira')}
                    </Button>
                    {importRuntime && (
                      <span className="text-[10px] text-muted">
                        {importRuntime.active}/{importRuntime.concurrency} active · {importRuntime.queued} queued
                      </span>
                    )}
                    {imports.filter((batch) => batch.connectionId === connection.id).map((batch) => (
                      <div key={batch.id} className="flex items-center gap-2 text-[10.5px] text-muted">
                        <span>{batch.status}</span>
                        <span>{batch.imported} imported</span>
                        <span>{batch.failed} failed</span>
                        {(batch.status === 'queued' || batch.status === 'running') && (
                          <button className="hover:text-err" onClick={() => cancelImport.mutate(batch.id)}>
                            {t('providers.cancel')}
                          </button>
                        )}
                        {(batch.status === 'failed' || batch.status === 'partial') && batch.nextStartAt > 0 && (
                          <button
                            className="hover:text-fg"
                            onClick={() => resumeImport.mutate(batch.id)}
                          >
                            {t('providers.resume')}
                          </button>
                        )}
                        {batch.status !== 'rolled_back' && (
                          <button className="ml-auto hover:text-err" onClick={() => rollbackImport.mutate(batch.id)}>
                            {t('providers.rollback')}
                          </button>
                        )}
                      </div>
                    ))}
                  </div>
                )}
                {connection.provider === 'gitlab' && gitlabConnection && (
                  <div className="mt-2">
                    <Button
                      size="sm"
                      variant="ghost"
                      disabled={createGitlabImport.isPending}
                      onClick={() => {
                        const query = window.prompt(t('providers.importGitlabQuery'), 'bug')
                        if (query?.trim()) createGitlabImport.mutate(query.trim())
                      }}
                    >
                      {t('providers.importGitlab')}
                    </Button>
                  </div>
                )}
                {showWebhook && (
                  <div className="mt-2 flex items-center gap-2">
                    <code className="min-w-0 flex-1 truncate rounded bg-panel2 px-2 py-1 font-mono text-[10.5px] text-muted">
                      {webhook || t('providers.noAutomation')}
                    </code>
                    <button
                      className="text-muted hover:text-fg disabled:opacity-40"
                      title={t('providers.copyWebhook')}
                      disabled={!webhook}
                      onClick={() => webhook && navigator.clipboard?.writeText(webhook)}
                    >
                      <Copy className="h-3.5 w-3.5" />
                    </button>
                  </div>
                )}
                {syncState[connection.id] && (
                  <p className="mt-1 text-[10.5px] text-muted">{syncState[connection.id]}</p>
                )}
              </div>
            )
          })}
          {connections.length === 0 && (
            <p className="text-[11.5px] text-muted">{t('providers.noConnections')}</p>
          )}
        </div>
      </section>

      <section className="mt-8">
        <h2 className="text-[13px] font-medium">{t('providers.identities')}</h2>
        <p className="mt-0.5 text-[11.5px] text-muted">{t('providers.identitiesHint')}</p>
        <div className="mt-2 space-y-2">
          {identities.map((identity) => (
            <div key={identity.id} className="flex flex-wrap items-center gap-2 rounded-lg border border-line bg-panel/40 p-2.5">
              <span className="text-[12px] font-medium">{identity.login || identity.externalId}</span>
              <span className="font-mono text-[10.5px] text-muted">{identity.provider}</span>
              <span className="font-mono text-[10.5px] text-muted">{identity.externalId}</span>
              <span className="text-[11px] text-muted">→</span>
              <Input
                className="h-7 w-28"
                value={linkMembers[identity.id] ?? identity.memberId}
                onChange={(e) => setLinkMembers((prev) => ({ ...prev, [identity.id]: e.target.value }))}
                placeholder={t('providers.identityMember')}
              />
              <button
                className="rounded border border-line2 px-1.5 py-0.5 text-[10px] text-muted hover:text-fg"
                onClick={() => linkIdentity.mutate({ id: identity.id, memberId: linkMembers[identity.id] ?? identity.memberId })}
              >
                {t('common.save')}
              </button>
              <button className="ml-auto text-muted hover:text-err" onClick={() => removeIdentity.mutate(identity.id)}>
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </div>
          ))}
          {identities.length === 0 && <p className="text-[11.5px] text-muted">{t('providers.noIdentities')}</p>}
          <form
            className="flex flex-wrap items-end gap-2 rounded-lg border border-line bg-panel/40 p-2.5"
            onSubmit={(e) => {
              e.preventDefault()
              if (identityExternal.trim() && identityMember.trim()) addIdentity.mutate()
            }}
          >
            <label className="text-[11px] text-muted">
              {t('meta.provider')}
              <select
                className="mt-1 h-8 rounded-md border border-line2 bg-panel px-2 text-[11px]"
                value={identityProvider}
                onChange={(e) => setIdentityProvider(e.target.value)}
              >
                {items.filter((provider) => provider.kind === 'task').map((provider) => (
                  <option key={provider.id} value={provider.id}>{provider.name}</option>
                ))}
              </select>
            </label>
            <label className="text-[11px] text-muted">
              {t('providers.bindingExternal')}
              <Input className="mt-1 h-8 w-36" value={identityExternal} onChange={(e) => setIdentityExternal(e.target.value)} />
            </label>
            <label className="text-[11px] text-muted">
              {t('providers.identityLogin')}
              <Input className="mt-1 h-8 w-32" value={identityLogin} onChange={(e) => setIdentityLogin(e.target.value)} />
            </label>
            <label className="text-[11px] text-muted">
              {t('providers.identityMember')}
              <Input className="mt-1 h-8 w-32" value={identityMember} onChange={(e) => setIdentityMember(e.target.value)} />
            </label>
            <Button type="submit" size="sm" variant="primary" disabled={!identityExternal.trim() || !identityMember.trim() || addIdentity.isPending}>
              {t('common.add')}
            </Button>
          </form>
          {addIdentity.error && <p className="text-[11.5px] text-err">{String(addIdentity.error.message)}</p>}
        </div>
      </section>

      <section className="mt-8">
        <h2 className="text-[13px] font-medium">{t('providers.memberIdentities')}</h2>
        <p className="mt-0.5 text-[11.5px] text-muted">{t('providers.memberIdentitiesHint')}</p>
        <div className="mt-2 space-y-2">
          {memberIdentities.map((identity) => (
            <div key={identity.id} className="flex flex-wrap items-center gap-2 rounded-lg border border-line bg-panel/40 p-2.5">
              <span className="text-[12px] font-medium">{identity.login || identity.externalId}</span>
              <span className="font-mono text-[10.5px] text-muted">{identity.provider}</span>
              <span className="font-mono text-[10.5px] text-muted">{identity.externalId}</span>
              <span className="text-[11px] text-muted">→ {identity.memberId}</span>
              <button className="ml-auto text-muted hover:text-err" onClick={() => removeMemberIdentity.mutate(identity.id)}>
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </div>
          ))}
          {memberIdentities.length === 0 && <p className="text-[11.5px] text-muted">{t('providers.noMemberIdentities')}</p>}
          <form
            className="flex flex-wrap items-end gap-2 rounded-lg border border-line bg-panel/40 p-2.5"
            onSubmit={(e) => {
              e.preventDefault()
              if (memberID.trim() && memberExternal.trim()) addMemberIdentity.mutate()
            }}
          >
            <label className="text-[11px] text-muted">
              {t('meta.provider')}
              <select
                className="mt-1 h-8 rounded-md border border-line2 bg-panel px-2 text-[11px]"
                value={memberProvider}
                onChange={(e) => setMemberProvider(e.target.value)}
              >
                {items.filter((provider) => provider.kind === 'task').map((provider) => (
                  <option key={provider.id} value={provider.id}>{provider.name}</option>
                ))}
              </select>
            </label>
            <label className="text-[11px] text-muted">
              {t('providers.identityMember')}
              <Input className="mt-1 h-8 w-28" value={memberID} onChange={(e) => setMemberID(e.target.value)} />
            </label>
            <label className="text-[11px] text-muted">
              {t('providers.bindingExternal')}
              <Input className="mt-1 h-8 w-32" value={memberExternal} onChange={(e) => setMemberExternal(e.target.value)} />
            </label>
            <label className="text-[11px] text-muted">
              {t('providers.identityLogin')}
              <Input className="mt-1 h-8 w-28" value={memberLogin} onChange={(e) => setMemberLogin(e.target.value)} />
            </label>
            <label className="text-[11px] text-muted">
              {t('providers.token')}
              <Input className="mt-1 h-8 w-44" type="password" value={memberToken} onChange={(e) => setMemberToken(e.target.value)} />
            </label>
            <Button type="submit" size="sm" variant="primary" disabled={!memberID.trim() || !memberExternal.trim() || addMemberIdentity.isPending}>
              {t('common.add')}
            </Button>
            {memberOAuthEnabled && (
              <Button
                type="button"
                size="sm"
                variant="ghost"
                disabled={!memberID.trim()}
                onClick={() => {
                  const params = new URLSearchParams({ workspace: workspaceId, memberId: memberID.trim() })
                  if (memberProvider === 'gitlab') params.set('baseUrl', 'https://gitlab.com')
                  window.location.href = `/api/v1/integrations/${memberProvider}/oauth/start?${params.toString()}`
                }}
              >
                <Plug className="h-3.5 w-3.5" /> {t('providers.authorizeMemberOAuth')}
              </Button>
            )}
          </form>
          {addMemberIdentity.error && <p className="text-[11.5px] text-err">{String(addMemberIdentity.error.message)}</p>}
          {memberProvider === 'jira' && jiraConnection && (
            <div className="rounded-lg border border-line bg-panel/40 p-2.5">
              <div className="flex gap-2">
                <Input
                  className="h-8"
                  value={memberUserQuery}
                  onChange={(e) => setMemberUserQuery(e.target.value)}
                  placeholder={t('providers.searchJiraUser')}
                />
                <Button
                  size="sm"
                  variant="ghost"
                  disabled={!memberUserQuery.trim() || searchMemberUsers.isPending}
                  onClick={() => searchMemberUsers.mutate()}
                >
                  {t('common.search')}
                </Button>
              </div>
              <div className="mt-2 space-y-1">
                {memberUsers.map((user) => (
                  <button
                    key={user.id}
                    className="flex w-full items-center gap-2 rounded border border-line2/60 px-2 py-1 text-left text-[11px] hover:text-fg"
                    onClick={() => {
                      setMemberExternal(user.id)
                      setMemberLogin(user.login)
                      setMemberEmail(user.email ?? '')
                    }}
                  >
                    <span>{user.name || user.login}</span>
                    <span className="font-mono text-[10px] text-muted">{user.email || user.id}</span>
                  </button>
                ))}
              </div>
            </div>
          )}
        </div>
      </section>

      <section className="mt-8">
        <h2 className="text-[13px] font-medium">{t('providers.rules')}</h2>
        <p className="mt-0.5 text-[11.5px] text-muted">{t('providers.rulesHint')}</p>
        <div className="mt-2 space-y-2">
          {rules.map((rule) => (
            <div key={rule.id} className="flex flex-wrap items-center gap-2 rounded-lg border border-line bg-panel/40 p-2.5">
              <span className="text-[12px] font-medium">{rule.name || rule.id}</span>
              <span className="font-mono text-[10.5px] text-muted">{rule.provider || '*'}</span>
              <span className="font-mono text-[10.5px] text-muted">{rule.instance || '*'}</span>
              {rule.label && <span className="rounded bg-panel2 px-1.5 py-0.5 text-[10px]">{rule.label}</span>}
              <span className="ml-auto text-[10.5px] text-muted">
                {rule.agentId} · {rule.autoDispatch ? t('providers.autoDispatch') : t('providers.assignOnly')}
              </span>
              <button className="text-muted hover:text-err" onClick={() => deleteRule.mutate(rule.id)}>
                <Trash2 className="h-3.5 w-3.5" />
              </button>
            </div>
          ))}
          {rules.length === 0 && <p className="text-[11.5px] text-muted">{t('providers.noRules')}</p>}
        </div>
        <form
          className="mt-3 flex flex-wrap items-end gap-2 rounded-lg border border-line bg-panel/40 p-2.5"
          onSubmit={(e) => {
            e.preventDefault()
            if (ruleAgent || ruleGroup) addRule.mutate()
          }}
        >
          <label className="text-[11px] text-muted">
            {t('providers.ruleName')}
            <Input className="mt-1 h-7 w-40" value={ruleName} onChange={(e) => setRuleName(e.target.value)} />
          </label>
          <label className="text-[11px] text-muted">
            {t('meta.provider')}
            <Input className="mt-1 h-7 w-28" value={ruleProvider} onChange={(e) => setRuleProvider(e.target.value)} />
          </label>
          <label className="text-[11px] text-muted">
            {t('providers.instance')}
            <Input className="mt-1 h-7 w-56" value={ruleInstance} onChange={(e) => setRuleInstance(e.target.value)} />
          </label>
          <label className="text-[11px] text-muted">
            {t('common.labels')}
            <Input className="mt-1 h-7 w-28" value={ruleLabel} onChange={(e) => setRuleLabel(e.target.value)} />
          </label>
          <label className="text-[11px] text-muted">
            {t('providers.agent')}
            <select
              className="mt-1 h-7 rounded-md border border-line2 bg-panel px-2 text-[11px]"
              value={ruleAgent}
              onChange={(e) => setRuleAgent(e.target.value)}
            >
              <option value="">—</option>
              {agents.map((agent) => (
                <option key={agent.id} value={agent.id}>
                  {agent.name}
                </option>
              ))}
            </select>
          </label>
          <label className="text-[11px] text-muted">
            {t('providers.group')}
            <select
              className="mt-1 h-7 rounded-md border border-line2 bg-panel px-2 text-[11px]"
              value={ruleGroup}
              onChange={(e) => setRuleGroup(e.target.value)}
            >
              <option value="">—</option>
              {groups.map((group) => (
                <option key={group.id} value={group.id}>
                  {group.name}
                </option>
              ))}
            </select>
          </label>
          <label className="flex items-center gap-1.5 pb-1.5 text-[11px] text-muted">
            <input type="checkbox" checked={ruleAuto} onChange={(e) => setRuleAuto(e.target.checked)} />
            {t('providers.autoDispatch')}
          </label>
          <Button type="submit" size="sm" variant="primary" disabled={(!ruleAgent && !ruleGroup) || addRule.isPending}>
            {t('common.add')}
          </Button>
        </form>
      </section>

      <Dialog open={!!connectFor} onOpenChange={(open) => !open && setConnectFor('')}>
        <DialogContent
          title={t('providers.connect') + (connectFor ? ` · ${connectFor}` : '')}
          description={activeProvider?.kind === 'task' ? t('providers.connectHint') : t('providers.connectHintChat')}
        >
          <form
            className="flex flex-col gap-3"
            onSubmit={(e) => {
              e.preventDefault()
              connect.mutate()
            }}
          >
            {authOptions.length > 1 && (
              <div className="flex flex-wrap gap-1.5">
                {authOptions.map((option) => (
                  <button
                    key={option.kind}
                    type="button"
                    className={`rounded border px-2 py-1 text-[11px] ${authKind === option.kind ? 'border-accent2 text-fg' : 'border-line2 text-muted'}`}
                    onClick={() => setAuthKind(option.kind)}
                  >
                    {t(option.labelKey ?? `providers.auth.${option.kind}`)}
                  </button>
                ))}
              </div>
            )}
            {configFields.length > 0 ? configFields.map((field, index) => {
              const label = field.labelKey ? t(field.labelKey) : field.label ?? field.key
              const value = fieldValue(field.key)
              return (
                <label key={field.key} className="text-[12px] text-muted">
                  {label}{field.required ? ' *' : ''}
                  {field.type === 'textarea' ? (
                    <textarea
                      className="mt-1 min-h-20 w-full rounded-md border border-line2 bg-panel px-2 py-1 text-[12px]"
                      autoFocus={index === 0}
                      value={value}
                      placeholder={field.placeholder}
                      onChange={(e) => setFieldValue(field.key, e.target.value)}
                    />
                  ) : field.type === 'select' ? (
                    <select
                      className="mt-1 h-8 w-full rounded-md border border-line2 bg-panel px-2 text-[12px]"
                      autoFocus={index === 0}
                      value={value}
                      onChange={(e) => setFieldValue(field.key, e.target.value)}
                    >
                      <option value="">{field.placeholder || '—'}</option>
                      {(field.options ?? []).map((option) => (
                        <option key={option} value={option}>{option}</option>
                      ))}
                    </select>
                  ) : (
                    <Input
                      className="mt-1"
                      type={field.type === 'password' || field.secret ? 'password' : field.type === 'url' ? 'url' : 'text'}
                      autoFocus={index === 0}
                      value={value}
                      placeholder={field.placeholder}
                      onChange={(e) => setFieldValue(field.key, e.target.value)}
                    />
                  )}
                  {(field.helpKey ? t(field.helpKey) : field.help) && (
                    <span className="mt-1 block text-[10.5px] text-muted/70">
                      {field.helpKey ? t(field.helpKey) : field.help}
                    </span>
                  )}
                </label>
              )
            }) : (
              <>
                <label className="text-[12px] text-muted">
                  {t('providers.token')}
                  <Input className="mt-1" type="password" autoFocus value={token} onChange={(e) => setToken(e.target.value)} />
                </label>
                <label className="text-[12px] text-muted">
                  {t('providers.baseUrl')}
                  <Input className="mt-1" value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} placeholder="https://api.example.com" />
                </label>
                <label className="text-[12px] text-muted">
                  {t('providers.instance')}
                  <Input className="mt-1" value={instance} onChange={(e) => setInstance(e.target.value)} placeholder="workspace/project" />
                </label>
              </>
            )}
            {connectFor === 'plane' && (
              <PlaneCatalogPicker
                token={fieldValue('token')}
                baseUrl={fieldValue('baseUrl')}
                onSelect={(value) => setFieldValue('instance', value)}
              />
            )}
            {oauthProviders.includes(connectFor) && (
              <Button
                type="button"
                variant="primary"
                onClick={() => {
                  const params = new URLSearchParams({
                    workspace: workspaceId,
                    instance: fieldValue('instance'),
                    baseUrl: fieldValue('baseUrl'),
                    channel: fieldValue('channel'),
                    name: connectionName,
                    email: fieldValue('email'),
                    syncDirection,
                  })
                  window.location.href = `/api/v1/integrations/${connectFor}/oauth/start?${params.toString()}`
                }}
              >
                <Plug className="h-3.5 w-3.5" /> {t('providers.authorizeOAuth')}
              </Button>
            )}
            <label className="text-[12px] text-muted">
              {t('providers.connectionName')}
              <Input
                className="mt-1"
                value={connectionName}
                onChange={(e) => setConnectionName(e.target.value)}
                placeholder={activeProvider ? `${activeProvider.name} · connection` : 'Connection name'}
              />
            </label>
            {activeProvider?.kind === 'task' && (
              <>
                <label className="text-[12px] text-muted">
                  {t('providers.syncDirection')}
                  <select
                    className="mt-1 h-8 w-full rounded-md border border-line2 bg-panel px-2 text-[12px]"
                    value={syncDirection}
                    onChange={(e) => setSyncDirection(e.target.value)}
                  >
                    <option value="read_write">{t('providers.syncReadWrite')}</option>
                    <option value="read">{t('providers.syncRead')}</option>
                  </select>
                </label>
                <label className="text-[12px] text-muted">
                  {t('providers.commentTemplate')}
                  <textarea
                    className="mt-1 min-h-20 w-full rounded-md border border-line2 bg-panel px-2 py-1 font-mono text-[11px]"
                    value={commentTemplate}
                    onChange={(e) => setCommentTemplate(e.target.value)}
                    placeholder="{{status}} · {{branch}} · {{url}}"
                  />
                </label>
                {!configFields.some((field) => field.key === 'webhookSecret') && (
                  <label className="text-[12px] text-muted">
                    {t('providers.webhookSecret')}
                    <Input
                      className="mt-1"
                      type="password"
                      value={webhookSecret}
                      onChange={(e) => setWebhookSecret(e.target.value)}
                      placeholder="optional"
                    />
                  </label>
                )}
                <label className="text-[12px] text-muted">
                  {t('providers.transition')}
                  <div className="mt-1 flex gap-2">
                    {states.length > 0 ? (
                      <select
                        className="h-8 min-w-0 flex-1 rounded-md border border-line2 bg-panel px-2 text-[13px]"
                        value={transition}
                        onChange={(e) => setTransition(e.target.value)}
                      >
                        <option value="">{t('providers.selectState')}</option>
                        {states.map((state) => (
                          <option key={state.id} value={state.id}>
                            {state.name}
                          </option>
                        ))}
                      </select>
                    ) : (
                      <Input
                        className="min-w-0 flex-1"
                        value={transition}
                        onChange={(e) => setTransition(e.target.value)}
                        placeholder="state-uuid"
                      />
                    )}
                    <Button
                      type="button"
                      variant="ghost"
                      disabled={!token || !instance || loadStates.isPending}
                      onClick={() => loadStates.mutate()}
                    >
                      {t('providers.loadStates')}
                    </Button>
                  </div>
                </label>
              </>
            )}
            {connectFor === 'github' && authKind === 'oauth2' && githubOAuth?.enabled && (
              <Button
                type="button"
                variant="primary"
                onClick={() => {
                  const params = new URLSearchParams({
                    workspace: workspaceId,
                    instance,
                    name: connectionName,
                  })
                  window.location.href = `/api/v1/integrations/github/oauth/start?${params}`
                }}
              >
                {t('providers.authorizeGithub')}
              </Button>
            )}
            {connectFor === 'github' && authKind === 'app' && githubApp?.enabled && (
              <Button
                type="button"
                variant="primary"
                onClick={() => {
                  const params = new URLSearchParams({ workspace: workspaceId, name: connectionName })
                  window.location.href = `/api/v1/integrations/github/app/start?${params}`
                }}
              >
                {t('providers.installGithubApp')}
              </Button>
            )}
            {connect.error && <p className="text-[12px] text-err">{String(connect.error.message)}</p>}
            <div className="flex justify-end gap-2">
              <Button type="button" variant="ghost" onClick={() => setConnectFor('')}>
                {t('common.cancel')}
              </Button>
              {(authKind === 'token' || authKind === 'bot') && (
                <Button type="submit" variant="primary" disabled={!configReady || connect.isPending}>
                  {connect.isPending ? t('providers.connecting') : t('providers.connect')}
                </Button>
              )}
            </div>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={!!accountsFor} onOpenChange={(open) => !open && setAccountsFor(null)}>
        <DialogContent
          title={`${t('providers.accounts')} · ${accountsFor?.name ?? ''}`}
          description={t('providers.accountsHint')}
        >
          <div className="flex flex-col gap-3">
            <div className="space-y-1">
              {(accountsData?.items ?? []).map((account) => (
                <div key={account.id} className="flex flex-wrap items-center gap-2 rounded border border-line px-2 py-1.5">
                  <span className="text-[12px]">{account.login || account.email || account.id}</span>
                  <span className="rounded bg-panel2 px-1.5 py-0.5 text-[10px]">{account.kind}</span>
                  {account.isDefault && (
                    <span className="rounded bg-accent/10 px-1.5 py-0.5 text-[10px] text-accent2">{t('providers.defaultAccount')}</span>
                  )}
                  {!account.isDefault && (
                    <button className="text-[10.5px] text-muted hover:text-fg" onClick={() => makeDefaultAccount.mutate(account.id)}>
                      {t('providers.makeDefault')}
                    </button>
                  )}
                  {!account.isDefault && (
                    <button className="ml-auto text-muted hover:text-err" onClick={() => removeAccount.mutate(account.id)}>
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  )}
                </div>
              ))}
              {(accountsData?.items ?? []).length === 0 && <p className="text-[11.5px] text-muted">{t('providers.noAccounts')}</p>}
            </div>
            <form
              className="flex flex-wrap items-end gap-2 border-t border-line pt-3"
              onSubmit={(e) => {
                e.preventDefault()
                if (accountToken.trim()) addAccount.mutate()
              }}
            >
              <label className="text-[11px] text-muted">
                {t('providers.accountKind')}
                <select
                  className="mt-1 h-8 rounded-md border border-line2 bg-panel px-2 text-[11px]"
                  value={accountKind}
                  onChange={(e) => setAccountKind(e.target.value)}
                >
                  <option value="user">{t('providers.accountUser')}</option>
                  <option value="org">{t('providers.accountOrg')}</option>
                  <option value="bot">{t('providers.accountBot')}</option>
                </select>
              </label>
              <label className="text-[11px] text-muted">
                {t('providers.token')}
                <Input className="mt-1 h-8 w-56" type="password" value={accountToken} onChange={(e) => setAccountToken(e.target.value)} />
              </label>
              <label className="text-[11px] text-muted">
                {t('providers.identityLogin')}
                <Input className="mt-1 h-8 w-32" value={accountLogin} onChange={(e) => setAccountLogin(e.target.value)} />
              </label>
              <label className="text-[11px] text-muted">
                {t('providers.email')}
                <Input className="mt-1 h-8 w-40" value={accountEmail} onChange={(e) => setAccountEmail(e.target.value)} />
              </label>
              <Button type="submit" size="sm" variant="primary" disabled={!accountToken.trim() || addAccount.isPending}>
                {t('common.add')}
              </Button>
            </form>
            {addAccount.error && <p className="text-[11.5px] text-err">{String(addAccount.error.message)}</p>}
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={!!bindingsFor} onOpenChange={(open) => !open && setBindingsFor(null)}>
        <DialogContent
          title={`${t('providers.bindings')} · ${bindingsFor?.name ?? ''}`}
          description={t('providers.bindingsHint')}
        >
          <div className="flex flex-col gap-3">
            <div className="space-y-1">
              {bindingsLoading && <p className="text-[11.5px] text-muted">{t('common.loading')}</p>}
              {!bindingsLoading && (bindingData?.items ?? []).length === 0 && (
                <p className="text-[11.5px] text-muted">{t('providers.noBindings')}</p>
              )}
              {(bindingData?.items ?? []).map((binding) => (
                <div key={binding.id} className="flex items-center gap-2 rounded border border-line px-2 py-1.5">
                  <span className="min-w-0 flex-1 truncate text-[12px]">{binding.name || binding.externalId}</span>
                  <span className="font-mono text-[10px] text-muted">{binding.externalId}</span>
                  <span className="rounded bg-panel2 px-1.5 py-0.5 text-[10px]">
                    {binding.direction === 'read' ? t('providers.syncRead') : t('providers.syncReadWrite')}
                  </span>
                  {binding.lastError ? (
                    <span className="max-w-40 truncate text-[10px] text-err" title={binding.lastError}>{binding.lastError}</span>
                  ) : binding.lastSyncedAt ? (
                    <span className="text-[10px] text-muted">{new Date(binding.lastSyncedAt).toLocaleString()}</span>
                  ) : null}
                  <button
                    className="text-muted hover:text-err"
                    onClick={() => removeBinding.mutate(binding.id)}
                    aria-label="delete binding"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              ))}
            </div>
            <form
              className="flex flex-wrap items-end gap-2 border-t border-line pt-3"
              onSubmit={(e) => {
                e.preventDefault()
                if (bindingExternal.trim()) addBinding.mutate()
              }}
            >
              <label className="text-[11px] text-muted">
                {t('providers.bindingExternal')}
                <Input
                  className="mt-1 h-8 w-44"
                  value={bindingExternal}
                  onChange={(e) => setBindingExternal(e.target.value)}
                  placeholder="owner/repo"
                />
              </label>
              <label className="text-[11px] text-muted">
                {t('providers.connectionName')}
                <Input
                  className="mt-1 h-8 w-36"
                  value={bindingName}
                  onChange={(e) => setBindingName(e.target.value)}
                />
              </label>
              <label className="text-[11px] text-muted">
                {t('providers.syncDirection')}
                <select
                  className="mt-1 h-8 rounded-md border border-line2 bg-panel px-2 text-[11px]"
                  value={bindingDirection}
                  onChange={(e) => setBindingDirection(e.target.value as 'read' | 'read_write')}
                >
                  <option value="read_write">{t('providers.syncReadWrite')}</option>
                  <option value="read">{t('providers.syncRead')}</option>
                </select>
              </label>
              <label className="text-[11px] text-muted">
                {t('providers.account')}
                <select
                  className="mt-1 h-8 rounded-md border border-line2 bg-panel px-2 text-[11px]"
                  value={bindingAccountID}
                  onChange={(e) => setBindingAccountID(e.target.value)}
                >
                  <option value="">{t('providers.defaultAccount')}</option>
                  {(bindingAccountsData?.items ?? []).map((account) => (
                    <option key={account.id} value={account.id}>
                      {account.login || account.email || account.id}
                    </option>
                  ))}
                </select>
              </label>
              <label className="flex items-center gap-1.5 pb-1.5 text-[11px] text-muted">
                <input type="checkbox" checked={bindingBacklink} onChange={(e) => setBindingBacklink(e.target.checked)} />
                {t('providers.prBacklink')}
              </label>
              <label className="flex items-center gap-1.5 pb-1.5 text-[11px] text-muted">
                <input type="checkbox" checked={bindingTrusted} onChange={(e) => setBindingTrusted(e.target.checked)} />
                {t('providers.trustedAuthorsOnly')}
              </label>
              <Button type="submit" size="sm" variant="primary" disabled={!bindingExternal.trim() || addBinding.isPending}>
                {t('common.add')}
              </Button>
            </form>
            {addBinding.error && <p className="text-[11.5px] text-err">{String(addBinding.error.message)}</p>}
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={!!reauthFor} onOpenChange={(open) => !open && setReauthFor(null)}>
        <DialogContent
          title={`${t('providers.reconnect')} · ${reauthFor?.name ?? ''}`}
          description={t('providers.reconnectHint')}
        >
          <form
            className="flex flex-col gap-3"
            onSubmit={(e) => {
              e.preventDefault()
              reconnect.mutate()
            }}
          >
            <label className="text-[12px] text-muted">
              {t('providers.token')}
              <Input
                className="mt-1"
                type="password"
                autoFocus
                value={reauthToken}
                onChange={(e) => setReauthToken(e.target.value)}
              />
            </label>
            {reconnect.error && <p className="text-[12px] text-err">{String(reconnect.error.message)}</p>}
            <div className="flex justify-end gap-2">
              <Button type="button" variant="ghost" onClick={() => setReauthFor(null)}>
                {t('common.cancel')}
              </Button>
              <Button type="submit" variant="primary" disabled={!reauthToken || reconnect.isPending}>
                {t('providers.reconnect')}
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>

      <Dialog open={!!editFor} onOpenChange={(open) => !open && setEditFor(null)}>
        <DialogContent
          title={`${t('providers.edit')} · ${editFor?.provider ?? ''}`}
          description={t('providers.config')}
        >
          <form
            className="flex flex-col gap-3"
            onSubmit={(e) => {
              e.preventDefault()
              saveEdit.mutate()
            }}
          >
            <label className="text-[12px] text-muted">
              {t('providers.connectionName')}
              <Input className="mt-1" value={editName} onChange={(e) => setEditName(e.target.value)} />
            </label>
            <label className="text-[12px] text-muted">
              {t('providers.baseUrl')}
              <Input className="mt-1" value={editBaseUrl} onChange={(e) => setEditBaseUrl(e.target.value)} />
            </label>
            <label className="text-[12px] text-muted">
              {t('providers.instance')}
              <Input className="mt-1" value={editInstance} onChange={(e) => setEditInstance(e.target.value)} />
            </label>
            <label className="flex items-center gap-2 text-[12px] text-muted">
              <input type="checkbox" checked={editEnabled} onChange={(e) => setEditEnabled(e.target.checked)} />
              {t('providers.enabled')}
            </label>
            {saveEdit.error && <p className="text-[12px] text-err">{String(saveEdit.error.message)}</p>}
            <div className="flex justify-end gap-2">
              <Button type="button" variant="ghost" onClick={() => setEditFor(null)}>
                {t('common.cancel')}
              </Button>
              <Button type="submit" variant="primary" disabled={saveEdit.isPending}>
                {t('providers.save')}
              </Button>
            </div>
          </form>
        </DialogContent>
      </Dialog>
      <OAuthAppsDialog open={oauthOpen} onOpenChange={setOauthOpen} />
    </div>
  )
}

function ConnectionDeliveries({ automationId }: { automationId: string }) {
  const qc = useQueryClient()
  const { data } = useQuery({
    queryKey: ['webhook-deliveries', automationId],
    queryFn: () => api.webhookDeliveries(automationId),
    refetchInterval: 15000,
  })
  const replay = useMutation({
    mutationFn: (id: string) => api.replayWebhookDelivery(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['webhook-deliveries', automationId] }),
  })
  const deliveries = data?.items ?? []
  if (!deliveries.length) return null
  return (
    <div className="mt-2 rounded border border-line bg-panel2/40 p-2">
      <p className="text-[10.5px] uppercase tracking-wider text-muted/70">Webhook deliveries</p>
      {deliveries.slice(0, 5).map((item) => (
        <div key={item.id} className="mt-1 flex items-center gap-2 text-[10.5px]">
          <span
            className={
              item.status === 'accepted'
                ? 'text-ok'
                : item.status === 'error'
                  ? 'text-err'
                  : 'text-muted'
            }
          >
            {item.status}
          </span>
          <span className="text-muted">{item.kind}</span>
          <span className="truncate text-muted/70">{item.reason}</span>
          <span className="ml-auto font-mono text-[10px] text-muted/60">
            {new Date(item.createdAt).toLocaleTimeString()}
          </span>
          <button
            className="text-accent2 hover:underline"
            onClick={() => replay.mutate(item.id)}
          >
            replay
          </button>
        </div>
      ))}
    </div>
  )
}
function ConnectionSyncStatus({ connectionId }: { connectionId: string }) {
  const { data } = useQuery({
    queryKey: ['sync-status', connectionId],
    queryFn: () => api.connectionSyncStatus(connectionId),
    refetchInterval: 30000,
  })
  const cursor = data?.cursor
  if (!cursor) return null
  return (
    <div className="mt-1 flex flex-wrap items-center gap-1 text-[10.5px] text-muted">
      <span>同步</span>
      <span>{cursor.lastSyncedAt ? new Date(cursor.lastSyncedAt).toLocaleString() : '尚未同步'}</span>
      {cursor.lastError ? (
        <span className="text-err">· {cursor.lastError}</span>
      ) : (
        <span className="text-ok">· 正常</span>
      )}
    </div>
  )
}