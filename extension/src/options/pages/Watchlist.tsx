import { useState } from 'react'

import type { SpoilerType, Strictness } from '@/shared/types'
import { newId, plural, splitList } from '@/shared/util'
import { Button, Icon, StepSlider } from '@/ui/components'

import type { PageProps } from '../Settings'

const TYPES: [SpoilerType, string][] = [
  ['anime', 'Anime'],
  ['game', 'Game'],
  ['show', 'Show'],
  ['movie', 'Movie'],
  ['sports', 'Sports'],
]
const TYPE_LABEL = Object.fromEntries(TYPES) as Record<SpoilerType, string>
const LEVELS = ['Relaxed', 'Balanced', 'Paranoid'] as const
const HINTS = ['Exact title only', 'Title + keywords', 'Anything adjacent'] as const
const SUGGESTIONS: [string, SpoilerType][] = [
  ['One Piece', 'anime'],
  ['Elden Ring DLC', 'game'],
  ['F1 Abu Dhabi GP', 'sports'],
  ['Dune: Part Three', 'movie'],
]

export function Watchlist({ settings, set, openVoice }: PageProps) {
  const [name, setName] = useState('')
  const [kw, setKw] = useState('')
  const [type, setType] = useState<SpoilerType>('anime')
  const watch = settings.watchlist

  const add = () => {
    const title = name.trim()
    if (!title) return
    set((s) => ({
      ...s,
      watchlist: [
        {
          id: newId(),
          name: title,
          type,
          keywords: splitList(kw),
          level: 1,
          createdAt: new Date().toISOString(),
        },
        ...s.watchlist.filter((w) => w.name.toLowerCase() !== title.toLowerCase()),
      ],
    }))
    setName('')
    setKw('')
  }
  const onEnter = (e: React.KeyboardEvent) => e.key === 'Enter' && add()
  const setLevel = (id: string, level: Strictness) =>
    set((s) => ({ ...s, watchlist: s.watchlist.map((w) => (w.id === id ? { ...w, level } : w)) }))
  const remove = (id: string) =>
    set((s) => ({ ...s, watchlist: s.watchlist.filter((w) => w.id !== id) }))

  return (
    <>
      <header className="set-header">
        <div className="eyebrow">01 · Spoiler watchlist</div>
        <h1 className="display">Nobody spoils it for you.</h1>
        <p>
          Add what you're watching, playing or following. We scrub it from every feed until you say
          otherwise.
        </p>
      </header>

      <div className="set-add">
        <div className="set-add-row">
          <input
            className="set-add-name"
            value={name}
            onChange={(e) => setName(e.target.value)}
            onKeyDown={onEnter}
            placeholder="Add a title, e.g. Chainsaw Man, Elden Ring, F1"
            maxLength={120}
          />
          <Button icon="plus" onClick={add}>
            Add
          </Button>
        </div>
        <div className="set-add-types">
          {TYPES.map(([key, label]) => (
            <button
              key={key}
              type="button"
              className="set-chip"
              data-on={type === key}
              onClick={() => setType(key)}
            >
              <Icon name={key} size={15} />
              {label}
            </button>
          ))}
          <span className="set-sep" />
          <input
            className="set-add-kw"
            value={kw}
            onChange={(e) => setKw(e.target.value)}
            onKeyDown={onEnter}
            placeholder="+ extra keywords, comma separated (optional)"
          />
        </div>
      </div>

      {watch.length === 0 ? (
        <div className="set-empty">
          <Icon name="shield" size={40} stroke={1.5} style={{ color: 'var(--mut)' }} />
          <div className="display">Nothing to protect yet.</div>
          <p>
            Add a show, game or match above, or just say it out loud. We'll catch spoilers before
            you scroll into them.
          </p>
          <button type="button" className="fa-btn fa-btn-outline" onClick={openVoice}>
            <Icon name="mic" size={16} stroke={2} />
            Say it instead
          </button>
          <div className="set-suggest">
            <span className="label">Try</span>
            {SUGGESTIONS.map(([label, t]) => (
              <button
                key={label}
                type="button"
                onClick={() => {
                  setName(label)
                  setType(t)
                }}
              >
                {label}
              </button>
            ))}
          </div>
        </div>
      ) : (
        <div className="set-list">
          <div className="set-list-head">
            <span>{plural(watch.length, 'title')} protected</span>
            <span style={{ width: 316 }}>Strictness</span>
          </div>
          {watch.map((w) => {
            const n = w.keywords.length
            const meta = `${TYPE_LABEL[w.type]} · ${plural(n, 'keyword')}${n ? ` · ${w.keywords.slice(0, 3).join(', ')}${n > 3 ? ` +${n - 3}` : ''}` : ''}`
            return (
              <div key={w.id} className="set-watch">
                <div className="set-tile">
                  <Icon name={w.type} size={22} />
                </div>
                <div className="set-watch-text">
                  <div className="set-watch-name">{w.name}</div>
                  <div className="set-watch-meta">{meta}</div>
                </div>
                <div className="set-watch-level">
                  <div className="set-watch-level-head">
                    <span style={{ color: w.level === 2 ? 'var(--acct)' : 'var(--fg)' }}>
                      {LEVELS[w.level]}
                    </span>
                    <span>{HINTS[w.level]}</span>
                  </div>
                  <StepSlider
                    value={w.level}
                    labels={LEVELS}
                    onChange={(i) => setLevel(w.id, i as Strictness)}
                  />
                </div>
                <Button variant="icon" icon="close" title="Remove" onClick={() => remove(w.id)} />
              </div>
            )
          })}
        </div>
      )}
    </>
  )
}
