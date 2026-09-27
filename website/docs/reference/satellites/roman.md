---
id: roman
title: roman
sidebar_position: 7
---

# 🔭 Package `roman`

Import path: `github.com/Bugs5382/go-astronomy/satellite/roman`

The Nancy Grace Roman Space Telescope, from JPL Horizons. Horizons lists it as -211 ("Roman Space Telescope (spacecraft)", alias "RST Nancy Grace"). It launched on 2026-08-30 and is on its way to L2. Horizons carries only its published predicted trajectory, which ran to 2026-10-19 when this package was written; an instant past the end gives `horizons.ErrOutsideCoverage`. The coverage grows as new predictions are published.

## 📦 API

```go
const (
	Target = "-211" // the Horizons command
	Name   = "Nancy Grace Roman Space Telescope"
)

func New(c *horizons.Client) *horizons.Tracker
```

`New(horizons.New()).Position(ctx, obs, t)` gives the topocentric altitude, azimuth, RA and Dec of date, and range (see [`horizons`](./horizons.md)). The client fetches a 30-day table once and interpolates locally.

## ❔ Why there is no Passes, and no SGP4

- **No TLE or SGP4.** Two-line element sets and SGP4 do not apply at L2. SGP4 models a satellite in Earth orbit, perturbed by the Earth's shape and drag, while an L2 telescope circles a point beyond the Moon under the Sun's and the Earth's gravity. The ephemeris comes from Horizons instead.
- **No passes.** From over a million km the telescope drifts across the stars by about a degree a day, near the point opposite the Sun. It rises and sets once a day with the sky, like a faint star, so there are no minutes-long passes to predict. It is also far too faint to see without a large telescope.

## 🚀 Examples

Each example is an `Example` test in `satellite/roman/example_test.go`; where an output is shown, it is what the test prints.

### New

Where the telescope is in Sydney's sky, from a stand-in serving the recorded Horizons window. In production, pass horizons.New().

```go
api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	b, _ := os.ReadFile("testdata/roman-window-full.txt")
	_, _ = w.Write(b)
}))
defer api.Close()

tracker := roman.New(horizons.New(horizons.WithBaseURL(api.URL)))
sydney := astronomy.Observer{Lat: -33.87, Lng: 151.21}
p, err := tracker.Position(context.Background(), sydney, time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC))
if err != nil {
	panic(err)
}
fmt.Printf("%s: altitude %.2f, azimuth %.2f, %.0f km away\n", tracker.Name(), p.Altitude, p.Azimuth, p.RangeKm)
```

```text
Nancy Grace Roman Space Telescope: altitude 69.12, azimuth 38.77, 1232376 km away
```
