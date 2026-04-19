package main

import (
	"image/color"
	"time"

	"github.com/ebitenui/ebitenui/widget"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const (
	// Horizontal inset emulating the phone's rounded-corner safe area.
	titleBarPadX     = 50
	titleBarSpacingX = 15
	titleBarStrokeW  = 2 // outline thickness for battery frame

	signalBarCount = 4
	signalBarW     = 8
	signalBarGap   = 5
	signalBaseH    = 10
	signalStepH    = 8

	batteryBodyW       = 55
	batteryBodyH       = 25
	batteryTipW        = 5
	batteryTipH        = 10
	batterySegments    = 4
	batterySegInset    = 5
	batterySegInterval = 3

	clockTimeFormat = "15:04"

	networkLabel = "5G"
)

var (
	titleBarFG    = color.NRGBA{R: 0xe8, G: 0xea, B: 0xf0, A: 0xff}
	titleBarDimFG = color.NRGBA{R: 0x55, G: 0x55, B: 0x5a, A: 0xff}
)

// populatePhoneTitleBar fills the empty status-bar container with a phone-style header:
// 24h clock on the left, signal-strength bars and a battery glyph on the right.
// Returns the clock Text widget so the game loop can refresh its label every frame.
func populatePhoneTitleBar(bar *widget.Container, face text.Face) *widget.Text {
	clock := widget.NewText(
		widget.TextOpts.Text(time.Now().Format(clockTimeFormat), &face, titleBarFG),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionStart,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
				Padding:            &widget.Insets{Left: titleBarPadX},
			}),
		),
	)

	right := widget.NewContainer(
		widget.ContainerOpts.Layout(widget.NewRowLayout(
			widget.RowLayoutOpts.Direction(widget.DirectionHorizontal),
			widget.RowLayoutOpts.Spacing(titleBarSpacingX),
			widget.RowLayoutOpts.Padding(&widget.Insets{Right: titleBarPadX}),
		)),
		widget.ContainerOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.AnchorLayoutData{
				HorizontalPosition: widget.AnchorLayoutPositionEnd,
				VerticalPosition:   widget.AnchorLayoutPositionCenter,
			}),
		),
	)

	right.AddChild(widget.NewGraphic(
		widget.GraphicOpts.Image(makeSignalIcon(signalBarCount, signalBarCount)),
		widget.GraphicOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Position: widget.RowLayoutPositionCenter}),
		),
	))
	right.AddChild(widget.NewText(
		widget.TextOpts.Text(networkLabel, &face, titleBarFG),
		widget.TextOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Position: widget.RowLayoutPositionCenter}),
		),
	))
	right.AddChild(widget.NewGraphic(
		widget.GraphicOpts.Image(makeBatteryIcon(batterySegments)),
		widget.GraphicOpts.WidgetOpts(
			widget.WidgetOpts.LayoutData(widget.RowLayoutData{Position: widget.RowLayoutPositionCenter}),
		),
	))

	bar.AddChild(clock)
	bar.AddChild(right)
	return clock
}

func makeSignalIcon(filled, total int) *ebiten.Image {
	w := total*signalBarW + (total-1)*signalBarGap
	h := signalBaseH + signalStepH*(total-1)
	img := ebiten.NewImage(w, h)
	for i := 0; i < total; i++ {
		bh := signalBaseH + signalStepH*i
		x := i * (signalBarW + signalBarGap)
		y := h - bh
		col := color.Color(titleBarDimFG)
		if i < filled {
			col = titleBarFG
		}
		vector.FillRect(img, float32(x), float32(y), float32(signalBarW), float32(bh), col, false)
	}
	return img
}

func makeBatteryIcon(filledSegments int) *ebiten.Image {
	w := batteryBodyW + batteryTipW
	img := ebiten.NewImage(w, batteryBodyH)

	const sw = titleBarStrokeW
	vector.StrokeRect(img, sw/2.0, sw/2.0, float32(batteryBodyW)-sw, float32(batteryBodyH)-sw, sw, titleBarFG, false)
	vector.FillRect(img,
		float32(batteryBodyW),
		float32(batteryBodyH-batteryTipH)/2,
		float32(batteryTipW),
		float32(batteryTipH),
		titleBarFG, false)

	innerW := batteryBodyW - 2*batterySegInset
	innerH := batteryBodyH - 2*batterySegInset
	totalGaps := (batterySegments - 1) * batterySegInterval
	segW := (innerW - totalGaps) / batterySegments
	for i := 0; i < filledSegments; i++ {
		x := batterySegInset + i*(segW+batterySegInterval)
		vector.FillRect(img,
			float32(x), float32(batterySegInset),
			float32(segW), float32(innerH),
			titleBarFG, false)
	}
	return img
}

// updateClock refreshes the title-bar clock label; cheap to call every frame.
func updateClock(clock *widget.Text) {
	if clock == nil {
		return
	}
	now := time.Now().Format(clockTimeFormat)
	if clock.Label != now {
		clock.Label = now
	}
}
