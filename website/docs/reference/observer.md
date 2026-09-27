---
id: observer
title: Observer and height
sidebar_position: 1
---

# 🧍 Observer and height

Import path: `github.com/Bugs5382/go-astronomy` (the root package), plus `github.com/Bugs5382/go-astronomy/openmeteo` for the optional lookup.

```go
type Observer struct {
	Lat    float64        // degrees, positive north, [-90, 90]
	Lng    float64        // degrees, positive east, [-180, 180]
	TZ     *time.Location // nil means UTC
	Height Height         // optional; the zero value is sea level
}
```

An observer's height above sea level is optional. From height the sea horizon lies below the astronomical horizon by the dip, so the Sun and Moon rise earlier and set later, and the height enters their parallax (under an arc second). Every function that takes an `Observer` uses its height: `earth.NewSunTimes`, `earth.SegmentAt`, `earth.SunPosition`, and the Moon's `Position`, `NextRise`, and `NextSet`.

## 📏 Height

```go
type Height struct{ /* metres */ }

const MetersPerFoot = 0.3048
var SeaLevel Height // the zero value

func Meters(m float64) Height
func Feet(ft float64) Height
func (h Height) Meters() float64
func (h Height) Feet() float64
func (h Height) String() string // "1524 m"
func (h Height) Err() error     // ErrInvalidHeight for NaN or infinity, else nil

var ErrInvalidHeight error // code 7011, CodeInvalidHeight
```

A `Height` is stored in metres, the unit every calculation uses. One foot is exactly 0.3048 m, so `Feet(5000)` equals `Meters(1524)`. Any real value is valid as given: there is no check against real terrain, so New York at 5000 ft is accepted on purpose, and so are the Dead Sea shore (−430 m), Everest (8849 m), and a plane at 36000 ft. Only NaN and the infinities are invalid; every function that returns an error rejects them with `ErrInvalidHeight`. `SunPosition`, which has no error to return, treats an invalid height as sea level.

## 🧭 The three modes

### Always sea level

```go
nyc := astronomy.Observer{Lat: 40.71, Lng: -74.01, TZ: time.UTC}
fmt.Println("height:", nyc.Height) // height: 0 m
// sunrise on 2027-06-21: 09:25:00 UTC
```

Leave `Height` out. The answers are exactly the sea-level answers, and nothing reaches the network.

### By hand

```go
high := astronomy.Observer{Lat: 40.71, Lng: -74.01, TZ: time.UTC, Height: astronomy.Feet(5000)}
same := astronomy.Observer{Lat: 40.71, Lng: -74.01, TZ: time.UTC, Height: astronomy.Meters(1524)}
fmt.Println(high == same)                        // true
fmt.Printf("%.3f\n", earth.HorizonDip(high.Height)) // 1.145 degrees
// sunrise 7.2 minutes earlier than at sea level
```

### Lookup

```go
type ElevationResolver interface {
	Elevation(ctx context.Context, lat, lng float64) (Height, string, error)
}

const (
	SourceCaller    = "caller"
	SourceStatic    = "static"
	SourceOpenMeteo = "open-meteo"
	SourceSeaLevel  = "sea-level"
)

func StaticElevation(h Height) ElevationResolver
func CallerElevation(h Height, ok bool) ElevationResolver
func ChainElevation(resolvers ...ElevationResolver) ElevationResolver
func ResolveObserverWith(ctx context.Context, r ElevationResolver, lat, lng float64) (Observer, string, error)
```

A resolver returns the height and the source that produced it. Every lookup failure is a fallback, not an error: the resolver answers `SeaLevel` with `SourceSeaLevel`. The error is only for a cancelled or expired context.

- `StaticElevation(h)` always answers `h` (`"static"`), for a fixed installation.
- `CallerElevation(h, ok)` answers `h` (`"caller"`) when `ok` is set, for a height the caller already has, such as the ground elevation in a weather reading. When `ok` is unset it reports sea level, so a chain moves on.
- `ChainElevation(...)` asks each resolver in order and returns the first answer that is not `"sea-level"`. It falls back to sea level only at the end.
- `ResolveObserverWith` builds the observer at the coordinate with the height found. It leaves `TZ` nil, so set it on the result.

```go
weatherHeight, haveIt := astronomy.Meters(0), false // no reading this time
resolver := astronomy.ChainElevation(
	astronomy.CallerElevation(weatherHeight, haveIt),
	astronomy.StaticElevation(astronomy.Meters(21)),
)
obs, source, err := astronomy.ResolveObserverWith(ctx, resolver, 40.71, -74.01)
// 21 m from static

obs, source, _ = astronomy.ResolveObserverWith(ctx, astronomy.ChainElevation(), 40.71, -74.01)
// 0 m from sea-level
```

## 🌐 Open-Meteo

