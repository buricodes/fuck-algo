# Fuck Algo server

The Go backend for Fuck Algo, the browser extension that keeps spoilers, repeat content and unwanted keywords out of your feeds. It verifies Clerk sign ins, tracks each user's free trial, sells the one time $5 upgrade through Dodo Payments, mints ElevenLabs speech to text tokens, and runs the Claude powered voice rules and AI checks.

Accounts are Clerk users. People sign in on the website and the extension shares that session through Clerk's sync host. Every `/v1` route except the Dodo webhook takes `Authorization: Bearer <Clerk session token>`. Paying marks the user active, on every browser they sign in to.

The contract with the extension and the website lives in [`../docs/API.md`](../docs/API.md). Change that first, then both sides.

## Running locally

```sh
cp .env.example .env   # every key is optional in development; set CLERK_SECRET_KEY to sign in
make run               # listens on :8787
make test              # unit and handler tests, no network and no database
make lint              # gofmt check and go vet
make build             # static binary in bin/server
```

With `APP_ENV=development` and an empty `DATABASE_URL` the server uses an in memory store and logs a warning; everything is lost on restart. Outside development a missing `DATABASE_URL` stops start up.

In development the server reads `.env`, then `../.env.local`, then `../.env` (the repo root, where the Neon project keeps `DATABASE_URL`). A variable that is already set always wins, so the first file to define it takes effect.

Without `CLERK_SECRET_KEY` the server starts with a warning and every signed in route answers `unavailable`; only `/healthz` and the webhook work. To try the signed in routes locally, put the development instance's `sk_test_...` key in `.env`, sign in on the website (or the extension) and copy a fresh session token (`await window.Clerk.session.getToken()` in the browser console, valid for about a minute):

```sh
curl -s localhost:8787/v1/me -H "Authorization: Bearer $TOKEN"
```

Migrations in `internal/store/migrations` are embedded and applied on start up, recorded in `schema_migrations`, under an advisory lock so several instances can start together.

To run the store contract tests against a real database as well as the memory store:

```sh
TEST_DATABASE_URL=postgres://user@localhost:5432/fa_test?sslmode=disable go test ./internal/store/
```

The tests truncate every table in that database, so point them at a throwaway one.

## Environment

| Variable | Purpose |
| --- | --- |
| `APP_ENV` | `development` (memory store allowed, Dodo test mode, any `chrome-extension://` origin allowed when `EXTENSION_ORIGINS` is empty) or anything else for production (Dodo live mode). |
| `PORT` | Listen port, default 8787. |
| `DATABASE_URL` | Postgres connection string. Required outside development. |
| `CLERK_SECRET_KEY` | Clerk secret key, used to fetch the signing keys (JWKS) and read a new user's email. Required outside development; empty in development makes every signed in route answer `unavailable`. |
| `CLERK_AUTHORIZED_PARTIES` | Comma separated origins accepted in the token's `azp` claim, for example `https://fuck-algo-website.vercel.app,chrome-extension://<id>`. Required outside development; empty in development skips the check. |
| `AI_CHECKS`, `AI_RULES` | Providers for AI checks (`jev`, `anthropic` or `cli`) and voice rules (`cli` or `anthropic`). |
| `OPEN_ROUTER_KEY` | Jev through OpenRouter, for `AI_CHECKS=jev`. |
| `ANTHROPIC_API_KEY` | Claude API, for the `anthropic` providers. |
| `ELEVENLABS_API_KEY` | ElevenLabs Scribe. Without it `/v1/voice/token` and `/v1/voice/transcribe` answer `unavailable`. |
| `DODO_PAYMENTS_API_KEY`, `DODO_PAYMENTS_WEBHOOK_SECRET`, `DODO_PRODUCT_ID` | Dodo Payments. All three are needed, otherwise `/v1/checkout` and `/v1/webhooks/dodo` answer `unavailable`. |
| `WEBSITE_URL` | The website origin for CORS and the Dodo return URL `<WEBSITE_URL>/thanks?session=<id>`. Set it in production. |
| `EXTENSION_ORIGINS` | Comma separated `chrome-extension://<id>` origins allowed by CORS. |
| `CONFIG_PATH` | Optional path to the yaml config, default `config/config.yaml`. |

Everything that is not a secret (models, trial limits, body limits, rate limits, cache size) is in [`config/config.yaml`](config/config.yaml).

