---
id: iss
title: iss
sidebar_position: 1
---

# 🛰️ Package `iss`

Import path: `github.com/Bugs5382/go-astronomy/satellite/iss`

ISS: the International Space Station. The package fixes the NORAD catalogue number and name, and wraps the [`satellite`](../satellite.md) engine. It never fetches anything itself: the caller passes an explicit element source, either its own TLE or OMM (`satellite.StaticElements`) or the [CelesTrak fetcher](./celestrak.md), which caches the set and hits the network at most every few hours. Every position and pass is then propagated locally with SGP4.

## 📦 API

```go
const (
	CatalogNumber = 25544
	Name          = "ISS (ZARYA)"
)

var StandardMagnitude float64 // satellite.ISSStandardMagnitude (−1.8, McCants)

func New(src satellite.ElementSource) *satellite.Tracker
```

The returned `*satellite.Tracker` gives:

```go
func (t *Tracker) Position(ctx context.Context, obs astronomy.Observer, at time.Time) (satellite.Look, error)
func (t *Tracker) Passes(ctx context.Context, obs astronomy.Observer, from, to time.Time) ([]satellite.Pass, error)
func (t *Tracker) PassesWith(ctx context.Context, obs astronomy.Observer, from, to time.Time, opt satellite.PassOptions) ([]satellite.Pass, error)
func (t *Tracker) Magnitude(l satellite.Look) float64
func (t *Tracker) Elements(ctx context.Context) (satellite.Elements, error)
func (t *Tracker) CatalogNumber() int
func (t *Tracker) Name() string
func (t *Tracker) StandardMagnitude() float64
```

- `Position` is the satellite as the observer sees it: altitude, azimuth, range, range rate, sunlight, the Sun's altitude for the observer, the phase angle, and the sub-satellite point (see [`satellite.Look`](../satellite.md)).
- `Passes` are intervals with rise, peak, and set, whether each is visible (the satellite sunlit while the observer is in darkness), and shadow entry and exit. `Passes` uses `satellite.DefaultPassOptions` with the standard magnitude, and `PassesWith` takes your own options.
- A source with no set for 25544 gives `satellite.ErrNoElements`. A stale set from the fetcher is still used, and the error wraps `satellite.ErrStaleElements`, so the caller can decide whether to trust it.
- Check `Elements.Age`: a low orbit's element set goes stale in days.

## 🚀 Examples

Each example is an `Example` test in `satellite/iss/example_test.go`; where an output is shown, it is what the test prints.

### New

The next passes over Denver, from an element set the caller already has.

```go
e := elements()
tracker := iss.New(satellite.StaticElements(e))
obs := astronomy.Observer{Lat: 39.74, Lng: -104.99}
from := time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)
passes, err := tracker.Passes(context.Background(), obs, from, from.Add(24*time.Hour))
if err != nil {
	panic(err)
}
fmt.Println("element set age:", e.Age(from).Round(time.Minute))
for _, p := range passes {
	fmt.Printf("%s to %s, peak %4.1f at %s, visible %v\n",
		p.Rise.Time.Format("15:04:05"), p.Set.Time.Format("15:04:05"),
		p.Peak.Altitude, p.Peak.Time.Format("15:04:05"), p.Visible)
}
```

```text
element set age: 3h34m0s
23:53:27 to 00:02:52, peak 13.1 at 23:58:10, visible false
01:30:21 to 01:41:05, peak 38.6 at 01:35:44, visible true
03:07:20 to 03:17:28, peak 23.2 at 03:12:25, visible true
18:13:46 to 18:23:40, peak 18.7 at 18:18:42, visible false
19:49:52 to 20:00:45, peak 47.6 at 19:55:18, visible false
21:27:59 to 21:37:34, peak 14.2 at 21:32:47, visible false
23:06:10 to 23:15:20, peak 11.7 at 23:10:46, visible false
```

### New_position

Where it is at one instant.

```go
tracker := iss.New(satellite.StaticElements(elements()))
obs := astronomy.Observer{Lat: 39.74, Lng: -104.99}
l, err := tracker.Position(context.Background(), obs, time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC))
if err != nil {
	panic(err)
}
fmt.Printf("altitude %.2f, azimuth %.2f, range %.0f km\n", l.Altitude, l.Azimuth, l.RangeKm)
fmt.Printf("over %.2f, %.2f at %.0f km, sunlit %v\n", l.Latitude, l.Longitude, l.AltitudeKm, l.Sunlit)
```

```text
altitude -35.80, azimuth 83.15, range 8155 km
over 13.80, -21.41 at 425 km, sunlit true
```

### New_celestrak

With the element set from CelesTrak instead: fetched once, cached for a day, and propagated locally. (Not run as a test, which never reaches the network.)

```go
tracker := iss.New(celestrak.New())
l, err := tracker.Position(context.Background(), astronomy.Observer{Lat: 39.74, Lng: -104.99}, time.Now())
if err != nil {
	fmt.Println("no element set:", err)
	return
}
fmt.Printf("altitude %.1f\n", l.Altitude)
```
