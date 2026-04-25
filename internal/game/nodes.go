package game

import (
	"image"
	"image/color"
	"math"
	"math/rand/v2"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	nodeRadius      = 28
	nodeStrokeW     = 5
	edgeStrokeW     = 3
	nodeLabelGapY   = 8 // px between node circle bottom and label top
	mapPaddingPx    = 25
	mapLabelFontPt  = 23
	mapOuterRingRel = 0.36 // ring radius relative to min(innerW, innerH)/2

	// Hidden-edge stubs: for each outer-ring node, every static-graph neighbour that
	// hasn't been revealed yet gets a short outward line segment. Visual cue that the
	// network extends beyond the current frontier. Fan the stubs symmetrically around
	// each node's radial-outward direction within stubFanAngle so multi-connection
	// nodes don't stack a single thick line.
	hiddenStubLenPx  = 40
	hiddenStubFanDeg = 40 // total angular spread for >1 stubs, in degrees

	attackInterval       = 3 * time.Second
	attackBlinkPeriodSec = 0.6
	attackBlinkMinAlpha  = 0.25 // floor of the pulse so the node stays visible

	// Per-node defense window: time the node spends in Attack before flipping to Infected
	// unless the player patches it. Values are sampled per-node from [defenseMin, defenseMax].
	defenseMin = 5 * time.Second
	defenseMax = 15 * time.Second

	// Infection-scale accumulation: each Infected node adds a one-time bump on transition
	// and a continuous drip while it remains infected. Total scale is clamped to [0, 100].
	infectionOneShotPct = 0.5 // %, added once when a node flips Infected
	infectionRatePerSec = 0.1 // % per infected node per second

	// Containment is the win-side counter. The rate starts at containmentRatePerSec and
	// compounds by containmentRateGrowthPerSec (relative) every second of live play, so
	// the blue team gets exponentially more efficient the longer they hold ground:
	//
	//   rate(t) = containmentRatePerSec * (1 + containmentRateGrowthPerSec)^t
	//
	// where t is seconds since g.epoch (frozen on game-end). At 100% the player wins
	// (Phil&Tropic's blue team finishes recapturing Logos). With 0.5%/s base and 1%/s
	// compounding the curve crosses 100% near ~2:00 of unimpeded play.
	containmentRatePerSec       = 0.5
	containmentRateGrowthPerSec = 0.01

	// Patch production: every patchProductionInterval, each Security node currently in
	// Normal state contributes +1 patch to the player's inventory.
	patchProductionInterval = 15 * time.Second

	// Visual badge for Security nodes: inner concentric ring drawn over the state fill.
	securityInnerRadius = 13
	securityRingW       = 3
	// Production-progress sector that fills inside the security ring; sweep angle =
	// elapsed/patchProductionInterval * 360° (starting at 12 o'clock, clockwise). Radius
	// sits just inside the ring so a thin gap stays visible at the rim.
	securityPieRadius = 10

	// Layered topology: hub at center, Infected nodes (other than the hub) condense onto
	// an inner ring, and everything else (Normal / Attack / Patched) lives on the outer
	// ring. mapOuterRingRel doubles as outerRingRadius for the visible mid layer.
	outerRingRadius = mapOuterRingRel
	innerRingRadius = 0.18

	// Reveal cap per Attack→Infected transition: up to this many previously hidden
	// neighbours fan out from the captured node onto the outer ring.
	revealMaxNeighbors = 3

	// Exponential easing rate for Node.X/Y → Node.TargetX/Y (per second). λ=4 settles
	// to ~half in 0.17 s, ~95% in 0.75 s — quick but visibly smooth.
	nodeEaseLambda = 4.0

	// "+1" float-ups above a Security node when it mints a patch. Rise is an ease-out
	// on the normalized lifetime; alpha fades linearly over the whole duration so the
	// tail is already translucent by the time it reaches its peak travel.
	patchFloatDuration = 1.2 // seconds
	patchFloatRisePx   = 70  // total upward travel at t=1
	patchFloatMarginPx = 10  // start above the node circle's top edge
)

// patchFloat is one live "+1" animation. spawnedAt is wall time so the animation
// driver doesn't need a per-frame dt; the cull happens in drawPatchFloats.
type patchFloat struct {
	visIdx    int
	spawnedAt time.Time
}

var patchFloatColor = color.NRGBA{R: 0xa8, G: 0xff, B: 0xf0, A: 0xff}

// NodeState mirrors the four states from CONCEPT (Норма / Атака / Заражен / Пропатчен).
// Only Normal is rendered today; the rest are reserved for the core loop.
type NodeState int

