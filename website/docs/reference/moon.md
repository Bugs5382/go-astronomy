---
id: moon
title: earth/moon
sidebar_position: 3
---

# 🌙 Package `earth/moon`

Import path: `github.com/Bugs5382/go-astronomy/earth/moon`

The Earth vantage on Luna, the Earth's moon: its topocentric horizontal position and apparent size, its phase and illumination, its rise and set for an observer, and an arc track over a time span. The Moon belongs to Earth in this library's architecture, so the package lives under `earth`; other bodies would own their own moons.

Positions are the center of the disc, in degrees, paired with the apparent angular diameter. Altitudes from `Position` are geometric (no refraction) and topocentric (moved to the observer on the flattened WGS84 Earth by the rigorous parallax correction); `ApparentPosition` adds atmospheric refraction. The diameter is topocentric too: it is sized from the observer's distance to the Moon, not the distance from the Earth's center. The package is stateless and concurrency-safe. Positions are computed on Terrestrial Time with nutation and agree with JPL Horizons DE441 to about 10″.

## 📍 Position

```go
func Position(obs astronomy.Observer, t time.Time) (astronomy.Position, error)
func ApparentPosition(obs astronomy.Observer, t time.Time) (astronomy.Position, error)
```

`Position` returns the geometric topocentric alt/az of the Moon's disc center, paired with its topocentric apparent diameter (the observer's distance, from the IAU mean lunar radius of 1737.4 km). `ApparentPosition` adds atmospheric refraction so the altitude matches what an observer sees, scaled by the standard atmosphere at the observer's height (`earth.StandardAtmosphere`; unchanged at sea level). For measured air, add `earth.MeasuredAtmosphere(p, t).Refraction(alt, obs.Height)` to `Position`'s altitude. Both validate the observer and return a coded error for an out-of-range latitude or longitude.

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

`NextRise` and `NextSet` return the first instant strictly after `t` at which the Moon's upper limb crosses the horizon, and a boolean that is false when no crossing occurs within the search window (which can happen at high latitudes, where the Moon can stay above or below the horizon for many days). The search spans a full synodic month. For an observer above sea level the horizon is lowered by the dip and the 34′ of horizon refraction is scaled by the thinner air (see [Refraction and height](./earth.md#-refraction-and-height)), so the Moon rises earlier and sets later, by the dip's worth less the refraction lost. Both return a coded error for an out-of-range observer, including `astronomy.ErrInvalidHeight` for a NaN or infinite height.

## 🌓 Bright limb

```go
type BrightLimb struct {
	PositionAngle float64 // from celestial north through east, [0, 360)
	Parallactic   float64 // parallactic angle at the Moon, (-180, 180]
	ZenithAngle   float64 // PositionAngle - Parallactic: from "up" on the observer's sky, [0, 360)
}

func BrightLimbAt(obs astronomy.Observer, t time.Time) (BrightLimb, error)
```

`BrightLimbAt` says which way the lit side faces. It is the direction from the Moon toward the Sun on the observer's sky (Meeus 48.5), from both bodies' topocentric apparent places, and it points at the Sun even when the Sun is below the horizon.

All three angles turn counter-clockwise as the observer sees the sky, so 90 is to the left of the reference direction and 270 to the right:

- `PositionAngle` is measured from the direction of the north celestial pole, through east. It is a property of the sky, referred to the true equator and equinox of date.
- `Parallactic` is the angle from celestial north to the direction of the zenith at the Moon (Meeus 14.1). It is negative while the Moon is east of the meridian.
- `ZenithAngle` folds the parallactic angle in. It is measured from "up" on the observer's sky, which is what a screen needs: with the top of the screen as up, 0 means the lit side faces the top, 90 the left, 180 the bottom, and 270 the right.

### Example

```go
obs := astronomy.Observer{Lat: 39.74, Lng: -104.99}
when := time.Date(2027, 6, 8, 4, 0, 0, 0, time.UTC) // 22:00 MDT on the 7th

bl, err := moon.BrightLimbAt(obs, when)
if err != nil {
	panic(err)
}
fmt.Printf("%.1f %.1f %.1f\n", bl.PositionAngle, bl.Parallactic, bl.ZenithAngle)
// 283.2 52.7 230.5: the Moon is 16% lit at 15° altitude in the west-northwest,
// and its lit side faces down and to the right, toward the Sun below the horizon.
```

To draw it, with the top of the screen as up and y growing downward, the unit vector toward the lit side is `(-sin a, -cos a)` for `a = ZenithAngle` in radians: `(0.77, 0.64)` here. Rotate the Moon image so its terminator is perpendicular to that vector, and scale the lit part by `Illumination`.

### Units, frames, and sources

- All three angles are degrees. `PositionAngle` and `ZenithAngle` are in `[0, 360)`, and `Parallactic` is in `(-180, 180]`.
- `PositionAngle` is referred to the true equator and equinox of date. JPL Horizons reports its `PsAng` from the J2000 (ICRF) pole, and precession since 2000 turns that by about 0.15° at the Moon. The package test converts before comparing.
- The formulas are Meeus, *Astronomical Algorithms*, 2nd ed., 48.5 (the position angle of the bright limb, anchored to example 48.a, 285.0°) and 14.1 (the parallactic angle).

The angles are always defined, because the Sun and the Moon never coincide exactly. Near New and Full Moon, with less than about 0.1% or more than about 99.9% of the disc lit (see `Illumination`), there is no visible bright limb to orient and the angle swings quickly, so treat it as undefined there. Against JPL Horizons DE441 (`PsAng`, turned from the J2000 pole to the pole of date) the position angle agrees to within 0.01°.

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
