import { useEffect, useMemo, useRef, useState, type PointerEvent as ReactPointerEvent, type ReactNode } from 'react'
import { Link, useParams } from 'react-router-dom'
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { ArrowLeft, Bot, Check, ChevronDown, Copy, FileCode2, FilePenLine, FileText, ExternalLink, FolderSearch, GitBranch, Globe, HelpCircle, Loader2, Paperclip, Pencil, RefreshCw, RotateCcw, Search, Share2, ShieldCheck, Slash, Sparkles, Square, Terminal, Upload, Wrench, X } from 'lucide-react'
import { api, type Approval, type SessionComment, type SessionMessage, type SessionShare, type PRState } from '../lib/api'
import { useWorkspace } from '../lib/workspace'
import { useI18n } from '../lib/i18n'
import { Button } from '../components/ui/button'
import { Dialog, DialogContent } from '../components/ui/dialog'
import { Input } from '../components/ui/input'
import { TerminalPanel } from '../components/terminal-panel'

const INLINE_IMAGE = /!\[([^\]]*)\]\((data:image\/[^)]+)\)/g

type PaneView = 'chat' | 'diff' | 'terminal'

type PaneLeaf = { type: 'leaf'; id: string; view: PaneView; sessionId?: string }
type PaneSplit = { type: 'split'; id: string; direction: 'row' | 'column'; ratio: number; children: [PaneNode, PaneNode] }
type PaneNode = PaneLeaf | PaneSplit

function newPaneID(prefix: string) {
  return `${prefix}_${Math.random().toString(36).slice(2, 9)}`
}

function paneLeaf(view: PaneView, sessionId?: string): PaneLeaf {
  return { type: 'leaf', id: newPaneID('pane'), view, sessionId }
}

function panePreset(sessionId?: string): PaneNode {
  return {
    type: 'split', id: newPaneID('split'), direction: 'row', ratio: 50,
    children: [paneLeaf('chat', sessionId), paneLeaf('diff', sessionId)],
  }
}

function replacePaneNode(node: PaneNode, id: string, next: PaneNode): PaneNode {
  if (node.id === id) return next
  if (node.type === 'leaf') return node
  return { ...node, children: [replacePaneNode(node.children[0], id, next), replacePaneNode(node.children[1], id, next)] }
}

function removePaneNode(node: PaneNode, id: string): PaneNode | null {
  if (node.type === 'leaf') return node.id === id ? null : node
  const left = removePaneNode(node.children[0], id)
  const right = removePaneNode(node.children[1], id)
  if (left && right) return { ...node, children: [left, right] }
  return left ?? right
}

function updatePaneRatio(node: PaneNode, id: string, ratio: number): PaneNode {
  if (node.type === 'leaf') return node
  if (node.id === id) return { ...node, ratio }
  return { ...node, children: [updatePaneRatio(node.children[0], id, ratio), updatePaneRatio(node.children[1], id, ratio)] }
}

function findPaneNode(node: PaneNode, id: string): PaneNode | null {
  if (node.id === id) return node
  if (node.type === 'leaf') return null
  return findPaneNode(node.children[0], id) ?? findPaneNode(node.children[1], id)
}
const SLASH_COMMANDS = [
  { command: '/diff', description: 'Open the session diff' },
  { command: '/chat', description: 'Switch to chat' },
  { command: '/terminal', description: 'Switch to terminal' },
  { command: '/ship', description: 'Open commit and push' },
  { command: '/worktree', description: 'Manage the worktree' },
  { command: '/clear', description: 'Clear the composer' },
]

function splitInlineImages(text: string) {
  const images: { alt: string; src: string }[] = []
  const cleaned = text.replace(INLINE_IMAGE, (_all, alt: string, src: string) => {
    images.push({ alt, src })
    return ''
  }).trim()
  return { text: cleaned, images }
}

function toolMeta(name: string) {
  const value = name.toLowerCase()
  if (value.includes('bash') || value.includes('shell') || value.includes('exec') || value.includes('terminal')) {
    return { label: 'Command', Icon: Terminal }
  }
  if (value.includes('edit') || value.includes('write') || value.includes('patch')) {
    return { label: 'Edit', Icon: FilePenLine }
  }
  if (value.includes('read')) {
    return { label: 'Read', Icon: FileText }
  }
  if (value.includes('grep') || value.includes('search')) {
    return { label: 'Search', Icon: Search }
  }
  if (value.includes('glob') || value.includes('list') || value.includes('find')) {
    return { label: 'Files', Icon: FolderSearch }
  }
  if (value.includes('fetch') || value.includes('http') || value.includes('web')) {
    return { label: 'Fetch', Icon: Globe }
  }
  return { label: 'Tool', Icon: Wrench }
}

type DiffRow = {
  kind: 'meta' | 'hunk' | 'add' | 'del' | 'ctx'
  text: string
  file: string
  line: number
  side: 'new' | 'old'
}

function diffFileId(file: string) {
  return `diff-file-${file.replace(/[^a-zA-Z0-9_-]/g, '-')}`
}

function parseDiff(diff: string): DiffRow[] {
  const rows: DiffRow[] = []
  let file = ''
  let oldLine = 0
  let newLine = 0
  for (const text of diff.split('\n')) {
    if (text.startsWith('diff --git ')) {
      const match = text.match(/ b\/(.+)$/)
      file = match?.[1] ?? file
      oldLine = 0
      newLine = 0
      rows.push({ kind: 'meta', text, file, line: 0, side: 'new' })
      continue
    }
    if (text.startsWith('+++ b/')) {
      file = text.slice(6)
      rows.push({ kind: 'meta', text, file, line: 0, side: 'new' })
      continue
    }
    if (text.startsWith('--- ') || text.startsWith('+++ ')) {
      rows.push({ kind: 'meta', text, file, line: 0, side: 'new' })
      continue
    }
    const hunk = text.match(/^@@ -(\d+)(?:,\d+)? \+(\d+)(?:,\d+)? @@/)
    if (hunk) {
      oldLine = Number(hunk[1])
      newLine = Number(hunk[2])
      rows.push({ kind: 'hunk', text, file, line: newLine, side: 'new' })
      continue
    }
    if (text.startsWith('+')) {
      rows.push({ kind: 'add', text, file, line: newLine, side: 'new' })
      newLine++
    } else if (text.startsWith('-')) {
      rows.push({ kind: 'del', text, file, line: oldLine, side: 'old' })
      oldLine++
    } else {
      rows.push({ kind: 'ctx', text, file, line: newLine, side: 'new' })
      oldLine++
      newLine++
    }
  }
  return rows
}

export default function SessionDetailPage() {
  const { id = '' } = useParams()
  return <SessionDetail id={id} />
}

