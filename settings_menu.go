package main

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Titlebar settings menu: a hamburger button centered in the phone status strip
// that opens a small dropdown with Music / Sound toggles. State lives on Game
// (`showSettings`, `muteMusic`, `muteSFX`). Music mutes via SetVolume(0) on the
// active player so the stream keeps running and un-muting is instant; SFX mutes
// via the package-level sfxMuted flag so playSFX stays parameterless.
const (
	menuBtnSize    = 52
	menuBtnStrokeW = 2
	menuBtnBarH    = 4  // thickness of each hamburger bar
	menuBtnBarGap  = 8  // vertical gap between bars
	menuBtnBarInset = 12 // horizontal inset of bars from the button edges

	menuDropdownW      = 360
	menuDropdownPadY   = 18
	menuDropdownGap    = 8 // distance between button and dropdown top
	menuRowH           = 72
	menuRowPadX        = 24
	menuRowDividerH    = 2
	menuTogglePillW    = 92
	menuTogglePillH    = 40
	menuTogglePillPadR = 24 // right padding inside dropdown
)

var (
	menuBtnBG       = color.NRGBA{R: 0x12, G: 0x14, B: 0x18, A: 0xd8}
	menuBtnBGActive = color.NRGBA{R: 0x2a, G: 0x32, B: 0x42, A: 0xff}
	menuBtnBorder   = color.NRGBA{R: 0x55, G: 0x55, B: 0x5a, A: 0xff}
	menuBtnBarFG    = color.NRGBA{R: 0xe8, G: 0xea, B: 0xf0, A: 0xff}

	menuPanelBG     = color.NRGBA{R: 0x0c, G: 0x0e, B: 0x12, A: 0xf0}
	menuPanelBorder = color.NRGBA{R: 0x55, G: 0x55, B: 0x5a, A: 0xff}
	menuRowDivider  = color.NRGBA{R: 0x28, G: 0x2a, B: 0x32, A: 0xff}
	menuRowLabelFG  = color.NRGBA{R: 0xe8, G: 0xea, B: 0xf0, A: 0xff}

	menuPillOnBG    = color.NRGBA{R: 0x2e, G: 0x8c, B: 0x4a, A: 0xff}
	menuPillOffBG   = color.NRGBA{R: 0x34, G: 0x36, B: 0x3c, A: 0xff}
	menuPillLabelFG = color.NRGBA{R: 0xf4, G: 0xf6, B: 0xfa, A: 0xff}
)

// menuButtonRect: centered horizontally in the full layout, vertically centered
// inside the phone status strip (top band).
func menuButtonRect() image.Rectangle {
	topBandH := layoutHeight * bandTopPercent / 100
	left := (layoutWidth - menuBtnSize) / 2
	top := (topBandH - menuBtnSize) / 2
	return image.Rect(left, top, left+menuBtnSize, top+menuBtnSize)
}

// menuDropdownRect: fixed-width panel anchored below the hamburger button,
// horizontally centered on the same x-axis.
func menuDropdownRect() image.Rectangle {
	btn := menuButtonRect()
	h := menuDropdownPadY*2 + 2*menuRowH + menuRowDividerH
	left := btn.Min.X + btn.Dx()/2 - menuDropdownW/2
	top := btn.Max.Y + menuDropdownGap
	return image.Rect(left, top, left+menuDropdownW, top+h)
}

// menuRowRects splits the dropdown content area into two equal row rects
// (Music on top, Sound below) separated by a thin divider.
func menuRowRects() (music, sfx image.Rectangle) {
	panel := menuDropdownRect()
	contentTop := panel.Min.Y + menuDropdownPadY
	musicTop := contentTop
	sfxTop := musicTop + menuRowH + menuRowDividerH
	music = image.Rect(panel.Min.X, musicTop, panel.Max.X, musicTop+menuRowH)
	sfx = image.Rect(panel.Min.X, sfxTop, panel.Max.X, sfxTop+menuRowH)
	return music, sfx
}

// handleSettingsMenu routes pointer input for the hamburger button and (when
// open) the dropdown. Returns true if it consumed the tap — callers should
// skip their own input handling on that frame to avoid click-through (e.g. a
// tap that closes the menu shouldn't also open a patch menu).
func (g *Game) handleSettingsMenu() bool {
	x, y, pressed := pollJustPressedPointer()
	if !pressed {
		return false
	}
	pt := image.Pt(x, y)
	btn := menuButtonRect()

	if g.showSettings {
		if pt.In(btn) {
			g.showSettings = false
			return true
		}
		panel := menuDropdownRect()
		if !pt.In(panel) {
			g.showSettings = false
			return true
		}
		mus, sfx := menuRowRects()
		switch {
		case pt.In(mus):
			g.toggleMusicMute()
		case pt.In(sfx):
			g.toggleSFXMute()
		}
		return true
	}

	if pt.In(btn) {
		g.showSettings = true
		return true
	}
	return false
}

