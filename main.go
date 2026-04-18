package main

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"math"

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

	testNewsEverySecs = 5.0

	// Higher = faster approach to feedScrollTarget (smooth auto-scroll).
	feedScrollSmoothPerSec = 9.0
	// When already at bottom, step back this much so smooth scroll can run toward new bottom.
	feedScrollBottomNudge = 0.14
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

	// Auto-scroll target: ScrollTop in0..1; smoothed in Update toward this value.
	feedScrollTarget float64

	testNewsTimer  float64
	testNewsSerial int
}

func loadFont(size float64) (text.Face, error) {
	src, err := text.NewGoTextFaceSource(bytes.NewReader(gomono.TTF))
	if err != nil {
		return nil, err
	}
	return &text.GoTextFace{Source: src, Size: size}, nil
}

// ebitenui ScrollContainer does not wire the wheel by default (see TextArea for the pattern).
func wireFeedScrollWheel(g *Game) {
	g.feedScroll.GetWidget().ScrolledEvent.AddHandler(func(args interface{}) {
		a, ok := args.(*widget.WidgetScrolledEventArgs)
		if !ok {
			return
		}
		scroll := g.feedScroll
		text := g.newsText
		viewH := float64(scroll.ViewRect().Dy())
		_, ch := text.PreferredSize()
		if float64(ch) <= viewH {
			return
		}
		page := int(math.Round(viewH / float64(ch) * 1000))
		p := page / 3
		if p < 1 {
			p = 1
		}
		scroll.ScrollTop -= a.Y * float64(p) / 1000
		if scroll.ScrollTop < 0 {
			scroll.ScrollTop = 0
		}
		if scroll.ScrollTop > 1 {
			scroll.ScrollTop = 1
		}
		g.feedScrollTarget = scroll.ScrollTop
	})
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
				StretchVertical:    true,
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
	g.feedScrollTarget = 1
	g.feedScroll.ScrollTop = 1
	wireFeedScrollWheel(g)
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
		StretchVertical:    true,
		Padding:            &widget.Insets{Top: h1 + h2},
	}
	fs.MinHeight = h3

	mw := float64(outsideW - 20)
	if mw < 40 {
		mw = 40
	}
	g.newsText.MaxWidth = mw

	g.root.RequestRelayout()
	g.requestFeedScrollBottom()
}

func (g *Game) requestFeedScrollBottom() {
	g.feedScrollTarget = 1
}

// nudgeFeedScrollIfPinnedToBottom moves ScrollTop slightly up when already at the end so
// smoothFeedScroll has a non-zero delta after content height increases (ScrollTop=1 and target=1 gives diff=0).
func (g *Game) nudgeFeedScrollIfPinnedToBottom() {
	const eps = 1e-3
	if g.feedScroll == nil {
		return
	}
	if g.feedScroll.ScrollTop < 1-eps || g.feedScrollTarget < 1-eps {
		return
	}
	g.feedScroll.ScrollTop = math.Max(0, g.feedScroll.ScrollTop-feedScrollBottomNudge)
}

func (g *Game) smoothFeedScroll(dt float64) {
	if g.feedScroll == nil {
		return
	}
	cur := g.feedScroll.ScrollTop
	tgt := g.feedScrollTarget
	diff := tgt - cur
	if math.Abs(diff) < 1e-4 {
		g.feedScroll.ScrollTop = tgt
		return
	}
	alpha := 1 - math.Exp(-feedScrollSmoothPerSec*dt)
	g.feedScroll.ScrollTop += diff * alpha
	if g.feedScroll.ScrollTop < 0 {
		g.feedScroll.ScrollTop = 0
	}
	if g.feedScroll.ScrollTop > 1 {
		g.feedScroll.ScrollTop = 1
	}
}

func (g *Game) pushTestNews() {
	g.testNewsSerial++
	line := fmt.Sprintf(
		"\n\n• [TEST %d] Simulated wire: Janus ingest queue +%d; timer tick.",
		g.testNewsSerial,
		g.testNewsSerial*7%97,
	)
	g.newsText.Label += line
	g.root.RequestRelayout()
	g.nudgeFeedScrollIfPinnedToBottom()
	g.requestFeedScrollBottom()
}

func (g *Game) Update() error {
	dt := 1.0 / 60.0
	if tps := ebiten.ActualTPS(); tps > 0 {
		dt = 1.0 / tps
	}
	g.testNewsTimer += dt
	for g.testNewsTimer >= testNewsEverySecs {
		g.testNewsTimer -= testNewsEverySecs
		g.pushTestNews()
	}

	g.ui.Update()
	g.smoothFeedScroll(dt)
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
