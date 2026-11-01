import { ICONS } from '@/shared/icons'
import type { BlockKind } from '@/shared/types'

export type OverlayState = 'blurred' | 'pending' | 'shown' | 'dismissed' | 'hidden'
export type Layout = 'card' | 'row'

export type OverlayActions = {
  show: () => void
  dismiss: () => void
  rehide: () => void
}

const STATE_ATTR = 'data-fa-state'
const BADGE_ICON: Record<BlockKind, string> = {
  spoiler: ICONS.eyeoff,
  repeat: ICONS.repeat,
  keyword: ICONS.hash,
}
const BADGE_LABEL: Record<BlockKind, string> = {
  spoiler: 'Spoiler',
  repeat: 'Repeat',
  keyword: 'Keyword',
}

const PAGE_CSS = `
[${STATE_ATTR}="hidden"]{display:none!important}
[${STATE_ATTR}="blurred"]>:not(fa-overlay){filter:blur(16px) saturate(.7)!important;transition:filter .35s cubic-bezier(.2,0,0,1)}
[${STATE_ATTR}="pending"]>:not(fa-overlay){filter:blur(10px) saturate(.6)!important;transition:filter .35s cubic-bezier(.2,0,0,1)}
[${STATE_ATTR}="dismissed"]>:not(fa-overlay){visibility:hidden!important}
[data-fa-layout="row"][${STATE_ATTR}="blurred"]>:not(fa-overlay){filter:blur(6px)!important}
[data-fa-relative]{position:relative!important}
fa-overlay{position:absolute;inset:0;z-index:2147483000;display:block}
fa-overlay[data-state="shown"]{inset:auto;top:10px;left:10px}
`

const SHADOW_CSS = `
:host{all:initial}
*{box-sizing:border-box}
.wrap{position:absolute;inset:0;display:flex;flex-direction:column;align-items:center;justify-content:center;gap:14px;padding:12px;text-align:center;background:rgba(10,10,11,.5);border-radius:12px;font-family:'Geist','Geist Variable',system-ui,-apple-system,sans-serif;color:#F5F5F0;overflow:hidden}
.row{flex-direction:row;justify-content:space-between;background:rgba(10,10,11,.45);padding:0 10px;border-radius:10px}
.badge{display:flex;align-items:center;gap:6px;background:#C6FF00;color:#0A0A0B;border-radius:999px;padding:5px 11px 5px 8px;font:600 11px/1.2 'Geist Mono',ui-monospace,SFMono-Regular,Menlo,monospace;letter-spacing:.04em;max-width:100%;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.badge svg{flex:none}
.buttons{display:flex;gap:8px;flex-wrap:wrap;justify-content:center}
button{height:32px;padding:0 12px;border-radius:999px;font:600 12.5px 'Geist',system-ui,-apple-system,sans-serif;cursor:pointer;white-space:nowrap}
.ghost{border:1px solid rgba(245,245,240,.4);background:rgba(10,10,11,.4);color:#F5F5F0}
.ghost:hover{border-color:#F5F5F0}
.solid{border:none;background:#F5F5F0;color:#0A0A0B}
.row .buttons{gap:14px;flex-wrap:nowrap}
.row button{border:none;background:none;height:auto;padding:4px}
.row .ghost{color:#F5F5F0}
.row .solid{color:#8E8E88}
.shimmer{position:absolute;inset:0;background:linear-gradient(100deg,transparent 30%,rgba(198,255,0,.14) 50%,transparent 70%);background-size:250% 100%;animation:sh 1.6s linear infinite}
.checking{position:relative;display:flex;align-items:center;gap:8px;background:rgba(10,10,11,.8);border:1px solid #2E2E32;border-radius:999px;padding:6px 12px;font:500 11px 'Geist Mono',ui-monospace,Menlo,monospace;letter-spacing:.06em;color:#F5F5F0}
.dot{width:7px;height:7px;border-radius:50%;background:#C6FF00;animation:pulse 1.1s ease-in-out infinite;flex:none}
.pending{background:transparent}
.dismissed{background:transparent;border:1px dashed #2E2E32;gap:8px}
.dismissed strong{font-size:14px;font-weight:600;color:#F5F5F0}
.undo{border:none;background:none;color:#C6FF00;font:600 12px 'Geist Mono',ui-monospace,Menlo,monospace;height:auto;padding:4px}
.rehide{display:flex;align-items:center;height:26px;padding:0 10px;border:none;background:rgba(10,10,11,.85);color:#C6FF00;font:600 10.5px 'Geist Mono',ui-monospace,Menlo,monospace;letter-spacing:.06em}
@keyframes sh{0%{background-position:150% 0}100%{background-position:-50% 0}}
@keyframes pulse{0%,100%{opacity:1}50%{opacity:.25}}
@media (prefers-reduced-motion:reduce){.shimmer,.dot{animation:none}}
`