const (
	NodeStateNormal NodeState = iota
	NodeStateAttack
	NodeStateInfected
	NodeStatePatched
)

// Node is the per-frame state of a visible network member. Coordinates are normalized
// [0..1] inside the inner map area; the hub at index 0 sits at (0.5, 0.5).
//
// Position is double-buffered: drawing reads X/Y, while game logic writes TargetX/TargetY
// (e.g. relayoutTargets after a reveal). easeNodes runs every frame to interpolate the
// pair, so any structural change to the visible set produces a smooth animated transition
// rather than a jump.
type Node struct {
	Name  string
	State NodeState
	// Security marks nodes whose business is defense (e.g. CrowdStrike, Cisco). While in
	// Normal state they accumulate ProductionElapsed; once it crosses
	// patchProductionInterval the node mints +1 patch and the accumulator wraps.
	Security          bool
	ProductionElapsed time.Duration
	// Defense is the dwell time in Attack before progressAttacks flips this node to Infected.
	// Sampled once at network creation; Phil&Tropic (already infected) leaves this zero.
	Defense    time.Duration
	AttackedAt time.Time // set when State transitions Normal -> Attack

	// DefIdx points back into Network.Defs; revealNeighbors uses it to look up the static
	// adjacency list for the node and decide which catalog entries to surface next.
	DefIdx int

	// Current and target normalized positions; eased every frame by easeNodes.
	X, Y             float64
	TargetX, TargetY float64
}

// Edge connects two node indices.
type Edge struct{ From, To int }

var (
	nodeFillNormal     = color.NRGBA{R: 0x6e, G: 0x70, B: 0x76, A: 0xff}
	nodeStrokeNormal   = color.NRGBA{R: 0xb8, G: 0xba, B: 0xc0, A: 0xff}
	nodeFillAttack     = color.NRGBA{R: 0xe6, G: 0xc8, B: 0x3a, A: 0xff}
	nodeStrokeAttack   = color.NRGBA{R: 0xff, G: 0xe8, B: 0x70, A: 0xff}
	nodeFillInfected   = color.NRGBA{R: 0xc0, G: 0x30, B: 0x30, A: 0xff}
	nodeStrokeInfect   = color.NRGBA{R: 0xff, G: 0x60, B: 0x60, A: 0xff}
	// Panopticon palette: infected-red morphs into amber/orange on victory to signal
	// that Project Panopticon has absorbed the network. Drawn via lerpColor with the
	// current win-morph progress, so nothing changes outside the win sequence.
	nodeFillPanopticon   = color.NRGBA{R: 0xc8, G: 0x80, B: 0x1a, A: 0xff}
	nodeStrokePanopticon = color.NRGBA{R: 0xff, G: 0xb8, B: 0x48, A: 0xff}
	nodeFillPatched    = color.NRGBA{R: 0x10, G: 0x10, B: 0x12, A: 0xff}
	nodeStrokePatch    = color.NRGBA{R: 0x55, G: 0x55, B: 0x5a, A: 0xff}
	edgeColor          = color.NRGBA{R: 0x40, G: 0x42, B: 0x48, A: 0xff}
	edgeSecurityColor  = color.NRGBA{R: 0x2e, G: 0x82, B: 0x7c, A: 0xff}
	labelColor         = color.NRGBA{R: 0xc8, G: 0xca, B: 0xd0, A: 0xff}
	securityRingFG     = color.NRGBA{R: 0x40, G: 0xc8, B: 0xc0, A: 0xff}
	securityProgressFG = color.NRGBA{R: 0xa8, G: 0xff, B: 0xf0, A: 0xff}
)

// initialRingNames is the canonical Project Panopticon ring shown at game start, in
// clockwise order from the 12-o'clock slot. Subsequent reveals splice new entries into
// the same outerRing slice, preserving angular continuity.
var initialRingNames = []string{
	"Sahara WS",
	"MacroFrame",
	"Giggle",
	"LeatherJacket",
	"Dongle",
	"BootLoop",
	"Fiasco Sys",
	"Monolith Foundation",
	"BroadCon",
	"GPMidas",
}

