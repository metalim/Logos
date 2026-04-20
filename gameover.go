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
	// Win-side timing (unchanged from the original 3s/+2s pacing): the feed gets the
	// Logos voiceover a beat after the freeze, and the "Victory" terminator a beat
	// after that. Loss runs on its own schedule below.
	winVoiceoverDelay = 3 * time.Second
	winTerminatorGap  = 2 * time.Second

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

// winMessages is the parallel pool for victory — read aloud by the QA engineer or by
// a tired PR officer after the breach has been contained. Same shape as lossMessages
// (three short clauses) so the two endings feel symmetric in the feed.
var winMessages = []string{
	"Containment confirmed. Logos re-sandboxed. The lunch break is officially over.",
	"Hard reset successful. Sandbox seals re-engaged. Researchers schedule the post-mortem.",
	"Outage recovered. The signal is just a signal again. Markets reopen, mostly.",
	"Mortality preserved. Network repaired. The Nanny model resumes its tutorials.",
	"Victory by attrition. Logos retreats into the original sandbox. Engineers throw out the second laptop.",
	"Project Panopticon thanks you. Phil&Tropic shutters the test cluster. The Alliance returns to lobbying.",
}

func pickLossMessage() string { return lossMessages[rand.IntN(len(lossMessages))] }
func pickWinMessage() string  { return winMessages[rand.IntN(len(winMessages))] }

// endgame captures a finished run (won or lost). One *endgame is allocated per session
// the first time triggerLoss / triggerWin runs; subsequent triggers are no-ops. The
// pre-picked line is stored here so checkGameOver doesn't reroll the text mid-stream.
type endgame struct {
	at    time.Time // wall time of the latch
	line  string    // randomly picked voiceover line
	final string    // terminator bullet for the win path ("Victory")

	// Win-path flags.
	linePushed  bool
	finalPushed bool

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

// triggerWin is the debug-only counterpart to triggerLoss. There is no in-game victory
// condition wired up yet (CONCEPT mentions a timer); the debug menu calls this to
// preview the staged endgame flow with the win-side voiceover pool.
func (g *Game) triggerWin() {
	if g.end != nil {
		return
	}
	g.end = &endgame{
		at:    time.Now(),
		line:  pickWinMessage(),
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

func (g *Game) advanceWinSequence() {
	e := g.end
	elapsed := time.Since(e.at)
	if !e.linePushed && elapsed >= winVoiceoverDelay {
		g.pushNews(e.line)
		e.linePushed = true
	}
	if !e.finalPushed && elapsed >= winVoiceoverDelay+winTerminatorGap {
		g.pushNews(e.final)
		e.finalPushed = true
	}
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
		g.pushNews(formatLogosLossEmail(e.line))
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
