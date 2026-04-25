//go:build js

package game

// Ebitengine on js forwards DOM wheel deltaY as-is (deltaMode unhandled upstream); on macOS Safari/Chrome
// values are DOM_DELTA_PIXEL — mouse smooth-wheel ≈ 4 px/tick, trackpad ≈ 1 px/step. The 2.5 multiplier
// matches the layout-vs-window scale (window=450x800, layout=900x1600), so one DOM pixel of finger
// travel still translates to one device pixel of scroll.
const feedWheelContentPixelsPerUnit = 2.5