// initVisibleNetwork resolves the static catalog into a Network, populates the initial
// visible set (hub + Alliance ring), seeds the per-ring slot lists, and writes initial
// positions = target positions (no animation on first frame). Called once from newGame.
func (g *Game) initVisibleNetwork() {
	g.network = buildNetwork()
	g.visibleByDef = make(map[int]int, len(g.network.Defs))
	g.nodes = g.nodes[:0]
	g.edges = g.edges[:0]
	g.outerRing = g.outerRing[:0]
	g.innerRing = g.innerRing[:0]

	hubIdx, ok := g.network.NameToIdx["Phil&Tropic"]
	if !ok {
		return
	}
	g.addVisibleNode(hubIdx, 0.5, 0.5)
	g.nodes[0].State = NodeStateInfected
	g.nodes[0].TargetX, g.nodes[0].TargetY = 0.5, 0.5

	for _, name := range initialRingNames {
		defIdx, ok := g.network.NameToIdx[name]
		if !ok {
			continue
		}
		visIdx := g.addVisibleNode(defIdx, 0.5, 0.5)
		g.outerRing = append(g.outerRing, visIdx)
	}

	g.relayoutTargets()
	for i := range g.nodes {
		g.nodes[i].X = g.nodes[i].TargetX
		g.nodes[i].Y = g.nodes[i].TargetY
	}
}

// addVisibleNode appends a fresh Node sourced from network.Defs[defIdx], placing it at
// (x, y) with target = (x, y). It also creates Edge entries to every already-visible
// neighbour per the static graph, so connectivity stays in sync as the visible set grows.
// Returns the new visible index.
func (g *Game) addVisibleNode(defIdx int, x, y float64) int {
	def := g.network.Defs[defIdx]
	n := Node{
		Name:     def.Name,
		Security: def.Security,
		DefIdx:   defIdx,
		X:        x,
		Y:        y,
		TargetX:  x,
		TargetY:  y,
	}
	if def.Name != "Phil&Tropic" {
		n.Defense = defenseMin + rand.N(defenseMax-defenseMin)
	}
	g.nodes = append(g.nodes, n)
	visIdx := len(g.nodes) - 1
	g.visibleByDef[defIdx] = visIdx

	for _, neighborDefIdx := range g.network.Adj[defIdx] {
		if neighborVisIdx, ok := g.visibleByDef[neighborDefIdx]; ok && neighborVisIdx != visIdx {
			g.edges = append(g.edges, Edge{From: visIdx, To: neighborVisIdx})
		}
	}
	return visIdx
}

// producingSecurityCount returns the number of currently visible Security nodes in
// Normal state — i.e. nodes that are actively accumulating toward the next patch.
// Used by revealNeighbors to decide whether to bias the reveal toward hidden security.
func (g *Game) producingSecurityCount() int {
	n := 0
	for i := range g.nodes {
		if g.nodes[i].Security && g.nodes[i].State == NodeStateNormal {
			n++
		}
	}
	return n
}

// revealNeighbors is called once a node finishes the Attack→Infected transition. It
//
//  1. Pulls the node's slot out of outerRing and appends it to innerRing (the captured
//     node migrates inward toward the hub).
//  2. Picks up to revealMaxNeighbors previously hidden static neighbours at random and
//     splices them into outerRing at the just-vacated slot, so the new nodes appear
//     in place of the parent and inherit its starting position before easing outward.
//  3. Recomputes every visible node's TargetX/Y via relayoutTargets.
//
// No-op if the catalog has no hidden neighbours left for this node.
func (g *Game) revealNeighbors(parentVisIdx int) {
	if parentVisIdx < 0 || parentVisIdx >= len(g.nodes) {
		return
	}
	parentDefIdx := g.nodes[parentVisIdx].DefIdx
	parentX, parentY := g.nodes[parentVisIdx].X, g.nodes[parentVisIdx].Y

	outerPos := indexOfInt(g.outerRing, parentVisIdx)
	if outerPos >= 0 {
		g.outerRing = append(g.outerRing[:outerPos], g.outerRing[outerPos+1:]...)
	}
	if !containsInt(g.innerRing, parentVisIdx) {
		g.innerRing = append(g.innerRing, parentVisIdx)
	}

	hidden := make([]int, 0, len(g.network.Adj[parentDefIdx]))
	for _, defIdx := range g.network.Adj[parentDefIdx] {
		if _, ok := g.visibleByDef[defIdx]; !ok {
			hidden = append(hidden, defIdx)
		}
	}
	rand.Shuffle(len(hidden), func(i, j int) { hidden[i], hidden[j] = hidden[j], hidden[i] })

	// Security bias: when the player's productive security pool runs thin, reorder
	// the shuffled hidden list so security nodes get priority in the revealMaxNeighbors
	// slice. Without this the RNG can starve patch production after a bad cascade and
	// make the game technically unwinnable. Threshold (<=1 producing security) keeps
	// the bias dormant in the normal early game where both starting security are up.
	if g.producingSecurityCount() <= 1 {
		sec := hidden[:0:0]
		other := make([]int, 0, len(hidden))
		for _, defIdx := range hidden {
			if g.network.Defs[defIdx].Security {
				sec = append(sec, defIdx)
			} else {
				other = append(other, defIdx)
			}
		}
		hidden = append(sec, other...)
	}

	if len(hidden) > revealMaxNeighbors {
		hidden = hidden[:revealMaxNeighbors]
	}

	insertAt := outerPos
	if insertAt < 0 || insertAt > len(g.outerRing) {
		insertAt = len(g.outerRing)
	}
	for _, defIdx := range hidden {
		visIdx := g.addVisibleNode(defIdx, parentX, parentY)
		g.outerRing = append(g.outerRing, 0)
		copy(g.outerRing[insertAt+1:], g.outerRing[insertAt:])
		g.outerRing[insertAt] = visIdx
		insertAt++
	}

	g.relayoutTargets()
}

