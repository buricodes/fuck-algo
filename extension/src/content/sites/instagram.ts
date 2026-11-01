import { info, scrollParentOf, textOf } from './types'
import type { Adapter, Found, Player } from './types'

function postId(href: string | null | undefined): string | null {
  return href?.match(/\/(?:p|reel|reels)\/([\w-]+)/)?.[1] ?? null
}

function scan(host: string): Found[] {
  const found: Found[] = []
  document.querySelectorAll<HTMLElement>('article').forEach((article) => {
    const id = postId(
      article
        .querySelector<HTMLAnchorElement>('a[href*="/p/"], a[href*="/reel/"]')
        ?.getAttribute('href'),
    )
    if (!id) return
    const caption = textOf(article, 'h1 | ul li span | span[dir="auto"]')
    const alt = article.querySelector('img[alt]')?.getAttribute('alt') ?? ''
    const author = textOf(article, 'header a[role="link"] | header a')
    const title = caption || alt
    if (!title) return
    const media =
      article
        .querySelector<HTMLElement>('div[role="presentation"], video, img[alt]')
        ?.closest<HTMLElement>('div') ?? article
    found.push({
      el: media,
      layout: 'card',
      info: info('instagram', host, `ig:${id}`, title.slice(0, 300), author, alt),
    })
  })
  document
    .querySelectorAll<HTMLAnchorElement>('main a[href*="/reel/"], main a[href*="/p/"]')
    .forEach((a) => {
      if (a.closest('article')) return
      const id = postId(a.getAttribute('href'))
      const alt =
        a.querySelector('img[alt]')?.getAttribute('alt') ?? a.getAttribute('aria-label') ?? ''
      if (!id || !alt) return
      found.push({
        el: a,
        layout: 'card',
        info: info('instagram', host, `ig:${id}`, alt.slice(0, 300)),
      })
    })
  return found
}

function reelsPlayer(host: string): Player | null {
  if (!location.pathname.startsWith('/reels/') && !location.pathname.startsWith('/reel/'))
    return null
  const id = postId(location.pathname)
  if (!id) return null
  const videos = [...document.querySelectorAll<HTMLVideoElement>('video')]
  const video =
    videos.find((v) => {
      const r = v.getBoundingClientRect()
      return r.top < innerHeight / 2 && r.bottom > innerHeight / 2
    }) ?? null
  const scope =
    video?.closest<HTMLElement>('div[role="presentation"]')?.parentElement ?? document.body
  const caption = textOf(scope, 'h1 | span[dir="auto"]')
  const author = textOf(scope, 'a[href^="/"][role="link"] span')
  return {
    key: `ig:${id}`,
    info: info('instagram', host, `ig:${id}`, caption.slice(0, 300), author),
    video,
    next: () => {
      const scroller = video ? scrollParentOf(video) : null
      if (scroller) scroller.scrollBy({ top: scroller.clientHeight, behavior: 'smooth' })
      else window.scrollBy({ top: innerHeight, behavior: 'smooth' })
    },
  }
}

export const instagram: Adapter = { site: 'instagram', scan, player: reelsPlayer }
