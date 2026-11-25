import type { SessionMessage } from './api'

export interface DisplayMessage {
  id: string
  role: string
  kind: string
  text: string
  runId: string
}

// Streaming backends append one message per token chunk. Merge consecutive
// chunks from the same run so the transcript renders as one message.
export function groupSessionMessages(messages: SessionMessage[]): DisplayMessage[] {
  const out: DisplayMessage[] = []
  const streamKinds = new Set(['message.chunk', 'thought', 'thinking', 'reasoning', 'delta'])
  for (let index = 0; index < messages.length; index += 1) {
    const message = messages[index]
    const previous = out[out.length - 1]
    if (
      streamKinds.has(message.kind) &&
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
