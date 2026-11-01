import type { FeedMemory, Settings, Stats } from './types'

export const REPEAT_WINDOW = 20
export const ACTIVITY_CAP = 300
export const ALLOW_CAP = 500
export const STATS_DAYS = 30
export const TOPIC_DAYS = 7

export function defaultSettings(now = new Date()): Settings {
  return {
    shield: true,
    shieldSince: now.toISOString(),
    pausedHosts: [],
    features: { autoScroll: true, spoiler: true, repeat: true, keywords: true },
    sites: { youtube: true, x: true, instagram: true, web: false },
    watchlist: [],
    repeat: { limit: 3, blockedTopics: [] },
    keywords: [],
    theme: 'dark',
  }
}

export const emptyStats = (): Stats => ({ days: {} })
export const emptyFeed = (): FeedMemory => ({ recent: [], topicsByDay: {} })

export function withDefaults(stored: Partial<Settings> | undefined): Settings {
  const base = defaultSettings()
  if (!stored) return base
  return {
    ...base,
    ...stored,
    features: { ...base.features, ...stored.features },
    sites: { ...base.sites, ...stored.sites },
    repeat: { ...base.repeat, ...stored.repeat },
  }
}
