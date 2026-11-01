import type { ApiErrorCode, Result } from '@/shared/types'

import { forgetToken, sessionToken } from './clerk'

export const API_URL =
  (import.meta.env.VITE_API_URL as string | undefined) ?? 'http://localhost:8787'
export const WEBSITE_URL =
  (import.meta.env.VITE_WEBSITE_URL as string | undefined) ?? 'https://fuck-algo-website.vercel.app'

type Envelope<T> =
  { ok: true; data: T } | { ok: false; error: { code: ApiErrorCode; message: string } }

export const SIGN_IN_MESSAGE = 'Sign in on the Fuck Algo website to use voice and AI checks.'

type CallOptions = { body?: unknown; form?: FormData; auth?: boolean; method?: 'GET' | 'POST' }

export async function call<T>(path: string, options: CallOptions = {}): Promise<Result<T>> {
  const headers: Record<string, string> = {}
  if (options.auth !== false) {
    const token = await sessionToken()
    if (!token) return { ok: false, code: 'unauthorised', message: SIGN_IN_MESSAGE }
    headers.Authorization = `Bearer ${token}`
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

  if (envelope.error.code === 'unauthorised') forgetToken()
  return { ok: false, code: envelope.error.code, message: envelope.error.message }
}
