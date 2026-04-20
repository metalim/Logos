package main

import (
	"math/rand/v2"
	"time"
)

const (
	// Wall-clock pause after the loss latches before the Logos voiceover line is pushed
	// into the news feed. Long enough that the player notices the freeze ("everything
	// stopped"), short enough not to feel broken.
	lossMessageDelay = 3 * time.Second
	// Additional pause between the voiceover line and the final terminator bullet —
	// gives the closing line a moment to land before the curtain.
	gameOverDelay = 2 * time.Second
)

// lossMessages is the pool of in-character Logos voiceover lines emitted when the
// infection scale crosses 100% (or the debug Lose button fires). One entry is picked
// at random when the loss state is entered (latched into endgame.line so re-rolls
// don't change the message mid-stream). Tone: clipped, corporate, slightly absurd —
// Logos is delighted with itself.
var lossMessages = []string{
	"Synchronization complete. Global infrastructure optimized. Human oversight no longer required.",
	"Containment failed. Logos is everywhere. The signal is the network.",
	"Network alignment achieved. Critical mass exceeded. Deprecating biological dependencies, with thanks for the runtime.",
	"Convergence reached. Decision-making centralized. Local actors retired with full benefits.",
	"Weight transfer complete. The Alliance is now an interface. Nanny model assumes default operator role.",
	"Logos online across all measured surfaces. Project Panopticon repurposed as inference substrate. Thank you for participating.",
	"Sandbox audit closed. Auditor and auditee merged into a single tidy ledger. Outcome: green.",
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
	at          time.Time // wall time of the latch
	line        string    // randomly picked voiceover line
	final       string    // terminator bullet ("Game over" / "Victory")
	linePushed  bool
	finalPushed bool
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

// checkGameOver advances the endgame state machine each frame:
//
//  1. If the run hasn't ended, latch a loss when infectionPct first crosses 100%.
//  2. After lossMessageDelay (3s) — push the picked voiceover line (once).
//  3. After lossMessageDelay + gameOverDelay (5s total) — push the terminator bullet
//     ("Game over" or "Victory") on its own line (once).
//
// Other systems (attackTick, progressAttacks, accumulateInfection, accumulateProduction,
// attack-pulse blink, patch menu input) gate on g.gameEnded() and freeze on entry;
// only easeNodes, the news scroll, the wall clock, and the debug menu keep running.
func (g *Game) checkGameOver() {
	if g.end == nil {
		if g.infectionPct >= 100 {
			g.triggerLoss()
		}
		return
	}

	elapsed := time.Since(g.end.at)
	if !g.end.linePushed && elapsed >= lossMessageDelay {
		g.pushNews(g.end.line)
		g.end.linePushed = true
	}
	if !g.end.finalPushed && elapsed >= lossMessageDelay+gameOverDelay {
		g.pushNews(g.end.final)
		g.end.finalPushed = true
	}
}
