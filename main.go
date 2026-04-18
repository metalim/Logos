package main

import (
	"bytes"
	"fmt"
	"image/color"
	"log"
	"math"
	"strings"
	"time"

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

	testNewsInterval = 5 * time.Second
	// feedScrollPx moves each frame by a fraction of (targetPx - feedScrollPx); lambda scales with dt (~seconds^-1).
	feedScrollLambda = 14.0
)

// feedWheelContentPixelsPerUnit lives in wheel_native.go / wheel_js.go: WidgetScrolledEventArgs.Y is in
// GLFW scroll units on desktop (~tenths of a line) and in DOM pixels on js (deltaMode ignored upstream).

func clampUnitInterval(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

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

	// Normalized scroll goal 0..1; pixel offset feedScrollPx eases toward feedScrollTarget * extra.
	feedScrollTarget float64
	// Smoothed vertical content offset (px). Negative until first sync from the widget.
	feedScrollPx float64
	// Last time smoothFeedScroll ran (for dt); proportional step toward target each frame.
	feedScrollLastSmooth time.Time
	// Set when layout/text changed; requestFeedScrollBottom runs after ui.Update (PreferredSize is unsafe during Layout).
	feedScrollNeedBottom bool

	testNewsSerial int
	lastTestNews   time.Time
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
		extra, _, haveSlack := g.feedScrollSlack()
		if !haveSlack || extra <= 0 {
			return
		}
		contentPx := a.Y * feedWheelContentPixelsPerUnit
		if contentPx != 0 && math.Abs(contentPx) < 1 {
			contentPx = math.Copysign(1, contentPx)
		}
		delta := contentPx / extra
		g.feedScrollTarget = clampUnitInterval(g.feedScrollTarget - delta)
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
		widget.TextOpts.Text(strings.Repeat(sampleNews, 10), &face, color.NRGBA{R: 0xe8, G: 0xea, B: 0xf0, A: 0xff}),
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
	g.feedScrollPx = -1
	g.lastTestNews = time.Now()
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
	g.feedScrollNeedBottom = true
}

func (g *Game) requestFeedScrollBottom() {
	g.feedScrollTarget = 1
	extra, _, ok := g.feedScrollSlack()
	if !ok || extra <= 0 {
		return
	}
	if g.feedScrollPx < 0 {
		g.feedScrollPx = g.feedScroll.ScrollTop * extra
	}
}

// feedScrollSlack is content height minus viewport height (pixels scrollable), or ok false if widgets missing.
func (g *Game) feedScrollSlack() (extra, viewH float64, ok bool) {
	if g.feedScroll == nil || g.newsText == nil {
		return 0, 0, false
	}
	_, ch := g.newsText.PreferredSize()
	viewH = float64(g.feedScroll.ViewRect().Dy())
	extra = float64(ch) - viewH
	return extra, viewH, true
}

// stepSmoothFeedScroll moves feedScrollPx toward feedScrollTarget*extra and writes ScrollTop.
// Used every frame after ui.Update for both auto “scroll to bottom” and manual wheel (target-only) input.
func (g *Game) stepSmoothFeedScroll() {
	extra, _, ok := g.feedScrollSlack()
	if !ok || extra <= 0 {
		if g.feedScroll != nil {
			g.feedScroll.ScrollTop = 0
		}
		g.feedScrollPx = 0
		return
	}

	targetPx := g.feedScrollTarget * extra
	if g.feedScrollPx < 0 {
		g.feedScrollPx = g.feedScroll.ScrollTop * extra
	}
	if g.feedScrollPx > extra {
		g.feedScrollPx = extra
	}

	dt := time.Since(g.feedScrollLastSmooth).Seconds()
	g.feedScrollLastSmooth = time.Now()
	if dt <= 0 || dt > 0.2 {
		dt = 1.0 / 60.0
	}
	k := 1 - math.Exp(-feedScrollLambda*dt)
	diff := targetPx - g.feedScrollPx
	if math.Abs(diff) < 0.25 {
		g.feedScrollPx = targetPx
	} else {
		g.feedScrollPx += diff * k
	}

	g.feedScroll.ScrollTop = g.feedScrollPx / extra
	if g.feedScroll.ScrollTop < 0 {
		g.feedScroll.ScrollTop = 0
		g.feedScrollPx = 0
	}
	if g.feedScroll.ScrollTop > 1 {
		g.feedScroll.ScrollTop = 1
		g.feedScrollPx = extra
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
	g.feedScrollNeedBottom = true
}

func (g *Game) Update() error {
	if time.Since(g.lastTestNews) >= testNewsInterval {
		g.lastTestNews = time.Now()
		g.pushTestNews()
	}

	g.ui.Update()
	if g.feedScrollNeedBottom {
		g.feedScrollNeedBottom = false
		g.requestFeedScrollBottom()
	}
	g.stepSmoothFeedScroll()
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
