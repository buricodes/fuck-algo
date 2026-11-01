import { handle } from '@/shared/messages'
import { read, update, watch, write } from '@/shared/storage'
import { dayKey } from '@/shared/util'

import { runChecks } from './checks'
import { recordBlock, recordSeen } from './ledger'
import { openCheckout, openSignIn, refreshAccount } from './account'
import { cancelVoice, rulesFor, startVoice, stopVoice, transcribe } from './voice'

handle({
  'checks:run': ({ items }) => runChecks(items),
  'activity:record': async ({ entry }) => {
    await recordBlock(entry)
    return null
  },
  'feed:seen': async ({ key, topic }) => {
    await recordSeen(key, topic)
    return null
  },
  'account:refresh': () => refreshAccount(),
  'account:signin': () => openSignIn(),
  'checkout:open': () => openCheckout(),
  'voice:start': () => startVoice(),
  'voice:stop': () => stopVoice(),
  'voice:cancel': () => cancelVoice(),
  'voice:rules': ({ transcript }) => rulesFor(transcript),
  'offscreen:transcribe': ({ audio, contentType }) => transcribe(audio, contentType),
})

async function onInstalled({ reason }: chrome.runtime.InstalledDetails) {
  await write('settings', await read('settings'))
  if (reason === chrome.runtime.OnInstalledReason.INSTALL) {
    await chrome.runtime.openOptionsPage()
  }
  await refreshAccount()
}

chrome.runtime.onInstalled.addListener((details) => {
  void onInstalled(details)
})

chrome.runtime.onStartup.addListener(() => {
  void refreshAccount()
})

async function paintBadge() {
  const [settings, stats] = await Promise.all([read('settings'), read('stats')])
  const day = stats.days[dayKey()]
  const total = day ? day.spoiler + day.repeat + day.keyword : 0
  await chrome.action.setBadgeBackgroundColor({ color: settings.shield ? '#C6FF00' : '#4A4A47' })
  await chrome.action.setBadgeTextColor({ color: '#0A0A0B' })
  await chrome.action.setBadgeText({
    text: !settings.shield ? 'OFF' : total ? String(Math.min(total, 999)) : '',
  })
}
watch('settings', () => void paintBadge())
watch('stats', () => void paintBadge())
void paintBadge()

void chrome.alarms.create('tidy', { periodInMinutes: 60 })

async function tidy() {
  const now = Date.now()
  await update('settings', (s) => ({
    ...s,
    keywords: s.keywords.filter((k) => !k.expiresAt || new Date(k.expiresAt).getTime() > now),
  }))
  await paintBadge()
}

chrome.alarms.onAlarm.addListener((alarm) => {
  if (alarm.name === 'tidy') void tidy()
})
