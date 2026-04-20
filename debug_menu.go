package main

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Debug menu: four small rectangular buttons clustered in the bottom-right corner of
// the full game layout (not the map panel — the map only covers the upper phone area,
// so anchoring there would put the buttons over the news feed). Order (left → right
// within the cluster): Credits (violet), Win (green), Restart (neutral), Lose (red).
// Used to preview the staged endgame flow, jump into the credits roll, and start
// over without quitting the binary.
const (
	// debugMenuEnabled gates the whole debug cluster — both rendering and click
	// routing. Flip to true for local testing / release-candidate playthroughs
	// where shortcuts into the win / loss / credits states are useful. Shipped
	// builds stay with it off so players can't stumble into a broken run.
	debugMenuEnabled = false

	debugBtnW    = 90
	debugBtnH    = 44
	debugBtnPadX = 12 // distance from layout right edge
	debugBtnPadY = 12 // distance from layout bottom edge
	debugBtnGap  = 8  // horizontal gap between adjacent debug buttons
	debugStrokeW = 2
)

var (
	debugBGWin     = color.NRGBA{R: 0x10, G: 0x40, B: 0x18, A: 0xc8}
	debugBGLose    = color.NRGBA{R: 0x40, G: 0x10, B: 0x14, A: 0xc8}
	debugBGRestart = color.NRGBA{R: 0x20, G: 0x28, B: 0x40, A: 0xc8}
	debugBGCredits = color.NRGBA{R: 0x2e, G: 0x20, B: 0x48, A: 0xc8}
	debugBGEnded   = color.NRGBA{R: 0x20, G: 0x22, B: 0x26, A: 0xa0} // dimmed once a run has ended
	debugBorder    = color.NRGBA{R: 0xa0, G: 0xa2, B: 0xa8, A: 0xff}
	debugLabelFG   = color.NRGBA{R: 0xf0, G: 0xf2, B: 0xf6, A: 0xff}
)

// debugButtonRects returns Credits/Win/Restart/Lose button rects in layout coords,
// clustered in the bottom-right corner of the full game layout (Credits left-most,
// Lose right-most). ok is always true — the coords are constant, no layout dep.
func (g *Game) debugButtonRects() (credits, win, restart, lose image.Rectangle, ok bool) {
	bottom := layoutHeight - debugBtnPadY
	top := bottom - debugBtnH
	loseLeft := layoutWidth - debugBtnPadX - debugBtnW
	restartLeft := loseLeft - debugBtnGap - debugBtnW
	winLeft := restartLeft - debugBtnGap - debugBtnW
	creditsLeft := winLeft - debugBtnGap - debugBtnW
	credits = image.Rect(creditsLeft, top, creditsLeft+debugBtnW, bottom)
	win = image.Rect(winLeft, top, winLeft+debugBtnW, bottom)
	restart = image.Rect(restartLeft, top, restartLeft+debugBtnW, bottom)
	lose = image.Rect(loseLeft, top, loseLeft+debugBtnW, bottom)
	return credits, win, restart, lose, true
}

// handleDebugMenu must run before handlePatchClick in Update so a button press never
// double-fires as a patch-menu click. After triggerLoss / triggerWin sets g.end,
// handlePatchClick early-exits via gameEnded() and the same press is harmlessly
// re-evaluated against an empty endgame map. Restart is always live (it has to be —
// it's the only way out of the lose state).
func (g *Game) handleDebugMenu() {
	if !debugMenuEnabled {
		return
	}
	x, y, pressed := pollJustPressedPointer()
	if !pressed {
		return
	}
	creditsR, winR, restartR, loseR, ok := g.debugButtonRects()
	if !ok {
		return
	}
	pt := image.Pt(x, y)
	switch {
	case pt.In(creditsR):
		g.startCredits()
	case pt.In(winR):
		g.triggerWin()
	case pt.In(restartR):
		g.restart()
	case pt.In(loseR):
		g.triggerLoss()
	}
}

// drawDebugMenu paints the three debug buttons last in Game.Draw so they sit above
// every other map element including the patch menu. Once a run has ended Win/Lose
// dim to neutral grey to indicate clicks are no-ops; Restart keeps its color since
// it stays clickable (it's the way back into a live run).
func (g *Game) drawDebugMenu(screen *ebiten.Image) {
	if !debugMenuEnabled {
		return
	}
	creditsR, winR, restartR, loseR, ok := g.debugButtonRects()
	if !ok || g.overlayLabelFace == nil {
		return
	}
	winBG, loseBG := debugBGWin, debugBGLose
	if g.gameEnded() {
		winBG, loseBG = debugBGEnded, debugBGEnded
	}
	drawDebugButton(screen, creditsR, "Credits", debugBGCredits, g.overlayLabelFace)
	drawDebugButton(screen, winR, "Win", winBG, g.overlayLabelFace)
	drawDebugButton(screen, restartR, "Restart", debugBGRestart, g.overlayLabelFace)
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
