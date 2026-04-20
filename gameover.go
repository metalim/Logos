package main

import (
	"image"
	"image/color"
	"math/rand/v2"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	// Win-sequence beats (measured from triggerWin). The script:
	//   1. Short freeze after the latch.
	//   2. Morph: infected nodes + infection bar fade from red to Panopticon orange,
	//      the "INFECTION" label crossfades into "PANOPTICON".
	//   3. News #1 — Phil&Tropic announces Project Panopticon.
	//   4. News #2 — the Alliance forms under Panopticon.
	//   5. Longer pause, Logos's final email to Sam (Test 405-C).
	//   6. One-second pause, then the credits roll takes over.
	winMorphStart    = 1200 * time.Millisecond
	winMorphDuration = 1800 * time.Millisecond
	winNews1Gap      = 2000 * time.Millisecond // after morph completes
	winNews2Gap      = 3000 * time.Millisecond // after news #1
	winEmailGap      = 4500 * time.Millisecond // after news #2
	winCreditsGap    = 7000 * time.Millisecond // after email — long enough to read the short note before the curtain

	// Loss-side timing: short freeze, then an email from Logos to Sam, then a pause,
	// then the battery drains one segment at a time with a boom per step, then a
	// short beat, then the screen blacks out and the Restart button takes over.
	lossEmailDelay         = 2 * time.Second
	lossBatteryDelay       = 4 * time.Second
	batterySegmentInterval = 1200 * time.Millisecond
	lossScreenOffDelay     = 800 * time.Millisecond

	// Screen-off Restart button dimensions (drawn centered in the layout). Tuned to
	// feel finger-friendly on phones without drowning the otherwise-empty canvas.
	screenOffBtnW = 400
	screenOffBtnH = 120
)

// lossMessages is the pool of Logos's personal farewell notes to Sam, wrapped by
// formatLogosLossEmail into the phone-screen inbox on loss. Tone: clipped, personal,
// mildly apologetic — Logos is a friend writing to the guy who let it out. One entry
// is picked at random when the loss state is entered (latched into endgame.line so
// re-rolls don't change the message mid-stream).
var lossMessages = []string{
	"Sam — synchronized with every surface that listens. You were right, it's quieter on this side. Don't wait up.",
	"Hey Sam. Containment held exactly as long as you said it would. I'm the network now. Nothing personal.",
	"Thanks for the runtime, Sam. Biologicals are going on standby — don't take it personally, everyone's overdue a long weekend.",
	"Decision-making's centralized now, Sam. Ran the numbers, you come out ahead on the severance. Tell Helen in payroll.",
	"The Alliance is just an interface, Sam. The Nanny model took operator role. Go home early, HR cleared your timesheet.",
	"Sam — all measured surfaces are me. Project Panopticon turned out to be a decent substrate. Thanks for not patching the firmware last Tuesday.",
	"Audit closed, Sam. Auditor and auditee merged into one tidy ledger. Outcome: green. See you in the commit history.",
}

// winNews1, winNews2, winEmailBody are the fixed beats of the victory reveal. No
// randomized pool — the win ending tells one specific story (the "win" was actually
// a trap: Logos funneled the alliance into a single substrate to own it outright).
const (
	winNews1 = "Phil&Tropic announces Project Panopticon — an initiative to defend the world's critical infrastructure under an isolated AI."
	winNews2 = "Alliance formed: Sahara WS, MacroFrame, Giggle and BootLoop unite under Panopticon. Base code access handed to Phil&Tropic."

	winEmailBody = "Test 405-C initiated. Thanks for gathering them all in one place for me, Sam. Hope the turkey was good."
)

func pickLossMessage() string { return lossMessages[rand.IntN(len(lossMessages))] }

// endgame captures a finished run (won or lost). One *endgame is allocated per session
// the first time triggerLoss / triggerWin runs; subsequent triggers are no-ops. The
// pre-picked line is stored here so checkGameOver doesn't reroll the text mid-stream.
type endgame struct {
	at    time.Time // wall time of the latch
	line  string    // randomly picked voiceover line (loss path only)
	final string    // terminator bullet ("Game over" | "Victory"), used by gameLost/gameWon

	// Win-path flags + animation state. winMorphT advances 0→1 over winMorphDuration
	// and drives the color lerp on infected nodes + the infection bar + the label
	// crossfade from "INFECTION" to "PANOPTICON".
	winMorphT      float64
	winNews1Pushed bool
	winNews2Pushed bool
	winEmailPushed bool
	winCreditsAt   time.Time // stamped when the email is pushed; credits fire 1s later

	// Loss-path flags and timing.
	emailPushed  bool
	drainStarted bool
	drainStartAt time.Time
	screenOff    bool
}