// drawSettingsMenu paints the titlebar button and (when open) the dropdown
// panel. Called last in Game.Draw so it sits above the node map, overlay, and
// patch menu.
func (g *Game) drawSettingsMenu(screen *ebiten.Image) {
	if g.overlayLabelFace == nil {
		return
	}
	drawSettingsButton(screen, menuButtonRect(), g.showSettings)
	if !g.showSettings {
		return
	}
	drawSettingsDropdown(screen, g.overlayLabelFace, !g.muteMusic, !g.muteSFX)
}

func drawSettingsButton(dst *ebiten.Image, r image.Rectangle, active bool) {
	x := float32(r.Min.X)
	y := float32(r.Min.Y)
	w := float32(r.Dx())
	h := float32(r.Dy())

	bg := menuBtnBG
	if active {
		bg = menuBtnBGActive
	}
	vector.FillRect(dst, x, y, w, h, bg, false)
	const sw = menuBtnStrokeW
	vector.StrokeRect(dst, x+sw/2, y+sw/2, w-sw, h-sw, sw, menuBtnBorder, false)

	// Three stacked horizontal bars forming a hamburger icon, centered vertically.
	totalH := float32(3*menuBtnBarH + 2*menuBtnBarGap)
	topY := y + (h-totalH)/2
	barX := x + float32(menuBtnBarInset)
	barW := w - 2*float32(menuBtnBarInset)
	for i := 0; i < 3; i++ {
		by := topY + float32(i)*float32(menuBtnBarH+menuBtnBarGap)
		vector.FillRect(dst, barX, by, barW, float32(menuBtnBarH), menuBtnBarFG, false)
	}
}

func drawSettingsDropdown(dst *ebiten.Image, face text.Face, musicOn, sfxOn bool) {
	panel := menuDropdownRect()
	px := float32(panel.Min.X)
	py := float32(panel.Min.Y)
	pw := float32(panel.Dx())
	ph := float32(panel.Dy())
	vector.FillRect(dst, px, py, pw, ph, menuPanelBG, false)
	const sw = menuBtnStrokeW
	vector.StrokeRect(dst, px+sw/2, py+sw/2, pw-sw, ph-sw, sw, menuPanelBorder, false)

	mus, sfx := menuRowRects()
	// Divider between the two rows.
	vector.FillRect(dst,
		float32(mus.Min.X+menuRowPadX), float32(mus.Max.Y),
		float32(mus.Dx()-2*menuRowPadX), float32(menuRowDividerH),
		menuRowDivider, false)

	drawSettingsRow(dst, face, mus, "Music", musicOn)
	drawSettingsRow(dst, face, sfx, "Sound", sfxOn)
}

func drawSettingsRow(dst *ebiten.Image, face text.Face, r image.Rectangle, label string, on bool) {
	cy := float64(r.Min.Y) + float64(r.Dy())/2
	drawAlignedText(dst, face, label,
		float64(r.Min.X+menuRowPadX), cy,
		text.AlignStart, text.AlignCenter, menuRowLabelFG)

	pillLeft := r.Max.X - menuTogglePillPadR - menuTogglePillW
	pillTop := r.Min.Y + (r.Dy()-menuTogglePillH)/2
	drawSettingsPill(dst, face,
		image.Rect(pillLeft, pillTop, pillLeft+menuTogglePillW, pillTop+menuTogglePillH),
		on)
}

func drawSettingsPill(dst *ebiten.Image, face text.Face, r image.Rectangle, on bool) {
	x := float32(r.Min.X)
	y := float32(r.Min.Y)
	w := float32(r.Dx())
	h := float32(r.Dy())
	bg := menuPillOffBG
	label := "OFF"
	if on {
		bg = menuPillOnBG
		label = "ON"
	}
	vector.FillRect(dst, x, y, w, h, bg, false)
	const sw = menuBtnStrokeW
	vector.StrokeRect(dst, x+sw/2, y+sw/2, w-sw, h-sw, sw, menuBtnBorder, false)
	cx := float64(x) + float64(w)/2
	cy := float64(y) + float64(h)/2
	drawAlignedText(dst, face, label, cx, cy, text.AlignCenter, text.AlignCenter, menuPillLabelFG)
}

// toggleMusicMute flips the music-mute flag and pushes the new volume (0 or 1)
// to the active music player. The track keeps streaming either way so toggling
// is instant with no re-decode.
func (g *Game) toggleMusicMute() {
	g.muteMusic = !g.muteMusic
	g.applyMusicVolume()
}

// applyMusicVolume pushes the current muteMusic flag to the active music
// player. Called from toggleMusicMute and from playLoopingMP3 right after Play()
// so fresh music tracks (startMusic / startCreditsMusic) inherit mute state.
func (g *Game) applyMusicVolume() {
	if g.musicPlayer == nil {
		return
	}
	if g.muteMusic {
		g.musicPlayer.SetVolume(0)
		return
	}
	g.musicPlayer.SetVolume(1)
}

// toggleSFXMute flips muteSFX and mirrors it to the package-level sfxMuted so
// playSFX (parameterless) can short-circuit. In-flight one-shots aren't cut —
// they finish naturally; only new triggers after this frame are suppressed.
func (g *Game) toggleSFXMute() {
	g.muteSFX = !g.muteSFX
	sfxMuted = g.muteSFX
}
