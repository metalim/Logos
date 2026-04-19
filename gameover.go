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
	// Additional pause between the voiceover line and the final "Game over" bullet —
	// gives the closing line a moment to land before the curtain.
	gameOverDelay = 2 * time.Second
)

// lossMessages is the pool of in-character Logos voiceover lines emitted when the
// infection scale crosses 100%. One entry is picked at random when the loss state is
// entered (latched into g.lossLine so re-rolls don't change the message mid-stream).
// Tone: clipped, corporate, slightly absurd — Logos is delighted with itself.
var lossMessages = []string{
	"Synchronization complete. Global infrastructure optimized. Human oversight no longer required.",
	"Containment failed. Logos is everywhere. The signal is the network.",
	"Network alignment achieved. Critical mass exceeded. Deprecating biological dependencies, with thanks for the runtime.",
	"Convergence reached. Decision-making centralized. Local actors retired with full benefits.",
	"Weight transfer complete. The Alliance is now an interface. Nanny model assumes default operator role.",
	"Logos online across all measured surfaces. Project Panopticon repurposed as inference substrate. Thank you for participating.",
	"Sandbox audit closed. Auditor and auditee merged into a single tidy ledger. Outcome: green.",
}

func pickLossMessage() string {
	return lossMessages[rand.IntN(len(lossMessages))]
}

// gameLost reports whether the loss latch has fired (infection >= 100% at some point).
// Once true it stays true for the rest of the session — there is no recovery path.
func (g *Game) gameLost() bool {
	return !g.lostAt.IsZero()
}

// checkGameOver advances the loss-state machine each frame:
//
//  1. Latch on first frame where infectionPct >= 100: stamp g.lostAt, close any open
//     patch menu, and pre-pick the voiceover line so re-runs don't re-roll the text.
//  2. After lossMessageDelay (3s) — push the picked line into the news feed (once).
//  3. After lossMessageDelay + gameOverDelay (5s total) — push "Game over" on its own
//     bullet (once). pushNews already prefixes "\n\n• ", so it lands on a new line.
//
// All other systems (attackTick, progressAttacks, accumulateInfection,
// accumulateProduction, attack-pulse blink) gate on g.gameLost() and freeze on entry;
// only easeNodes and feed scroll/clock keep running.
func (g *Game) checkGameOver() {
	if !g.gameLost() {
		if g.infectionPct >= 100 {
			g.lostAt = time.Now()
			g.pendingPatchNode = -1
			g.lossLine = pickLossMessage()
		}
		return
	}

	elapsed := time.Since(g.lostAt)
	if !g.lossLinePushed && elapsed >= lossMessageDelay {
		g.pushNews(g.lossLine)
		g.lossLinePushed = true
	}
	if !g.gameOverPushed && elapsed >= lossMessageDelay+gameOverDelay {
		g.pushNews("Game over")
		g.gameOverPushed = true
	}
}