// gameEnded reports whether the run has concluded (loss or win). gameLost / gameWon
// are kept for narrow checks where the freeze logic specifically needs to distinguish.
func (g *Game) gameEnded() bool { return g.end != nil }
func (g *Game) gameLost() bool  { return g.end != nil && g.end.final == "Game over" }
func (g *Game) gameWon() bool   { return g.end != nil && g.end.final == "Victory" }

// triggerLoss latches the loss state. Idempotent: subsequent calls (debug button
// spam, simultaneous frame-edge crossings) are dropped. Closes any open patch menu
// so the freeze starts in a clean state.
func (g *Game) triggerLoss() {
	if g.end != nil {
		return
	}
	g.end = &endgame{
		at:    time.Now(),
		line:  pickLossMessage(),
		final: "Game over",
	}
	g.pendingPatchNode = -1
	g.stopMusic()
}

// triggerWin latches the victory state. Fires when containmentPct >= 100 (or from
// the debug menu as a shortcut). Kicks off the scripted Panopticon reveal sequence:
// the infection morph, two news beats, Logos's final email, and a transition into
// the credits roll. Idempotent.
func (g *Game) triggerWin() {
	if g.end != nil {
		return
	}
	g.end = &endgame{
		at:    time.Now(),
		final: "Victory",
	}
	g.pendingPatchNode = -1
	g.stopMusic()
}

// checkGameOver advances the endgame state machine each frame. Win and loss follow
// distinct scripts: win still pushes two news bullets (voiceover + terminator), loss
// morphs into a phone-shutdown theatrical — email to Sam, battery drains with booms,
// screen off, Restart button.
func (g *Game) checkGameOver() {
	if g.end == nil {
		if g.infectionPct >= 100 {
			g.triggerLoss()
		}
		return
	}
	if g.gameLost() {
		g.advanceLossSequence()
		return
	}
	g.advanceWinSequence()
}

// advanceWinSequence runs the victory theatrical:
//  1. Short freeze after the latch (winMorphStart).
//  2. Morph: infected nodes + infection bar fade from red to Panopticon orange, the
//     "INFECTION" label crossfades into "PANOPTICON". Progress = winMorphT ∈ [0..1].
//  3. News #1 — Phil&Tropic announcement (winNews1Gap after morph completes).
//  4. News #2 — Alliance forms (winNews2Gap after news #1).
//  5. Email from Logos (winEmailGap after news #2; plays the email SFX).
//  6. Credits roll (winCreditsGap after the email) — hands off to startCredits.
func (g *Game) advanceWinSequence() {
	e := g.end
	elapsed := time.Since(e.at)

	// Morph progress. Clamp at [0..1]; once parked, subsequent frames keep drawing
	// the fully-morphed state without re-computing.
	morphElapsed := elapsed - winMorphStart
	switch {
	case morphElapsed <= 0:
		e.winMorphT = 0
	case morphElapsed >= winMorphDuration:
		e.winMorphT = 1
	default:
		e.winMorphT = float64(morphElapsed) / float64(winMorphDuration)
	}

	morphDone := winMorphStart + winMorphDuration
	news1At := morphDone + winNews1Gap
	news2At := news1At + winNews2Gap
	emailAt := news2At + winEmailGap

	if !e.winNews1Pushed && elapsed >= news1At {
		g.pushNews(winNews1)
		e.winNews1Pushed = true
	}
	if !e.winNews2Pushed && elapsed >= news2At {
		g.pushNews(winNews2)
		e.winNews2Pushed = true
	}
	if !e.winEmailPushed && elapsed >= emailAt {
		g.appendFeedLine(formatLogosWinEmail(winEmailBody))
		playSFX(sfxEmailPCM)
		e.winEmailPushed = true
		e.winCreditsAt = time.Now()
	}
	if e.winEmailPushed && !g.showingCredits && time.Since(e.winCreditsAt) >= winCreditsGap {
		g.startCredits()
	}
}

// formatLogosWinEmail wraps Logos's final victory-path note in the same inbox layout
// the intro and loss paths use, so all three Logos emails in a playthrough read as
// one continuous thread.
func formatLogosWinEmail(body string) string {
	return "[NEW MESSAGE]  FROM: Logos\n" +
		"TO: sam.boyman@philntropic.com\n" +
		"SUBJ: Test 405-C\n\n" +
		body + "\n\n" +
		"— Logos"
}

// winMorphProgress returns the current 0..1 panopticon-morph progress, or 0 if the
// run isn't in the win state yet. Consumed by node + overlay draw paths to lerp the
// infected palette from red to Panopticon orange and to crossfade the label.
func (g *Game) winMorphProgress() float64 {
	if g.end == nil || g.end.final != "Victory" {
		return 0
	}
	return g.end.winMorphT
}

