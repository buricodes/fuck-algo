import { normalise } from '@/engine/text'
import type { Site } from '@/shared/types'
import { Icon, Range, Toggle } from '@/ui/components'
import { useStore } from '@/ui/hooks/useStore'

import type { PageProps } from '../Settings'

const SITES: [Site, string, string][] = [
  ['youtube', 'YouTube', 'Shorts, Home, Up next'],
  ['x', 'X', 'For you, Explore'],
  ['instagram', 'Instagram', 'Reels, Explore'],
  ['web', 'Other sites', 'Reddit, news, any feed'],
]

export function Repeat({ settings, set }: PageProps) {
  const [feed] = useStore('feed')
  const limit = settings.repeat.limit
  const blocked = settings.repeat.blockedTopics

  const counts = new Map<string, number>()
  for (const day of Object.values(feed?.topicsByDay ?? {})) {
    for (const [topic, n] of Object.entries(day)) counts.set(topic, (counts.get(topic) ?? 0) + n)
  }
  for (const topic of blocked) if (!counts.has(topic)) counts.set(topic, 0)
  const topics = [...counts.entries()].sort((a, b) => b[1] - a[1]).slice(0, 10)
  const isBlocked = (topic: string) => blocked.some((b) => normalise(b) === normalise(topic))

  const toggleTopic = (topic: string) =>
    set((s) => ({
      ...s,
      repeat: {
        ...s.repeat,
        blockedTopics: isBlocked(topic)
          ? s.repeat.blockedTopics.filter((b) => normalise(b) !== normalise(topic))
          : [...s.repeat.blockedTopics, topic],
      },
    }))

  return (
    <>
      <header className="set-header" style={{ maxWidth: 780 }}>
        <div className="eyebrow">02 · Repeat blocker</div>
        <h1 className="display set-h1-sm">Seen too much of the same thing? We skip it.</h1>
      </header>

      <div className="set-limit">
        <div className="set-limit-value">
          <span>max</span>
          <span className="display">{limit}</span>
          <span>similar videos in your last 20</span>
        </div>
        <div className="set-limit-slider">
          <Range
            value={limit}
            min={1}
            max={10}
            label="Repeat limit"
            onChange={(n) => set((s) => ({ ...s, repeat: { ...s.repeat, limit: n } }))}
          />
          <div className="set-limit-ends">
            <span>1 · Ruthless</span>
            <span>10 · Chill</span>
          </div>
        </div>
      </div>

      <div className="set-list">
        <div className="set-fed-head">
          <span className="set-h3">What you've been fed</span>
          <span className="mono">last 7 days · all sites</span>
          <span className="label">Block</span>
        </div>
        {topics.length === 0 && (
          <p className="set-muted">
            Nothing yet. Topics show up here as you scroll, so you can block the ones flooding your
            feed.
          </p>
        )}
        {topics.map(([topic, count]) => {
          const b = isBlocked(topic)
          const flooding = count >= 10
          return (
            <div key={topic} className="set-topic">
              <div className="set-topic-name">
                <span
                  style={{
                    color: b ? 'var(--mut)' : 'var(--fg)',
                    textDecoration: b ? 'line-through' : 'none',
                  }}
                >
                  {topic}
                </span>
                <span
                  className="mono"
                  style={{ color: !b && flooding ? 'var(--acct)' : 'var(--mut)' }}
                >
                  {b ? 'Blocked · skipping all' : flooding ? 'Flooding your feed' : 'Within limit'}
                </span>
              </div>
              <div className="set-bar">
                <div
                  style={{
                    width: `${Math.min(100, (count / 20) * 100)}%`,
                    background: b ? 'var(--line)' : flooding ? 'var(--acc)' : 'var(--mut)',
                  }}
                />
              </div>
              <span className="mono set-topic-count">×{count}</span>
              <Toggle on={b} onChange={() => toggleTopic(topic)} label={`Block ${topic}`} />
            </div>
          )
        })}
      </div>

      <div className="set-where">
        <span className="set-h3">Where it runs</span>
        <div className="set-sites">
          {SITES.map(([key, name, sub]) => (
            <div key={key} className="set-site" style={{ opacity: settings.sites[key] ? 1 : 0.6 }}>
              <div className="set-site-top">
                <Icon name={key} size={22} />
                <Toggle
                  on={settings.sites[key]}
                  label={name}
                  onChange={(on) => set((s) => ({ ...s, sites: { ...s.sites, [key]: on } }))}
                />
              </div>
              <div className="set-site-text">
                <span>{name}</span>
                <span>{sub}</span>
              </div>
            </div>
          ))}
        </div>
      </div>
    </>
  )
}
