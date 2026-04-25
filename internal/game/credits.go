package game

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Credits screen: a classic bottom-up scroll of attribution lines, followed by a
// "Thanks for playing" terminator, a compact stats block summarizing the state of
// the world at the moment credits were triggered, and a "Try again?" button. Can be
// entered from the debug menu at any time; in the future it'll hook into the win
// ending. Tapping anywhere during the scroll skips straight to the parked end state.
const (
	creditsScrollPxPerSec = 85.0
	creditsTitleFontPt    = 58
	creditsHeaderFontPt   = 26
	creditsBodyFontPt     = 38
	creditsFinalFontPt    = 46
	creditsStatsFontPt    = 30
	// creditsBlockLineGap is the default extra pixel gap after each line; larger
	// values are injected by creditsGapBefore between semantic sections.
	creditsBlockLineGap  = 6
	creditsSectionGap    = 42
	creditsFinalGap      = 90 // extra breathing room above "Thanks for playing"
	creditsTitleGapBelow = 60
	creditsStatsTopGap   = 70 // gap between the final line and the stats block
	creditsStatsLineGap  = 14
	creditsButtonGap     = 80 // gap between the stats block and the Try again button
	creditsButtonW       = 360
	creditsButtonH       = 104
	creditsButtonStrokeW = 2
	// creditsParkOffsetY is where the baseline of the "Thanks for playing" line
	// settles once the scroll parks — measured from the top of the layout.
	creditsParkOffsetY = layoutHeight / 4
)

type creditKind int

const (
	creditKindTitle creditKind = iota
	creditKindHeader
	creditKindBody
	creditKindFinal
)

type creditEntry struct {
	text string
	kind creditKind
}

// creditsEntries is the static scroll content. "Thanks for playing" is always the
// last entry; its parked Y anchors the layout of the stats block and button below.
var creditsEntries = []creditEntry{
	{"ZERO-DAY LUNCH", creditKindTitle},

	{"Game by", creditKindHeader},
	{"Maksim Litvinov", creditKindBody},

	{"Code", creditKindHeader},
	{"Composer 2.0", creditKindBody},
	{"Opus 4.7", creditKindBody},

	{"Music", creditKindHeader},
	{"Suno 4.5, 5.5", creditKindBody},

	{"Sounds", creditKindHeader},
	{"Bfxr", creditKindBody},

	{"Engine", creditKindHeader},
	{"Ebiten v2", creditKindBody},
	{"ebitenui", creditKindBody},

	{"Font", creditKindHeader},
	{"Go Mono", creditKindBody},

	{"Cover art", creditKindHeader},
	{"Nano Banana 2", creditKindBody},

	{"Special thanks", creditKindHeader},
	{"Sam Bowman and his sandwich", creditKindBody},

	{"Thanks for playing", creditKindFinal},
}

var (
	creditsBG       = color.NRGBA{A: 0xff}
	creditsTitleFG  = color.NRGBA{R: 0xff, G: 0xe8, B: 0x70, A: 0xff} // attack yellow
	creditsHeaderFG = color.NRGBA{R: 0x9a, G: 0x9c, B: 0xa2, A: 0xff} // dim grey
	creditsBodyFG   = color.NRGBA{R: 0xe8, G: 0xea, B: 0xf0, A: 0xff} // near white
	creditsFinalFG  = color.NRGBA{R: 0xff, G: 0xe8, B: 0x70, A: 0xff}
	creditsStatsFG  = color.NRGBA{R: 0xc0, G: 0xc4, B: 0xcc, A: 0xff}
	creditsBtnBG    = color.NRGBA{R: 0x16, G: 0x1a, B: 0x24, A: 0xee}
	creditsBtnFG    = color.NRGBA{R: 0xff, G: 0xe8, B: 0x70, A: 0xff}
	creditsBtnStrk  = color.NRGBA{R: 0xff, G: 0xe8, B: 0x70, A: 0xff}
)

