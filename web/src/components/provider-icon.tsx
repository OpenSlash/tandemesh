import {
  siGithub,
  siGitlab,
  siJira,
  siLinear,
  siNotion,
  siOutline,
  siPlane,
  type SimpleIcon,
} from 'simple-icons'
import { cn } from '../lib/utils'

const ICONS: Record<string, SimpleIcon> = {
  github: siGithub,
  gitlab: siGitlab,
  jira: siJira,
  linear: siLinear,
  notion: siNotion,
  outline: siOutline,
  plane: siPlane,
}

/**
 * Brand icon for an integration provider. Falls back to a monogram for
 * providers that are not in Simple Icons (Feishu, Slack, Outlook).
 */
export function ProviderIcon({ id, className }: { id: string; className?: string }) {
  const icon = ICONS[id]
  if (!icon) {
    return (
      <span
        aria-hidden
        className={cn(
          'inline-flex h-4 w-4 shrink-0 items-center justify-center rounded bg-panel2 text-[8px] font-semibold uppercase text-muted',
          className,
        )}
      >
        {id.slice(0, 1)}
      </span>
    )
  }
  return (
    <svg
      aria-hidden
      viewBox="0 0 24 24"
      fill="currentColor"
      className={cn('inline-block h-4 w-4 shrink-0', className)}
    >
      <path d={icon.path} />
    </svg>
  )
}
