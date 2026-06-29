// Package gui wires the simulation to Ebiten: it draws the world, overlays
// (population graph, HUD, inspector), and handles keyboard/mouse input, behind a
// uniform Run/Available seam. It is the only package that depends on Ebiten,
// keeping the simulation core free of any graphics concerns.
//
// The Ebiten-backed implementation is compiled only under the "ebiten" build
// tag. Without that tag Run is a stub that reports the GUI is unavailable, so
// the simulation core and the headless command build and lint with no graphics
// or cgo dependencies.
package gui
