import { useParams } from 'react-router-dom'
import { useQuery } from '@tanstack/react-query'
import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { Loader2, Lock } from 'lucide-react'
import { api } from '../lib/api'

export default function SharedSessionPage() {
  const { token = '' } = useParams()
  const { data, isLoading, error } = useQuery({
    queryKey: ['shared-session', token],
    queryFn: () => api.sharedSession(token),
    retry: false,
  })
  const messages = data?.messages ?? []

  return (
    <div className="mx-auto flex min-h-screen max-w-[860px] flex-col gap-4 px-4 py-6">
      <header className="border-b border-line pb-3">
        <div className="flex items-center gap-2 text-[11px] text-muted">
          <Lock className="h-3 w-3" /> Read-only shared session
        </div>
        <h1 className="mt-1 text-[18px] font-semibold tracking-tight">{data?.session.title ?? 'Shared session'}</h1>
        {data?.session.repo && <p className="mt-0.5 text-[11px] text-muted">{data.session.repo}</p>}
      </header>

      {isLoading && (
        <div className="flex items-center gap-2 text-[12px] text-muted">
          <Loader2 className="h-3.5 w-3.5 animate-spin" /> Loading shared transcript...
        </div>
      )}
      {error && (
        <div className="rounded-md border border-err/30 bg-err/5 p-3 text-[12px] text-err">
          {error instanceof Error ? error.message : 'This share link is invalid or expired.'}
        </div>
      )}

      <div className="flex flex-col gap-4">
        {messages.map((message, index) => {
          if (message.kind === 'tool' || message.kind === 'result') {
            return (
              <div key={message.id ?? index} className="ml-8 rounded border border-line bg-panel/30 px-2 py-1 font-mono text-[10.5px] text-muted">
                {message.text}
              </div>
            )
          }
          if (message.role === 'user') {
            return (
              <div key={message.id ?? index} className="flex justify-end">
                <div className="max-w-[78%] whitespace-pre-wrap rounded-2xl bg-panel2 px-4 py-2.5 text-[12.5px] text-fg">
                  {message.text}
                </div>
              </div>
            )
          }
          return (
            <div key={message.id ?? index} className="text-[12.5px] leading-relaxed text-fg [&_code]:rounded [&_code]:bg-panel2 [&_code]:px-1 [&_code]:py-0.5 [&_pre]:overflow-x-auto [&_pre]:rounded [&_pre]:border [&_pre]:border-line [&_pre]:bg-bg [&_pre]:p-2">
              <ReactMarkdown remarkPlugins={[remarkGfm]}>{message.text}</ReactMarkdown>
            </div>
          )
        })}
      </div>
    </div>
  )
}
