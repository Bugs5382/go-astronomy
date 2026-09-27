---
id: elevation
title: elevation
sidebar_position: 8
---

# ⛰️ Packages `elevation` and `elevation/openmeteo`

Import paths: `github.com/Bugs5382/go-astronomy/elevation` and `github.com/Bugs5382/go-astronomy/elevation/openmeteo`

The core packages never reach the network. A caller that already knows its height sets `astronomy.Observer.Elevation` and imports nothing from here. A caller that wants the height looked up from a coordinate uses a `Lookup`, and `Fill` copies the result into the observer. The height feeds the horizon dip described in [`earth`](./earth.md).

## 🔌 The Lookup interface

```go
type Lookup interface {
	Elevation(ctx context.Context, lat, lng float64) (float64, error)
}

func Fill(ctx context.Context, l Lookup, obs astronomy.Observer) (astronomy.Observer, error)
func ValidateCoordinate(lat, lng float64) error
func LookupFailed(cause error) error

var ErrInvalidLatitude, ErrInvalidLongitude, ErrLookupFailed error
```

`Fill` validates the coordinate, asks the lookup, rejects a height outside the range the observer accepts, and returns the observer with `Elevation` set. A lookup failure carries code 7012 (`CodeElevationLookup`) and wraps the underlying cause, so `errors.Is` matches both `ErrLookupFailed` and, for example, `context.DeadlineExceeded`.

## 🌐 Open-Meteo adapter

```go
func New(opts ...Option) *Client
func WithHTTPClient(h *http.Client) Option
func WithBaseURL(u string) Option
func WithRound(deg float64) Option  // cache grid, default 0.01 degree
func WithLogger(l log.Logger) Option // go-log; silent without it

func (c *Client) Elevation(ctx context.Context, lat, lng float64) (float64, error)
func (c *Client) Cached(lat, lng float64) (float64, bool)
func (c *Client) Store(lat, lng, elevationM float64)
```

`openmeteo.Client` looks elevations up from the [Open-Meteo elevation API](https://open-meteo.com/en/docs/elevation-api) (Copernicus DEM, 90 m, no key) with the standard library's `net/http` only. Elevation does not change, so every value is kept in an in-process map with no expiry, keyed by the coordinate rounded to a grid (0.01°, about 1.1 km, by default). The query is made at the cell's rounded coordinate, so every point in a cell shares one value. A durable cache across processes is the consumer's: read the map with `Cached` and seed it with `Store`.

```go
c := openmeteo.New()
obs, err := elevation.Fill(ctx, c, astronomy.Observer{Lat: 39.74, Lng: -104.99, TZ: tz})
if err != nil {
	// fall back to sea level, or report it
}
day, err := earth.NewSunTimes(obs, time.Now())
```

The client logs nothing unless given a go-log `Logger` with `WithLogger`, in which case it logs cache hits, misses, requests (target, status, duration), and failures.
