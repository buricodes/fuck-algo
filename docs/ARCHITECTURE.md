# Architecture

## Goals

* Filter feeds on YouTube, X, Instagram and other sites before the person sees what they did not want.
* Decide most things on the device, instantly and for free. Use AI only where words alone cannot decide.
* Let people set rules by talking, then confirm them.
* No accounts. One $5 licence key works on up to five browsers.

## Non goals (version 1)

* Mobile apps or Safari.
* Syncing rules between browsers (each browser keeps its own rules).
* Blocking ads or tracking.

## System

```
 ┌──────────────── Chrome ────────────────┐            ┌──────── server (Go) ────────┐
 │ content script  (every http/https tab)  │            │  /v1/installs  /v1/me        │
 │   site adapter → engine → overlay       │            │  /v1/checks   → Claude Haiku │
 │        │ checks, activity, seen          │   HTTPS    │  /v1/voice/*  → ElevenLabs,  │
 │        ▼                                 │ ─────────▶ │                 Claude Sonnet│
 │ background service worker               │  Bearer    │  /v1/checkout → Dodo         │
 │   api client · checks batcher · ledger  │  install   │  /v1/webhooks/dodo           │
 │        ▲ voice events                    │  token     │        │                     │
 │ offscreen document (microphone)          │            │        ▼                     │
 │ popup · settings page (React)            │            │  Postgres on Neon            │
 └─────────────────────────────────────────┘            └──────────────────────────────┘
```

## Extension (`extension/src`)

| Folder | Role |
| --- | --- |
| `shared/` | Types, defaults, typed `chrome.storage` access, typed runtime messages, icons, small helpers. |
| `engine/` | Pure rule logic: text matching, local verdicts, repeat limits, folding voice rules into settings. Unit tested. |
| `content/` | Runs on every page. A site adapter finds feed items, the engine decides, the overlay blurs, hides or skips. |
| `content/sites/` | One adapter per site: `youtube` (tiles and the Shorts player), `x`, `instagram` (posts and Reels), `web` (Reddit and generic articles). |
| `background/` | Service worker. Owns the install token, calls the API, batches AI checks with a cache, and is the single writer of activity, stats and feed memory. |
| `offscreen/` | Holds the microphone, streams audio to ElevenLabs realtime, falls back to upload and transcribe. |
| `popup/` | Toolbar popup (360 × 560). |
| `options/` | Full settings page: watchlist, repeat blocker, keyword blocks, activity, licence. |
| `ui/` | Tokens, base styles, shared components, the voice panel and hooks. |

### Deciding an item

1. The adapter reads `title`, `channel` and a short `text` and gives the item a stable key (`yt:<id>`, `x:<id>`, `ig:<id>`, `rd:<id>`).
2. `localVerdict` checks keywords, then watchlist titles (Relaxed: title only; Balanced: title, channel and text, plus extra keywords).
3. If nothing matched and either a Paranoid title exists or the repeat blocker is on, the item goes to the background for an AI check. While a Paranoid check is in flight the item shows the "checking…" state.
4. The check returns `spoilerOf` and a `topic`. A spoiler blurs. Otherwise the topic is compared with the last 20 items actually seen: a blocked topic, or one already seen `limit` times, is skipped.
5. Items that pass and are at least 60% visible are recorded as seen with their topic.
6. Without AI (no network, trial used up) the channel stands in for the topic, so channel repeats are still capped.

### Actions

| Action | Feed tile or post | Shorts or Reels player |
| --- | --- | --- |
| blur | Blurred, badge, Show anyway, Not interested | Skipped |
| hide | Removed from layout | Skipped |
| skip | Removed from layout | Scrolled to the next one, with a short toast |

Spoilers always blur. Repeats always skip. Keywords use their own action.

### Storage (`chrome.storage.local`)

| Key | Contents | Writer |
| --- | --- | --- |
| `settings` | Shield, paused hosts, feature toggles, sites, watchlist, repeat limit and blocked topics, keywords, theme | UI pages |
| `stats` | Per day counts by kind, 30 days | background |
| `activity` | Last 300 blocked items, with undo state | background (undo from settings) |
| `feed` | Last 20 seen items with topic, topic counts per day for 7 days | background |
| `allow` | Item keys shown anyway, last 500 | content and settings |
| `install`, `licence` | Install id and token, licence status and trial counters | background |
| `checkCache` | AI check results by item key, last 3,000 | background |

### Voice

1. The panel asks for microphone permission. A popup cannot show the prompt, so it opens the settings page with `?voice=1`.
2. The background asks the server for a single use ElevenLabs token (this spends one trial voice use) and starts the offscreen document.
3. The offscreen document streams 16 kHz PCM to ElevenLabs realtime and broadcasts partial transcripts. It stops on "Done talking", 2.5 seconds of silence after speech, or 60 seconds.
4. If the socket fails, the whole recording is uploaded to `/v1/voice/transcribe`.
5. The transcript goes to `/v1/voice/rules`. The person ticks the rules to keep and `applyRules` merges them into settings.

## Server (`server/`)

Go 1.26 with Gin, pgx v5 and slog. See `server/README.md` for the layout.

* `installs`: one row per browser, holding a SHA-256 of its token, trial counters and an optional licence.
* `licences`: one row per paid key, holding only a SHA-256 of the key and its last four characters.
* `checkout_sessions`: links a Dodo checkout to an install and holds the key for 24 hours so the thanks page can show it.
* `webhook_events`: idempotency for Dodo webhooks.
* AI check results are cached in memory for 24 hours, keyed by item and Paranoid watchlist.
* Migrations are embedded and applied on start.

## Website (separate repo)

Vite and React, static, on Vercel. It calls only `/v1/checkout` and `/v1/checkout/:sessionId`.

## Performance budget

* Content script: under 2 ms per scan on a typical YouTube home page, scans debounced to 250 ms.
* AI checks: batched up to 25 items, flushed after 350 ms, cached locally and on the server.
* Extension bundle: under 500 kB of JavaScript, fonts bundled.
