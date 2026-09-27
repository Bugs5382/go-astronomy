---
id: moon
title: earth/moon
sidebar_position: 3
---

# 🌙 Package `earth/moon`

Import path: `github.com/Bugs5382/go-astronomy/earth/moon`

The Earth vantage on Luna, the Earth's moon: its topocentric horizontal position and apparent size, its phase and illumination, its rise and set for an observer, and an arc track over a time span. The Moon belongs to Earth in this library's architecture, so the package lives under `earth`; other bodies would own their own moons.

Positions are the center of the disc, in degrees, paired with the apparent angular diameter. Altitudes from `Position` are geometric (no refraction) and topocentric (moved to the observer on the flattened WGS84 Earth by the rigorous parallax correction); `ApparentPosition` adds atmospheric refraction. The diameter is topocentric too: it is sized from the observer's distance to the Moon, not the distance from the Earth's center. The package is stateless and concurrency-safe. Accuracy is arcminute-class; nutation and ΔT are below that floor and are not modeled.

## 📍 Position

```go
func Position(obs astronomy.Observer, t time.Time) (astronomy.Position, error)
func ApparentPosition(obs astronomy.Observer, t time.Time) (astronomy.Position, error)
```

`Position` returns the geometric topocentric alt/az of the Moon's disc center, paired with its topocentric apparent diameter (the observer's distance, from the IAU mean lunar radius of 1737.4 km). `ApparentPosition` adds atmospheric refraction so the altitude matches what an observer sees. Both validate the observer and return a coded error for an out-of-range latitude or longitude.

## 🌗 Phases and illumination

```go
const SynodicMonth = 29.530588853 // mean New-to-New interval, days

type Phase int
const (
	New Phase = iota
	WaxingCrescent
	FirstQuarter
	WaxingGibbous
	Full
	WaningGibbous
	LastQuarter
	WaningCrescent
)
func (p Phase) String() string

func Age(t time.Time) float64          // days since the previous New Moon
func PhaseAngle(t time.Time) float64   // Sun-Moon phase angle, degrees
func Illumination(t time.Time) float64 // illuminated fraction, [0, 1]
func PhaseAt(t time.Time) Phase        // the named phase at t
func NextNew(t time.Time) time.Time    // next New Moon after t
func NextFull(t time.Time) time.Time   // next Full Moon after t
```

`Phase` is one of the eight conventional named phases; `String` returns a lowercase snake_case name such as `waxing_crescent`. `Illumination` drives the illuminated-limb shape a consumer draws. These phase functions depend only on time, not on the observer.

`PhaseAt` names the phase from the Moon's **elongation** — its difference from the Sun in apparent ecliptic longitude — in eight 45° sectors, each centred on its named point, so New, First Quarter, Full and Last Quarter each name the sector straddling their exact instant. Elongation runs the whole way round the cycle, so it carries the waxing or waning sense as well as the shape.

It is deliberately not derived from `Age`. Age over `SynodicMonth` is a clock, and `SynodicMonth` is a mean: individual cycles run several hours either side of it, and the Moon's speed varies within a cycle, so equal stretches of time do not fall on equal stretches of the geometry. Measured hourly over four years, naming the phase by age disagrees with the sky **10.2% of the time**, in runs of up to **22 hours**. `Age` is still there for callers who want the age itself.

:::note Phase angle precision
`PhaseAngle` uses the accurate method of Meeus chapter 48, formulae (48.2) and (48.3), which combines the Moon's geocentric position with the Sun's apparent longitude and distance. Measured against JPL Horizons across a synodic month it agrees to better than 0.02°, and `Illumination` to better than 0.0002, including within an hour of New Moon.

The angle reaches neither end of its range. At New Moon and at Full Moon the Moon's apparent longitude is aligned with the Sun's, so the elongation left over is the Moon's ecliptic latitude — up to about 5.3° — and the phase angle stops that far short of 180° and of 0°. Read `Illumination` for "how full is the disc"; it does reach 0 and 1.
:::

## 🌒 Rise and set

```go
func NextRise(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
func NextSet(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
```

`NextRise` and `NextSet` return the first instant strictly after `t` at which the Moon's upper limb crosses the horizon, and a boolean that is false when no crossing occurs within the search window (which can happen at high latitudes, where the Moon can stay above or below the horizon for many days). The search spans a full synodic month. Both return a coded error for an out-of-range observer.

## 📈 Track

```go
type Sample struct {
	Time         time.Time
	Altitude     float64
	Azimuth      float64
	TimeProgress float64
	Illumination float64
	Phase        Phase
}

func Track(obs astronomy.Observer, from, to time.Time, samples int) ([]Sample, error)
```

`Track` samples the Moon's arc from `from` to `to`, inclusive of both endpoints, with exactly `samples` points evenly spaced in time. Each sample carries the geometric topocentric altitude and azimuth plus the illuminated fraction and named phase. Fewer than two samples returns `nil`; an out-of-range observer returns a coded error.

Full godoc: [pkg.go.dev/github.com/Bugs5382/go-astronomy/earth/moon](https://pkg.go.dev/github.com/Bugs5382/go-astronomy/earth/moon).