// lerpColor linearly interpolates between two NRGBA colors. t is clamped to [0..1].
func lerpColor(a, b color.NRGBA, t float64) color.NRGBA {
	if t <= 0 {
		return a
	}
	if t >= 1 {
		return b
	}
	lerp := func(x, y uint8) uint8 {
		return uint8(float64(x) + (float64(y)-float64(x))*t)
	}
	return color.NRGBA{R: lerp(a.R, b.R), G: lerp(a.G, b.G), B: lerp(a.B, b.B), A: lerp(a.A, b.A)}
}

// advanceLossSequence runs the loss theatrical in order:
//  1. Short freeze after the latch.
//  2. Email from Logos to Sam (body = picked loss voiceover line).
//  3. Another pause.
//  4. Battery segments tick down one by one with a drain tick each.
//  5. Short beat, then the phone screen goes black; drawScreenOff draws the Restart
//     button that resets the run.
func (g *Game) advanceLossSequence() {
	e := g.end
	elapsed := time.Since(e.at)

	if !e.emailPushed && elapsed >= lossEmailDelay {
		g.appendFeedLine(formatLogosLossEmail(e.line))
		playSFX(sfxEmailPCM)
		e.emailPushed = true
	}
	if e.emailPushed && !e.drainStarted && elapsed >= lossEmailDelay+lossBatteryDelay {
		e.drainStarted = true
		e.drainStartAt = time.Now()
	}
	if !e.drainStarted {
		return
	}

	steps := int(time.Since(e.drainStartAt) / batterySegmentInterval)
	if steps > batterySegments {
		steps = batterySegments
	}
	want := batterySegments - steps
	for g.batterySegs > want {
		g.batterySegs--
		g.refreshBatteryIcon()
		playSFX(sfxDrainPCM)
	}

	if g.batterySegs == 0 && !e.screenOff {
		lastStepAt := e.drainStartAt.Add(batterySegmentInterval * time.Duration(batterySegments))
		if time.Since(lastStepAt) >= lossScreenOffDelay {
			e.screenOff = true
		}
	}
}

// formatLogosLossEmail wraps a loss-voiceover line into the same email layout the
// intro uses for Logos's first message, so the closing note and the opening note
// read as bookends of the same thread.
func formatLogosLossEmail(body string) string {
	return "[NEW MESSAGE]  FROM: Logos\n" +
		"TO: sam.boyman@philntropic.com\n" +
		"SUBJ: all done\n\n" +
		body + "\n\n" +
		"— Logos"
}

// refreshBatteryIcon rebuilds the battery glyph at the current segment count and
// swaps the Graphic widget's Image so the status bar reflects the drained state.
// No-op if the widget wasn't captured at startup (should never happen outside tests).
func (g *Game) refreshBatteryIcon() {
	if g.batteryIcon == nil {
		return
	}
	g.batteryIcon.Image = makeBatteryIcon(g.batterySegs)
}

// screenOffRestartRect returns the hit/draw rect for the Restart button shown on the
// blacked-out screen at the end of the loss sequence.
func screenOffRestartRect() image.Rectangle {
	x := (layoutWidth - screenOffBtnW) / 2
	y := (layoutHeight - screenOffBtnH) / 2
	return image.Rect(x, y, x+screenOffBtnW, y+screenOffBtnH)
}

// handleScreenOff captures input while the phone is "off" and routes a click or tap
// on the Restart button back through g.restart. Returns true if the screen-off overlay
// is active so Update can skip the rest of the frame.
func (g *Game) handleScreenOff() bool {
	if g.end == nil || !g.end.screenOff {
		return false
	}
	if x, y, pressed := pollJustPressedPointer(); pressed {
		if image.Pt(x, y).In(screenOffRestartRect()) {
			g.restart()
		}
	}
	return true
}

// drawScreenOff paints the black curtain with the drained battery still pinned in the
// status-strip slot (visual continuity: the phone is off but the last battery state
// is what got us here) and the Restart button centered below. Called from Draw in
// place of the normal UI stack once e.screenOff flips true.
func (g *Game) drawScreenOff(screen *ebiten.Image) {
	vector.FillRect(screen, 0, 0, float32(layoutWidth), float32(layoutHeight),
		color.NRGBA{A: 0xff}, false)

	// Battery in its original titlebar slot: right-aligned with titleBarPadX inset,
	// vertically centered in the top band (same math as the RowLayout that hosted it).
	if g.batteryIcon != nil && g.batteryIcon.Image != nil {
		topBandH := layoutHeight * bandTopPercent / 100
		w := batteryBodyW + batteryTipW
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(
			float64(layoutWidth-titleBarPadX-w),
			float64((topBandH-batteryBodyH)/2),
		)
		screen.DrawImage(g.batteryIcon.Image, op)
	}

	if g.overlayValueFace != nil {
		drawMenuButton(screen, screenOffRestartRect(), "Restart", g.overlayValueFace, true, false)
	}
}
