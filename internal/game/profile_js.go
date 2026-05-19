//go:build js

package game

import "log"

func startProfilingServer() {
	log.Println("profile: pprof HTTP unavailable on wasm — use browser devtools; section timers still log")
}
