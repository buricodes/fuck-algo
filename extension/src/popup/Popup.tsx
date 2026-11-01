import { useEffect, useState } from 'react'

import type { IconName } from '@/shared/icons'
import type { Settings } from '@/shared/types'
import { SITE_NAMES, clock, dayKey, plural, siteOf } from '@/shared/util'
import { Icon, Mark, Pill, StatTile, Toggle } from '@/ui/components'
import { useStore } from '@/ui/hooks/useStore'
import { VoicePanel } from '@/ui/voice/VoicePanel'

import './popup.css'

type Tab = { host: string; path: string; web: boolean }

function useActiveTab(): Tab | null {
  const [tab, setTab] = useState<Tab | null>(null)
  useEffect(() => {
    void chrome.tabs.query({ active: true, currentWindow: true }).then(([t]) => {
      try {
        const url = new URL(t?.url ?? '')
        const web = url.protocol === 'http:' || url.protocol === 'https:'
        setTab({ host: url.hostname.replace(/^www\./, ''), path: url.pathname, web })
      } catch {
        setTab({ host: '', path: '', web: false })
      }
    })
  }, [])
  return tab
}

function surfaceName(tab: Tab): string {
  const site = siteOf(tab.host)
  if (site === 'youtube' && tab.path.startsWith('/shorts')) return 'YouTube Shorts'
  if (site === 'instagram' && tab.path.startsWith('/reel')) return 'Instagram Reels'
  return site === 'web' ? tab.host : SITE_NAMES[site]
}

function since(iso: string): string {
  const d = new Date(iso)
  return dayKey(d) === dayKey()
    ? clock(iso)
    : d.toLocaleDateString('en-GB', { day: 'numeric', month: 'short' })
}

export function Popup() {
  const [settings, setSettings] = useStore('settings')
  const [stats] = useStore('stats')
  const tab = useActiveTab()
  const [voiceOpen, setVoiceOpen] = useState(false)

  if (!settings || !stats || !tab) return <div className="pop" />

  const today = stats.days[dayKey()] ?? { spoiler: 0, repeat: 0, keyword: 0 }
  const paused = settings.pausedHosts.includes(tab.host)
  const siteOff = tab.web && !settings.sites[siteOf(tab.host)]
  const shield = settings.shield
  const live = shield && !paused && tab.web && !siteOff
  const status = !shield
    ? 'off'
    : !tab.web
      ? 'nothing to guard here'
      : siteOff
        ? 'not watched'
        : paused
          ? 'paused'
          : 'active'

  const set = (fn: (s: Settings) => Settings) => setSettings(fn)
  const toggleShield = () =>
    set((s) => ({
      ...s,
      shield: !s.shield,
      shieldSince: s.shield ? s.shieldSince : new Date().toISOString(),
    }))
  const togglePause = () =>
    shield &&
    tab.web &&
    set((s) => ({
      ...s,
      pausedHosts: paused
        ? s.pausedHosts.filter((h) => h !== tab.host)
        : [...s.pausedHosts, tab.host],
    }))
  const flip = (key: keyof Settings['features']) => (on: boolean) =>
    set((s) => ({ ...s, features: { ...s.features, [key]: on } }))

  const toggles: { key: keyof Settings['features']; label: string; sub: string; icon: IconName }[] =
    [
      {
        key: 'autoScroll',
        label: 'Auto-scroll Shorts',
        sub: 'Swipes past the slop for you',
        icon: 'scroll',
      },
      {
        key: 'spoiler',
        label: 'Spoiler Shield',
        sub: settings.watchlist.length
          ? `${plural(settings.watchlist.length, 'title')} on watch`
          : 'Nothing on watch yet',
        icon: 'eyeoff',
      },
      {
        key: 'repeat',
        label: 'Repeat Blocker',
        sub: `Max ${settings.repeat.limit} similar in last 20`,
        icon: 'repeat',
      },
    ]

  return (
    <div className="pop">
      <div className="pop-head">
        <div className="pop-brand">
          <span className="display">FUCK ALGO</span>
          <Mark size={28} />
        </div>
        <button
          type="button"
          className="pop-round"
          title="Full settings"
          onClick={() => void chrome.runtime.openOptionsPage()}
        >
          <Icon name="sliders" />
        </button>
      </div>

      <div className="pop-shield">
        <div className="label">Shield</div>
        <div className="pop-shield-row">
          <div className="display pop-big" style={{ color: shield ? 'var(--acc)' : 'var(--off)' }}>
            {shield ? 'ON' : 'OFF'}
          </div>
          <Toggle size="lg" on={shield} onChange={toggleShield} label="Toggle shield" />
        </div>
        <div className="pop-sub">
          {!shield
            ? 'The algorithm is driving again.'
            : paused
              ? 'Paused here. Everywhere else: guarded.'
              : `Guarding your feed since ${since(settings.shieldSince)}`}
        </div>
      </div>

      <div className="pop-site">
        <div className="pop-site-icon">
          <Icon name={tab.web ? siteOf(tab.host) : 'web'} size={20} />
        </div>
        <div className="pop-site-text">
          <div className="mono">
            {tab.web ? `${tab.host}${tab.path.split('/').slice(0, 2).join('/')}` : 'browser page'}
          </div>
          <div>
            {tab.web ? surfaceName(tab) : 'This tab'}: {status}
          </div>
        </div>
        <Pill live={live}>{!shield ? 'OFF' : paused ? 'PAUSED' : live ? 'LIVE' : 'IDLE'}</Pill>
      </div>

      <div className="pop-stats" style={{ opacity: shield ? 1 : 0.35 }}>
        <StatTile n={today.spoiler + today.repeat + today.keyword} label="blocked today" accent />
        <StatTile n={today.spoiler} label="spoilers caught" />
        <StatTile n={today.repeat} label="repeats skipped" />
      </div>

      <div className="pop-toggles" style={{ opacity: shield ? 1 : 0.35 }}>
        {toggles.map((t) => (
          <div key={t.key} className="pop-toggle">
            <span className="pop-toggle-icon">
              <Icon name={t.icon} />
            </span>
            <div>
              <span>{t.label}</span>
              <span>{t.sub}</span>
            </div>
            <Toggle on={settings.features[t.key]} onChange={flip(t.key)} label={t.label} />
          </div>
        ))}
      </div>

      <div className="pop-foot">
        <button
          type="button"
          className="fa-btn fa-btn-primary pop-voice"
          onClick={() => setVoiceOpen(true)}
        >
          <Icon name="mic" size={18} stroke={2} />
          Tell it what to block
        </button>
        <div className="pop-foot-row">
          <button
            type="button"
            className="pop-pause"
            disabled={!shield || !tab.web}
            style={{ color: paused && shield ? 'var(--acc)' : 'var(--fg)' }}
            onClick={togglePause}
          >
            <Icon name={paused ? 'play' : 'pause'} size={14} stroke={2} />
            {paused ? 'Resume on this site' : 'Pause on this site'}
          </button>
          <button
            type="button"
            className="pop-link"
            onClick={() => void chrome.runtime.openOptionsPage()}
          >
            Full settings
            <Icon name="arrow" size={14} stroke={2} />
          </button>
        </div>
      </div>

      {voiceOpen && <VoicePanel compact inTab={false} onClose={() => setVoiceOpen(false)} />}
    </div>
  )
}
