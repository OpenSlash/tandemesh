import { useMemo, useState } from 'react'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { Check, MessageSquarePlus, Send } from 'lucide-react'
import { api, type DiffAnnotation, type Run } from '../lib/api'
import { useI18n } from '../lib/i18n'
import { Button } from './ui/button'

type DiffLine = {
  kind: 'add' | 'del' | 'ctx' | 'hunk'
  text: string
  oldNo?: number
  newNo?: number
}

type DiffFile = {
  path: string
  oldPath?: string
  status: 'modified' | 'added' | 'deleted'
  additions: number
  deletions: number
  lines: DiffLine[]
}

function parseDiff(raw: string): DiffFile[] {
  const files: DiffFile[] = []
  let current: DiffFile | null = null
  let oldNo = 0
  let newNo = 0
  for (const line of raw.split('\n')) {
    if (line.startsWith('diff --git ')) {
      const m = line.match(/ b\/(.+)$/)
      current = { path: m?.[1] ?? 'unknown', status: 'modified', additions: 0, deletions: 0, lines: [] }
      files.push(current)
      continue
    }
    if (!current) continue
    if (line.startsWith('new file mode')) {
      current.status = 'added'
      continue
    }
    if (line.startsWith('deleted file mode')) {
      current.status = 'deleted'
      continue
    }
    if (line.startsWith('--- ')) {
      const p = line.slice(4).replace(/^a\//, '')
      if (p !== '/dev/null') current.oldPath = p
      continue
    }
    if (line.startsWith('+++ ')) {
      const p = line.slice(4).replace(/^b\//, '')
      if (p !== '/dev/null') current.path = p
      continue
    }
    const hunk = line.match(/^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/)
    if (hunk) {
      oldNo = Number(hunk[1])
      newNo = Number(hunk[2])
      current.lines.push({ kind: 'hunk', text: line })
      continue
    }
    if (line.startsWith('\\ No newline')) continue
    if (line.startsWith('index ') || line.startsWith('similarity ') || line.startsWith('rename ')) continue
    if (line.startsWith('+')) {
      current.lines.push({ kind: 'add', text: line.slice(1), newNo })
      current.additions++
      newNo++
    } else if (line.startsWith('-')) {
      current.lines.push({ kind: 'del', text: line.slice(1), oldNo })
      current.deletions++
      oldNo++
    } else if (line.startsWith(' ')) {
      current.lines.push({ kind: 'ctx', text: line.slice(1), oldNo, newNo })
      oldNo++
      newNo++
    }
  }
  return files.filter((f) => f.lines.length > 0)
}

export function TaskDiffPanel({ taskId, runs }: { taskId: string; runs: Run[] }) {
  const { t } = useI18n()
  const qc = useQueryClient()
  const run = [...runs].reverse().find((r) => r.status === 'succeeded' || r.status === 'running') ?? runs[runs.length - 1]
  const { data } = useQuery({
    queryKey: ['run-diff', run?.id],
    queryFn: () => api.runDiff(run!.id),
    enabled: !!run?.id,
    refetchInterval: 5000,
  })
  const { data: notesData } = useQuery({
    queryKey: ['task-annotations', taskId],
    queryFn: () => api.listAnnotations(taskId),
    refetchInterval: 5000,
  })
  const files = useMemo(() => parseDiff(data?.diff ?? ''), [data?.diff])
  const annotations = notesData?.annotations ?? []
  const [selected, setSelected] = useState<{ file: string; line: number; side: string } | null>(null)
  const [body, setBody] = useState('')
  const [collapsed, setCollapsed] = useState<Record<string, boolean>>({})

  const refresh = () => {
    void qc.invalidateQueries({ queryKey: ['task-annotations', taskId] })
    void qc.invalidateQueries({ queryKey: ['run-diff', run?.id] })
  }
  const add = useMutation({
    mutationFn: () =>
      api.createAnnotation(taskId, {
        runId: run?.id,
        filePath: selected!.file,
        line: selected!.line,
        side: selected!.side,
        body,
      }),
    onSuccess: () => {
      setBody('')
      setSelected(null)
      refresh()
    },
  })
  const send = useMutation({
    mutationFn: () => api.sendAnnotations(taskId, { runId: run?.id }),
    onSuccess: refresh,
  })
  const resolve = useMutation({
    mutationFn: (id: string) => api.resolveAnnotation(id),
    onSuccess: refresh,
  })

  const openCount = annotations.filter((a) => a.status === 'open').length
  const byLine = (file: string, line: number, side: string): DiffAnnotation[] =>
    annotations.filter((a) => a.filePath === file && a.line === line && (a.side || 'new') === side)

  if (!run) {
    return <p className="text-[12px] text-muted">{t('diff.noRun')}</p>
  }

  return (
    <div className="rounded-lg border border-line bg-panel/40 p-4">
      <div className="flex items-center gap-2">
        <h2 className="text-[11px] font-medium uppercase tracking-wider text-muted">{t('diff.title')}</h2>
        <span className="font-mono text-[10px] text-muted/60">{run.id.slice(-10)}</span>
        <span className="ml-auto text-[10px] text-muted">{openCount} {t('diff.open')}</span>
        <Button size="sm" variant="ghost" disabled={openCount === 0 || send.isPending} onClick={() => send.mutate()}>
          <Send className="h-3 w-3" /> {t('diff.send')}
        </Button>
      </div>
      {data?.truncated && <p className="mt-1 text-[10px] text-warn">{t('diff.truncated')}</p>}
      {!data?.diff && <p className="mt-2 text-[11px] text-muted/60">{t('diff.empty')}</p>}
      <div className="mt-2 space-y-2">
        {files.map((file) => {
          const isCollapsed = collapsed[file.path]
          return (
            <div key={file.path} className="overflow-hidden rounded border border-line2 bg-panel">
              <button
                className="flex w-full items-center gap-2 bg-panel px-2 py-1 text-left text-[11px]"
                onClick={() => setCollapsed({ ...collapsed, [file.path]: !isCollapsed })}
              >
                <span className="font-mono text-fg">{file.path}</span>
                {file.oldPath && file.oldPath !== file.path && <span className="text-muted/50">← {file.oldPath}</span>}
                <span className="ml-auto text-ok">+{file.additions}</span>
                <span className="text-err">-{file.deletions}</span>
              </button>
              {!isCollapsed && (
                <div className="max-h-[420px] overflow-auto font-mono text-[11px]">
                  {file.lines.map((line, index) => {
                    if (line.kind === 'hunk') {
                      return (
                        <div key={index} className="bg-panel/80 px-2 py-0.5 text-[10px] text-accent2/80">{line.text}</div>
                      )
                    }
                    const side = line.kind === 'del' ? 'old' : 'new'
                    const no = line.kind === 'del' ? line.oldNo! : line.newNo!
                    const comments = byLine(file.path, no, side)
                    const isSelected = selected?.file === file.path && selected.line === no && selected.side === side
                    return (
                      <div key={index}>
                        <div
                          className={`group flex cursor-pointer ${line.kind === 'add' ? 'bg-ok/10' : line.kind === 'del' ? 'bg-err/10' : ''}`}
                          onClick={() => { setSelected({ file: file.path, line: no, side }); setBody('') }}
                        >
                          <span className="w-10 shrink-0 select-none border-r border-line2 px-1 text-right text-muted/40">{line.oldNo ?? ''}</span>
                          <span className="w-10 shrink-0 select-none border-r border-line2 px-1 text-right text-muted/40">{line.newNo ?? ''}</span>
                          <span className="w-4 shrink-0 select-none text-center text-muted/30">{line.kind === 'add' ? '+' : line.kind === 'del' ? '-' : ''}</span>
                          <span className="whitespace-pre-wrap break-all pr-2 text-fg">{line.text || ' '}</span>
                          <MessageSquarePlus className="ml-auto mr-1 hidden h-3 w-3 shrink-0 text-muted group-hover:block" />
                        </div>
                        {comments.map((a) => (
                          <div key={a.id} className="flex items-start gap-2 border-l-2 border-accent2/40 bg-panel/70 px-3 py-1 text-[10.5px]">
                            <span className={a.status === 'open' ? 'text-accent2' : 'text-muted/50'}>{a.author || 'you'}</span>
                            <span className="min-w-0 flex-1 text-muted">{a.body}</span>
                            {a.status === 'open' && (
                              <button className="text-muted hover:text-ok" onClick={() => resolve.mutate(a.id)}><Check className="h-3 w-3" /></button>
                            )}
                          </div>
                        ))}
                        {isSelected && (
                          <div className="flex gap-1 border-l-2 border-accent2 bg-panel/80 px-2 py-1.5">
                            <input
                              autoFocus
                              value={body}
                              onChange={(e) => setBody(e.target.value)}
                              placeholder={t('diff.placeholder')}
                              className="min-w-0 flex-1 rounded border border-line2 bg-panel px-1.5 py-0.5 font-sans text-[11px] text-fg"
                              onKeyDown={(e) => {
                                if (e.key === 'Enter' && body.trim()) add.mutate()
                                if (e.key === 'Escape') setSelected(null)
                              }}
                            />
                            <Button size="sm" variant="ghost" disabled={!body.trim() || add.isPending} onClick={() => add.mutate()}>{t('common.save')}</Button>
                          </div>
                        )}
                      </div>
                    )
                  })}
                </div>
              )}
            </div>
          )
        })}
      </div>
      {send.error && <p className="mt-1 text-[11px] text-err">{String(send.error.message)}</p>}
    </div>
  )
}