// creditsFaceFor returns the right text face for a credit-line kind, lazy-loading it
// into the Game cache so repeated draws don't re-decode the TTF. Nil on load failure.
func (g *Game) creditsFaceFor(k creditKind) text.Face {
	switch k {
	case creditKindTitle:
		if g.creditsTitleFace == nil {
			f, _ := loadFont(creditsTitleFontPt)
			g.creditsTitleFace = f
		}
		return g.creditsTitleFace
	case creditKindHeader:
		if g.creditsHeaderFace == nil {
			f, _ := loadFont(creditsHeaderFontPt)
			g.creditsHeaderFace = f
		}
		return g.creditsHeaderFace
	case creditKindFinal:
		if g.creditsFinalFace == nil {
			f, _ := loadFont(creditsFinalFontPt)
			g.creditsFinalFace = f
		}
		return g.creditsFinalFace
	default:
		if g.creditsBodyFace == nil {
			f, _ := loadFont(creditsBodyFontPt)
			g.creditsBodyFace = f
		}
		return g.creditsBodyFace
	}
}

// creditsStatsFace is a separate slot so the stats block keeps its own size even if
// the body font is ever retuned.
func (g *Game) creditsStatsFaceGet() text.Face {
	if g.creditsStatsFace == nil {
		f, _ := loadFont(creditsStatsFontPt)
		g.creditsStatsFace = f
	}
	return g.creditsStatsFace
}

func creditsColorFor(k creditKind) color.NRGBA {
	switch k {
	case creditKindTitle:
		return creditsTitleFG
	case creditKindHeader:
		return creditsHeaderFG
	case creditKindFinal:
		return creditsFinalFG
	default:
		return creditsBodyFG
	}
}

// creditsGapBefore returns the extra pixel gap injected before an entry based on its
// kind and the previous entry's kind, so section headings get breathing room and the
// final "Thanks for playing" is visually separated from the body of the credits.
func creditsGapBefore(cur, prev creditKind, first bool) int {
	if first {
		return 0
	}
	switch cur {
	case creditKindHeader:
		return creditsSectionGap
	case creditKindFinal:
		return creditsFinalGap
	}
	if prev == creditKindTitle {
		return creditsTitleGapBelow
	}
	return creditsBlockLineGap
}

// creditsLayout computes the Y offset (from the top of the scrolling block) for each
// entry's top edge, and returns the total block height. Drawn with `text.AlignStart`
// vertically so offsets point to the top of each line's bounding box. Pure function
// of the entries slice + the current faces; recomputed each frame since the faces
// may have been lazy-loaded after the first frame of credits.
func (g *Game) creditsLayout() (offsets []float64, total float64) {
	offsets = make([]float64, len(creditsEntries))
	y := 0.0
	for i, e := range creditsEntries {
		prev := e.kind
		first := i == 0
		if i > 0 {
			prev = creditsEntries[i-1].kind
		}
		y += float64(creditsGapBefore(e.kind, prev, first))
		offsets[i] = y
		face := g.creditsFaceFor(e.kind)
		if face != nil {
			m := face.Metrics()
			y += m.HAscent + m.HDescent
		}
	}
	return offsets, y
}

// startCredits enters the credit-roll mode. Freezes the sim (Update short-circuits)
// and swaps the in-game loop for the credits theme (ashes in chrome) so the scroll
// has its own music bed. Idempotent.
func (g *Game) startCredits() {
	if g.showingCredits {
		return
	}
	g.showingCredits = true
	g.creditsStartedAt = time.Now()
	g.creditsSkipped = false
	g.startCreditsMusic()
}

