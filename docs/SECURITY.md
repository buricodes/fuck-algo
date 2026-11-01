# Security and privacy

## Principles

* Keep data in the browser. Rules, activity, stats and history never leave it.
* Send the minimum to the server, only for features the person uses.
* Fail open. A broken check or adapter must never leave a page blurred.
* No accounts, no tracking, no ads, no selling data.

## What leaves the browser

| Data | Sent to | When | Kept |
| --- | --- | --- | --- |
| Extension version and browser name | Our server | First run | With the install row |
| Item title (≤300 chars), channel (≤120), snippet (≤600), site | Our server, then OpenRouter (Jev) | AI checks, only when a Paranoid title exists or the repeat blocker is on | In memory cache for 24 hours, never written to the database |
| Paranoid watchlist titles and their keywords | Our server, then OpenRouter (Jev) | With AI checks | Not stored |
| Microphone audio | ElevenLabs (realtime, with a single use token) or our server then ElevenLabs (fallback upload) | Only while the voice panel is listening | Not stored by us |
| Transcript, names of existing rules | Our server, then Claude Haiku (CLI in development, Anthropic API in production) | Turning speech into rules | Not stored |
| Licence key | Our server | Activation | Only its SHA-256 and last four characters |

Never sent: page URLs, cookies, browsing history, anything typed into a website, account names on the sites.

## Threat model

| Threat | Mitigation |
| --- | --- |
| Install token theft | Random 32 bytes, stored only as SHA-256 on the server. A stolen token can only spend that install's trial or licence. |
| Licence key sharing | Five installs per key. Keys are stored hashed. |
| Trial abuse by re-registering | Per IP rate limit on `/v1/installs`. Low stakes: a trial costs a few cents. |
| Prompt injection through page titles | Titles are data inside a structured request; the model can only answer the JSON schema. Results are validated against the watchlist names the extension sent. |
| A site injecting script into our overlay | Overlay lives in a shadow root, text is inserted with `textContent` or escaped. No `innerHTML` with page data. |
| Forged payment webhooks | Standard Webhooks signature checked with the Dodo secret, five minute timestamp window, idempotent on `webhook-id`. |
| Over large requests | Body limits (1 MB, 6 MB for audio), item and length caps from docs/API.md. |
| Secrets in the bundle | The extension holds no secrets. ElevenLabs is reached with single use tokens minted by the server. |

## Permissions the extension asks for

* `storage`, `unlimitedStorage`: rules, activity and the check cache.
* `tabs`: read the active tab's URL for the popup's site line and pause button.
* `offscreen`: hold the microphone for voice rules.
* `alarms`: tidy expired keyword rules hourly.
* Host access to `http://*/*` and `https://*/*`: feeds can be on any site the person turns on.

## Reporting

Report security issues to buricodess@gmail.com. Please do not open a public issue.
