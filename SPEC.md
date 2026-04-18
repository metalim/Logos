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
- **Scroll wheel:** ebitenui does not hook the wheel on `ScrollContainer` by default; the game registers `ScrolledEvent` on the scroll widget and adjusts `ScrollTop` manually.

### Scroll math

- `contentH` = preferred height of the news `Text`.
- `viewH` = `feedScroll.ViewRect().Dy()`.
- `extra = contentH - viewH` = scrollable slack in **pixels** (≤ 0 means no vertical scroll).

**Normalized position:** `ScrollTop` ∈ [0, 1] (ebitenui convention: 0 top, 1 bottom).

**Wheel delta:** for event argument `a.Y` (Ebiten wheel step),

`delta = a.Y * viewH / extra` added to normalized scroll (`ScrollTop -= delta`, then clamped). One unit `|a.Y| == 1` moves by one viewport height of *content slack*.

### Smoothed scroll state (game-owned)

The game keeps scroll goals in sync with the widget:

| Field | Meaning |
|-------|---------|
| `feedScrollTarget` | Normalized goal [0, 1]; bottom =1. |
| `feedScrollPx` | Smoothed offset in pixels along slack; `ScrollTop = feedScrollPx / extra` after update. |
| `feedScrollLastSmooth` | Wall time for computing `dt` between `smoothFeedScroll` calls. |
| `feedScrollNeedBottom` | Layout or content changed; bottom request must run **after** `ui.Update`. |

**Why defer `requestFeedScrollBottom`:** `Layout` can run inside `ui.Update`. Calling `newsText.PreferredSize()` from `requestFeedScrollBottom` during `Layout` has triggered nil derefs inside ebitenui `Text.measure`. Flow: set `feedScrollNeedBottom` from `applyVerticalBands` / `pushTestNews`; after `g.ui.Update()`, clear the flag and call `requestFeedScrollBottom()`.

**`requestFeedScrollBottom`:** sets `feedScrollTarget = 1`. If `feedScrollPx` is still uninitialized (`< 0`) and `extra > 0`, initializes `feedScrollPx` from current `ScrollTop * extra`.

**`smoothFeedScroll` (each frame after UI update):**

1. Recompute `extra`; if `extra <= 0`, set `ScrollTop = 0`, `feedScrollPx = 0`, return.
2. `targetPx = feedScrollTarget * extra`.
3. Sync/clamp `feedScrollPx` if uninitialized or past end.
4. Let `dt = time.Since(feedScrollLastSmooth)` (seconds), update `feedScrollLastSmooth`, clamp `dt` to `(0, 0.2]` (use `1/60` if invalid or after a long gap).
5. `k = 1 - exp(-feedScrollLambda * dt)` with `feedScrollLambda = 14` (tunable).
6. `diff = targetPx - feedScrollPx`. If `|diff| < 0.25` px, snap to `targetPx`; else `feedScrollPx += diff * k`.
7. Write `ScrollTop = feedScrollPx / extra` and clamp widget + `feedScrollPx`.

This is **proportional easing** (smaller steps near the target), not fixed-duration linear motion.

**Manual wheel:** updates `ScrollTop`, then sets `feedScrollTarget` and `feedScrollPx` to match so smoothing does not fight the user.

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