```go
const DefaultBaseURL = "https://api.open-meteo.com/v1/elevation"
const DefaultRound = 0.01
const DefaultTimeout = 10 * time.Second

func New(opts ...Option) *Resolver
func Default() *Resolver // shared, so its cache lasts for the process
func ResolveObserver(ctx context.Context, lat, lng float64) (astronomy.Observer, string, error)

func WithHTTPClient(c *http.Client) Option
func WithTimeout(d time.Duration) Option
func WithBaseURL(u string) Option
func WithRound(deg float64) Option
func WithLogger(l log.Logger) Option

func (r *Resolver) Elevation(ctx context.Context, lat, lng float64) (astronomy.Height, string, error)
func (r *Resolver) Cached(lat, lng float64) (astronomy.Height, bool)
func (r *Resolver) Store(lat, lng float64, h astronomy.Height)
```

`openmeteo.Resolver` asks the [Open-Meteo elevation API](https://open-meteo.com/en/docs/elevation-api) at `https://api.open-meteo.com/v1/elevation?latitude=..&longitude=..`. The API is worldwide and free, needs no key, and serves the Copernicus DEM at 90 m. It uses only `net/http`, and it is the only package in the module that reaches the network. A program that never imports it never links it.

- **Context and timeout:** it respects `ctx`, and bounds each lookup by `WithTimeout` (10 s by default) on top of any deadline the context has.
- **Fallbacks:** a timeout, a non-200 reply, bad JSON, a missing or null value, or an out-of-range coordinate answers sea level with `"sea-level"`, and is not cached.
- **Cache:** heights never change, so each is kept in an in-process map with no expiry, keyed by the coordinate rounded to `WithRound` (0.01°, about 1.1 km). The query is made at the cell's rounded coordinate, so every point in a cell shares one value. `Cached` and `Store` connect the map to a durable cache you keep.
- **Logging:** it logs through go-log. Coordinates, cache hits, requests, their status, and their duration go to debug, and fallbacks to warn. It logs nothing else about the caller.

```go
lookup := openmeteo.New(openmeteo.WithHTTPClient(client))
h, source, _ := astronomy.ChainElevation(astronomy.CallerElevation(astronomy.SeaLevel, false), lookup).
	Elevation(ctx, 39.74, -104.99)
// 1597 m from open-meteo

h, source, _ = astronomy.ChainElevation(astronomy.StaticElevation(astronomy.Meters(1609)), lookup).
	Elevation(ctx, 39.74, -104.99)
// 1609 m from static (the API is not asked)
```

## ✈️ Moving observers

A caller in motion, such as a plane, passes its position and height for each instant to each time-based call; nothing about the observer is captured between calls. This is a flight from New York (JFK) to London (LHR) on the June solstice, leaving at 00:00 UTC and cruising at 36000 ft. From altitude the Sun clears the dipped horizon before it rises for anyone on the ground below.

```go
jfk, lhr := [2]float64{40.64, -73.78}, [2]float64{51.47, -0.45}
depart := time.Date(2027, 6, 21, 0, 0, 0, 0, time.UTC)
for _, f := range []float64{0, 0.25, 0.5, 0.7, 0.75, 1} {
	lat, lng := greatCircle(jfk, lhr, f) // f of the way along the route
	height := astronomy.Feet(36000)
	if f == 0 || f == 1 {
		height = astronomy.SeaLevel // on the runway
	}
	obs := astronomy.Observer{Lat: lat, Lng: lng, Height: height}
	when := depart.Add(time.Duration(f * float64(7*time.Hour)))
	sun := earth.SunPosition(obs, when)
	horizon := earth.HorizonAltitude - earth.HorizonDip(obs.Height)
	fmt.Println(when.Format("15:04"), sun.Altitude > horizon, sun.Altitude > earth.HorizonAltitude)
}
```

```text
00:00  40.64  -73.78      0 ft sun   3.98 up in the air true  on the ground true
01:45  47.58  -59.32  36000 ft sun -12.86 up in the air false on the ground false
03:30  52.22  -41.30  36000 ft sun -13.76 up in the air false on the ground false
04:54  53.64  -24.90  36000 ft sun  -2.43 up in the air true  on the ground false
05:15  53.64  -20.70  36000 ft sun   1.66 up in the air true  on the ground true
07:00  51.47   -0.45      0 ft sun  26.78 up in the air true  on the ground true
```

The full program, with the great-circle helper, is `Example_flightNYCToLondon` in `example_height_test.go`.

## 🎯 The dip, accuracy, and limits

- **The dip:** `earth.HorizonDip(h)` is `1.76′ × √h` for `h` in metres (`earth.DipArcminPerRootMetre`). That is the observed dip from the Nautical Almanac, which includes standard terrestrial refraction. It is zero at or below sea level. At 36000 ft it is 3.07°.
- **Twilight:** by the USNO convention, only sunrise and sunset move. `earth.DefaultSegmentation` marks only those levels `DipCorrected`, and `Segmentation.WithTwilightDip()` moves every level.
- **The horizon it assumes:** the dip assumes an unobstructed sea horizon. It is right on a mountaintop, a coast, or in the air, and optimistic in a valley, where terrain hides the horizon and sunrise comes later. A height looked up from a DEM is a grid-cell average, not the ground under the observer.
- **Against JPL Horizons:** Horizons' "true visual horizon" uses the geometric dip `1.93′ × √h`. At Denver and La Paz this library rises up to about 30 s later and sets up to about 75 s earlier than Horizons TVH, and it agrees within Horizons' one-minute step at sea level. The package tests pin both.
