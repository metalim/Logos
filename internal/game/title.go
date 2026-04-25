package game

import (
	"bytes"
	_ "embed"
	"image"
	_ "image/jpeg"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// coverBytes holds the portrait cover art embedded at build time. The embed pattern
// is quoted because the source filename contains a space.
//go:embed "assets/cover 9x16.jpg"
var coverBytes []byte

// coverImage is the decoded ebiten image, cached after first successful decode.
// nil means "not loaded yet or decode failed"; loadCoverImage retries on every call
// until it either succeeds or returns the cached error via a nil result.
var coverImage *ebiten.Image

func loadCoverImage() *ebiten.Image {
	if coverImage != nil {
		return coverImage
	}
	img, _, err := image.Decode(bytes.NewReader(coverBytes))
	if err != nil {
		return nil
	}
	coverImage = ebiten.NewImageFromImage(img)
	return coverImage
}

// handleTitleScreen dismisses the title on any mouse click, touch, or keypress.
// Returns true while the title is still up so Update can skip the sim.
func (g *Game) handleTitleScreen() bool {
	if !g.showTitle {
		return false
	}
	dismissed := inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) ||
		len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 ||
		len(inpututil.AppendJustPressedKeys(nil)) > 0
	if dismissed {
		g.dismissTitle()
	}
	return g.showTitle
}

// dismissTitle hides the cover and re-stamps the sim clocks so the time the player
// spent looking at the title doesn't count against the first-attack schedule or the
// containment curve. resetGameState already stamped these once in newGame, but that
// was N frames ago.
func (g *Game) dismissTitle() {
	g.showTitle = false
	now := time.Now()
	g.epoch = now
	g.lastAttackAt = now
	g.lastSimTick = now
	g.feedScrollLastSmooth = now
	g.startMusic()
}

// drawTitleScreen paints the cover scaled to fill the entire layout. The asset is
// exactly 9:16 so it matches layoutWidth:layoutHeight (900:1600) without letterboxing;
// the scale is still computed from the actual source dimensions so a re-exported cover
// at a different resolution stretches the same way.
func (g *Game) drawTitleScreen(screen *ebiten.Image) {
	img := loadCoverImage()
	if img == nil {
		return
	}
	sw, sh := screen.Bounds().Dx(), screen.Bounds().Dy()
	iw, ih := img.Bounds().Dx(), img.Bounds().Dy()
	if iw == 0 || ih == 0 {
		return
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(float64(sw)/float64(iw), float64(sh)/float64(ih))
	screen.DrawImage(img, op)
}
