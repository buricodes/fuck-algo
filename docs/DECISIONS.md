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

## 3. Clerk accounts, signed in on the website

Status: accepted (replaces "no accounts: install tokens and licence keys")

Reason: The same structure as Bui. People sign in on the website and the extension shares that session through Clerk's sync host, so there is one sign in and nothing to paste. A purchase belongs to the account, so it unlocks every browser the person signs in on, and a lost device loses nothing.

Alternatives considered: Install tokens and pasted licence keys. No sign up, but keys get lost or shared and trials are trivially reset.

Trade off: AI features need an account. Local blocking still works signed out.

## 4. Go server, reusing server-bui

Status: accepted

Reason: The same stack as server-bui (Gin, pgx, slog, Dodo, ElevenLabs, Anthropic), so proven code is reused and one person can run both.

## 5. Jev for checks, Claude Haiku 5.5 for voice rules

Status: accepted

Reason: Checks only ever pick from a list: which watched title (or none) and which topic. Jev (`typesafe/jev-1.13` on OpenRouter, as in Bui) answers typed choice questions with probabilities, costs about $0.00002 a call and keeps topic labels stable because it picks from a fixed list plus the person's own topics. Voice rules need generated names, keywords and dates, so they use Claude Haiku 5.5 with structured output.

During development the voice rules run through the local Claude CLI (`claude -p`, no tools, no session), which needs no API credits. Production switches `AI_RULES=anthropic` with an API key. Checks can also fall back to `anthropic` or `cli`.

Trade off: The CLI is slow (about 6 seconds) and serial, fine for development, not for real traffic.

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
