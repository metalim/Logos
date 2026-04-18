//go:build !js

package main

// GLFW scroll units (~0.1 per smallest tick on macOS); 18 px/unit ≈ one 14pt line per full unit.
const feedWheelContentPixelsPerUnit = 18.0
