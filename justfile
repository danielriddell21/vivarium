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
    # One headless program renders every clip: no window, no display, no
    # ebiten build tag. Give a clip an .mp4 extension to record video instead.
    go run ./tools/demogen
