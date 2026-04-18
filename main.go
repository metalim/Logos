package main

import (
	"bytes"
	"image/color"
	"log"

	"github.com/ebitenui/ebitenui"
	eimage "github.com/ebitenui/ebitenui/image"
	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gomono"
)

const sampleNews = `• SAN FIASCO — Sahara Web Services reports cascading latency after an emergency routing patch.

• MacroFrame advisory: reboot required for Overcast edge nodes; GDP forecasts cut 0.4pp.

• Logos containment update: BootLoop perimeter holds; equities whipsaw on hourly headlines.

• Giggle Janus query throttling extended. Analysts expect a statement within the hour.

• Fiasco Systems: cross-Pacific routes stabilized after manual drain of poisoned AS paths.`

const (
	screenWidth  = 360
	screenHeight = 640
)

// Game is the root game state. Extend this struct with your systems and assets.
type Game struct {
	ui         *ebitenui.UI
	root       *widget.Container
	statusBar  *widget.Container
	mapPanel   *widget.Container
	feedScroll *widget.ScrollContainer
	newsText   *widget.Text

	lastW int
	lastH int
}

func loadFont(size float64) (text.Face, error) {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(gomono.TTF))
	if err != nil {
		return nil, err
	}
	return &text.GoTextFace{Source: src, Size: size}, nil
}

func newGame() (*Game, error) {
	face, err := loadFont(14)
	if err != nil {
		return nil, err
	}

	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.BackgroundImage(
			eimage.NewNineSliceColor(color.NRGBA{R: 0x12, G: 0x12, B: 0x14, A: 0xff}),
		),
	)

	statusBar := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.BackgroundImage(
			eimage.NewNineSliceColor(color.NRGBA{R: 0x22, G: 0x24, B: 0x2a, A: 0xff}),
		),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionStart,
				VerticalPosition:   widget.AnchorLayoutPositionStart,
				StretchHorizontal:  true,
				StretchVertical:    false,
			}),
			widget.WidgetOpts.MinSize(0, 1),
		),
	)

	mapPanel := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewAnchorLayout()),
		widget.ContainerOpts.BackgroundImage(
			eimage.NewNineSliceColor(color.NRGBA{R: 0x2c, G: 0x2e, B: 0x34, A: 0xff}),
		),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionStart,
				VerticalPosition:   widget.AnchorLayoutPositionStart,
				StretchHorizontal:  true,
				StretchVertical:    false,
				Padding:            &widget.Insets{Top: 1},
			}),
			widget.WidgetOpts.MinSize(0, 1),
		),
	)

	newsText := widget.NewText(
		widget.TextOpts.Text(sampleNews, &face, color.NRGBA{R: 0xe8, G: 0xea, B: 0xf0, A: 0xff}),
		widget.TextOpts.MaxWidth(300),
		widget.TextOpts.Padding(widget.NewInsetsSimple(8)),
	)

	feedScroll := widget.NewScrollContainer(
		widget.ScrollContainerOpts.Content(newsText),
		widget.ScrollContainerOpts.StretchContentWidth(),
		widget.ScrollContainerOpts.Image(&widget.ScrollContainerImage{
			Idle: eimage.NewNineSliceColor(color.NRGBA{R: 0x18, G: 0x1a, B: 0x20, A: 0xff}),
			Mask: eimage.NewNineSliceColor(color.NRGBA{R: 0x18, G: 0x1a, B: 0x20, A: 0xff}),
		}),
		widget.ScrollContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionStart,
				VerticalPosition:   widget.AnchorLayoutPositionStart,
				StretchHorizontal:  true,
				StretchVertical:    false,
				Padding:            &widget.Insets{Top: 1},
			}),
			widget.WidgetOpts.MinSize(0, 1),
		),
	)

	root.AddChild(statusBar)
	root.AddChild(mapPanel)
	root.AddChild(feedScroll)

	g := &Game{
		ui: &ebitenui.UI{
			Container: root,
		},
		root:       root,
		statusBar:  statusBar,
		mapPanel:   mapPanel,
		feedScroll: feedScroll,
		newsText:   newsText,
	}
	return g, nil
}

func (g *Game) applyVerticalBands(outsideW, outsideH int) {
	if outsideW == g.lastW && outsideH == g.lastH {
		return
	}
	g.lastW, g.lastH = outsideW, outsideH
	if outsideH <= 0 || outsideW <= 0 {
		return
	}

	h1 := outsideH * 10 / 100
	h2 := outsideH * 60 / 100
	h3 := outsideH - h1 - h2

	st := g.statusBar.GetWidget()
	st.LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		StretchHorizontal:  true,
		StretchVertical:    false,
	}
	st.MinHeight = h1

	mp := g.mapPanel.GetWidget()
	mp.LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		StretchHorizontal:  true,
		StretchVertical:    false,
		Padding:            &widget.Insets{Top: h1},
	}
	mp.MinHeight = h2

	fs := g.feedScroll.GetWidget()
	fs.LayoutData = widget.AnchorLayoutData{
		HorizontalPosition: widget.AnchorLayoutPositionStart,
		VerticalPosition:   widget.AnchorLayoutPositionStart,
		StretchHorizontal:  true,
		StretchVertical:    false,
		Padding:            &widget.Insets{Top: h1 + h2},
	}
	fs.MinHeight = h3

	mw := float64(outsideW - 20)
	if mw < 40 {
		mw = 40
	}
	g.newsText.MaxWidth = mw

	g.root.RequestRelayout()
}

func (g *Game) Update() error {
	g.ui.Update()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.ui.Draw(screen)
}

func (g *Game) Layout(_, _ int) (int, int) {
	g.applyVerticalBands(screenWidth, screenHeight)
	return screenWidth, screenHeight
}

func main() {
	ebiten.SetWindowTitle("Zero-Day Lunch")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
	ebiten.SetWindowSize(screenWidth, screenHeight)

	g, err := newGame()
	if err != nil {
		log.Fatal(err)
	}

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
