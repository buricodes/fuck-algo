import { useState } from 'react'

import { request } from '@/shared/messages'
import type { ProposedRule, Site } from '@/shared/types'
import { SITE_NAMES } from '@/shared/util'

import { Icon, PulseDot } from '../components'
import type { IconName } from '@/shared/icons'
import { useVoice } from '../hooks/useVoice'

import './voice.css'

const EXAMPLES = [
  "I haven't watched the new Black Clover episode yet.",
  'Stop showing me MrBeast style challenge videos.',
  'Blur anything about the election until Friday.',
]

const TYPE_LABEL = {
  anime: 'Anime',
  game: 'Game',
  show: 'Show',
  movie: 'Movie',
  sports: 'Sports',
} as const
const LEVEL_LABEL = ['Relaxed', 'Balanced', 'Paranoid'] as const
const ACTION_LABEL = { hide: 'Hide', blur: 'Blur', skip: 'Skip' } as const
const RULE_ICON: Record<ProposedRule['kind'], IconName> = {
  spoiler: 'eyeoff',
  repeat: 'repeat',
  keyword: 'hash',
}

function describe(rule: ProposedRule): { kind: string; name: string; detail: string } {
  if (rule.kind === 'spoiler') {
    const extra = rule.keywords.length ? ` · + ${rule.keywords.slice(0, 3).join(', ')}` : ''
    return {
      kind: 'Spoiler watchlist',
      name: rule.name,
      detail: `${TYPE_LABEL[rule.type]} · ${LEVEL_LABEL[rule.level]}${extra}`,
    }
  }
  if (rule.kind === 'repeat') {
    const sites = rule.sites.map((s: Site) => SITE_NAMES[s]).join(', ')
    return {
      kind: 'Repeat blocker',
      name: rule.topic,
      detail: sites ? `Block topic on ${sites}` : 'Block topic · skip on sight',
    }
  }
  const expires = rule.expiresAt
    ? ` · expires ${new Date(rule.expiresAt).toLocaleDateString('en-GB', { weekday: 'long', day: 'numeric', month: 'short' })}`
    : ' everywhere'
  return { kind: 'Keyword', name: rule.word, detail: `${ACTION_LABEL[rule.action]}${expires}` }
}

function Wave({ bars }: { bars: number }) {
  return (
    <div className="fa-wave">
      {Array.from({ length: bars }, (_, i) => (
        <span
          key={i}
          data-accent={i % 7 === 3}
          style={{
            animationDuration: `${0.7 + ((i * 37) % 50) / 100}s`,
            animationDelay: `${-((i * 53) % 90) / 100}s`,
          }}
        />
      ))}
    </div>
  )
}

