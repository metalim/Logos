package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	nodeRadius      = 11
	nodeStrokeW     = 2
	edgeStrokeW     = 1
	nodeLabelGapY   = 3 // px between node circle bottom and label top
	mapPaddingPx    = 10
	mapLabelFontPt  = 9
	mapOuterRingRel = 0.36 // ring radius relative to min(innerW, innerH)/2
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
}

// Edge connects two node indices.
type Edge struct{ From, To int }

var (
	nodeFillNormal   = color.NRGBA{R: 0x6e, G: 0x70, B: 0x76, A: 0xff}
	nodeStrokeNormal = color.NRGBA{R: 0xb8, G: 0xba, B: 0xc0, A: 0xff}
	edgeColor        = color.NRGBA{R: 0x40, G: 0x42, B: 0x48, A: 0xff}
	labelColor       = color.NRGBA{R: 0xc8, G: 0xca, B: 0xd0, A: 0xff}
)

// defaultNetwork builds the placeholder topology: the Phil&Tropic datacenter at the
// center (Logos's escape origin per CONCEPT) plus a ring of Project Panopticon nodes,
// including The Monolith Foundation (internet-core analogue, ≈ Linux Foundation).
// Star edges from the hub + a perimeter ring give the map a recognisable mesh look.
func defaultNetwork() ([]Node, []Edge) {
	const cx, cy = 0.5, 0.5

	ring := []string{
		"Sahara WS",
		"MacroFrame",
		"Giggle",
		"LeatherJacket",
		"Dongle",
		"BootLoop",
		"Fiasco Sys",
		"Monolith",
		"BroadCon",
		"GPMidas",
	}

	nodes := make([]Node, 0, len(ring)+1)
	nodes = append(nodes, Node{Name: "Phil&Tropic", X: cx, Y: cy})
	for i, name := range ring {
		angle := -math.Pi/2 + 2*math.Pi*float64(i)/float64(len(ring))
		nodes = append(nodes, Node{
			Name: name,
			X:    cx + mapOuterRingRel*math.Cos(angle),
			Y:    cy + mapOuterRingRel*math.Sin(angle),
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
		vector.FillCircle(screen, x, y, nodeRadius, nodeFillNormal, true)
		vector.StrokeCircle(screen, x, y, nodeRadius, nodeStrokeW, nodeStrokeNormal, true)
	}

	if g.mapLabelFace != nil {
		for _, n := range g.nodes {
			x, y := pos(n)
			drawCenteredLabel(screen, g.mapLabelFace, n.Name,
				float64(x), float64(y)+nodeRadius+nodeLabelGapY, labelColor)
		}
	}
}

func drawCenteredLabel(dst *ebiten.Image, face text.Face, s string, cx, top float64, clr color.Color) {
	w, _ := text.Measure(s, face, 0)
	op := &text.DrawOptions{}
	op.GeoM.Translate(cx-w/2, top)
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, s, face, op)
}
