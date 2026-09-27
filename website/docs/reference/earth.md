---
id: earth
title: earth
sidebar_position: 2
---

# 🌍 Package `earth`

Import path: `github.com/Bugs5382/go-astronomy/earth`

The Earth vantage. It adds Earth-bound conventions on top of the universal [`sun`](./sun.md) geometry: alt/az of the Sun for an observer, named twilight bands, sunrise and sunset, solar noon, atmospheric refraction, polar states, seasons, a ground-darkness factor, and star projection. All altitudes are geometric center altitudes in degrees. The package is stateless and concurrency-safe: time is always a parameter.

## 🌅 Sun position and track

```go
func SunPosition(obs astronomy.Observer, t time.Time) astronomy.Position

type SunSample struct {
	Time         time.Time
	Altitude     float64
	Azimuth      float64
	TimeProgress float64
}

func SunTrack(obs astronomy.Observer, date time.Time, samples int) []SunSample
```

- `SunPosition` returns the geometric alt/az of the Sun's disc center, paired with its apparent diameter, at instant `t`. A negative altitude is below the horizon and is a valid answer, so there is no error return.
- `SunTrack` samples the Sun's arc across the civil day containing `date`, resolved in the observer's time zone from local midnight to the next local midnight. Exactly `samples` points are returned, evenly spaced in time and inclusive of both endpoints; the span honors daylight-saving transitions (23 or 25 hours). Fewer than two samples returns `nil`.

## 🌇 Twilight bands (SunTimes)

```go
func NewSunTimes(obs astronomy.Observer, date time.Time) (*SunTimes, error)
func NewSunTimesWith(obs astronomy.Observer, date time.Time, seg Segmentation) (*SunTimes, error)

func (s *SunTimes) Segments() []Segment
func (s *SunTimes) SolarNoon() (time.Time, bool)
func (s *SunTimes) Polar() (PolarState, bool)
func (s *SunTimes) DayStart() time.Time
func (s *SunTimes) DayEnd() time.Time
func (s *SunTimes) DayLength() time.Duration
```

`NewSunTimes` resolves one civil day of named bands in `obs.TZ` using `DefaultSegmentation`. `NewSunTimesWith` takes a custom `Segmentation`. `SolarNoon` and `Polar` return a boolean that is false when the value does not apply (for example, no solar noon crossing on a polar day).

### Instant resolution and midnight rollover

```go
func SegmentAt(obs astronomy.Observer, t time.Time) (Segment, float64, error)
func SegmentAtWith(obs astronomy.Observer, t time.Time, seg Segmentation) (Segment, float64, error)
```

`SegmentAt` answers "which band, and how far through it (a `[0, 1]` fraction), at `t`". It stitches across midnight, so a caller asks only for `now` and a live clock has no gap at the day boundary. See [Concepts](../concepts.md).

## 🧩 Segmentation model

```go
type Segment struct {
	Label   string
	From    time.Time
	To      time.Time
	Seconds float64
}
func (s Segment) Contains(t time.Time) bool

type Level struct {
	Altitude     float64 // geometric center altitude of the boundary, degrees
	Rising       string  // label of the band above this level while the Sun climbs
	Setting      string  // label of the band above this level while the Sun sinks
	DipCorrected bool    // lower this level by the horizon dip for the observer's height
	Refracted    bool    // this level includes the 34′ horizon refraction, scaled by the air
}

type Segmentation struct {
	Night            string     // band below the lowest level
	Horizon          float64    // altitude used for sunrise/sunset and polar detection
	HorizonRefracted bool       // Horizon includes the 34′ horizon refraction
	Levels           []Level    // altitude boundaries in ascending order
	Atmosphere       Atmosphere // the air that scales the refraction; zero value = ISA
}

var DefaultSegmentation Segmentation

func (s Segmentation) WithTwilightDip() Segmentation
func (s Segmentation) WithAtmosphere(a Atmosphere) Segmentation
```

A `Segmentation` divides the Sun's altitude over a civil day into named bands. `DefaultSegmentation` is the Earth default: astronomical (−18°), nautical (−12°), and civil (−6°) twilight, the sunrise/sunset horizon crossing (−0.833°, upper limb including refraction), a short sunrise/sunset band up to −0.3°, golden hour up to +6°, and full day above that. A band takes its `Rising` label while the Sun climbs through it and its `Setting` label while the Sun sinks, so a twilight band that runs past local midnight keeps one label on both dates. The day is split at solar noon, and at solar midnight when the Sun stays above the lowest level all night, so the daytime band has morning (`Rising`) and afternoon (`Setting`) halves. Treat `DefaultSegmentation` as read-only; build a fresh value to customize.

