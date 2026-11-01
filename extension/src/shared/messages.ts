import type { Account, ActivityEntry, CheckResult, ItemInfo, ProposedRule, Result } from './types'

export type Requests = {
  'checks:run': { req: { items: ItemInfo[] }; res: Result<CheckResult[]> }
  'activity:record': { req: { entry: Omit<ActivityEntry, 'id' | 'at' | 'undone'> }; res: null }
  'feed:seen': { req: { key: string; topic: string }; res: null }
  'account:refresh': { req: Record<string, never>; res: Result<Account | null> }
  'account:signin': { req: Record<string, never>; res: null }
  'checkout:open': { req: Record<string, never>; res: Result<null> }
  'voice:start': { req: Record<string, never>; res: Result<null> }
  'voice:stop': { req: Record<string, never>; res: null }
  'voice:cancel': { req: Record<string, never>; res: null }
  'voice:rules': {
    req: { transcript: string }
    res: Result<{ rules: ProposedRule[]; summary: string }>
  }
  'offscreen:start': { req: { token: string | null }; res: Result<null> }
  'offscreen:stop': { req: Record<string, never>; res: null }
  'offscreen:cancel': { req: Record<string, never>; res: null }
  'offscreen:transcribe': { req: { audio: string; contentType: string }; res: Result<string> }
}

export type Events = {
  'voice:partial': { text: string }
  'voice:final': { transcript: string | null; error: string | null }
}

type Envelope = { type: string; payload: unknown; event?: boolean }

export async function request<K extends keyof Requests>(
  type: K,
  payload: Requests[K]['req'],
): Promise<Requests[K]['res']> {
  return chrome.runtime.sendMessage({ type, payload } satisfies Envelope)
}

export function emit<K extends keyof Events>(type: K, payload: Events[K]) {
  chrome.runtime
    .sendMessage({ type, payload, event: true } satisfies Envelope)
    .catch(() => undefined)
}

type Handler<K extends keyof Requests> = (
  payload: Requests[K]['req'],
  sender: chrome.runtime.MessageSender,
) => Promise<Requests[K]['res']> | Requests[K]['res']

export function handle(handlers: { [K in keyof Requests]?: Handler<K> }) {
  chrome.runtime.onMessage.addListener((message: Envelope, sender, reply) => {
    if (message.event) return false
    const handler = handlers[message.type as keyof Requests] as Handler<keyof Requests> | undefined
    if (!handler) return false
    Promise.resolve(handler(message.payload as never, sender)).then(reply, (error: unknown) => {
      console.error(`[fuck-algo] ${message.type} failed`, error)
      reply({ ok: false, code: 'internal', message: 'Something went wrong.' })
    })
    return true
  })
}

export function onEvent<K extends keyof Events>(type: K, listener: (payload: Events[K]) => void) {
  const wrapped = (message: Envelope) => {
    if (message.event && message.type === type) listener(message.payload as Events[K])
    return false
  }
  chrome.runtime.onMessage.addListener(wrapped)
  return () => chrome.runtime.onMessage.removeListener(wrapped)
}
