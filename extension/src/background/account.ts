import { read, write } from '@/shared/storage'
import type { Account, Result } from '@/shared/types'

import { WEBSITE_URL, call } from './api'

export async function refreshAccount(): Promise<Result<Account | null>> {
  const result = await call<Account>('/v1/me')
  if (result.ok) {
    await write('account', result.data)
    return result
  }
  if (result.code === 'unauthorised') {
    await write('account', null)
    return { ok: true, data: null }
  }
  return result
}

export async function openSignIn() {
  await chrome.tabs.create({ url: `${WEBSITE_URL}/sign-in` })
  return null
}

export async function openCheckout(): Promise<Result<null>> {
  await chrome.tabs.create({ url: `${WEBSITE_URL}/?buy=1#pricing` })
  return { ok: true, data: null }
}

export async function spend(kind: 'voice' | 'checks', amount: number) {
  const account = await read('account')
  if (!account || account.licence.status === 'active') return
  const trial = { ...account.licence.trial }
  if (kind === 'voice') trial.voiceLeft = Math.max(0, trial.voiceLeft - amount)
  else trial.checksLeft = Math.max(0, trial.checksLeft - amount)
  await write('account', { ...account, licence: { ...account.licence, trial } })
}
