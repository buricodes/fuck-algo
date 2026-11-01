# API contract

The contract between the extension (`extension/`), the website (separate repo `fuck-algo-website`) and the Go server (`server/`). Change this file first, then both sides.

## Basics

* Base URL: set per environment. Local `http://localhost:8787`, production is the deployed server URL (no custom domain yet; the website runs on `https://fuck-algo-website.vercel.app`).
* All bodies are JSON (`Content-Type: application/json`) unless stated.
* Every response uses one envelope:

```json
{ "ok": true, "data": { } }
{ "ok": false, "error": { "code": "trial_exhausted", "message": "Your free voice rules are used up." } }
```

* Error codes (stable, the extension switches on them): `bad_request`, `unauthorised`, `not_found`, `trial_exhausted`, `licence_invalid`, `licence_full`, `rate_limited`, `upstream`, `unavailable`, `internal`.
* Authenticated routes take `Authorization: Bearer <installToken>`.
* Timestamps are RFC 3339 strings in UTC.

## Shared types

```ts
type Licence = {
  status: 'trial' | 'active'
  keyHint: string | null        // last 4 characters of the key, e.g. "Q7KD"
  trial: { voiceLeft: number; checksLeft: number }  // ignored when status is active
}

type SpoilerType = 'anime' | 'game' | 'show' | 'movie' | 'sports'
type Strictness = 0 | 1 | 2      // 0 Relaxed, 1 Balanced, 2 Paranoid
type KeywordAction = 'hide' | 'blur' | 'skip'
type Site = 'youtube' | 'x' | 'instagram' | 'web'

type WatchContext = { name: string; type: SpoilerType; level: Strictness; keywords: string[] }

type ProposedRule =
  | { kind: 'spoiler'; name: string; type: SpoilerType; level: Strictness; keywords: string[] }
  | { kind: 'repeat'; topic: string; sites: Site[] }
  | { kind: 'keyword'; word: string; action: KeywordAction; expiresAt: string | null }
```

## Routes

### `GET /healthz`

No auth. `200 { ok: true, data: { status: "ok", db: "ok" | "down" } }`.

### `POST /v1/installs`

No auth. Called once when the extension is installed. Rate limited per IP.

Request: `{ "version": "1.0.0", "browser": "chrome" }`

Response: `{ "installId": "uuid", "token": "fa_inst_...", "licence": Licence }`

The token is shown only here. The server stores a SHA-256 hash.

### `GET /v1/me`

Auth. Response: `{ "installId": "uuid", "licence": Licence }`. Also bumps `last_seen_at`.

### `POST /v1/licence/activate`

Auth. Request: `{ "key": "FA-XXXX-XXXX-XXXX-XXXX" }` (case and spacing insensitive).

Response: `{ "licence": Licence }`. Errors: `licence_invalid`, `licence_full` (more than 5 installs on one key).

### `POST /v1/voice/token`

Auth. Returns a single use ElevenLabs realtime speech to text token. Spends one voice use while on trial.

Response: `{ "token": "..." }`. Errors: `trial_exhausted`, `unavailable` (no ElevenLabs key configured).

### `POST /v1/voice/transcribe`

Auth. Fallback when the realtime socket fails. `multipart/form-data` with one file field `audio` (webm or ogg, at most 5 MB). Does not spend a voice use (the token call already did, or the client calls this only after a failed token session).

Response: `{ "transcript": "..." }`

### `POST /v1/voice/rules`

Auth. Turns what the person said into rules they then confirm. Does not spend a voice use.

Request:

```json
{
  "transcript": "No GTA 6 spoilers, way fewer gym bro videos, blur anything about crypto.",
  "now": "2026-11-01T10:00:00Z",
  "context": { "watchlist": ["Black Clover"], "keywords": ["MrBeast"], "topics": ["Gym bro advice"] }
}
```

Response:

```json
{
  "rules": [
    { "kind": "spoiler", "name": "GTA 6", "type": "game", "level": 2, "keywords": ["leak", "trailer", "ending"] },
    { "kind": "repeat", "topic": "Gym bro advice", "sites": ["youtube", "instagram"] },
    { "kind": "keyword", "word": "crypto", "action": "blur", "expiresAt": null }
  ],
  "summary": "Three rules: one spoiler title, one topic, one keyword."
}
```

Transcript limit 2,000 characters. At most 8 rules. An empty `rules` array is valid (nothing actionable heard).

### `POST /v1/checks`

Auth. AI checks for items the local rules cannot decide. Spends one check per item while on trial.

Request:

```json
{
  "items": [
    { "id": "yt:dQw4w9WgXcQ", "site": "youtube", "title": "Ch. 371 just changed everything", "channel": "MangaTalk", "text": "" }
  ],
  "watchlist": [ { "name": "Black Clover", "type": "anime", "level": 2, "keywords": ["Asta"] } ],
  "topics": ["Gym bro advice", "Sigma / grindset edits"]
}
```

Response:

```json
{ "results": [ { "id": "yt:dQw4w9WgXcQ", "spoilerOf": "Black Clover", "topic": "Manga reactions" } ] }
```

* At most 25 items per call, title at most 300 characters, text at most 600.
* `spoilerOf` is the watchlist name the item spoils, or `null`. Only Paranoid (level 2) titles are judged by the model; Relaxed and Balanced are decided locally.
* `topic` is a short, reusable label (2 to 4 words, Title case) used by the repeat blocker. The server is told the existing `topics` and reuses one when it fits, so labels stay stable.
* The server caches results per item id and watchlist for 24 hours.

### `POST /v1/checkout`

No auth (the website and the extension both call it). Request: `{ "installId": "uuid" | null, "email": "optional" }`.

Response: `{ "sessionId": "cks_...", "checkoutUrl": "https://checkout.dodopayments.com/..." }`.

The Dodo return URL is `<WEBSITE_URL>/thanks?session=<sessionId>`.

### `GET /v1/checkout/:sessionId`

No auth. Response: `{ "status": "pending" | "paid", "licenceKey": "FA-..." | null }`.

The full key is returned while the session is less than 24 hours old, so the thanks page can show it. After that only `status` comes back. If the checkout carried an `installId`, that install is already activated.

### `POST /v1/webhooks/dodo`

Dodo webhook (Standard Webhooks signature in `webhook-id`, `webhook-timestamp`, `webhook-signature`). Handles `payment.succeeded`: creates the licence, stores the key hash, attaches it to the checkout session and to the install if one was given. Idempotent on `webhook-id`.

## Trial limits

Configured in `server/config/config.yaml`. Defaults: 5 voice uses and 300 AI checks per install. Local blocking (keywords, watchlist at Relaxed and Balanced, repeat limits by channel) never needs the server and never expires.
