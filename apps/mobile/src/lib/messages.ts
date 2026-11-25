import type { SessionMessage } from './api'

export interface DisplayMessage {
  id: string
  role: string
  kind: string
  text: string
  runId: string
}

// Chain-of-thought is rendered separately and collapsed by default.
const REASONING_KINDS = new Set(['thought', 'thinking', 'reasoning'])
const STREAM_KINDS = new Set(['message.chunk', 'delta', 'reasoning'])

// Streaming backends append one event per token chunk. Merge consecutive
// chunks from the same run so each logical message renders as one block.
export function groupSessionMessages(messages: SessionMessage[]): DisplayMessage[] {
  const out: DisplayMessage[] = []
  for (let index = 0; index < messages.length; index += 1) {
    const message = messages[index]
    const kind = REASONING_KINDS.has(message.kind) ? 'reasoning' : message.kind
    const previous = out[out.length - 1]
    if (
      STREAM_KINDS.has(kind) &&
      previous &&
      previous.kind === kind &&
      previous.runId === message.runId &&
      previous.role === message.role
    ) {
      previous.text += message.text
      continue
    }
    out.push({
      id: message.id ?? `${message.runId}-${message.ts}-${index}`,
      role: message.role,
      kind,
      text: message.text,
      runId: message.runId,
    })
  }
  // Some ACP backends replay the prior transcript on session resume, and the
  // replay is recorded as output of the new run. Drop assistant blocks whose
  // text already appeared earlier in this session.
  const seen = new Set<string>()
  const deduped: DisplayMessage[] = []
  for (const message of out) {
    if (message.role === 'assistant') {
      const text = message.text.trim()
      const key = `${message.kind}|${text}`
      if (text && seen.has(key)) continue
      if (text) seen.add(key)
    }
    deduped.push(message)
  }
  return deduped
}
