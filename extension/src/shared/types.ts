export type Site = 'youtube' | 'x' | 'instagram' | 'web'
export type SpoilerType = 'anime' | 'game' | 'show' | 'movie' | 'sports'
export type Strictness = 0 | 1 | 2
export type KeywordAction = 'hide' | 'blur' | 'skip'
export type BlockKind = 'spoiler' | 'repeat' | 'keyword'
export type Theme = 'dark' | 'light'

export type WatchItem = {
  id: string
  name: string
  type: SpoilerType
  keywords: string[]
  level: Strictness
  createdAt: string
}

export type KeywordRule = {
  id: string
  word: string
  action: KeywordAction
  expiresAt: string | null
}

export type Settings = {
  shield: boolean
  shieldSince: string
  pausedHosts: string[]
  features: { autoScroll: boolean; spoiler: boolean; repeat: boolean; keywords: boolean }
  sites: Record<Site, boolean>
  watchlist: WatchItem[]
  repeat: { limit: number; blockedTopics: string[] }
  keywords: KeywordRule[]
  theme: Theme
}

export type ActivityEntry = {
  id: string
  at: string
  key: string
  site: Site
  host: string
  title: string
  kind: BlockKind
  detail: string
  action: KeywordAction
  undone: boolean
}

export type DayStats = { spoiler: number; repeat: number; keyword: number }
export type Stats = { days: Record<string, DayStats> }

export type FeedMemory = {
  recent: { key: string; topic: string; at: string }[]
  topicsByDay: Record<string, Record<string, number>>
}

export type Install = { id: string; token: string }

export type Licence = {
  status: 'trial' | 'active'
  keyHint: string | null
  trial: { voiceLeft: number; checksLeft: number }
}

export type ItemInfo = {
  key: string
  site: Site
  host: string
  title: string
  channel: string
  text: string
}

export type CheckResult = { id: string; spoilerOf: string | null; topic: string | null }

export type ProposedRule =
  | { kind: 'spoiler'; name: string; type: SpoilerType; level: Strictness; keywords: string[] }
  | { kind: 'repeat'; topic: string; sites: Site[] }
  | { kind: 'keyword'; word: string; action: KeywordAction; expiresAt: string | null }

export type ApiErrorCode =
  | 'bad_request'
  | 'unauthorised'
  | 'not_found'
  | 'trial_exhausted'
  | 'licence_invalid'
  | 'licence_full'
  | 'rate_limited'
  | 'upstream'
  | 'unavailable'
  | 'internal'
  | 'network'
  | 'mic_needs_grant'
  | 'mic_blocked'

export type Result<T> = { ok: true; data: T } | { ok: false; code: ApiErrorCode; message: string }
