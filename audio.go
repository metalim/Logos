package main

import (
	"bytes"
	_ "embed"
	"log"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
)

// musicBytes holds the in-game loop track embedded at build time. The embed pattern
// is quoted because the source filename contains a space.
//go:embed "assets/neon firewall.mp3"
var musicBytes []byte

// audioSampleRate is the single rate the whole game mixes at. 48 kHz matches the
// source MP3 and is the preferred WebAudio rate, so the browser mixer doesn't have
// to resample.
const audioSampleRate = 48000

// audioCtx is the process-wide audio.Context. ebiten forbids more than one context
// per process, so we create it lazily on first use and reuse forever.
var audioCtx *audio.Context

func ensureAudioCtx() *audio.Context {
	if audioCtx == nil {
		audioCtx = audio.NewContext(audioSampleRate)
	}
	return audioCtx
}

// startMusic (re)creates an infinite-loop MP3 player for the in-game track and starts
// playback. Any previously running player is closed first so a restart doesn't leave
// a detached player running in the background. Errors are logged and swallowed —
// music is ambiance, not gameplay, and we don't want a decode hiccup to kill the run.
func (g *Game) startMusic() {
	g.stopMusic()
	ctx := ensureAudioCtx()
	stream, err := mp3.DecodeWithSampleRate(audioSampleRate, bytes.NewReader(musicBytes))
	if err != nil {
		log.Printf("music: decode failed: %v", err)
		return
	}
	loop := audio.NewInfiniteLoop(stream, stream.Length())
	p, err := ctx.NewPlayer(loop)
	if err != nil {
		log.Printf("music: new player failed: %v", err)
		return
	}
	g.musicPlayer = p
	p.Play()
}

// stopMusic halts and releases the current player. Safe to call when no player is
// active — that's the common case on startup and right after a fresh restart.
func (g *Game) stopMusic() {
	if g.musicPlayer == nil {
		return
	}
	_ = g.musicPlayer.Close()
	g.musicPlayer = nil
}
