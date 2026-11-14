// Web E2E smoke test.
//
// Runs the real Vite app in headless Edge/Chromium and mocks the Orca HTTP API
// at the browser boundary. This verifies rendering, routing, data loading and
// responsive layout without needing a control-plane binary (which may be
// blocked by host application-control policy).

import { spawn } from 'node:child_process'
import { existsSync } from 'node:fs'
import { setTimeout as sleep } from 'node:timers/promises'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import assert from 'node:assert/strict'
import { chromium } from 'playwright-core'

const here = path.dirname(fileURLToPath(import.meta.url))
const webRoot = path.resolve(here, '..')
const viteBin = path.join(webRoot, 'node_modules', 'vite', 'bin', 'vite.js')
const port = 5174
const baseURL = `http://127.0.0.1:${port}`

const now = '2026-11-11T10:00:00.000Z'

const task = {
  id: 'task_1',
  workspaceId: 'ws_default',
  provider: 'manual',
  instance: 'default',
  workType: 'task',
  workId: 'ORC-1',
  url: 'https://example.test/ORC-1',
  revision: '1',
  title: 'Fix login redirect',
  body: 'Users are redirected to the wrong page after SSO login.',
  labels: ['bug', 'auth'],
  author: 'alice',
  assignee: 'alice',
  assigneeAgentId: 'claude',
  status: 'open',
  executionState: 'ready',
  inboxState: 'inbox',
  inboxReadAt: null,
  iterationId: null,
  parentId: null,
  doneAt: null,
  commentCount: 1,
  lastComment: 'Reproduced on staging.',
  lastCommentBy: 'bob',
  runCount: 1,
  activeRuns: 0,
  failedRuns: 0,
  succeededRuns: 1,
  readyToClose: false,
  createdAt: now,
  updatedAt: now,
  syncedAt: now,
}

const run = {
  id: 'run_1',
  work: { provider: 'manual', instance: 'default', type: 'task', id: 'ORC-1' },
  agentId: 'claude',
  machineId: 'mch_1',
  status: 'succeeded',
  priority: 0,
  attempt: 1,
  fence: 1,
  repo: 'acme/web',
  base: 'main',
  branch: 'orca/task-1',
  prompt: 'Fix login redirect',
  taskId: task.id,
  role: 'implementer',
  createdAt: now,
  startedAt: now,
  endedAt: now,
}

const notification = {
  id: 'ntf_1',
  workspaceId: 'ws_default',
  type: 'run',
  title: 'Agent finished run_1',
  body: 'Fix login redirect completed successfully.',
  link: '/runs/run_1',
  readAt: null,
  createdAt: now,
}

const machine = {
  id: 'mch_1',
  name: 'dev-1',
  labels: ['os=windows', 'gpu'],
  agents: ['claude'],
  runtimes: { claude: '1.0.0' },
  concurrency: 2,
  status: 'online',
  version: '0.2.0',
  lastSeen: now,
}

const agent = {
  id: 'claude',
  workspaceId: 'ws_default',
  machineId: 'mch_1',
  name: 'Claude',
  instructions: '',
  backend: 'claude',
  role: 'implementer',
  capabilities: [],
  model: 'claude',
  skills: [],
  repos: [],
  tools: [],
  createdAt: now,
  updatedAt: now,
}

function apiResponse(url, method) {
  const p = url.pathname
  if (p === '/api/v1/workspaces') return { items: [{ id: 'ws_default', name: 'Default', slug: 'default', createdAt: now }] }
  if (p === '/api/v1/tasks' && method === 'GET') return { items: [task], total: 1 }
  if (p === '/api/v1/tasks/task_1' && method === 'GET') return { task, runs: [run] }
  if (p === '/api/v1/tasks/task_1/activity') return { items: [] }
  if (p === '/api/v1/tasks/task_1/notes') return { notes: [] }
  if (p === '/api/v1/tasks/task_1/comments') return { comments: [] }
  if (p === '/api/v1/tasks/task_1/children') return { items: [] }
  if (p === '/api/v1/tasks/task_1/dependencies') return { items: [] }
  if (p === '/api/v1/tasks/task_1/dispatches') return { dispatches: [] }
  if (p === '/api/v1/tasks/task_1/gates') return { gates: [] }
  if (p === '/api/v1/tasks/task_1/annotations') return { annotations: [] }
  if (p === '/api/v1/tasks/task_1/messages') return { messages: [] }
  if (p === '/api/v1/tasks/task_1/messages/check') return { messages: [] }
  if (p === '/api/v1/notifications') return { notifications: [notification] }
  if (p === '/api/v1/approvals') return { items: [] }
  if (p === '/api/v1/agents') return { items: [agent] }
  if (p === '/api/v1/iterations') return { items: [] }
  if (p === '/api/v1/groups') return { items: [] }
  if (p === '/api/v1/connections') return { items: [] }
  if (p === '/api/v1/views') return { items: [] }
  if (p === '/api/v1/machines') return { items: [machine] }
  if (p === '/api/v1/runs') return { items: [] }
  if (p === '/api/v1/providers') return { items: [] }
  if (p === '/api/v1/worktrees') return { worktrees: [] }
  if (p === '/api/v1/dispatches') return { dispatches: [] }
  if (p === '/api/v1/tasks/task_1' && method === 'PATCH') return task
  if (method !== 'GET') return {}
  return { items: [] }
}

