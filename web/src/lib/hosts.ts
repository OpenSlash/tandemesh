import { isDesktopRuntime } from '../components/app-topbar'
import type { Machine } from './api'

/**
 * Reports whether a host is the computer running this desktop app. The
 * desktop starts its own daemon with the "desktop" label; in a plain browser
 * no host is local.
 */
export function isLocalHost(machine: Pick<Machine, 'labels'>): boolean {
  return isDesktopRuntime() && (machine.labels ?? []).includes('desktop')
}
