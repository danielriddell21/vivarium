# Vivarium

A small 2D ecosystem simulation in Go + [Ebiten](https://ebitengine.org) where
agent **behaviour evolves** rather than being programmed. Each organism is steered
by a tiny hand-rolled neural network; there is no backpropagation — populations
improve purely through reproduction with mutation.

![demo](docs/demo.gif)

## The model

A toroidal (wrap-around) world with three trophic tiers:

| Tier | Drawn as | Eats | Eaten by |
| --- | --- | --- | --- |
| Plants (food) | green dots | — | herbivores |
| Herbivores | blue circles | plants | carnivores |
| Carnivores | red triangles | herbivores | — |

Every agent has an energy budget. Energy drains each tick (a basal cost) and with
movement; eating restores it. At zero energy the agent dies. Plants regrow energy
over time toward a cap.

## The brain

Each agent's behaviour comes from a recurrent neural net (7 inputs → 8 hidden →
3 outputs, `tanh` activations), implemented from scratch with flat `float64`
slices — no ML libraries. The hidden layer feeds its previous activations back in
(an Elman-style memory), so an agent can act on the recent past — keep fleeing for
a moment after a predator drops out of range, wander, etc. — rather than reacting
to the current senses alone. Offspring inherit the weights but start with a blank
memory.

**Inputs:** normalised energy; the relative bearing (cos/sin) and proximity of the
nearest *target* (food for herbivores, prey for carnivores); and the bearing +
proximity of the nearest *threat* (a predator). The proximity values act as the
raycast-style distance sensors.

**Outputs:** turn, speed, and an eat/act decision.

## Evolution

When an agent's energy crosses a threshold it reproduces, splitting its energy with
an offspring. The child inherits a **clone of the parent's brain with small
Gaussian weight mutations**, and its morphological traits — **size, max speed, and
sense radius** — co-evolve via the same mutate-on-inherit rule. Over generations
the population drifts toward viable strategies: foraging, seeking, and fleeing.

### Population stability

Naïve predator–prey agent worlds tend to collapse: carnivores overshoot, eat every
herbivore, then starve all at once. Three mechanisms damp this into coexistence:

- **Predator metabolism** — carnivores burn energy ~3x faster than herbivores, so
  when prey is scarce they decline quickly instead of lingering.
- **Gestation cooldown** — a kill can't be turned into an instant litter; predators
  also reproduce more slowly than prey (as in real food webs), and an energy cap
  stops a big meal from being hoarded into many births.
- **Rescue effect** (`-rescue`, on by default) — if a tier nears extinction, rare
  immigrants arrive (descended from survivors when any remain, so evolution
  continues), modelling a metapopulation rescue. Pass `-rescue=false` for the raw,
  collapse-prone dynamics.

## Controls

| Key / action | Effect |
| --- | --- |
| `space` | pause / resume |
| `+` or `=` | double simulation speed |
| `-` | halve simulation speed |
| left click | select the nearest agent and open its inspector |

The HUD shows run state and live counts, a line chart tracks plant/herbivore/
carnivore counts over time, and the inspector shows a selected agent's energy, age,
generation, traits, and current neural inputs/outputs.

## Running

```sh
go run ./cmd/vivarium            # default run
go run ./cmd/vivarium -seed 42   # reproducible run
```

All randomness is drawn from a single seeded generator, so a given `-seed`
reproduces the same run exactly.

### Flags

| Flag | Default | Description |
| --- | --- | --- |
| `-seed` | 1 | random seed |
| `-width`, `-height` | 960, 720 | world size in pixels |
| `-plants` | 200 | initial plant count |
| `-herbivores` | 80 | initial herbivore count |
| `-carnivores` | 8 | initial carnivore count |
| `-rescue` | true | immigration when a tier nears extinction (set `false` for raw dynamics) |

> On Linux you need the usual Ebiten build dependencies (OpenGL + X11 dev headers,
> e.g. `libgl1-mesa-dev xorg-dev libxxf86vm-dev libasound2-dev`).

## Layout

```
cmd/vivarium      entry point: flags, seeding, window setup
internal/geom     2D vectors + toroidal math
internal/neural   hand-rolled feedforward brain (+ tests)
internal/sim      World, Agent, Food, Traits, evolution (+ tests)
internal/render   Ebiten game loop, drawing, overlays, input
```

The `geom`, `neural`, and `sim` packages are pure Go with no Ebiten dependency, so
the simulation core is unit-testable headlessly: `go test ./internal/...`.

## Related work

- [Polyworld](https://github.com/polyworld/polyworld)
- [The Bibites](https://www.thebibites.com/)
- [Karl Sims' Evolved Virtual Creatures](https://en.wikipedia.org/wiki/Karl_Sims)
- [Framsticks](https://www.framsticks.com/)
