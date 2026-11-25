export interface MobileDevice {
  id: string
  name: string
  platform: string
  createdAt: number
  lastUsedAt: number
}

export interface Workspace {
  id: string
  name: string
}

export interface Task {
  id: string
  title: string
  status: string
  assigneeAgentId?: string
  updatedAt: string
}

export interface Run {
  id: string
  status: string
  agentId?: string
  prompt?: string
  errorMessage?: string
  createdAt: string
}

interface PairingResponse {
  token: string
  device: MobileDevice
  user: string
}

interface ListResponse<T> {
  items: T[]
}

function normalizeBaseUrl(value: string) {
  return value.trim().replace(/\/+$/, '')
}

export async function exchangePairing(
  baseUrl: string,
  code: string,
  deviceName: string,
  platform: string,
): Promise<PairingResponse> {
  const response = await fetch(`${normalizeBaseUrl(baseUrl)}/api/v1/mobile/pairings/exchange`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ code, deviceName, platform }),
  })
  if (!response.ok) {
    throw new Error((await response.text()) || `HTTP ${response.status}`)
  }
  return (await response.json()) as PairingResponse
}

export class TandemeshClient {
  constructor(
    private readonly baseUrl: string,
    private readonly token: string,
  ) {}

  private async request<T>(path: string): Promise<T> {
    const response = await fetch(`${normalizeBaseUrl(this.baseUrl)}${path}`, {
      headers: { Authorization: `Bearer ${this.token}` },
    })
    if (!response.ok) {
      throw new Error((await response.text()) || `HTTP ${response.status}`)
    }
    if (response.status === 204) return undefined as T
    return (await response.json()) as T
  }

  workspaces() {
    return this.request<ListResponse<Workspace>>('/api/v1/workspaces')
  }

  tasks(workspaceId: string) {
    return this.request<ListResponse<Task>>(
      `/api/v1/tasks?workspace=${encodeURIComponent(workspaceId)}`,
    )
  }

  task(id: string) {
    return this.request<{ task: Task; runs: Run[] }>(`/api/v1/tasks/${id}`)
  }

  approvals(workspaceId: string) {
    return this.request<ListResponse<{ id: string; runId: string; tool: string; risk: string; status: string }>>(
      `/api/v1/approvals?workspaceId=${encodeURIComponent(workspaceId)}`,
    )
  }

  run(id: string) {
    return this.request<Run>(`/api/v1/runs/${id}`)
  }

  runTranscript(id: string) {
    return this.request<{ items: { kind: string; text: string; tsMs: number; seq: number }[]; runId: string }>(
      `/api/v1/runs/${id}/transcript`,
    )
  }

  notifications(workspaceId: string) {
    return this.request<ListResponse<{ id: string; title: string; body?: string; readAt?: string }>>(
      `/api/v1/notifications?workspaceId=${encodeURIComponent(workspaceId)}`,
    )
  }
}

export function parsePairingLink(url: string): { host?: string; code?: string } {
  if (!url.startsWith('tandemesh://pair')) return {}
  const query = url.split('?')[1] ?? ''
  const params: Record<string, string> = {}
  for (const part of query.split('&')) {
    const [key, value] = part.split('=')
    if (key && value) params[decodeURIComponent(key)] = decodeURIComponent(value)
  }
  return {
    host: params.host,
    code: params.code,
  }
}