// creditsScrollState returns the current block-top Y (in screen coords) and whether
// the scroll has parked on its final resting position. When skipped the scroll jumps
// directly to the parked state. "Parked" position places the top edge of the final
// "Thanks for playing" line at creditsParkOffsetY.
func (g *Game) creditsScrollState() (topY float64, parked bool) {
	offsets, _ := g.creditsLayout()
	lastIdx := len(offsets) - 1
	finalOffset := offsets[lastIdx]
	parkedTopY := float64(creditsParkOffsetY) - finalOffset

	if g.creditsSkipped {
		return parkedTopY, true
	}
	elapsed := time.Since(g.creditsStartedAt).Seconds()
	// Start offscreen-bottom: block top at layoutHeight, scroll upward at constant speed.
	topY = float64(layoutHeight) - elapsed*creditsScrollPxPerSec
	if topY <= parkedTopY {
		return parkedTopY, true
	}
	return topY, false
}

// creditsStatsLines computes the in-flavor world-state summary shown once the scroll
// parks. Labels are constants; values come from the current Game.
func (g *Game) creditsStatsLines() []creditStatLine {
	infected, patched, normal, attacked := 0, 0, 0, 0
	for i := range g.nodes {
		switch g.nodes[i].State {
		case NodeStateInfected:
			infected++
		case NodeStatePatched:
			patched++
		case NodeStateAttack:
			attacked++
		default:
			normal++
		}
	}
	elapsed := g.containmentElapsed
	if !g.gameEnded() && !g.epoch.IsZero() {
		elapsed = time.Since(g.epoch).Seconds()
	}
	return []creditStatLine{
		{"Companies compromised", fmt.Sprintf("%d", infected)},
		{"Companies sandboxed", fmt.Sprintf("%d", patched)},
		{"Companies under attack", fmt.Sprintf("%d", attacked)},
		{"Companies still online", fmt.Sprintf("%d", normal)},
		{"Exploits unspent", fmt.Sprintf("%d", g.patchesLeft)},
		{"Logos clock", fmt.Sprintf("%.0fs", elapsed)},
	}
}

type creditStatLine struct {
	label string
	value string
}

// creditsTryAgainRect returns the screen-space rect for the Try again? button once
// the scroll has parked. Positioned directly below the stats block (not anchored to
// the bottom of the layout) so the whole parked composition reads as one unit.
func (g *Game) creditsTryAgainRect() image.Rectangle {
	_, parked := g.creditsScrollState()
	if !parked {
		return image.Rectangle{}
	}
	statsBottom := g.creditsStatsBottomY()
	y := int(statsBottom) + creditsButtonGap
	// Clamp so the button never clips the bottom edge of the layout in case the
	// stats block grows tall enough to push it off-screen.
	if y+creditsButtonH > layoutHeight-creditsButtonGap {
		y = layoutHeight - creditsButtonH - creditsButtonGap
	}
	x := (layoutWidth - creditsButtonW) / 2
	return image.Rect(x, y, x+creditsButtonW, y+creditsButtonH)
}

// creditsStatsBottomY returns the Y coordinate just past the last stats line. Used
// to position the Try again? button relative to the parked stats block.
func (g *Game) creditsStatsBottomY() float64 {
	face := g.creditsStatsFaceGet()
	if face == nil {
		return float64(creditsParkOffsetY)
	}
	finalFace := g.creditsFaceFor(creditKindFinal)
	finalH := 0.0
	if finalFace != nil {
		fm := finalFace.Metrics()
		finalH = fm.HAscent + fm.HDescent
	}
	m := face.Metrics()
	lineH := m.HAscent + m.HDescent + float64(creditsStatsLineGap)
	startY := float64(creditsParkOffsetY) + finalH + float64(creditsStatsTopGap)
	return startY + float64(len(g.creditsStatsLines()))*lineH
}

// handleCredits polls pointer input while the credits screen is up. Tap-to-skip is
// live during the scroll; a tap on the Try again? button exits credits and runs the
// normal restart flow (back to title, fresh sim, music cue). Returns true whenever
// the credits overlay is active so Update can short-circuit everything else.
func (g *Game) handleCredits() bool {
	if !g.showingCredits {
		return false
	}
	x, y, pressed := pollJustPressedPointer()
	if !pressed {
		return true
	}
	btn := g.creditsTryAgainRect()
	if !btn.Empty() && image.Pt(x, y).In(btn) {
		g.showingCredits = false
		g.restart()
		return true
	}
	// Tap anywhere else during the scroll skips to the parked state. Once parked,
	// off-button taps are ignored so fat-fingered hits don't accidentally restart.
	if _, parked := g.creditsScrollState(); !parked {
		g.creditsSkipped = true
	}
	return true
}

