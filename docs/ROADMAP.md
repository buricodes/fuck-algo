# Roadmap

Build in phases. Each phase ends with something that runs and can be checked.

## Phase 0: Scaffold (done)

* [x] Monorepo with `extension/`, `server/`, `docs/`
* [x] Extension on Vite, React 19, TypeScript strict, crxjs, Vitest, ESLint
* [x] Go server on Gin and pgx with embedded migrations
* [x] API contract in docs/API.md
* [x] Neon project linked (`neon.ts`)

Acceptance: both sides build, lint and test cleanly.

## Phase 1: Local blocking (done in code, needs live checks)

* [x] Rule engine: keywords, watchlist strictness, repeat limits
* [x] Site adapters: YouTube tiles and Shorts, X, Instagram posts and Reels, Reddit and generic articles
* [x] Overlay: blurred, checking, shown, dismissed, hidden
* [x] Shorts auto-scroll after one full watch
* [ ] Live check on YouTube home, search, sidebar and Shorts
* [ ] Live check on X For you, Instagram feed and Reels, Reddit

Acceptance: with the server off, a keyword and a Balanced title are blocked on every supported site, and Show anyway, Not interested and Undo work.

## Phase 2: Popup and settings (done in code)

* [x] Popup in its three states
* [x] Settings: watchlist, repeat blocker, keyword blocks, activity, dark and light
* [x] Empty states
* [ ] Visual review against the design at 100% zoom

Acceptance: every control in the design changes real settings and the content script reacts without a reload.

## Phase 3: Server and AI

* [x] Installs, trial counters, licence activation
* [x] AI checks with batching and caching
* [x] Voice token, transcribe fallback and voice rules
* [ ] Neon `DATABASE_URL` in place and migrations applied
* [ ] Prompt evaluation set: 50 titles per Paranoid case, 30 voice transcripts

Acceptance: a Paranoid title catches an adjacent post that has none of its words; voice turns the three design examples into the expected rules.

## Phase 4: Payments and website

* [x] Dodo one time checkout and webhook
* [x] Website with how it works, pricing, thanks page
* [ ] Dodo product, webhook secret and return URL set in production
* [ ] Website live on Vercel, API on a public domain

Acceptance: a test mode payment shows a key on the thanks page and unlocks the paying browser.

## Phase 5: Launch

* [x] Real brand assets (hooded eye mark, cyber mascot hero)
* [ ] Custom domain (website on Vercel's default domain for now)
* [ ] Chrome Web Store listing, screenshots and privacy answers from docs/SECURITY.md
* [ ] Error reporting and basic server metrics

## Later

* Rule sync between a person's browsers using their licence key
* Firefox and Safari
* Per site action overrides
