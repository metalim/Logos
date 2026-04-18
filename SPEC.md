# Logos — technical specification

Narrative design, lore, and target gameplay loop live in [CONCEPT.md](CONCEPT.md). This document describes **what the codebase implements today** and **non-obvious UI rules** so behavior stays consistent when changing code.

## Stack

- **Go**, [Ebiten v2](https://ebitengine.org/), [ebitenui](https://github.com/ebitenui/ebitenui)
- **Font:** `golang.org/x/image/font/gofont/gomono` via `ebiten/v2/text/v2`
- **Desktop build:** `CGO_ENABLED=0` (see [Makefile](Makefile))
- **WASM:** `make wasm` → `dist/wasm/` (`game.wasm`, `wasm_exec.js`, `index.html`)

## Window

| Property | Value |
|----------|--------|
| Logical size | 360×640 (9:16) |
| Resizing | Disabled |
| Window title | `Zero-Day Lunch` |

`Layout` always reports this size; band heights are derived from `outsideH` when it changes.

## UI layout (vertical bands)

Percents are integer fractions of the **outside** height passed into `applyVerticalBands`.

| Band | Role | Min height |
|------|------|------------|
| Top 10% | Status bar (placeholder container) | `outsideH * 10 / 100` |
| Middle 60% | Map panel (placeholder container) | `outsideH * 60 / 100` |
| Bottom remainder (~30%) | News feed (`ScrollContainer` + `Text`) | `outsideH - h1 - h2` |

- **Anchor layout** on root: bands are stacked using `AnchorLayoutData` + top padding so the feed fills the lower strip.
- **News `Text`:** `MaxWidth = outsideW - 20`, clamped to at least 40 px.

## News feed widget

- **Content:** `widget.Text` inside `widget.ScrollContainer` (`StretchContentWidth`).
- **Initial text:** long placeholder block (`sampleNews` repeated) plus optional test lines from `pushTestNews`.
- **Scroll wheel:** ebitenui does not hook the wheel on `ScrollContainer` by default; the game registers `ScrolledEvent` and updates **`feedScrollTarget`** only. **`ScrollTop`** is written from smoothed state in **`stepSmoothFeedScroll`** (same path as auto–scroll-to-bottom).

### Scroll math

- `contentH` = preferred height of the news `Text`.
- `viewH` = `feedScroll.ViewRect().Dy()`.
- `extra = contentH - viewH` = scrollable slack in **pixels** (≤ 0 means no vertical scroll).

**`feedScrollSlack()`** returns `(extra, viewH, ok)` from the current `newsText` + `feedScroll` so wheel and smoothing share one measurement.

**Normalized position:** `ScrollTop` ∈ [0, 1] (ebitenui: 0 top, 1 bottom). The game tracks **`feedScrollPx`** in **pixels** along the slack; after smoothing, `ScrollTop = feedScrollPx / extra` (clamped).

**Wheel (manual):** for `WidgetScrolledEventArgs` vertical `a.Y`:

1. `contentPx = a.Y * feedWheelContentPixelsPerUnit` (default **18** — about one line at 14pt; tunable in `main.go`).
2. If `contentPx != 0` and `|contentPx| < 1`, use **±1** content pixel (avoids a dead zone on tiny trackpad deltas).
3. Normalized step `delta = contentPx / extra`; `feedScrollTarget` is clamped to [0, 1] after subtracting `delta`.

The wheel does **not** jump by a full viewport per notch (unlike `a.Y * viewH / extra`).

### Smoothed scroll state (game-owned)

| Field | Meaning |
|-------|---------|
| `feedScrollTarget` | Normalized goal [0, 1]; bottom = 1. |
| `feedScrollPx` | Smoothed offset in pixels along slack; drives `ScrollTop` after `stepSmoothFeedScroll`. |
| `feedScrollLastSmooth` | Wall time for `dt` between `stepSmoothFeedScroll` calls. |
| `feedScrollNeedBottom` | Layout or content changed; bottom request must run **after** `ui.Update`. |

**Why defer `requestFeedScrollBottom`:** `Layout` can run inside `ui.Update`. Calling `newsText.PreferredSize()` during `Layout` has triggered nil derefs in ebitenui `Text.measure`. Flow: set `feedScrollNeedBottom` from `applyVerticalBands` / `pushTestNews`; after `g.ui.Update()`, clear the flag and call `requestFeedScrollBottom()`.

**`requestFeedScrollBottom`:** sets `feedScrollTarget = 1`. If `feedScrollPx < 0` (uninitialized) and `extra > 0`, sets `feedScrollPx = ScrollTop * extra`.

**`stepSmoothFeedScroll` (every frame after `ui.Update`):**

1. Recompute `extra` via `feedScrollSlack()`; if `extra <= 0`, set `ScrollTop = 0`, `feedScrollPx = 0`, return.
2. `targetPx = feedScrollTarget * extra`.
3. Sync/clamp `feedScrollPx` if uninitialized or past end.
4. `dt = time.Since(feedScrollLastSmooth)`, update `feedScrollLastSmooth`, clamp `dt` to `(0, 0.2]` (else use `1/60` after a stall).
5. `k = 1 - exp(-feedScrollLambda * dt)` with `feedScrollLambda = 14`.
6. `diff = targetPx - feedScrollPx`. If `|diff| < 0.25` px, snap; else `feedScrollPx += diff * k`.
7. `ScrollTop = feedScrollPx / extra` and clamp widget + `feedScrollPx`.

**Proportional easing** toward `feedScrollTarget * extra`: same step runs for **new content (scroll to bottom)** and **manual wheel** (target moves; `feedScrollPx` follows).

## Test harness

- Every **`testNewsInterval`** (5 seconds, wall clock), append a formatted test bullet to `newsText.Label`, `RequestRelayout`, set `feedScrollNeedBottom`.
- Used to validate feed growth and auto-scroll without full game logic.

## Implemented vs CONCEPT

| CONCEPT | Code today |
|---------|------------|
| Status bar content (time, infection, patches) | Empty styled container |
| Interactive node map | Empty styled container |
| Core loop (attack / patch / fail) | Not implemented |
| News as consequence stream | Placeholder + test ticker |

## File map

| Path | Role |
|------|------|
| `main.go` | Game struct, UI tree, bands, feed scroll, test news |
| `wasm/index.html` | WASM shell copied to `dist/wasm/` |
| `Makefile` | `build`, `wasm`, `serve-wasm`, `clean` |
