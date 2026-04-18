//go:build js

package main

// Ebitengine on js forwards DOM wheel deltaY as-is (deltaMode unhandled upstream); on macOS Safari/Chrome
// values are DOM_DELTA_PIXEL — mouse smooth-wheel ≈ 4 px/tick, trackpad ≈ 1 px/step. 1:1 mapping keeps wheel
// and trackpad as fine-grained as native GLFW (~1.8 px minimum step).
const feedWheelContentPixelsPerUnit = 1.0
