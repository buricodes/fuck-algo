import { createClerkClient } from '@clerk/chrome-extension/client'

const PUBLISHABLE_KEY = import.meta.env.VITE_CLERK_PUBLISHABLE_KEY as string | undefined
const SYNC_HOST = import.meta.env.VITE_CLERK_SYNC_HOST as string | undefined
const READY_TIMEOUT_MS = 15_000
const TOKEN_REUSE_MARGIN_MS = 15_000

type Clerk = Awaited<ReturnType<typeof createClerkClient>>

let clerkPromise: Promise<Clerk> | null = null
let held: { token: string; expiresAt: number } | null = null

export const clerkConfigured = () => Boolean(PUBLISHABLE_KEY && SYNC_HOST)

function client(): Promise<Clerk> {
  clerkPromise ??= createClerkClient({
    publishableKey: PUBLISHABLE_KEY ?? '',
    syncHost: SYNC_HOST,
    background: true,
  })
  return clerkPromise
}

function reset() {
  clerkPromise = null
  held = null
}

function expiryOf(token: string): number {
  try {
    const payload = token.split('.')[1] ?? ''
    const claims = JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/'))) as {
      exp?: unknown
    }
    return typeof claims.exp === 'number' ? claims.exp * 1000 : 0
  } catch {
    return 0
  }
}

async function ready(): Promise<Clerk | null> {
  const timeout = new Promise<null>((resolve) => setTimeout(() => resolve(null), READY_TIMEOUT_MS))
  return Promise.race([client(), timeout])
}

async function signedInClerk(): Promise<Clerk | null> {
  let clerk = await ready()
  if (clerk?.session) return clerk
  reset()
  clerk = await ready()
  return clerk?.session ? clerk : null
}

export async function sessionToken(): Promise<string | null> {
  if (!clerkConfigured()) return null
  if (held && held.expiresAt - Date.now() > TOKEN_REUSE_MARGIN_MS) return held.token
  try {
    const clerk = await signedInClerk()
    const token = clerk?.session ? await clerk.session.getToken() : null
    held = token ? { token, expiresAt: expiryOf(token) } : null
    return token
  } catch {
    reset()
    return null
  }
}

export function forgetToken() {
  held = null
}
