import { cn } from '../lib/utils'

/**
 * An agent's icon from the ACP Registry, served through the plane. Registry
 * icons are single-colour SVGs, so they are drawn as a mask over the current
 * text colour and follow the theme. Agents outside the registry get an
 * initial instead.
 */
export function AgentIcon({ registryId, name, className }: { registryId?: string; name: string; className?: string }) {
  if (!registryId) {
    return (
      <span
        aria-hidden
        className={cn(
          'inline-flex h-4 w-4 shrink-0 items-center justify-center rounded bg-panel2 text-[9px] font-semibold uppercase text-muted',
          className,
        )}
      >
        {name.slice(0, 1)}
      </span>
    )
  }
  const iconUrl = `/api/v1/acp/icons/${encodeURIComponent(registryId)}`
  return (
    <span
      aria-hidden
      className={cn('inline-block h-4 w-4 shrink-0 bg-current', className)}
      style={{
        maskImage: `url(${iconUrl})`,
        WebkitMaskImage: `url(${iconUrl})`,
        maskRepeat: 'no-repeat',
        WebkitMaskRepeat: 'no-repeat',
        maskPosition: 'center',
        WebkitMaskPosition: 'center',
        maskSize: 'contain',
        WebkitMaskSize: 'contain',
      }}
    />
  )
}

/**
 * Maps backend ids to the registry agent whose icon represents them: catalog
 * backends via their registry alias, registry-only backends via the hosts
 * that installed them.
 */
export function registryIdsByBackend(
  catalog: { id: string; registryId?: string }[],
  machines: { runtimeDetails?: { id: string; registryId?: string }[] }[],
): Map<string, string> {
  const out = new Map<string, string>()
  for (const machine of machines) {
    for (const status of machine.runtimeDetails ?? []) {
      if (status.registryId) out.set(status.id, status.registryId)
    }
  }
  for (const runtime of catalog) {
    if (runtime.registryId) out.set(runtime.id, runtime.registryId)
  }
  return out
}
