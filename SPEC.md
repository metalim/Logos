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

Percents are constants (`bandTopPercent = 6`, `bandMidPercent = 60`); bottom band fills the remainder. Top band tracks a real phone status strip (iPhone is ~5–6% of screen height). Heights are written as `MinHeight` on each band by `applyVerticalBands`.

| Band | Role | Min height |
|------|------|------------|
| Top `bandTopPercent`% | Phone-style title bar (clock + signal + battery) | `outsideH * bandTopPercent / 100` |
| Middle `bandMidPercent`% | Map panel (placeholder container) | `outsideH * bandMidPercent / 100` |
| Bottom remainder | News feed (`ScrollContainer` + `Text`) | `outsideH - h1 - h2 - 2*bandSpacingPx` |

- **Root layout:** vertical `RowLayout` with `Spacing(bandSpacingPx)` (1 px gap shows the dark root background between bands). Each band carries `RowLayoutData{Stretch: true}` + `MinSize(0, 1)`; real height is set later by `applyVerticalBands`. Helper `newBandContainer(bg, innerLayout)` removes the per-band `WidgetOpts` boilerplate.
- **News `Text`:** `MaxWidth = outsideW - newsTextSideInset` (20 px), clamped to `newsTextMinWidth` (40 px).

## Phone title bar

- **Container:** the top band (`statusBar`) uses `AnchorLayout` with 2 px vertical padding and a near-black background, mimicking a phone status strip.
- **Left:** `widget.Text` (`clockText`) anchored start/center, `15:04` 24-hour format, refreshed every frame in `Update` via `updateClock` (no-op if label unchanged).
- **Right:** `widget.Container` with `RowLayout` (horizontal, spacing 6, right padding 8), holding two `widget.Graphic`s built once with `vector.DrawFilledRect` / `StrokeRect`:
  - **Signal:** `signalBarCount = 4` ascending bars (`signalBarW = 3`, gap 2, base 4 px, step 3 px). Filled bars use `titleBarFG`, missing bars use `titleBarDimFG`.
  - **Battery:** outlined body (`22×10`) with `2×4` tip on the right; inside, `batterySegments = 4` filled cells (`batterySegInset = 2`, `batterySegInterval = 1`).
- Game-state header (infection %, patches, timer per CONCEPT) is **not** in this bar — it will be a separate band added later.

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

1. `contentPx = a.Y * feedWheelContentPixelsPerUnit`.
2. If `contentPx != 0` and `|contentPx| < 1`, use **±1** content pixel (avoids a dead zone on tiny trackpad deltas).
3. Normalized step `delta = contentPx / extra`; `feedScrollTarget` is clamped to [0, 1] after subtracting `delta`.

The wheel does **not** jump by a full viewport per notch (unlike `a.Y * viewH / extra`).

**`feedWheelContentPixelsPerUnit` is platform-split via build tags** because `a.Y` units differ:

| Platform | Source of `a.Y` | Min observed | Constant | File |
|----------|-----------------|--------------|----------|------|
| Desktop (`!js`) | GLFW scroll units (fractional lines) | ~0.1 / step | **18.0** (≈ px per line at 14pt) | `wheel_native.go` |
| WASM (`js`) | DOM `WheelEvent.deltaY`, **`deltaMode` ignored upstream** in ebiten v2 (`internal/ui/input_js.go`) | mouse ≈ 4 px/tick, trackpad ≈ 1 px/step (macOS) | **1.0** (1:1 px) | `wheel_js.go` |

A single constant cannot work for both: 18 on js turns one mouse tick (4 → 72 px ≈ 5 lines) into a jump; 1.0 on desktop would make the wheel near-inert (0.1 → 0.1 px, snapped to 1).

### Drag-to-scroll (touch + left mouse)

Implemented in `feed_drag.go`, called from `Update` between `requestFeedScrollBottom` and `stepSmoothFeedScroll`.

**State (game-owned):**

| Field | Meaning |
|-------|---------|
| `feedDragActive` | A press started inside `feedScroll.GetWidget().Rect` and the pointer is still down. |
| `feedDragMouse` | True for left-mouse drag; false for touch. |
| `feedDragTouchID` | `ebiten.TouchID` captured at press; ignored when `feedDragMouse`. |
| `feedDragStartY` | Pointer Y at press (screen coords). |
| `feedDragStartPx` | `feedScrollPx` snapshot at press; if uninitialized (`< 0`), seeded from `ScrollTop * extra`. |

**Begin:** scan `inpututil.AppendJustPressedTouchIDs` first (mobile takes priority), then `IsMouseButtonJustPressed(MouseButtonLeft)`. Press must lie inside `feedScroll.GetWidget().Rect`.

**Move (per frame, while active):**

1. Confirm the pointer is still down — touch ID still in `ebiten.AppendTouchIDs(nil)`, or left mouse still pressed. Otherwise clear `feedDragActive`.
2. If `extra <= 0`, no-op (content shorter than viewport).
3. `newPx = feedDragStartPx - (currentY - feedDragStartY)`, clamped to `[0, extra]`.
4. **Write `feedScrollPx = newPx` directly** (1:1 with finger, bypassing easing) **and** `feedScrollTarget = newPx / extra`.

**Why bypass easing:** mobile expectation is finger-locked content. Setting both `feedScrollPx` and `feedScrollTarget` to the same value means `stepSmoothFeedScroll` sees `diff = 0` after release and leaves the position unchanged until the next wheel/auto-bottom request.

**Interaction with auto–scroll-to-bottom:** `requestFeedScrollBottom` runs **before** `handleFeedDrag` in `Update`, so a drag in progress overrides any pending bottom request for that frame. New content arriving mid-drag still grows `extra`; the user keeps the same `feedScrollPx`, so the visible position stays put.

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
| Status bar (time + game stats) | Phone strip (clock + signal + battery); game stats deferred to a separate header |
| Interactive node map | Empty styled container |
| Core loop (attack / patch / fail) | Not implemented |
| News as consequence stream | Placeholder + test ticker |

## File map

| Path | Role |
|------|------|
| `main.go` | Game struct, UI tree, bands, feed scroll, test news |
| `titlebar.go` | Phone-style title bar (clock + signal + battery icons via `vector`) |
| `feed_drag.go` | Touch / left-mouse drag-to-scroll for the news feed |
| `wheel_native.go` | `feedWheelContentPixelsPerUnit = 18.0` (build tag `!js`) |
| `wheel_js.go` | `feedWheelContentPixelsPerUnit = 1.0` (build tag `js`) |
| `wasm/index.html` | WASM shell copied to `dist/wasm/` |
| `Makefile` | `build`, `wasm`, `serve-wasm`, `clean` |