// relayoutTargets recomputes TargetX/Y for every visible node by walking the outer and
// inner ring slot lists in order and mapping each slot to a polar position. The hub is
// pinned at (0.5, 0.5). Patched / Normal / Attack / non-hub Infected do not need special
// handling here — their slot membership in outerRing vs innerRing already encodes intent.
func (g *Game) relayoutTargets() {
	assignRing := func(ring []int, radius float64) {
		n := len(ring)
		if n == 0 {
			return
		}
		for i, visIdx := range ring {
			angle := -math.Pi/2 + 2*math.Pi*float64(i)/float64(n)
			g.nodes[visIdx].TargetX = 0.5 + radius*math.Cos(angle)
			g.nodes[visIdx].TargetY = 0.5 + radius*math.Sin(angle)
		}
	}
	assignRing(g.outerRing, outerRingRadius)
	assignRing(g.innerRing, innerRingRadius)
	// Hub pin runs last so a stray ring assignment can never overwrite (0.5, 0.5).
	for i := range g.nodes {
		if g.nodes[i].Name == "Phil&Tropic" {
			g.nodes[i].TargetX, g.nodes[i].TargetY = 0.5, 0.5
		}
	}
}

// easeNodes interpolates X/Y toward TargetX/Y using a per-second exponential decay
// (k = 1 - exp(-λ·dt)). Same shape as stepSmoothFeedScroll's easing so the feel matches
// across the app. dt clamped to [0, 0.2] to absorb stalls / first-frame negative deltas.
func (g *Game) easeNodes(dt time.Duration) {
	secs := dt.Seconds()
	if secs <= 0 {
		return
	}
	if secs > 0.2 {
		secs = 0.2
	}
	k := 1.0 - math.Exp(-nodeEaseLambda*secs)
	for i := range g.nodes {
		g.nodes[i].X += (g.nodes[i].TargetX - g.nodes[i].X) * k
		g.nodes[i].Y += (g.nodes[i].TargetY - g.nodes[i].Y) * k
	}
}

func indexOfInt(s []int, v int) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

func containsInt(s []int, v int) bool {
	return indexOfInt(s, v) >= 0
}

