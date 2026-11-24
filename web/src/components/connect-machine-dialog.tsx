import { useEffect, useMemo, useRef, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Check, Copy, Laptop, RefreshCw, Terminal } from 'lucide-react'
import { api } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { Button } from './ui/button'
import { Dialog, DialogContent, DialogTrigger } from './ui/dialog'
import { cn } from '../lib/utils'

type HostOS = 'windows' | 'macos' | 'linux'

export function ConnectMachineDialog() {
  const { t } = useI18n()
  const qc = useQueryClient()
  const [open, setOpen] = useState(false)
  const [os, setOs] = useState<HostOS>('windows')
  const [copied, setCopied] = useState(false)
  const generated = useRef(false)

  const { data: info } = useQuery({
    queryKey: ['system-info'],
    queryFn: api.systemInfo,
    enabled: open,
  })
  const token = useMutation({ mutationFn: api.machineJoinToken })
  const baseURL = useMemo(() => {
    if (typeof window === 'undefined') return 'http://localhost:8080'
    const host = window.location.hostname
    const browserLoopback = host === 'localhost' || host === '127.0.0.1'
    const configured = info?.baseUrl
    if (!configured) {
      return `${window.location.protocol}//${host}:8080`
    }
    const configuredLoopback = /localhost|127\.0\.0\.1/.test(configured)
    if (configuredLoopback && !browserLoopback) {
      return `${window.location.protocol}//${host}:8080`
    }
    return configured
  }, [info])

  useEffect(() => {
    if (!open) {
      generated.current = false
      setCopied(false)
      return
    }
    if (!generated.current) {
      generated.current = true
      token.mutate()
    }
  }, [open])

  const tokenValue = token.data?.token
  const query = tokenValue ? `?token=${encodeURIComponent(tokenValue)}` : ''
  const command =
    os === 'windows'
      ? `powershell -NoProfile -Command "irm ${baseURL}/install/agentd.ps1${query} | iex"`
      : `curl -fsSL ${baseURL}/install/agentd.sh${query} | sh`

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(command)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      setCopied(false)
    }
  }

  const refresh = () => {
    void qc.invalidateQueries({ queryKey: ['machines'] })
    setOpen(false)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="primary" size="sm">
          <Laptop className="h-3.5 w-3.5" />
          {t('machines.connect')}
        </Button>
      </DialogTrigger>
      <DialogContent
        title={t('machines.connectTitle')}
        description={t('machines.connectHint')}
        className="w-[620px]"
      >
        <div className="flex gap-1 rounded-md bg-panel2 p-1">
          {(['windows', 'macos', 'linux'] as HostOS[]).map((value) => (
            <button
              key={value}
              onClick={() => setOs(value)}
              className={cn(
                'h-7 flex-1 rounded text-[11.5px]',
                os === value ? 'bg-panel text-fg shadow-sm' : 'text-muted hover:text-fg',
              )}
            >
              {value === 'windows' ? 'Windows' : value === 'macos' ? 'macOS' : 'Linux'}
            </button>
          ))}
        </div>

        <div className="mt-3 rounded-lg border border-line bg-panel/40 p-3">
          <div className="flex items-center gap-1.5 text-[11.5px] font-medium">
            <Terminal className="h-3.5 w-3.5 text-muted" />
            {t('machines.connectCommand')}
          </div>
          <div className="mt-2 rounded-md bg-[#111] p-3">
            <code className="block break-all font-mono text-[11px] leading-5 text-[#d8e0e8]">
              {command}
            </code>
            <div className="mt-2 flex items-center justify-between gap-2">
              <span className="text-[10px] text-[#8b98a5]">
                {token.data ? t('machines.tokenExpires') : t('machines.tokenLoading')}
              </span>
              <button
                onClick={copy}
                disabled={!token.data}
                className="inline-flex h-6 items-center gap-1 rounded border border-white/15 px-2 text-[10.5px] text-[#d8e0e8] hover:bg-white/10 disabled:opacity-40"
              >
                {copied ? <Check className="h-3 w-3" /> : <Copy className="h-3 w-3" />}
                {copied ? t('machines.copied') : t('machines.copy')}
              </button>
            </div>
          </div>
          <div className="mt-2 flex items-center justify-between gap-2">
            <p className="text-[10.5px] text-muted">{t('machines.connectBinaryNote')}</p>
            <Button
              size="sm"
              variant="ghost"
              disabled={token.isPending}
              onClick={() => token.mutate()}
            >
              <RefreshCw className={cn('h-3 w-3', token.isPending && 'animate-spin')} />
              {t('machines.regenerateToken')}
            </Button>
          </div>
        </div>

        <ol className="mt-3 space-y-1.5 text-[11.5px] text-muted">
          <li>1. {t('machines.connectStep1')}</li>
          <li>2. {t('machines.connectStep2')}</li>
          <li>3. {t('machines.connectStep3')}</li>
        </ol>

        <div className="mt-4 flex items-center justify-between gap-2 border-t border-line pt-3">
          <p className="text-[10.5px] text-muted">{t('machines.connectDetectHint')}</p>
          <Button size="sm" variant="primary" onClick={refresh}>
            {t('machines.iHaveRunIt')}
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
