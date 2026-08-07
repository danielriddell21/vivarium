# list available recipes
default:
    @just --list

# build everything (headless; the Ebiten GUI is gated behind the `ebiten` tag)
[group('build')]
build:
    go build ./...

# run the tests
[group('test')]
test:
    go test ./...

# run the linter
[group('dev')]
lint:
    golangci-lint run

# format the code
[group('dev')]
fmt:
    golangci-lint fmt

# tidy module dependencies
[group('dev')]
tidy:
    go mod tidy

# full gate: lint + test + build. all must pass before committing
[group('dev')]
ci: lint test build

# build the GUI binary (Ebiten window; on Linux this needs the OpenGL/X11 libs)
[group('build')]
build-gui:
    go build -tags ebiten -o bin/vivarium ./cmd/vivarium

# run the GUI (Ebiten)
[group('run')]
gui *ARGS:
    go run -tags ebiten ./cmd/vivarium {{ARGS}}

# run the headless batch simulator
[group('run')]
run *ARGS:
    go run ./cmd/vivarium headless {{ARGS}}

# regenerate the documentation media
[group('dev')]
demos:
    # Rendered headlessly through the software canvas: no window, no display,
    # no ebiten build tag. A .mp4 path records video instead of a GIF.
    mkdir -p docs/demos
    go run ./cmd/vivarium --record docs/demos/overview.gif --record-frames 200 --seed 5
