---
id: satellite
title: satellite
sidebar_position: 5
---

# 🛰️ Package `satellite`

Import path: `github.com/Bugs5382/go-astronomy/satellite`

Earth satellites, the ISS among them, placed in an observer's sky from an element set the caller supplies. An element set is measured, not derived, and goes stale in days, so it is an input: the package parses it and never fetches one.

## 🗂️ The satellite packages

| package | import path | what it is |
| --- | --- | --- |
| `satellite` | `github.com/Bugs5382/go-astronomy/satellite` | this page: the engine (TLE and OMM parsing, SGP4/SDP4, look angles, passes, magnitude), trackers, element sources, and the cache |
| `iss` | `.../satellite/iss` | [the International Space Station](./satellites/iss.md), NORAD 25544 |
| `hubble` | `.../satellite/hubble` | [the Hubble Space Telescope](./satellites/hubble.md), NORAD 20580 |
| `tiangong` | `.../satellite/tiangong` | [the Tiangong space station](./satellites/tiangong.md), NORAD 48274 |
| `celestrak` | `.../satellite/celestrak` | [the CelesTrak element fetcher](./satellites/celestrak.md) |
| `horizons` | `.../satellite/horizons` | [the JPL Horizons ephemeris fetcher](./satellites/horizons.md) |
| `jwst` | `.../satellite/jwst` | [the James Webb Space Telescope](./satellites/jwst.md), at L2, from Horizons |
| `roman` | `.../satellite/roman` | [the Nancy Grace Roman Space Telescope](./satellites/roman.md), bound for L2, from Horizons |

Nothing is fetched unless the caller builds a fetcher and passes it in. The engine and the named packages compute everything locally.

## 🚀 Quick example

```go
e, err := satellite.ParseTLE(line1, line2) // from CelesTrak or Space-Track, fetched by you
if err != nil {
	panic(err)
}
denver := astronomy.Observer{Lat: 39.74, Lng: -104.99}
opt := satellite.DefaultPassOptions()
opt.StdMagnitude = satellite.ISSStandardMagnitude
from := time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)
passes, err := satellite.Passes(denver, e, from, from.Add(6*time.Hour), opt)
if err != nil {
	panic(err)
}
for _, p := range passes {
	fmt.Printf("%s-%s peak %4.1f at %s visible %-5v",
		p.Rise.Time.Format("15:04:05"), p.Set.Time.Format("15:04:05"),
		p.Peak.Altitude, p.Peak.Time.Format("15:04:05"), p.Visible)
	if !p.ShadowEntry.IsZero() {
		fmt.Printf(" enters shadow %s", p.ShadowEntry.Format("15:04:05"))
	}
	fmt.Println()
}
```

```text
01:18:12-01:28:44 peak 34.7 at 01:23:28 visible true  enters shadow 01:27:04
02:55:02-03:05:31 peak 28.7 at 03:00:16 visible true  enters shadow 03:00:03
04:33:21-04:42:27 peak 11.8 at 04:37:54 visible false
```

The examples on this page use a synthetic ISS-like element set with an epoch of 2026-09-26 (it is in `satellite/example_test.go`), and every output shown is what those Example tests print.

## 📄 Element sets

```go
func ParseTLE(line1, line2 string) (Elements, error)
func ParseOMM(r io.Reader) ([]Elements, error)

func (e Elements) Epoch() time.Time
func (e Elements) Age(t time.Time) time.Duration
func (e Elements) Propagate(t time.Time) (posKm, velKmS [3]float64, err error) // TEME
```

`ParseTLE` validates both checksums and the column layout and reads Alpha-5 catalogue numbers (`A0001` is 100001). `ParseOMM` reads CCSDS Orbit Mean-Elements Messages as JSON (an object or an array, as CelesTrak serves them) or XML, and rejects any message whose mean element theory is not SGP4 or whose frame is not TEME. `Age` says how far an instant is from the epoch, so a consumer can decide whether an answer is still worth trusting: a low orbit's position degrades by kilometres a day.

