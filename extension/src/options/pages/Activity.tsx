import { useState } from 'react'

import type { IconName } from '@/shared/icons'
import { update } from '@/shared/storage'
import type { ActivityEntry, BlockKind } from '@/shared/types'
import { clock, dayKey } from '@/shared/util'
import { Icon } from '@/ui/components'
import { useStore } from '@/ui/hooks/useStore'

type Filter = 'all' | BlockKind

const KIND_ICON: Record<BlockKind, IconName> = {
  spoiler: 'eyeoff',
  repeat: 'repeat',
  keyword: 'hash',
}
const KIND_LABEL: Record<BlockKind, string> = {
  spoiler: 'Spoiler',
  repeat: 'Repeat',
  keyword: 'Keyword',
}
const PAST = { hide: 'Hidden', blur: 'Blurred', skip: 'Skipped' } as const

function dayLabel(iso: string): string {
  const day = dayKey(new Date(iso))
  if (day === dayKey()) return 'Today'
  if (day === dayKey(new Date(Date.now() - 86_400_000))) return 'Yesterday'
  return new Date(iso).toLocaleDateString('en-GB', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
  })
}

async function toggleUndo(entry: ActivityEntry) {
  const undone = !entry.undone
  await update('activity', (log) => log.map((e) => (e.id === entry.id ? { ...e, undone } : e)))
  await update('allow', (keys) =>
    undone ? [...new Set([...keys, entry.key])] : keys.filter((k) => k !== entry.key),
  )
}

export function Activity() {
  const [log] = useStore('activity')
  const [filter, setFilter] = useState<Filter>('all')
  const entries = log ?? []

  const counts: Record<Filter, number> = { all: entries.length, spoiler: 0, repeat: 0, keyword: 0 }
  for (const e of entries) counts[e.kind]++
  const rows = entries.filter((e) => filter === 'all' || e.kind === filter)

  const groups: [string, ActivityEntry[]][] = []
  for (const row of rows) {
    const label = dayLabel(row.at)
    const last = groups[groups.length - 1]
    if (last?.[0] === label) last[1].push(row)
    else groups.push([label, [row]])
  }

  const filters: [Filter, string][] = [
    ['all', 'All'],
    ['spoiler', 'Spoilers'],
    ['repeat', 'Repeats'],
    ['keyword', 'Keywords'],
  ]

  return (
    <>
      <header className="set-header">
        <div className="eyebrow">04 · Activity</div>
        <h1 className="display">What we took off your feed.</h1>
      </header>

      <div className="set-filters">
        {filters.map(([key, label]) => (
          <button
            key={key}
            type="button"
            className="set-filter"
            data-on={filter === key}
            onClick={() => setFilter(key)}
          >
            {label}
            <span className="mono">{counts[key]}</span>
          </button>
        ))}
      </div>

      {rows.length === 0 ? (
        <div className="set-empty">
          <Icon name="log" size={40} stroke={1.5} style={{ color: 'var(--mut)' }} />
          <div className="display">{entries.length === 0 ? 'Clean slate.' : 'Nothing here.'}</div>
          <p>
            {entries.length === 0
              ? "Nothing blocked yet. Go scroll, we'll catch what you don't want to see."
              : 'No items match this filter.'}
          </p>
        </div>
      ) : (
        groups.map(([label, items]) => (
          <div key={label} className="set-log">
            <div className="label set-log-day">{label}</div>
            {items.map((l) => (
              <div key={l.id} className="set-log-row">
                <span className="mono set-log-time">{clock(l.at)}</span>
                <div className="set-log-line">
                  <div />
                  <span style={{ background: l.undone ? 'var(--line)' : 'var(--acc)' }} />
                </div>
                <div className="set-log-site" style={{ opacity: l.undone ? 0.45 : 1 }}>
                  <Icon name={l.site} size={18} />
                </div>
                <div className="set-log-text" style={{ opacity: l.undone ? 0.45 : 1 }}>
                  <div className="set-log-title">{l.title || 'Untitled item'}</div>
                  <div className="set-log-meta">
                    <span className="set-reason">
                      <Icon name={KIND_ICON[l.kind]} size={12} stroke={2} />
                      {KIND_LABEL[l.kind]} · {l.detail}
                    </span>
                    <span>
                      {l.host} · {l.undone ? 'Restored' : PAST[l.action]}
                    </span>
                  </div>
                </div>
                <button type="button" className="set-undo" onClick={() => void toggleUndo(l)}>
                  <Icon name={l.undone ? 'reblock' : 'undo'} size={14} stroke={2} />
                  {l.undone ? 'Re-block' : 'Undo'}
                </button>
              </div>
            ))}
          </div>
        ))
      )}
    </>
  )
}
