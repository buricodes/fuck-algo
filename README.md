<div align="center">

<img src="extension/public/assets/fa-mark.png" alt="Fuck Algo" width="88" />

# FUCK ALGO

**Your feed. Your rules.**

A browser extension that takes spoilers, repeat slop and words you are done with off YouTube, X, Instagram and any other site, before they reach your screen. Tell it what to block out loud.

</div>

---

## What it does

* **Spoiler watchlist.** Add the shows, games and matches you follow. Each title has a strictness: Relaxed (exact title), Balanced (title plus keywords) or Paranoid (anything adjacent, judged by AI).
* **Repeat blocker.** Caps how many similar videos you see in your last 20 (default 3). Block a whole topic in one tap from "What you've been fed".
* **Keyword blocks.** Any word or phrase, with an action: hide, blur or skip.
* **Voice rules.** Say "no GTA 6 spoilers, way fewer gym bro videos, blur anything about crypto" and confirm the rules it heard.
* **In page overlay.** Blocked items are blurred with a badge, "Show anyway" and "Not interested". Shorts and Reels are skipped automatically.
* **Activity log.** Everything taken off your feed, with undo.

$5 once. No account, no subscription, no ads.

## Repository

```
fuck-algo/
  extension/   Chrome extension (Manifest V3, React 19, TypeScript, Vite)
  server/      Go API: Clerk auth, AI checks, voice rules, payments
  docs/        Architecture, design, decisions, security, roadmap, API contract
  neon.ts      Neon Postgres project config
```

The marketing website lives in its own repository, `fuck-algo-website`, deployed on Vercel.

## Documents

* [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md): how the system is built, part by part
* [docs/API.md](docs/API.md): the contract between extension, website and server
* [docs/DESIGN.md](docs/DESIGN.md): product and visual design, tokens and components
* [docs/DECISIONS.md](docs/DECISIONS.md): why we chose each major approach
* [docs/SECURITY.md](docs/SECURITY.md): privacy, threat model and data handling
* [docs/ROADMAP.md](docs/ROADMAP.md): phases, tasks and acceptance criteria
* [CLAUDE.md](CLAUDE.md): working instructions for Claude Code

## Quick start

Requirements: Node 22, pnpm 10 (via corepack), Go 1.26, Chrome 116 or newer.

```
cd server && cp .env.example .env && make run      # API on http://localhost:8787
cd extension && cp .env.example .env && pnpm install && pnpm dev
```

Then open `chrome://extensions`, turn on Developer mode, choose **Load unpacked** and pick `extension/dist-dev` (or `extension/dist` after `pnpm build`).

The server runs without a database in development (in memory store). Set `DATABASE_URL` to the Neon connection string to persist users and payments.

## Checks

```
cd extension && pnpm lint && pnpm typecheck && pnpm test && pnpm build
cd server && go vet ./... && go test ./...
```

## Licence

Proprietary. All rights reserved.
