import { normalise } from '@/engine/text'
import { request } from '@/shared/messages'
import { read } from '@/shared/storage'
import type { ProposedRule, Result } from '@/shared/types'

import { call } from './api'
import { spend } from './licence'

const OFFSCREEN_URL = 'src/offscreen/index.html'
let creating: Promise<void> | null = null

async function ensureOffscreen() {
  if (await chrome.offscreen.hasDocument()) return
  creating ??= chrome.offscreen
    .createDocument({
      url: OFFSCREEN_URL,
      reasons: [chrome.offscreen.Reason.USER_MEDIA],
      justification: 'Listen while you say what you want blocked',
    })
    .finally(() => {
      creating = null
    })
  await creating
}

export async function startVoice(): Promise<Result<null>> {
  const token = await call<{ token: string }>('/v1/voice/token', { method: 'POST' })
  if (!token.ok && token.code !== 'unavailable' && token.code !== 'upstream') return token
  if (token.ok) await spend('voice', 1)
  await ensureOffscreen()
  return request('offscreen:start', { token: token.ok ? token.data.token : null })
}

export async function stopVoice() {
  if (await chrome.offscreen.hasDocument()) await request('offscreen:stop', {})
  return null
}

export async function cancelVoice() {
  if (await chrome.offscreen.hasDocument()) await request('offscreen:cancel', {})
  return null
}

export async function transcribe(audio: string, contentType: string): Promise<Result<string>> {
  const bytes = Uint8Array.from(atob(audio), (c) => c.charCodeAt(0))
  const form = new FormData()
  form.append('audio', new Blob([bytes], { type: contentType }), 'speech.webm')
  const result = await call<{ transcript: string }>('/v1/voice/transcribe', { form })
  return result.ok ? { ok: true, data: result.data.transcript } : result
}

export async function rulesFor(
  transcript: string,
): Promise<Result<{ rules: ProposedRule[]; summary: string }>> {
  const settings = await read('settings')
  const feed = await read('feed')
  const topics = new Set<string>(settings.repeat.blockedTopics)
  for (const day of Object.values(feed.topicsByDay)) {
    for (const topic of Object.keys(day)) if (!topic.startsWith('Channel: ')) topics.add(topic)
  }
  const result = await call<{ rules: ProposedRule[]; summary: string }>('/v1/voice/rules', {
    body: {
      transcript: transcript.slice(0, 2000),
      now: new Date().toISOString(),
      context: {
        watchlist: settings.watchlist.map((w) => w.name),
        keywords: settings.keywords.map((k) => k.word),
        topics: [...topics].slice(0, 40),
      },
    },
  })
  if (!result.ok) return result
  const seen = new Set<string>()
  const rules = result.data.rules.filter((rule) => {
    const id = `${rule.kind}:${normalise(rule.kind === 'spoiler' ? rule.name : rule.kind === 'repeat' ? rule.topic : rule.word)}`
    if (seen.has(id)) return false
    seen.add(id)
    return true
  })
  return { ok: true, data: { rules, summary: result.data.summary } }
}
