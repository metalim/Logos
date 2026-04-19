package main

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

	attackInterval       = 3 * time.Second
	attackBlinkPeriodSec = 0.6
	attackBlinkMinAlpha  = 0.25 // floor of the pulse so the node stays visible

	// Per-node defense window: time the node spends in Attack before flipping to Infected
	// unless the player patches it. Values are sampled per-node from [defenseMin, defenseMax].
	defenseMin = 5 * time.Second
	defenseMax = 15 * time.Second

	// Infection-scale accumulation: each Infected node adds a one-time bump on transition
	// and a continuous drip while it remains infected. Total scale is clamped to [0, 100].
	infectionOneShotPct = 1.0 // %, added once when a node flips Infected
	infectionRatePerSec = 0.1 // % per infected node per second

	// Patch production: every patchProductionInterval, each Security node currently in
	// Normal state contributes +1 patch to the player's inventory.
	patchProductionInterval = 15 * time.Second

	// Visual badge for Security nodes: inner concentric ring drawn over the state fill.
	securityInnerRadius = 13
	securityRingW       = 3
)

// NodeState mirrors the four states from CONCEPT (Норма / Атака / Заражен / Пропатчен).
// Only Normal is rendered today; the rest are reserved for the core loop.
type NodeState int

const (
	NodeStateNormal NodeState = iota
	NodeStateAttack
	NodeStateInfected
	NodeStatePatched
)

// Node carries normalized coordinates [0..1] inside the inner map area.
// Hub at index 0 sits in the center; the rest are arranged on a single ring.
type Node struct {
	Name  string
	X, Y  float64
	State NodeState
	// Security marks nodes whose business is defense (e.g. CrowdStrike, Cisco). While in
	// Normal state they produce patches on the global production tick.
	Security bool
	// Defense is the dwell time in Attack before progressAttacks flips this node to Infected.
	// Sampled once at network creation; Phil&Tropic (already infected) leaves this zero.
	Defense    time.Duration
	AttackedAt time.Time // set when State transitions Normal -> Attack
}

// Edge connects two node indices.
type Edge struct{ From, To int }

var (
	nodeFillNormal   = color.NRGBA{R: 0x6e, G: 0x70, B: 0x76, A: 0xff}
	nodeStrokeNormal = color.NRGBA{R: 0xb8, G: 0xba, B: 0xc0, A: 0xff}
	nodeFillAttack   = color.NRGBA{R: 0xe6, G: 0xc8, B: 0x3a, A: 0xff}
	nodeStrokeAttack = color.NRGBA{R: 0xff, G: 0xe8, B: 0x70, A: 0xff}
	nodeFillInfected = color.NRGBA{R: 0xc0, G: 0x30, B: 0x30, A: 0xff}
	nodeStrokeInfect = color.NRGBA{R: 0xff, G: 0x60, B: 0x60, A: 0xff}
	nodeFillPatched  = color.NRGBA{R: 0x10, G: 0x10, B: 0x12, A: 0xff}
	nodeStrokePatch  = color.NRGBA{R: 0x55, G: 0x55, B: 0x5a, A: 0xff}
	edgeColor        = color.NRGBA{R: 0x40, G: 0x42, B: 0x48, A: 0xff}
	labelColor       = color.NRGBA{R: 0xc8, G: 0xca, B: 0xd0, A: 0xff}
	securityRingFG   = color.NRGBA{R: 0x40, G: 0xc8, B: 0xc0, A: 0xff}
)

