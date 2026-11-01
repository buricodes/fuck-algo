# Instructions for Claude Code

## What this project is

Fuck Algo is a paid ($5 once) Chrome extension that filters social feeds: a spoiler watchlist, a repeat blocker, keyword blocks and voice rules. This monorepo holds the extension (`extension/`) and its Go API (`server/`). The website is a separate repo, `fuck-algo-website`.

Read these before writing code, in this order: README.md, docs/ARCHITECTURE.md, docs/API.md, docs/DESIGN.md, docs/SECURITY.md, docs/DECISIONS.md, docs/ROADMAP.md.

## How to work

1. docs/API.md is the contract. Change it first, then the server, then the extension, in one change.
2. If a doc and the code disagree, stop and ask, then update the doc.
3. Do not change a decision in docs/DECISIONS.md without asking. If one proves wrong, explain the evidence and propose a new entry.
4. Keep changes small and commit often with clear messages.

## Hard rules

* No comments in code. Names and small functions carry the meaning; docs carry the why.
* Local rules (keywords, Relaxed and Balanced watchlist titles, channel based repeats) must work with no network and no licence.
* Only item titles, channel names and short text snippets leave the browser, and only for AI checks. Never send URLs of the page, cookies, history or anything typed into a site.
* Voice audio goes to the speech to text provider only while the person is talking. It is never stored by us.
* Fail open: if the server, AI or a site adapter breaks, the page shows as normal. Never leave content stuck blurred or pending.
* The background service worker is the only writer of activity, stats and feed memory.
* Secrets live in environment variables only, never in the repo or the extension bundle. The Clerk publishable key is the only key the extension and website may hold.

## Technical conventions

* Extension: TypeScript strict, React 19, plain CSS with tokens from `src/ui/styles/tokens.css`, no UI library. Path alias `@/` is `src/`.
* Engine code in `extension/src/engine` is pure and unit tested with Vitest.
* Server: Go 1.26, Gin, pgx v5, slog. Handlers stay thin; logic lives in `internal/service`.
* British English in docs and user facing text.

## Commands

```
cd extension && pnpm dev | pnpm build | pnpm test | pnpm lint | pnpm typecheck
cd server && make run | make test | make lint | make build
```

## Definition of done

* Code, tests and docs updated together.
* Lint, type checks and tests pass on both sides.
* The extension loads unpacked with no console errors on YouTube, X and Instagram.
* No new data leaves the browser without a line in docs/SECURITY.md.

## Things to ask the human about

* A custom domain (the website runs on https://fuck-algo-website.vercel.app for now) and where to host the API.
* Dodo product id and webhook secret before the first paid release.