// drawNodeMap renders edges, node circles, and labels onto screen, clipped
// (visually) to the mapPanel's laid-out rectangle.
func (g *Game) drawNodeMap(screen *ebiten.Image) {
	if g.mapPanel == nil {
		return
	}
	rect := g.mapPanel.GetWidget().Rect
	if rect.Empty() {
		return
	}

	innerX := float64(rect.Min.X + mapPaddingPx)
	innerY := float64(rect.Min.Y + mapPaddingPx)
	innerW := float64(rect.Dx() - 2*mapPaddingPx)
	innerH := float64(rect.Dy() - 2*mapPaddingPx)
	if innerW <= 0 || innerH <= 0 {
		return
	}

	pos := func(n Node) (float32, float32) {
		return float32(innerX + n.X*innerW), float32(innerY + n.Y*innerH)
	}

	for _, e := range g.edges {
		x1, y1 := pos(g.nodes[e.From])
		x2, y2 := pos(g.nodes[e.To])
		vector.StrokeLine(screen, x1, y1, x2, y2, edgeStrokeW, edgeColor, true)
	}

	centerX := float32(innerX + 0.5*innerW)
	centerY := float32(innerY + 0.5*innerH)
	g.drawHiddenEdgeStubs(screen, pos, centerX, centerY)

	frozen := g.gameEnded()
	morphT := g.winMorphProgress()
	for _, n := range g.nodes {
		x, y := pos(n)
		fill, stroke := nodeColors(n.State)
		// Freeze the attack pulse on endgame: the node stays at full alpha so the
		// player sees the world halt mid-tick rather than continuing to wink at them.
		if n.State == NodeStateAttack && !frozen {
			fill = scaleAlpha(fill, attackBlinkAlpha(time.Since(g.epoch)))
		}
		// Panopticon morph (victory only): infected nodes ease from red to amber as
		// the final reveal plays out; other states are unaffected.
		if n.State == NodeStateInfected && morphT > 0 {
			fill = lerpColor(fill, nodeFillPanopticon, morphT)
			stroke = lerpColor(stroke, nodeStrokePanopticon, morphT)
		}
		vector.FillCircle(screen, x, y, nodeRadius, fill, true)
		vector.StrokeCircle(screen, x, y, nodeRadius, nodeStrokeW, stroke, true)
		if n.Security {
			vector.StrokeCircle(screen, x, y, securityInnerRadius, securityRingW, securityRingFG, true)
			// Pie also stays visible during Attack so the player can see the timer is frozen
			// (accumulateProduction skips non-Normal states, so progress doesn't advance).
			if n.State == NodeStateNormal || n.State == NodeStateAttack {
				progress := float64(n.ProductionElapsed) / float64(patchProductionInterval)
				if progress < 0 {
					progress = 0
				}
				if progress > 1 {
					progress = 1
				}
				fillPieSector(screen, x, y, securityPieRadius, -math.Pi/2, 2*math.Pi*progress, securityProgressFG)
			}
		}
	}

	if g.mapLabelFace != nil {
		for _, n := range g.nodes {
			x, y := pos(n)
			drawCenteredLabel(screen, g.mapLabelFace, n.Name,
				float64(x), float64(y)+nodeRadius+nodeLabelGapY, labelColor)
		}
	}
}

// drawPatchFloats renders every live "+1" patch-production animation above its node
// and culls expired ones in the same pass. Rise uses an ease-out curve (1 - (1-t)^2)
// so the "+1" lifts off quickly and glides to a stop; alpha fades linearly over the
// whole lifetime. Called from Game.Draw after drawNodeMap so the float sits on top of
// nodes/labels but below the game overlay (which owns the top of the map strip).
func (g *Game) drawPatchFloats(screen *ebiten.Image) {
	if g.overlayValueFace == nil || len(g.patchFloats) == 0 {
		return
	}
	now := time.Now()
	kept := g.patchFloats[:0]
	for _, f := range g.patchFloats {
		elapsed := now.Sub(f.spawnedAt).Seconds()
		if elapsed >= patchFloatDuration {
			continue
		}
		cx, cy, ok := g.nodeScreenPos(f.visIdx)
		if !ok {
			kept = append(kept, f)
			continue
		}
		t := elapsed / patchFloatDuration
		ease := 1 - (1-t)*(1-t)
		y := float64(cy) - float64(nodeRadius) - patchFloatMarginPx - patchFloatRisePx*ease
		c := patchFloatColor
		c.A = uint8(float64(c.A) * (1 - t))
		drawAlignedText(screen, g.overlayValueFace, "+1",
			float64(cx), y, text.AlignCenter, text.AlignCenter, c)
		kept = append(kept, f)
	}
	g.patchFloats = kept
}

// drawHiddenEdgeStubs emits a short outward line segment for every static-graph edge
// that leaves an outer-ring node toward a neighbour that is not currently visible.
// Stubs fan around the node's radial-outward direction so a node with several hidden
// neighbours grows a small "antenna bundle" rather than one over-thick line. Only
// outer-ring members are considered: the hub has no hidden neighbours, and inner-ring
// (captured) nodes would shoot stubs through the outer ring, which reads wrong.
func (g *Game) drawHiddenEdgeStubs(screen *ebiten.Image,
	pos func(Node) (float32, float32), centerX, centerY float32,
) {
	if g.network == nil {
		return
	}
	fan := float64(hiddenStubFanDeg) * math.Pi / 180.0
	// Two-pass: render non-security stubs first, then security stubs on top so the
	// teal highlight survives visual overlap with neutral stubs of the same node.
	type stub struct{ sx, sy, ex, ey float32 }
	var neutral, secure []stub
	var hiddenDefs []int
	for _, visIdx := range g.outerRing {
		n := g.nodes[visIdx]
		hiddenDefs = hiddenDefs[:0]
		for _, neighDefIdx := range g.network.Adj[n.DefIdx] {
			if _, ok := g.visibleByDef[neighDefIdx]; !ok {
				hiddenDefs = append(hiddenDefs, neighDefIdx)
			}
		}
		if len(hiddenDefs) == 0 {
			continue
		}
		cx, cy := pos(n)
		dx := float64(cx - centerX)
		dy := float64(cy - centerY)
		if dx*dx+dy*dy < 1e-6 {
			continue
		}
		baseAngle := math.Atan2(dy, dx)
		count := len(hiddenDefs)
		for k, defIdx := range hiddenDefs {
			offset := 0.0
			if count > 1 {
				offset = (float64(k)/float64(count-1) - 0.5) * fan
			}
			a := baseAngle + offset
			ca, sa := math.Cos(a), math.Sin(a)
			s := stub{
				sx: cx + float32(ca)*nodeRadius,
				sy: cy + float32(sa)*nodeRadius,
				ex: cx + float32(ca)*(nodeRadius+hiddenStubLenPx),
				ey: cy + float32(sa)*(nodeRadius+hiddenStubLenPx),
			}
			if g.network.Defs[defIdx].Security {
				secure = append(secure, s)
			} else {
				neutral = append(neutral, s)
			}
		}
	}
	for _, s := range neutral {
		vector.StrokeLine(screen, s.sx, s.sy, s.ex, s.ey, edgeStrokeW, edgeColor, true)
	}
	for _, s := range secure {
		vector.StrokeLine(screen, s.sx, s.sy, s.ex, s.ey, edgeStrokeW, edgeSecurityColor, true)
	}
}

