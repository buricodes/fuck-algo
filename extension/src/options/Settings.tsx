import { useEffect, useState } from 'react'

import type { IconName } from '@/shared/icons'
import { request } from '@/shared/messages'
import type { Settings as SettingsShape } from '@/shared/types'
import { dayKey } from '@/shared/util'
import { Icon, Mark } from '@/ui/components'
import { useStore } from '@/ui/hooks/useStore'
import { VoicePanel } from '@/ui/voice/VoicePanel'

import { LicenceCard } from './LicenceCard'
import { Activity } from './pages/Activity'
import { Keywords } from './pages/Keywords'
import { Repeat } from './pages/Repeat'
import { Watchlist } from './pages/Watchlist'

import './settings.css'

type Page = 'watchlist' | 'repeat' | 'keywords' | 'activity'
const PAGES: Page[] = ['watchlist', 'repeat', 'keywords', 'activity']

export type PageProps = {
  settings: SettingsShape
  set: (fn: (s: SettingsShape) => SettingsShape) => void
  openVoice: () => void
}

function pageFromHash(): Page {
  const hash = location.hash.slice(1) as Page
  return PAGES.includes(hash) ? hash : 'watchlist'
}

export function Settings() {
  const [settings, setSettings] = useStore('settings')
  const [activity] = useStore('activity')
  const [stats] = useStore('stats')
  const [page, setPage] = useState<Page>(pageFromHash)
  const [voiceOpen, setVoiceOpen] = useState(() =>
    new URLSearchParams(location.search).has('voice'),
  )
  const [today] = useState(() => dayKey())
  const [weekAgo] = useState(() => dayKey(new Date(Date.now() - 7 * 86_400_000)))

  useEffect(() => {
    const onHash = () => setPage(pageFromHash())
    window.addEventListener('hashchange', onHash)
    void request('licence:refresh', {})
    return () => window.removeEventListener('hashchange', onHash)
  }, [])

  useEffect(() => {
    if (settings) document.documentElement.dataset.theme = settings.theme
  }, [settings])

  if (!settings || !activity || !stats) return <div className="set" />

  const go = (next: Page) => {
    history.replaceState(null, '', `#${next}`)
    setPage(next)
  }
  const live = activity.filter((a) => !a.undone && dayKey(new Date(a.at)) === today).length
  const week = Object.entries(stats.days)
    .filter(([day]) => day > weekAgo)
    .reduce((n, [, d]) => n + d.spoiler + d.repeat + d.keyword, 0)

  const nav: [Page, string, IconName, string][] = [
    ['watchlist', 'Spoiler watchlist', 'shield', String(settings.watchlist.length)],
    ['repeat', 'Repeat blocker', 'repeat', `${settings.repeat.limit}/20`],
    ['keywords', 'Keyword blocks', 'hash', String(settings.keywords.length)],
    ['activity', 'Activity', 'log', String(live)],
  ]
  const props: PageProps = { settings, set: setSettings, openVoice: () => setVoiceOpen(true) }

  return (
    <div className="set">
      <aside className="set-side">
        <div className="set-brand">
          <div className="set-brand-row">
            <div className="display set-word">
              FUCK
              <br />
              ALGO
            </div>
            <Mark size={52} />
          </div>
          <div className="set-tag">Your feed. Your rules.</div>
        </div>
        <button
          type="button"
          className="fa-btn fa-btn-primary set-voice"
          onClick={() => setVoiceOpen(true)}
        >
          <Icon name="mic" size={18} stroke={2} />
          Tell it what to block
        </button>
        <nav className="set-nav">
          {nav.map(([key, label, icon, count]) => (
            <button key={key} type="button" data-on={page === key} onClick={() => go(key)}>
              <Icon name={icon} />
              <span>{label}</span>
              <span className="mono">{count}</span>
            </button>
          ))}
        </nav>
        <div className="set-side-foot">
          <div className="set-card">
            <div className="set-card-title">
              <span className="set-dot" data-on={settings.shield} />
              {settings.shield ? 'Shield on' : 'Shield off'}
              <button
                type="button"
                className="set-theme"
                title={settings.theme === 'dark' ? 'Light theme' : 'Dark theme'}
                onClick={() =>
                  setSettings((s) => ({ ...s, theme: s.theme === 'dark' ? 'light' : 'dark' }))
                }
              >
                <Icon name={settings.theme === 'dark' ? 'sun' : 'moon'} size={16} />
              </button>
            </div>
            <div className="mono set-card-sub">
              {week.toLocaleString('en-GB')} blocked this week
            </div>
          </div>
          <LicenceCard />
        </div>
      </aside>

      <div className="set-main-wrap">
        {voiceOpen && <VoicePanel compact={false} inTab onClose={() => setVoiceOpen(false)} />}
        <main className="set-main">
          {page === 'watchlist' && <Watchlist {...props} />}
          {page === 'repeat' && <Repeat {...props} />}
          {page === 'keywords' && <Keywords {...props} />}
          {page === 'activity' && <Activity />}
        </main>
      </div>
    </div>
  )
}