The default band labels are exported as constants: `LabelNight`, `LabelAstronomicalDawn`, `LabelNauticalDawn`, `LabelCivilDawn`, `LabelSunrise`, `LabelGoldenHour`, `LabelDay`, `LabelSunset`, `LabelCivilDusk`, `LabelNauticalDusk`, `LabelAstronomicalDusk`.

## 🌫️ Refraction and the horizon

```go
const HorizonAltitude = -0.833
const HorizonRefraction = 34.0 / 60 // degrees
func Refraction(apparentAltDeg float64) float64
```

`Refraction` returns the atmospheric refraction, in degrees, that lifts a body seen at a given apparent altitude above its true geometric altitude, using Bennett's formula for sea-level air at 1010 hPa and 10 °C (about 0.57° at the horizon, falling to zero near the zenith). `HorizonAltitude` (−0.833°) is the geometric center altitude at which the Sun's upper limb sits on the horizon under the standard 34′ of horizon refraction (`HorizonRefraction`) plus the mean solar semidiameter; it is the sea-level sunrise/sunset threshold.

## ⛰️ Observer height and the horizon dip

```go
const DipArcminPerRootMetre = 1.76
func HorizonDip(h astronomy.Height) float64 // degrees
```

`astronomy.Observer.Height` is optional; the zero value is sea level and reproduces the sea-level answers exactly. From height the sea horizon sits below the astronomical horizon by the dip, `1.76′ × √h`, so `NewSunTimes` and `SegmentAt` move sunrise earlier and sunset later: 6.8 minutes at Denver (1609 m) in June, which is 7.3 minutes of dip less half a minute because the thinner air refracts less (see the next section). `HorizonDip` returns the dip in degrees, and zero at or below sea level.

```go
fmt.Printf("%.3f\n", earth.HorizonDip(astronomy.Meters(1609)))  // 1.177
fmt.Printf("%.3f\n", earth.HorizonDip(astronomy.Feet(36000)))   // 3.073
```

| height | dip |
| --- | --- |
| 10 m | 0.09° |
| 100 m | 0.29° |
| 1524 m (5000 ft) | 1.15° |
| 1609 m (Denver) | 1.18° |
| 3640 m (La Paz) | 1.77° |
| 10973 m (36000 ft) | 3.07° |

By default only sunrise and sunset move. `DefaultSegmentation` marks the sunrise and sunset crossing (−0.833°) and the top of the sunrise band (−0.3°) as `DipCorrected`, and the segmentation's `Horizon` always takes the dip. The civil, nautical, and astronomical twilight levels do not: by the USNO convention, they are the Sun's depression below the astronomical horizon. `WithTwilightDip` returns a copy with every level dip-corrected; at Denver in March it moves civil dawn 6.1 minutes earlier. A NaN or infinite height is rejected with `astronomy.ErrInvalidHeight`. See [Observer and height](./observer.md) for the three ways to set a height, the resolvers, and a moving-observer example.

## 🎈 Refraction and height

```go
type Atmosphere struct{ /* unexported */ }
var StandardAtmosphere Atmosphere // the zero value: the ISA at the observer's height
func MeasuredAtmosphere(pressureHPa, temperatureC float64) Atmosphere

func (a Atmosphere) Factor(h astronomy.Height) float64
func (a Atmosphere) Refraction(apparentAltDeg float64, h astronomy.Height) float64
func (a Atmosphere) Measured() (pressureHPa, temperatureC float64, ok bool)
func (a Atmosphere) Err() error // ErrInvalidAtmosphere for air that cannot exist

func HorizonAltitudeAt(h astronomy.Height) float64 // the Sun's rise/set altitude at height

var ErrInvalidAtmosphere error // code 7017, CodeInvalidAtmosphere
```

The dip and the refraction change together with height, in opposite directions. The dip lowers the horizon: it is the Nautical Almanac's observed dip, so it already includes the bending of the line of sight down to the sea horizon. The refraction that lifts the body itself scales with the density of the air, and the air above sea level is thinner, so a body on the horizon is lifted less and the horizon threshold rises by the refraction lost.

`StandardAtmosphere.Factor(h)` is the ISA density ratio `P(h)/P₀ × T₀/T(h)`: exactly 1 at sea level, so every sea-level answer is unchanged, above 1 below sea level, and following the ISA layers to 84.9 km (it holds well past 15 km). `MeasuredAtmosphere(p, t)` uses a local reading of station pressure in hPa and temperature in °C instead, with the absolute factor `P/1010 × 283/(273+T)`, whatever the height; use the station pressure, not the sea-level pressure most weather reports give. Air that cannot exist (a pressure that is not positive, a temperature at or below −273 °C, NaN) is rejected with `ErrInvalidAtmosphere` by `NewSunTimesWith` and `SegmentAtWith`.

