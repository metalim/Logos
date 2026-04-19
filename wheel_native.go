//go:build !js

package main

// GLFW scroll units (~0.1 per smallest tick on macOS); 45 layout px/unit ≈ one newsFontPt
// line per full unit at the 2.5x layout scale.
const feedWheelContentPixelsPerUnit = 45.0
