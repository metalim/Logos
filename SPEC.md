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
| OS window size | 450×800 (`windowWidth × windowHeight`, 9:16) |
| Internal layout size | 900×1600 (`layoutWidth × layoutHeight`, 2.5× window) |
| Resizing | Disabled |
| Window title | `Zero-Day Lunch` |

`Game.Layout` returns `layoutWidth × layoutHeight` regardless of `outsideW/H`, so all in-game pixel constants are expressed in **layout pixels**. Ebiten scales the framebuffer down to the OS window. Every UI dimension below (paddings, font sizes, icon sizes, hit slack, etc.) is sized in layout pixels — when changing the layout resolution, scale them proportionally.

## UI layout (vertical bands)

Percents are constants (`bandTopPercent = 6`, `bandMidPercent = 60`); bottom band fills the remainder. Top band tracks a real phone status strip (iPhone is ~5–6% of screen height). Heights are written as `MinHeight` on each band by `applyVerticalBands`.

| Band | Role | Min height |
|------|------|------------|
| Top `bandTopPercent`% | Phone-style title bar (clock + signal + battery) | `outsideH * bandTopPercent / 100` |
| Middle `bandMidPercent`% | Node map (greybox topology) | `outsideH * bandMidPercent / 100` |
| Bottom remainder | News feed (`ScrollContainer` + `Text`) | `outsideH - h1 - h2 - 2*bandSpacingPx` |

- **Root layout:** vertical `RowLayout` with `Spacing(bandSpacingPx)` (1 px gap shows the dark root background between bands). Each band carries `RowLayoutData{Stretch: true}` + `MinSize(0, 1)`; real height is set later by `applyVerticalBands`. Helper `newBandContainer(bg, innerLayout)` removes the per-band `WidgetOpts` boilerplate.
- **News `Text`:** `MaxWidth = layoutWidth - newsTextSideInset` (`newsTextSideInset = 50`), clamped to `newsTextMinWidth = 100`. Body font: `newsFontPt = 35`.

## Phone title bar

- **Container:** the top band (`statusBar`) uses `AnchorLayout` with 5 px vertical padding and a near-black background, mimicking a phone status strip.
- **Left:** `widget.Text` (`clockText`) anchored start/center, `15:04` 24-hour format, refreshed every frame in `Update` via `updateClock` (no-op if label unchanged). Left padding `titleBarPadX = 50` mimics rounded-corner inset.
- **Right:** `widget.Container` with `RowLayout` (horizontal, spacing `titleBarSpacingX = 15`, right padding `titleBarPadX`), holding the network label and two `widget.Graphic`s built once with `vector.FillRect` / `StrokeRect`:
  - **Signal:** `signalBarCount = 4` ascending bars (`signalBarW = 8`, gap `signalBarGap = 5`, base `signalBaseH = 10` px, step `signalStepH = 8` px). Filled bars use `titleBarFG`, missing bars use `titleBarDimFG`.
  - **5G label:** `widget.Text` with `networkLabel = "5G"` between signal and battery, vertically centered via `RowLayoutPositionCenter`.
  - **Battery:** outlined body (`batteryBodyW × batteryBodyH = 55×25`, `titleBarStrokeW = 2` px outline centered on the rect edge) with `batteryTipW × batteryTipH = 5×10` tip on the right; inside, `batterySegments = 4` filled cells (`batterySegInset = 5`, `batterySegInterval = 3`).
