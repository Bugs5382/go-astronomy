---
id: tiangong
title: tiangong
sidebar_position: 3
---

# 🛰️ Package `tiangong`

Import path: `github.com/Bugs5382/go-astronomy/satellite/tiangong`

Tiangong: the Tiangong space station, whose catalogue entry is its Tianhe core module. The package fixes the NORAD catalogue number and name, and wraps the [`satellite`](../satellite.md) engine. It never fetches anything itself: the caller passes an explicit element source, either its own TLE or OMM (`satellite.StaticElements`) or the [CelesTrak fetcher](./celestrak.md), which caches the set and hits the network at most every few hours. Every position and pass is then propagated locally with SGP4.

## 📦 API

```go
const (
	CatalogNumber = 48274
	Name          = "CSS (TIANHE)"
)

var StandardMagnitude float64 // NaN: no standard magnitude is used, so magnitudes are left out rather than guessed

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
- A source with no set for 48274 gives `satellite.ErrNoElements`. A set too old for SGP4 to propagate gives the propagation error and no result, never a wrong position.
- Every `Look` and `Pass` carries `ElementEpoch`, the epoch of the set it was computed from, so a caller can see how old the data is. The CelesTrak fetcher keeps its set fresh in the background (see [celestrak](./celestrak.md)).

## 🚀 Examples

Each example is an `Example` test in `satellite/tiangong/example_test.go`; where an output is shown, it is what the test prints.

### New

The next passes over Madrid, from an element set the caller already has.

```go
e := elements()
tracker := tiangong.New(satellite.StaticElements(e))
obs := astronomy.Observer{Lat: 40.42, Lng: -3.70}
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
element set age: 12h13m0s
06:22:16 to 06:31:07, peak 12.7 at 06:26:41, visible false
07:57:55 to 08:08:20, peak 54.7 at 08:03:07, visible false
09:34:45 to 09:45:16, peak 70.9 at 09:40:00, visible false
11:11:41 to 11:22:12, peak 78.0 at 11:16:57, visible false
12:48:42 to 12:58:23, peak 20.6 at 12:53:33, visible false
14:27:41 to 14:31:32, peak  1.4 at 14:29:36, visible false
```

### New_position

Where it is at one instant.

```go
tracker := tiangong.New(satellite.StaticElements(elements()))
obs := astronomy.Observer{Lat: 40.42, Lng: -3.70}
l, err := tracker.Position(context.Background(), obs, time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC))
if err != nil {
	panic(err)
}
fmt.Printf("altitude %.2f, azimuth %.2f, range %.0f km\n", l.Altitude, l.Azimuth, l.RangeKm)
fmt.Printf("over %.2f, %.2f at %.0f km, sunlit %v\n", l.Latitude, l.Longitude, l.AltitudeKm, l.Sunlit)
```

```text
altitude -79.63, azimuth 100.87, range 12933 km
over -41.51, 149.55 at 398 km, sunlit false
```

### New_celestrak

With the element set from CelesTrak instead: fetched once, cached for a day, and propagated locally. (Not run as a test, which never reaches the network.)

```go
tracker := tiangong.New(celestrak.New())
l, err := tracker.Position(context.Background(), astronomy.Observer{Lat: 40.42, Lng: -3.70}, time.Now())
if err != nil {
	fmt.Println("no element set:", err)
	return
}
fmt.Printf("altitude %.1f\n", l.Altitude)
```
