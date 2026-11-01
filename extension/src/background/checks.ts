import { normalise } from '@/engine/text'
import { paranoidTitles } from '@/engine/rules'
import { read } from '@/shared/storage'
import type { CheckResult, ItemInfo, Result } from '@/shared/types'

import { call } from './api'
import { spend } from './licence'

const BATCH_SIZE = 25
const BATCH_DELAY_MS = 350
const CACHE_LIMIT = 3000
const CACHE_KEY = 'checkCache'

type Cached = CheckResult & { watch: string }
type Waiter = { item: ItemInfo; resolve: (result: CheckResult | null) => void }

let cache: Map<string, Cached> | null = null
const queue: Waiter[] = []
let timer: ReturnType<typeof setTimeout> | null = null
let pausedUntil = 0
let lastError: Result<never> | null = null

async function loadCache(): Promise<Map<string, Cached>> {
  if (cache) return cache
  const stored = (await chrome.storage.local.get(CACHE_KEY))[CACHE_KEY] as
    [string, Cached][] | undefined
  cache = new Map(stored ?? [])
  return cache
}

async function saveCache() {
  if (!cache) return
  const entries = [...cache.entries()].slice(-CACHE_LIMIT)
  cache = new Map(entries)
  await chrome.storage.local.set({ [CACHE_KEY]: entries })
}

async function watchSignature(): Promise<string> {
  const settings = await read('settings')
  return paranoidTitles(settings)
    .map((w) => normalise(`${w.name} ${w.keywords.join(' ')}`))
    .sort()
    .join('|')
}

async function knownTopics(): Promise<string[]> {
  const [feed, settings] = await Promise.all([read('feed'), read('settings')])
  const counts = new Map<string, number>()
  for (const day of Object.values(feed.topicsByDay)) {
    for (const [topic, n] of Object.entries(day)) {
      if (!topic.startsWith('Channel: ')) counts.set(topic, (counts.get(topic) ?? 0) + n)
    }
  }
  const topics = [...counts.entries()].sort((a, b) => b[1] - a[1]).map(([t]) => t)
  return [...new Set([...settings.repeat.blockedTopics, ...topics])].slice(0, 40)
}

async function flush() {
  timer = null
  const batch = queue.splice(0, BATCH_SIZE)
  if (queue.length) timer = setTimeout(() => void flush(), BATCH_DELAY_MS)
  if (!batch.length) return

  const settings = await read('settings')
  const watchlist = paranoidTitles(settings).map(({ name, type, level, keywords }) => ({
    name,
    type,
    level,
    keywords,
  }))
  const result = await call<{ results: CheckResult[] }>('/v1/checks', {
    body: {
      items: batch.map(({ item }) => ({
        id: item.key,
        site: item.site,
        title: item.title.slice(0, 300),
        channel: item.channel.slice(0, 120),
        text: item.text.slice(0, 600),
      })),
      watchlist,
      topics: await knownTopics(),
    },
  })

  if (!result.ok) {
    lastError = result
    pausedUntil = Date.now() + (result.code === 'trial_exhausted' ? 30 * 60_000 : 60_000)
    batch.forEach((w) => w.resolve(null))
    return
  }

  const signature = await watchSignature()
  const store = await loadCache()
  const byId = new Map(result.data.results.map((r) => [r.id, r]))
  for (const waiter of batch) {
    const found = byId.get(waiter.item.key) ?? null
    if (found) store.set(waiter.item.key, { ...found, watch: signature })
    waiter.resolve(found)
  }
  await Promise.all([saveCache(), spend('checks', batch.length)])
}

function enqueue(item: ItemInfo): Promise<CheckResult | null> {
  return new Promise((resolve) => {
    queue.push({ item, resolve })
    if (queue.length >= BATCH_SIZE) {
      if (timer) clearTimeout(timer)
      void flush()
    } else timer ??= setTimeout(() => void flush(), BATCH_DELAY_MS)
  })
}

export async function runChecks(items: ItemInfo[]): Promise<Result<CheckResult[]>> {
  const signature = await watchSignature()
  const store = await loadCache()
  const results: CheckResult[] = []
  const missing: ItemInfo[] = []
  for (const item of items) {
    const hit = store.get(item.key)
    if (hit && hit.watch === signature)
      results.push({ id: hit.id, spoilerOf: hit.spoilerOf, topic: hit.topic })
    else missing.push(item)
  }
  if (missing.length && Date.now() < pausedUntil) {
    return results.length
      ? { ok: true, data: results }
      : (lastError ?? { ok: false, code: 'unavailable', message: '' })
  }
  const fresh = await Promise.all(missing.map(enqueue))
  for (const r of fresh) if (r) results.push(r)
  return { ok: true, data: results }
}