// nodeColors maps a state to (fill, stroke). Attack callers further modulate fill alpha
// via attackBlinkAlpha to produce the yellow pulse.
func nodeColors(s NodeState) (fill, stroke color.NRGBA) {
	switch s {
	case NodeStateAttack:
		return nodeFillAttack, nodeStrokeAttack
	case NodeStateInfected:
		return nodeFillInfected, nodeStrokeInfect
	case NodeStatePatched:
		return nodeFillPatched, nodeStrokePatch
	default:
		return nodeFillNormal, nodeStrokeNormal
	}
}

// attackBlinkAlpha returns a [attackBlinkMinAlpha .. 1] pulse driven by the wall clock,
// so all attacking nodes blink in phase regardless of when each one entered Attack.
func attackBlinkAlpha(t time.Duration) float64 {
	phase := math.Sin(2 * math.Pi * t.Seconds() / attackBlinkPeriodSec)
	k := 0.5 + 0.5*phase
	return attackBlinkMinAlpha + (1-attackBlinkMinAlpha)*k
}

func scaleAlpha(c color.NRGBA, a float64) color.NRGBA {
	if a < 0 {
		a = 0
	} else if a > 1 {
		a = 1
	}
	c.A = uint8(float64(c.A) * a)
	return c
}

// nodeScreenPos returns the on-screen center of node i in the current map layout,
// or ok=false if the map area is missing/degenerate.
func (g *Game) nodeScreenPos(i int) (cx, cy float32, ok bool) {
	if g.mapPanel == nil || i < 0 || i >= len(g.nodes) {
		return 0, 0, false
	}
	rect := g.mapPanel.GetWidget().Rect
	innerW := float64(rect.Dx() - 2*mapPaddingPx)
	innerH := float64(rect.Dy() - 2*mapPaddingPx)
	if innerW <= 0 || innerH <= 0 {
		return 0, 0, false
	}
	n := g.nodes[i]
	return float32(float64(rect.Min.X+mapPaddingPx) + n.X*innerW),
		float32(float64(rect.Min.Y+mapPaddingPx) + n.Y*innerH),
		true
}

// handlePatchClick routes just-pressed pointer events (mouse or touch) for the patch menu:
//
//  1. If a menu is already open and the player has patches, an in-button click triggers the
//     action (Defend or "Patch"); any other click cancels the menu (and may open a new one
//     if it lands on another Attack node).
//  2. If a menu is already open but patches == 0, the buttons are inert (rendered dimmed by
//     drawPatchMenu) and any click just closes the menu.
//  3. Otherwise a click on an Attack node inside the map area opens the menu for that node —
//     regardless of patches, so the player can still inspect the situation with zero stock.
func (g *Game) handlePatchClick() {
	if g.mapPanel == nil || g.gameEnded() {
		return
	}
	if g.pendingPatchNode >= 0 {
		// Menu target may have flipped Infected (defense ran out) since last frame.
		if g.pendingPatchNode >= len(g.nodes) || g.nodes[g.pendingPatchNode].State != NodeStateAttack {
			g.pendingPatchNode = -1
		}
	}

	x, y, pressed := pollJustPressedPointer()
	if !pressed {
		return
	}

	if g.pendingPatchNode >= 0 {
		if g.patchesLeft > 0 {
			if defendR, patchR, ok := g.patchMenuLayout(g.pendingPatchNode); ok {
				pt := image.Pt(x, y)
				switch {
				case pt.In(defendR):
					g.applyDefend(g.pendingPatchNode)
					g.pendingPatchNode = -1
					return
				case pt.In(patchR):
					g.applyPatch(g.pendingPatchNode)
					g.pendingPatchNode = -1
					return
				}
			}
		}
		g.pendingPatchNode = -1
	}

	if !image.Pt(x, y).In(g.mapPanel.GetWidget().Rect) {
		return
	}
	if idx := g.attackNodeAt(x, y); idx >= 0 {
		g.pendingPatchNode = idx
		playSFX(sfxBlipPCM)
		dismissHint()
	}
}

