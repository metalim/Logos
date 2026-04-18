package main

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// handleFeedDrag implements 1:1 finger/mouse drag-to-scroll over the news feed.
// On press inside feedScroll.Rect we capture (startY, feedScrollPx); on move we write
// feedScrollPx directly (bypassing easing for zero lag) and sync feedScrollTarget so
// stepSmoothFeedScroll keeps the same value after release (diff = 0).
func (g *Game) handleFeedDrag() {
	if g.feedScroll == nil {
		return
	}
	extra, _, slackOK := g.feedScrollSlack()

	if g.feedDragActive {
		currentY, stillDown := g.activeDragPointer()
		if !stillDown {
			g.feedDragActive = false
			return
		}
		if !slackOK || extra <= 0 {
			return
		}
		newPx := g.feedDragStartPx - float64(currentY-g.feedDragStartY)
		if newPx < 0 {
			newPx = 0
		}
		if newPx > extra {
			newPx = extra
		}
		g.feedScrollPx = newPx
		g.feedScrollTarget = newPx / extra
		return
	}

	rect := g.feedScroll.GetWidget().Rect

	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		if image.Pt(x, y).In(rect) {
			g.beginFeedDrag(false, id, y, extra, slackOK)
			return
		}
	}

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if image.Pt(x, y).In(rect) {
			g.beginFeedDrag(true, 0, y, extra, slackOK)
		}
	}
}

func (g *Game) beginFeedDrag(mouse bool, touchID ebiten.TouchID, y int, extra float64, slackOK bool) {
	g.feedDragActive = true
	g.feedDragMouse = mouse
	g.feedDragTouchID = touchID
	g.feedDragStartY = y
	switch {
	case g.feedScrollPx >= 0:
		g.feedDragStartPx = g.feedScrollPx
	case slackOK && extra > 0:
		g.feedDragStartPx = g.feedScroll.ScrollTop * extra
	default:
		g.feedDragStartPx = 0
	}
}

func (g *Game) activeDragPointer() (y int, down bool) {
	if g.feedDragMouse {
		if !ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			return 0, false
		}
		_, y = ebiten.CursorPosition()
		return y, true
	}
	for _, id := range ebiten.AppendTouchIDs(nil) {
		if id == g.feedDragTouchID {
			_, y = ebiten.TouchPosition(id)
			return y, true
		}
	}
	return 0, false
}
