---
id: horizons
title: horizons
sidebar_position: 5
---

# 🔭 Package `horizons`

Import path: `github.com/Bugs5382/go-astronomy/satellite/horizons`

An explicit ephemeris fetcher for the JPL Horizons API (`https://ssd.jpl.nasa.gov/api/horizons.api`), for objects SGP4 and element sets cannot describe, such as the telescopes at the Sun-Earth L2 point ([`jwst`](./jwst.md), [`roman`](./roman.md)).

## 📦 API

```go
const (
	DefaultBaseURL = "https://ssd.jpl.nasa.gov/api/horizons.api"
	DefaultWindow  = 30 * 24 * time.Hour
	DefaultStep    = time.Hour
	DefaultTimeout = 60 * time.Second
)

var ErrFetch, ErrOutsideCoverage, ErrInvalidLatitude, ErrInvalidLongitude error

type Geocentric struct {
	RA, Dec    float64 // apparent, of date, degrees
	DistanceKm float64
}

type Position struct {
	astronomy.Horizontal // airless altitude and azimuth, degrees
	RA, Dec              float64 // topocentric apparent, of date, degrees
	RangeKm              float64
}

func New(opts ...Option) *Client
func (c *Client) Geocentric(ctx context.Context, target string, t time.Time) (Geocentric, error)
func (c *Client) Position(ctx context.Context, target string, obs astronomy.Observer, t time.Time) (Position, error)

func NewTracker(target, name string, c *Client) *Tracker
func (t *Tracker) Position(ctx context.Context, obs astronomy.Observer, at time.Time) (Position, error)
func (t *Tracker) Geocentric(ctx context.Context, at time.Time) (Geocentric, error)
func (t *Tracker) Target() string
func (t *Tracker) Name() string

func WithHTTPClient(c *http.Client) Option
func WithBaseURL(u string) Option
func WithTimeout(d time.Duration) Option
func WithWindow(d time.Duration) Option  // span of one request, default 30 days
func WithStep(d time.Duration) Option    // table step, whole minutes, default 1 hour
func WithCache(c satellite.Cache) Option
func WithLogger(l log.Logger) Option
```

## 🗄️ One request per window, interpolated locally

- **One table per window.** A request fetches the target's geocentric apparent place (Horizons quantities 2 and 20) over a whole window: 30 days at a one-hour step by default, padded four steps on each side. The table is cached under the target, window, and step.
- **Local interpolation.** Positions inside the window are interpolated locally with an eight-point Lagrange polynomial. The right ascension is unwrapped across 0/360°.
- **Refetch rule.** A new request is made only when an instant falls outside the cached window. Windows are aligned to whole multiples of the window length from the Unix epoch, so every instant in one maps to the same request and cache entry.
- **Observers share a fetch.** The window is observer-independent. `Position` applies the rigorous topocentric correction (Meeus chapter 40, WGS84) and apparent sidereal time locally, so every observer shares one fetch.
- **Coverage edges.** When the trajectory ends (or starts) inside a window, Horizons says so ("No ephemeris for target ... after A.D. ..."). The client then clips the window one step inside that edge and asks once more. An instant beyond the trajectory gives `ErrOutsideCoverage`, never an extrapolation.
- **Cache and logging.** The cache is pluggable (`satellite.Cache`), the same as CelesTrak's. The client logs through go-log.

## 🎯 Accuracy

- **Interpolation:** interpolating the hourly JWST table at every point of a ten-minute Horizons table (2026-10-01 to 10-03) is within 4.4 milliarcseconds and 0.7 m.
- **Topocentric correction:** the local correction agrees with Horizons' own topocentric places for Greenwich to 0.2″ in RA and Dec, 0.26″ in altitude and azimuth, and 0.6 km in range.
- **Time scale:** UT1 is taken as UTC, as elsewhere in the library.

## 🚀 Examples

Each example is an `Example` test in `satellite/horizons/example_test.go`; where an output is shown, it is what the test prints.

### Client_Position

The James Webb Space Telescope from Greenwich: one request for the window, then everything is interpolated locally.

```go
api := standIn()
defer api.Close()
c := horizons.New(horizons.WithBaseURL(api.URL))
greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005}
p, err := c.Position(context.Background(), "-170", greenwich, time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC))
if err != nil {
	panic(err)
}
fmt.Printf("altitude %.4f, azimuth %.4f\n", p.Altitude, p.Azimuth)
fmt.Printf("RA %.5f, Dec %.5f, range %.0f km\n", p.RA, p.Dec, p.RangeKm)
```

```text
altitude 56.5685, azimuth 156.4667
RA 23.26297, Dec 19.77391, range 1294229 km
```

### Client_Geocentric

The geocentric place, shared by every observer.

```go
api := standIn()
defer api.Close()
c := horizons.New(horizons.WithBaseURL(api.URL))
g, err := c.Geocentric(context.Background(), "-170", time.Date(2026, 9, 20, 12, 30, 0, 0, time.UTC))
if err != nil {
	panic(err)
}
fmt.Printf("RA %.5f, Dec %.5f, %.0f km\n", g.RA, g.Dec, g.DistanceKm)
```

```text
RA 359.88542, Dec 13.78232, 1227623 km
```