export function VoicePanel({
  compact,
  inTab,
  onClose,
}: {
  compact: boolean
  inTab: boolean
  onClose: () => void
}) {
  const voice = useVoice({ inTab })
  const [typed, setTyped] = useState('')
  const nOn = voice.rules.length - voice.off.size

  const titles = {
    idle: 'What are you done seeing?',
    listening: "Go on, we're listening.",
    thinking: 'Working it out…',
    review: voice.rules.length ? "Here's what we heard." : 'Nothing to block in that.',
    done: `${voice.applied} ${voice.applied === 1 ? 'rule' : 'rules'} live.`,
  }

  const close = () => {
    voice.reset()
    onClose()
  }

  const sendTyped = () => {
    voice.submitText(typed)
    setTyped('')
  }

  return (
    <div className="fa-voice" data-compact={compact}>
      <div className="fa-voice-head">
        <div>
          <div className="eyebrow">{voice.stage === 'done' ? 'Applied' : 'Voice rules'}</div>
          <div className="display fa-voice-title">{titles[voice.stage]}</div>
        </div>
        <button type="button" className="fa-voice-close" title="Close" onClick={close}>
          <Icon name="close" size={16} stroke={2} />
        </button>
      </div>

      {voice.error && (
        <div className="fa-voice-error" role="alert">
          <span>{voice.error.message}</span>
          {voice.error.code === 'trial_exhausted' && (
            <button type="button" onClick={() => void request('checkout:open', {})}>
              Unlock for $5
            </button>
          )}
        </div>
      )}

      {voice.stage === 'idle' && (
        <>
          <div className="fa-voice-idle">
            <button
              type="button"
              className="fa-voice-mic"
              aria-label="Start talking"
              onClick={() => void voice.start()}
            >
              <span className="fa-voice-ring" />
              <Icon name="mic" size={28} stroke={2} />
            </button>
            <div>
              <strong>Tap and talk.</strong>
              <span>Say it like you'd tell a friend.</span>
            </div>
          </div>
          <div className="fa-voice-examples">
            <span className="label">Try saying</span>
            {EXAMPLES.slice(0, compact ? 2 : 3).map((text) => (
              <button key={text} type="button" onClick={() => voice.submitText(text)}>
                “{text}”
              </button>
            ))}
          </div>
          <div className="fa-voice-type">
            <input
              value={typed}
              onChange={(e) => setTyped(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && sendTyped()}
              placeholder="…or type it"
              maxLength={2000}
            />
            <button type="button" title="Send" onClick={sendTyped}>
              <Icon name="arrow" size={16} stroke={2} />
            </button>
          </div>
        </>
      )}

      {voice.stage === 'listening' && (
        <>
          <div className="fa-voice-live">
            <div className="fa-voice-transcript">
              {voice.transcript}
              <span className="fa-caret">▍</span>
            </div>
            <Wave bars={compact ? 28 : 48} />
          </div>
          <div className="fa-voice-row">
            <span className="fa-voice-rec">
              <span />
              Listening
            </span>
            <button
              type="button"
              className="fa-btn fa-btn-primary fa-voice-stop"
              onClick={voice.stop}
            >
              <span className="fa-voice-square" />
              Done talking
            </button>
          </div>
        </>
      )}

      {voice.stage === 'thinking' && (
        <div className="fa-voice-live">
          <div className="fa-voice-quote">“{voice.transcript}”</div>
          <span className="fa-voice-checking">
            <PulseDot />
            turning that into rules…
          </span>
        </div>
      )}

      {voice.stage === 'review' && (
        <>
          <div className="fa-voice-quote">“{voice.transcript}”</div>
          <div className="fa-voice-rules">
            {voice.rules.map((rule, i) => {
              const d = describe(rule)
              const on = !voice.off.has(i)
              return (
                <div
                  key={`${rule.kind}-${d.name}`}
                  className="fa-voice-rule"
                  style={{ opacity: on ? 1 : 0.45 }}
                >
                  <div className="fa-voice-rule-icon">
                    <Icon name={RULE_ICON[rule.kind]} />
                  </div>
                  <div className="fa-voice-rule-text">
                    <span className="fa-voice-kind">{d.kind}</span>
                    <span className="fa-voice-name">{d.name}</span>
                    <span className="fa-voice-detail">{d.detail}</span>
                  </div>
                  <button
                    type="button"
                    title="Include"
                    aria-pressed={on}
                    className="fa-voice-check"
                    data-on={on}
                    onClick={() => voice.toggleRule(i)}
                  >
                    <Icon name="check" size={14} stroke={3} />
                  </button>
                </div>
              )
            })}
          </div>
          <div className="fa-voice-actions">
            <button
              type="button"
              className="fa-btn fa-btn-ghost"
              onClick={() => void voice.start()}
            >
              <Icon name="mic" size={16} stroke={2} />
              Add more
            </button>
            <button
              type="button"
              className="fa-btn fa-btn-primary fa-grow"
              disabled={!nOn}
              onClick={() => void voice.apply()}
            >
              {nOn ? `Apply ${nOn} ${nOn === 1 ? 'rule' : 'rules'}` : 'Nothing selected'}
            </button>
          </div>
        </>
      )}

      {voice.stage === 'done' && (
        <>
          <div className="fa-voice-done">
            <div className="fa-voice-tick">
              <Icon name="check" size={32} stroke={2.5} />
            </div>
            <p>Changes are live on every site. Undo anytime from Activity.</p>
          </div>
          <div className="fa-voice-actions">
            <button type="button" className="fa-btn fa-btn-ghost fa-grow" onClick={voice.reset}>
              Say something else
            </button>
            <button type="button" className="fa-btn fa-btn-light fa-grow" onClick={close}>
              Done
            </button>
          </div>
        </>
      )}
    </div>
  )
}
