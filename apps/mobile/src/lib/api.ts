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