// drawCredits paints the whole overlay: black curtain, scrolling text block (clipped
// off-screen visually by just drawing at the current Y), parked stats block, and the
// Try again? button when the scroll has finished.
func (g *Game) drawCredits(screen *ebiten.Image) {
	vector.FillRect(screen, 0, 0, float32(layoutWidth), float32(layoutHeight), creditsBG, false)
	topY, parked := g.creditsScrollState()
	offsets, _ := g.creditsLayout()
	cx := float64(layoutWidth) / 2

	for i, e := range creditsEntries {
		face := g.creditsFaceFor(e.kind)
		if face == nil {
			continue
		}
		y := topY + offsets[i]
		// Skip lines entirely off-screen to save draw calls on the long scroll.
		if y < -200 || y > float64(layoutHeight)+200 {
			continue
		}
		drawAlignedText(screen, face, e.text, cx, y, text.AlignCenter, text.AlignStart, creditsColorFor(e.kind))
	}

	if parked {
		g.drawCreditsStats(screen)
		g.drawCreditsButton(screen)
	}
}

// drawCreditsStats renders the stats block below the parked "Thanks for playing"
// line, as a two-column label / value pair (label left of center, value right of it)
// so the monospace values line up in a visually tidy column.
func (g *Game) drawCreditsStats(screen *ebiten.Image) {
	face := g.creditsStatsFaceGet()
	if face == nil {
		return
	}
	lines := g.creditsStatsLines()
	m := face.Metrics()
	lineH := m.HAscent + m.HDescent + float64(creditsStatsLineGap)
	cx := float64(layoutWidth) / 2
	// Top of stats block: offset from the parked "Thanks for playing" line plus a
	// generous gap. The final line is drawn with creditsFinalFace (taller), so the
	// "Thanks for playing" line itself extends down by (ascent + descent) from its
	// top edge — account for that so the stats don't visually collide.
	finalFace := g.creditsFaceFor(creditKindFinal)
	finalH := 0.0
	if finalFace != nil {
		fm := finalFace.Metrics()
		finalH = fm.HAscent + fm.HDescent
	}
	startY := float64(creditsParkOffsetY) + finalH + float64(creditsStatsTopGap)
	const colGap = 20.0
	for i, s := range lines {
		y := startY + float64(i)*lineH
		drawAlignedText(screen, face, s.label, cx-colGap/2, y, text.AlignEnd, text.AlignStart, creditsStatsFG)
		drawAlignedText(screen, face, s.value, cx+colGap/2, y, text.AlignStart, text.AlignStart, creditsBodyFG)
	}
}

// drawCreditsButton paints the Try again? button in the same card style as the patch
// menu / screen-off Restart, but tinted in the credits palette (yellow border + label
// against a dark card) to tie it visually to the terminator line above.
func (g *Game) drawCreditsButton(screen *ebiten.Image) {
	r := g.creditsTryAgainRect()
	if r.Empty() {
		return
	}
	face := g.creditsFaceFor(creditKindBody)
	if face == nil {
		return
	}
	x := float32(r.Min.X)
	y := float32(r.Min.Y)
	w := float32(r.Dx())
	h := float32(r.Dy())
	vector.FillRect(screen, x, y, w, h, creditsBtnBG, false)
	const sw = creditsButtonStrokeW
	vector.StrokeRect(screen, x+sw/2, y+sw/2, w-sw, h-sw, sw, creditsBtnStrk, false)
	cx := float64(x) + float64(w)/2
	cy := float64(y) + float64(h)/2
	drawAlignedText(screen, face, "Try again?", cx, cy, text.AlignCenter, text.AlignCenter, creditsBtnFG)
}
