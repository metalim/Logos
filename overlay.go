package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	overlayHeightPx     = 24
	overlayMarginPx     = 6
	overlayInnerPadPx   = 8
	overlayItemGapPx    = 6
	infectionBarW       = 90
	infectionBarH       = 6
	overlayLabelFontPt  = 9
	overlayValueFontPt  = 12
	overlayMinInfection = 1
)

var (
	overlayBG      = color.NRGBA{R: 0x0a, G: 0x0b, B: 0x0e, A: 0xc8}
	overlayBorder  = color.NRGBA{R: 0x40, G: 0x42, B: 0x48, A: 0xff}
	overlayLabel   = color.NRGBA{R: 0x9a, G: 0x9c, B: 0xa2, A: 0xff}
	overlayValue   = color.NRGBA{R: 0xe8, G: 0xea, B: 0xf0, A: 0xff}
	infectionTrack = color.NRGBA{R: 0x30, G: 0x32, B: 0x36, A: 0xff}
	infectionFill  = color.NRGBA{R: 0xd0, G: 0x40, B: 0x40, A: 0xff}
)

// drawGameOverlay draws the game-state strip (infection % + remaining patches)
// across the top of the map area, on top of edges/nodes drawn earlier in Draw.
func (g *Game) drawGameOverlay(screen *ebiten.Image) {
	if g.mapPanel == nil || g.overlayLabelFace == nil || g.overlayValueFace == nil {
		return
	}
	rect := g.mapPanel.GetWidget().Rect
	if rect.Empty() {
		return
	}

	x := float32(rect.Min.X + overlayMarginPx)
	y := float32(rect.Min.Y + overlayMarginPx)
	w := float32(rect.Dx() - 2*overlayMarginPx)
	h := float32(overlayHeightPx)

	vector.FillRect(screen, x, y, w, h, overlayBG, false)
	vector.StrokeRect(screen, x+0.5, y+0.5, w-1, h-1, 1, overlayBorder, false)

	cy := float64(y) + float64(h)/2

	leftX := float64(x + overlayInnerPadPx)
	leftX += drawAlignedText(screen, g.overlayLabelFace, "INFECTION",
		leftX, cy, text.AlignStart, text.AlignCenter, overlayLabel)
	leftX += overlayItemGapPx

	barX := float32(leftX)
	barY := float32(cy) - infectionBarH/2
	vector.FillRect(screen, barX, barY, infectionBarW, infectionBarH, infectionTrack, false)
	pct := g.infectionPct
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	if fillW := infectionBarW * float32(pct/100.0); fillW > 0 {
		vector.FillRect(screen, barX, barY, fillW, infectionBarH, infectionFill, false)
	}
	leftX += float64(infectionBarW) + overlayItemGapPx

	drawAlignedText(screen, g.overlayValueFace, fmt.Sprintf("%d%%", int(pct+0.5)),
		leftX, cy, text.AlignStart, text.AlignCenter, overlayValue)

	rightX := float64(x+w) - overlayInnerPadPx
	rightX -= drawAlignedText(screen, g.overlayValueFace, fmt.Sprintf("x%d", g.patchesLeft),
		rightX, cy, text.AlignEnd, text.AlignCenter, overlayValue)
	rightX -= overlayItemGapPx
	drawAlignedText(screen, g.overlayLabelFace, "PATCHES",
		rightX, cy, text.AlignEnd, text.AlignCenter, overlayLabel)
}

// drawAlignedText renders s with the given primary/secondary alignment around (x, y).
// Returns the rendered text width so callers can chain inline elements.
func drawAlignedText(dst *ebiten.Image, face text.Face, s string,
	x, y float64, h, v text.Align, clr color.Color,
) float64 {
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.PrimaryAlign = h
	op.SecondaryAlign = v
	op.ColorScale.ScaleWithColor(clr)
	text.Draw(dst, s, face, op)
	w, _ := text.Measure(s, face, 0)
	return w
}
