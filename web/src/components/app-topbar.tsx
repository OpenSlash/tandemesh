import { useEffect, useState } from 'react'
import { ChevronsUpDown, ListTree, Minus, Search, Square, X } from 'lucide-react'
import { Window } from '@wailsio/runtime'
import { useI18n } from '../lib/i18n'
import type { WorkspaceMode } from '../lib/workspace'

export function isDesktopRuntime(): boolean {
  return typeof window !== 'undefined' && Boolean((window as unknown as { wails?: unknown }).wails)
}

function isMacPlatform(): boolean {
  return typeof navigator !== 'undefined' && /Mac/i.test(navigator.userAgent)
}

const drag = { ['--wails-draggable' as string]: 'drag' }
const noDrag = { ['--wails-draggable' as string]: 'no-drag' }

function useDesktopShell() {
  const [desktop, setDesktop] = useState(false)

  useEffect(() => {
    const active = isDesktopRuntime()
    if (active) document.documentElement.dataset.desktop = '1'
    setDesktop(active)
  }, [])

  // macOS draws native traffic lights over the top-left corner of the window;
  // the frameless Windows shell needs caption buttons from the page instead.
  return { desktop, nativeControls: desktop && isMacPlatform() }
}

function Brand({ nativeControls }: { nativeControls: boolean }) {
  const { t } = useI18n()
  return (
    <div className={`flex w-[236px] shrink-0 items-center gap-2 ${nativeControls ? 'pl-[80px] pr-4' : 'px-4'}`}>
      <div className="flex h-5 w-5 items-center justify-center rounded bg-fg text-[11px] font-bold text-bg">T</div>
      <span className="truncate text-[13px] font-semibold tracking-tight">Tandemesh</span>
      <span className="ml-auto rounded border border-line2 px-1 text-[10px] text-muted">{t('shell.dev')}</span>
    </div>
  )
}

function WindowButtons() {
  const { t } = useI18n()
  return (
    <div className="flex h-full shrink-0 items-center">
      <button type="button" aria-label={t('shell.minimise')} className="flex h-full w-11 items-center justify-center text-muted hover:bg-panel2 hover:text-fg" style={noDrag} onClick={() => void Window.Minimise()}>
        <Minus className="h-3.5 w-3.5" />
      </button>
      <button type="button" aria-label={t('shell.maximise')} className="flex h-full w-11 items-center justify-center text-muted hover:bg-panel2 hover:text-fg" style={noDrag} onClick={() => void Window.Maximise()}>
        <Square className="h-3 w-3" />
      </button>
      <button type="button" aria-label={t('shell.close')} className="flex h-full w-11 items-center justify-center text-muted hover:bg-err hover:text-white" style={noDrag} onClick={() => void Window.Close()}>
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  )
}

/**
 * Window title bar for desktop pages rendered outside the main layout. It
 * keeps the window draggable and clear of the platform's window controls.
 */
export function DesktopTitleBar() {
  const { desktop, nativeControls } = useDesktopShell()
  if (!desktop) return null
  return (
    <div className="flex h-11 w-full shrink-0 select-none items-center border-b border-line bg-bg" style={drag}>
      <Brand nativeControls={nativeControls} />
      <div className="flex-1" />
      {!nativeControls && <WindowButtons />}
    </div>
  )
}

export function AppTopbar({
  onSearch,
  workspaces,
  workspaceId,
  workspaceMode,
  onWorkspace,
  onWorkspaceMode,
}: {
  onSearch: () => void
  workspaces: { id: string; name: string }[]
  workspaceId: string
  workspaceMode: WorkspaceMode
  onWorkspace: (id: string) => void
  onWorkspaceMode: (mode: WorkspaceMode) => void
}) {
  const { t } = useI18n()
  const { desktop, nativeControls } = useDesktopShell()
  const mac = isMacPlatform()

  return (
    <div
      className="relative z-50 hidden h-11 w-full shrink-0 select-none items-center border-b border-line bg-bg md:flex"
      style={desktop ? drag : undefined}
    >
      <Brand nativeControls={nativeControls} />

      <div className="flex shrink-0 items-center gap-1">
        <select
          value={workspaceId}
          onChange={(event) => onWorkspace(event.target.value)}
          title={t('shell.workspaces')}
          className="h-7 w-[160px] rounded-md border border-line2 bg-panel px-2 text-[11.5px] text-fg"
        >
          {workspaces.map((workspace) => (
            <option key={workspace.id} value={workspace.id}>{workspace.name}</option>
          ))}
        </select>
        <button
          type="button"
          title={workspaceMode === 'select' ? t('settings.list') : t('settings.dropdown')}
          onClick={() => onWorkspaceMode(workspaceMode === 'select' ? 'list' : 'select')}
          className="flex h-7 w-7 items-center justify-center rounded border border-line2 text-muted hover:text-fg"
        >
          {workspaceMode === 'select' ? <ListTree className="h-3.5 w-3.5" /> : <ChevronsUpDown className="h-3.5 w-3.5" />}
        </button>
      </div>

      <div className="flex min-w-0 flex-1 justify-center px-3">
        <button
          type="button"
          onClick={onSearch}
          className="flex h-7 w-[420px] max-w-[42vw] items-center gap-2 rounded-md border border-line2 bg-panel px-3 text-[11.5px] text-muted hover:text-fg"
        >
          <Search className="h-3.5 w-3.5" />
          <span className="flex-1 text-left">{t('shell.search')}</span>
          <kbd className="rounded border border-line2 px-1 text-[9px]">{mac ? '⌘K' : 'Ctrl K'}</kbd>
        </button>
      </div>

      {desktop && !nativeControls && <WindowButtons />}
    </div>
  )
}