function edgePath() {
  const candidates = [
    process.env.EDGE_PATH,
    'C:\\Program Files (x86)\\Microsoft\\Edge\\Application\\msedge.exe',
    'C:\\Program Files\\Microsoft\\Edge\\Application\\msedge.exe',
  ].filter(Boolean)
  return candidates.find((candidate) => existsSync(candidate)) || chromium.executablePath()
}

async function waitFor(url, timeoutMs = 15000) {
  const started = Date.now()
  while (Date.now() - started < timeoutMs) {
    try {
      const resp = await fetch(url)
      if (resp.ok) return
    } catch {
      // not ready
    }
    await sleep(150)
  }
  throw new Error(`timeout waiting for ${url}`)
}

async function main() {
  const vite = spawn(process.execPath, [viteBin, '--port', String(port), '--strictPort', '--host', '127.0.0.1'], {
    cwd: webRoot,
    stdio: 'ignore',
    windowsHide: true,
  })

  let browser
  const errors = []
  try {
    await waitFor(baseURL)
    browser = await chromium.launch({ headless: true, executablePath: edgePath() })
    const context = await browser.newContext({ locale: 'en-US', viewport: { width: 1280, height: 900 } })

    await context.route('**/*', async (route) => {
      const request = route.request()
      const url = new URL(request.url())
      if (url.pathname === '/auth/me') {
        return route.fulfill({ status: 200, contentType: 'application/json', body: JSON.stringify({ user: 'alice' }) })
      }
      if (url.pathname.startsWith('/api/v1/')) {
        return route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify(apiResponse(url, request.method())),
        })
      }
      return route.continue()
    })

    const page = await context.newPage()
    page.on('pageerror', (error) => errors.push(error.message))
    page.on('console', (msg) => {
      if (msg.type() === 'error') errors.push(msg.text())
    })

    await page.goto(`${baseURL}/tasks`, { waitUntil: 'networkidle' })
    await page.getByRole('heading', { name: 'Tasks', exact: true }).waitFor()
    await page.getByText('Fix login redirect').first().waitFor()
    assert.equal(await page.getByText('Fix login redirect').count() > 0, true, 'task fixture should render')

    await page.getByText('Fix login redirect').first().click()
    await page.getByRole('link', { name: 'open' }).click()
    await page.waitForURL(/\/tasks\/task_1$/)
    await page.getByRole('heading', { name: 'Fix login redirect', exact: true }).waitFor()
    await page.getByText('Users are redirected to the wrong page after SSO login.').waitFor()

    await page.goto(`${baseURL}/notifications`, { waitUntil: 'networkidle' })
    await page.getByText('Agent finished run_1').waitFor()

    await page.goto(`${baseURL}/machines`, { waitUntil: 'networkidle' })
    await page.getByText('dev-1').waitFor()
    await page.getByText('gpu', { exact: true }).waitFor()

    for (const width of [820, 390]) {
      await page.setViewportSize({ width, height: 900 })
      await page.goto(`${baseURL}/tasks`, { waitUntil: 'networkidle' })
      const overflow = await page.evaluate(() => document.documentElement.scrollWidth - window.innerWidth)
      assert.ok(overflow <= 1, `layout overflows at ${width}px by ${overflow}px`)
    }

    assert.deepEqual(errors, [], `browser errors: ${errors.join('\n')}`)
    console.log('PASS tasks list -> task detail -> notifications -> machines -> responsive 820/390')
  } finally {
    if (browser) await browser.close()
    vite.kill()
  }
}

main().catch((error) => {
  console.error(error)
  process.exit(1)
})
