import { clsx, type ClassValue } from 'clsx'
import { twMerge } from 'tailwind-merge'

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function shortId(id: string, n = 8) {
  return id.length <= n ? id : id.slice(-n)
}

// refId strips the object prefix and keeps the first n characters of the ULID,
// which are time-ordered and stable, e.g. run_01JBQ4M8Z3 -> 01JBQ4M8.
export function refId(id: string, n = 8) {
  const value = id.replace(/^(run|mch|apr|agt|bnd)_/, '')
  return value.slice(0, n)
}

// compactId keeps the prefix but shortens the ULID, e.g. run_01JBQ4M8.
export function compactId(id: string, n = 8) {
  const match = id.match(/^(run|mch|apr|agt|bnd)_(.+)$/)
  return match ? `${match[1]}_${match[2].slice(0, n)}` : id.slice(0, n)
}

export function age(iso: string) {
  const then = new Date(iso).getTime()
  if (!then) return ''
  const s = Math.max(0, Math.floor((Date.now() - then) / 1000))
  if (s < 60) return `${s}s`
  const m = Math.floor(s / 60)
  if (m < 60) return `${m}m`
  const h = Math.floor(m / 60)
  if (h < 24) return `${h}h`
  return `${Math.floor(h / 24)}d`
}
