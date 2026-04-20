package main

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// First-run hint: a small banner that appears just below the overlay strip the first
// time an Attack node is live on screen, explaining the core interaction. Dismissed
// permanently (for this process) the moment the player opens a patch menu — if they
// restart mid-session the hint doesn't come back. The flag is process-wide because
// there's no persistent storage yet; a fresh WASM page load or binary start re-shows
// it, which matches "first launch" well enough in practice.
const (
	hintText      = "Tap yellow nodes to open the defense menu."
	hintPadX      = 24
	hintPadY      = 14
	hintMarginTop = 12 // gap between overlay strip bottom and hint box top
	hintStrokeW   = 2
)

var (
	hintBG     = color.NRGBA{R: 0x0a, G: 0x0b, B: 0x0e, A: 0xd8}
	hintBorder = color.NRGBA{R: 0xff, G: 0xe8, B: 0x70, A: 0xff} // matches nodeStrokeAttack
	hintFG     = color.NRGBA{R: 0xf4, G: 0xf6, B: 0xfa, A: 0xff}
)

// hintEverDismissed latches true the first time the player opens a patch menu. Process-
// scoped so a restart inside the same session doesn't re-show the hint, but a fresh
// binary / browser tab reload does.
var hintEverDismissed bool

// shouldShowHint reports whether the first-run hint should be rendered this frame.
// Conditions: not already dismissed, not on title, not in an endgame latch, and there
// is at least one live Attack node on screen (nothing to point at before that).
func (g *Game) shouldShowHint() bool {
	if hintEverDismissed || g.showTitle || g.gameEnded() {
		return false
	}
	for i := range g.nodes {
		if g.nodes[i].State == NodeStateAttack {
			return true
		}
	}
	return false
}

// dismissHint latches the process-wide hint flag. Called from handlePatchClick the
// first time the player actually opens a patch menu.
func dismissHint() { hintEverDismissed = true }

// drawHint paints the banner below the overlay strip inside the map panel. No-op when
// the hint shouldn't show or the map/overlay face isn't ready yet.
func (g *Game) drawHint(screen *ebiten.Image) {
	if !g.shouldShowHint() || g.overlayLabelFace == nil || g.mapPanel == nil {
		return
	}
	rect := g.mapPanel.GetWidget().Rect
	if rect.Empty() {
		return
	}
	tw, th := text.Measure(hintText, g.overlayLabelFace, 0)
	boxW := int(tw) + hintPadX*2
	boxH := int(th) + hintPadY*2
	cx := (rect.Min.X + rect.Max.X) / 2
	x := cx - boxW/2
	y := rect.Min.Y + overlayMarginPx + overlayHeightPx + hintMarginTop
	r := image.Rect(x, y, x+boxW, y+boxH)
	fx, fy := float32(r.Min.X), float32(r.Min.Y)
	fw, fh := float32(boxW), float32(boxH)
	vector.FillRect(screen, fx, fy, fw, fh, hintBG, false)
	const sw = hintStrokeW
	vector.StrokeRect(screen, fx+sw/2, fy+sw/2, fw-sw, fh-sw, sw, hintBorder, false)
	drawAlignedText(screen, g.overlayLabelFace, hintText,
		float64(cx), float64(y)+float64(boxH)/2,
		text.AlignCenter, text.AlignCenter, hintFG)
}
