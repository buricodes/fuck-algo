import type { Site } from './types'

export function siteOf(host: string): Site {
  const h = host.replace(/^www\./, '').replace(/^m\./, '')
  if (h === 'youtube.com' || h.endsWith('.youtube.com')) return 'youtube'
  if (h === 'x.com' || h === 'twitter.com') return 'x'
  if (h === 'instagram.com') return 'instagram'
  return 'web'
}

export const SITE_NAMES: Record<Site, string> = {
  youtube: 'YouTube',
  x: 'X',
  instagram: 'Instagram',
  web: 'Other sites',
}

export function dayKey(date = new Date()): string {
  const y = date.getFullYear()
  const m = String(date.getMonth() + 1).padStart(2, '0')
  const d = String(date.getDate()).padStart(2, '0')
  return `${y}-${m}-${d}`
}

export function clock(iso: string): string {
  return new Date(iso).toLocaleTimeString('en-GB', { hour: '2-digit', minute: '2-digit' })
}

export function newId(): string {
  return crypto.randomUUID()
}

export function plural(n: number, one: string, many = `${one}s`): string {
  return `${n} ${n === 1 ? one : many}`
}

export function splitList(value: string): string[] {
  const seen = new Set<string>()
  return value
    .split(',')
    .map((part) => part.trim())
    .filter((part) => {
      const key = part.toLowerCase()
      if (!part || seen.has(key)) return false
      seen.add(key)
      return true
    })
}
