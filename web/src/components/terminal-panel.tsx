import { useEffect, useRef, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { useMutation, useQuery } from '@tanstack/react-query'
import { Loader2, MessageSquare, Power, RotateCw } from 'lucide-react'
import { Terminal as XTerm } from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'
import { api, type AgentSession, type TerminalSession } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { Button } from './ui/button'

export function TerminalPanel({ session }: { session: AgentSession }) {
  const { t } = useI18n()
  const navigate = useNavigate()
  const [terminal, setTerminal] = useState<TerminalSession | null>(null)
  const [error, setError] = useState('')
  const [hasOutput, setHasOutput] = useState(false)
  const lastSeq = useRef(0)
  const created = useRef(false)
  const hostRef = useRef<HTMLDivElement>(null)
  const termRef = useRef<XTerm | null>(null)
  const inputRef = useRef<(data: string) => void>(() => {})
  const sendQueue = useRef<Promise<unknown>>(Promise.resolve())
  const previousTerminalID = useRef('')

  const { data: list } = useQuery({
    queryKey: ['terminal-sessions', session.workspaceId],
    queryFn: () => api.listTerminalSessions(session.workspaceId),
    refetchInterval: 5000,
  })

  useEffect(() => {
    if (terminal || created.current || !list) return
    const existing = list.items.find(
      (x) => x.agentSessionId === session.id && x.status !== 'exited' && x.status !== 'closed',
    )
    if (existing) {
      setTerminal(existing)
      return
    }
    created.current = true
    api
      .createTerminalSession({ workspaceId: session.workspaceId, agentSessionId: session.id, cols: 120, rows: 32 })
      .then(setTerminal)
      .catch((e) => setError(String(e.message ?? e)))
  }, [list, session.id, session.workspaceId, terminal])

  useEffect(() => {
    const host = hostRef.current
    if (!host) return
    const term = new XTerm({
      cursorBlink: true,
      fontFamily: 'Consolas, "Cascadia Mono", monospace',
      fontSize: 12,
      lineHeight: 1.25,
      scrollback: 5000,
      theme: {
        background: '#0b0d10',
        foreground: '#d6e2ea',
        cursor: '#f4f4f5',
        selectionBackground: '#334155',
        black: '#0b0d10',
        brightBlack: '#64748b',
      },
    })
    term.open(host)
    term.resize(120, 32)
    termRef.current = term
    const disposable = term.onData((data) => inputRef.current(data))
    const parsed = term.onWriteParsed(() => term.refresh(0, term.rows - 1))
    term.focus()
    return () => {
      disposable.dispose()
      parsed.dispose()
      term.dispose()
      termRef.current = null
    }
  }, [])

  useEffect(() => {
    if (!terminal?.id) return
    if (previousTerminalID.current && previousTerminalID.current !== terminal.id) {
      lastSeq.current = 0
      setHasOutput(false)
      termRef.current?.reset()
    }
    previousTerminalID.current = terminal.id
  }, [terminal?.id])

  useEffect(() => {
    if (!terminal) return
    let stop = false
    const poll = async () => {
      try {
        const res = await api.terminalOutput(terminal.id, lastSeq.current)
        if (stop) return
        if (res.events.length > 0) {
          lastSeq.current = res.events[res.events.length - 1].seq
          const chunk = res.events.map((event) => event.data).join('')
          termRef.current?.write(chunk)
          setHasOutput(true)
        }
        setTerminal((prev) => (prev ? { ...prev, ...res.terminal } : res.terminal))
      } catch (e) {
        if (!stop) setError(String((e as Error).message ?? e))
      }
    }
    void poll()
    const timer = window.setInterval(poll, 120)
    return () => {
      stop = true
      window.clearInterval(timer)
    }
  }, [terminal?.id])

  const ended = terminal?.status === 'exited' || terminal?.status === 'closed'
  const write = (data: string) => {
    if (!terminal || ended) return
    sendQueue.current = sendQueue.current
      .then(() => api.terminalInput(terminal.id, data))
      .catch((e) => setError(String((e as Error).message ?? e)))
  }
  inputRef.current = write

  const close = useMutation({
    mutationFn: () => api.closeTerminalSession(terminal!.id),
    onSuccess: () => setTerminal((prev) => (prev ? { ...prev, status: 'closed' } : prev)),
  })
  const restart = useMutation({
    mutationFn: () =>
      api.createTerminalSession({ workspaceId: session.workspaceId, agentSessionId: session.id, cols: 120, rows: 32 }),
    onSuccess: (t) => {
      lastSeq.current = 0
      setHasOutput(false)
      termRef.current?.reset()
      setTerminal(t)
    },
  })

  const continueChat = useMutation({
    mutationFn: () => api.continueTerminalChat(terminal!.id),
    onSuccess: (chat) => navigate(`/sessions?session=${chat.id}`),
  })

  return (
    <div className="rounded-lg border border-line bg-[#0b0d10]">
      <div className="flex items-center gap-2 border-b border-line px-3 py-1.5">
        <span className="text-[11px] font-medium uppercase tracking-wider text-muted">{t('terminal.title')}</span>
        <span className="font-mono text-[10px] text-muted/60">{terminal?.id.slice(-10)}</span>
        <span className={`text-[10px] ${ended ? 'text-muted' : 'text-ok'}`}>{terminal?.status ?? 'starting'}</span>
        <div className="ml-auto flex gap-1">
          <Button size="sm" variant="ghost" onClick={() => continueChat.mutate()} disabled={!terminal || continueChat.isPending}>
            <MessageSquare className="h-3 w-3" /> {t('terminal.continueChat')}
          </Button>
          <Button size="sm" variant="ghost" onClick={() => restart.mutate()} disabled={restart.isPending}>
            <RotateCw className="h-3 w-3" /> {t('terminal.restart')}
          </Button>
          <Button size="sm" variant="ghost" onClick={() => close.mutate()} disabled={!terminal || ended || close.isPending}>
            <Power className="h-3 w-3" /> {t('terminal.close')}
          </Button>
        </div>
      </div>
      <div className="relative h-[560px] w-full overflow-hidden p-1">
        <div ref={hostRef} className="h-full w-full" onClick={() => termRef.current?.focus()} />
        {!hasOutput && !error && (
          <div className="pointer-events-none absolute inset-0 flex items-center justify-center gap-2 text-[12px] text-muted">
            {(terminal?.status === 'pending' || !terminal) && <Loader2 className="h-3 w-3 animate-spin" />}
            {t('terminal.waiting')}
          </div>
        )}
      </div>
      {error && <p className="border-t border-line px-3 py-1 text-[11px] text-err">{error}</p>}
      <p className="border-t border-line px-3 py-1 text-[10px] text-muted/60">
        {t('terminal.xtermHint')}
      </p>
    </div>
  )
}
