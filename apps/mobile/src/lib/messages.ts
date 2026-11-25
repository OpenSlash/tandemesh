import type { SessionMessage } from './api'

export interface DisplayMessage {
  id: string
  role: string
  kind: string
  text: string
  runId: string
}

// Chain-of-thought and tool-planning events are internal model output.
// They must never be rendered as user-visible chat messages.
const INTERNAL_KINDS = new Set(['thought', 'thinking', 'reasoning'])
const STREAM_KINDS = new Set(['message.chunk', 'delta'])

// Streaming backends append one message per token chunk. Merge consecutive
// chunks from the same run so the transcript renders as one message.
export function groupSessionMessages(messages: SessionMessage[]): DisplayMessage[] {
  const out: DisplayMessage[] = []
  for (let index = 0; index < messages.length; index += 1) {
    const message = messages[index]
    if (INTERNAL_KINDS.has(message.kind)) continue
    const previous = out[out.length - 1]
    if (
      STREAM_KINDS.has(message.kind) &&
      previous &&
      previous.kind === message.kind &&
      previous.runId === message.runId &&
      previous.role === message.role
    ) {
      previous.text += message.text
      continue
    }
    out.push({
      id: message.id ?? `${message.runId}-${message.ts}-${index}`,
      role: message.role,
      kind: message.kind,
      text: message.text,
      runId: message.runId,
    })
  }
  return out
}