// defaultNetwork builds the placeholder topology: the Phil&Tropic datacenter at the
// center (Logos's escape origin per CONCEPT) plus a ring of Project Panopticon nodes,
// including The Monolith Foundation (internet-core analogue, ≈ Linux Foundation).
// Star edges from the hub + a perimeter ring give the map a recognisable mesh look.
func defaultNetwork() ([]Node, []Edge) {
	const cx, cy = 0.5, 0.5

	ring := []struct {
		name     string
		security bool
	}{
		{"Sahara WS", false},           // AWS
		{"MacroFrame", false},          // Microsoft
		{"Giggle", false},              // Google
		{"LeatherJacket", false},       // NVIDIA
		{"Dongle", false},              // Apple
		{"BootLoop", true},             // CrowdStrike — security vendor
		{"Fiasco Sys", true},           // Cisco — networking + security
		{"Monolith Foundation", false}, // Linux Foundation
		{"BroadCon", false},            // Broadcom
		{"GPMidas", false},             // JP Morgan Chase
	}

	nodes := make([]Node, 0, len(ring)+1)
	nodes = append(nodes, Node{Name: "Phil&Tropic", X: cx, Y: cy, State: NodeStateInfected})
	for i, e := range ring {
		angle := -math.Pi/2 + 2*math.Pi*float64(i)/float64(len(ring))
		nodes = append(nodes, Node{
			Name:     e.name,
			X:        cx + mapOuterRingRel*math.Cos(angle),
			Y:        cy + mapOuterRingRel*math.Sin(angle),
			Security: e.security,
			Defense:  defenseMin + rand.N(defenseMax-defenseMin),
		})
	}

	edges := make([]Edge, 0, 2*len(ring))
	for i := 1; i <= len(ring); i++ {
		edges = append(edges, Edge{From: 0, To: i})
		next := i + 1
		if next > len(ring) {
			next = 1
		}
		edges = append(edges, Edge{From: i, To: next})
	}
	return nodes, edges
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

	for _, n := range g.nodes {
		x, y := pos(n)
		fill, stroke := nodeColors(n.State)
		if n.State == NodeStateAttack {
			fill = scaleAlpha(fill, attackBlinkAlpha(time.Since(g.epoch)))
		}
		vector.FillCircle(screen, x, y, nodeRadius, fill, true)
		vector.StrokeCircle(screen, x, y, nodeRadius, nodeStrokeW, stroke, true)
		if n.Security {
			vector.StrokeCircle(screen, x, y, securityInnerRadius, securityRingW, securityRingFG, true)
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

// handlePatchClick consumes a just-pressed pointer (mouse or touch) over the map area
// and patches the topmost Attack node under it, paying one patch from the inventory.
// No-op when patches are exhausted, when the pointer misses every Attack node, or when
// the press happened outside the map area (so feed-drag etc. stay independent).
func (g *Game) handlePatchClick() {
	if g.mapPanel == nil || g.patchesLeft <= 0 {
		return
	}
	mapRect := g.mapPanel.GetWidget().Rect

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if image.Pt(x, y).In(mapRect) {
			g.tryPatchAt(x, y)
		}
		return
	}
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		if image.Pt(x, y).In(mapRect) {
			if g.tryPatchAt(x, y) {
				return
			}
		}
	}
}

// tryPatchAt patches an Attack node whose hit-disc covers (x, y); returns true on success.
// Hit radius is slightly inflated for finger-friendly tapping on touch screens.
func (g *Game) tryPatchAt(x, y int) bool {
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
			g.nodes[i].State = NodeStatePatched
			g.patchesLeft--
			return true
		}
	}
	return false
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
// crediting the one-time infectionOneShotPct bump. Runs every Update so capture timing
// is independent of attackInterval. The continuous drip is handled by accumulateInfection.
func (g *Game) progressAttacks() {
	now := time.Now()
	for i := range g.nodes {
		if g.nodes[i].State != NodeStateAttack {
			continue
		}
		if now.Sub(g.nodes[i].AttackedAt) >= g.nodes[i].Defense {
			g.nodes[i].State = NodeStateInfected
			g.infectionPct = clampInfection(g.infectionPct + infectionOneShotPct)
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

// producePatches grants +1 patch per Security node currently in Normal state. Called from
// Update on patchProductionInterval cadence (single global timer; per-node timers would
// reward exact-tick captures, which we don't want).
func (g *Game) producePatches() {
	add := 0
	for i := range g.nodes {
		if g.nodes[i].Security && g.nodes[i].State == NodeStateNormal {
			add++
		}
	}
	if add > 0 {
		g.patchesLeft += add
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

func drawCenteredLabel(dst *ebiten.Image, face text.Face, s string, cx, top float64, clr color.Color) {
	w, _ := text.Measure(s, face, 0)
	op := &text.DrawOptions{}
	op.GeoM.Translate(cx-w/2, top)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, s, face, op)
}