// attackNodeAt returns the index of the Attack node whose hit disc covers (x, y), or -1.
// Hit radius is slightly inflated (hitSlackPx) for finger-friendly tapping on touch screens.
func (g *Game) attackNodeAt(x, y int) int {
	const hitSlackPx = 10
	rSq := float64(nodeRadius+hitSlackPx) * float64(nodeRadius+hitSlackPx)
	for i := range g.nodes {
		if g.nodes[i].State != NodeStateAttack {
			continue
		}
		cx, cy, ok := g.nodeScreenPos(i)
		if !ok {
			continue
		}
		dx := float64(x) - float64(cx)
		dy := float64(y) - float64(cy)
		if dx*dx+dy*dy <= rSq {
			return i
		}
	}
	return -1
}

// pollJustPressedPointer returns the position of the most recent just-pressed pointer
// (touch first to match mobile-priority), or pressed=false if no new press this frame.
func pollJustPressedPointer() (x, y int, pressed bool) {
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y = ebiten.TouchPosition(id)
		return x, y, true
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y = ebiten.CursorPosition()
		return x, y, true
	}
	return 0, 0, false
}

// attackTick promotes one Normal neighbour of any Infected node to Attack, if any are left.
// Called from Update on attackInterval cadence.
func (g *Game) attackTick() {
	frontier := infectedFrontier(g.nodes, g.edges)
	if len(frontier) == 0 {
		return
	}
	pick := frontier[rand.IntN(len(frontier))]
	g.nodes[pick].State = NodeStateAttack
	g.nodes[pick].AttackedAt = time.Now()
}

// initialInfectionPct returns the infection scale implied by nodes already Infected at
// game start: each such node is credited with its one-time +infectionOneShotPct, same
// as nodes that flip during play via progressAttacks.
func initialInfectionPct(nodes []Node) float64 {
	pct := 0.0
	for _, n := range nodes {
		if n.State == NodeStateInfected {
			pct += infectionOneShotPct
		}
	}
	return pct
}

// progressAttacks flips any Attack node whose Defense window has elapsed to Infected,
// crediting the one-time infectionOneShotPct bump and triggering a reveal of up to
// revealMaxNeighbors previously hidden static neighbours. Runs every Update so capture
// timing is independent of attackInterval. Iteration uses range (snapshot length), so
// nodes appended by revealNeighbors are not re-visited within the same frame.
func (g *Game) progressAttacks() {
	now := time.Now()
	for i := range g.nodes {
		if g.nodes[i].State != NodeStateAttack {
			continue
		}
		if now.Sub(g.nodes[i].AttackedAt) >= g.nodes[i].Defense {
			g.nodes[i].State = NodeStateInfected
			g.infectionPct = clampInfection(g.infectionPct + infectionOneShotPct)
			g.revealNeighbors(i)
			playSFX(sfxInfectedPCM)
		}
	}
}

// accumulateInfection adds infectionRatePerSec * dt for every currently Infected node,
// producing the slow baseline climb that rewards letting patches stack up.
func (g *Game) accumulateInfection(dt time.Duration) {
	if dt <= 0 || g.infectionPct >= 100 {
		return
	}
	count := countInfected(g.nodes)
	if count == 0 {
		return
	}
	g.infectionPct = clampInfection(g.infectionPct + infectionRatePerSec*dt.Seconds()*float64(count))
}

// currentContainmentRate returns the instantaneous containment growth rate (% / s)
// at the current accumulated live-play time. Used by the overlay's "+R.R%/s" readout
// so the displayed number tracks what's actually being added to the bar — and freezes
// the moment the run ends, since g.containmentElapsed stops advancing then.
func (g *Game) currentContainmentRate() float64 {
	return containmentRatePerSec * math.Pow(1+containmentRateGrowthPerSec, g.containmentElapsed)
}

