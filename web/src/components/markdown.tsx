import ReactMarkdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { cn } from '../lib/utils'

export function Markdown({ children, className }: { children: string; className?: string }) {
  return (
    <div className={cn('min-w-0 text-[12.5px] leading-relaxed text-fg/90', className)}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm]}
        components={{
          p: ({ children }) => <p className="my-1 whitespace-pre-wrap">{children}</p>,
          a: ({ href, children }) => (
            <a href={href} target="_blank" rel="noreferrer" className="text-accent2 underline">
              {children}
            </a>
          ),
          ul: ({ children }) => <ul className="my-1 list-disc pl-5">{children}</ul>,
          ol: ({ children }) => <ol className="my-1 list-decimal pl-5">{children}</ol>,
          li: ({ children }) => <li className="my-0.5">{children}</li>,
          h1: ({ children }) => <h1 className="mb-1 mt-2 text-[15px] font-semibold">{children}</h1>,
          h2: ({ children }) => <h2 className="mb-1 mt-2 text-[14px] font-semibold">{children}</h2>,
          h3: ({ children }) => <h3 className="mb-1 mt-2 text-[13px] font-semibold">{children}</h3>,
          blockquote: ({ children }) => (
            <blockquote className="my-1 border-l-2 border-line2 pl-3 text-muted">{children}</blockquote>
          ),
          pre: ({ children }) => (
            <pre className="my-1 overflow-x-auto rounded-md bg-panel2 p-2 font-mono text-[11px]">{children}</pre>
          ),
          code: ({ className: codeClass, children }) => (
            <code className={cn('rounded bg-panel2 px-1 py-0.5 font-mono text-[11px]', codeClass)}>{children}</code>
          ),
          table: ({ children }) => (
            <div className="my-2 overflow-x-auto">
              <table className="w-full border-collapse text-[11.5px]">{children}</table>
            </div>
          ),
          th: ({ children }) => <th className="border border-line px-2 py-1 text-left font-medium">{children}</th>,
          td: ({ children }) => <td className="border border-line px-2 py-1">{children}</td>,
          hr: () => <hr className="my-2 border-line" />,
        }}
      >
        {children}
      </ReactMarkdown>
    </div>
  )
}
