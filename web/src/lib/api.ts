const JSON_HEADERS = { 'Content-Type': 'application/json' }

async function request<T>(url: string, init?: RequestInit): Promise<T> {
  const headers = new Headers(init?.headers)
  const workspace = localStorage.getItem('orca.workspace')
  if (workspace) headers.set('X-Tandemesh-Workspace', workspace)
  const resp = await fetch(url, { ...init, headers })
  if (!resp.ok) {
    const text = await resp.text()
    throw new Error(text || `HTTP ${resp.status}`)
  }
  if (resp.status === 204) return undefined as T
  return (await resp.json()) as T
}

export type RunStatus =
  | 'queued'
  | 'leased'
  | 'preparing'
  | 'running'
  | 'waiting_approval'
  | 'finalizing'
  | 'succeeded'
  | 'failed'
  | 'cancelled'

export interface WorkRef {
  provider: string
  instance: string
  type: string
  id: string
  url: string
  revision: string
}

export interface Run {
  id: string
  work: WorkRef
  agentId: string
  machineId?: string
  status: RunStatus
  priority: number
  attempt: number
  fence: number
  repo?: string
  base?: string
  branch?: string
  prompt?: string
  variantOf?: string
  variantLabel?: string
  taskId?: string
  role?: string
  inputTokens?: number
  outputTokens?: number
  cacheReadTokens?: number
  cacheWriteTokens?: number
  costUsd?: number
  usageEstimated?: boolean
  model?: string
  errorCode?: string
  errorMessage?: string
  createdAt: string
  startedAt?: string
  endedAt?: string
  leaseExpiresAt?: string
}

export interface RunEvent {
  runId: string
  seq: number
  ts: string
  type: string
  payload?: Record<string, unknown>
}

export interface Approval {
  id: string
  runId: string
  tool: string
  command: string
  risk: string
  status: string
  requestedAt: string
  expiresAt: string
}

export interface AgentQuestion {
  id: string
  sessionId: string
  runId: string
  question: string
  options?: string[]
  status: string
  answer?: string
  createdAt: string
  answeredAt?: string
  expiresAt?: string
}

export interface Task {
  id: string
  workspaceId: string
  provider: string
  instance: string
  workType: string
  workId: string
  url: string
  revision: string
  title: string
  body: string
  labels: string[]
  author: string
  assignee: string
  assigneeMemberId?: string
  origin?: string
  fields?: Record<string, string[]>
  assigneeAgentId: string
  assigneeGroupId?: string
  status: string
  executionState?: string
  inboxState: string
  inboxReadAt?: string
  iterationId?: string
  parentId?: string
  doneAt?: string
  commentCount: number
  lastComment?: string
  lastCommentBy?: string
  runCount: number
  activeRuns: number
  activeAgentId?: string
  failedRuns: number
  succeededRuns: number
  readyToClose: boolean
  createdAt: string
  updatedAt: string
  syncedAt: string
}

export interface Dispatch {
  id: string
  taskId: string
  workspaceId: string
  runId?: string
  agentId?: string
  agentIds?: string[]
  agents?: { agentId: string; prompt?: string; model?: string; role?: string }[]
  attempt: number
  status: string
  outcome?: string
  result?: string
  filesModified?: string[]
  failureCount: number
  createdAt: string
  updatedAt: string
  endedAt?: string
}

export interface AgentSession {
  id: string
  workspaceId: string
  taskId?: string
  agentId: string
  backend?: string
  model?: string
  reasoning?: string
  createdBy?: string
  mode: string
  title: string
  repo?: string
  base?: string
  worktreeId?: string
  status: string
  lastRunStatus?: string
  backendSessionId?: string
  lastRunId?: string
  createdAt: string
  updatedAt: string
  closedAt?: string
}

export interface SessionGitStatus {
  worktreeId?: string
  branch: string
  clean: boolean
  changes: string[]
  lastCommit?: string
  remote?: boolean
  commits?: { hash: string; subject: string; author: string; at: string }[]
}

export interface PRState {
  url: string
  state: string
  draft: boolean
  title?: string
}

export interface GitJob {
  id: string
  sessionId: string
  worktreeId: string
  machineId: string
  path: string
  action: string
  message: string
  status: string
  resultJson?: string
  error?: string
}

export interface SessionComment {
  id: string
  sessionId: string
  file: string
  line: number
  side: string
  body: string
  status: string
  createdAt: string
}

export interface HistoryEntry {
  id: string
  agent: string
  title: string
  cwd: string
  branch?: string
  model?: string
  preview?: string
  path?: string
  messageCount: number
  updatedAt: string
  machineId?: string
  machineName?: string
}

export interface SessionMessage {
  id?: string
  runId: string
  role: string
  kind: string
  text: string
  ts: number
}

export interface TerminalEvent {
  terminalId: string
  seq: number
  data: string
  createdAt: string
}

export interface TerminalSession {
  id: string
  workspaceId: string
  agentSessionId?: string
  machineId?: string
  shell: string
  cwd?: string
  cols: number
  rows: number
  status: string
  exitCode?: number
  createdAt: string
  lastSeenAt?: string
}
export interface WorktreeCleanup {
  id: string
  worktreeId: string
  machineId: string
  status: string
  error?: string
  createdAt: string
  completedAt?: string
}

export interface Worktree {
  id: string
  workspaceId: string
  taskId: string
  runId: string
  repo: string
  path: string
  branch: string
  baseRef: string
  startRef: string
  status: string
  checkpointComment?: string
  workspaceStatus?: string
  createdAt: string
  updatedAt: string
  archivedAt?: string
}

export interface AgentMessage {
  id: string
  workspaceId: string
  taskId: string
  runId?: string
  dispatchId?: string
  sender: string
  recipient: string
  type: string
  subject: string
  body: string
  status: string
  createdAt: string
  handledAt?: string
}

export interface DecisionGate {
  id: string
  taskId: string
  dispatchId?: string
  question: string
  options: string[]
  resolution?: string
  status: string
  createdAt: string
  resolvedAt?: string
}

export interface DiffAnnotation {
  id: string
  workspaceId: string
  taskId: string
  runId?: string
  filePath: string
  line: number
  side: string
  body: string
  author: string
  status: string
  batchId?: string
  createdAt: string
  sentAt?: string
  resolvedAt?: string
}

export interface Workspace {
  id: string
  name: string
  slug: string
  createdAt: string
}

export interface Iteration {
  id: string
  workspaceId: string
  name: string
  startDate: string
  endDate: string
  status: string
  taskCount: number
  doneCount: number
  createdAt: string
}

export interface BurndownPoint {
  date: string
  remaining: number
  total: number
}

export interface IterationStats {
  total: number
  done: number
  open: number
  inProgress: number
}

export interface SavedView {
  id: string
  workspaceId: string
  name: string
  filters: Record<string, string>
  createdAt: string
}

export interface Group {
  id: string
  workspaceId: string
  name: string
  description: string
  createdAt: string
}

export interface GroupMember {
  groupId: string
  agentId: string
  role: string
}

export interface WorkItem {
  ref: {
    provider: string
    instance: string
    type: string
    id: string
    url: string
    revision: string
  }
  title: string
  body?: string
  labels?: string[]
  assignee?: string
  updated: string
}

export interface RuntimeInfo {
  id: string
  name: string
  install: string
  docs: string
  acp: boolean
}

export interface Machine {
  id: string
  name: string
  labels: string[]
  agents: string[]
  runtimes?: Record<string, string>
  concurrency: number
  status: string
  version?: string
  lastSeen: string
  metrics?: MachineMetrics
}

export interface MachineMetrics {
  cpuPercent: number
  memoryUsedBytes: number
  memoryTotalBytes: number
  diskUsedBytes: number
  diskTotalBytes: number
  collectedAt: string
}

