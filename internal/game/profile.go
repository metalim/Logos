package game

import (
	"fmt"
	"log"
	"sort"
	"time"
)

// profilingEnabled turns on in-game section timers (stdout) and, on native builds,
// an HTTP pprof server. Flip to true locally, rebuild, play the main loop for a
// few seconds, then capture CPU:
//
//	go tool pprof -http=:8081 'http://127.0.0.1:6060/debug/pprof/profile?seconds=30'
//
// Section keys are hierarchical (map.edges, map.labels.measure, …). Nested timers
// overlap — e.g. feed.stepSmooth includes feed.PreferredSize, and map.labels includes
// map.labels.measure + map.labels.draw. Phase totals sum every key; use leaf rows for
// where time actually goes.
//
// Shipped / itch builds stay false — zero overhead when off.
const (
	profilingEnabled   = false
	profilingAddr      = "127.0.0.1:6060"
	profileReportEvery = 1 * time.Second
)

var (
	profileFrameActive bool
	profileFrameStart  time.Time
	profileCurUpdate   map[string]time.Duration
	profileCurDraw     map[string]time.Duration
	profileTotUpdate   map[string]time.Duration
	profileTotDraw     map[string]time.Duration
	profileTotWall     time.Duration
	profileFrames      int

	// Snapshot from the last drawNodeMap in each profiled frame (max seen per window).
	profileNodes, profileEdges int

	profileReportAt time.Time
)

func initProfiling() {
	if !profilingEnabled {
		return
	}
	profileCurUpdate = make(map[string]time.Duration)
	profileCurDraw = make(map[string]time.Duration)
	profileTotUpdate = make(map[string]time.Duration)
	profileTotDraw = make(map[string]time.Duration)
	profileReportAt = time.Now()
	startProfilingServer()
	log.Printf("profile: enabled — report every %s; pprof http://%s/debug/pprof/", profileReportEvery, profilingAddr)
}

func (g *Game) profileGameplayFrame() bool {
	return !g.showTitle && !g.showingCredits && (g.end == nil || !g.end.screenOff)
}

func profileMaybeBeginFrame(g *Game) {
	if !profilingEnabled || !g.profileGameplayFrame() {
		profileFrameActive = false
		return
	}
	profileFrameActive = true
	profileFrameStart = time.Now()
	clear(profileCurUpdate)
	clear(profileCurDraw)
}

func profileSection(phase, name string) func() {
	if !profileFrameActive {
		return func() {}
	}
	start := time.Now()
	return func() {
		d := time.Since(start)
		if phase == "draw" {
			profileCurDraw[name] += d
		} else {
			profileCurUpdate[name] += d
		}
	}
}

func profileNoteMapSize(nodes, edges int) {
	if !profileFrameActive {
		return
	}
	if nodes > profileNodes {
		profileNodes = nodes
	}
	if edges > profileEdges {
		profileEdges = edges
	}
}

func profileEndFrame() {
	if !profileFrameActive {
		return
	}
	profileFrameActive = false
	profileFrames++
	profileTotWall += time.Since(profileFrameStart)
	for k, v := range profileCurUpdate {
		profileTotUpdate[k] += v
	}
	for k, v := range profileCurDraw {
		profileTotDraw[k] += v
	}
	if time.Since(profileReportAt) < profileReportEvery {
		return
	}
	profileEmitReport()
	profileFrames = 0
	profileTotWall = 0
	clear(profileTotUpdate)
	clear(profileTotDraw)
	profileNodes = 0
	profileEdges = 0
	profileReportAt = time.Now()
}

func profileEmitReport() {
	if profileFrames == 0 {
		return
	}
	frames := float64(profileFrames)
	var updTotal, drawTotal time.Duration
	for _, d := range profileTotUpdate {
		updTotal += d
	}
	for _, d := range profileTotDraw {
		drawTotal += d
	}
	log.Printf("profile: %d gameplay frame(s) — wall %s | instrumented update %s + draw %s | map ≤%d nodes %d edges",
		profileFrames,
		profileDurPerFrame(profileTotWall, frames),
		profileDurPerFrame(updTotal, frames),
		profileDurPerFrame(drawTotal, frames),
		profileNodes, profileEdges,
	)
	profileEmitPhase("update", profileTotUpdate, frames)
	profileEmitPhase("draw", profileTotDraw, frames)
}

func profileEmitPhase(phase string, totals map[string]time.Duration, frames float64) {
	if len(totals) == 0 {
		return
	}
	var total time.Duration
	for _, d := range totals {
		total += d
	}
	if total == 0 {
		return
	}
	names := make([]string, 0, len(totals))
	for name := range totals {
		names = append(names, name)
	}
	sort.Strings(names)
	log.Printf("profile %s (keys sum %s/frame — nested keys overlap):", phase, profileDurPerFrame(total, frames))
	for _, name := range names {
		d := totals[name]
		pct := 100 * float64(d) / float64(total)
		log.Printf("  %-24s %s/frame  %5.1f%%", name, profileDurPerFrame(d, frames), pct)
	}
}

func profileDurPerFrame(d time.Duration, frames float64) string {
	if frames <= 0 {
		return "n/a"
	}
	perFrame := float64(d) / frames
	ms := perFrame / float64(time.Millisecond)
	if ms >= 0.01 {
		return fmt.Sprintf("%.2f ms", ms)
	}
	us := perFrame / float64(time.Microsecond)
	return fmt.Sprintf("%.0f µs", us)
}
