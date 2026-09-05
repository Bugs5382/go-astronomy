# AGENTS.md - go-astronomy

Technical reference for agents and developers *consuming* `go-astronomy`. It is
example-first: read the code, then the rules. For the repository working
agreement and the hook-enforced commit/branch/PR rules, see `CLAUDE.md`.

Some packages below are still under construction; signatures marked `(roadmap)`
are indicative and may change before v1.0.0. Keep this file current when the
build, layout, or public API changes.

## What this is

A stateless, MIT-licensed Go library that computes observer-aware positions of
the Sun, Moon, stars, and constellations for a given
`(latitude, longitude, timezone, time)`. It emits geometry as **degrees**
(altitude, azimuth) plus time-progress; it never emits pixels or colors. The
math is built on `github.com/soniakeys/meeus/v3`.

The library owns universal geometry only. Meaning (phase names, twilight labels)
and appearance (colors, layout) are the consumer's concern.

## Import-path / package map

Module path: `github.com/Bugs5382/go-astronomy`

| Import path | Kind | What it provides |
|---|---|---|
| `github.com/Bugs5382/go-astronomy` | universal | Package doc and shared types (`Observer`, coordinate types). |
| `github.com/Bugs5382/go-astronomy/sun` | universal | `sun.Position(obs, t)`, `sun.Track(obs, date, samples)` — geometric alt/az of the disc center; no Earth labels. |
| `github.com/Bugs5382/go-astronomy/star` | universal (roadmap) | Embedded HYG catalog (RA/Dec + distance); projects to alt/az for the observer/instant. |
| `github.com/Bugs5382/go-astronomy/constellation` | universal (roadmap) | Boundary lookup by RA/Dec; IAU dataset is the overridable default. |
| `github.com/Bugs5382/go-astronomy/earth` | Earth traits (roadmap) | `earth.NewSunTimes(obs, date)`, twilight bands, `earth.DefaultSegmentation`, `earth.Refraction`, polar states. |
| `github.com/Bugs5382/go-astronomy/earth/moon` | Earth traits (roadmap) | Luna: `Position`/`ApparentPosition`, `NextRise`/`NextSet`, phases (`Age`, `Illumination`, `PhaseAngle`, next new/full), `Track`. |
| `github.com/Bugs5382/go-astronomy/internal/...` | internal | Math core (`angles`, `julian`, `coordinates`, `project`). Unexported by policy — do not import. |

Only the Sun is universal to the solar system. Moons are body-specific (Luna
lives under `earth`). Import the universal packages plus the one body package you
need. A future `mars/` would not change `sun`.

## Example: Sun position and today's Sun times

```go
package main

import (
	"fmt"
	"time"

	"github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/sun"
)

func main() {
	tz, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}
	obs := astronomy.Observer{Lat: 40.678, Lon: -73.944, TZ: tz}

	// Pass the instant on every call; nothing is captured at construction.
	t := time.Date(1982, 5, 3, 12, 0, 0, 0, tz)

	pos := sun.Position(obs, t) // degrees, disc center
	fmt.Printf("alt=%.2f az=%.2f diameter=%.4f\n", pos.Altitude, pos.Azimuth, pos.AngularDiameter)

	day, err := earth.NewSunTimes(obs, t) // one civil day, resolved in obs.TZ
	if err != nil {
		panic(err)
	}
	if noon, ok := day.SolarNoon(); ok {
		fmt.Println("solar noon:", noon)
	}
	if state, isPolar := day.Polar(); isPolar {
		fmt.Println("polar state:", state) // MidnightSun or PolarNight
	}
}
```

## Hard rules for consumers

1. **Stateless — never cache the library across instants.** Every call takes the
   observer and `time.Time`. Do not build a value "at construction" and reuse it
   for a later moment; a stale snapshot is a bug, not an optimization.
2. **Pass `time.Time` on every call.** The instant is always a parameter. The
   library computes internally in UTC and applies the caller's timezone at the
   boundary; hand it a zone-aware `time.Time` (via `time.LoadLocation`) and DST
   is handled correctly (23h/25h days). Never assume a fixed UTC offset.
3. **Degrees, not pixels.** Positions come back as altitude/azimuth in degrees
   plus a time-progress fraction. Screen mapping is the projection layer's job,
   not something to hardcode against. Do not treat any returned number as a pixel
   coordinate.
4. **Positions are disc centers plus angular diameter.** Size and place the disc
   yourself; compute alignment/overlap (eclipses, occultations) from center +
   diameter. There is no single "point" for the Sun or Moon.
5. **Colors are the consumer's concern.** The library emits no palette and no
   color. Twilight band names and thresholds are a *segmentation* you can
   override (Earth defaults ship); the altitude-to-label mapping is data, not a
   hardcoded convention.
6. **Absence is explicit.** Expect `(value, bool)` / `(value, error)` where a
   result may not exist (no sunrise on a polar day). There are no `-1` / `false`
   sentinels; polar conditions are an explicit state, never a nil panic.
7. **Do not import `internal/`.** It is unexported by policy and may change
   without notice. Consume only the packages in the map above.
8. **Accuracy is arcminute-class.** Nutation, ΔT, and leap seconds are
   deliberately omitted (below the precision floor). Do not rely on the library
   for higher-precision ephemeris work.

## Build, test, lint (contributors)

Commands come from the `Taskfile.yaml`:

- Build: `task build` (`go build ./...`)
- Test: `task test` (`go test ./...`); with coverage: `task test-cover`
- Format: `task fmt` (`gofmt` + `goimports`)
- Lint: `task lint` (`gofmt` check, `golangci-lint`, `yamllint`)
- License headers: `task license` (verify, dry run) / `task license:fix` (inject)

No emoji or AI tells in Go source or commit messages; emoji are allowed in
Markdown docs and workflow files only. The branch/commit/PR rules in `CLAUDE.md`
are enforced by the git hooks in `.claude/hooks` (run
`bash .claude/hooks/install.sh` once per clone).

## Deeper docs

- `README.md` — overview, quick start, feature list, and the architecture
  diagram.
- API reference: [pkg.go.dev/github.com/Bugs5382/go-astronomy](https://pkg.go.dev/github.com/Bugs5382/go-astronomy).
- `CLAUDE.md` — the repository working agreement and governance.