```go
e, err := satellite.ParseTLE(line1, line2)
fmt.Println(e.SatNum, e.IntlDesignator, e.Epoch().Format(time.RFC3339))
// 25544 98067A 2026-09-26T12:25:40Z

sets, err := satellite.ParseOMM(strings.NewReader(celestrakJSON))
// []Elements, one per record; sets[0].Epoch() keeps the microseconds of EPOCH

age := e.Age(time.Now())
if age > 3*24*time.Hour {
	// refresh the element set before trusting a low orbit
}
```

`type Elements struct` holds the parsed element set and the initialised propagator; it is read-only after parsing and safe to share between goroutines. It exposes the parsed fields: `CatalogNumber` (as printed, possibly Alpha-5), `SatNum`, `Classification`, `IntlDesignator`, `EpochYear`, `EpochDay`, `NDot`, `NDDot`, `BStar`, `EphemerisType`, `ElementSetNumber`, `Inclination`, `RAAN`, `Eccentricity`, `ArgPerigee`, `MeanAnomaly` (degrees), `MeanMotion` (revolutions per day), and `RevNumber`.

`Propagate(t)` returns the TEME position in km and velocity in km/s; `PropagateMinutes(m)` does the same for minutes since the epoch, the form the verification data uses:

```go
pos, vel, err := e.Propagate(e.Epoch().Add(90 * time.Minute))
// TEME position 3604.8 -2242.9 5294.2 km
// TEME velocity 3.391 6.851 0.588 km/s
```