export function installPageStyles() {
  if (document.getElementById('fa-page-styles')) return
  const style = document.createElement('style')
  style.id = 'fa-page-styles'
  style.textContent = PAGE_CSS
  ;(document.head ?? document.documentElement).append(style)
}

function svg(path: string, size: number, stroke = 2.25): string {
  return `<svg width="${size}" height="${size}" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="${stroke}" stroke-linecap="round" stroke-linejoin="round"><path d="${path}"/></svg>`
}

function escape(text: string): string {
  const div = document.createElement('div')
  div.textContent = text
  return div.innerHTML
}

function overlayFor(el: HTMLElement): HTMLElement {
  let overlay = el.querySelector<HTMLElement>(':scope > fa-overlay')
  if (!overlay) {
    overlay = document.createElement('fa-overlay')
    overlay.attachShadow({ mode: 'open' })
    el.append(overlay)
  }
  if (getComputedStyle(el).position === 'static') el.setAttribute('data-fa-relative', '')
  return overlay
}

function render(overlay: HTMLElement, html: string, wire: (root: ShadowRoot) => void) {
  const root = overlay.shadowRoot
  if (!root) return
  root.innerHTML = `<style>${SHADOW_CSS}</style>${html}`
  root.querySelectorAll('button').forEach((b) =>
    b.addEventListener('click', (e) => {
      e.preventDefault()
      e.stopPropagation()
    }),
  )
  wire(root)
}

function on(root: ShadowRoot, selector: string, action: () => void) {
  root.querySelector(selector)?.addEventListener('click', action)
}

export function setState(
  el: HTMLElement,
  state: OverlayState,
  options: { kind?: BlockKind; detail?: string; layout?: Layout; actions?: OverlayActions } = {},
) {
  el.setAttribute(STATE_ATTR, state)
  if (options.layout === 'row') el.setAttribute('data-fa-layout', 'row')
  else el.removeAttribute('data-fa-layout')
  if (state === 'hidden') {
    el.querySelector(':scope > fa-overlay')?.remove()
    return
  }
  const overlay = overlayFor(el)
  overlay.dataset.state = state
  const actions = options.actions
  const layout = options.layout ?? 'card'

  if (state === 'pending') {
    render(
      overlay,
      `<div class="wrap pending"><div class="shimmer"></div><div class="checking"><span class="dot"></span>checking…</div></div>`,
      () => undefined,
    )
    return
  }
  if (state === 'shown') {
    render(
      overlay,
      `<button class="rehide">RE-HIDE</button>`,
      (root) => actions && on(root, '.rehide', actions.rehide),
    )
    return
  }
  if (state === 'dismissed') {
    render(
      overlay,
      `<div class="wrap dismissed"><strong>Got it. Fewer like this.</strong><button class="undo">UNDO</button></div>`,
      (root) => actions && on(root, '.undo', actions.rehide),
    )
    return
  }
  const kind = options.kind ?? 'keyword'
  const label = `${BADGE_LABEL[kind]}: ${options.detail ?? ''}`
  render(
    overlay,
    `<div class="wrap ${layout === 'row' ? 'row' : ''}"><span class="badge" title="${escape(label)}">${svg(BADGE_ICON[kind], layout === 'row' ? 12 : 13)}${escape(label)}</span><div class="buttons"><button class="ghost show">Show anyway</button><button class="solid dismiss">Not interested</button></div></div>`,
    (root) => {
      if (!actions) return
      on(root, '.show', actions.show)
      on(root, '.dismiss', actions.dismiss)
    },
  )
}

export function clearState(el: HTMLElement) {
  el.removeAttribute(STATE_ATTR)
  el.removeAttribute('data-fa-layout')
  el.removeAttribute('data-fa-relative')
  el.querySelector(':scope > fa-overlay')?.remove()
}

export function clearAll() {
  document.querySelectorAll<HTMLElement>(`[${STATE_ATTR}]`).forEach(clearState)
}

export function toast(kind: BlockKind, detail: string) {
  document.getElementById('fa-toast')?.remove()
  const host = document.createElement('div')
  host.id = 'fa-toast'
  host.style.cssText =
    'position:fixed;top:72px;left:50%;transform:translateX(-50%);z-index:2147483646;pointer-events:none'
  const root = host.attachShadow({ mode: 'open' })
  root.innerHTML = `<style>${SHADOW_CSS}.t{display:flex;align-items:center;gap:8px;background:#0A0A0B;border:1px solid #2E2E32;border-radius:999px;padding:6px 8px 6px 12px;font:500 12px 'Geist',system-ui,sans-serif;color:#F5F5F0;box-shadow:0 12px 32px rgba(0,0,0,.4)}</style><div class="t">Skipped<span class="badge">${svg(BADGE_ICON[kind], 12)}${escape(`${BADGE_LABEL[kind]}: ${detail}`)}</span></div>`
  document.body.append(host)
  setTimeout(() => host.remove(), 2200)
}
