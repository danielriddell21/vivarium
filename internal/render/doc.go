// Package render wires the simulation to Ebiten: it draws the world, overlays
// (population graph, HUD, inspector), and handles keyboard/mouse input. It is the
// only package besides cmd that depends on Ebiten, keeping the simulation core
// free of any graphics concerns.
//
// The Ebiten-backed implementation is compiled only under the "ebiten" build
// tag. Without that tag this package is empty, so the simulation core and the
// headless command build and lint with no graphics or cgo dependencies.
package render
