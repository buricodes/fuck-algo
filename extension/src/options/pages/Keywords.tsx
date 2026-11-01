import { useState } from 'react'

import type { IconName } from '@/shared/icons'
import type { KeywordAction } from '@/shared/types'
import { newId } from '@/shared/util'
import { Button, Icon, Segmented } from '@/ui/components'

import type { PageProps } from '../Settings'

const ACTS = [
  ['hide', 'Hide'],
  ['blur', 'Blur'],
  ['skip', 'Skip'],
] as const satisfies readonly (readonly [KeywordAction, string])[]

const LEGEND: { icon: IconName; label: string; text: string }[] = [
  { icon: 'eyeoff', label: 'Hide', text: 'Gone. No trace, no gap in the feed.' },
  { icon: 'blur', label: 'Blur', text: 'Covered with a badge. Click to peek.' },
  { icon: 'skip', label: 'Skip', text: 'Auto-scrolled right past. You never see it.' },
]

export function Keywords({ settings, set }: PageProps) {
  const [word, setWord] = useState('')
  const [action, setAction] = useState<KeywordAction>('hide')

  const add = () => {
    const w = word.trim()
    if (!w) return
    set((s) => ({
      ...s,
      keywords: [
        ...s.keywords.filter((k) => k.word.toLowerCase() !== w.toLowerCase()),
        { id: newId(), word: w, action, expiresAt: null },
      ],
    }))
    setWord('')
  }
  const setAct = (id: string, a: KeywordAction) =>
    set((s) => ({ ...s, keywords: s.keywords.map((k) => (k.id === id ? { ...k, action: a } : k)) }))
  const remove = (id: string) =>
    set((s) => ({ ...s, keywords: s.keywords.filter((k) => k.id !== id) }))

  return (
    <>
      <header className="set-header">
        <div className="eyebrow">03 · Keyword blocks</div>
        <h1 className="display">Words you're done with.</h1>
        <p>Add a word or phrase, then decide what happens when it shows up anywhere.</p>
      </header>

      <div className="set-kw-add">
        <input
          value={word}
          onChange={(e) => setWord(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && add()}
          placeholder="Add a keyword or phrase"
          maxLength={80}
        />
        <Segmented value={action} options={ACTS} onChange={setAction} />
        <Button icon="plus" onClick={add}>
          Add
        </Button>
      </div>

      <div className="set-chips">
        {settings.keywords.map((k) => (
          <div key={k.id} className="set-kw">
            <span>{k.word}</span>
            {k.expiresAt && (
              <span className="mono set-kw-exp">
                until{' '}
                {new Date(k.expiresAt).toLocaleDateString('en-GB', {
                  weekday: 'short',
                  day: 'numeric',
                })}
              </span>
            )}
            <Segmented
              size="sm"
              value={k.action}
              options={ACTS}
              onChange={(a) => setAct(k.id, a)}
            />
            <button type="button" className="set-kw-x" title="Remove" onClick={() => remove(k.id)}>
              <Icon name="close" size={14} stroke={2} />
            </button>
          </div>
        ))}
        {settings.keywords.length === 0 && (
          <div className="set-muted">No blocked words. Lucky you.</div>
        )}
      </div>

      <div className="set-legend">
        {LEGEND.map((l) => (
          <div key={l.label}>
            <Icon name={l.icon} size={22} />
            <span className="mono">{l.label}</span>
            <span>{l.text}</span>
          </div>
        ))}
      </div>
    </>
  )
}
