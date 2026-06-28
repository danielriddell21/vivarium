# list available recipes
default:
    @just --list

# build everything (headless; the Ebiten GUI is gated behind the `ebiten` tag)
build:
    go build ./...

# build the GUI binary (Ebiten window; on Linux this needs the OpenGL/X11 libs)
build-gui:
    go build -tags ebiten -o bin/vivarium ./cmd/vivarium

# run the GUI (Ebiten)
gui *ARGS:
    go run -tags ebiten ./cmd/vivarium {{ARGS}}

# run the headless batch simulator
run *ARGS:
    go run ./cmd/vivarium-headless {{ARGS}}

# run the tests
test:
    go test ./...

# run the linter
lint:
    golangci-lint run

# vet the code
vet:
    go vet ./...

# format the code
fmt:
    gofmt -w .

# tidy module dependencies
tidy:
    go mod tidy
