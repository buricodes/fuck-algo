import { REPEAT_WINDOW } from '@/shared/defaults'
import type {
  BlockKind,
  CheckResult,
  FeedMemory,
  ItemInfo,
  KeywordAction,
  Settings,
} from '@/shared/types'

import { hasPhrase, normalise } from './text'

export type Verdict = { kind: BlockKind; detail: string; action: KeywordAction }

const SPOILER_ACTION: KeywordAction = 'blur'

function itemText(item: ItemInfo): string {
  return normalise(`${item.title} ${item.channel} ${item.text}`)
}

export function localVerdict(item: ItemInfo, settings: Settings, now = new Date()): Verdict | null {
  const text = itemText(item)

  if (settings.features.keywords) {
    for (const rule of settings.keywords) {
      if (rule.expiresAt && new Date(rule.expiresAt) <= now) continue
      if (hasPhrase(text, rule.word))
        return { kind: 'keyword', detail: rule.word, action: rule.action }
    }
  }

  if (settings.features.spoiler) {
    const title = normalise(item.title)
    for (const watch of settings.watchlist) {
      const haystack = watch.level === 0 ? title : text
      const terms = watch.level === 0 ? [watch.name] : [watch.name, ...watch.keywords]
      if (terms.some((term) => hasPhrase(haystack, term))) {
        return { kind: 'spoiler', detail: watch.name, action: SPOILER_ACTION }
      }
    }
  }

  return null
}

export function paranoidTitles(settings: Settings) {
  return settings.features.spoiler ? settings.watchlist.filter((w) => w.level === 2) : []
}

export function wantsCheck(settings: Settings): { spoiler: boolean; topic: boolean } {
  return { spoiler: paranoidTitles(settings).length > 0, topic: settings.features.repeat }
}

export function spoilerVerdict(result: CheckResult, settings: Settings): Verdict | null {
  if (!result.spoilerOf) return null
  const match = paranoidTitles(settings).find(
    (w) => normalise(w.name) === normalise(result.spoilerOf ?? ''),
  )
  return match ? { kind: 'spoiler', detail: match.name, action: SPOILER_ACTION } : null
}

export function countTopic(feed: FeedMemory, topic: string): number {
  const t = normalise(topic)
  return feed.recent.slice(-REPEAT_WINDOW).filter((seen) => normalise(seen.topic) === t).length
}

export function repeatVerdict(topic: string, settings: Settings, feed: FeedMemory): Verdict | null {
  if (!settings.features.repeat || !topic) return null
  const t = normalise(topic)
  if (settings.repeat.blockedTopics.some((blocked) => normalise(blocked) === t)) {
    return { kind: 'repeat', detail: topic, action: 'skip' }
  }
  const seen = countTopic(feed, topic)
  if (seen >= settings.repeat.limit) {
    return { kind: 'repeat', detail: `${topic} ×${seen + 1}`, action: 'skip' }
  }
  return null
}

export function fallbackTopic(item: ItemInfo): string {
  return item.channel ? `Channel: ${item.channel}` : ''
}
