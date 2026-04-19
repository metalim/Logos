package main

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Debug menu: two small rectangular buttons pinned to the bottom corners of the map
// panel. Win on the left, Lose on the right. Used to preview the staged endgame flow
// without having to play out a full run; tinted green/red to make the affordance
// obvious without taking up real estate.
const (
	debugBtnW    = 90
	debugBtnH    = 44
	debugBtnPadX = 12 // distance from map left/right edges
	debugBtnPadY = 12 // distance from map bottom edge
	debugStrokeW = 2
)

var (
	debugBGWin   = color.NRGBA{R: 0x10, G: 0x40, B: 0x18, A: 0xc8}
	debugBGLose  = color.NRGBA{R: 0x40, G: 0x10, B: 0x14, A: 0xc8}
	debugBGEnded = color.NRGBA{R: 0x20, G: 0x22, B: 0x26, A: 0xa0} // dimmed once a run has ended
	debugBorder  = color.NRGBA{R: 0xa0, G: 0xa2, B: 0xa8, A: 0xff}
	debugLabelFG = color.NRGBA{R: 0xf0, G: 0xf2, B: 0xf6, A: 0xff}
)

// debugButtonRects returns Win/Lose button rects in screen coords, anchored to the
// bottom-left and bottom-right corners of the map panel. ok=false if the panel hasn't
// been laid out yet (zero rect).
func (g *Game) debugButtonRects() (win, lose image.Rectangle, ok bool) {
	if g.mapPanel == nil {
		return image.Rectangle{}, image.Rectangle{}, false
	}
	r := g.mapPanel.GetWidget().Rect
	if r.Empty() {
		return image.Rectangle{}, image.Rectangle{}, false
	}
	bottom := r.Max.Y - debugBtnPadY
	top := bottom - debugBtnH
	winLeft := r.Min.X + debugBtnPadX
	loseLeft := r.Max.X - debugBtnPadX - debugBtnW
	win = image.Rect(winLeft, top, winLeft+debugBtnW, bottom)
	lose = image.Rect(loseLeft, top, loseLeft+debugBtnW, bottom)
	return win, lose, true
}

// handleDebugMenu must run before handlePatchClick in Update so a button press never
// double-fires as a patch-menu click. After triggerLoss / triggerWin sets g.end,
// handlePatchClick early-exits via gameEnded() and the same press is harmlessly
// re-evaluated against an empty endgame map.
func (g *Game) handleDebugMenu() {
	x, y, pressed := pollJustPressedPointer()
	if !pressed {
		return
	}
	winR, loseR, ok := g.debugButtonRects()
	if !ok {
		return
	}
	pt := image.Pt(x, y)
	switch {
	case pt.In(winR):
		g.triggerWin()
	case pt.In(loseR):
		g.triggerLoss()
	}
}

// drawDebugMenu paints the two debug buttons last in Game.Draw so they sit above
// every other map element including the patch menu. Once a run has ended both
// buttons dim to neutral grey to indicate clicks are no-ops.
func (g *Game) drawDebugMenu(screen *ebiten.Image) {
	winR, loseR, ok := g.debugButtonRects()
	if !ok || g.overlayLabelFace == nil {
		return
	}
	winBG, loseBG := debugBGWin, debugBGLose
	if g.gameEnded() {
		winBG, loseBG = debugBGEnded, debugBGEnded
	}
	drawDebugButton(screen, winR, "Win", winBG, g.overlayLabelFace)
	drawDebugButton(screen, loseR, "Lose", loseBG, g.overlayLabelFace)
}

func drawDebugButton(dst *ebiten.Image, r image.Rectangle, label string, bg color.NRGBA, face text.Face) {
	x := float32(r.Min.X)
	y := float32(r.Min.Y)
	w := float32(r.Dx())
	h := float32(r.Dy())
	vector.FillRect(dst, x, y, w, h, bg, false)
	const sw = debugStrokeW
	vector.StrokeRect(dst, x+sw/2.0, y+sw/2.0, w-sw, h-sw, sw, debugBorder, false)
	cx := float64(x) + float64(w)/2
	cy := float64(y) + float64(h)/2
	drawAlignedText(dst, face, label, cx, cy, text.AlignCenter, text.AlignCenter, debugLabelFG)
}
