import { read, write } from '@/shared/storage'
import type { ApiErrorCode, Install, Result } from '@/shared/types'

export const API_URL =
  (import.meta.env.VITE_API_URL as string | undefined) ?? 'http://localhost:8787'
export const WEBSITE_URL =
  (import.meta.env.VITE_WEBSITE_URL as string | undefined) ?? 'https://fuckalgo.app'

type Envelope<T> =
  { ok: true; data: T } | { ok: false; error: { code: ApiErrorCode; message: string } }

type CallOptions = { body?: unknown; form?: FormData; auth?: boolean; method?: 'GET' | 'POST' }

let registering: Promise<Install | null> | null = null

export async function ensureInstall(): Promise<Install | null> {
  const stored = await read('install')
  if (stored) return stored
  registering ??= (async () => {
    const result = await call<{ installId: string; token: string; licence: unknown }>(
      '/v1/installs',
      {
        auth: false,
        body: { version: chrome.runtime.getManifest().version, browser: 'chrome' },
      },
    )
    if (!result.ok) return null
    const install = { id: result.data.installId, token: result.data.token }
    await write('install', install)
    return install
  })().finally(() => {
    registering = null
  })
  return registering
}

export async function call<T>(path: string, options: CallOptions = {}): Promise<Result<T>> {
  const headers: Record<string, string> = {}
  if (options.auth !== false) {
    const install = await ensureInstall()
    if (!install) return { ok: false, code: 'network', message: 'Could not reach Fuck Algo.' }
    headers.Authorization = `Bearer ${install.token}`
  }
  let body: BodyInit | undefined
  if (options.form) body = options.form
  else if (options.body !== undefined) {
    headers['Content-Type'] = 'application/json'
    body = JSON.stringify(options.body)
  }

  let response: Response
  try {
    response = await fetch(`${API_URL}${path}`, {
      method: options.method ?? (body ? 'POST' : 'GET'),
      headers,
      body,
    })
  } catch {
    return {
      ok: false,
      code: 'network',
      message: 'Could not reach Fuck Algo. Check your connection.',
    }
  }

  let envelope: Envelope<T>
  try {
    envelope = (await response.json()) as Envelope<T>
  } catch {
    return { ok: false, code: 'upstream', message: `Unexpected reply (${response.status}).` }
  }
  if (envelope.ok) return { ok: true, data: envelope.data }

  if (envelope.error.code === 'unauthorised' && options.auth !== false) {
    await chrome.storage.local.remove('install')
  }
  return { ok: false, code: envelope.error.code, message: envelope.error.message }
}
