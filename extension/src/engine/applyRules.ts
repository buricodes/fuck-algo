import type { ProposedRule, Settings } from '@/shared/types'
import { newId } from '@/shared/util'

import { normalise } from './text'

const same = (a: string, b: string) => normalise(a) === normalise(b)

function mergeWords(current: string[], extra: string[]): string[] {
  const out = [...current]
  for (const word of extra) if (!out.some((w) => same(w, word))) out.push(word)
  return out
}

export function applyRules(settings: Settings, rules: ProposedRule[], now = new Date()): Settings {
  let next = { ...settings }
  for (const rule of rules) {
    if (rule.kind === 'spoiler') {
      const existing = next.watchlist.find((w) => same(w.name, rule.name))
      next = {
        ...next,
        features: { ...next.features, spoiler: true },
        watchlist: existing
          ? next.watchlist.map((w) =>
              w === existing
                ? {
                    ...w,
                    type: rule.type,
                    level: rule.level,
                    keywords: mergeWords(w.keywords, rule.keywords),
                  }
                : w,
            )
          : [
              {
                id: newId(),
                name: rule.name,
                type: rule.type,
                level: rule.level,
                keywords: mergeWords([], rule.keywords),
                createdAt: now.toISOString(),
              },
              ...next.watchlist,
            ],
      }
    } else if (rule.kind === 'repeat') {
      const sites = { ...next.sites }
      for (const site of rule.sites) sites[site] = true
      next = {
        ...next,
        sites,
        features: { ...next.features, repeat: true },
        repeat: {
          ...next.repeat,
          blockedTopics: mergeWords(next.repeat.blockedTopics, [rule.topic]),
        },
      }
    } else {
      const existing = next.keywords.find((k) => same(k.word, rule.word))
      next = {
        ...next,
        features: { ...next.features, keywords: true },
        keywords: existing
          ? next.keywords.map((k) =>
              k === existing ? { ...k, action: rule.action, expiresAt: rule.expiresAt } : k,
            )
          : [
              ...next.keywords,
              { id: newId(), word: rule.word, action: rule.action, expiresAt: rule.expiresAt },
            ],
      }
    }
  }
  return next
}
