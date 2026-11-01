# API contract

The contract between the extension (`extension/`), the website (separate repo `fuck-algo-website`) and the Go server (`server/`). Change this file first, then every side.

## Basics

* Base URL: set per environment. Local `http://localhost:8787`. Production is the deployed server URL (no custom domain yet; the website runs on `https://fuck-algo-website.vercel.app`).
* All bodies are JSON (`Content-Type: application/json`) unless stated.
* Every response uses one envelope:

```json
{ "ok": true, "data": { } }
{ "ok": false, "error": { "code": "trial_exhausted", "message": "Your free voice rules are used up." } }
```

* Error codes (stable, clients switch on them): `bad_request`, `unauthorised`, `not_found`, `trial_exhausted`, `rate_limited`, `upstream`, `unavailable`, `internal`.

## Authentication

Accounts are Clerk users. People sign in on the website; the extension shares that session through Clerk's sync host (`@clerk/chrome-extension` with `syncHost` set to the website), exactly as in Bui.

* Authenticated routes take `Authorization: Bearer <Clerk session token>`.
* The server verifies the token against the Clerk instance's JWKS, checks expiry and `azp` against the allowed origins (the website and the extension), then finds or creates the user by Clerk user id (`sub`). The email is read once from the Clerk Backend API when the user row is created.
* Signed out, the extension still runs every local rule. Voice and AI checks answer `unauthorised` and the extension asks the person to sign in.

## Shared types

```ts
type Licence = {
  status: 'trial' | 'active'
  trial: { voiceLeft: number; checksLeft: number }
}

type SpoilerType = 'anime' | 'game' | 'show' | 'movie' | 'sports'
type Strictness = 0 | 1 | 2
type KeywordAction = 'hide' | 'blur' | 'skip'
type Site = 'youtube' | 'x' | 'instagram' | 'web'

type WatchContext = { name: string; type: SpoilerType; level: Strictness; keywords: string[] }

type ProposedRule =
  | { kind: 'spoiler'; name: string; type: SpoilerType; level: Strictness; keywords: string[] }
  | { kind: 'repeat'; topic: string; sites: Site[] }
  | { kind: 'keyword'; word: string; action: KeywordAction; expiresAt: string | null }
```

Strictness: 0 Relaxed, 1 Balanced, 2 Paranoid. `trial` is ignored when `status` is `active`.

## Routes

### `GET /healthz`

No auth. `{ status: "ok", db: "ok" | "down" }`.

### `GET /v1/me`

Auth. Response: `{ "userId": "user_...", "email": "a@b.c", "licence": Licence }`. Creates the user on first call.

### `POST /v1/voice/token`

Auth. A single use ElevenLabs realtime speech to text token. Spends one voice use while on trial.

Response: `{ "token": "..." }`. Errors: `trial_exhausted`, `unavailable`.

### `POST /v1/voice/transcribe`

Auth. Fallback when the realtime socket fails. `multipart/form-data` with one file field `audio` (webm or ogg, at most 5 MB). Does not spend a voice use.

Response: `{ "transcript": "..." }`

### `POST /v1/voice/rules`

Auth. Turns what the person said into rules they confirm. Does not spend a voice use.

Request:

```json
{
  "transcript": "No GTA 6 spoilers, way fewer gym bro videos, blur anything about crypto.",
  "now": "2026-11-01T10:00:00Z",
  "context": { "watchlist": ["Black Clover"], "keywords": ["MrBeast"], "topics": ["Gym bro advice"] }
}
```

Response: `{ "rules": ProposedRule[], "summary": "..." }`. Transcript at most 2,000 characters, at most 8 rules, an empty list is valid.

### `POST /v1/checks`

Auth. AI checks for items local rules cannot decide. Spends one check per uncached item while on trial.

Request: `{ "items": [{ "id", "site", "title", "channel", "text" }], "watchlist": WatchContext[], "topics": string[] }`

Response: `{ "results": [{ "id", "spoilerOf": string | null, "topic": string }] }`

* At most 25 items, title at most 300 characters, text at most 600.
* Only Paranoid titles are judged. `topic` is picked from the person's own topics plus a fixed list, so labels stay stable.
* Cached per item and Paranoid watchlist for 24 hours.

### `POST /v1/checkout`

Auth. Starts a $5 one time Dodo checkout for the signed in user, with their email prefilled. Request body: `{}`.

Response: `{ "sessionId": "cks_...", "checkoutUrl": "https://checkout.dodopayments.com/..." }`. Error `bad_request` if the user is already active.

The Dodo return URL is `<WEBSITE_URL>/thanks?session=<sessionId>`.

### `GET /v1/checkout/:sessionId`

Auth, and the session must belong to the caller. Response: `{ "status": "pending" | "paid" }`.

### `POST /v1/webhooks/dodo`

Dodo webhook, Standard Webhooks signature in `webhook-id`, `webhook-timestamp`, `webhook-signature`. Handles `payment.succeeded`: records the payment, marks the checkout session paid and the user active. Idempotent on `webhook-id`.

## Trial limits

In `server/config/config.yaml`. Defaults: 5 voice uses and 300 AI checks per user. Local blocking never needs the server and never expires.