Propagation is SGP4/SDP4 (near-Earth and deep-space), ported from the public-domain reference code of Vallado, Crawford, Hujsak and Kelso (2006, *Revisiting Spacetrack Report #3*), with WGS-72 constants and the improved operation mode. It matches the reference verification output to under 0.1 m.

## 🔭 Position

```go
type Look struct {
	astronomy.Horizontal         // geometric altitude and azimuth, degrees
	RangeKm, RangeRateKmS float64
	Sunlit                bool    // the Sun's centre is visible from the satellite
	SunAltitude           float64 // the Sun's altitude for the observer
	PhaseAngle            float64 // Sun-satellite-observer, degrees
	Latitude, Longitude, AltitudeKm float64 // sub-satellite point, WGS84
}

func Position(obs astronomy.Observer, e Elements, t time.Time) (Look, error)
func (l Look) Magnitude(stdMag float64) float64
const ISSStandardMagnitude = -1.8
```

```go
sydney := astronomy.Observer{Lat: -33.87, Lng: 151.21}
l, err := satellite.Position(sydney, e, time.Date(2026, 9, 26, 4, 5, 3, 0, time.UTC))
// altitude 88.51, azimuth 294.8, range 433.5 km
// sunlit true, Sun altitude 44.0
// over -33.83, 151.11 at 433.4 km
```

| field | unit | meaning |
| --- | --- | --- |
| `Altitude`, `Azimuth` | degrees | geometric (no refraction); azimuth clockwise from true north |
| `RangeKm`, `RangeRateKmS` | km, km/s | from the observer; the rate is positive when moving away |
| `Sunlit` | | the Sun's centre is visible from the satellite (outside the umbra, at least half lit through any penumbra) |
| `SunAltitude` | degrees | the Sun's geometric altitude for the observer |
| `PhaseAngle` | degrees | Sun-satellite-observer; 0 when the observer sees the fully lit side |
| `Latitude`, `Longitude`, `AltitudeKm` | degrees, km | the WGS84 geodetic sub-satellite point and height |

The TEME state is rotated to the Earth-fixed frame through GMST 1982, the angle SGP4 defines TEME against (polar motion, a metre-level effect, is ignored, and UT1 is taken as UTC). `Magnitude` models the satellite as a diffusely reflecting sphere from its standard magnitude at 1000 km and a 90° phase angle. The standard magnitude is not part of an element set, so the caller supplies it; the ISS's is exported for convenience. It returns `+Inf` in shadow.

```go
std := satellite.Look{RangeKm: 1000, PhaseAngle: 90, Sunlit: true}
near := satellite.Look{RangeKm: 450, PhaseAngle: 60, Sunlit: true}
fmt.Printf("%.2f %.2f\n", std.Magnitude(satellite.ISSStandardMagnitude), near.Magnitude(satellite.ISSStandardMagnitude))
// -1.80 -4.24
```

The magnitude is `std + 5 log10(range / 1000 km) − 2.5 log10(sin β + (π − β) cos β)` for a phase angle `β`: a diffuse sphere, normalised so that 1000 km and 90° give the standard magnitude.

## 🌠 Passes

```go
type PassOptions struct {
	MinAltitude     float64       // degrees, 0 for the geometric horizon
	DarkSunAltitude float64       // the observer is dark below this Sun altitude, -6 by default
	Step            time.Duration // scan interval, 10 s by default
	StdMagnitude    float64       // NaN (the default) for no magnitudes
}
func DefaultPassOptions() PassOptions

type Pass struct {
	Rise, Peak, Set         PassEvent
	Visible                 bool
	VisibleFrom, VisibleTo  time.Time
	ShadowEntry, ShadowExit time.Time // zero when the pass has none
}

func Passes(obs astronomy.Observer, e Elements, from, to time.Time, opt PassOptions) ([]Pass, error)
```

A pass is an interval, not an instant: the station crosses the sky in minutes, and anything sampled at the Sun's cadence misses it. `Visible` means that for part of the pass the satellite is sunlit while the Sun is below `DarkSunAltitude` for the observer, and `ShadowEntry` is where it vanishes into the Earth's shadow. The window is at most 31 days (`MaxPassWindow`).

`DefaultPassOptions()` is `{MinAltitude: 0, DarkSunAltitude: -6, Step: 10s, StdMagnitude: NaN}`. Each `PassEvent` carries the `Time`, the `Altitude` and `Azimuth`, the `RangeKm`, `Sunlit`, and the `Magnitude` (`NaN` without a standard magnitude, `+Inf` in shadow).

Against Skyfield, run on the same element set for two observers over two days, rise, peak, and set agree to 0.2 s, shadow crossings to 0.05 s, and peak altitude to 0.01°.

## 🎯 Trackers and element sources

```go
type ElementSource interface {
	Elements(ctx context.Context, catalog int) (Elements, error)
}
func StaticElements(sets ...Elements) ElementSource

var ErrNoElements, ErrStaleElements error

func NewTracker(catalog int, name string, stdMag float64, src ElementSource) *Tracker
func (t *Tracker) Position(ctx context.Context, obs astronomy.Observer, at time.Time) (Look, error)
func (t *Tracker) Passes(ctx context.Context, obs astronomy.Observer, from, to time.Time) ([]Pass, error)
func (t *Tracker) PassesWith(ctx context.Context, obs astronomy.Observer, from, to time.Time, opt PassOptions) ([]Pass, error)
func (t *Tracker) Magnitude(l Look) float64
func (t *Tracker) Elements(ctx context.Context) (Elements, error)
func (t *Tracker) CatalogNumber() int
func (t *Tracker) Name() string
func (t *Tracker) StandardMagnitude() float64
```

- **Tracker.** A `Tracker` follows one satellite by its catalogue number, taking element sets from an explicit `ElementSource`. That source is `StaticElements` over sets you already have, or a fetcher such as `celestrak.Client`. The named packages (`iss`, `hubble`, `tiangong`) return one from `New(src)`.
- **Missing and stale sets.** A source with no set for the number gives `ErrNoElements`. A source that could only return an old set returns it with an error wrapping `ErrStaleElements`. The tracker still computes the position and passes from it and hands that error back, so the caller decides.

## 🗄️ The cache

```go
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, expiry time.Duration) error
}
func NewMemoryCache(opts ...CacheOption) *MemoryCache
func WithCacheClock(now func() time.Time) CacheOption
```

- **What it is for.** The CelesTrak and Horizons fetchers keep what they fetch in a `Cache`. The default is the in-process `MemoryCache`.
- **Plugging in your own.** The interface is small and free of any store dependency. A consumer can plug in Redis (or anything else) to share fetches across restarts and replicas: `celestrak.New(celestrak.WithCache(myRedisCache))`.
- **Semantics.** An expiry of zero keeps a value until it is replaced. A store error is treated as a miss.

## 🧭 Frames and units

| quantity | frame | unit |
| --- | --- | --- |
| `Propagate` | TEME (true equator, mean equinox), the SGP4 output frame | km, km/s |
| `Look` | local horizon of a WGS84 observer, geometric | degrees, km |
| sub-satellite point | WGS84 geodetic | degrees, km |
| element angles | as printed in the TLE or OMM | degrees, revolutions per day |
| times | `time.Time`, UTC | |

## 🎯 Accuracy and staleness

The propagator reproduces the reference verification output (all 666 states of 33 satellites in `SGP4-VER.TLE` and `tcppver.out`) to under 0.1 m, and python-sgp4 to 7e-11 km. What limits an answer is the element set, not the arithmetic: an ISS element set is good to about a kilometre for a day or two and degrades quickly after, because the station is low enough for drag to matter and it manoeuvres. Check `Elements.Age` and refresh the set before it matters.

## 🚑 Errors

Every error is go-apperr coded, so `errors.Is` matches the sentinel and `apperr.Code` recovers the number.

| code | sentinels | when |
| --- | --- | --- |
| 7014 `CodeInvalidElements` | `ErrMalformedTLE`, `ErrChecksum`, `ErrMalformedOMM` | a TLE or OMM that cannot be parsed, fails a checksum, or is not SGP4/TEME/UTC |
| 7015 `CodeSatellitePropagation` | `ErrEccentricity` (1), `ErrMeanMotion` (2), `ErrPerturbedEccentricity` (3), `ErrSemiLatusRectum` (4), `ErrDecayed` (6), `ErrNotInitialised` | SGP4 cannot propagate to the instant; the reference error code is in parentheses |
| 7016 `CodeInvalidPassWindow` | `ErrInvalidPassWindow` | a pass window whose end is not after its start, or longer than `MaxPassWindow` (31 days) |
| 7001, 7002 | `ErrInvalidLatitude`, `ErrInvalidLongitude` | an observer out of range |

A propagation failure also carries a `*PropagationError`:

```go
type PropagationError struct {
	Code    int     // the reference code: 1, 2, 3, 4, or 6
	Minutes float64 // minutes since the epoch of the failing request
	Value   float64 // the offending quantity (eccentricity, radius, and so on)
}
```

```go
_, _, err := e.PropagateMinutes(1e6)
var pe *satellite.PropagationError
if errors.As(err, &pe) && errors.Is(err, satellite.ErrDecayed) {
	fmt.Printf("decayed %.0f minutes after the epoch\n", pe.Minutes)
}
```

For a decayed satellite `Propagate` still returns the last position with the error, as the reference code does, and `Position` fills in the `Look`.

## 📖 Sources

- D. A. Vallado, P. Crawford, R. Hujsak, and T. S. Kelso, "Revisiting Spacetrack Report #3", AIAA 2006-6753, and its public-domain reference code, ported here by way of python-sgp4 (Brandon Rhodes, MIT).
- CCSDS 502.0-B, Orbit Data Messages, for the OMM fields.
- Skyfield (Brandon Rhodes) with JPL DE421, for the independent pass checks.
