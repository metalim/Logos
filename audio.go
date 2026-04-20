package main

import (
	"bytes"
	_ "embed"
	"io"
	"log"

	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/mp3"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

// musicBytes holds the in-game loop track embedded at build time. The embed pattern
// is quoted because the source filename contains a space.
//go:embed "assets/neon firewall.mp3"
var musicBytes []byte

// SFX blobs. Filenames match bfxr export conventions: silent placeholders live in
// the repo so the build never breaks; drop real wavs in-place to replace them.
//
//go:embed assets/sfx/blip.wav
var sfxBlipBytes []byte

//go:embed assets/sfx/boom.wav
var sfxBoomBytes []byte

//go:embed assets/sfx/power.wav
var sfxPowerBytes []byte

//go:embed assets/sfx/pickup.wav
var sfxPickupBytes []byte

//go:embed assets/sfx/drain.wav
var sfxDrainBytes []byte

//go:embed assets/sfx/email.wav
var sfxEmailBytes []byte

// audioSampleRate is the single rate the whole game mixes at. 48 kHz matches the
// source MP3 and is the preferred WebAudio rate, so the browser mixer doesn't have
// to resample.
const audioSampleRate = 48000

// audioCtx is the process-wide audio.Context. ebiten forbids more than one context
// per process, so we create it lazily on first use and reuse forever.
var audioCtx *audio.Context

// Decoded PCM buffers for one-shot SFX, filled once by ensureAudioCtx. NewPlayerFromBytes
// over a shared buffer is cheap, so a rapid-fire sequence (e.g. "+1" spam from multiple
// security nodes) creates a new ephemeral player per hit without re-decoding the source.
var (
	sfxBlipPCM   []byte
	sfxBoomPCM   []byte
	sfxPowerPCM  []byte
	sfxPickupPCM []byte
	sfxDrainPCM  []byte
	sfxEmailPCM  []byte
)

func ensureAudioCtx() *audio.Context {
	if audioCtx != nil {
		return audioCtx
	}
	audioCtx = audio.NewContext(audioSampleRate)
	sfxBlipPCM = decodeWAV(sfxBlipBytes)
	sfxBoomPCM = decodeWAV(sfxBoomBytes)
	sfxPowerPCM = decodeWAV(sfxPowerBytes)
	sfxPickupPCM = decodeWAV(sfxPickupBytes)
	sfxDrainPCM = decodeWAV(sfxDrainBytes)
	sfxEmailPCM = decodeWAV(sfxEmailBytes)
	return audioCtx
}

// decodeWAV decodes a WAV blob to raw PCM bytes at audioSampleRate, ready to be fed
// to Context.NewPlayerFromBytes. Returns nil on any error so callers can no-op the
// playback path — an SFX hiccup should never take down the run.
func decodeWAV(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}
	stream, err := wav.DecodeWithSampleRate(audioSampleRate, bytes.NewReader(data))
	if err != nil {
		log.Printf("sfx: decode failed: %v", err)
		return nil
	}
	pcm, err := io.ReadAll(stream)
	if err != nil {
		log.Printf("sfx: read failed: %v", err)
		return nil
	}
	return pcm
}

// playSFX fires a one-shot player for the given PCM buffer. Each call creates a new
// ephemeral Player so overlapping SFX don't cut each other off; the GC collects them
// once playback finishes.
func playSFX(pcm []byte) {
	if len(pcm) == 0 || audioCtx == nil {
		return
	}
	p := audioCtx.NewPlayerFromBytes(pcm)
	p.Play()
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
