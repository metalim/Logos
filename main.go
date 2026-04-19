package main

import (
	"bytes"
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
	// Window is the OS-level pixel size; layout is the logical drawing surface returned by
	// Game.Layout. layout = window * 2 gives a "retina" 2x framebuffer over a 1.25x-larger
	// window vs the original 360x640 design — all UI sizes below are scaled by 2.5x to
	// match (visualSize ≈ layout / 2 ≈ original * 1.25).
	windowWidth  = 450
	windowHeight = 800
	layoutWidth  = 900
	layoutHeight = 1600

	// Vertical band proportions (integer percent of outsideH); bottom band fills the remainder.
	// Top ≈ real phone status strip (iPhone ~5–6% of screen height).
	bandTopPercent = 6
	bandMidPercent = 60
	// Pixels of root background showing between bands (RowLayout spacing).
	bandSpacingPx = 3

	// News text horizontal inset (px) inside the feed band; clamped to a sane minimum.
	newsTextSideInset = 50
	newsTextMinWidth  = 100

	newsFontPt    = 35
	newsTextPadPx = 20

	// Initial player resource per CONCEPT (placeholder until the core loop is wired).
	startingPatchCount = 5

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
	clockText  *widget.Text

	mapLabelFace     text.Face
	overlayLabelFace text.Face
	overlayValueFace text.Face
	nodes            []Node
	edges            []Edge

	// Static catalog + adjacency (built once in initVisibleNetwork). Visible nodes
	// reference catalog entries via Node.DefIdx; revealNeighbors consults Network.Adj
	// to surface hidden neighbours when a node is captured.
	network      *Network
	visibleByDef map[int]int // catalog index → visible Node index (g.nodes)
	// Per-state slot lists. Order = current angular slot in the ring; relayoutTargets
	// translates slot index to TargetX/Y. Hub is in neither list (always at center).
	outerRing []int // visible indices for Normal / Attack / Patched (excl. hub)
	innerRing []int // visible indices for Infected (excl. hub)

	// Game-state overlay (drawn on top of the map's upper edge).
	infectionPct float64
	patchesLeft  int

	// Index of the Attack node whose patch menu is currently shown, or -1 for none.
	pendingPatchNode int

	// Attack scheduling and animation clock.
	epoch        time.Time
	lastAttackAt time.Time
	lastSimTick  time.Time

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

	// Drag-to-scroll state for the news feed (touch on mobile, left mouse on desktop).
	feedDragActive  bool
	feedDragMouse   bool
	feedDragTouchID ebiten.TouchID
	feedDragStartY  int
	feedDragStartPx float64

	// Endgame latch. nil until checkGameOver / triggerLoss / triggerWin fires; gates
	// every sim system and the attack-pulse animation. Holds the pre-picked voiceover
	// line + terminator so checkGameOver doesn't reroll the text mid-stream.
	end *endgame
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

// setBandHeight pins a band to an exact height in the root RowLayout: MinHeight handles
// empty children (statusBar/mapPanel), MaxHeight clamps oversized PreferredSize (feedScroll's
// ScrollContainer reports content size). Stretch keeps the band full-width.
func setBandHeight(w *widget.Widget, h int) {
	w.MinHeight = h
	w.LayoutData = widget.RowLayoutData{Stretch: true, MaxHeight: h}
}

// newBandContainer builds one of the three vertical bands: stretched to root width,
// height filled by MinHeight written in applyVerticalBands, with the given inner layout and bg.
func newBandContainer(bg color.NRGBA, inner widget.Layouter) *widget.Container {
	return widget.NewContainer(
		widget.ContainerOpts.Layout(inner),
		widget.ContainerOpts.BackgroundImage(eimage.NewNineSliceColor(bg)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
			widget.WidgetOpts.MinSize(0, 1),
		),
	)
}

func newGame() (*Game, error) {
	face, err := loadFont(newsFontPt)
	if err != nil {
		return nil, err
	}

	// Root stacks the three bands top-to-bottom; per-band height comes from MinHeight set in
	// applyVerticalBands, width is stretched via RowLayoutData on each child.
	root := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionVertical),
			widget.RowLayoutOpts.Spacing(bandSpacingPx),
		)),
		widget.ContainerOpts.BackgroundImage(
			eimage.NewNineSliceColor(color.NRGBA{R: 0x12, G: 0x12, B: 0x14, A: 0xff}),
		),
	)

	statusBar := newBandContainer(
		color.NRGBA{R: 0x0a, G: 0x0b, B: 0x0e, A: 0xff},
		widget.NewAnchorLayout(
			widget.AnchorLayoutOpts.Padding(&widget.Insets{Top: 5, Bottom: 5}),
		),
	)
	mapPanel := newBandContainer(
		color.NRGBA{R: 0x2c, G: 0x2e, B: 0x34, A: 0xff},
		widget.NewAnchorLayout(),
	)

	newsText := widget.NewText(
		widget.TextOpts.Text(strings.Repeat(sampleNews, 10), &face, color.NRGBA{R: 0xe8, G: 0xea, B: 0xf0, A: 0xff}),
		widget.TextOpts.MaxWidth(layoutWidth-newsTextSideInset),
		widget.TextOpts.Padding(widget.NewInsetsSimple(newsTextPadPx)),
	)

	feedScroll := widget.NewScrollContainer(
		widget.ScrollContainerOpts.Content(newsText),
		widget.ScrollContainerOpts.StretchContentWidth(),
		widget.ScrollContainerOpts.Image(&widget.ScrollContainerImage{
			Idle: eimage.NewNineSliceColor(color.NRGBA{R: 0x18, G: 0x1a, B: 0x20, A: 0xff}),
			Mask: eimage.NewNineSliceColor(color.NRGBA{R: 0x18, G: 0x1a, B: 0x20, A: 0xff}),
		}),
		widget.ScrollContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Stretch: true}),
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
	g.clockText = populatePhoneTitleBar(statusBar, face)
	if labelFace, err := loadFont(mapLabelFontPt); err == nil {
		g.mapLabelFace = labelFace
	}
	if f, err := loadFont(overlayLabelFontPt); err == nil {
		g.overlayLabelFace = f
	}
	if f, err := loadFont(overlayValueFontPt); err == nil {
		g.overlayValueFace = f
	}
	g.initVisibleNetwork()
	g.infectionPct = initialInfectionPct(g.nodes)
	g.patchesLeft = startingPatchCount
	g.pendingPatchNode = -1
	g.epoch = time.Now()
	g.lastAttackAt = g.epoch
	g.lastSimTick = g.epoch
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

	h1 := outsideH * bandTopPercent / 100
	h2 := outsideH * bandMidPercent / 100
	h3 := outsideH - h1 - h2 - 2*bandSpacingPx

	// MaxHeight is critical for feedScroll: ScrollContainer.PreferredSize() returns the
	// (huge) content size, which RowLayout would otherwise hand it as the laid-out height,
	// collapsing the scroll slack to zero. MinHeight covers empty bands; MaxHeight pins them.
	setBandHeight(g.statusBar.GetWidget(), h1)
	setBandHeight(g.mapPanel.GetWidget(), h2)
	setBandHeight(g.feedScroll.GetWidget(), h3)

	mw := float64(outsideW - newsTextSideInset)
	if mw < newsTextMinWidth {
		mw = newsTextMinWidth
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

// pushNews appends a single bullet line to the news feed and schedules an auto-scroll
// to the bottom on the next frame (deferred via feedScrollNeedBottom because triggering
// PreferredSize during Layout has crashed ebitenui in the past). Empty strings are
// silently ignored so callers don't have to guard.
func (g *Game) pushNews(line string) {
	if line == "" || g.newsText == nil {
		return
	}
	g.newsText.Label += "\n\n• " + line
	if g.root != nil {
		g.root.RequestRelayout()
	}
	g.feedScrollNeedBottom = true
}

func (g *Game) Update() error {
	g.checkGameOver()

	now := time.Now()
	dt := now.Sub(g.lastSimTick)
	g.lastSimTick = now

	if !g.gameEnded() {
		if time.Since(g.lastAttackAt) >= attackInterval {
			g.lastAttackAt = time.Now()
			g.attackTick()
		}
		g.progressAttacks()
		g.accumulateInfection(dt)
		g.accumulateProduction(dt)
	}
	// easeNodes always runs so any in-flight migrate-inward animation finishes cleanly
	// even after the loss latch — frozen sim, but no jarring half-moved nodes.
	g.easeNodes(dt)

	updateClock(g.clockText)

	g.ui.Update()
	if g.feedScrollNeedBottom {
		g.feedScrollNeedBottom = false
		g.requestFeedScrollBottom()
	}
	g.handleDebugMenu()
	g.handlePatchClick()
	g.handleFeedDrag()
	g.stepSmoothFeedScroll()
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	g.ui.Draw(screen)
	g.drawNodeMap(screen)
	g.drawGameOverlay(screen)
	g.drawPatchMenu(screen)
	g.drawDebugMenu(screen)
}

func (g *Game) Layout(_, _ int) (int, int) {
	g.applyVerticalBands(layoutWidth, layoutHeight)
	return layoutWidth, layoutHeight
}

func main() {
	ebiten.SetWindowTitle("Zero-Day Lunch")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeDisabled)
	ebiten.SetWindowSize(windowWidth, windowHeight)

	g, err := newGame()
	if err != nil {
		log.Fatal(err)
	}

	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
