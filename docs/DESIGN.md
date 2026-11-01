# Design

The source of truth is the Claude Design project "Fuck Algo" (`Fuck Algo.dc.html` and the files it imports: `FAPopup`, `FASettings`, `FAVoice`, `FAHowItWorks2`, `FAPricing`). When this document and the design disagree, the design wins.

## Voice and tone

Blunt, short and on your side. "Nobody spoils it for you." "Words you're done with." "The algorithm is driving again." No exclamation marks, no corporate filler, no swearing beyond the name.

## Tokens

| Token | Dark | Light | Use |
| --- | --- | --- | --- |
| `--bg` | #0A0A0B | #F2F1EB | Page background |
| `--s1` | #131315 | #FFFFFF | Cards, inputs |
| `--s2` | #1D1D20 | #E6E5DE | Tiles, toggle tracks, hover |
| `--line` | #2A2A2E | #D5D4CC | Borders and rules |
| `--fg` | #F5F5F0 | #0A0A0B | Text |
| `--mut` | #8E8E88 | #5C5C56 | Secondary text |
| `--acc` | #C6FF00 | #C6FF00 | The one accent: acid lime |
| `--onacc` | #0A0A0B | #0A0A0B | Text on lime |
| `--acct` | #C6FF00 | #4B6600 | Lime used as text (darker on light for contrast) |

One accent only. Red (#FF4D3D) appears once, as the recording dot.

## Type

* Display: Archivo, width 125%, weight 900. Headlines, the ON/OFF word, stat numbers.
* Body: Geist 400 to 700.
* Data and labels: Geist Mono, uppercase with 0.08 to 0.14em tracking.

Fonts are bundled with the extension (`@fontsource-variable`), so nothing loads from Google at runtime.

## Icons

A single set on a 24 grid with a 1.75 stroke, in `extension/src/shared/icons.ts`. Overlay badges use a 2.25 stroke.

## Surfaces

### Toolbar popup (360 × 560)

States: shield on, paused on this site, shield off. The big ON/OFF word and master toggle, the current site with a LIVE, PAUSED, OFF or IDLE pill, today's stats, three feature toggles, "Tell it what to block", "Pause on this site", "Full settings". When the shield is off or paused, stats and toggles dim to 35%.

### Settings (sidebar plus four pages, dark and light)

1. Spoiler watchlist: add a title with a type chip and extra keywords, strictness slider per title, empty state with "Say it instead" and suggestions.
2. Repeat blocker: "max N similar videos in your last 20" slider (1 Ruthless to 10 Chill), "What you've been fed" topic bars with block toggles, "Where it runs" site cards.
3. Keyword blocks: add with HIDE, BLUR or SKIP, chips with the same segmented control, a legend.
4. Activity: filter chips with counts, a timeline grouped by day, Undo and Re-block.

The sidebar holds the shield card, the theme switch and the licence card (trial counters, "Unlock · $5 once", "I have a key").

### Voice panel

Opens over the popup (compact) or over the settings main area (full). Stages: idle (mic, examples, type it), listening (live transcript, waveform, Done talking), thinking, review (tick the rules to apply), done.

### In page overlay

Injected into a shadow root so site styles cannot leak in. Card layout for video tiles and media posts; row layout for text only posts. States: blurred, checking (shimmer), shown (RE-HIDE), dismissed ("Got it. Fewer like this." with UNDO), hidden.

## Components

`Toggle` (sm, lg), `Button` (primary, light, ghost, outline, icon), `Segmented`, `StepSlider`, `Range`, `StatTile`, `Pill`, `Icon`, `Mark`, `PulseDot`, all in `extension/src/ui/components`.

## Accessibility

* Every toggle is a `role="switch"` with `aria-checked` and a label.
* Focus rings use the lime accent.
* `prefers-reduced-motion` stops waveform, shimmer and pulse animations.
* Light theme lime text uses `--acct` (#4B6600) for contrast.
