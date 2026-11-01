# Decisions

A short record of the main choices and the reasons for them. Add a new entry whenever a major choice changes.

## 1. Browser extension, Chrome first

Status: accepted

Reason: Feeds live in the browser. An extension can read and change the page before the person looks at it. Chrome and Arc share Manifest V3, which covers most of our audience.

Trade off: No mobile apps, where most Shorts and Reels are watched.

Revisit if: Safari or Firefox demand is clear.

## 2. Local rules first, AI only where needed

Status: accepted

Reason: Keywords and exact titles can be matched instantly, offline and for free. AI is used only for Paranoid spoiler checks ("is this adjacent to Black Clover?") and for topic labels for the repeat blocker.

Trade off: Without AI, the repeat blocker falls back to channel based limits.

## 3. No accounts: install tokens and licence keys

Status: accepted

Reason: A $5 one time purchase should not need a sign up. Each browser registers once and gets a random token. Paying creates a key like `FA-XXXX-XXXX-XXXX-XXXX` that unlocks up to five browsers.

Alternatives considered: Clerk accounts as in Bui. More friction and more personal data for little gain.

Trade off: Lost keys are recovered through the Dodo receipt email, not a password reset.

## 4. Go server, reusing server-bui

Status: accepted

Reason: The same stack as server-bui (Gin, pgx, slog, Dodo, ElevenLabs, Anthropic), so proven code is reused and one person can run both.

## 5. Claude Haiku for checks, Claude Sonnet for voice rules

Status: accepted

Reason: Checks are frequent, short and must be cheap. Voice rules are rare and must understand casual speech, dates ("until Friday") and intent. Both use structured JSON output, so replies are validated, never parsed from prose.

## 6. ElevenLabs realtime speech to text through an offscreen document

Status: accepted

Reason: The design shows a live transcript. Bui already proved realtime Scribe through an offscreen document, which is the only place in Manifest V3 that can hold the microphone while the popup is open or closed.

Trade off: Needs a one time permission grant in a tab, because popups cannot show the prompt.

## 7. Trial: 5 voice uses and 300 AI checks

Status: accepted

Reason: Enough to feel the product, small enough to cost little. Local blocking never expires.

Revisit if: conversion data says otherwise.

## 8. Plain CSS with tokens, no UI library

Status: accepted

Reason: The design is precise and small. Tokens map one to one to the design; a UI library would fight it and add weight to the bundle.

## 9. Postgres on Neon

Status: accepted

Reason: Serverless Postgres with branching, pgx compatible, and the user's chosen provider. The server keeps a memory store for development without a database.

## 10. The website is a separate repository

Status: accepted

Reason: It deploys on Vercel on its own schedule, and has no shared code with the extension beyond the API contract.
