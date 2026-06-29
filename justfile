# list available recipes
default:
    @just --list

# build everything (headless; the Ebiten GUI is gated behind the `ebiten` tag)
[group('build')]
build:
    go build ./...

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

# run the tests
[group('test')]
test:
    go test ./...

# run the linter
[group('dev')]
lint:
    golangci-lint run

# vet the code
[group('dev')]
vet:
    go vet ./...

# format the code
[group('dev')]
fmt:
    gofmt -w .

# tidy module dependencies
[group('dev')]
tidy:
    go mod tidy
