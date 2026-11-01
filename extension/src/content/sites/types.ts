import type { ItemInfo, Site } from '@/shared/types'

import type { Layout } from '../overlay'

export type Found = { el: HTMLElement; info: ItemInfo; layout: Layout }

export type Player = {
  key: string
  info: ItemInfo
  video: HTMLVideoElement | null
  next: () => void
}

export type Adapter = {
  site: Site
  scan: (host: string) => Found[]
  player?: (host: string) => Player | null
}

export function textOf(root: ParentNode, selectors: string): string {
  for (const selector of selectors.split('|')) {
    const el = root.querySelector<HTMLElement>(selector.trim())
    const text = (
      el?.getAttribute('title') ||
      el?.textContent ||
      el?.getAttribute('aria-label') ||
      ''
    ).trim()
    if (text) return text.replace(/\s+/g, ' ')
  }
  return ''
}

export function info(
  site: Site,
  host: string,
  key: string,
  title: string,
  channel = '',
  text = '',
) {
  return { key, site, host, title, channel, text } satisfies ItemInfo
}

export function pressArrowDown(target: EventTarget = document) {
  const init = {
    key: 'ArrowDown',
    code: 'ArrowDown',
    keyCode: 40,
    which: 40,
    bubbles: true,
    cancelable: true,
  }
  target.dispatchEvent(new KeyboardEvent('keydown', init))
  if (target !== document) document.dispatchEvent(new KeyboardEvent('keydown', init))
}

export function scrollParentOf(el: HTMLElement): HTMLElement | null {
  let node = el.parentElement
  while (node) {
    const { overflowY } = getComputedStyle(node)
    if ((overflowY === 'auto' || overflowY === 'scroll') && node.scrollHeight > node.clientHeight)
      return node
    node = node.parentElement
  }
  return null
}
