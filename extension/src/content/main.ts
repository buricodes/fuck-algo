import {
  fallbackTopic,
  localVerdict,
  repeatVerdict,
  spoilerVerdict,
  wantsCheck,
} from '@/engine/rules'
import type { Verdict } from '@/engine/rules'
import { emptyFeed, withDefaults } from '@/shared/defaults'
import { request } from '@/shared/messages'
import { read, update, watch } from '@/shared/storage'
import type { CheckResult, FeedMemory, ItemInfo, Settings } from '@/shared/types'
import { siteOf } from '@/shared/util'

import { clearAll, clearState, installPageStyles, setState, toast } from './overlay'
import { instagram } from './sites/instagram'
import type { Adapter, Found, Player } from './sites/types'
import { web } from './sites/web'
import { x } from './sites/x'
import { youtube } from './sites/youtube'

const ADAPTERS: Record<ReturnType<typeof siteOf>, Adapter> = { youtube, x, instagram, web }
const host = location.hostname.replace(/^www\./, '')
const adapter = ADAPTERS[siteOf(host)]

type Tracked = {
  key: string
  info: ItemInfo
  layout: Found['layout']
  verdict: Verdict | null
  topic: string | null
}

let settings: Settings = withDefaults(undefined)
let feed: FeedMemory = emptyFeed()
let allow = new Set<string>()
let generation = 0
const tracked = new WeakMap<HTMLElement, Tracked>()
const trackedEls = new Set<HTMLElement>()
const checks = new Map<string, Promise<CheckResult | null>>()
const seenKeys = new Set<string>()

function active(): boolean {
  return settings.shield && settings.sites[adapter.site] && !settings.pausedHosts.includes(host)
}

function record(item: ItemInfo, verdict: Verdict) {
  void request('activity:record', {
    entry: {
      key: item.key,
      site: item.site,
      host: host + (location.pathname.startsWith('/shorts') ? '/shorts' : ''),
      title: item.title,
      kind: verdict.kind,
      detail: verdict.detail,
      action: verdict.action,
    },
  })
}

const pendingBatch: ItemInfo[] = []
let batchTimer: ReturnType<typeof setTimeout> | null = null
const resolvers = new Map<string, (r: CheckResult | null) => void>()

function check(item: ItemInfo): Promise<CheckResult | null> {
  const known = checks.get(item.key)
  if (known) return known
  const promise = new Promise<CheckResult | null>((resolve) => {
    resolvers.set(item.key, resolve)
    pendingBatch.push(item)
    batchTimer ??= setTimeout(() => void sendBatch(), 150)
  })
  checks.set(item.key, promise)
  return promise
}

async function sendBatch() {
  batchTimer = null
  const items = pendingBatch.splice(0)
  let results: CheckResult[] = []
  try {
    const reply = await request('checks:run', { items })
    if (reply.ok) results = reply.data
  } catch {
    results = []
  }
  const byId = new Map(results.map((r) => [r.id, r]))
  for (const item of items) {
    const result = byId.get(item.key) ?? null
    if (!result) checks.delete(item.key)
    resolvers.get(item.key)?.(result)
    resolvers.delete(item.key)
  }
}

const visibility = new IntersectionObserver(
  (entries) => {
    for (const entry of entries) {
      if (!entry.isIntersecting) continue
      const el = entry.target as HTMLElement
      const t = tracked.get(el)
      visibility.unobserve(el)
      if (!t || t.verdict || !t.topic || seenKeys.has(t.key)) continue
      seenKeys.add(t.key)
      void request('feed:seen', { key: t.key, topic: t.topic })
    }
  },
  { threshold: 0.6 },
)

function paint(el: HTMLElement, t: Tracked) {
  if (!t.verdict) {
    clearState(el)
    return
  }
  if (allow.has(t.key)) {
    setState(el, 'shown', { layout: t.layout, actions: actionsFor(el, t) })
    return
  }
  const state = t.verdict.action === 'blur' ? 'blurred' : 'hidden'
  setState(el, state, {
    kind: t.verdict.kind,
    detail: t.verdict.detail,
    layout: t.layout,
    actions: actionsFor(el, t),
  })
}

function actionsFor(el: HTMLElement, t: Tracked) {
  return {
    show: () => {
      allow.add(t.key)
      void update('allow', (keys) => [...new Set([...keys, t.key])].slice(-500))
      paint(el, t)
    },
    dismiss: () => {
      setState(el, 'dismissed', { layout: t.layout, actions: actionsFor(el, t) })
      const topic = t.topic ?? fallbackTopic(t.info)
      if (topic) void request('feed:seen', { key: `${t.key}:not-interested`, topic })
    },
    rehide: () => {
      allow.delete(t.key)
      void update('allow', (keys) => keys.filter((k) => k !== t.key))
      paint(el, t)
    },
  }
}

