import { describe, expect, it } from 'vitest'

import { defaultSettings, emptyFeed } from '@/shared/defaults'
import type { ItemInfo, Settings } from '@/shared/types'

import { applyRules } from './applyRules'
import { localVerdict, repeatVerdict, spoilerVerdict, wantsCheck } from './rules'
import { hasPhrase, normalise } from './text'

const item = (title: string, extra: Partial<ItemInfo> = {}): ItemInfo => ({
  key: 'yt:1',
  site: 'youtube',
  host: 'youtube.com',
  title,
  channel: '',
  text: '',
  ...extra,
})

const settings = (change: Partial<Settings> = {}): Settings => ({ ...defaultSettings(), ...change })

describe('text matching', () => {
  it('matches whole words only', () => {
    expect(hasPhrase(normalise('Cash out now'), 'ash')).toBe(false)
    expect(hasPhrase(normalise('Ash wins again'), 'ash')).toBe(true)
  })

  it('ignores case, accents and punctuation', () => {
    expect(hasPhrase(normalise('GTA-6: the LEAK!'), 'gta 6')).toBe(true)
    expect(hasPhrase(normalise('Pokémon reveal'), 'pokemon')).toBe(true)
  })

  it('matches multi word phrases written as one hashtag', () => {
    expect(hasPhrase(normalise('wow #BlackClover371'), 'Black Clover')).toBe(true)
  })
})

describe('local verdicts', () => {
  const watch = (level: 0 | 1 | 2) =>
    settings({
      watchlist: [
        { id: 'w', name: 'Black Clover', type: 'anime', keywords: ['Asta'], level, createdAt: '' },
      ],
    })

  it('Relaxed only matches the exact title in the item title', () => {
    expect(localVerdict(item('Asta finally does it'), watch(0))).toBeNull()
    expect(localVerdict(item('Black Clover ch. 371'), watch(0))?.kind).toBe('spoiler')
  })

  it('Balanced also matches extra keywords anywhere in the item', () => {
    expect(localVerdict(item('Asta finally does it'), watch(1))).toEqual({
      kind: 'spoiler',
      detail: 'Black Clover',
      action: 'blur',
    })
  })

  it('keywords win and carry their action', () => {
    const s = settings({ keywords: [{ id: 'k', word: 'crypto', action: 'hide', expiresAt: null }] })
    expect(localVerdict(item('5 crypto coins'), s)).toEqual({
      kind: 'keyword',
      detail: 'crypto',
      action: 'hide',
    })
  })

  it('expired keywords stop matching', () => {
    const s = settings({
      keywords: [{ id: 'k', word: 'election', action: 'blur', expiresAt: '2026-01-01T00:00:00Z' }],
    })
    expect(localVerdict(item('Election night'), s, new Date('2026-02-01'))).toBeNull()
  })

  it('switched off features do nothing', () => {
    const s = watch(1)
    s.features.spoiler = false
    expect(localVerdict(item('Black Clover'), s)).toBeNull()
  })

  it('only asks for spoiler checks when a Paranoid title exists', () => {
    expect(wantsCheck(watch(1)).spoiler).toBe(false)
    expect(wantsCheck(watch(2)).spoiler).toBe(true)
    expect(
      spoilerVerdict({ id: 'x', spoilerOf: 'black clover', topic: null }, watch(2))?.detail,
    ).toBe('Black Clover')
  })
})

describe('repeat blocker', () => {
  const feed = (topics: string[]) => ({
    ...emptyFeed(),
    recent: topics.map((topic, i) => ({ key: `k${i}`, topic, at: '' })),
  })

  it('allows a topic until the limit is reached', () => {
    const s = settings({ repeat: { limit: 3, blockedTopics: [] } })
    expect(repeatVerdict('Gym edits', s, feed(['Gym edits', 'Gym edits']))).toBeNull()
    expect(repeatVerdict('Gym edits', s, feed(['Gym edits', 'gym edits', 'Gym Edits']))).toEqual({
      kind: 'repeat',
      detail: 'Gym edits ×4',
      action: 'skip',
    })
  })

  it('always skips blocked topics', () => {
    const s = settings({ repeat: { limit: 10, blockedTopics: ['Sigma edits'] } })
    expect(repeatVerdict('sigma edits', s, feed([]))?.action).toBe('skip')
  })
})

describe('applying voice rules', () => {
  it('adds new rules and merges into existing ones', () => {
    const start = settings({
      watchlist: [
        { id: 'w', name: 'GTA 6', type: 'game', keywords: ['leak'], level: 1, createdAt: '' },
      ],
    })
    const next = applyRules(start, [
      { kind: 'spoiler', name: 'gta 6', type: 'game', level: 2, keywords: ['Leak', 'trailer'] },
      { kind: 'repeat', topic: 'Gym bro advice', sites: ['instagram'] },
      { kind: 'keyword', word: 'crypto', action: 'blur', expiresAt: null },
    ])
    expect(next.watchlist).toHaveLength(1)
    expect(next.watchlist[0]?.level).toBe(2)
    expect(next.watchlist[0]?.keywords).toEqual(['leak', 'trailer'])
    expect(next.repeat.blockedTopics).toEqual(['Gym bro advice'])
    expect(next.keywords.map((k) => k.word)).toEqual(['crypto'])
  })
})
