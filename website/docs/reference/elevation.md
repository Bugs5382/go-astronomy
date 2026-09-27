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
type Client struct{ /* unexported */ } // safe for concurrent use
type Option func(*Client)

const DefaultBaseURL = "https://api.open-meteo.com/v1/elevation"
const DefaultRound = 0.01

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

### Options

| option | default | meaning |
| --- | --- | --- |
| `WithHTTPClient(h)` | a client with a 10 s timeout | the `*http.Client` for requests |
| `WithBaseURL(u)` | `openmeteo.DefaultBaseURL` (`https://api.open-meteo.com/v1/elevation`) | a self-hosted Open-Meteo or a test server |
| `WithRound(deg)` | `openmeteo.DefaultRound` (0.01°, about 1.1 km) | the cache grid; the square root in the dip changes by under 1% across most 1 km cells |
| `WithLogger(l)` | none | a go-log `Logger` for debug and warn lines |

### Seeding from a durable cache

```go
client := openmeteo.New()
if h, ok := redisLookup(lat, lng); ok { // your own store
	client.Store(lat, lng, h)
}
h, err := client.Elevation(ctx, lat, lng) // served from the map, no request
if err == nil {
	redisSave(lat, lng, h) // elevation never changes: store it with no expiry
}
```

### Units and errors

Heights are metres above sea level; coordinates are degrees, latitude positive north and longitude positive east. `Fill` rejects a looked-up height outside −1000 m to 9000 m with `ErrLookupFailed`, so a bad reply never becomes a nonsense horizon.

### Writing your own Lookup

Any type with `Elevation(ctx, lat, lng) (float64, error)` works, for example a lookup against your own DEM tiles. Call `elevation.ValidateCoordinate` first and wrap failures with `elevation.LookupFailed(err)` to keep the coded errors consistent.