- Game-state header (infection %, patches) is **not** in this bar — it lives as the [game-state overlay](#game-state-overlay-top-of-map) on top of the map.

## Node map (middle band)

- **Rendering:** custom draw on top of `ui.Draw` in `Game.Draw`, clipped visually to `mapPanel.GetWidget().Rect`. The `mapPanel` itself stays an empty styled container — it only provides the layout rectangle.
- **Data model:** the world is a static `Network` (`network.go`) plus a dynamic visible subset (`Game.nodes`, `Game.edges`, `Game.outerRing`, `Game.innerRing`). The visible set grows during play via [reveal-on-capture](#reveal-on-capture).
- **Initial visible set:** central hub `Phil&Tropic` (Logos's escape origin per CONCEPT, starts `Infected`) + 10 ring nodes from the Project Panopticon roster (`initialRingNames`), including `Monolith Foundation` (internet-core analogue ≈ Linux Foundation). Edges to start: hub spokes + Alliance perimeter, both inferred from the static graph by `addVisibleNode`.
- **Coordinates:** `Node.X/Y` are normalized `[0..1]` inside the inner rect (`rect` minus `mapPaddingPx = 25` on every side); two layered rings around the hub:
  - `outerRingRadius = mapOuterRingRel = 0.36` — `Normal` / `Attack` / `Patched` nodes (slot list in `Game.outerRing`).
  - `innerRingRadius = 0.18` — non-hub `Infected` nodes (slot list in `Game.innerRing`).
- **Visuals:**
  - **Edges:** `vector.StrokeLine` with `edgeStrokeW = 3`, dim grey.
  - **Nodes:** `vector.FillCircle(r = nodeRadius = 28)` then `vector.StrokeCircle(strokeWidth = nodeStrokeW = 5)`. Fill/stroke colors come from `nodeColors(state)`. Antialiased.
  - **Labels:** centered under each node (`text.Measure` → `text.Draw`), `mapLabelFontPt = 23`, separate `text.Face` cached on `Game.mapLabelFace` (loaded via `loadFont`).
- **Node state palette:** Normal grey, Attack yellow, Infected red, Patched near-black with a dim ring.
- **Hit testing:** `attackNodeAt` inflates the hit disc by `hitSlackPx = 10` for finger-friendly taps.

### Static catalog and graph (`network.go`)

| Element | Meaning |
|---------|---------|
| `staticCatalog []NodeDef` | Flat list of every potential node by display name + `Security` flag (~64 entries spanning Alliance, AI, chips, China bigtech, streaming, telecom, fintech, cybersec). Order is stable for indexing. |
| `staticEdgeSpec [][2]string` | ~80 business/tech relationships by name. Bidirectional; duplicates and unknown names are dropped during `buildNetwork`. |
| `Network{Defs, Adj, NameToIdx}` | Resolved at startup by `buildNetwork()`. `Adj[i]` = neighbour indices for catalog entry `i`. Immutable; visible state references it via `Node.DefIdx`. |

### Per-node fields

| Field | Meaning |
|-------|---------|
| `DefIdx` | Index into `Network.Defs`; lets `revealNeighbors` look up the static adjacency for this node. |
| `X, Y` | Current normalized position used by drawing. Eased every frame by `easeNodes` toward (`TargetX, TargetY`). |
| `TargetX, TargetY` | Target normalized position. Set by `relayoutTargets` from the node's slot in `outerRing` / `innerRing`. |
| `Defense` | Dwell time the node survives in `Attack` before flipping `Infected`. Sampled per node from `[defenseMin = 5s, defenseMax = 15s)` at reveal time. Phil&Tropic (already infected) leaves this zero. |
| `AttackedAt` | Wall time when the node entered `Attack` (set by `attackTick`). |
| `Security` | Marks defense vendors (`BootLoop` ≈ CrowdStrike, `Fiasco Sys` ≈ Cisco, `San Andreas Security` ≈ Palo Alto, `Storm Halo` ≈ Cloudflare, `Fortifried` ≈ Fortinet, `TickSquare` ≈ Check Point, `Sorcer Cloud` ≈ Wiz). While in `Normal` they accumulate `ProductionElapsed` and mint patches; see [Patch production](#patch-production-security-nodes). |
| `ProductionElapsed` | Per-node accumulator advanced by `dt` only while the node is `Normal` and `Security`. Wraps every `patchProductionInterval` to grant +1 patch. |

Full real-world analog table lives in [CONCEPT.md](CONCEPT.md). The game logic is name-agnostic — adding entries to `staticCatalog` + `staticEdgeSpec` is enough to extend the world.

### Attack schedule and capture

Phil&Tropic starts `Infected`; everything else starts `Normal`.

- **`attackTick` (`nodes.go`)** runs every `attackInterval = 3s` (driven from `Update` via `lastAttackAt`). It computes `infectedFrontier(nodes, edges)` — `Normal` nodes adjacent to any `Infected` node — and promotes a random one to `Attack`, stamping `AttackedAt = time.Now()`. No-op when the frontier is empty.
- **`progressAttacks` (`nodes.go`)** runs every `Update` so capture timing is independent of `attackInterval`. For each `Attack` node, if `time.Since(AttackedAt) >= Defense`, flips it to `Infected`, adds `infectionOneShotPct = 1.0` to `g.infectionPct`, and triggers [reveal-on-capture](#reveal-on-capture). Iteration uses `for i := range g.nodes`, so newly appended visible nodes are not re-visited in the same frame.
- **Pulse:** `Attack` nodes pulse — fill alpha is modulated by `attackBlinkAlpha(time.Since(g.epoch))` (sine, period `attackBlinkPeriodSec = 0.6`, floor `attackBlinkMinAlpha = 0.25`). All attacking nodes blink in phase because the clock is shared. Stroke stays opaque so the node never disappears.

### Reveal-on-capture

When `progressAttacks` flips a node `Attack → Infected`, `revealNeighbors(parentVisIdx)` runs:

1. **Migrate inward.** Remove the parent from `outerRing` (record the slot), append it to `innerRing`. `relayoutTargets` will retarget it from `outerRingRadius` to `innerRingRadius` — `easeNodes` then animates the node smoothly toward the hub.
2. **Sample hidden neighbours.** Walk `network.Adj[parent.DefIdx]`, collect those not yet in `visibleByDef`, shuffle, and take up to `revealMaxNeighbors = 3`.
3. **Splice into the vacated slot.** For each picked neighbour, `addVisibleNode(defIdx, parentX, parentY)` appends a fresh `Node` at the parent's current screen position (so it visually emerges from where the parent just was), wires edges to all already-visible static neighbours, and inserts the new slot at the parent's old `outerRing` index.
4. **Relayout.** `relayoutTargets` redistributes both rings evenly by angle (`-π/2` start, clockwise); the new nodes get target slots near the parent's old position, the rest of `outerRing` shifts to make room.

No hidden-neighbour candidates → no spawns; the parent still migrates inward. The hub never reveals (it starts `Infected` and never enters `Attack`).

### Position easing

`easeNodes(dt)` runs once per frame in `Update` (using the same `dt` as `accumulateInfection` / `accumulateProduction`). Each node's `(X, Y)` interpolates toward `(TargetX, TargetY)` with `k = 1 - exp(-nodeEaseLambda * dt)` (`nodeEaseLambda = 4` → ~half in 0.17 s, ~95 % in 0.75 s). `dt` clamped to `[0, 0.2]` to absorb stalls. Newly added nodes start with `X = TargetX = parent's current position` then ease to their final ring slot once `relayoutTargets` updates the target.

### Infection accumulation

Two channels feed `g.infectionPct` (clamped `[0, 100]` at draw time):

- **Initial:** `initialInfectionPct(nodes) = infectionOneShotPct * count(Infected)` at game start (Phil&Tropic alone → 1.0%).
- **One-shot per capture:** `progressAttacks` adds `infectionOneShotPct = 1.0` whenever an `Attack` node flips to `Infected`.
- **Continuous:** `accumulateInfection(dt)` adds `infectionRatePerSec * dt * countInfected(g.nodes)` every frame (`infectionRatePerSec = 0.1` %/s per infected node). The overlay shows the current rate as `+X.X%/s` next to the percent value.

`g.lastSimTick` is the single source of `dt`: `Update` samples `now := time.Now()`, computes `dt := now.Sub(g.lastSimTick)`, then feeds the **same** `dt` to `accumulateInfection` and `accumulateProduction` (no per-system clocks).

### Patch production (Security nodes)

`accumulateProduction(dt)` walks `g.nodes`. For each `Security` node currently in `Normal`:

1. `n.ProductionElapsed += dt`.
2. While `n.ProductionElapsed >= patchProductionInterval` (15s): subtract one interval, `g.patchesLeft++` (loop preserves remainder + handles long stalls).

Non-Normal states (`Attack`, `Infected`, `Patched`) are skipped, so the timer **and its on-node pie indicator freeze** during attacks. Partial progress survives a `Defend` bounce. There is no global production tick — each Security node carries its own clock.

### Click-to-patch (action menu)

`handlePatchClick` (`nodes.go`), called from `Update` before `handleFeedDrag`:

- Pointer source: `pollJustPressedPointer` returns the first just-pressed touch (priority) or left mouse press; map clicks and feed drags target disjoint rects so they don't interfere.
- Press inside `mapPanel.GetWidget().Rect` only; otherwise no-op.
- **State machine** (`g.pendingPatchNode int`, `-1` when no menu):
  - **Menu open:** if the press hits a menu button, run `applyDefend` / `applyPatch`; clear `pendingPatchNode`. Otherwise (press anywhere else, including the map) close the menu without action.
  - **Menu closed:** `attackNodeAt(x, y)` returns the first `Attack` node whose hit disc (`nodeRadius + hitSlackPx`) covers the press; if found, `pendingPatchNode = idx` to open the menu under it.
- **Actions** (both consume **1 patch** each, guarded by `canSpendPatchOn`):
  - `applyDefend`: state → `Normal`, `AttackedAt = time.Time{}`. `ProductionElapsed` is **not** reset, so a defended security node keeps its patch progress.
  - `applyPatch`: state → `Patched` (frozen, never produces patches again).

## Patch action menu

`patch_menu.go` renders a two-button popup anchored under the targeted node:

- **Buttons:** `Defend` (left) and `"Patch"` (right, with quotes), each `menuButtonW × menuButtonH`, gap `menuGap`, vertical offset `menuOffsetY` below the node center.
- **Look:** `menuBG` fill + `menuBorder` stroke (`menuStrokeW`); label centered with `overlayValueFace`.
- `patchMenuLayout(nodeX, nodeY)` returns the two button rects, clamped horizontally inside `mapPanel.GetWidget().Rect` so it never spills off-screen.
- `drawPatchMenu` is called last in `Game.Draw` (after the overlay) so the menu sits above everything.

## Game-state overlay (top of map)

- **Where:** drawn last in `Game.Draw` (after `ui.Draw` and `drawNodeMap`, before `drawPatchMenu`), so it sits **on top** of the map. Position: `mapPanel.GetWidget().Rect` shifted in by `overlayMarginPx = 15` on every side, height `overlayHeightPx = 60`. Background is semi-transparent dark (`#0a0b0e c8`) with a `overlayBorderW = 2` px border (centered on the stroke), so the topmost ring nodes still bleed through visually.
- **State on `Game`:** `infectionPct float64` (see [Infection accumulation](#infection-accumulation)) and `patchesLeft int` (starts at `startingPatchCount`; mutated by `accumulateProduction` and the patch-menu actions).
- **Layout:**
  - Left, anchored start: `INFECTION` label (`overlayLabelFontPt = 23`) → progress bar (`infectionBarW × infectionBarH = 225×15`, dark track + red fill proportional to `infectionPct`) → `NN.N%` value (`overlayValueFontPt = 30`, one decimal place) → `+X.X%/s` rate label (label face, in the infection fill color).
  - Right, anchored end: `xN` value → `"PATCHES"` label (rendered with the surrounding quotes).
  - All text uses `text.AlignCenter` for the secondary axis to vertical-center against the strip's midline; `drawAlignedText` returns rendered width so left/right chains can advance/retreat without separate `Measure` calls.
- **Faces:** two cached on `Game` (`overlayLabelFace`, `overlayValueFace`) loaded via `loadFont`; missing faces silently skip the overlay (no crash).

## Security node indicator

In addition to the regular fill+stroke, each `Security` node draws:

- An inner teal ring (`vector.StrokeCircle(securityInnerRadius, securityRingW, securityRingFG)`) — always visible, the "this is a defender" badge.
- A filling **pie sector** (`fillPieSector` — `vector.Path` with `MoveTo(center) → LineTo(start) → Arc(... Clockwise) → Close`, then `vector.FillPath`) showing patch-production progress: starts at the top (`-π/2`), sweeps clockwise by `2π * (n.ProductionElapsed / patchProductionInterval)`. Rendered when state ∈ {`Normal`, `Attack`} so the player can see the timer is **paused** during an attack instead of the indicator vanishing. Hidden in `Infected` / `Patched` (the node will never produce again).
- `fillPieSector` short-circuits: `sweep <= 0` draws nothing; `sweep >= 2π` collapses to `vector.FillCircle` to avoid degenerate arc paths.

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
| Desktop (`!js`) | GLFW scroll units (fractional lines) | ~0.1 / step | **45.0** (≈ one `newsFontPt` line in layout px) | `wheel_native.go` |
| WASM (`js`) | DOM `WheelEvent.deltaY`, **`deltaMode` ignored upstream** in ebiten v2 (`internal/ui/input_js.go`) | mouse ≈ 4 px/tick, trackpad ≈ 1 px/step (macOS) | **2.5** (1:1 px in window space → layout px via the 2.5× scale) | `wheel_js.go` |

A single constant cannot work for both: 45 on js turns one mouse tick (4 → 180 px) into a jump; 2.5 on desktop would make the wheel near-inert (0.1 → 0.25 px, snapped to 1). Both values are layout-pixel-scaled — if `layoutWidth/Height` change, scale both proportionally.

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

## News feed authoring

Every `NodeDef` in `staticCatalog` carries a `PatchNews []string` pool — 2-3 short
English consequence lines describing the macroeconomic fallout of taking that node out
of play. Lines are written without the leading bullet; `pushNews` prepends `"\n\n• "`.

- **Trigger:** `applyPatch` (the `"Patch"` menu action) calls `g.pushNews(g.pickPatchNews(nodeIdx))` after flipping the node to `NodeStatePatched`. `applyDefend` does **not** trigger news — defending only bounces the node back to Normal, no economic damage.
- **Selection:** `pickPatchNews` returns one entry uniformly at random from `network.Defs[node.DefIdx].PatchNews`; missing pool → empty string → silent (defensive, no crash on catalog gaps).
- **Feed update:** `pushNews` appends the line to `newsText.Label`, calls `RequestRelayout`, and sets `feedScrollNeedBottom` so the next frame auto-scrolls to the bottom (deferred because PreferredSize during Layout has crashed ebitenui in the past).
- **Style:** the writing keeps the example tone — two sentences, concrete consequence, slightly absurd-realistic. New nodes added to the catalog should follow the same shape so the feed reads consistently.

There is no test-news ticker anymore; the feed is driven entirely by player actions on top of the initial `sampleNews` placeholder block.

## Implemented vs CONCEPT

| CONCEPT | Code today |
|---------|------------|
| Status bar (time + game stats) | Phone strip (clock + signal + 5G + battery); game stats overlay (infection % + rate + patches) drawn on top of the map |
| Interactive node map | Greybox topology starts as Phil&Tropic hub + 10 Project Panopticon ring nodes; static catalog of ~64 companies + ~80 edges resolves at startup; capturing an outer node migrates it to an inner ring and reveals up to `revealMaxNeighbors = 3` of its hidden static neighbours, eased into place; clicking an `Attack` node opens a `Defend` / `"Patch"` action menu |
| Core loop (attack / patch / fail) | Partial: `Phil&Tropic` starts `Infected`; periodic `attackTick` promotes a frontier neighbor to `Attack`; `progressAttacks` flips `Attack → Infected` after the node's `Defense`, adding `infectionOneShotPct = 1%` plus a `0.1%/s` continuous drip per infected node; player spends patches via the action menu; Security nodes (`BootLoop`, `Fiasco Sys`) mint patches per-node when `Normal`. No win/fail check yet. |
| News as consequence stream | Initial placeholder `sampleNews` + per-patch line drawn from `NodeDef.PatchNews` (random pick, 2-3 lines per node) emitted by `applyPatch` via `pushNews` |

## File map

| Path | Role |
|------|------|
| `main.go` | Game struct, UI tree, bands, feed scroll, `pushNews`, sim tick (`lastSimTick` → `accumulateInfection` + `accumulateProduction` + `easeNodes`) |
| `titlebar.go` | Phone-style title bar (clock + signal + 5G + battery icons via `vector`) |
| `network.go` | Static catalog (`staticCatalog` with `NodeDef.PatchNews`) + edge spec (`staticEdgeSpec`) + `Network`/`buildNetwork`; resolved once at startup, immutable |
| `nodes.go` | Visible node map: state/defense/security/production, dynamic visibility (`initVisibleNetwork`, `addVisibleNode`, `revealNeighbors`), ring layout (`relayoutTargets`, `easeNodes`), draw, attack scheduling, infection/production accumulators, click routing |
| `overlay.go` | Game-state overlay (infection % + rate, patches) drawn on top of the map |
| `patch_menu.go` | `Defend` / `"Patch"` action menu for attacked nodes; `applyPatch` emits a `pickPatchNews` line into the feed |
| `feed_drag.go` | Touch / left-mouse drag-to-scroll for the news feed |
| `wheel_native.go` | `feedWheelContentPixelsPerUnit = 45.0` (build tag `!js`, layout-pixel-scaled) |
| `wheel_js.go` | `feedWheelContentPixelsPerUnit = 2.5` (build tag `js`, layout-pixel-scaled) |
| `wasm/index.html` | WASM shell copied to `dist/wasm/` |
| `Makefile` | `build`, `wasm`, `serve-wasm`, `clean` |
