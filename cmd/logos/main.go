// Command logos is the desktop / WASM entry point for the game. All actual game
// logic lives in internal/game; this binary stays a one-liner so adding a second
// host (alternate frontend, headless eval, tests) doesn't require touching the
// game package.
package main

import "github.com/metalim/Logos/internal/game"

func main() {
	game.Run()
}
