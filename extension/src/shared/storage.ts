import { emptyFeed, emptyStats, withDefaults } from './defaults'
import type { Account, ActivityEntry, FeedMemory, Settings, Stats } from './types'

export type Store = {
  settings: Settings
  stats: Stats
  activity: ActivityEntry[]
  feed: FeedMemory
  allow: string[]
  account: Account | null
}

type Key = keyof Store

const fallback: { [K in Key]: () => Store[K] } = {
  settings: () => withDefaults(undefined),
  stats: emptyStats,
  activity: () => [],
  feed: emptyFeed,
  allow: () => [],
  account: () => null,
}

export async function read<K extends Key>(key: K): Promise<Store[K]> {
  const got = await chrome.storage.local.get(key)
  const value = got[key] as Store[K] | undefined
  if (key === 'settings') return withDefaults(value as Partial<Settings> | undefined) as Store[K]
  return value ?? fallback[key]()
}

export async function write<K extends Key>(key: K, value: Store[K]): Promise<void> {
  await chrome.storage.local.set({ [key]: value })
}

export async function update<K extends Key>(key: K, change: (current: Store[K]) => Store[K]) {
  const next = change(await read(key))
  await write(key, next)
  return next
}

export function watch<K extends Key>(key: K, onChange: (value: Store[K]) => void): () => void {
  const listener = (changes: Record<string, chrome.storage.StorageChange>, area: string) => {
    if (area !== 'local' || !(key in changes)) return
    const value = changes[key]?.newValue as Store[K] | undefined
    if (key === 'settings') onChange(withDefaults(value as Partial<Settings>) as Store[K])
    else onChange(value ?? fallback[key]())
  }
  chrome.storage.onChanged.addListener(listener)
  return () => chrome.storage.onChanged.removeListener(listener)
}
