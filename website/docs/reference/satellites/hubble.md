---
id: hubble
title: hubble
sidebar_position: 2
---

# 🛰️ Package `hubble`

Import path: `github.com/Bugs5382/go-astronomy/satellite/hubble`

Hubble: the Hubble Space Telescope. The package fixes the NORAD catalogue number and name, and wraps the [`satellite`](../satellite.md) engine. It never fetches anything itself: the caller passes an explicit element source, either its own TLE or OMM (`satellite.StaticElements`) or the [CelesTrak fetcher](./celestrak.md), which caches the set and hits the network at most every few hours. Every position and pass is then propagated locally with SGP4.

## 📦 API

```go
const (
	CatalogNumber = 20580
	Name          = "HST"
)

var StandardMagnitude float64 // 2.2 (McCants; approximate, since Hubble's brightness depends on its attitude)

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
- A source with no set for 20580 gives `satellite.ErrNoElements`. A stale set from the fetcher is still used, and the error wraps `satellite.ErrStaleElements`, so the caller can decide whether to trust it.
- Check `Elements.Age`: a low orbit's element set goes stale in days.

## 🚀 Examples

Each example is an `Example` test in `satellite/hubble/example_test.go`; where an output is shown, it is what the test prints.

### New

The next passes over Nairobi, from an element set the caller already has.

```go
e := elements()
tracker := hubble.New(satellite.StaticElements(e))
obs := astronomy.Observer{Lat: -1.29, Lng: 36.82}
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
element set age: 18h32m0s
01:34:25 to 01:42:52, peak  7.3 at 01:38:39, visible true
03:12:38 to 03:24:15, peak 43.4 at 03:18:27, visible false
04:52:31 to 05:03:30, peak 22.8 at 04:58:01, visible false
06:35:15 to 06:40:38, peak  2.3 at 06:37:56, visible false
13:19:11 to 13:25:52, peak  3.9 at 13:22:31, visible false
14:56:44 to 15:08:00, peak 29.2 at 15:02:22, visible false
16:36:14 to 16:47:38, peak 32.8 at 16:41:55, visible true
18:18:01 to 18:25:36, peak  5.4 at 18:21:48, visible false
```

### New_position

Where it is at one instant.

```go
tracker := hubble.New(satellite.StaticElements(elements()))
obs := astronomy.Observer{Lat: -1.29, Lng: 36.82}
l, err := tracker.Position(context.Background(), obs, time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC))
if err != nil {
	panic(err)
}
fmt.Printf("altitude %.2f, azimuth %.2f, range %.0f km\n", l.Altitude, l.Azimuth, l.RangeKm)
fmt.Printf("over %.2f, %.2f at %.0f km, sunlit %v\n", l.Latitude, l.Longitude, l.AltitudeKm, l.Sunlit)
```

```text
altitude -30.76, azimuth 95.28, range 7366 km
over -5.40, 104.43 at 469 km, sunlit true
```

### New_celestrak

With the element set from CelesTrak instead: fetched once, cached for a day, and propagated locally. (Not run as a test, which never reaches the network.)

```go
tracker := hubble.New(celestrak.New())
l, err := tracker.Position(context.Background(), astronomy.Observer{Lat: -1.29, Lng: 36.82}, time.Now())
if err != nil {
	fmt.Println("no element set:", err)
	return
}
fmt.Printf("altitude %.1f\n", l.Altitude)
```
