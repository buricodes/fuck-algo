import { info, textOf } from './types'
import type { Adapter, Found } from './types'

const keys = new WeakMap<HTMLElement, string>()

function keyFor(el: HTMLElement, host: string, seed: string): string {
  const known = keys.get(el)
  if (known) return known
  let h = 2166136261
  for (let i = 0; i < seed.length; i++) h = Math.imul(h ^ seed.charCodeAt(i), 16777619)
  const key = `web:${host}:${(h >>> 0).toString(36)}`
  keys.set(el, key)
  return key
}

function reddit(host: string): Found[] {
  const found: Found[] = []
  document
    .querySelectorAll<HTMLElement>(
      'shreddit-post, article[data-testid="post-container"], .thing.link',
    )
    .forEach((post) => {
      const title =
        post.getAttribute('post-title') ?? textOf(post, 'a[slot="title"] | h3 | a.title')
      if (!title) return
      const id =
        post.getAttribute('id') ??
        post.getAttribute('data-fullname') ??
        post.getAttribute('permalink') ??
        title
      const sub = post.getAttribute('subreddit-prefixed-name') ?? textOf(post, 'a[href^="/r/"]')
      found.push({ el: post, layout: 'card', info: info('web', host, `rd:${id}`, title, sub) })
    })
  return found
}

function articles(host: string): Found[] {
  const found: Found[] = []
  document.querySelectorAll<HTMLElement>('article, [role="article"]').forEach((el) => {
    if (el.parentElement?.closest('article, [role="article"]')) return
    const title = textOf(el, 'h1 | h2 | h3')
    if (!title || title.length < 8) return
    const summary = textOf(el, 'p').slice(0, 600)
    found.push({
      el,
      layout: 'card',
      info: info('web', host, keyFor(el, host, title), title.slice(0, 300), '', summary),
    })
  })
  return found
}

export const web: Adapter = {
  site: 'web',
  scan: (host) => (host.endsWith('reddit.com') ? reddit(host) : articles(host)),
}