function decide(t: Tracked, verdict: Verdict | null, el: HTMLElement) {
  t.verdict = verdict
  paint(el, t)
  if (verdict) record(t.info, verdict)
  else visibility.observe(el)
}

async function process(found: Found, gen: number) {
  const { el, info, layout } = found
  const t: Tracked = { key: info.key, info, layout, verdict: null, topic: null }
  tracked.set(el, t)
  trackedEls.add(el)

  const local = localVerdict(info, settings)
  if (local) return decide(t, local, el)

  const wants = wantsCheck(settings)
  if (!wants.spoiler && !wants.topic) return decide(t, null, el)

  if (wants.spoiler) setState(el, 'pending', { layout })
  const result = await check(info)
  if (gen !== generation || tracked.get(el) !== t) return

  t.topic = result?.topic ?? fallbackTopic(info)
  const verdict =
    (result && spoilerVerdict(result, settings)) ??
    (t.topic ? repeatVerdict(t.topic, settings, feed) : null)
  decide(t, verdict, el)
}

function scan() {
  if (!active()) return
  const gen = generation
  for (const found of adapter.scan(host)) {
    const existing = tracked.get(found.el)
    if (existing?.key === found.info.key) continue
    if (existing) clearState(found.el)
    void process(found, gen)
  }
  for (const el of trackedEls) if (!el.isConnected) trackedEls.delete(el)
}

function reset() {
  generation++
  checks.clear()
  for (const el of trackedEls) tracked.delete(el)
  trackedEls.clear()
  clearAll()
  scan()
  playerKey = null
}

let playerKey: string | null = null
let watchCleanup: (() => void) | null = null

function trackWatches(player: Player) {
  watchCleanup?.()
  const video = player.video
  if (!video || !settings.features.autoScroll || adapter.site !== 'youtube') return
  let furthest = 0
  let done = false
  const finished = () => {
    if (done || playerKey !== player.key) return
    done = true
    setTimeout(() => playerKey === player.key && player.next(), 900)
  }
  const onTime = () => {
    if (video.currentTime > furthest) furthest = video.currentTime
    else if (video.duration && furthest >= video.duration - 1.5 && video.currentTime < furthest - 1)
      finished()
  }
  video.addEventListener('timeupdate', onTime)
  video.addEventListener('ended', finished)
  watchCleanup = () => {
    video.removeEventListener('timeupdate', onTime)
    video.removeEventListener('ended', finished)
  }
}

async function handlePlayer() {
  if (!active() || !adapter.player) return
  const player = adapter.player(host)
  if (!player || player.key === playerKey || !player.info.title) return
  playerKey = player.key
  const gen = generation

  let verdict = localVerdict(player.info, settings)
  let topic: string | null = null
  if (!verdict && !allow.has(player.key)) {
    const wants = wantsCheck(settings)
    if (wants.spoiler || wants.topic) {
      const result = await check(player.info)
      if (gen !== generation || playerKey !== player.key) return
      topic = result?.topic ?? fallbackTopic(player.info)
      verdict =
        (result && spoilerVerdict(result, settings)) ??
        (topic ? repeatVerdict(topic, settings, feed) : null)
    } else topic = fallbackTopic(player.info)
  }

  if (verdict && !allow.has(player.key)) {
    const skip = { ...verdict, action: 'skip' as const }
    record(player.info, skip)
    toast(skip.kind, skip.detail)
    player.next()
    return
  }
  if (topic && !seenKeys.has(player.key)) {
    seenKeys.add(player.key)
    void request('feed:seen', { key: player.key, topic })
  }
  trackWatches(player)
}

let scheduled = false
function schedule() {
  if (scheduled || document.hidden) return
  scheduled = true
  setTimeout(() => {
    scheduled = false
    scan()
    void handlePlayer()
  }, 250)
}

async function start() {
  ;[settings, feed] = await Promise.all([read('settings'), read('feed')])
  allow = new Set(await read('allow'))
  installPageStyles()

  watch('settings', (next) => {
    const wasActive = active()
    settings = next
    if (!active()) {
      clearAll()
      watchCleanup?.()
      if (wasActive) generation++
      return
    }
    reset()
  })
  watch('feed', (next) => {
    feed = next
  })
  watch('allow', (next) => {
    allow = new Set(next)
    for (const el of trackedEls) {
      const t = tracked.get(el)
      if (t?.verdict) paint(el, t)
    }
  })

  new MutationObserver(schedule).observe(document.documentElement, {
    childList: true,
    subtree: true,
  })
  document.addEventListener('visibilitychange', schedule)
  window.addEventListener('yt-navigate-finish', schedule)
  setInterval(schedule, 1500)
  schedule()
}

void start().catch(() => {
  clearAll()
})