| height | dip | factor | refraction lost | `HorizonAltitudeAt` |
| --- | --- | --- | --- | --- |
| sea level | 0° | 1.000 | 0° | −0.833° |
| 1524 m (5000 ft) | 1.145° | 0.862 | 0.078° | −1.900° |
| 1609 m (Denver) | 1.177° | 0.854 | 0.082° | −1.927° |
| 3640 m (La Paz) | 1.770° | 0.695 | 0.173° | −2.430° |
| 10668 m (35000 ft) | 3.030° | 0.311 | 0.391° | −3.472° |
| 15000 m | 3.593° | 0.159 | 0.477° | −3.949° |

`HorizonAltitudeAt(h)` is `HorizonAltitude + (1 − factor) × HorizonRefraction − HorizonDip(h)` in the standard atmosphere. The segmentation applies the same correction to the `Horizon` (when `HorizonRefracted`) and to every `Refracted` level; `DefaultSegmentation` marks the sunrise crossing and the top of the sunrise band, the two levels that take the dip, and leaves the twilight levels, which are geometric depressions with no refraction in them. `WithAtmosphere` swaps in measured air:

```go
cold := earth.MeasuredAtmosphere(845, -15) // a cold winter morning in Denver
day, err := earth.NewSunTimesWith(obs, date, earth.DefaultSegmentation.WithAtmosphere(cold))
// standard air factor 0.854, measured air factor 0.918: sunrise 13 s earlier
```

Every height-aware call uses the standard atmosphere by default: `NewSunTimes`, `SegmentAt`, the Moon's `NextRise`, `NextSet`, and `ApparentPosition`, and every planet's `NextRise` and `NextSet`. Measured air goes through the segmentation (`WithAtmosphere`) or directly through `Atmosphere.Refraction` for a single altitude; the Moon and planet rise and set searches take the standard atmosphere only. JPL Horizons refracts with the sea-level air at any height, so to compare with it use `MeasuredAtmosphere(1010, 10)`, whose factor is exactly 1.

## ❄️ Polar states

```go
type PolarState int
const (
	NotPolar PolarState = iota
	MidnightSun
	PolarNight
)
func (p PolarState) String() string
```

`MidnightSun` means the Sun stays above the horizon for the whole day; `PolarNight` means it stays below. `String` returns `not_polar`, `midnight_sun`, or `polar_night`.

## 🍂 Seasons

```go
type Season int // Spring, Summer, Fall, Winter
func (s Season) String() string

type SeasonInfo struct {
	Season        Season
	Next          Season
	Start         time.Time // UTC instant the season began
	End           time.Time // UTC instant the next season begins
	Progress      float64   // fraction of the season elapsed, [0, 1)
	DaysElapsed   float64
	DaysUntilNext float64
}

func SeasonAt(obs astronomy.Observer, t time.Time) (SeasonInfo, error)
```

`SeasonAt` returns the astronomical season in progress, computed from the equinox and solstice instants and flipped for the Southern Hemisphere by the sign of the observer's latitude. It returns a coded error for an out-of-range latitude or longitude.

## 🌑 Ground darkness

```go
const NightDarknessDayAltitude = 0.0
const NightDarknessNightAltitude = -18.0
func NightDarkness(obs astronomy.Observer, t time.Time) float64
```

`NightDarkness` returns a `[0, 1]` geometry factor describing how dark the ground is, driven purely by the Sun's altitude: 0 at or above the horizon (full day), ramping smoothly (a smoothstep) to 1 as the Sun sinks to the astronomical threshold (−18°), then staying at 1 through deep night. This is geometry, not appearance — the consumer multiplies it into its own ground colors.

## ⭐ Star projection

```go
func StarPosition(s star.Star, obs astronomy.Observer, t time.Time) (astronomy.Horizontal, error)
```

`StarPosition` projects a [`star.Star`](./star.md)'s J2000 equatorial position to the observer's local alt/az at instant `t`. Turning a catalog star into a local direction needs a rotating body and an observer, which is why this Earth-vantage function lives here rather than in the universal `star` package.

## 🚑 Errors

The observer-validating functions return `go-apperr` coded errors. Match with `errors.Is` against `ErrInvalidLatitude`, `ErrInvalidLongitude`, or `ErrInvalidSegmentation`, or recover the stable code with `apperr.Code`. See [Errors](./errors.md).

Full godoc: [pkg.go.dev/github.com/Bugs5382/go-astronomy/earth](https://pkg.go.dev/github.com/Bugs5382/go-astronomy/earth).
