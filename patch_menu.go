package main

import (
	"image"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	menuButtonW = 160
	menuButtonH = 60
	menuGap     = 10
	// Vertical clearance from the node center to the top of the menu (must clear the
	// node circle and its label).
	menuOffsetY = 50
	menuStrokeW = 2
)

var (
	menuBG      = color.NRGBA{R: 0x12, G: 0x14, B: 0x18, A: 0xee}
	menuBorder  = color.NRGBA{R: 0x9a, G: 0x9c, B: 0xa2, A: 0xff}
	menuLabelFG = color.NRGBA{R: 0xe8, G: 0xea, B: 0xf0, A: 0xff}
)

// patchMenuLayout returns the screen-space rects for the Defend (left) and "Patch" (right)
// buttons of the action menu anchored under the given node, or ok=false if the node has
// no valid screen position.
func (g *Game) patchMenuLayout(nodeIdx int) (defend, patch image.Rectangle, ok bool) {
	cx, cy, posOK := g.nodeScreenPos(nodeIdx)
	if !posOK {
		return image.Rectangle{}, image.Rectangle{}, false
	}
	totalW := menuButtonW*2 + menuGap
	leftX := int(cx) - totalW/2
	topY := int(cy) + menuOffsetY
	bottomY := topY + menuButtonH
	defend = image.Rect(leftX, topY, leftX+menuButtonW, bottomY)
	patch = image.Rect(leftX+menuButtonW+menuGap, topY, leftX+menuButtonW+menuGap+menuButtonW, bottomY)
	return defend, patch, true
}

// drawPatchMenu draws the action menu for g.pendingPatchNode if one is set. Drawn after
// the overlay so the buttons sit on top of every other map element.
func (g *Game) drawPatchMenu(screen *ebiten.Image) {
	if g.pendingPatchNode < 0 || g.overlayValueFace == nil {
		return
	}
	defend, patch, ok := g.patchMenuLayout(g.pendingPatchNode)
	if !ok {
		return
	}
	drawMenuButton(screen, defend, "Defend", g.overlayValueFace)
	drawMenuButton(screen, patch, `"Patch"`, g.overlayValueFace)
}

func drawMenuButton(dst *ebiten.Image, r image.Rectangle, label string, face text.Face) {
	x := float32(r.Min.X)
	y := float32(r.Min.Y)
	w := float32(r.Dx())
	h := float32(r.Dy())
	vector.FillRect(dst, x, y, w, h, menuBG, false)
	const sw = menuStrokeW
	vector.StrokeRect(dst, x+sw/2.0, y+sw/2.0, w-sw, h-sw, sw, menuBorder, false)
	cx := float64(x) + float64(w)/2
	cy := float64(y) + float64(h)/2
	drawAlignedText(dst, face, label, cx, cy, text.AlignCenter, text.AlignCenter, menuLabelFG)
}

// applyPatch is the "Patch" action: lock the node down (NodeStatePatched) so it cannot be
// re-attacked but also no longer produces (relevant for Security nodes). Costs one patch.
func (g *Game) applyPatch(nodeIdx int) {
	if !g.canSpendPatchOn(nodeIdx) {
		return
	}
	g.nodes[nodeIdx].State = NodeStatePatched
	g.patchesLeft--
}

// applyDefend is the Defend action: bounce the node back to Normal so it stays in play
// (and keeps producing patches if Security), at the cost of one patch. The node remains a
// future-attack target — defense window stays the same so re-attacks can flip it Infected.
func (g *Game) applyDefend(nodeIdx int) {
	if !g.canSpendPatchOn(nodeIdx) {
		return
	}
	g.nodes[nodeIdx].State = NodeStateNormal
	g.nodes[nodeIdx].AttackedAt = time.Time{}
	g.patchesLeft--
}

func (g *Game) canSpendPatchOn(nodeIdx int) bool {
	return g.patchesLeft > 0 && nodeIdx >= 0 && nodeIdx < len(g.nodes)
}
