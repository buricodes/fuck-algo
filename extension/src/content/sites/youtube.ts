import { info, pressArrowDown, textOf } from './types'
import type { Adapter, Found, Player } from './types'

const TILES = [
  'ytd-rich-item-renderer',
  'ytd-video-renderer',
  'ytd-compact-video-renderer',
  'ytd-grid-video-renderer',
  'ytd-reel-item-renderer',
  'ytm-shorts-lockup-view-model-v2',
  'ytm-shorts-lockup-view-model',
  'yt-lockup-view-model',
].join(',')

const TITLE =
  '#video-title | a#video-title-link | h3 a | .yt-lockup-metadata-view-model__title | .yt-lockup-metadata-view-model-wiz__title | .shortsLockupViewModelHostMetadataTitle | h3'
const CHANNEL =
  'ytd-channel-name #text | ytd-channel-name a | a[href^="/@"] | .yt-content-metadata-view-model__metadata-text'

function videoId(href: string | null): string | null {
  if (!href) return null
  const shorts = href.match(/\/shorts\/([\w-]{6,})/)
  if (shorts?.[1]) return shorts[1]
  try {
    return new URL(href, location.origin).searchParams.get('v')
  } catch {
    return null
  }
}

function scanTiles(host: string): Found[] {
  const found: Found[] = []
  document.querySelectorAll<HTMLElement>(TILES).forEach((el) => {
    if (el.parentElement?.closest(TILES)) return
    const link = el.querySelector<HTMLAnchorElement>('a[href*="watch?v="], a[href*="/shorts/"]')
    const id = videoId(link?.getAttribute('href') ?? null)
    const title = textOf(el, TITLE)
    if (!id || !title) return
    found.push({
      el,
      layout: 'card',
      info: info('youtube', host, `yt:${id}`, title, textOf(el, CHANNEL)),
    })
  })
  return found
}

function shortsPlayer(host: string): Player | null {
  const id = location.pathname.match(/^\/shorts\/([\w-]+)/)?.[1]
  if (!id) return null
  const active =
    document.querySelector<HTMLElement>('ytd-reel-video-renderer[is-active]') ??
    document.querySelector<HTMLElement>('ytd-shorts') ??
    document.body
  const title = textOf(
    active,
    'h2.ytShortsVideoTitleViewModelShortsVideoTitle | yt-shorts-video-title-view-model h2 | ytd-reel-player-header-renderer h2 | h1.ytd-reel-player-header-renderer | .ytReelMultiFormatLinkViewModelTitle',
  )
  const channel = textOf(
    active,
    'ytd-channel-name a | yt-reel-channel-bar-view-model a | a[href^="/@"]',
  )
  const hashtags = [...active.querySelectorAll('a[href*="/hashtag/"]')]
    .map((a) => a.textContent?.trim() ?? '')
    .join(' ')
  const video =
    active.querySelector<HTMLVideoElement>('video') ??
    document.querySelector<HTMLVideoElement>('video')
  return {
    key: `yt:${id}`,
    info: info('youtube', host, `yt:${id}`, title, channel, hashtags),
    video,
    next: () => {
      const button = document.querySelector<HTMLElement>(
        '#navigation-button-down button, button[aria-label="Next video"]',
      )
      if (button) button.click()
      else pressArrowDown(video ?? document)
    },
  }
}

export const youtube: Adapter = { site: 'youtube', scan: scanTiles, player: shortsPlayer }
