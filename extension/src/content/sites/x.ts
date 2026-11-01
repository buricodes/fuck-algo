import { info, textOf } from './types'
import type { Adapter, Found } from './types'

function scan(host: string): Found[] {
  const found: Found[] = []
  document.querySelectorAll<HTMLElement>('article[data-testid="tweet"]').forEach((article) => {
    const status = article
      .querySelector<HTMLAnchorElement>('a[href*="/status/"]')
      ?.getAttribute('href')
    const id = status?.match(/\/status\/(\d+)/)?.[1]
    const body = article.querySelector<HTMLElement>('[data-testid="tweetText"]')
    const text = body?.textContent?.trim() ?? ''
    if (!id || !text) return
    const author = textOf(article, '[data-testid="User-Name"] a span')
    const card = textOf(article, '[data-testid="card.wrapper"]')
    const media = article.querySelector<HTMLElement>(
      '[data-testid="tweetPhoto"], [data-testid="videoPlayer"]',
    )
    const target = media ? article : (body?.parentElement ?? article)
    found.push({
      el: target,
      layout: media ? 'card' : 'row',
      info: info('x', host, `x:${id}`, text.slice(0, 300), author, card),
    })
  })
  return found
}

export const x: Adapter = { site: 'x', scan }
