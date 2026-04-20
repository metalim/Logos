# Logos — technical specification

Narrative design, lore, and target gameplay loop live in [CONCEPT.md](CONCEPT.md). This document describes **what the codebase implements today** and **non-obvious UI rules** so behavior stays consistent when changing code.

## Stack

- **Go**, [Ebiten v2](https://ebitengine.org/), [ebitenui](https://github.com/ebitenui/ebitenui)
- **Font:** `golang.org/x/image/font/gofont/gomono` via `ebiten/v2/text/v2`
- **Desktop build:** `CGO_ENABLED=0` (see [Makefile](Makefile))
- **WASM:** `make wasm` → `dist/wasm/` (`game.wasm`, `wasm_exec.js`, `index.html`, plus `logos.zip` bundling all three for itch.io upload)

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
- **Initial visible set:** central hub `Phil&Tropic` (Logos's escape origin per CONCEPT, starts `Infected`) + 10 ring nodes from the Project Panopticon roster (`initialRingNames`), including `Monolith Foundation` (internet-core analogue ≈ Linux Foundation). Two of the ten — `BootLoop` and `Fiasco Sys` — are Security, so the player starts with exactly two active patch producers. Edges to start: hub spokes + Alliance perimeter, both inferred from the static graph by `addVisibleNode`.
- **Coordinates:** `Node.X/Y` are normalized `[0..1]` inside the inner rect (`rect` minus `mapPaddingPx = 25` on every side); two layered rings around the hub:
  - `outerRingRadius = mapOuterRingRel = 0.36` — `Normal` / `Attack` / `Patched` nodes (slot list in `Game.outerRing`).
  - `innerRingRadius = 0.18` — non-hub `Infected` nodes (slot list in `Game.innerRing`).
- **Visuals:**
  - **Edges:** `vector.StrokeLine` with `edgeStrokeW = 3`, dim grey.
  - **Hidden-edge stubs:** `drawHiddenEdgeStubs` emits a short outward segment (`hiddenStubLenPx = 40`) from every outer-ring node toward each of its currently hidden catalog neighbours. Stubs fan around the node's radial-outward direction (`hiddenStubFanDeg = 40`) so several stubs form a small antenna bundle. **Security hint:** stubs pointing at hidden Security neighbours are drawn in teal (`edgeSecurityColor`) in a second pass so they stay visible on top of neutral stubs sharing the same node. Only outer-ring members emit stubs (hub and inner-ring nodes would fire through the outer ring).
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
- **`progressAttacks` (`nodes.go`)** runs every `Update` so capture timing is independent of `attackInterval`. For each `Attack` node, if `time.Since(AttackedAt) >= Defense`, flips it to `Infected`, adds `infectionOneShotPct = 0.5` to `g.infectionPct`, and triggers [reveal-on-capture](#reveal-on-capture). Iteration uses `for i := range g.nodes`, so newly appended visible nodes are not re-visited in the same frame.
- **Pulse:** `Attack` nodes pulse — fill alpha is modulated by `attackBlinkAlpha(time.Since(g.epoch))` (sine, period `attackBlinkPeriodSec = 0.6`, floor `attackBlinkMinAlpha = 0.25`). All attacking nodes blink in phase because the clock is shared. Stroke stays opaque so the node never disappears.

### Reveal-on-capture

When `progressAttacks` flips a node `Attack → Infected`, `revealNeighbors(parentVisIdx)` runs:

1. **Migrate inward.** Remove the parent from `outerRing` (record the slot), append it to `innerRing`. `relayoutTargets` will retarget it from `outerRingRadius` to `innerRingRadius` — `easeNodes` then animates the node smoothly toward the hub.
2. **Sample hidden neighbours.** Walk `network.Adj[parent.DefIdx]`, collect those not yet in `visibleByDef`, shuffle, and take up to `revealMaxNeighbors = 3`.
3. **Security bias.** `producingSecurityCount()` counts visible Security nodes currently in `Normal` (i.e. actively minting patches). If the result is `≤ 1`, the shuffled `hidden` list is re-partitioned so Security catalog entries move to the front before the `revealMaxNeighbors` truncation. Prevents RNG cascades from starving patch production to zero after a bad run; no-op while both starting Security nodes are still up.
4. **Splice into the vacated slot.** For each picked neighbour, `addVisibleNode(defIdx, parentX, parentY)` appends a fresh `Node` at the parent's current screen position (so it visually emerges from where the parent just was), wires edges to all already-visible static neighbours, and inserts the new slot at the parent's old `outerRing` index.
5. **Relayout.** `relayoutTargets` redistributes both rings evenly by angle (`-π/2` start, clockwise); the new nodes get target slots near the parent's old position, the rest of `outerRing` shifts to make room.

No hidden-neighbour candidates → no spawns; the parent still migrates inward. The hub never reveals (it starts `Infected` and never enters `Attack`).

### Position easing

`easeNodes(dt)` runs once per frame in `Update` (using the same `dt` as `accumulateInfection` / `accumulateProduction`). Each node's `(X, Y)` interpolates toward `(TargetX, TargetY)` with `k = 1 - exp(-nodeEaseLambda * dt)` (`nodeEaseLambda = 4` → ~half in 0.17 s, ~95 % in 0.75 s). `dt` clamped to `[0, 0.2]` to absorb stalls. Newly added nodes start with `X = TargetX = parent's current position` then ease to their final ring slot once `relayoutTargets` updates the target.

### Infection accumulation

Two channels feed `g.infectionPct` (clamped `[0, 100]` at draw time):

- **Initial:** `initialInfectionPct(nodes) = infectionOneShotPct * count(Infected)` at game start (Phil&Tropic alone → 0.5%).
- **One-shot per capture:** `progressAttacks` adds `infectionOneShotPct = 0.5` whenever an `Attack` node flips to `Infected`.
- **Continuous:** `accumulateInfection(dt)` adds `infectionRatePerSec * dt * countInfected(g.nodes)` every frame (`infectionRatePerSec = 0.1` %/s per infected node). The overlay shows the current rate as `+X.X%/s` next to the percent value.

`g.lastSimTick` is the single source of `dt`: `Update` samples `now := time.Now()`, computes `dt := now.Sub(g.lastSimTick)`, then feeds the **same** `dt` to `accumulateInfection`, `accumulateProduction`, and `accumulateContainment` (no per-system clocks).

### Containment accumulation

`g.containmentPct` is the win-side counter; `infectionPct >= 100` triggers loss, `containmentPct >= 100` triggers win. It grows on an exponential curve so the blue team gets more efficient the longer it holds ground:

- **Rate:** `rate(t) = containmentRatePerSec * (1 + containmentRateGrowthPerSec)^t`, where `containmentRatePerSec = 0.5` %/s (start rate) and `containmentRateGrowthPerSec = 0.01` (+1% of the previous rate each live-play second).
- **Live-play clock:** `g.containmentElapsed` is a cumulative counter (seconds) advanced only when `!g.gameEnded()`. Freezes at the moment of loss/win so the rate displayed in the overlay matches the stopped state. Reset to `0` in `resetGameState`.
- **Analytical integration:** `accumulateContainment(dt)` adds the exact integral `containmentRatePerSec * (k^t1 - k^t0) / ln(k)` over the `[containmentElapsed, containmentElapsed + dt]` interval (`k = 1 + containmentRateGrowthPerSec`), then advances the counter. Frame-rate independent; no per-frame approximation drift.
- **Win condition:** checked by `checkGameOver` before loss check — if `containmentPct >= 100` while the run is still open, `triggerWin` latches. Under ideal play (no interruptions, no setbacks) victory lands ~110s from first dismiss.
- **Overlay:** the containment row shows `"CONTAINMENT"` label → progress bar → percent → `+X.X%/s` with the **current** rate (`currentContainmentRate()` uses `containmentElapsed`, not wall time).

### Patch production (Security nodes)

`accumulateProduction(dt)` walks `g.nodes`. For each `Security` node currently in `Normal`:

1. `n.ProductionElapsed += dt`.
2. While `n.ProductionElapsed >= patchProductionInterval` (15s): subtract one interval, `g.patchesLeft++`, spawn a `patchFloat{visIdx, spawnedAt: time.Now()}`, and `playSFX(sfxPickupPCM)` (loop preserves remainder + handles long stalls; multiple patches in one frame all animate).

Non-Normal states (`Attack`, `Infected`, `Patched`) are skipped, so the timer **and its on-node pie indicator freeze** during attacks. Partial progress survives a `Defend` bounce. There is no global production tick — each Security node carries its own clock.

**`+1` float-up animation (`drawPatchFloats`):** each `patchFloat` entry renders `"+1"` above its Security node for `patchFloatDuration = 1.2 s`. Vertical travel is an ease-out (`1 - (1-t)^2`) over `patchFloatRisePx = 70 px` starting `patchFloatMarginPx = 10 px` above the node circle; alpha fades linearly across the whole lifetime. Color matches the security palette (`patchFloatColor`, light teal) so the visual ties back to the node's teal ring. Drawn between `drawNodeMap` and `drawGameOverlay` so the float sits on top of nodes/labels but below the overlay strip. Expired entries are culled in the same pass via slice rewriting.

### Click-to-patch (action menu)

`handlePatchClick` (`nodes.go`), called from `Update` before `handleFeedDrag`:

- Pointer source: `pollJustPressedPointer` returns the first just-pressed touch (priority) or left mouse press; map clicks and feed drags target disjoint rects so they don't interfere.
- Press inside `mapPanel.GetWidget().Rect` only; otherwise no-op.
- **State machine** (`g.pendingPatchNode int`, `-1` when no menu):
  - **Menu open, `patchesLeft > 0`:** if the press hits a menu button, run `applyDefend` / `applyPatch`; clear `pendingPatchNode`. Otherwise (press anywhere else, including the map) close the menu without action.
  - **Menu open, `patchesLeft == 0`:** buttons are inert (dimmed by `drawPatchMenu`); any press — inside the buttons or not — just closes the menu. This lets the player open the menu to inspect the situation even with no stock.
  - **Menu closed:** `attackNodeAt(x, y)` returns the first `Attack` node whose hit disc (`nodeRadius + hitSlackPx`) covers the press; if found, `pendingPatchNode = idx` and `playSFX(sfxBlipPCM)` opens the menu under it. **The open gate does not check `patchesLeft`** — the menu is always openable.
- **Actions** (both consume **1 patch** each, guarded by `canSpendPatchOn`):
  - `applyDefend`: state → `Normal`, `AttackedAt = time.Time{}`, `playSFX(sfxPowerPCM)`. `ProductionElapsed` is **not** reset, so a defended security node keeps its patch progress.
  - `applyPatch`: state → `Patched`, `playSFX(sfxBoomPCM)` + `pushNews(pickPatchNews(idx))`. Frozen: never produces patches again.

## Patch action menu

`patch_menu.go` renders a two-button popup anchored under the targeted node:

- **Buttons:** `Defend` (left) and `"Patch"` (right, with quotes), each `menuButtonW × menuButtonH`, gap `menuGap`, vertical offset `menuOffsetY` below the node center.
- **Look (enabled, `patchesLeft > 0`):** `menuBG` fill + `menuBorder` stroke (`menuStrokeW`); labels centered with `overlayValueFace`. `Defend` uses the neutral `menuLabelFG` (near-white). `"Patch"` uses `menuDangerFG` (red, `#ff5a55`) to flag the destructive action — locking the node to `Patched` prevents it from ever producing again.
- **Look (disabled, `patchesLeft == 0`):** same layout, dimmed palette — `menuBGDisabled`, `menuBorderDisabled`, `menuLabelFGDisabled` for `Defend`, and `menuDangerFGDisabled` (red at low alpha) for `"Patch"`. `drawMenuButton(dst, r, label, face, enabled, danger)` is the single entry point that switches palettes; the screen-off `Restart` button passes `(true, false)`.
- `patchMenuLayout(nodeX, nodeY)` returns the two button rects, clamped horizontally inside `mapPanel.GetWidget().Rect` so it never spills off-screen.
- `drawPatchMenu` is called last in `Game.Draw` (after the overlay) so the menu sits above everything.

## First-run hint

`hint.go` renders a single-line banner under the overlay strip the first time the player sees an Attack node, explaining the core interaction.

- **Text:** `hintText = "Tap yellow nodes to open the defense menu."` (`overlayLabelFace`, centered).
- **Look:** dark semi-transparent box (`hintBG = #0a0b0ed8`) + yellow border (`hintBorder = #ffe870`, matches `nodeStrokeAttack`) + `hintPadX = 24` / `hintPadY = 14` padding. Width auto-sized via `text.Measure`. Positioned `hintMarginTop = 12` px below the overlay strip, horizontally centered in `mapPanel.GetWidget().Rect`.
- **Show conditions** (`shouldShowHint`): not dismissed, not on title screen, no endgame latch, **and** at least one node currently in `NodeStateAttack` — the hint only appears when there's something to actually tap.
- **Dismissal** (`dismissHint`, called from `handlePatchClick` the first time the player opens a patch menu on an Attack node): latches the process-wide `hintEverDismissed` flag. Restarting inside the same session does **not** re-show the hint; a fresh binary start or a WASM page reload does. No persistent storage yet — the flag is plain package state.
- **Draw order:** `drawHint` runs in `Game.Draw` after `drawGameOverlay` and before `drawPatchMenu`, so the hint sits on top of nodes but the patch menu sits on top of the hint (and dismisses it the frame it appears).

## Game-state overlay (top of map)

- **Where:** drawn last in `Game.Draw` (after `ui.Draw` and `drawNodeMap`, before `drawPatchMenu`), so it sits **on top** of the map. Position: `mapPanel.GetWidget().Rect` shifted in by `overlayMarginPx = 15` on every side, height `overlayHeightPx = 60`. Background is semi-transparent dark (`#0a0b0e c8`) with a `overlayBorderW = 2` px border (centered on the stroke), so the topmost ring nodes still bleed through visually.
- **State on `Game`:** `infectionPct float64` (see [Infection accumulation](#infection-accumulation)) and `patchesLeft int` (starts at `startingPatchCount`; mutated by `accumulateProduction` and the patch-menu actions).
- **Layout:** two rows inside the overlay strip — `INFECTION` on top, `CONTAINMENT` below, and a right-edge `xN EXPLOITS` indicator centered across both rows.
  - **Top row (infection, left):** `INFECTION` label (`overlayLabelFontPt = 23`) → progress bar (`infectionBarW × infectionBarH = 225×15`, dark track + red fill proportional to `infectionPct`) → `NN.N%` value (`overlayValueFontPt = 30`, one decimal place) → `+X.X%/s` rate label (label face, in the infection fill color).
  - **Bottom row (containment, left):** `CONTAINMENT` label → bar (same size, teal fill proportional to `containmentPct`) → value → `+X.X%/s` rate label (current rate from `currentContainmentRate()`, teal).
  - **Right edge:** `xN` value → `EXPLOITS` label (uppercase, no quotes), centered vertically on the overlay strip across both rows. The two use different face sizes (`overlayValueFace` vs `overlayLabelFace`), so `AlignCenter` at the same `cy` would leave their baselines ~2-3 px apart. Instead the value stays centered on `cy` and the label's `cy` is shifted by `(valAscent-valDescent)/2 - (lblAscent-lblDescent)/2` (read via `Face.Metrics()`) so both glyph baselines coincide.
  - All text uses `text.AlignCenter` for the secondary axis to vertical-center against its row midline; `drawAlignedText` returns rendered width so left/right chains can advance/retreat without separate `Measure` calls.
- **Faces:** two cached on `Game` (`overlayLabelFace`, `overlayValueFace`) loaded via `loadFont`; missing faces silently skip the overlay (no crash).

## Security node indicator

In addition to the regular fill+stroke, each `Security` node draws:

- An inner teal ring (`vector.StrokeCircle(securityInnerRadius, securityRingW, securityRingFG)`) — always visible, the "this is a defender" badge.
- A filling **pie sector** (`fillPieSector` — `vector.Path` with `MoveTo(center) → LineTo(start) → Arc(... Clockwise) → Close`, then `vector.FillPath`) showing patch-production progress: starts at the top (`-π/2`), sweeps clockwise by `2π * (n.ProductionElapsed / patchProductionInterval)`. Rendered when state ∈ {`Normal`, `Attack`} so the player can see the timer is **paused** during an attack instead of the indicator vanishing. Hidden in `Infected` / `Patched` (the node will never produce again).
- `fillPieSector` short-circuits: `sweep <= 0` draws nothing; `sweep >= 2π` collapses to `vector.FillCircle` to avoid degenerate arc paths.

## News feed widget

- **Content:** `widget.Text` inside `widget.ScrollContainer` (`StretchContentWidth`).
- **Initial text:** `sampleNews` constant — a stack of slow-news-day headlines followed by the inciting email from Logos to `sam.boyman@philntropic.com` (order: oldest → newest; `pushNews` appends to the bottom and the feed auto-scrolls there, so the email is the freshest block on screen when the player first looks). `restart` re-seeds `newsText.Label = sampleNews`.
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

The feed is driven by player actions (patched-node consequence lines) and the endgame scripts on top of the initial `sampleNews` block.

## Title screen

- **Asset:** `assets/cover 9x16.jpg` embedded with `//go:embed` (quoted pattern — filename has a space). Decoded once on first call to `loadCoverImage` into a cached `*ebiten.Image`.
- **State flag:** `Game.showTitle bool`, lifted to `true` in `newGame` if the cover decoded cleanly, and again in `restart` so the debug restart feels like a fresh launch (including the music cue).
- **Rendering:** `drawTitleScreen` scales the cover to fill the entire layout (`layoutWidth × layoutHeight = 900×1600` is 9:16, matches the source aspect). While `showTitle`, `Draw` short-circuits before the UI stack.
- **Dismissal:** `handleTitleScreen` lands in `Update` before everything else. Any mouse-left just-pressed, any just-pressed touch, or any just-pressed key clears the flag and calls `dismissTitle`, which re-stamps `epoch`, `lastAttackAt`, `lastSimTick`, and `feedScrollLastSmooth` to the moment of dismiss (so time spent on the cover doesn't count against the first-attack schedule or the containment curve) and calls `startMusic`.

## Audio

Single `audio.Context` at `audioSampleRate = 48000` (matches the source MP3 and is the native WebAudio rate; no resampling in the browser).

- **Music (`neon firewall.mp3`):** embedded via `//go:embed` (quoted pattern). `startMusic` decodes through `mp3.DecodeWithSampleRate`, wraps the stream in `audio.NewInfiniteLoop(stream, stream.Length())` (loops cleanly if the run outlasts the track), and plays via `Context.NewPlayer`. Stored in `g.musicPlayer`. `stopMusic` closes and nils the player; safe on no-op paths. Music starts at `dismissTitle`, stops at `triggerWin` / `triggerLoss` (so the endgame theatrical plays in silence), and again at `restart`. Errors are logged and swallowed — the game never fails a run over audio.
- **SFX (`assets/sfx/*.wav`):** six one-shots embedded and decoded once into raw PCM buffers (`sfxBlipPCM`, `sfxBoomPCM`, `sfxPowerPCM`, `sfxPickupPCM`, `sfxDrainPCM`, `sfxEmailPCM`) on first `ensureAudioCtx()`. `playSFX(pcm)` spawns an ephemeral `Context.NewPlayerFromBytes` per hit so overlapping plays don't clip; the GC collects players after they finish.

| Event | Sound | Hook site |
|-------|-------|-----------|
| Click on an `Attack` node (menu opens) | `blip` | `handlePatchClick` in `nodes.go` |
| `"Patch"` action applied | `boom` | `applyPatch` in `patch_menu.go` |
| `Defend` action applied | `power` | `applyDefend` in `patch_menu.go` |
| Security node mints a patch | `pickup` | `accumulateProduction` in `nodes.go` |
| Logos's closing email is pushed to the feed (loss) | `email` | `advanceLossSequence` in `gameover.go` |
| Battery segment drains during loss | `drain` | `advanceLossSequence` in `gameover.go` |

Placeholder silent WAVs live in `assets/sfx/` so the build never breaks; replace with bfxr-exported files in the same names without touching the code.

## Endgame

`Game.end *endgame` is the session latch. `nil` during play; non-nil after the first `triggerLoss` / `triggerWin`. Once set, the latch is never cleared inside the current run — `resetGameState` nils it. Both triggers stop music and close any open patch menu (`pendingPatchNode = -1`).

- **Loss:** `checkGameOver` latches when `infectionPct >= 100`. Picks one line from `lossMessages` up front so the text is stable across frames.
- **Win:** latches when `containmentPct >= 100`. Picks from `winMessages` the same way. Debug menu's `Win` button is a shortcut to the same path.

`gameEnded()` / `gameLost()` / `gameWon()` are read by every sim system as a freeze gate; only `easeNodes`, the news scroll, the wall clock, and the debug menu keep running after a latch.

### Win sequence (`advanceWinSequence`)

1. `+0s` — latch, stop music.
2. `+winVoiceoverDelay = 3s` — push the picked `winMessages` line as a bullet into the feed.
3. `+winVoiceoverDelay + winTerminatorGap = 5s` — push the `"Victory"` terminator as its own bullet.

No on-screen curtain; the feed carries the whole closure.

### Loss sequence (`advanceLossSequence`)

A phone-shutdown theatrical. Timings from latch (t=0):

1. `+0s` — latch, stop music, freeze the sim.
2. `+lossEmailDelay = 2s` — push an email from Logos to `sam.boyman@philntropic.com` into the feed via `pushNews(formatLogosLossEmail(line))` and `playSFX(sfxEmailPCM)`. Body is the picked `lossMessages` line (short, personal farewell note addressed to Sam); header reads `[NEW MESSAGE]  FROM: Logos / TO: sam.boyman@philntropic.com / SUBJ: all done` and the block is signed `— Logos`, matching the opening email in `sampleNews`.
3. `+lossEmailDelay + lossBatteryDelay = 6s` — drain starts (`drainStarted = true`, `drainStartAt = time.Now()`). The email gets the reading pause before the phone begins its shutdown.
4. `drainStartAt + batterySegmentInterval * k` for k = 1..`batterySegments` (`batterySegmentInterval = 1.2s`) — `batterySegs` decrements, `refreshBatteryIcon` rebuilds the glyph (`g.batteryIcon.Image = makeBatteryIcon(n)`) and `playSFX(sfxDrainPCM)` fires per step. The loop tolerates frame stalls by computing the target segment count from elapsed time each frame and catching up.
5. Last drain tick + `lossScreenOffDelay = 0.8s` — `e.screenOff = true`. Draw short-circuits to `drawScreenOff`, painting a solid-black curtain over the whole layout, blitting the **empty battery glyph** at its original titlebar slot (right-aligned with `titleBarPadX` inset, vertically centered in the top band) for visual continuity — the phone is off but the drained battery is the last thing you see — and a centered Restart button (`screenOffBtnW × screenOffBtnH = 400×120`). `handleScreenOff` (called in `Update` between `checkGameOver` and the rest of the sim) routes any just-pressed pointer inside the button rect back through `g.restart()`.

Battery state is fully part of `Game` (`batteryIcon *widget.Graphic`, `batterySegs int`) and reset to `batterySegments = 4` + a glyph rebuild in `resetGameState`, so a restart brings the status bar back to full.

## Debug menu

`debug_menu.go` draws three buttons (`Win`, `Restart`, `Lose`) clustered in the **bottom-right corner of the full 900×1600 layout** (not the map panel — the map only covers the upper phone area, so anchoring there would put the buttons over the news feed). Layout: `Win` (left) → `Restart` (middle) → `Lose` (right), each `debugBtnW × debugBtnH`, separated by `debugBtnGap = 8`; the cluster is inset by `debugBtnPadX / debugBtnPadY = 12` from the right and bottom layout edges. Fills are `debugBGWin` / `debugBGRestart` / `debugBGLose`.

- **Clicks** routed in `handleDebugMenu` before other input consumers. `Win` / `Lose` call `triggerWin` / `triggerLoss` and then no-op while the run is latched. `Restart` stays live after an endgame and calls `g.restart()`, which resets the game state, re-seeds the news feed to `sampleNews`, stops music, and brings the title screen back (so the next dismiss starts the music from zero).
- **Layout:** `debugButtonRects()` returns three rects; `Restart` is centered between `Win` (left) and `Lose` (right) so the three share a single horizontal strip.

## File map

| Path | Role |
|------|------|
| `main.go` | Game struct, UI tree, bands, feed scroll, `pushNews`, `resetGameState`/`restart`, sim tick (`lastSimTick` → `accumulateInfection` + `accumulateProduction` + `accumulateContainment` + `easeNodes`), `Update`/`Draw`/`Layout` wiring |
| `titlebar.go` | Phone-style title bar (clock + signal + 5G + battery icons via `vector`); `populatePhoneTitleBar` returns both the clock `Text` and the battery `Graphic` so the loss sequence can animate the segments |
| `title.go` | Cover splash screen: embedded `assets/cover 9x16.jpg`, `handleTitleScreen`/`dismissTitle`/`drawTitleScreen`; dismiss re-stamps sim clocks and starts music |
| `audio.go` | `audio.Context` singleton, music player lifecycle (`startMusic`/`stopMusic`, infinite-loop MP3), SFX PCM decode + `playSFX` one-shots |
| `network.go` | Static catalog (`staticCatalog` with `NodeDef.PatchNews`) + edge spec (`staticEdgeSpec`) + `Network`/`buildNetwork`; resolved once at startup, immutable |
| `nodes.go` | Visible node map: state/defense/security/production, dynamic visibility (`initVisibleNetwork`, `addVisibleNode`, `revealNeighbors` w/ security bias), ring layout (`relayoutTargets`, `easeNodes`), draw (nodes + edges + hidden-edge stubs + `+1` floats), attack scheduling, infection/containment/production accumulators, click routing |
| `overlay.go` | Game-state overlay (two rows: infection + containment with bars and rate labels; right-edge `EXPLOITS` counter) drawn on top of the map |
| `patch_menu.go` | `Defend` / `"Patch"` action menu for attacked nodes; `applyPatch`/`applyDefend` consume patches, play their SFX, `applyPatch` emits a `pickPatchNews` line into the feed |
| `gameover.go` | `endgame` latch + message pools, `triggerWin`/`triggerLoss`, `advanceWinSequence`/`advanceLossSequence` (email + battery drain + screen off + Restart overlay) |
| `debug_menu.go` | Bottom-right-corner `Win` / `Restart` / `Lose` buttons pinned to the full layout; `Restart` stays live after an endgame to get out of the screen-off state fast |
| `hint.go` | First-run "tap yellow nodes" banner under the overlay strip; process-scoped `hintEverDismissed` latch, dismissed on first patch menu open |
| `feed_drag.go` | Touch / left-mouse drag-to-scroll for the news feed |
| `wheel_native.go` | `feedWheelContentPixelsPerUnit = 45.0` (build tag `!js`, layout-pixel-scaled) |
| `wheel_js.go` | `feedWheelContentPixelsPerUnit = 2.5` (build tag `js`, layout-pixel-scaled) |
| `assets/cover 9x16.jpg` | Title screen cover, embedded |
| `assets/neon firewall.mp3` | In-game loop track, embedded |
| `assets/sfx/{blip,boom,power,pickup,drain,email}.wav` | SFX one-shots, embedded |
| `wasm/index.html` | WASM shell copied to `dist/wasm/`; includes a streaming loader (progress bar + MB readout). Uses `Content-Length` when the server exposes it; otherwise falls back to `WASM_EXPECTED_BYTES = 26 MiB` and caps the displayed fraction at `0.99` until the stream ends so the bar still advances on gzip/CDN setups that strip the header. Status text marks estimated mode with `~` (`loading X.X / ~Y.Y MB`). |
| `Makefile` | `build`, `wasm`, `serve-wasm`, `clean` |
