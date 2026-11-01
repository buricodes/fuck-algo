# Fuck Algo server

The Go backend for Fuck Algo, the browser extension that keeps spoilers, repeat content and unwanted keywords out of your feeds. It hands out install tokens, tracks the free trial, sells the one time $5 licence through Dodo Payments, mints ElevenLabs speech to text tokens, and runs the Claude powered voice rules and AI checks.

There are no accounts. An install gets a bearer token; a licence key unlocks up to five installs.

The contract with the extension and the website lives in [`../docs/API.md`](../docs/API.md). Change that first, then both sides.

## Running locally

```sh
cp .env.example .env   # every key is optional in development
make run               # listens on :8787
make test              # unit and handler tests, no network and no database
make lint              # gofmt check and go vet
make build             # static binary in bin/server
```

With `APP_ENV=development` and an empty `DATABASE_URL` the server uses an in memory store and logs a warning; everything is lost on restart. Outside development a missing `DATABASE_URL` stops start up.

In development the server reads `.env`, then `../.env.local`, then `../.env` (the repo root, where the Neon project keeps `DATABASE_URL`). A variable that is already set always wins, so the first file to define it takes effect.

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
| `ANTHROPIC_API_KEY` | Claude. Without it `/v1/checks` and `/v1/voice/rules` answer `unavailable`. |
| `ELEVENLABS_API_KEY` | ElevenLabs Scribe. Without it `/v1/voice/token` and `/v1/voice/transcribe` answer `unavailable`. |
| `DODO_PAYMENTS_API_KEY`, `DODO_PAYMENTS_WEBHOOK_SECRET`, `DODO_PRODUCT_ID` | Dodo Payments. All three are needed, otherwise `/v1/checkout` and `/v1/webhooks/dodo` answer `unavailable`. |
| `WEBSITE_URL` | The website origin for CORS and the Dodo return URL `<WEBSITE_URL>/thanks?session=<id>`. Set it in production. |
| `EXTENSION_ORIGINS` | Comma separated `chrome-extension://<id>` origins allowed by CORS. |
| `CONFIG_PATH` | Optional path to the yaml config, default `config/config.yaml`. |

Everything that is not a secret (models, trial limits, body limits, rate limits, cache size, installs per licence) is in [`config/config.yaml`](config/config.yaml).

## Layout

```
cmd/server/            start up: env files, config, store, services, graceful shutdown
config/config.yaml     models, trial limits, body and rate limits, licence settings
internal/config/       yaml and environment loading, .env reader
internal/domain/       the shared types from API.md
internal/http/         router, one file of handlers per area, auth
internal/http/middleware/  CORS, body limit, rate limit, request log, panic recovery
internal/store/        Store interface, Postgres and memory implementations, migrations
internal/service/
  anthropic/           lean Messages API client with structured JSON output
  elevenlabs/          realtime token and file transcription
  payments/            Dodo checkout sessions and webhook parsing
  licence/             licence keys, install tokens, ids and hashing
  checks/              batching, Paranoid watchlist filtering, 24 hour cache
  voice/               voice rules prompt call and rule clean up
internal/prompts/      the system prompts as plain text files
pkg/                   error codes, response envelope, outbound HTTP, Standard Webhooks
```

## How the important parts work

**Install tokens** are `fa_inst_` followed by 32 random bytes in base64url. Only their SHA-256 is stored.

**Licence keys** look like `FA-XXXX-XXXX-XXXX-XXXX`, drawn with `crypto/rand` from an alphabet without 0, O, 1 or I. Activation accepts any case, spacing and dashes, with or without the `FA` prefix. Only the SHA-256 of the canonical key and its last four characters are stored. The full key is kept on the checkout session for 24 hours so the thanks page can show it, then cleared by an hourly sweep (and never returned after that window regardless).

**Trial spending** is a single atomic `UPDATE` that only succeeds while the counter is under the limit and returns how much it granted, so parallel requests cannot overspend. If the upstream call then fails, the spend is refunded. An install with an active licence never touches the counters.

**AI checks** send every uncached item in one Claude Haiku call with a JSON schema. Only Paranoid watchlist titles go to the model; Relaxed and Balanced are left to the extension. The model is told the existing topic labels so it reuses them. Results are cached in memory for 24 hours keyed by item id and a hash of the Paranoid watchlist, so only uncached items cost trial checks.

**Voice rules** go to Claude Sonnet with a prompt that maps casual speech to spoiler, repeat and keyword rules, resolves expiries such as "until Friday" against the client's `now`, and is told never to invent rules. The server then cleans the result: unknown kinds, bad values and past expiries are dropped and the list is capped at eight.

**Payments** are one time. `POST /v1/checkout` creates our own `cks_` session, then a Dodo checkout carrying `install_id` and `session_id` in its metadata. The `payment.succeeded` webhook is verified with Standard Webhooks signatures and handled in one transaction that records the webhook id, creates the licence (unique on the Dodo payment id), marks the session paid and activates the install. Replays are harmless.

**Thinking** is switched off for both models. Claude Haiku 5.5 takes `thinking: {type: "disabled"}`; Claude Sonnet 5.5 rejects that and takes `between_tools` instead, which is what `config.yaml` sets.

## Deploying

```sh
docker build -t fuck-algo-server .
docker run -p 8787:8787 --env-file .env.production fuck-algo-server
```

The image runs as a non root user, defaults to `APP_ENV=production` and serves `GET /healthz` for health checks. If it sits behind a proxy, list the proxy addresses in `server.trusted_proxies` so per IP rate limits see real client addresses.
