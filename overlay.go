package main

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	// Two-row strip: INFECTION on top, CONTAINMENT below. Row centers sit at h/4 and
	// 3h/4 of overlayHeightPx; PATCHES stays vertical-centered on the right edge.
	overlayHeightPx    = 100
	overlayMarginPx    = 15
	overlayInnerPadPx  = 20
	overlayItemGapPx   = 15
	overlayBorderW     = 2
	progressBarW       = 225
	progressBarH       = 15
	overlayLabelFontPt = 23
	overlayValueFontPt = 30
)

var (
	overlayBG        = color.NRGBA{R: 0x0a, G: 0x0b, B: 0x0e, A: 0xc8}
	overlayBorder    = color.NRGBA{R: 0x40, G: 0x42, B: 0x48, A: 0xff}
	overlayLabel     = color.NRGBA{R: 0x9a, G: 0x9c, B: 0xa2, A: 0xff}
	overlayValue     = color.NRGBA{R: 0xe8, G: 0xea, B: 0xf0, A: 0xff}
	infectionTrack   = color.NRGBA{R: 0x30, G: 0x32, B: 0x36, A: 0xff}
	infectionFill    = color.NRGBA{R: 0xd0, G: 0x40, B: 0x40, A: 0xff}
	containmentTrack = color.NRGBA{R: 0x30, G: 0x36, B: 0x32, A: 0xff}
	containmentFill  = color.NRGBA{R: 0x40, G: 0xc8, B: 0x68, A: 0xff}
)

// drawGameOverlay draws the two-row game-state strip across the top of the map area,
// on top of edges/nodes drawn earlier in Draw. Top row: INFECTION + bar + value +
// rate. Bottom row: CONTAINMENT + bar + value + rate. Right edge: x{patches} PATCHES,
// vertical-centered across both rows since it's the only right-side element.
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
	const sw = overlayBorderW
	vector.StrokeRect(screen, x+sw/2.0, y+sw/2.0, w-sw, h-sw, sw, overlayBorder, false)

	cy := float64(y) + float64(h)/2
	cyTop := float64(y) + float64(h)/4
	cyBot := float64(y) + 3*float64(h)/4

	// Bars share an x anchor so the wider label ("CONTAINMENT") doesn't push its bar
	// to the right of the narrower one ("INFECTION"). Compute once from the actual
	// rendered widths so a font/label change keeps the alignment automatic.
	labelInfW, _ := text.Measure("INFECTION", g.overlayLabelFace, 0)
	labelConW, _ := text.Measure("CONTAINMENT", g.overlayLabelFace, 0)
	labelMaxW := labelInfW
	if labelConW > labelMaxW {
		labelMaxW = labelConW
	}
	barX := float64(x) + overlayInnerPadPx + labelMaxW + overlayItemGapPx

	infectionRate := infectionRatePerSec * float64(countInfected(g.nodes))
	g.drawProgressRow(screen, float64(x), barX, cyTop,
		"INFECTION", g.infectionPct, infectionRate, infectionTrack, infectionFill)
	g.drawProgressRow(screen, float64(x), barX, cyBot,
		"CONTAINMENT", g.containmentPct, containmentRatePerSec, containmentTrack, containmentFill)

	rightX := float64(x+w) - overlayInnerPadPx
	rightX -= drawAlignedText(screen, g.overlayValueFace, fmt.Sprintf("x%d", g.patchesLeft),
		rightX, cy, text.AlignEnd, text.AlignCenter, overlayValue)
	rightX -= overlayItemGapPx
	drawAlignedText(screen, g.overlayLabelFace, `"PATCHES"`,
		rightX, cy, text.AlignEnd, text.AlignCenter, overlayLabel)
}

// drawProgressRow lays out one labelled progress bar row inside the overlay strip:
// LABEL (left-aligned at xLeft+pad) → bar (anchored at barX so all rows align) →
// "NN.N%" value → "+R.R%/s" rate. Caller decides the row's vertical center (cy),
// the shared bar anchor (barX), and color palette so the same helper covers both
// INFECTION (red) and CONTAINMENT (green) with vertically aligned bars.
func (g *Game) drawProgressRow(screen *ebiten.Image, xLeft, barX, cy float64,
	label string, pct, rate float64, trackClr, fillClr color.NRGBA,
) {
	drawAlignedText(screen, g.overlayLabelFace, label,
		xLeft+overlayInnerPadPx, cy, text.AlignStart, text.AlignCenter, overlayLabel)

	bx := float32(barX)
	by := float32(cy) - progressBarH/2
	vector.FillRect(screen, bx, by, progressBarW, progressBarH, trackClr, false)
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	if fillW := progressBarW * float32(pct/100.0); fillW > 0 {
		vector.FillRect(screen, bx, by, fillW, progressBarH, fillClr, false)
	}

	textX := barX + float64(progressBarW) + overlayItemGapPx
	textX += drawAlignedText(screen, g.overlayValueFace, fmt.Sprintf("%.1f%%", pct),
		textX, cy, text.AlignStart, text.AlignCenter, overlayValue)
	textX += overlayItemGapPx
	drawAlignedText(screen, g.overlayLabelFace, fmt.Sprintf("+%.1f%%/s", rate),
		textX, cy, text.AlignStart, text.AlignCenter, fillClr)
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
