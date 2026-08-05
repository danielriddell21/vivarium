# vivarium

> *An ecosystem where behaviour evolves.*

[![CI](https://github.com/danielriddell21/vivarium/actions/workflows/ci.yaml/badge.svg)](https://github.com/danielriddell21/vivarium/actions/workflows/ci.yaml)
[![codecov](https://codecov.io/gh/danielriddell21/vivarium/graph/badge.svg)](https://codecov.io/gh/danielriddell21/vivarium)
[![Quality Gate Status](https://sonarcloud.io/api/project_badges/measure?project=danielriddell21_vivarium&metric=alert_status)](https://sonarcloud.io/summary/new_code?id=danielriddell21_vivarium)
[![Go 1.26](https://img.shields.io/badge/go-1.26-blue)](https://go.dev)
[![MIT License](https://img.shields.io/badge/licence-MIT-green)](LICENSE)

A 2D ecosystem simulation in Go + [Ebiten](https://ebitengine.org) where agent **behaviour evolves** rather than being programmed. Each organism is steered by a tiny hand-rolled neural network; there is no backpropagation between generations — populations improve through reproduction with mutation, and adapt within a lifetime through reward-driven learning.

![overview](docs/demos/overview.gif)

## Commands

| Command | Description |
|---|---|
| `vivarium` | Run the GUI |
| `vivarium headless` | Run display-free and stream CSV statistics |

## Install

### Homebrew
```sh
brew install danielriddell21/tap/vivarium
brew install --cask danielriddell21/tap/vivarium
```

### Go install
```sh
go install github.com/danielriddell21/vivarium/cmd/vivarium@latest
```

### Quick start
```sh
go run ./cmd/vivarium              # run the GUI
go run ./cmd/vivarium --seed 42    # reproducible run
```

> On Linux you need the usual Ebiten build dependencies (OpenGL + X11 dev headers,
> e.g. `libgl1-mesa-dev xorg-dev libxxf86vm-dev libasound2-dev`).

All randomness comes from a single seeded generator, so a given `--seed` reproduces
the same run exactly.

## Documentation

Full documentation lives in the [vivarium wiki](https://github.com/danielriddell21/vivarium/wiki) — controls, CLI flags, the JSON config, headless experiments, and the feature gallery.