// accumulateContainment advances the win-side counter; on hitting 100% it triggers
// the win flow exactly once (triggerWin is idempotent if the player has already lost
// on the same frame). Caller (Update) gates this on !gameEnded() so both the counter
// and g.containmentElapsed freeze at whatever values they had when the run concluded.
//
// To stay accurate as the rate grows we integrate the analytic curve over the live
// interval [t0, t1] instead of using rate(t1) * dt:
//
//	∫ base * k^t dt = base * (k^t1 - k^t0) / ln(k)
//
// where k = 1 + containmentRateGrowthPerSec. Cheap, exact, and frame-rate independent.
// Tracking elapsed live-time on the Game (rather than time.Since(epoch)) is what makes
// the rate freeze on game-end: real wall time keeps moving, this counter doesn't.
func (g *Game) accumulateContainment(dt time.Duration) {
	if dt <= 0 || g.containmentPct >= 100 {
		return
	}
	t0 := g.containmentElapsed
	t1 := t0 + dt.Seconds()
	g.containmentElapsed = t1
	k := 1 + containmentRateGrowthPerSec
	delta := containmentRatePerSec * (math.Pow(k, t1) - math.Pow(k, t0)) / math.Log(k)
	g.containmentPct = clampInfection(g.containmentPct + delta)
	if g.containmentPct >= 100 {
		g.triggerWin()
	}
}

// countInfected returns the number of nodes currently in Infected state. Used both by
// the per-second drip and by the overlay's "+x.x%/s" rate readout.
func countInfected(nodes []Node) int {
	n := 0
	for i := range nodes {
		if nodes[i].State == NodeStateInfected {
			n++
		}
	}
	return n
}

// accumulateProduction advances each Security node's ProductionElapsed by dt while it is
// in Normal state and converts every full patchProductionInterval into +1 patch. Attack /
// Infected / Patched nodes are skipped, so the timer (and its visual pie sector) pauses
// during attacks; partial progress is preserved across Defend bounces.
func (g *Game) accumulateProduction(dt time.Duration) {
	if dt <= 0 {
		return
	}
	for i := range g.nodes {
		if !g.nodes[i].Security || g.nodes[i].State != NodeStateNormal {
			continue
		}
		g.nodes[i].ProductionElapsed += dt
		for g.nodes[i].ProductionElapsed >= patchProductionInterval {
			g.nodes[i].ProductionElapsed -= patchProductionInterval
			g.patchesLeft++
			g.patchFloats = append(g.patchFloats, patchFloat{visIdx: i, spawnedAt: time.Now()})
			playSFX(sfxPickupPCM)
		}
	}
}

func clampInfection(p float64) float64 {
	switch {
	case p < 0:
		return 0
	case p > 100:
		return 100
	default:
		return p
	}
}

// infectedFrontier returns indices of Normal nodes that share an edge with any Infected node.
// Duplicates removed; order is deterministic w.r.t. edge order, which keeps spread predictable.
func infectedFrontier(nodes []Node, edges []Edge) []int {
	seen := make(map[int]bool)
	out := make([]int, 0, len(edges))
	for _, e := range edges {
		var n int
		switch {
		case nodes[e.From].State == NodeStateInfected && nodes[e.To].State == NodeStateNormal:
			n = e.To
		case nodes[e.To].State == NodeStateInfected && nodes[e.From].State == NodeStateNormal:
			n = e.From
		default:
			continue
		}
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	return out
}

// fillPieSector fills a circular sector centered at (cx, cy) with radius r, starting at
// startAngle and sweeping clockwise (in screen coords) by sweep radians. Sweep <= 0 draws
// nothing; sweep >= 2π collapses to a full filled circle (avoiding degenerate Arc paths).
func fillPieSector(dst *ebiten.Image, cx, cy, r float32, startAngle, sweep float64, clr color.Color) {
	if sweep <= 0 {
		return
	}
	if sweep >= 2*math.Pi {
		vector.FillCircle(dst, cx, cy, r, clr, true)
		return
	}
	p := &vector.Path{}
	p.MoveTo(cx, cy)
	p.LineTo(cx+r*float32(math.Cos(startAngle)), cy+r*float32(math.Sin(startAngle)))
	p.Arc(cx, cy, r, float32(startAngle), float32(startAngle+sweep), vector.Clockwise)
	p.Close()
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(clr)
	vector.FillPath(dst, p, nil, op)
}

func drawCenteredLabel(dst *ebiten.Image, face text.Face, s string, cx, top float64, clr color.Color) {
	w, _ := text.Measure(s, face, 0)
	op := &text.DrawOptions{}
	op.GeoM.Translate(cx-w/2, top)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, s, face, op)
}
