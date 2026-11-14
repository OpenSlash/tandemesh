import { spawn } from 'node:child_process'
import { existsSync } from 'node:fs'
import { setTimeout as sleep } from 'node:timers/promises'
import { fileURLToPath } from 'node:url'
import path from 'node:path'
import assert from 'node:assert/strict'
import { chromium } from 'playwright-core'

const here = path.dirname(fileURLToPath(import.meta.url))
const webRoot = path.resolve(here, '..')
const planeURL = process.env.ORCA_BASE_URL || 'http://127.0.0.1:8080'
const port = 5175
const baseURL = `http://127.0.0.1:${port}`
const stamp = Date.now().toString(36)
const title = `E2E live ${stamp}`

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
    } catch {}
    await sleep(150)
  }
  throw new Error(`timeout waiting for ${url}`)
}

async function createAndWaitRun() {
  const created = await fetch(`${planeURL}/api/v1/runs`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      work: { provider: 'manual', instance: 'default', type: 'task', id: `E2E-${stamp}` },
      prompt: title,
      repo: 'acme/api',
      agentId: 'claude',
    }),
  })
  assert.ok([200, 201].includes(created.status), `create run status ${created.status}`)
  const run = await created.json()

  const started = Date.now()
  while (Date.now() - started < 60000) {
    const resp = await fetch(`${planeURL}/api/v1/runs/${run.id}`)
    assert.equal(resp.status, 200, `get run status ${resp.status}`)
    const current = await resp.json()
    if (['succeeded', 'failed', 'cancelled'].includes(current.status)) {
      assert.equal(current.status, 'succeeded', `run ended ${current.status}: ${current.errorMessage ?? ''}`)
      return run
    }
    await sleep(500)
  }
  throw new Error(`run ${run.id} did not finish; is orca-agentd running?`)
}

const vite = spawn(process.execPath, [
  path.join(webRoot, 'node_modules', 'vite', 'bin', 'vite.js'),
  '--port', String(port), '--strictPort', '--host', '127.0.0.1',
], { cwd: webRoot, stdio: 'ignore', windowsHide: true })

let browser
try {
  const run = await createAndWaitRun()
  console.log(`run ${run.id} succeeded`)
  await waitFor(baseURL)

  browser = await chromium.launch({ headless: true, executablePath: edgePath() })
  const context = await browser.newContext({ locale: 'en-US', viewport: { width: 1280, height: 900 } })
  const page = await context.newPage()
  const errors = []
  page.on('pageerror', (e) => errors.push(`pageerror: ${e.message}`))
  page.on('response', (resp) => {
    if (resp.status() >= 400) errors.push(`http ${resp.status()} ${resp.url()}`)
  })
  page.on('console', (msg) => {
    if (msg.type() === 'error' && !msg.text().includes('Failed to load resource')) errors.push(`console: ${msg.text()}`)
  })

  await page.goto(`${baseURL}/tasks`, { waitUntil: 'networkidle' })
  await page.getByRole('heading', { name: 'Tasks', exact: true }).waitFor()
  await page.getByText(title).first().waitFor()

  await page.getByText(title).first().click()
  await page.getByRole('link', { name: 'open' }).click()
  await page.waitForURL(/\/tasks\/task_/)
  await page.getByRole('heading', { name: title, exact: true }).waitFor()

  await page.goto(`${baseURL}/runs`, { waitUntil: 'networkidle' })
  await page.getByText('succeeded').first().waitFor()

  await page.goto(`${baseURL}/machines`, { waitUntil: 'networkidle' })
  await page.getByText('e2e-1').first().waitFor()

  assert.deepEqual(errors, [], errors.join('\n'))
  console.log('PASS live web: tasks -> task detail -> runs -> machines')
} finally {
  if (browser) await browser.close()
  vite.kill()
}