export interface Skill {
  id: string
  workspaceId: string
  name: string
  description: string
  version?: string
  source: string
  sourceUrl?: string
  tags?: string[]
  enabled: boolean
  checksum?: string
  body: string
  createdAt: string
  updatedAt: string
}

export interface MarketSkill {
  id: string
  name: string
  description: string
  version: string
  author: string
  tags: string[]
  body: string
  installed: boolean
  source?: string
  sourceUrl?: string
}

export interface TaskComment {
  id: string
  taskId: string
  author: string
  body: string
  createdAt: string
}

export interface TaskPullRequest {
  id: string
  taskId: string
  workspaceId: string
  provider: string
  repo: string
  number: number
  url: string
  title: string
  state: string
  branch?: string
  author?: string
  createdAt: string
  updatedAt: string
}

export interface ImportBatch {
  id: string
  workspaceId: string
  provider: string
  connectionId: string
  query: string
  status: string
  imported: number
  nextStartAt: number
  failed: number
  error?: string
  createdAt: string
  finishedAt?: string
}
export interface FieldMetadata {
  id: string
  name: string
  type?: string
  required: boolean
  allowed?: string[]
}
export interface TaskAttachment {
  id: string
  taskId: string
  provider: string
  externalId: string
  name: string
  url: string
  mime?: string
  size?: number
  author?: string
  createdAt: string
}
export interface TaskCommit {
  id: string
  taskId: string
  workspaceId: string
  provider: string
  repo: string
  sha: string
  url: string
  message: string
  author?: string
  createdAt: string
}

export interface ExternalComment {
  id: string
  taskId: string
  provider: string
  externalId: string
  author: string
  body: string
  threadId?: string
  parentExternalId?: string
  authorExternalId?: string
  authorMemberId?: string
  direction?: 'inbound' | 'outbound'
  createdAt: string
}

export interface CommentThread {
  id: string
  workspaceId: string
  taskId: string
  provider: string
  externalId: string
  source: string
  resolved: boolean
  createdAt: string
  updatedAt: string
}

export interface TaskNote {
  id: string
  taskId: string
  author: string
  body: string
  createdAt: string
}

export interface TaskActivityItem {
  source: 'task' | 'run' | string
  type: string
  ts: string
  actor?: string
  runId?: string
  agentId?: string
  agentIds?: string[]
  agents?: { agentId: string; prompt?: string; model?: string; role?: string }[]
  seq?: number
  payload?: Record<string, unknown>
}

export interface ProviderConfigField {
  key: string
  label?: string
  labelKey?: string
  type?: 'text' | 'password' | 'url' | 'textarea' | 'select'
  placeholder?: string
  help?: string
  helpKey?: string
  required?: boolean
  secret?: boolean
  options?: string[]
}

export interface ProviderAuthOption {
  kind: string
  labelKey?: string
  default?: boolean
  fields?: ProviderConfigField[]
}

export interface Provider {
  id: string
  name: string
  kind: 'task' | 'chat' | 'doc' | 'email' | 'agent'
  auth: string
  authOptions?: ProviderAuthOption[]
  capabilities: string[]
  configFields?: ProviderConfigField[]
}

export interface CredentialMeta {
  provider: string
  subject: string
  kind: string
  expiresAt?: string
  updatedAt: string
}

export interface Automation {
  id: string
  name: string
  schedule: string
  agentId: string
  repo: string
  base: string
  prompt: string
  enabled: boolean
  precheck?: string
  missedGraceSeconds: number
  sessionId?: string
  reuseSession: boolean
  lastStatus?: string
  lastError?: string
  nextRun?: string
  lastRun?: string
  createdAt: string
}

export interface WorkflowNode {
  id: string
  kind: 'script' | 'parallel' | 'fanout' | 'merge' | 'gate' | 'agent' | 'workflow' | 'condition'
  name: string
  agentId?: string
  agentIds?: string[]
  agents?: { agentId: string; prompt?: string; model?: string; role?: string }[]
  role?: string
  model?: string
  workflowId?: string
  prompt?: string
  command?: string
  commands?: string[]
  when?: string
  next?: string
  needs?: string[]
  position?: { x: number; y: number }
  onFail?: string
  retries?: number
  retryDelaySeconds?: number
  timeoutSeconds?: number
}

export interface UsageRow {
  agentId: string
  model: string
  runs: number
  inputTokens: number
  outputTokens: number
  cacheReadTokens: number
  cacheWriteTokens: number
}

export interface UsageSummary extends UsageRow {
  from: number
  to: number
  items: UsageRow[]
}
export interface ACPRegistryEntry {
  id: string
  name: string
  version: string
  description?: string
  repository?: string
  website?: string
  license?: string
}
export interface WorkflowTemplate {
  id: string
  name: string
  description: string
  nodes: WorkflowNode[]
}

export interface Workflow {
  id: string
  workspaceId: string
  name: string
  description: string
  nodes: WorkflowNode[]
  createdAt: string
  updatedAt: string
}

export interface WorkflowArtifact {
  name: string
  path: string
  size: number
  content?: string
}

export interface WorkflowNodeRun {
  nodeId: string
  kind: string
  name: string
  status: string
  output?: string
  error?: string
  runId?: string
  childRunIds?: string[]
  artifacts?: WorkflowArtifact[]
  startedAt?: string
  finishedAt?: string
}

export interface WorkflowRun {
  id: string
  workflowId: string
  workspaceId: string
  taskId?: string
  status: string
  currentNode?: string
  nodes: WorkflowNodeRun[]
  error?: string
  createdAt: string
  updatedAt: string
}
export interface UnifiedAutomation {
  id: string
  kind: 'schedule' | 'event'
  name: string
  enabled: boolean
  agentId: string
  triggerKind: string
  triggerValue?: string
  provider?: string
  instance?: string
  schedule?: string
  prompt?: string
  repo?: string
  nextRun?: string
  lastRun?: string
  lastStatus?: string
  lastError?: string
}

export interface SessionShare {
  token: string
  sessionId: string
  createdBy?: string
  createdAt: string
  expiresAt?: string
}

export interface AuditLog {
  id: string
  workspaceId: string
  actor: string
  action: string
  resource: string
  detail?: string
  createdAt: string
}

export interface Agent {
  id: string
  workspaceId: string
  workspaceIds?: string[]
  machineId?: string
  name: string
  instructions: string
  backend: string
  role?: string
  capabilities?: string[]
  model: string
  skills?: string[]
  repos?: string[]
  sessionMode?: 'off' | 'task' | 'agent'
  tools?: Record<string, unknown>
}

export interface Document {
  id: string
  workspaceId: string
  title: string
  body: string
  tags?: string[]
  createdBy?: string
  createdAt: string
  updatedAt: string
}
export interface Notification {
  id: string
  workspaceId: string
  type: string
  title: string
  body?: string
  link?: string
  readAt?: string
  createdAt: string
}
export interface AssignmentRule {
  id: string
  workspaceId: string
  name: string
  provider: string
  instance: string
  label: string
  author: string
  assignee: string
  agentId: string
  repo: string
  autoDispatch: boolean
  enabled: boolean
  priority: number
  createdAt: string
}

export interface List<T> {
  items: T[]
}

export interface Connection {
  id: string
  workspaceId: string
  provider: string
  name: string
  status: 'pending' | 'connected' | 'needs_reauth' | 'revoked'
  authMethod: 'oauth' | 'access_token'
  syncDirection: 'read' | 'read_write'
  instance: string
  baseUrl: string
  accountLogin: string
  executionMode: string
  runnerMachineId: string
  enabled: boolean
  automationId?: string
  transition?: string
  agentId?: string
  agentIds?: string[]
  agents?: { agentId: string; prompt?: string; model?: string; role?: string }[]
  createdAt: string
  updatedAt: string
  bindings?: { id: string; externalId: string; name: string; kind: string; enabled: boolean }[]
  allBindings?: { id: string; externalId: string; name: string; kind: string; enabled: boolean }[]
}

