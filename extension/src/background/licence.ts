import { read, write } from '@/shared/storage'
import type { Licence, Result } from '@/shared/types'

import { WEBSITE_URL, call, ensureInstall } from './api'

export async function refreshLicence(): Promise<Result<Licence>> {
  const result = await call<{ installId: string; licence: Licence }>('/v1/me')
  if (!result.ok) return result
  await write('licence', result.data.licence)
  return { ok: true, data: result.data.licence }
}

export async function activateLicence(key: string): Promise<Result<Licence>> {
  const result = await call<{ licence: Licence }>('/v1/licence/activate', { body: { key } })
  if (!result.ok) return result
  await write('licence', result.data.licence)
  return { ok: true, data: result.data.licence }
}

export async function openCheckout(): Promise<Result<null>> {
  const install = await ensureInstall()
  const result = await call<{ sessionId: string; checkoutUrl: string }>('/v1/checkout', {
    auth: false,
    body: { installId: install?.id ?? null },
  })
  const url = result.ok ? result.data.checkoutUrl : `${WEBSITE_URL}/#pricing`
  await chrome.tabs.create({ url })
  return { ok: true, data: null }
}

export async function spend(kind: 'voice' | 'checks', amount: number) {
  const licence = await read('licence')
  if (!licence || licence.status === 'active') return
  const trial = { ...licence.trial }
  if (kind === 'voice') trial.voiceLeft = Math.max(0, trial.voiceLeft - amount)
  else trial.checksLeft = Math.max(0, trial.checksLeft - amount)
  await write('licence', { ...licence, trial })
}
