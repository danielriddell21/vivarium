// letsgo.mod

// The four targets the GoReleaser config built the CLI for. Deliberately no
// darwin: on macOS this ships as the cask below, which is the same program
// with the window compiled in and `vivarium headless` still available.
build (
	linux/amd64
	linux/arm64
	windows/amd64
	windows/arm64
)

// The same command built a second time with the Ebiten window compiled in.

// darwin only, as the GoReleaser config published: the GUI build used to need
// cgo for Metal and Cocoa, so it could only be built on a macOS runner. That
// is no longer true — Ebiten 2.10 goes through purego and cross-compiles with
// CGO_ENABLED=0 — but widening the list is a decision about what this game
// supports rather than part of moving release tools, so it is left alone.

// The GoReleaser config gave both archives the same name template and got
// away with it because the two target sets do not overlap. letsgo suffixes
// the variant's archives instead, which says which is which without relying
// on that. The binary inside both is still `vivarium`.
variant gui (
	build darwin/amd64 darwin/arm64
	tags ebiten
)

// The CLI build's formula. The GUI build's cask is written after the release
// by letsgo-cask, which letsgo does not run and cannot be changed by.
brew danielriddell21/tap

// What the formula says after installing, which is the one part of it nothing
// else can supply. Needs letsgo v0.8.0 or later.
brew caveats "The native GUI is macOS-only: on macOS install the cask with 'brew install --cask vivarium'. Elsewhere use 'vivarium headless' for batch runs."

// The shared GoReleaser workflow marked releases as pre-releases after
// publishing; letsgo does it while publishing, so promote.yaml still fires on
// manual promotion.
release prerelease=true
