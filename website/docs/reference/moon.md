---
id: moon
title: earth/moon
sidebar_position: 3
---

# 🌙 Package `earth/moon`

Import path: `github.com/Bugs5382/go-astronomy/earth/moon`

The Earth vantage on Luna, the Earth's moon: its topocentric horizontal position and apparent size, its phase and illumination, its rise and set for an observer, and an arc track over a time span. The Moon belongs to Earth in this library's architecture, so the package lives under `earth`; other bodies would own their own moons.

Positions are the center of the disc, in degrees, paired with the apparent angular diameter. Altitudes from `Position` are geometric (no refraction) and topocentric (corrected for lunar parallax); `ApparentPosition` adds atmospheric refraction. The package is stateless and concurrency-safe. Accuracy is arcminute-class; nutation, ΔT, and Earth flattening are below that floor and are not modeled.

## 📍 Position

```go
func Position(obs astronomy.Observer, t time.Time) (astronomy.Position, error)
func ApparentPosition(obs astronomy.Observer, t time.Time) (astronomy.Position, error)
```

`Position` returns the geometric topocentric alt/az of the Moon's disc center, paired with its apparent diameter. `ApparentPosition` adds atmospheric refraction so the altitude matches what an observer sees. Both validate the observer and return a coded error for an out-of-range latitude or longitude.

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
