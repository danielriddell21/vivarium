# Contributing to vivarium

## Requirements

* [Go](https://go.dev) (stable — version from `go.mod`)
* [just](https://github.com/casey/just)
* [golangci-lint](https://golangci-lint.run/welcome/install/) (for `just lint`)
* [gremlins](https://github.com/go-gremlins/gremlins) (for mutation testing)

A C toolchain is only needed for the optional Ebiten GUI (`just build-gui`).

## Development workflow

```
just build      # build everything (headless; GUI is gated behind the `ebiten` tag)
just build-gui  # build the GUI binary (-tags ebiten, needs cgo)
just gui        # run the Ebiten GUI
just run        # run the headless batch simulator (vivarium headless)
just test       # run unit tests
just lint       # golangci-lint
just vet        # go vet
just fmt        # gofmt
just tidy       # go mod tidy
```

Run `just --list` to see every recipe. Run `just lint` and `just test` before each commit. CI runs lint + test + build (headless) on every push to `trunk` and every pull request targeting `trunk`.

## Project layout

The Ebiten window is compiled only under the `ebiten` build tag, so the default
build and CI need no graphics or cgo dependencies.

```
cmd/vivarium/            CLI entry point (GUI by default; `headless` subcommand)
internal/cli/            Cobra root, the headless subcommand, completion
internal/gui/            Ebiten window + Run/Available seam (build-tagged)
internal/sim/            simulation core
internal/                neural, analytics, geom, …
docs/                    documentation
```

## Commit style

```
type(scope): short imperative description
```

Types: `feat`, `fix`, `docs`, `refactor`, `test`, `chore`. No period at the end of the subject line; keep it under 72 characters.

## Releases

Releases are triggered by pushing a semver tag — maintainers only. A GitHub Actions workflow runs GoReleaser to build the headless CLI formula and the macOS Ebiten GUI cask and update the Homebrew tap; it requires the tap app credentials configured as repository secrets.