export function SessionDetail({ id, embedded = false, pane = false, paneView = 'chat' }: { id: string; embedded?: boolean; pane?: boolean; paneView?: PaneView }) {
  const { t } = useI18n()
  const qc = useQueryClient()
  const [prompt, setPrompt] = useState('')
  const [attachments, setAttachments] = useState<{ name: string; content: string; image?: boolean; dataUrl?: string }[]>([])
  const [skillsOpen, setSkillsOpen] = useState(false)
  const [copiedKey, setCopiedKey] = useState('')
  const [questionDrafts, setQuestionDrafts] = useState<Record<string, string>>({})
  const [searchOpen, setSearchOpen] = useState(false)
  const [messageSearch, setMessageSearch] = useState('')
  const fileRef = useRef<HTMLInputElement>(null)
  const transcriptEndRef = useRef<HTMLDivElement>(null)
  const [view, setView] = useState<'chat' | 'terminal' | 'diff'>('chat')
  const [split, setSplit] = useState(false)
  const [paneTree, setPaneTree] = useState<PaneNode>(() => paneLeaf('chat', id))
  const [newPaneView, setNewPaneView] = useState<PaneView>('diff')
  const [paneReady, setPaneReady] = useState(false)
  const [commentTarget, setCommentTarget] = useState<{ file: string; line: number; side: 'new' | 'old' } | null>(null)
  const [diffFileCursor, setDiffFileCursor] = useState(0)
  const [commentBody, setCommentBody] = useState('')
  const [editingCommentId, setEditingCommentId] = useState<string | null>(null)
  const [commentDraft, setCommentDraft] = useState('')
  const [shipOpen, setShipOpen] = useState(false)
  const [worktreeOpen, setWorktreeOpen] = useState(false)
  const [shareOpen, setShareOpen] = useState(false)
  const [share, setShare] = useState<SessionShare | null>(null)
  const [prSnapshot, setPrSnapshot] = useState<PRState | null>(null)
  const [editingTitle, setEditingTitle] = useState(false)
  const [titleDraft, setTitleDraft] = useState('')
  const [editingMessageId, setEditingMessageId] = useState<string | null>(null)
  const [messageDraft, setMessageDraft] = useState('')
  const [gitJob, setGitJob] = useState<{ id: string; action: 'ship' | 'pr' | 'pr_status' } | null>(null)
  const [commitMessage, setCommitMessage] = useState('')
  const [prUrl, setPrUrl] = useState('')
  useEffect(() => {
    const saved = window.localStorage.getItem(`orca.session.pr.${id}`)
    if (saved) setPrUrl(saved)
  }, [id])
  const { data } = useQuery({
    queryKey: ['session', id],
    queryFn: () => api.session(id),
    refetchInterval: 4000,
  })
  const { data: msgData } = useQuery({
    queryKey: ['session-messages', id],
    queryFn: () => api.sessionMessages(id),
    staleTime: Infinity,
  })
  const { data: paneSessionsData } = useQuery({
    queryKey: ['sessions', 'all'],
    queryFn: () => api.sessions('all'),
    enabled: split,
  })
  const paneSessions = paneSessionsData?.sessions ?? []
  const sess = data?.session
  const { workspaceId } = useWorkspace()
  const { data: agentsData } = useQuery({
    queryKey: ['agents', workspaceId],
    queryFn: () => api.agents(workspaceId),
  })
  const agentName = agentsData?.items.find((agent) => agent.id === sess?.agentId)?.name
  const agentModel = sess?.model || agentsData?.items.find((agent) => agent.id === sess?.agentId)?.model
  const agentSkills = agentsData?.items.find((agent) => agent.id === sess?.agentId)?.skills ?? []
  const { data: worktree } = useQuery({
    queryKey: ['worktree', sess?.worktreeId],
    queryFn: () => api.worktree(sess!.worktreeId!),
    enabled: Boolean(sess?.worktreeId),
  })
  const { data: diffData } = useQuery({
    queryKey: ['session-diff', id],
    queryFn: () => api.sessionDiff(id),
    enabled: true,
    refetchInterval: view === 'diff' || split ? 5000 : false,
  })
  const { data: commentsData } = useQuery({
    queryKey: ['session-comments', id],
    queryFn: () => api.sessionComments(id),
  })
  const comments = commentsData?.items ?? []
  const openComments = comments.filter((comment) => comment.status === 'open')
  const createComment = useMutation({
    mutationFn: () => api.createSessionComment(id, {
      file: commentTarget!.file,
      line: commentTarget!.line,
      side: commentTarget!.side,
      body: commentBody,
    }),
    onSuccess: () => {
      setCommentTarget(null)
      setCommentBody('')
      void qc.invalidateQueries({ queryKey: ['session-comments', id] })
    },
  })
  const sendComments = useMutation({
    mutationFn: () => api.sendSessionComments(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['session-comments', id] })
      void qc.invalidateQueries({ queryKey: ['session-messages', id] })
      void qc.invalidateQueries({ queryKey: ['session', id] })
      void qc.invalidateQueries({ queryKey: ['sessions'] })
    },
  })
  const resolveComment = useMutation({
    mutationFn: (commentId: string) => api.resolveSessionComment(commentId),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['session-comments', id] }),
  })
  const updateComment = useMutation({
    mutationFn: ({ commentId, body }: { commentId: string; body: string }) => api.updateSessionComment(commentId, body),
    onSuccess: () => {
      setEditingCommentId(null)
      void qc.invalidateQueries({ queryKey: ['session-comments', id] })
    },
  })
  const deleteComment = useMutation({
    mutationFn: (commentId: string) => api.deleteSessionComment(commentId),
    onSuccess: () => void qc.invalidateQueries({ queryKey: ['session-comments', id] }),
  })
  const { data: questionsData } = useQuery({
    queryKey: ['session-questions', id],
    queryFn: () => api.sessionQuestions(id),
    refetchInterval: 3000,
  })
  const pendingQuestions = questionsData?.items ?? []
  const answerQuestion = useMutation({
    mutationFn: ({ questionId, answer }: { questionId: string; answer: string }) => api.answerQuestion(questionId, answer),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['session-questions', id] })
      void qc.invalidateQueries({ queryKey: ['session-messages', id] })
      void qc.invalidateQueries({ queryKey: ['session', id] })
    },
  })
  const diffRows = useMemo(() => parseDiff(diffData?.diff ?? ''), [diffData?.diff])
  const modifiedFiles = useMemo(() => [...new Set(diffRows.map((row) => row.file).filter(Boolean))], [diffRows])
  const goToDiffFile = (index: number) => {
    if (modifiedFiles.length === 0) return
    const next = Math.max(0, Math.min(modifiedFiles.length - 1, index))
    setDiffFileCursor(next)
    setView('diff')
    window.setTimeout(() => document.getElementById(diffFileId(modifiedFiles[next]))?.scrollIntoView({ block: 'start' }), 50)
  }
  const { data: gitStatus, error: gitError } = useQuery({
    queryKey: ['session-git', id],
    queryFn: () => api.sessionGit(id),
    enabled: view === 'diff' || split,
    retry: false,
  })
  const { data: gitJobData } = useQuery({
    queryKey: ['git-job', id, gitJob?.id],
    queryFn: () => api.sessionGitJob(id, gitJob!.id),
    enabled: Boolean(gitJob),
    refetchInterval: 2000,
  })
  const { data: prState } = useQuery({
    queryKey: ['session-pr', id, prUrl],
    queryFn: () => api.sessionPRStatus(id),
    enabled: Boolean(prUrl),
    refetchInterval: 15000,
    retry: false,
  })
  const prInfo = prSnapshot ?? prState
  const { data: worktreeInfo } = useQuery({
    queryKey: ['session-worktree', id],
    queryFn: () => api.sessionWorktree(id),
    enabled: Boolean(sess?.worktreeId),
    retry: false,
  })
  const archiveWorktree = useMutation({
    mutationFn: () => api.archiveSessionWorktree(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['session-worktree', id] })
      void qc.invalidateQueries({ queryKey: ['session', id] })
    },
  })
  const cleanupWorktree = useMutation({
    mutationFn: () => api.cleanupSessionWorktree(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['session-worktree', id] })
    },
  })
  const ship = useMutation({
    mutationFn: () => api.shipSession(id, commitMessage.trim() || sess?.title || 'session changes'),
    onSuccess: (result) => {
      if ('jobId' in result) {
        setGitJob({ id: result.jobId, action: 'ship' })
        return
      }
      setShipOpen(false)
      setCommitMessage('')
      void qc.invalidateQueries({ queryKey: ['session-git', id] })
      void qc.invalidateQueries({ queryKey: ['session-diff', id] })
    },
  })
  const createPR = useMutation({
    mutationFn: () => api.createSessionPR(id),
    onSuccess: (data) => {
      if ('jobId' in data) {
        setGitJob({ id: data.jobId, action: 'pr' })
        return
      }
      setPrUrl(data.url)
      window.localStorage.setItem(`orca.session.pr.${id}`, data.url)
    },
  })
  const refreshPR = useMutation({
    mutationFn: () => api.refreshSessionPR(id),
    onSuccess: (data) => {
      if ('jobId' in data) {
        setGitJob({ id: data.jobId, action: 'pr_status' })
        return
      }
      setPrSnapshot(data)
    },
  })
  useEffect(() => {
    if (!prUrl || !gitStatus?.remote || gitJob) return
    const timer = window.setInterval(() => {
      if (!refreshPR.isPending) refreshPR.mutate()
    }, 60000)
    return () => window.clearInterval(timer)
  }, [prUrl, gitStatus?.remote, gitJob, refreshPR.isPending, refreshPR.mutate])
  useEffect(() => {
    if (!gitJob || !gitJobData || gitJobData.status === 'pending') return
    if (gitJobData.status === 'succeeded') {
      try {
        const result = JSON.parse(gitJobData.resultJson || '{}') as { url?: string; state?: string; draft?: boolean; title?: string }
        if (result.url) {
          setPrUrl(result.url)
          window.localStorage.setItem(`orca.session.pr.${id}`, result.url)
        }
        if (gitJob.action === 'pr_status' && result.state) {
          setPrSnapshot({ url: result.url ?? prUrl, state: result.state, draft: Boolean(result.draft), title: result.title })
        }
      } catch {
        // keep the job result opaque
      }
      void qc.invalidateQueries({ queryKey: ['session-git', id] })
      void qc.invalidateQueries({ queryKey: ['session-diff', id] })
      if (gitJob.action === 'ship') {
        setShipOpen(false)
        setCommitMessage('')
      }
      setGitJob(null)
    }
  }, [gitJob, gitJobData, id, qc])
  useEffect(() => {
    if (sess) setView(sess.mode === 'terminal' ? 'terminal' : 'chat')
  }, [sess?.id, sess?.mode])
  useEffect(() => {
    const savedSplit = window.localStorage.getItem(`orca.session.split.${id}`) === '1'
    setSplit(savedSplit)
    const raw = window.localStorage.getItem(`orca.session.paneTree.${id}`)
    let restored = false
    if (raw) {
      try {
        const parsed = JSON.parse(raw) as PaneNode
        if (parsed && (parsed.type === 'leaf' || parsed.type === 'split')) {
          setPaneTree(parsed)
          restored = true
        }
      } catch {
        // fall through to the default layout
      }
    }
    if (!restored) setPaneTree(savedSplit ? panePreset(id) : paneLeaf('chat', id))
    setPaneReady(true)
  }, [id])
  useEffect(() => {
    if (!paneReady) return
    window.localStorage.setItem(`orca.session.paneTree.${id}`, JSON.stringify(paneTree))
  }, [paneTree, paneReady, id])
  const { data: approvalsData } = useQuery({
    queryKey: ['approvals', 'all'],
    queryFn: () => api.approvals('all'),
    refetchInterval: 3000,
  })
  const pendingApprovals: Approval[] = (approvalsData?.items ?? []).filter(
    (approval) => sess?.lastRunId && approval.runId === sess.lastRunId,
  )
  const decideApproval = useMutation({
    mutationFn: ({ id, decision }: { id: string; decision: 'approved' | 'denied' }) => api.decide(id, decision),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['approvals'] })
      void qc.invalidateQueries({ queryKey: ['session', id] })
    },
  })
  const initialMessages = msgData?.messages ?? []
  const [liveMessages, setLiveMessages] = useState<SessionMessage[]>([])
  const messages = useMemo(() => {
    const seen = new Set<string>()
    const out: SessionMessage[] = []
    for (const message of [...initialMessages, ...liveMessages]) {
      const key = message.id || `${message.runId}-${message.ts}-${message.text}`
      if (seen.has(key)) continue
      seen.add(key)
      out.push(message)
    }
    const merged: SessionMessage[] = []
    for (const message of out) {
      const previous = merged[merged.length - 1]
      if (
        previous &&
        previous.runId === message.runId &&
        previous.role === 'assistant' &&
        (previous.kind === 'message.chunk' || previous.kind === 'thought') &&
        message.role === 'assistant' &&
        message.kind === previous.kind
      ) {
        merged[merged.length - 1] = { ...previous, text: previous.text + message.text, ts: message.ts }
      } else {
        merged.push(message)
      }
    }
    return merged
  }, [initialMessages, liveMessages])
  const lastTool = [...messages].reverse().find((message) => message.kind === 'tool')
  const displayMessages = messageSearch.trim()
    ? messages.filter((message) => message.text.toLowerCase().includes(messageSearch.trim().toLowerCase()))
    : messages
  const lastAssistantIndex = displayMessages.reduce((index, message, i) => (message.role === 'assistant' ? i : index), -1)
  const after = initialMessages[initialMessages.length - 1]?.id ?? ''

  useEffect(() => {
    setLiveMessages([])
    if (!id) return
    const source = new EventSource(`/api/v1/sessions/${id}/events?after=${encodeURIComponent(after)}`)
    source.onmessage = (event) => {
      try {
        const message = JSON.parse(event.data) as SessionMessage
        setLiveMessages((prev) => {
          const key = message.id || `${message.runId}-${message.ts}-${message.text}`
          if (prev.some((item) => (item.id || `${item.runId}-${item.ts}-${item.text}`) === key)) return prev
          return [...prev, message]
        })
      } catch {
        // ignore malformed stream frames
      }
    }
    source.addEventListener('end', () => source.close())
    return () => source.close()
  }, [id, after])
  const send = useMutation({
    mutationFn: (text: string) => api.createSessionRun(id, { prompt: text }),
    onSuccess: () => {
      setPrompt('')
      setAttachments([])
      void qc.invalidateQueries({ queryKey: ['session-messages', id] })
      void qc.invalidateQueries({ queryKey: ['session', id] })
    },
  })
  const regenerate = useMutation({
    mutationFn: () => api.regenerateSession(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['session-messages', id] })
      void qc.invalidateQueries({ queryKey: ['session', id] })
    },
  })
  const editMessage = useMutation({
    mutationFn: async ({ messageId, text }: { messageId: string; text: string }) => {
      await api.editSessionMessage(id, messageId, text)
      return api.createSessionRun(id, { prompt: text, skipUserMessage: true })
    },
    onSuccess: () => {
      setEditingMessageId(null)
      setMessageDraft('')
      void qc.invalidateQueries({ queryKey: ['session-messages', id] })
      void qc.invalidateQueries({ queryKey: ['session', id] })
    },
  })
  const slashQuery = prompt.startsWith('/') && !/\s/.test(prompt) ? prompt.slice(1).toLowerCase() : ''
  const slashMatches = slashQuery
    ? SLASH_COMMANDS.filter((item) => item.command.slice(1).startsWith(slashQuery))
    : []
  const runSlash = (command: string) => {
    setPrompt('')
    switch (command) {
      case '/diff':
        setView('diff')
        break
      case '/chat':
        setView('chat')
        break
      case '/terminal':
        setView('terminal')
        break
      case '/ship':
        setShipOpen(true)
        break
      case '/worktree':
        setWorktreeOpen(true)
        break
    }
  }
  const submitPrompt = () => {
    const command = prompt.trim()
    if (SLASH_COMMANDS.some((item) => item.command === command)) {
      runSlash(command)
      return
    }
    if ((!prompt.trim() && attachments.length === 0) || send.isPending) return
    let text = prompt
    for (const attachment of attachments) {
      if (attachment.image && attachment.dataUrl) {
        text += `\n\n![${attachment.name}](${attachment.dataUrl})`
        continue
      }
      text += `\n\nAttached file: ${attachment.name}\n\`\`\`\n${attachment.content}\n\`\`\``
    }
    send.mutate(text)
  }
  const addFiles = async (files: FileList | null) => {
    if (!files) return
    const next: { name: string; content: string; image?: boolean; dataUrl?: string }[] = []
    for (const file of Array.from(files)) {
      if (file.type.startsWith('image/')) {
        if (file.size > 2_000_000) continue
        const dataUrl = await new Promise<string>((resolve) => {
          const reader = new FileReader()
          reader.onload = () => resolve(String(reader.result ?? ''))
          reader.readAsDataURL(file)
        })
        if (dataUrl) next.push({ name: file.name, content: '', image: true, dataUrl })
        continue
      }
      if (file.size > 200_000) continue
      next.push({ name: file.name, content: await file.text() })
    }
    if (next.length > 0) setAttachments((current) => [...current, ...next])
  }
  const close = useMutation({
    mutationFn: () => api.closeSession(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['session', id] })
      void qc.invalidateQueries({ queryKey: ['sessions'] })
    },
  })
  const createShare = useMutation({
    mutationFn: () => api.createSessionShare(id),
    onSuccess: (data) => {
      setShare(data)
      setShareOpen(true)
      void navigator.clipboard.writeText(`${window.location.origin}/shared/${data.token}`)
    },
  })
  const revokeShare = useMutation({
    mutationFn: () => api.revokeSessionShare(id, share?.token ?? ''),
    onSuccess: () => {
      setShare(null)
      setShareOpen(false)
    },
  })
  const rename = useMutation({
    mutationFn: (title: string) => api.renameSession(id, title),
    onSuccess: () => {
      setEditingTitle(false)
      void qc.invalidateQueries({ queryKey: ['session', id] })
      void qc.invalidateQueries({ queryKey: ['sessions'] })
    },
  })
  const archive = useMutation({
    mutationFn: () => api.archiveSession(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['session', id] })
      void qc.invalidateQueries({ queryKey: ['sessions'] })
    },
  })
  const stop = useMutation({
    mutationFn: () => api.stopSession(id),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['session', id] })
      void qc.invalidateQueries({ queryKey: ['session-messages', id] })
      void qc.invalidateQueries({ queryKey: ['sessions'] })
    },
  })
  const retry = useMutation({
    mutationFn: () => api.retryRun(sess?.lastRunId ?? ''),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ['session', id] })
      void qc.invalidateQueries({ queryKey: ['session-messages', id] })
      void qc.invalidateQueries({ queryKey: ['sessions'] })
    },
  })
  const busy = ['queued', 'leased', 'preparing', 'running', 'finalizing'].includes(sess?.lastRunStatus ?? '')
  const waiting = sess?.lastRunStatus === 'waiting_approval'
  const queued = sess?.lastRunStatus === 'queued'
  useEffect(() => {
    transcriptEndRef.current?.scrollIntoView({ block: 'end' })
  }, [messages.length, busy])
  const toggleSplit = () => {
    setSplit((value) => {
      const next = !value
      if (next && paneTree.type === 'leaf') setPaneTree(panePreset(id))
      window.localStorage.setItem(`orca.session.split.${id}`, next ? '1' : '0')
      return next
    })
  }
  const copyMessage = (key: string, text: string) => {
    void navigator.clipboard.writeText(text)
    setCopiedKey(key)
    window.setTimeout(() => setCopiedKey(''), 1200)
  }
  const splitPane = (id: string, direction: 'row' | 'column') => {
    const node = findPaneNode(paneTree, id)
    if (!node || node.type !== 'leaf') return
    const next: PaneSplit = {
      type: 'split', id: newPaneID('split'), direction, ratio: 50,
      children: [node, paneLeaf(newPaneView, id)],
    }
    setPaneTree(replacePaneNode(paneTree, id, next))
  }
  const closePane = (id: string) => {
    const next = removePaneNode(paneTree, id)
    if (next) setPaneTree(next)
  }
  const startPaneResize = (event: ReactPointerEvent<HTMLDivElement>, node: PaneSplit) => {
    event.preventDefault()
    const rect = event.currentTarget.parentElement?.getBoundingClientRect()
    if (!rect) return
    const move = (pointer: PointerEvent) => {
      const ratio = node.direction === 'column'
        ? (pointer.clientY - rect.top) / rect.height
        : (pointer.clientX - rect.left) / rect.width
      setPaneTree((tree) => updatePaneRatio(tree, node.id, Math.min(80, Math.max(20, Math.round(ratio * 100)))))
    }
    const stop = () => {
      window.removeEventListener('pointermove', move)
      window.removeEventListener('pointerup', stop)
    }
    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', stop)
  }
  const approvalCards = pendingApprovals.map((approval) => (
    <div key={approval.id} className="rounded-xl border border-warn/40 bg-warn/5 p-3">
      <div className="flex items-center gap-2 text-[12px] text-warn">
        <ShieldCheck className="h-3.5 w-3.5" />
        <span className="font-medium">{approval.tool}</span>
        <span className="text-[10px] uppercase">{approval.risk}</span>
        <span className="ml-auto text-[10px] text-muted">Needs your approval</span>
      </div>
      {approval.command && (
        <pre className="mt-1 overflow-x-auto rounded border border-warn/20 bg-bg px-2 py-1.5 font-mono text-[11px] text-fg/80">
          {approval.command}
        </pre>
      )}
      <div className="mt-2 flex justify-end gap-2">
        <Button
          size="sm"
          variant="secondary"
          disabled={decideApproval.isPending}
          onClick={() => decideApproval.mutate({ id: approval.id, decision: 'denied' })}
        >
          <X className="h-3 w-3" /> Deny
        </Button>
        <Button
          size="sm"
          variant="primary"
          disabled={decideApproval.isPending}
          onClick={() => decideApproval.mutate({ id: approval.id, decision: 'approved' })}
        >
          <Check className="h-3 w-3" /> Approve
        </Button>
      </div>
    </div>
  ))
  const questionCards = pendingQuestions.map((question) => (
    <div key={question.id} className="rounded-xl border border-line bg-panel/40 p-3">
      <div className="flex items-center gap-2 text-[12px] text-fg">
        <HelpCircle className="h-3.5 w-3.5 text-muted" />
        <span className="font-medium">Agent question</span>
      </div>
      <p className="mt-1.5 whitespace-pre-wrap text-[12.5px] text-fg">{question.question}</p>
      {question.expiresAt && <p className="mt-1 text-[10px] text-muted">Expires {new Date(question.expiresAt).toLocaleTimeString()}</p>}
      {question.options && question.options.length > 0 && (
        <div className="mt-2 flex flex-wrap gap-1.5">
          {question.options.map((option) => (
            <Button
              key={option}
              size="sm"
              variant="secondary"
              disabled={answerQuestion.isPending}
              onClick={() => answerQuestion.mutate({ questionId: question.id, answer: option })}
            >
              {option}
            </Button>
          ))}
        </div>
      )}
      <div className="mt-2 flex gap-2">
        <Input
          value={questionDrafts[question.id] ?? ''}
          onChange={(event) => setQuestionDrafts((drafts) => ({ ...drafts, [question.id]: event.target.value }))}
          placeholder="Type an answer"
          onKeyDown={(event) => {
            if (event.key === 'Enter' && (questionDrafts[question.id] ?? '').trim()) {
              event.preventDefault()
              answerQuestion.mutate({ questionId: question.id, answer: questionDrafts[question.id].trim() })
            }
          }}
        />
        <Button
          size="sm"
          variant="primary"
          disabled={answerQuestion.isPending || !(questionDrafts[question.id] ?? '').trim()}
          onClick={() => answerQuestion.mutate({ questionId: question.id, answer: questionDrafts[question.id].trim() })}
        >
          Answer
        </Button>
      </div>
      {answerQuestion.error && <p className="mt-1 text-[10.5px] text-err">{String(answerQuestion.error.message)}</p>}
    </div>
  ))
  const transcript = (
    <div className="mx-auto flex w-full max-w-[860px] flex-col gap-5">
      {searchOpen && (
        <div className="flex items-center gap-2 rounded-md border border-line bg-panel px-2 py-1.5">
          <Search className="h-3.5 w-3.5 text-muted" />
          <Input
            autoFocus
            value={messageSearch}
            onChange={(event) => setMessageSearch(event.target.value)}
            placeholder="Search this conversation"
            className="h-7 flex-1 border-0 bg-transparent px-0 focus:ring-0"
          />
          <span className="text-[10.5px] text-muted">{displayMessages.length}/{messages.length}</span>
          <button onClick={() => { setSearchOpen(false); setMessageSearch('') }} className="text-muted hover:text-fg">
            <X className="h-3.5 w-3.5" />
          </button>
        </div>
      )}
      {displayMessages.length === 0 && messageSearch && (
        <p className="py-6 text-center text-[12px] text-muted">No messages match “{messageSearch}”.</p>
      )}
      {displayMessages.map((m, i) => {
        const key = `${m.runId}-${i}`
        const previous = displayMessages[i - 1]
        const next = displayMessages[i + 1]
        if (m.role === 'user') {
          const userMessage = splitInlineImages(m.text)
          const messageId = m.id ?? key
          if (editingMessageId === messageId) {
            return (
              <div key={key} className="flex justify-end">
                <div className="w-full max-w-[78%] rounded-2xl border border-line bg-panel2 p-2">
                  <textarea
                    autoFocus
                    value={messageDraft}
                    onChange={(event) => setMessageDraft(event.target.value)}
                    rows={3}
                    className="w-full resize-none rounded border border-line2 bg-panel px-2 py-1.5 text-[12.5px] text-fg"
                  />
                  <div className="mt-1 flex justify-end gap-1">
                    <Button size="sm" variant="ghost" onClick={() => setEditingMessageId(null)}>Cancel</Button>
                    <Button
                      size="sm"
                      variant="primary"
                      disabled={editMessage.isPending || !messageDraft.trim()}
                      onClick={() => editMessage.mutate({ messageId, text: messageDraft.trim() })}
                    >
                      Save & resend
                    </Button>
                  </div>
                </div>
              </div>
            )
          }
          return (
            <div key={key} className="group flex items-center justify-end gap-1">
              <button
                onClick={() => {
                  setEditingMessageId(messageId)
                  setMessageDraft(m.text)
                }}
                className="text-muted opacity-0 transition-opacity hover:text-fg group-hover:opacity-100"
                title="Edit message"
              >
                <Pencil className="h-3.5 w-3.5" />
              </button>
              <button
                onClick={() => copyMessage(key, m.text)}
                className="text-muted opacity-0 transition-opacity hover:text-fg group-hover:opacity-100"
                title="Copy message"
              >
                {copiedKey === key ? <Check className="h-3.5 w-3.5 text-ok" /> : <Copy className="h-3.5 w-3.5" />}
              </button>
              <div className="max-w-[78%] whitespace-pre-wrap rounded-2xl bg-panel2 px-4 py-2.5 text-[12.5px] leading-relaxed text-fg">
                {userMessage.images.length > 0 && (
                  <div className="mb-2 flex flex-wrap gap-2">
                    {userMessage.images.map((image) => (
                      <img key={image.src.slice(0, 64)} src={image.src} alt={image.alt} className="max-h-56 rounded-lg border border-line object-cover" />
                    ))}
                  </div>
                )}
                {userMessage.text}
              </div>
            </div>
          )
        }
        if (m.kind === 'tool') {
          const space = m.text.indexOf(' ')
          const tool = space > 0 ? m.text.slice(0, space) : m.text
          const rawArgs = space > 0 ? m.text.slice(space + 1) : ''
          const args = rawArgs === '{}' ? '' : rawArgs
          const meta = toolMeta(tool)
          const ToolIcon = meta.Icon
          const result = next?.kind === 'result' && next.runId === m.runId ? next : null
          const resultText = result?.text ?? ''
          const divider = resultText.indexOf(': ')
          const resultStatus = divider >= 0 ? resultText.slice(0, divider) : resultText
          const resultOutput = divider >= 0 ? resultText.slice(divider + 2) : ''
          const statusClass = /fail|error|denied/i.test(resultStatus) ? 'text-err' : 'text-ok'
          return (
            <details key={key} className="group ml-10 rounded-lg border border-line bg-panel/30">
              <summary className="flex cursor-pointer list-none items-center gap-2 px-3 py-1.5 text-[11px] text-muted">
                <ToolIcon className="h-3 w-3 shrink-0" />
                <span className="shrink-0 text-[9.5px] uppercase tracking-wide text-muted">{meta.label}</span>
                <span className="shrink-0 font-medium text-fg/80">{tool}</span>
                <span className="min-w-0 truncate font-mono text-[10.5px]">{args}</span>
                {result && <Check className={`ml-1 h-3 w-3 shrink-0 ${statusClass}`} />}
                <ChevronDown className="ml-auto h-3 w-3 shrink-0 transition-transform group-open:rotate-180" />
              </summary>
              {meta.label === 'Command' ? (
                <div className="border-t border-line bg-[#0b0d10] px-3 py-2 font-mono text-[11px]">
                  <div className="flex gap-2 text-fg/80">
                    <span className="select-none text-muted">$</span>
                    <span className="break-all">{args}</span>
                  </div>
                  {resultOutput && (
                    <pre className="mt-1 overflow-x-auto whitespace-pre-wrap border-t border-line/60 pt-1 text-fg/65">{resultOutput}</pre>
                  )}
                  {resultStatus && <div className={`mt-1 text-[9.5px] uppercase tracking-wide ${statusClass}`}>{resultStatus}</div>}
                </div>
              ) : (
                <div className="border-t border-line px-3 py-2">
                  <div className="flex items-center gap-2 font-mono text-[11px] text-fg/75">
                    <ToolIcon className="h-3 w-3 shrink-0 text-muted" />
                    <span className="break-all">{args || tool}</span>
                  </div>
                  {resultOutput && <pre className="mt-1 overflow-x-auto whitespace-pre-wrap text-[11px] text-fg/65">{resultOutput}</pre>}
                  {resultStatus && <div className={`mt-1 text-[9.5px] uppercase tracking-wide ${statusClass}`}>{resultStatus}</div>}
                </div>
              )}
            </details>
          )
        }
        if (m.kind === 'thought') {
          return (
            <details key={key} className="ml-10 text-[11px] text-muted">
              <summary className="cursor-pointer list-none">Thought</summary>
              <p className="mt-1 whitespace-pre-wrap italic">{m.text}</p>
            </details>
          )
        }
        if (m.kind === 'result' && previous?.kind === 'tool' && previous.runId === m.runId) {
          return null
        }
        if (m.role === 'system') {
          return (
            <div key={key} className="ml-10 flex items-center gap-2 text-[10.5px] text-muted">
              <Check className="h-3 w-3 text-ok" />
              <span className="uppercase tracking-wide">{m.kind}</span>
              <span className="truncate">{m.text}</span>
            </div>
          )
        }
        const assistantText = m.text.replace(/^Warning: Model metadata for `[^`]+` not found\.[^\n]*(?:\n|$)/, '')
        return (
          <div key={key} className="group flex gap-3">
            <div className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-line bg-panel2 text-muted">
              <Bot className="h-3.5 w-3.5" />
            </div>
            <div className="min-w-0 flex-1">
              <div className="mb-1 flex items-center gap-2 text-[10.5px] text-muted">
                <span className="font-medium text-fg/80">{agentName ?? 'Agent'}</span>
                <span>{new Date(m.ts).toLocaleTimeString()}</span>
                <button
                  onClick={() => copyMessage(key, m.text)}
                  className="text-muted opacity-0 transition-opacity hover:text-fg group-hover:opacity-100"
                  title="Copy message"
                >
                  {copiedKey === key ? <Check className="h-3 w-3 text-ok" /> : <Copy className="h-3 w-3" />}
                </button>
                {i === lastAssistantIndex && !busy && (
                  <button
                    onClick={() => regenerate.mutate()}
                    className="text-muted opacity-0 transition-opacity hover:text-fg group-hover:opacity-100 disabled:opacity-30"
                    title="Regenerate response"
                    disabled={regenerate.isPending}
                  >
                    <RotateCcw className="h-3 w-3" />
                  </button>
                )}
              </div>
              <div className="text-[12.5px] leading-relaxed text-fg [&_code]:rounded [&_code]:bg-panel2 [&_code]:px-1 [&_code]:py-0.5 [&_code]:font-mono [&_code]:text-[11.5px] [&_h1]:mb-2 [&_h1]:text-[14px] [&_h2]:mb-2 [&_h2]:text-[13px] [&_li]:ml-4 [&_li]:list-disc [&_p]:mb-2 [&_pre]:mb-2 [&_pre]:overflow-x-auto [&_pre]:rounded [&_pre]:border [&_pre]:border-line [&_pre]:bg-bg [&_pre]:p-2 [&_a]:underline [&_img]:max-h-64 [&_img]:rounded-lg [&_img]:border [&_img]:border-line">
                <ReactMarkdown remarkPlugins={[remarkGfm]}>{assistantText}</ReactMarkdown>
                {busy && i === lastAssistantIndex && <span className="ml-0.5 inline-block h-3.5 w-1.5 animate-pulse bg-fg/70 align-middle" />}
              </div>
            </div>
          </div>
        )
      })}

      {questionCards}
      {approvalCards}

      {busy && (
        <div className="flex gap-3">
          <div className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full border border-line bg-panel2 text-muted">
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          </div>
          <div className="text-[12px] text-muted">
            <span className="font-medium text-fg/80">{queued ? 'Queued' : 'Working…'}</span>
            <p className="mt-0.5">
              {queued ? 'This turn starts after the current run.' : lastTool ? `Running ${lastTool.text.split(' ')[0]}…` : 'The agent is running this turn.'}
            </p>
          </div>
        </div>
      )}

      {modifiedFiles.length > 0 && !busy && (
        <div className="ml-10 rounded-lg border border-line bg-panel/30 px-3 py-2">
          <div className="flex items-center gap-2 text-[11px] text-muted">
            <FileCode2 className="h-3 w-3" />
            Modified {modifiedFiles.length} file{modifiedFiles.length === 1 ? '' : 's'}
          </div>
          <div className="mt-1 flex flex-wrap gap-x-3 gap-y-1 font-mono text-[10.5px] text-fg/70">
            {modifiedFiles.map((file, index) => (
              <button
                key={file}
                onClick={() => goToDiffFile(index)}
                className="text-left hover:underline"
              >
                {file}
              </button>
            ))}
          </div>
        </div>
      )}

      {messages.length === 0 && <p className="text-[12px] text-muted">{t('session.noMessages')}</p>}
      <div ref={transcriptEndRef} />
    </div>
  )
  const diffView = (
    <div className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border border-line bg-bg">
      <div className="flex shrink-0 items-center gap-2 border-b border-line px-3 py-1.5">
        <span className="text-[10.5px] text-muted">{openComments.length} open review comments</span>
        {modifiedFiles.length > 0 && (
          <div className="flex items-center gap-1">
            <button onClick={() => goToDiffFile(diffFileCursor - 1)} className="px-1 text-[12px] text-muted hover:text-fg" title="Previous file">‹</button>
            <select
              value={modifiedFiles[Math.min(diffFileCursor, modifiedFiles.length - 1)] ?? ''}
              onChange={(event) => goToDiffFile(modifiedFiles.indexOf(event.target.value))}
              className="max-w-[260px] truncate rounded border border-line bg-panel px-1 py-0.5 font-mono text-[10px] text-fg"
            >
              {modifiedFiles.map((file) => <option key={file} value={file}>{file}</option>)}
            </select>
            <button onClick={() => goToDiffFile(diffFileCursor + 1)} className="px-1 text-[12px] text-muted hover:text-fg" title="Next file">›</button>
          </div>
        )}
        {gitStatus?.commits && gitStatus.commits.length > 0 && (
          <details className="relative">
            <summary className="cursor-pointer list-none text-[10px] text-muted hover:text-fg">History</summary>
            <div className="absolute left-0 top-5 z-20 w-[420px] rounded-md border border-line bg-panel p-2 shadow-lg">
              {gitStatus.commits.map((commit) => (
                <div key={commit.hash} className="flex gap-2 border-b border-line/60 py-1 last:border-0">
                  <span className="font-mono text-[9.5px] text-muted">{commit.hash}</span>
                  <div className="min-w-0 flex-1">
                    <p className="truncate text-[10.5px] text-fg/80">{commit.subject}</p>
                    <p className="text-[9px] text-muted">{commit.author} · {commit.at}</p>
                  </div>
                </div>
              ))}
            </div>
          </details>
        )}
        {sendComments.error && <span className="text-[10px] text-err">{String(sendComments.error.message)}</span>}
        <div className="ml-auto flex items-center gap-2">
          {gitStatus && (
            <span className="flex items-center gap-1 text-[10px] text-muted">
              <GitBranch className="h-3 w-3" />
              {gitStatus.branch} · {gitStatus.remote ? 'remote' : gitStatus.clean ? 'clean' : `${gitStatus.changes.length} changed`}
            </span>
          )}
          {prInfo && (
            <span className="flex items-center gap-1 rounded border border-line px-1.5 py-0.5 text-[9.5px] uppercase text-muted" title={prInfo.title}>
              PR {prInfo.state}{prInfo.draft ? ' · draft' : ''}
              <button onClick={() => refreshPR.mutate()} disabled={refreshPR.isPending} title={t('session.refreshPR')} className="hover:text-fg">
                <RefreshCw className={`h-2.5 w-2.5 ${refreshPR.isPending ? 'animate-spin' : ''}`} />
              </button>
            </span>
          )}
          {!gitStatus && gitError && <span className="text-[10px] text-muted">{t('session.gitUnavailable')}</span>}
          <Button
            size="sm"
            variant="secondary"
            disabled={!gitStatus || (gitStatus.clean && !gitStatus.remote) || !sess}
            onClick={() => setShipOpen(true)}
          >
            <Upload className="h-3 w-3" /> {t('session.ship')}
          </Button>
          <Button
            size="sm"
            variant="primary"
            disabled={openComments.length === 0 || sendComments.isPending}
            onClick={() => sendComments.mutate()}
          >
            Send to agent
          </Button>
        </div>
      </div>
      <div className="min-h-0 flex-1 overflow-auto">
        {diffData?.diff ? (
          <div className="font-mono text-[11.5px] leading-relaxed">
            {diffRows.map((row, index) => {
              const lineComments = comments.filter(
                (comment) => comment.file === row.file && comment.line === row.line && comment.side === row.side,
              )
              const canComment = row.kind !== 'meta' && row.kind !== 'hunk' && row.file && row.line > 0
              const target = commentTarget
              const editing = target?.file === row.file && target?.line === row.line && target?.side === row.side
              return (
                <div key={index}>
                  <div
                    id={row.kind === 'meta' && row.text.startsWith('diff --git') ? diffFileId(row.file) : undefined}
                    onClick={() => {
                      if (canComment) {
                        setCommentTarget({ file: row.file, line: row.line, side: row.side })
                        setCommentBody('')
                      }
                    }}
                    className={
                      row.kind === 'add'
                        ? 'flex cursor-pointer gap-2 bg-ok/10 px-2 text-ok hover:bg-ok/15'
                        : row.kind === 'del'
                          ? 'flex cursor-pointer gap-2 bg-err/10 px-2 text-err hover:bg-err/15'
                          : row.kind === 'hunk'
                            ? 'flex gap-2 bg-panel2 px-2 py-0.5 text-muted'
                            : row.kind === 'meta'
                              ? 'flex gap-2 px-2 py-0.5 text-muted/70'
                              : 'flex cursor-pointer gap-2 px-2 text-fg/70 hover:bg-panel2/60'
                    }
                  >
                    <span className="w-9 shrink-0 select-none text-right text-[10px] text-muted/60">{row.line || ''}</span>
                    <span className="whitespace-pre-wrap break-all">{row.text || ' '}</span>
                  </div>
                  {lineComments.map((comment) => (
                    <div key={comment.id} className="ml-12 border-l border-warn/40 bg-warn/5 px-2 py-1 text-[11px] text-fg/80">
                      <span className="mr-1 text-[9.5px] uppercase text-warn">{comment.status}</span>
                      {editingCommentId === comment.id ? (
                        <div className="mt-1">
                          <textarea
                            autoFocus
                            value={commentDraft}
                            onChange={(event) => setCommentDraft(event.target.value)}
                            rows={2}
                            className="w-full resize-none rounded border border-line2 bg-panel px-2 py-1 text-[11px] text-fg"
                          />
                          <div className="mt-1 flex justify-end gap-1">
                            <Button size="sm" variant="ghost" onClick={() => setEditingCommentId(null)}>Cancel</Button>
                            <Button
                              size="sm"
                              variant="primary"
                              disabled={updateComment.isPending || !commentDraft.trim()}
                              onClick={() => updateComment.mutate({ commentId: comment.id, body: commentDraft.trim() })}
                            >
                              Save
                            </Button>
                          </div>
                        </div>
                      ) : (
                        <>
                          {comment.body}
                          {comment.status === 'open' && (
                            <button
                              onClick={(event) => {
                                event.stopPropagation()
                                resolveComment.mutate(comment.id)
                              }}
                              className="ml-2 text-[10px] text-muted hover:text-fg"
                            >
                              Resolve
                            </button>
                          )}
                          <button
                            onClick={(event) => {
                              event.stopPropagation()
                              setEditingCommentId(comment.id)
                              setCommentDraft(comment.body)
                            }}
                            className="ml-2 text-[10px] text-muted hover:text-fg"
                          >
                            Edit
                          </button>
                          <button
                            onClick={(event) => {
                              event.stopPropagation()
                              deleteComment.mutate(comment.id)
                            }}
                            className="ml-2 text-[10px] text-muted hover:text-err"
                          >
                            Delete
                          </button>
                        </>
                      )}
                    </div>
                  ))}
                  {editing && (
                    <div className="ml-12 border-l border-line bg-panel2/50 p-2">
                      <textarea
                        autoFocus
                        value={commentBody}
                        onChange={(event) => setCommentBody(event.target.value)}
                        rows={2}
                        placeholder={`Comment on ${row.file}:${row.line}`}
                        className="w-full resize-none rounded border border-line2 bg-panel px-2 py-1 text-[11.5px] text-fg"
                      />
                      <div className="mt-1 flex justify-end gap-1">
                        <Button size="sm" variant="ghost" onClick={() => setCommentTarget(null)}>Cancel</Button>
                        <Button
                          size="sm"
                          variant="primary"
                          disabled={!commentBody.trim() || createComment.isPending}
                          onClick={() => createComment.mutate()}
                        >
                          Comment
                        </Button>
                      </div>
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        ) : (
          <p className="p-4 text-[12px] text-muted">No changes in this session yet.</p>
        )}
      </div>
      {diffData?.truncated && <p className="shrink-0 border-t border-line px-3 py-1 text-[10px] text-warn">Diff truncated</p>}
    </div>
  )
  const composer = sess && sess.status !== 'closed' ? (
    <form
      className="mt-3 flex flex-col gap-2"
      onSubmit={(e) => {
        e.preventDefault()
        submitPrompt()
      }}
    >
      {attachments.length > 0 && (
        <div className="flex flex-wrap gap-1">
          {attachments.map((attachment, index) => (
            <span key={`${attachment.name}-${index}`} className="flex items-center gap-1 rounded border border-line bg-panel2 px-1.5 py-0.5 text-[10.5px] text-muted">
              {attachment.image && attachment.dataUrl && (
                <img src={attachment.dataUrl} alt={attachment.name} className="h-9 w-9 rounded object-cover" />
              )}
              {attachment.name}
              <button
                type="button"
                onClick={() => setAttachments((current) => current.filter((_, i) => i !== index))}
                className="hover:text-fg"
              >
                <X className="h-3 w-3" />
              </button>
            </span>
          ))}
        </div>
      )}
      {slashMatches.length > 0 && (
        <div className="mb-2 overflow-hidden rounded-md border border-line bg-panel">
          {slashMatches.map((item) => (
            <button
              key={item.command}
              type="button"
              onClick={() => runSlash(item.command)}
              className="flex w-full items-center gap-3 px-3 py-1.5 text-left text-[11.5px] text-muted hover:bg-panel2 hover:text-fg"
            >
              <Slash className="h-3 w-3 shrink-0" />
              <span className="font-mono text-fg/80">{item.command}</span>
              <span className="truncate">{item.description}</span>
            </button>
          ))}
        </div>
      )}
      <div className="flex gap-2">
        <input
          ref={fileRef}
          type="file"
          multiple
          className="hidden"
          onChange={(event) => {
            void addFiles(event.target.files)
            event.target.value = ''
          }}
        />
        <Button type="button" variant="ghost" size="sm" onClick={() => fileRef.current?.click()} title="Attach files or images">
          <Paperclip className="h-3.5 w-3.5" />
        </Button>
        {agentSkills.length > 0 && (
          <div className="relative">
            <Button type="button" variant="ghost" size="sm" onClick={() => setSkillsOpen((value) => !value)} title="Use a skill">
              <Sparkles className="h-3.5 w-3.5" />
            </Button>
            {skillsOpen && (
              <div className="absolute bottom-8 left-0 z-10 w-52 rounded-md border border-line bg-panel p-1 shadow-lg">
                {agentSkills.map((skill) => (
                  <button
                    key={skill}
                    type="button"
                    onClick={() => {
                      setPrompt((value) => `${value}${value ? '\n\n' : ''}Use the ${skill} skill.`)
                      setSkillsOpen(false)
                    }}
                    className="block w-full truncate rounded px-2 py-1.5 text-left text-[11.5px] text-muted hover:bg-panel2 hover:text-fg"
                  >
                    {skill}
                  </button>
                ))}
              </div>
            )}
          </div>
        )}
        <textarea
          value={prompt}
          onChange={(e) => setPrompt(e.target.value)}
          onKeyDown={(event) => {
            if ((event.metaKey || event.ctrlKey) && event.key === 'Enter' && (prompt.trim() || attachments.length > 0) && !send.isPending) {
              event.preventDefault()
              submitPrompt()
            }
          }}
          rows={3}
          placeholder={t('session.promptPlaceholder')}
          className="min-w-0 flex-1 resize-none rounded-md border border-line2 bg-panel px-3 py-2 text-[12.5px] text-fg placeholder:text-muted/60 focus:border-line2 focus:outline-none"
        />
        <Button type="submit" variant="primary" disabled={(!prompt.trim() && attachments.length === 0) || send.isPending}>{t('session.send')}</Button>
      </div>
    </form>
  ) : null

  if (pane) {
    return (
      <div className="flex h-full min-h-0 min-w-0 flex-col">
        {paneView === 'chat' && (
          <>
            <div className="min-h-0 flex-1 overflow-y-auto p-2">{transcript}</div>
            {composer && <div className="border-t border-line p-2">{composer}</div>}
          </>
        )}
        {paneView === 'diff' && <div className="flex min-h-0 flex-1 flex-col p-1">{diffView}</div>}
        {paneView === 'terminal' && sess && <TerminalPanel session={sess} />}
      </div>
    )
  }

  const renderPane = (node: PaneNode): ReactNode => {
    if (node.type === 'split') {
      return (
        <div className={`flex h-full min-h-0 min-w-0 flex-1 ${node.direction === 'row' ? 'flex-row' : 'flex-col'} gap-1`}>
          <div style={{ flexBasis: `${node.ratio}%`, flexGrow: 0, flexShrink: 0 }} className="flex min-h-0 min-w-0">
            {renderPane(node.children[0])}
          </div>
          <div
            onPointerDown={(event) => startPaneResize(event, node)}
            className={
              node.direction === 'row'
                ? 'w-1 shrink-0 cursor-col-resize rounded bg-line hover:bg-muted/50'
                : 'h-1 shrink-0 cursor-row-resize rounded bg-line hover:bg-muted/50'
            }
          />
          <div className="flex min-h-0 min-w-0 flex-1">{renderPane(node.children[1])}</div>
        </div>
      )
    }
    const leafSessionId = node.sessionId ?? id
    const foreign = leafSessionId !== id
    const leafTitle = paneSessions.find((item) => item.id === leafSessionId)?.title ?? (leafSessionId === id ? sess?.title : leafSessionId)
    return (
      <div
        data-pane-leaf={node.id}
        onDragOver={(event) => event.preventDefault()}
        onDrop={(event) => {
          const dropped = event.dataTransfer.getData('text/orca-session')
          if (!dropped) return
          event.preventDefault()
          setPaneTree((tree) => replacePaneNode(tree, node.id, { ...node, sessionId: dropped }))
        }}
        className="flex h-full min-h-0 min-w-0 flex-1 flex-col overflow-hidden rounded-lg border border-line bg-panel/20"
      >
        <div className="flex shrink-0 items-center gap-2 border-b border-line px-2 py-1">
          <select
            value={leafSessionId}
            onChange={(event) => setPaneTree((tree) => replacePaneNode(tree, node.id, { ...node, sessionId: event.target.value }))}
            aria-label={t('session.paneSession')}
            className="min-w-0 max-w-[190px] truncate bg-transparent text-[10px] text-fg/80"
            title={leafTitle}
          >
            {!paneSessions.some((item) => item.id === leafSessionId) && <option value={leafSessionId}>{leafTitle}</option>}
            {paneSessions.map((item) => <option key={item.id} value={item.id}>{item.title}</option>)}
          </select>
          <span className="shrink-0 text-[9px] uppercase tracking-wider text-muted">
            {node.view === 'chat' ? t('session.chatTab') : node.view === 'terminal' ? t('session.terminalTab') : t('session.diffTab')}
          </span>
          <div className="ml-auto flex items-center gap-1">
            <button onClick={() => splitPane(node.id, 'row')} className="px-1 text-[11px] text-muted hover:text-fg" title={t('session.splitRight')}>↔</button>
            <button onClick={() => splitPane(node.id, 'column')} className="px-1 text-[11px] text-muted hover:text-fg" title={t('session.splitDown')}>↕</button>
            {paneTree.type === 'split' && (
              <button onClick={() => closePane(node.id)} className="px-1 text-[11px] text-muted hover:text-fg" title={t('session.closePane')}>×</button>
            )}
          </div>
        </div>
        {foreign ? (
          <SessionDetail id={leafSessionId} pane paneView={node.view} />
        ) : (
          <>
            {node.view === 'chat' && (
              <>
                <div className="min-h-0 flex-1 overflow-y-auto p-2">{transcript}</div>
                {composer && <div className="border-t border-line p-2">{composer}</div>}
              </>
            )}
            {node.view === 'diff' && <div className="flex min-h-0 flex-1 flex-col p-1">{diffView}</div>}
            {node.view === 'terminal' && sess && <TerminalPanel session={sess} />}
          </>
        )}
      </div>
    )
  }
  return (
    <div className={embedded ? 'flex h-full min-h-0 flex-col p-4' : 'flex h-screen min-h-0 flex-col p-4'}>
      <div className="flex flex-wrap items-center gap-2">
        {!embedded && <Link to="/sessions" className="text-muted hover:text-fg"><ArrowLeft className="h-4 w-4" /></Link>}
        <div className="min-w-0 flex-1">
          {editingTitle ? (
            <Input
              autoFocus
              value={titleDraft}
              onChange={(event) => setTitleDraft(event.target.value)}
              onBlur={() => {
                const next = titleDraft.trim()
                if (next && next !== sess?.title) rename.mutate(next)
                else setEditingTitle(false)
              }}
              onKeyDown={(event) => {
                if (event.key === 'Enter') {
                  event.preventDefault()
                  const next = titleDraft.trim()
                  if (next) rename.mutate(next)
                }
                if (event.key === 'Escape') setEditingTitle(false)
              }}
              className="h-8 text-[15px] font-semibold"
            />
          ) : (
            <button
              onClick={() => {
                setTitleDraft(sess?.title ?? '')
                setEditingTitle(true)
              }}
              className="max-w-full text-left"
              title="Rename session"
            >
              <h1 className="truncate text-[16px] font-semibold tracking-tight">{sess?.title ?? t('page.session')}</h1>
            </button>
          )}
          <p className="truncate text-[11px] text-muted">
            {agentName ?? sess?.agentId} · {sess?.backend || sess?.mode}
            {agentModel ? ` · ${agentModel}` : ''}
            {sess?.reasoning ? ` · ${sess.reasoning} reasoning` : ''}
            {sess?.createdBy ? ` · ${sess.createdBy}` : ''} ·{' '}
            {waiting ? t('session.waitingForYou') : busy ? t('session.working') : sess?.status}
            {sess?.repo ? ` · ${sess.repo}` : ''}
            {worktree?.branch ? ` · ${worktree.branch}` : ''}
          </p>
        </div>
        <div className="flex w-full flex-wrap items-center gap-2 md:ml-auto md:w-auto">
          {busy && (
          <Button size="sm" variant="secondary" disabled={stop.isPending || !sess?.lastRunId} onClick={() => stop.mutate()}>
            <Square className="h-3 w-3" /> {t('session.stop')}
          </Button>
        )}
        {!busy && (sess?.lastRunStatus === 'failed' || sess?.lastRunStatus === 'cancelled') && (
          <Button size="sm" variant="secondary" disabled={retry.isPending} onClick={() => retry.mutate()}>
            <RotateCcw className="h-3 w-3" /> {t('session.retry')}
          </Button>
        )}
        <Button size="sm" variant="ghost" disabled={createShare.isPending} onClick={() => createShare.mutate()}>
          <Share2 className="h-3 w-3" /> {t('session.share')}
        </Button>
        <Button
          size="sm"
          variant="ghost"
          onClick={() => window.open(`/sessions/${id}?popout=1`, `orca-session-${id}`, 'width=1100,height=800')}
        >
          <ExternalLink className="h-3 w-3" /> {t('session.popout')}
        </Button>
        {sess?.worktreeId && (
          <Button size="sm" variant="ghost" onClick={() => setWorktreeOpen(true)}>
            <GitBranch className="h-3 w-3" /> {t('session.worktree')}
          </Button>
        )}
        {sess && sess.status !== 'closed' && (
          <>
            <Button size="sm" variant="ghost" disabled={archive.isPending} onClick={() => archive.mutate()} title={t('session.archiveTitle')}>
              {t('session.archive')}
            </Button>
            <Button size="sm" variant="ghost" disabled={close.isPending} onClick={() => close.mutate()}>{t('session.close')}</Button>
          </>
        )}
        </div>
      </div>

      <div className="mt-2 flex gap-1 border-b border-line">
        {(['chat', 'terminal', 'diff'] as const).map((value) => (
          <button
            key={value}
            onClick={() => setView(value)}
            className={
              view === value
                ? 'border-b-2 border-fg px-2 py-1.5 text-[11.5px] font-medium text-fg'
                : 'px-2 py-1.5 text-[11.5px] text-muted hover:text-fg'
            }
          >
            {value === 'chat' ? t('session.chatTab') : value === 'terminal' ? t('session.terminalTab') : t('session.diffTab')}
          </button>
        ))}
        <button
          onClick={() => {
            setSearchOpen((value) => !value)
            if (searchOpen) setMessageSearch('')
          }}
          className="ml-auto px-2 py-1.5 text-[11px] text-muted hover:text-fg"
          title={t('session.searchMessages')}
        >
          <Search className="h-3.5 w-3.5" />
        </button>
        <button
          onClick={toggleSplit}
          className="px-2 py-1.5 text-[11px] text-muted hover:text-fg"
          title={t('session.toggleSplit')}
        >
          {split ? t('session.singlePane') : t('session.splitPane')}
        </button>
        {split && (
          <>
            <select
              value={newPaneView}
              onChange={(event) => setNewPaneView(event.target.value as PaneView)}
              className="rounded border border-line bg-panel px-1.5 text-[10.5px] text-muted"
              title={t('session.newPaneView')}
            >
              <option value="chat">{t('session.newPaneChat')}</option>
              <option value="diff">{t('session.newPaneDiff')}</option>
              <option value="terminal">{t('session.newPaneTerminal')}</option>
            </select>
            <button
              onClick={() => setPaneTree(panePreset(id))}
              className="rounded border border-line bg-panel px-1.5 text-[10.5px] text-muted hover:text-fg"
              title={t('session.resetLayout')}
            >
              {t('session.resetLayout')}
            </button>
          </>
        )}
      </div>

      {split ? (
        <div className="mt-3 flex min-h-0 flex-1 flex-col">{renderPane(paneTree)}</div>
      ) : (
        <>
          {view === 'diff' && <div className="mt-3 flex min-h-0 flex-1 flex-col">{diffView}</div>}

          {view === 'terminal' && sess && (
            <div className="mt-3 flex min-h-0 flex-1 flex-col">
              <p className="mb-2 text-[11px] text-muted">
                {sess.backendSessionId
                  ? 'Resuming the agent CLI with this session’s backend id.'
                  : 'No backend session yet; the terminal starts the agent CLI without resume.'}
              </p>
              <TerminalPanel session={sess} />
            </div>
          )}

          {view === 'chat' && (
            <div className='mt-3 min-h-0 flex-1 overflow-y-auto pr-1'>{transcript}</div>
          )}

          {view === 'chat' && composer}
        </>
      )}

      <Dialog open={shipOpen} onOpenChange={setShipOpen}>
        <DialogContent title={t('session.shipTitle')} description={gitStatus ? `${gitStatus.branch} · ${gitStatus.changes.length} changed files` : t('session.worktreeStatus')}>
          <div className="flex flex-col gap-3">
            {gitStatus && (
              <div className="max-h-40 overflow-y-auto rounded border border-line bg-bg p-2 font-mono text-[10.5px] text-muted">
                {gitStatus.changes.length === 0 ? t('session.noChanges') : gitStatus.changes.map((change) => <div key={change}>{change}</div>)}
              </div>
            )}
            <Input
              autoFocus
              value={commitMessage}
              onChange={(event) => setCommitMessage(event.target.value)}
              placeholder={sess?.title ? `${t('session.commitMessage')} (${sess.title})` : t('session.commitMessage')}
            />
            {ship.error && <p className="text-[11px] text-err">{String(ship.error.message)}</p>}
            {createPR.error && <p className="text-[11px] text-err">{String(createPR.error.message)}</p>}
            {gitJob && (
              <p className="text-[11px] text-muted">
                {gitJob.action === 'ship' ? 'Commit & push' : 'PR'} queued on {gitJobData?.machineId ?? 'the agent host'} · {gitJobData?.status ?? 'pending'}
              </p>
            )}
            {gitJobData?.error && <p className="text-[11px] text-err">{gitJobData.error}</p>}
            {prUrl && (
              <a href={prUrl} target="_blank" rel="noreferrer" className="truncate text-[11px] underline">
                {prUrl}
              </a>
            )}
            <div className="flex justify-end gap-2">
              <Button variant="secondary" disabled={createPR.isPending || !gitStatus || Boolean(gitJob)} onClick={() => createPR.mutate()}>
                {t('session.createPR')}
              </Button>
              <Button variant="ghost" onClick={() => setShipOpen(false)}>{t('session.cancel')}</Button>
              <Button variant="primary" disabled={ship.isPending || !gitStatus || Boolean(gitJob)} onClick={() => ship.mutate()}>
                <Upload className="h-3 w-3" /> {t('session.commitPush')}
              </Button>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={worktreeOpen} onOpenChange={setWorktreeOpen}>
        <DialogContent title={t('session.worktreeTitle')} description={worktreeInfo?.worktree?.repo || t('session.worktreeLifecycle')}>
          {worktreeInfo?.worktree ? (
            <div className="flex flex-col gap-3">
              <div className="rounded border border-line bg-bg p-2 font-mono text-[10.5px] text-muted">
                <div>{worktreeInfo.worktree.branch}</div>
                <div className="truncate">{worktreeInfo.worktree.path || t('session.pathPending')}</div>
              </div>
              <div className="flex items-center gap-2 text-[11px] text-muted">
                <span className="uppercase">{worktreeInfo.worktree.status}</span>
                {worktreeInfo.cleanup && <span>· cleanup {worktreeInfo.cleanup.status}</span>}
              </div>
              {cleanupWorktree.error && <p className="text-[11px] text-err">{String(cleanupWorktree.error.message)}</p>}
              {archiveWorktree.error && <p className="text-[11px] text-err">{String(archiveWorktree.error.message)}</p>}
              <div className="flex justify-end gap-2">
                <Button
                  variant="secondary"
                  disabled={cleanupWorktree.isPending || worktreeInfo.worktree.status === 'archived'}
                  onClick={() => cleanupWorktree.mutate()}
                >
                  {t('session.cleanupOnly')}
                </Button>
                <Button
                  variant="primary"
                  disabled={archiveWorktree.isPending || worktreeInfo.worktree.status === 'archived'}
                  onClick={() => archiveWorktree.mutate()}
                >
                  {t('session.archiveCleanup')}
                </Button>
              </div>
            </div>
          ) : (
            <p className="text-[12px] text-muted">{t('session.noWorktree')}</p>
          )}
        </DialogContent>
      </Dialog>

      <Dialog open={shareOpen} onOpenChange={setShareOpen}>
        <DialogContent title={t('session.shareTitle')} description={t('session.shareHint')}>
          {share && (
            <div className="flex flex-col gap-3">
              <Input
                readOnly
                value={`${window.location.origin}/shared/${share.token}`}
                onFocus={(event) => event.target.select()}
              />
              <p className="text-[10.5px] text-muted">
                {t('session.expires', { when: share.expiresAt ? new Date(share.expiresAt).toLocaleString() : t('session.never') })}
              </p>
              <div className="flex justify-end gap-2">
                <Button
                  variant="secondary"
                  onClick={() => void navigator.clipboard.writeText(`${window.location.origin}/shared/${share.token}`)}
                >
                  {t('session.copyLink')}
                </Button>
                <Button variant="ghost" disabled={revokeShare.isPending} onClick={() => revokeShare.mutate()}>
                  {t('session.revoke')}
                </Button>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </div>
  )
}