export interface ConnectionAccount {
  id: string
  workspaceId: string
  connectionId: string
  provider: string
  kind: 'org' | 'user' | 'bot'
  login: string
  email?: string
  externalId?: string
  credentialSubject: string
  isDefault: boolean
  createdAt: string
  updatedAt: string
}

export interface ExternalIdentity {
  id: string
  workspaceId: string
  provider: string
  externalId: string
  login: string
  email?: string
  memberId: string
  createdAt: string
  updatedAt: string
}

export interface ProviderUser {
  id: string
  login: string
  name?: string
  email?: string
}

export interface MemberIdentity {
  id: string
  workspaceId: string
  memberId: string
  provider: string
  externalId: string
  login: string
  email?: string
  credentialSubject: string
  createdAt: string
  updatedAt: string
}

export interface ResourceBinding {
  id: string
  workspaceId: string
  connectionId: string
  provider: string
  externalId: string
  name: string
  kind: string
  direction: 'read' | 'read_write'
  settings?: Record<string, string>
  enabled: boolean
  lastSyncedAt?: string
  lastError?: string
  createdAt: string
  updatedAt: string
}

export const api = {
  runDiff: (runId: string) => request<{ runId: string; diff: string; truncated: boolean }>(`/api/v1/runs/${runId}/diff`),
  runTranscript: (id: string, before?: number) =>
    request<{ items: { kind: string; text: string; tsMs: number; seq: number }[]; runId: string }>(`/api/v1/runs/${id}/transcript${before ? `?before=${before}` : ''}`),

  runs: (workspace: string, status?: string) => {
    const query = new URLSearchParams({ workspace })
    if (status) query.set('status', status)
    return request<List<Run>>(`/api/v1/runs?${query.toString()}`)
  },
  run: (id: string) => request<Run>(`/api/v1/runs/${id}`),
  createTask: (body: { title: string; body?: string; workspaceId?: string; labels?: string[]; agentId?: string; memberId?: string }) =>
    request<Task>('/api/v1/tasks', { method: 'POST', body: JSON.stringify(body) }),
  tasks: (params?: {
    workspace?: string
    status?: string
    provider?: string
    label?: string
    agent?: string
    inbox?: string
    assignee?: string
    member?: string
    iteration?: string
    unassigned?: boolean
  }) => {
    const query = new URLSearchParams()
    if (params?.workspace) query.set('workspace', params.workspace)
    if (params?.inbox) query.set('inbox', params.inbox)
    if (params?.assignee) query.set('assignee', params.assignee)
    if (params?.member) query.set('member', params.member)
    if (params?.iteration) query.set('iteration', params.iteration)
    if (params?.status) query.set('status', params.status)
    if (params?.provider) query.set('provider', params.provider)
    if (params?.label) query.set('label', params.label)
    if (params?.agent) query.set('agent', params.agent)
    if (params?.unassigned) query.set('unassigned', '1')
    const suffix = query.toString() ? `?${query.toString()}` : ''
    return request<List<Task>>(`/api/v1/tasks${suffix}`)
  },
  task: (id: string) => request<{ task: Task; runs: Run[] }>(`/api/v1/tasks/${id}`),
  documents: (workspaceId: string) =>
    request<{ documents: Document[] }>(`/api/v1/documents?workspaceId=${encodeURIComponent(workspaceId)}`),
  createDocument: (body: { workspaceId: string; title: string; body: string; tags?: string[] }) =>
    request<Document>('/api/v1/documents', { method: 'POST', headers: JSON_HEADERS, body: JSON.stringify(body) }),
  updateDocument: (id: string, body: { title?: string; body: string; tags?: string[] }) =>
    request<Document>(`/api/v1/documents/${id}`, { method: 'PUT', body: JSON.stringify(body) }),
  deleteDocument: (id: string) =>
    request<{ status: string }>(`/api/v1/documents/${id}`, { method: 'DELETE' }),  notifications: (workspaceId: string, unread = false) =>
    request<{ notifications: Notification[] }>(
      `/api/v1/notifications?workspaceId=${encodeURIComponent(workspaceId)}${unread ? '&unread=1' : ''}`,
    ),
  markNotificationRead: (id: string) =>
    request<{ status: string }>(`/api/v1/notifications/${id}/read`, { method: 'POST' }),
  markAllNotificationsRead: (workspaceId: string) =>
    request<{ status: string }>(
      `/api/v1/notifications/read-all?workspaceId=${encodeURIComponent(workspaceId)}`,
      { method: 'POST' },
    ),
  listPlatformComments: (id: string) =>
    request<{ items: ExternalComment[] }>(`/api/v1/tasks/${id}/platform-comments`),
  commentThreads: (id: string) =>
    request<{ items: CommentThread[]; comments: Record<string, ExternalComment[]> }>(`/api/v1/tasks/${id}/comment-threads`),
  resolveCommentThread: (taskId: string, threadId: string, resolved: boolean) =>
    request<{ resolved: boolean }>(`/api/v1/tasks/${taskId}/comment-threads/${threadId}`, {
      method: 'PATCH',
      headers: JSON_HEADERS,
      body: JSON.stringify({ resolved }),
    }),
  addPlatformComment: (id: string, body: string, threadId?: string) =>
    request<{ comment: ExternalComment; url: string }>(`/api/v1/tasks/${id}/platform-comments`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify({ body, threadId }),
    }),
  updateTaskFields: (id: string, body: { priority?: string; components?: string[]; fixVersions?: string[]; customFields?: Record<string, string> }) =>
    request<{ status: string }>(`/api/v1/tasks/${id}/fields`, {
      method: 'PATCH',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),  assignLocalMember: (id: string, memberId: string, login?: string, autoDispatch?: boolean) =>
    request<Task>(`/api/v1/tasks/${id}/member-assignee`, { method: 'PATCH', body: JSON.stringify({ memberId, login, autoDispatch }) }),
  assignTaskMember: (id: string, memberId: string) =>
    request<{ assignee: string }>(`/api/v1/tasks/${id}/assignee`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify({ memberId }),
    }),
  taskPullRequests: (id: string) =>
    request<{ items: TaskPullRequest[] }>(`/api/v1/tasks/${id}/pull-requests`),
  syncTaskPullRequests: (id: string) =>
    request<{ items: TaskPullRequest[]; linked: number }>(`/api/v1/tasks/${id}/pull-requests/sync`, { method: 'POST' }),
  taskAttachments: (id: string) =>
    request<{ items: TaskAttachment[] }>(`/api/v1/tasks/${id}/attachments`),
  taskFieldOptions: (id: string) =>
    request<{ options: Record<string, string[]> }>(`/api/v1/tasks/${id}/field-options`),  taskFieldMetadata: (id: string) =>
    request<{ fields: Record<string, FieldMetadata> }>(`/api/v1/tasks/${id}/field-metadata`),  taskCommits: (id: string) =>
    request<{ items: TaskCommit[] }>(`/api/v1/tasks/${id}/commits`),
  syncTaskCommits: (id: string) =>
    request<{ items: TaskCommit[]; linked: number }>(`/api/v1/tasks/${id}/commits/sync`, { method: 'POST' }),
  deleteTaskPullRequest: (id: string) =>
    request<void>(`/api/v1/pull-requests/${id}`, { method: 'DELETE' }),
  taskActivity: (id: string, limit = 100, offset?: number) => {
    const query = new URLSearchParams({ limit: String(limit) })
    if (offset !== undefined) query.set('offset', String(offset))
    return request<{
      items: TaskActivityItem[]
      total: number
      offset: number
      hasMore: boolean
      task: Task
    }>(`/api/v1/tasks/${id}/activity?${query.toString()}`)
  },
  patchTask: (id: string, status: string) =>
    request<Task>(`/api/v1/tasks/${id}`, {
      method: 'PATCH',
      headers: JSON_HEADERS,
      body: JSON.stringify({ status }),
    }),
  setTaskInbox: (id: string, state: 'inbox' | 'later' | 'cleared') =>
    request<Task>(`/api/v1/tasks/${id}/inbox`, {
      method: 'PATCH',
      headers: JSON_HEADERS,
      body: JSON.stringify({ state }),
    }),
  markTaskRead: (id: string) => request<void>(`/api/v1/tasks/${id}/read`, { method: 'POST' }),
  me: () => request<{ name: string }>('/api/v1/me'),
  search: (q: string, workspace: string) =>
    request<{ tasks: Task[]; groups: Group[] }>(
      `/api/v1/search?q=${encodeURIComponent(q)}&workspace=${encodeURIComponent(workspace)}`,
    ),
  sessions: (workspaceId: string, taskId?: string, status?: string) =>
    request<{ sessions: AgentSession[] }>(`/api/v1/sessions?workspaceId=${encodeURIComponent(workspaceId)}${taskId ? `&taskId=${encodeURIComponent(taskId)}` : ''}${status ? `&status=${encodeURIComponent(status)}` : ''}`),
  createSession: (body: { workspaceId: string; taskId?: string; agentId?: string; backend?: string; model?: string; mode?: string; title?: string; repo?: string; base?: string; worktreeId?: string; reasoning?: string }) =>
    request<AgentSession>('/api/v1/sessions', { method: 'POST', headers: JSON_HEADERS, body: JSON.stringify(body) }),
  session: (id: string) => request<{ session: AgentSession; runCount: number }>(`/api/v1/sessions/${id}`),
  createSessionShare: (id: string, ttlHours = 168) =>
    request<SessionShare>(`/api/v1/sessions/${id}/share`, { method: 'POST', headers: JSON_HEADERS, body: JSON.stringify({ ttlHours }) }),
  revokeSessionShare: (id: string, token: string) =>
    request<{ status: string }>(`/api/v1/sessions/${id}/share/${token}`, { method: 'DELETE' }),
  sharedSession: (token: string) =>
    request<{ session: { title: string; repo?: string; backend?: string; model?: string; createdAt: string }; messages: SessionMessage[] }>(`/api/v1/shared/sessions/${token}`),
  auditLogs: (workspaceId: string) =>
    request<{ auditLogs: AuditLog[] }>(`/api/v1/audit-logs?workspaceId=${encodeURIComponent(workspaceId)}`),
  usageTrend: (workspaceId: string, days = 14) =>
    request<{ points: { date: string; tokens: number; cost: number; runs: number }[]; totalTokens: number; totalCost: number; totalRuns: number }>(`/api/v1/usage/trend?workspaceId=${encodeURIComponent(workspaceId)}&days=${days}`),
  sessionHistory: (query?: string) =>
    request<{ items: HistoryEntry[]; host: string }>(`/api/v1/session-history?all=1${query ? `&query=${encodeURIComponent(query)}` : ''}`),
  scanHistory: () =>
    request<{ queued: number }>('/api/v1/session-history/scan', { method: 'POST' }),
  resumeHistory: (body: { workspaceId: string; sessionId: string; title?: string; repo?: string; model?: string; agentId?: string; machineId?: string }) =>
    request<AgentSession>('/api/v1/session-history/resume', {
      method: 'POST', headers: JSON_HEADERS, body: JSON.stringify(body),
    }),
  closeSession: (id: string) => request<AgentSession>(`/api/v1/sessions/${id}/close`, { method: 'POST' }),
  renameSession: (id: string, title: string) =>
    request<AgentSession>(`/api/v1/sessions/${id}`, { method: 'PATCH', headers: JSON_HEADERS, body: JSON.stringify({ title }) }),
  archiveSession: (id: string) =>
    request<AgentSession>(`/api/v1/sessions/${id}/archive`, { method: 'POST' }),
  stopSession: (id: string) => request<{ stopped: number }>(`/api/v1/sessions/${id}/stop`, { method: 'POST' }),
  sessionRuns: (id: string) => request<{ items: Run[] }>(`/api/v1/sessions/${id}/runs`),
  createSessionRun: (id: string, body: { prompt: string; repo?: string; base?: string; skipUserMessage?: boolean }) =>
    request<Run>(`/api/v1/sessions/${id}/runs`, { method: 'POST', headers: JSON_HEADERS, body: JSON.stringify(body) }),
  regenerateSession: (id: string, runId?: string) =>
    request<Run>(`/api/v1/sessions/${id}/regenerate`, {
      method: 'POST', headers: JSON_HEADERS, body: JSON.stringify({ runId: runId ?? '' }),
    }),
  editSessionMessage: (id: string, messageId: string, text: string) =>
    request<{ status: string }>(`/api/v1/sessions/${id}/messages/${messageId}`, { method: 'PATCH', headers: JSON_HEADERS, body: JSON.stringify({ text }) }),
  sessionMessages: (id: string) => request<{ messages: SessionMessage[] }>(`/api/v1/sessions/${id}/messages`),
  sessionDiff: (id: string) =>
    request<{ runId?: string; diff: string; truncated: boolean }>(`/api/v1/sessions/${id}/diff`),
  sessionComments: (id: string) =>
    request<{ items: SessionComment[] }>(`/api/v1/sessions/${id}/comments`),
  createSessionComment: (id: string, body: { file: string; line: number; side?: string; body: string }) =>
    request<SessionComment>(`/api/v1/sessions/${id}/comments`, {
      method: 'POST', headers: JSON_HEADERS, body: JSON.stringify(body),
    }),
  sendSessionComments: (id: string) =>
    request<Run>(`/api/v1/sessions/${id}/comments/send`, { method: 'POST' }),
  updateSessionComment: (id: string, body: string) =>
    request<SessionComment>(`/api/v1/session-comments/${id}`, { method: 'PATCH', headers: JSON_HEADERS, body: JSON.stringify({ body }) }),
  deleteSessionComment: (id: string) =>
    request<{ status: string }>(`/api/v1/session-comments/${id}`, { method: 'DELETE' }),
  resolveSessionComment: (id: string) =>
    request<SessionComment>(`/api/v1/session-comments/${id}/resolve`, { method: 'POST' }),
  sessionQuestions: (id: string, status = 'pending') =>
    request<{ items: AgentQuestion[] }>(`/api/v1/sessions/${id}/questions?status=${encodeURIComponent(status)}`),
  answerQuestion: (id: string, answer: string) =>
    request<AgentQuestion>(`/api/v1/questions/${id}/answer`, {
      method: 'POST', headers: JSON_HEADERS, body: JSON.stringify({ answer }),
    }),
  sessionGit: (id: string) => request<SessionGitStatus>(`/api/v1/sessions/${id}/git`),
  shipSession: (id: string, message: string) =>
    request<SessionGitStatus | { status: 'queued'; jobId: string; machineId: string }>(`/api/v1/sessions/${id}/git/ship`, {
      method: 'POST', headers: JSON_HEADERS, body: JSON.stringify({ message }),
    }),
  createSessionPR: (id: string) =>
    request<{ url: string } | { status: 'queued'; jobId: string; machineId: string }>(`/api/v1/sessions/${id}/git/pr`, { method: 'POST' }),
  sessionPRStatus: (id: string) =>
    request<PRState>(`/api/v1/sessions/${id}/git/pr`),
  refreshSessionPR: (id: string) =>
    request<PRState | { status: 'queued'; jobId: string; machineId: string }>(`/api/v1/sessions/${id}/git/pr/refresh`, { method: 'POST' }),
  sessionGitJob: (id: string, jobId: string) =>
    request<GitJob>(`/api/v1/sessions/${id}/git/jobs/${jobId}`),
  listTerminalSessions: (workspace: string) =>
    request<{ items: TerminalSession[] }>(`/api/v1/terminal-sessions?workspace=${encodeURIComponent(workspace)}`),
  createTerminalSession: (body: { workspaceId: string; agentSessionId?: string; machineId?: string; shell?: string; cwd?: string; cols?: number; rows?: number }) =>
    request<TerminalSession>('/api/v1/terminal-sessions', { method: 'POST', headers: JSON_HEADERS, body: JSON.stringify(body) }),
  terminalSession: (id: string) => request<TerminalSession>(`/api/v1/terminal-sessions/${id}`),
  terminalInput: (id: string, data: string) =>
    request<void>(`/api/v1/terminal-sessions/${id}/input`, { method: 'POST', headers: JSON_HEADERS, body: JSON.stringify({ data }) }),
  terminalOutput: (id: string, after: number) =>
    request<{ terminal: TerminalSession; events: TerminalEvent[] }>(`/api/v1/terminal-sessions/${id}/output?after=${after}`),
  closeTerminalSession: (id: string) =>
    request<{ status: string }>(`/api/v1/terminal-sessions/${id}/close`, { method: 'POST' }),
  continueTerminalChat: (id: string, body: { title?: string; agentId?: string } = {}) =>
    request<AgentSession>(`/api/v1/terminal-sessions/${id}/continue-chat`, {
      method: 'POST', headers: JSON_HEADERS, body: JSON.stringify(body),
    }),
  worktrees: (params: { workspaceId?: string; taskId?: string; repo?: string; status?: string }) => {
    const q = new URLSearchParams()
    if (params.workspaceId) q.set('workspaceId', params.workspaceId)
    if (params.taskId) q.set('taskId', params.taskId)
    if (params.repo) q.set('repo', params.repo)
    if (params.status) q.set('status', params.status)
    return request<{ worktrees: Worktree[] }>(`/api/v1/worktrees?${q.toString()}`)
  },
  repositories: (workspaceId: string) =>
    request<{ items: { repo: string; baseRef?: string; branch?: string; path?: string; worktreeCount: number }[] }>(`/api/v1/repositories?workspaceId=${encodeURIComponent(workspaceId)}`),
  worktree: (id: string) => request<Worktree>(`/api/v1/worktrees/${id}`),
  sessionWorktree: (id: string) =>
    request<{ worktree: Worktree; cleanup?: WorktreeCleanup }>(`/api/v1/sessions/${id}/worktree`),
  archiveSessionWorktree: (id: string) =>
    request<Worktree>(`/api/v1/sessions/${id}/worktree/archive`, { method: 'POST' }),
  cleanupSessionWorktree: (id: string) =>
    request<WorktreeCleanup>(`/api/v1/sessions/${id}/worktree/cleanup`, { method: 'POST' }),
  runWorktree: (runId: string) => request<Worktree>(`/api/v1/runs/${runId}/worktree`),
  setWorktreeCheckpoint: (id: string, body: { comment: string; workspaceStatus?: string }) =>
    request<Worktree>(`/api/v1/worktrees/${id}/checkpoint`, {
      method: 'POST', headers: JSON_HEADERS, body: JSON.stringify(body),
    }),
  checkpointRunWorktree: (runId: string, body: { comment: string; workspaceStatus?: string }) =>
    request<Worktree>(`/api/v1/runs/${runId}/worktree/checkpoint`, {
      method: 'POST', headers: JSON_HEADERS, body: JSON.stringify(body),
    }),
  cleanupWorktree: (id: string) => request<{ id: string; status: string }>(`/api/v1/worktrees/${id}/cleanup`, { method: 'POST' }),
  archiveWorktree: (id: string) => request<Worktree>(`/api/v1/worktrees/${id}/archive`, { method: 'POST' }),  listDispatches: (taskId: string) => request<{ dispatches: Dispatch[] }>(`/api/v1/tasks/${taskId}/dispatches`),
  completeDispatch: (id: string, body: { taskId: string; outcome: 'succeeded' | 'failed'; result?: string; filesModified?: string[] }) =>
    request<{ task: Task; promoted?: string[]; duplicate?: boolean }>(`/api/v1/dispatches/${id}/complete`, {
      method: 'POST', headers: JSON_HEADERS, body: JSON.stringify(body),
    }),
  listGates: (taskId: string) => request<{ gates: DecisionGate[] }>(`/api/v1/tasks/${taskId}/gates`),
  createGate: (taskId: string, body: { question: string; options?: string[]; dispatchId?: string }) =>
    request<DecisionGate>(`/api/v1/tasks/${taskId}/gates`, {
      method: 'POST', headers: JSON_HEADERS, body: JSON.stringify(body),
    }),
  resolveGate: (id: string, resolution: string) =>
    request<DecisionGate>(`/api/v1/gates/${id}/resolve`, {
      method: 'POST', headers: JSON_HEADERS, body: JSON.stringify({ resolution }),
    }),
  taskMessages: (taskId: string, runId?: string) =>
    request<{ messages: AgentMessage[] }>(`/api/v1/tasks/${taskId}/messages${runId ? `?runId=${encodeURIComponent(runId)}` : ''}`),
  sendTaskMessage: (taskId: string, body: { to: string; type?: string; subject?: string; body?: string; fromRunId?: string; dispatchId?: string }) =>
    request<{ messages: AgentMessage[] }>(`/api/v1/tasks/${taskId}/messages`, {
      method: 'POST', headers: JSON_HEADERS, body: JSON.stringify(body),
    }),
  checkTaskMessage: (taskId: string, runId: string) =>
    request<{ messages: AgentMessage[] }>(`/api/v1/tasks/${taskId}/messages/check?runId=${encodeURIComponent(runId)}`, { method: 'POST' }),
  ackTaskMessage: (id: string) => request<void>(`/api/v1/messages/${id}/ack`, { method: 'POST' }),
  listAnnotations: (taskId: string, runId?: string) =>
    request<{ annotations: DiffAnnotation[] }>(`/api/v1/tasks/${taskId}/annotations${runId ? `?runId=${encodeURIComponent(runId)}` : ''}`),
  createAnnotation: (taskId: string, body: { runId?: string; filePath: string; line: number; side?: string; body: string }) =>
    request<DiffAnnotation>(`/api/v1/tasks/${taskId}/annotations`, {
      method: 'POST', headers: JSON_HEADERS, body: JSON.stringify(body),
    }),
  sendAnnotations: (taskId: string, body: { runId?: string }) =>
    request<{ batchId: string; count: number; runId: string; prompt: string; message: AgentMessage }>(`/api/v1/tasks/${taskId}/annotations/send`, {
      method: 'POST', headers: JSON_HEADERS, body: JSON.stringify(body),
    }),
  resolveAnnotation: (id: string) => request<DiffAnnotation>(`/api/v1/annotations/${id}/resolve`, { method: 'POST' }),  startTaskRun: (id: string, body: { agentId: string; repo: string; role?: string; prompt?: string }) =>
    request<Run>(`/api/v1/tasks/${id}/runs`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  workspaces: () => request<List<Workspace>>('/api/v1/workspaces'),
  updateWorkspace: (id: string, name: string) =>
    request<Workspace>(`/api/v1/workspaces/${id}`, {
      method: 'PATCH',
      headers: JSON_HEADERS,
      body: JSON.stringify({ name }),
    }),
  createWorkspace: (body: { name: string; slug?: string }) =>
    request<Workspace>('/api/v1/workspaces', {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  savedViews: (id: string) => request<List<SavedView>>(`/api/v1/workspaces/${id}/views`),
  iterations: (id: string) => request<List<Iteration>>(`/api/v1/workspaces/${id}/iterations`),
  iterationBurndown: (id: string) =>
    request<{ iteration: Iteration; points: BurndownPoint[]; stats: IterationStats }>(
      `/api/v1/iterations/${id}/burndown`,
    ),
  createIteration: (
    id: string,
    body: { name: string; startDate: string; endDate: string; status?: string },
  ) =>
    request<Iteration>(`/api/v1/workspaces/${id}/iterations`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  patchIteration: (id: string, status: string) =>
    request<{ status: string }>(`/api/v1/iterations/${id}`, {
      method: 'PATCH',
      headers: JSON_HEADERS,
      body: JSON.stringify({ status }),
    }),
  deleteIteration: (id: string) =>
    request<void>(`/api/v1/iterations/${id}`, { method: 'DELETE' }),
  setTaskIteration: (taskId: string, iterationId: string) =>
    request<Task>(`/api/v1/tasks/${taskId}/iteration`, {
      method: 'PATCH',
      headers: JSON_HEADERS,
      body: JSON.stringify({ iterationId }),
    }),
  setTaskParent: (taskId: string, parentId: string) =>
    request<Task>(`/api/v1/tasks/${taskId}/parent`, {
      method: 'PATCH',
      headers: JSON_HEADERS,
      body: JSON.stringify({ parentId }),
    }),
  taskChildren: (taskId: string) => request<List<Task>>(`/api/v1/tasks/${taskId}/children`),
  taskDependencies: (taskId: string) =>
    request<List<Task>>(`/api/v1/tasks/${taskId}/dependencies`),
  addTaskDependency: (taskId: string, dependsOn: string) =>
    request<void>(`/api/v1/tasks/${taskId}/dependencies`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify({ dependsOn }),
    }),
  removeTaskDependency: (taskId: string, dependsOn: string) =>
    request<void>(`/api/v1/tasks/${taskId}/dependencies/${dependsOn}`, { method: 'DELETE' }),
  createSavedView: (id: string, body: { name: string; filters: Record<string, string> }) =>
    request<SavedView>(`/api/v1/workspaces/${id}/views`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  deleteSavedView: (id: string, view: string) =>
    request<void>(`/api/v1/workspaces/${id}/views/${view}`, { method: 'DELETE' }),
  renameSavedView: (id: string, view: string, name: string) =>
    request<{ name: string }>(`/api/v1/workspaces/${id}/views/${view}`, {
      method: 'PATCH',
      headers: JSON_HEADERS,
      body: JSON.stringify({ name }),
    }),
  groups: (workspace: string) =>
    request<List<Group>>(`/api/v1/groups?workspace=${encodeURIComponent(workspace)}`),
  createGroup: (body: { workspaceId: string; name: string; description?: string }) =>
    request<Group>('/api/v1/groups', {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  group: (id: string) =>
    request<{ group: Group; members: GroupMember[] }>(`/api/v1/groups/${id}`),
  addGroupMember: (id: string, body: { agentId: string; role: string }) =>
    request<GroupMember>(`/api/v1/groups/${id}/members`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),

  work: (provider: string, instance?: string) => {
    const query = new URLSearchParams({ provider })
    if (instance) query.set('instance', instance)
    return request<{ items: WorkItem[]; provider: string }>(`/api/v1/work?${query.toString()}`)
  },
  connectProvider: (body: {
    provider: string
    token: string
    baseUrl?: string
    instance?: string
    agentId?: string
  agentIds?: string[]
  agents?: { agentId: string; prompt?: string; model?: string; role?: string }[]
    name?: string
    workspaceId?: string
    comment?: boolean
    transition?: string
    webhookSecret?: string
    reuseConnectionId?: string
    commentTemplate?: string
    fields?: Record<string, string>
  }) =>
    request<{ provider: string; automation?: UnifiedAutomation; webhookId?: string }>('/api/v1/integrations/connect', {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  githubAppStatus: () =>
    request<{ enabled: boolean }>('/api/v1/integrations/github/app/status'),  githubOAuthStatus: () =>
    request<{ enabled: boolean }>('/api/v1/integrations/github/oauth/status'),
  oauthStatus: () =>
    request<{ providers: string[] }>('/api/v1/integrations/oauth/status'),
  capabilities: () =>
    request<{ profile: string; authMode: string; publicUrl: string; oauthRedirect: string; webhook: boolean; polling: boolean }>('/api/v1/capabilities'),
  catalog: (provider: string, body: { token?: string; baseUrl?: string; workspace?: string; connectionId?: string }) =>
    request<{ workspaces?: { id: string; name: string; slug?: string }[]; projects?: { id: string; name: string; slug?: string }[] }>('/api/v1/integrations/' + provider + '/catalog', { method: 'POST', body: JSON.stringify(body) }),
  oauthApp: (provider: string) =>
    request<{ provider: string; configured: boolean; clientId: string; clientSecret: string; callbackUrl: string }>('/api/v1/integrations/' + provider + '/oauth/config'),
  saveOAuthApp: (provider: string, body: { clientId: string; clientSecret: string }) =>
    request<{ provider: string; configured: boolean; clientId: string; callbackUrl: string }>('/api/v1/integrations/' + provider + '/oauth/config', { method: 'PUT', body: JSON.stringify(body) }),
  deleteOAuthApp: (provider: string) =>
    request<void>('/api/v1/integrations/' + provider + '/oauth/config', { method: 'DELETE' }),
  integrationStates: (body: {
    provider: string
    token: string
    baseUrl?: string
    instance: string
  }) =>
    request<List<{ id: string; name: string; group?: string }>>('/api/v1/integrations/states', {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  assignmentRules: (workspace: string) =>
    request<List<AssignmentRule>>(`/api/v1/assignment-rules?workspace=${workspace}`),
  createAssignmentRule: (body: {
    workspaceId: string
    name: string
    provider?: string
    instance?: string
    label?: string
    agentId?: string
  agentIds?: string[]
  agents?: { agentId: string; prompt?: string; model?: string; role?: string }[]
    groupId?: string
    repo?: string
    autoDispatch: boolean
  }) =>
    request<AssignmentRule>('/api/v1/assignment-rules', {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  deleteAssignmentRule: (id: string) =>
    request<void>(`/api/v1/assignment-rules/${id}`, { method: 'DELETE' }),
  registerWebhook: (id: string) =>
    request<{ webhookId: string; url: string }>(`/api/v1/connections/${id}/register-webhook`, { method: 'POST' }),
  connectionWebhook: (id: string) =>
    request<{ url: string; automationId: string }>(`/api/v1/connections/${id}/webhook`),
  bindings: (connectionId: string) =>
    request<List<ResourceBinding>>(`/api/v1/connections/${connectionId}/bindings`),
  createBinding: (connectionId: string, body: { externalId: string; name?: string; kind?: string; direction?: string; accountId?: string; settings?: Record<string, string> }) =>
    request<{ binding: ResourceBinding }>(`/api/v1/connections/${connectionId}/bindings`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  deleteBinding: (id: string) =>
    request<void>(`/api/v1/bindings/${id}`, { method: 'DELETE' }),
  accounts: (connectionId: string) =>
    request<List<ConnectionAccount>>(`/api/v1/connections/${connectionId}/accounts`),
  createAccount: (connectionId: string, body: { token: string; kind?: string; login?: string; email?: string; externalId?: string; isDefault?: boolean }) =>
    request<{ account: ConnectionAccount }>(`/api/v1/connections/${connectionId}/accounts`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  setDefaultAccount: (id: string) =>
    request<{ account: ConnectionAccount }>(`/api/v1/accounts/${id}`, {
      method: 'PATCH',
      headers: JSON_HEADERS,
      body: JSON.stringify({ isDefault: true }),
    }),
  deleteAccount: (id: string) =>
    request<void>(`/api/v1/accounts/${id}`, { method: 'DELETE' }),
  identities: (workspace: string) =>
    request<List<ExternalIdentity>>(`/api/v1/workspaces/${workspace}/identities`),
  createIdentity: (workspace: string, body: { provider: string; externalId: string; login?: string; email?: string; memberId: string }) =>
    request<{ identity: ExternalIdentity }>(`/api/v1/workspaces/${workspace}/identities`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  linkIdentity: (id: string, memberId: string) =>
    request<{ identity: ExternalIdentity }>(`/api/v1/identities/${id}`, {
      method: 'PATCH',
      headers: JSON_HEADERS,
      body: JSON.stringify({ memberId }),
    }),  deleteIdentity: (id: string) =>
    request<void>(`/api/v1/identities/${id}`, { method: 'DELETE' }),
  memberIdentities: (workspace: string) =>
    request<List<MemberIdentity>>(`/api/v1/workspaces/${workspace}/member-identities`),
  createMemberIdentity: (workspace: string, body: { memberId: string; provider: string; externalId: string; login?: string; email?: string; token?: string }) =>
    request<{ identity: MemberIdentity }>(`/api/v1/workspaces/${workspace}/member-identities`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  deleteMemberIdentity: (id: string) =>
    request<void>(`/api/v1/member-identities/${id}`, { method: 'DELETE' }),
  connectionUsers: (id: string, q: string) =>
    request<{ items: ProviderUser[] }>(`/api/v1/connections/${id}/users?q=${encodeURIComponent(q)}`),
  createJiraImport: (workspace: string, body: { connectionId: string; jql: string; limit?: number; startAt?: number }) =>
    request<{ batch: ImportBatch }>(`/api/v1/workspaces/${workspace}/imports/jira`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  importRuntime: () =>
    request<{ concurrency: number; active: number; queued: number; running: number }>('/api/v1/imports/runtime'),  imports: (workspace: string) => request<{ items: ImportBatch[] }>(`/api/v1/workspaces/${workspace}/imports`),
  cancelImport: (id: string) =>
    request<{ status: string }>(`/api/v1/imports/${id}/cancel`, { method: 'POST' }),  resumeImport: (id: string) =>
    request<{ batch: ImportBatch }>(`/api/v1/imports/${id}/resume`, { method: 'POST' }),  rollbackImport: (id: string) =>
    request<{ deleted: number; skipped: number }>(`/api/v1/imports/${id}/rollback`, { method: 'POST' }),  connectionSyncStatus: (id: string) =>
    request<{ cursor: { lastSyncedAt: string; lastFullAt: string; lastError: string } }>(`/api/v1/connections/${id}/sync-status`),
  webhookDeliveries: (automationId: string) =>
    request<{ items: { id: string; provider: string; kind: string; status: string; reason: string; runId: string; createdAt: string }[] }>('/api/v1/webhook-deliveries/' + automationId),
  replayWebhookDelivery: (id: string) =>
    request<{ status: number; result: string }>('/api/v1/webhook-deliveries/' + id + '/replay', { method: 'POST' }),
  registerBindingWebhook: (connectionId: string, bindingId: string) =>
    request<{ webhookId: string; url: string; bindingId: string }>(
      '/api/v1/connections/' + connectionId + '/bindings/' + bindingId + '/register-webhook',
      { method: 'POST' },
    ),
  mergeConnection: (targetId: string, sourceId: string) =>
    request<{ moved: number }>('/api/v1/connections/' + targetId + '/merge/' + sourceId, { method: 'POST' }),
  connections: (workspace: string) =>
    request<List<Connection>>(`/api/v1/connections?workspace=${workspace}`),
  reconnectConnection: (id: string, body: { token: string; baseUrl?: string }) =>
    request<{ connection: Connection }>(`/api/v1/connections/${id}/reconnect`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  patchConnection: (id: string, body: { name?: string; baseUrl?: string; instance?: string; agentId?: string; enabled?: boolean; transition?: string; stateMap?: Record<string, string> }) =>
    request<{ connection: Connection }>(`/api/v1/connections/${id}`, {
      method: 'PATCH',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  connectionStates: (id: string) =>
    request<List<{ id: string; name: string; group?: string }>>(`/api/v1/connections/${id}/states`),
  deleteConnection: (id: string) =>
    request<void>(`/api/v1/connections/${id}`, { method: 'DELETE' }),
  workSync: (provider: string, instance?: string, workspaceId?: string, connectionId?: string) =>
    request<{ synced: number; provider: string }>('/api/v1/work/sync', {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify({ provider, instance, workspaceId, connectionId }),
    }),
  importWork: (item: WorkItem, workspaceId?: string) =>
    request<Task>('/api/v1/work/import', {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify({
        provider: item.ref.provider,
        instance: item.ref.instance,
        type: item.ref.type,
        id: item.ref.id,
        url: item.ref.url,
        revision: item.ref.revision,
        title: item.title,
        body: item.body ?? '',
        labels: item.labels ?? [],
        assignee: item.assignee ?? '',
        workspaceId,
      }),
    }),
  summonTask: (id: string) =>
    request<{ agent: Agent; run: Run }>(`/api/v1/tasks/${id}/summon`, { method: 'POST' }),
  assignTask: (id: string, body: { agentId?: string; groupId?: string }) =>
    request<Task>(`/api/v1/tasks/${id}/assignee`, {
      method: 'PATCH',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  startTaskTeam: (
    id: string,
    body: { repo: string; groupId?: string; roles: { role: string; agentId: string }[] },
  ) =>
    request<{ items: Run[] }>(`/api/v1/tasks/${id}/team`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  taskCommand: (id: string, body: { text: string; author?: string; repo?: string }) =>
    request<{ action: string }>(`/api/v1/tasks/${id}/command`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  taskNotes: (id: string) => request<List<TaskNote>>(`/api/v1/tasks/${id}/notes`),
  taskComments: (id: string) => request<List<TaskComment>>(`/api/v1/tasks/${id}/comments`),
  addTaskComment: (id: string, body: string, author = 'human') =>
    request<TaskComment>(`/api/v1/tasks/${id}/comments`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify({ body, author }),
    }),
  addTaskNote: (id: string, body: string, author = 'human') =>
    request<TaskNote>(`/api/v1/tasks/${id}/notes`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify({ body, author }),
    }),
  createRun: (body: { repo: string; prompt: string; agentId?: string; workspaceId: string }) =>
    request<Run>('/api/v1/runs', {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify({
        work: { provider: 'manual', instance: 'web', type: 'task', id: String(Date.now()) },
        workspaceId: body.workspaceId,
        agentId: body.agentId || 'claude',
        repo: body.repo,
        prompt: body.prompt,
      }),
    }),
  retryRun: (id: string) => request<Run>(`/api/v1/runs/${id}/retry`, { method: 'POST' }),
  cancelRun: (id: string) =>
    request<{ status: string }>(`/api/v1/runs/${id}/cancel`, { method: 'POST' }),
  runVariants: (id: string) =>
    request<List<Run> & { root: string }>(`/api/v1/runs/${id}/variants`),
  createVariants: (id: string, agents: string[]) =>
    request<List<Run>>(`/api/v1/runs/${id}/variants`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify({ agents }),
    }),
  approvals: (workspace: string) =>
    request<List<Approval>>(
      `/api/v1/approvals?status=pending&workspace=${encodeURIComponent(workspace)}`,
    ),
  decide: (id: string, decision: 'approved' | 'denied') =>
    request<{ status: string }>(`/api/v1/approvals/${id}/decision`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify({ decision, by: 'console' }),
    }),

  providers: () => request<List<Provider>>('/api/v1/providers'),
  probeProvider: (id: string) =>
    request<{ ok: boolean; error?: string }>(`/api/v1/providers/${encodeURIComponent(id)}/probe`, {
      method: 'POST',
    }),
  credentials: () => request<List<CredentialMeta>>('/api/v1/credentials'),
  putCredential: (body: { provider: string; subject?: string; accessToken: string }) =>
    request<void>('/api/v1/credentials', {
      method: 'PUT',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  deleteCredential: (provider: string, subject: string) =>
    request<void>(`/api/v1/credentials/${encodeURIComponent(provider)}/${encodeURIComponent(subject)}`, {
      method: 'DELETE',
    }),

  workflowTemplates: () => request<List<WorkflowTemplate>>('/api/v1/workflow-templates'),
  acpRegistry: (q = '') => request<{ items: ACPRegistryEntry[]; fetchedAt: string }>(`/api/v1/acp/registry?q=${encodeURIComponent(q)}`),
  installACP: (id: string) => request<{ id: string; version: string; dir: string; command: string; verified: boolean }>('/api/v1/acp/install', { method: 'POST', headers: JSON_HEADERS, body: JSON.stringify({ id }) }),
  usage: (workspace: string, days = 30) => request<UsageSummary>(`/api/v1/usage?workspace=${workspace}&days=${days}`),
  workflows: (workspace: string) =>
    request<List<Workflow>>(`/api/v1/workflows?workspace=${workspace}`),
  createWorkflow: (body: { workspaceId: string; name: string; description?: string; nodes: WorkflowNode[] }) =>
    request<Workflow>('/api/v1/workflows', { method: 'POST', headers: JSON_HEADERS, body: JSON.stringify(body) }),
  updateWorkflow: (id: string, body: { workspaceId: string; name: string; description?: string; nodes: WorkflowNode[] }) =>
    request<Workflow>(`/api/v1/workflows/${id}`, { method: 'PUT', headers: JSON_HEADERS, body: JSON.stringify(body) }),
  deleteWorkflow: (id: string) =>
    request<void>(`/api/v1/workflows/${id}`, { method: 'DELETE' }),
  startWorkflow: (id: string, body: { taskId?: string }) =>
    request<WorkflowRun>(`/api/v1/workflows/${id}/runs`, { method: 'POST', headers: JSON_HEADERS, body: JSON.stringify(body) }),
  workflowRuns: (workspace: string, task?: string) =>
    request<List<WorkflowRun>>(`/api/v1/workflow-runs?workspace=${workspace}${task ? `&task=${encodeURIComponent(task)}` : ''}`),
  workflowRun: (id: string) =>
    request<WorkflowRun>(`/api/v1/workflow-runs/${id}`),
  retryWorkflowRun: (id: string, nodeId?: string) =>
    request<WorkflowRun>(`/api/v1/workflow-runs/${id}/retry`, { method: 'POST', headers: JSON_HEADERS, body: JSON.stringify({ nodeId }) }),
  cancelWorkflowRun: (id: string) =>
    request<WorkflowRun>(`/api/v1/workflow-runs/${id}/cancel`, { method: 'POST' }),
  gateWorkflowRun: (id: string, body: { nodeId?: string; action: 'approve' | 'reject' | 'retry'; target?: string; note?: string }) =>
    request<WorkflowRun>(`/api/v1/workflow-runs/${id}/gate`, { method: 'POST', headers: JSON_HEADERS, body: JSON.stringify(body) }),
  automations: (workspace: string) =>
    request<List<UnifiedAutomation>>(`/api/v1/automations?workspace=${workspace}`),
  createAutomation: (body: {
    workspaceId: string
    kind?: 'schedule' | 'event'
    name: string
    schedule?: string
    agentId: string
    repo?: string
    prompt?: string
    precheck?: string
    missedGraceSeconds?: number
    reuseSession?: boolean
    provider?: string
    instance?: string
    triggerKind?: string
    triggerValue?: string
    writeback?: Record<string, unknown>
    stateMap?: Record<string, string>
    enabled: boolean
  }) =>
    request<UnifiedAutomation>('/api/v1/automations', {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  updateAutomation: (id: string, body: Record<string, unknown>) =>
    request<UnifiedAutomation>(`/api/v1/automations/${id}`, {
      method: 'PUT',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  runAutomation: (id: string) =>
    request<Run>(`/api/v1/automations/${id}/run`, { method: 'POST' }),
  deleteAutomation: (id: string) =>
    request<void>(`/api/v1/automations/${id}`, { method: 'DELETE' }),

  machines: () => request<List<Machine>>('/api/v1/machines'),
  machine: (id: string) => request<Machine>(`/api/v1/machines/${id}`),
  systemInfo: () => request<{ baseUrl: string }>('/api/v1/system/info'),
  machineJoinToken: () =>
    request<{ token: string; expiresAt: string }>('/api/v1/machines/join-token', {
      method: 'POST',
    }),
  runtimes: () => request<List<RuntimeInfo>>('/api/v1/runtimes'),
  takeoverMachine: (id: string, targetId: string) =>
    request<{ agents: number; runs: number }>(`/api/v1/machines/${id}/takeover`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify({ targetId }),
    }),
  setMachineStatus: (id: string, status: string) =>
    request<{ status: string }>(`/api/v1/machines/${id}/status`, { method: 'POST', headers: JSON_HEADERS, body: JSON.stringify({ status }) }),
  skills: (workspace: string) => request<List<Skill>>(`/api/v1/skills?workspace=${workspace}`),
  createSkill: (body: { workspaceId: string; name: string; description: string; body: string; version?: string; source?: string; tags?: string[]; enabled?: boolean }) =>
    request<Skill>('/api/v1/skills', {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  updateSkill: (id: string, body: { name: string; description: string; body: string }) =>
    request<Skill>(`/api/v1/skills/${id}`, {
      method: 'PUT',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  deleteSkill: (id: string) => request<void>(`/api/v1/skills/${id}`, { method: 'DELETE' }),
installSkill: (body: { workspaceId: string; name?: string; body?: string; sourceUrl?: string; version?: string; source?: string; tags?: string[] }) =>
    request<Skill>('/api/v1/skills/install', { method: 'POST', headers: JSON_HEADERS, body: JSON.stringify(body) }),
  skillMarket: (workspace: string) =>
    request<List<MarketSkill>>(`/api/v1/skills/market?workspace=${encodeURIComponent(workspace)}`),
  installMarketSkill: (id: string, body: { workspaceId: string; agentIds?: string[] }) =>
    request<{ skill: Skill; agents: number }>(`/api/v1/skills/market/${id}/install`, {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  setSkillEnabled: (id: string, enabled: boolean) =>
    request<Skill>(`/api/v1/skills/${id}/enabled`, { method: 'POST', headers: JSON_HEADERS, body: JSON.stringify({ enabled }) }),

  agents: (workspace: string) => request<List<Agent>>(`/api/v1/agents?workspace=${workspace}`),
  createAgent: (body: {
    name: string
    instructions?: string
    backend: string
    model: string
    skills: string[]
    repos: string[]
    sessionMode?: 'off' | 'task' | 'agent'
    workspaceId: string
    workspaceIds?: string[]
    machineId?: string
    tools?: Record<string, unknown>
  }) =>
    request<Agent>('/api/v1/agents', {
      method: 'POST',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  deleteAgent: (id: string) =>
    request<void>(`/api/v1/agents/${id}`, { method: 'DELETE' }),
  updateAgent: (id: string, body: Record<string, unknown>) =>
    request<Agent>(`/api/v1/agents/${id}`, {
      method: 'PUT',
      headers: JSON_HEADERS,
      body: JSON.stringify(body),
    }),
  agentEnv: (id: string) => request<{ keys: string[] }>(`/api/v1/agents/${id}/env`),
  setAgentSkills: (id: string, skills: string[]) =>
    request<Agent>(`/api/v1/agents/${id}/skills`, {
      method: 'PUT',
      headers: JSON_HEADERS,
      body: JSON.stringify({ skills }),
    }),
  putAgentEnv: (id: string, vars: Record<string, string>) =>
    request<{ keys: number }>(`/api/v1/agents/${id}/env`, {
      method: 'PUT',
      headers: JSON_HEADERS,
      body: JSON.stringify({ vars }),
    }),
}