## Layout

```
cmd/server/            start up: env files, config, store, services, graceful shutdown
config/config.yaml     models, trial limits, body and rate limits
internal/config/       yaml and environment loading, .env reader
internal/domain/       the shared types from API.md
internal/http/         router, one file of handlers per area, Clerk auth and user creation
internal/http/middleware/  CORS, body limit, rate limit, request log, panic recovery
internal/store/        Store interface, Postgres and memory implementations, migrations
internal/service/
  anthropic/           lean Messages API client with structured JSON output
  jev/                 Jev typed choice questions through OpenRouter
  claudecli/           the local `claude` CLI as a model, for development
  elevenlabs/          realtime token and file transcription
  payments/            Dodo checkout sessions and webhook parsing
  clerkauth/           Clerk session token verification (cached JWKS, azp check) and email lookup
  checks/              batching, Paranoid watchlist filtering, 24 hour cache
  voice/               voice rules prompt call and rule clean up
internal/prompts/      the system prompts as plain text files
pkg/                   error codes, response envelope, outbound HTTP, Standard Webhooks
```

## How the important parts work

**Sign in.** The server reads the `kid` from the session token, fetches the matching key from the Clerk instance's JWKS (cached for an hour), verifies the signature, expiry and issuer, and checks `azp` against `CLERK_AUTHORIZED_PARTIES`. The token's `sub` is the Clerk user id. The first time a user id is seen the server creates the user row and reads their primary email once from the Clerk Backend API; if that lookup fails the user is still created with an empty email and the lookup is retried on a later `GET /v1/me` or checkout. Verification and the email lookup sit behind small interfaces, so the handler tests use fakes and never reach Clerk.

**Trial spending** is a single atomic `UPDATE` that only succeeds while the counter is under the limit and returns how much it granted, so parallel requests cannot overspend. If the upstream call then fails, the spend is refunded. The counters live on the user, so they follow the person across browsers. An active user never touches them.

**AI checks** (`AI_CHECKS=jev`, the default) ask Jev through OpenRouter two typed choice questions per item, in parallel: which Paranoid title it is plausibly about (or none, counted only at 50% or more), and which topic it is, picked from the person's own topics plus a fixed list in `internal/service/checks/topics.go`. Only Paranoid watchlist titles are judged; Relaxed and Balanced are left to the extension. `AI_CHECKS=anthropic` or `cli` sends the batch to Claude Haiku 5.5 with a JSON schema instead. Results are cached in memory for 24 hours keyed by item id and a hash of the Paranoid watchlist, so only uncached items cost trial checks.

**Voice rules** go to Claude Haiku 5.5 with a prompt that maps casual speech to spoiler, repeat and keyword rules, resolves expiries such as "until Friday" against the client's `now`, and is told never to invent rules. The server then cleans the result: unknown kinds, bad values and past expiries are dropped and the list is capped at eight.

**Payments** are one time. `POST /v1/checkout` refuses users who are already active, creates our own `cks_` session for the signed in user, then a Dodo checkout with their email prefilled and `user_id` (the Clerk user id) and `session_id` in its metadata. `GET /v1/checkout/:sessionId` only answers the user who started it. The `payment.succeeded` webhook is verified with Standard Webhooks signatures and handled in one transaction that records the webhook id, records the payment (unique on the Dodo payment id), marks the session paid and sets the user's `licensed_at`. The payer is the session's owner, or the metadata `user_id` when the session is unknown; a payment that matches nobody is still recorded and logged. Replays are harmless.

**Claude CLI in development.** `AI_RULES=cli` (the default) runs `claude -p --model claude-haiku-5-5` with no tools, no session and `--json-schema`, using the local Claude login (the server strips `ANTHROPIC_API_KEY` from its environment so API credits are not used). Set `AI_RULES=anthropic` with `ANTHROPIC_API_KEY` for the API. Thinking is switched off on the API path.

## Deploying

```sh
docker build -t fuck-algo-server .
docker run -p 8787:8787 --env-file .env.production fuck-algo-server
```

The image runs as a non root user, defaults to `APP_ENV=production` and serves `GET /healthz` for health checks. If it sits behind a proxy, list the proxy addresses in `server.trusted_proxies` so request logs and `ClientIP` see real client addresses. Rate limits on the signed in routes are per user.
