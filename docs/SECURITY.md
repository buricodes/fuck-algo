# Security and privacy

## Principles

* Keep data in the browser. Rules, activity, stats and history never leave it.
* Send the minimum to the server, only for features the person uses.
* Fail open. A broken check or adapter must never leave a page blurred.
* Accounts only for unlocking AI features (Clerk). No tracking, no ads, no selling data.

## What leaves the browser

| Data | Sent to | When | Kept |
| --- | --- | --- | --- |
| Email and Clerk user id | Clerk, then our server | Signing in on the website | With the user row |
| Item title (≤300 chars), channel (≤120), snippet (≤600), site | Our server, then OpenRouter (Jev) | AI checks, only when a Paranoid title exists or the repeat blocker is on | In memory cache for 24 hours, never written to the database |
| Paranoid watchlist titles and their keywords | Our server, then OpenRouter (Jev) | With AI checks | Not stored |
| Microphone audio | ElevenLabs (realtime, with a single use token) or our server then ElevenLabs (fallback upload) | Only while the voice panel is listening | Not stored by us |
| Transcript, names of existing rules | Our server, then Claude Haiku (CLI in development, Anthropic API in production) | Turning speech into rules | Not stored |

Never sent: page URLs, cookies, browsing history, anything typed into a website, account names on the sites.

## Threat model

| Threat | Mitigation |
| --- | --- |
| Session token theft | Clerk session tokens are short lived (about a minute) and checked for expiry and `azp`. The extension keeps them in memory only. |
| Account sharing | A purchase belongs to one Clerk account. Clerk handles sessions, devices and sign out. |
| Trial abuse with new accounts | Clerk bot protection on sign up, per user rate limits. Low stakes: a trial costs a few cents. |
| Prompt injection through page titles | Titles are data inside a structured request; the model can only answer the JSON schema. Results are validated against the watchlist names the extension sent. |
| A site injecting script into our overlay | Overlay lives in a shadow root, text is inserted with `textContent` or escaped. No `innerHTML` with page data. |
| Forged payment webhooks | Standard Webhooks signature checked with the Dodo secret, five minute timestamp window, idempotent on `webhook-id`. |
| Over large requests | Body limits (1 MB, 6 MB for audio), item and length caps from docs/API.md. |
| Secrets in the bundle | The extension and website hold only the Clerk publishable key, which is public by design. `CLERK_SECRET_KEY`, AI and payment keys live only in the server environment. ElevenLabs is reached with single use tokens minted by the server. |

## Permissions the extension asks for

* `storage`, `unlimitedStorage`: rules, activity and the check cache.
* `tabs`: read the active tab's URL for the popup's site line and pause button.
* `offscreen`: hold the microphone for voice rules.
* `alarms`: tidy expired keyword rules hourly.
* `cookies` and host access to the website and Clerk: read the Clerk session the website created, the same way Bui does (sync host). Only Clerk's own cookies for our website are read.
* Host access to `http://*/*` and `https://*/*`: feeds can be on any site the person turns on.

## Reporting

Report security issues to buricodess@gmail.com. Please do not open a public issue.
