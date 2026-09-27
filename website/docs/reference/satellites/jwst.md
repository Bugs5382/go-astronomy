---
id: jwst
title: jwst
sidebar_position: 6
---

# 🔭 Package `jwst`

Import path: `github.com/Bugs5382/go-astronomy/satellite/jwst`

The James Webb Space Telescope, from JPL Horizons. It launched on 2021-12-25 and circles the Sun-Earth L2 point in a halo orbit, 1.2 to 1.8 million km from Earth. Horizons carries its reconstructed trajectory and five years of predictions.

## 📦 API

```go
const (
	Target = "-170" // the Horizons command
	Name   = "James Webb Space Telescope"
)

func New(c *horizons.Client) *horizons.Tracker
```

`New(horizons.New()).Position(ctx, obs, t)` gives the topocentric altitude, azimuth, RA and Dec of date, and range (see [`horizons`](./horizons.md)). The client fetches a 30-day table once and interpolates locally.

## ❔ Why there is no Passes, and no SGP4

- **No TLE or SGP4.** Two-line element sets and SGP4 do not apply at L2. SGP4 models a satellite in Earth orbit, perturbed by the Earth's shape and drag, while an L2 telescope circles a point beyond the Moon under the Sun's and the Earth's gravity. The ephemeris comes from Horizons instead.
- **No passes.** From over a million km the telescope drifts across the stars by about a degree a day, near the point opposite the Sun. It rises and sets once a day with the sky, like a faint star, so there are no minutes-long passes to predict. It is also far too faint to see without a large telescope.

## 🚀 Examples

Each example is an `Example` test in `satellite/jwst/example_test.go`; where an output is shown, it is what the test prints.

### New

Where the telescope is in Sydney's sky, from a stand-in serving the recorded Horizons window. In production, pass horizons.New().

```go
api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	b, _ := os.ReadFile("testdata/jwst-window.txt")
	_, _ = w.Write(b)
}))
defer api.Close()

tracker := jwst.New(horizons.New(horizons.WithBaseURL(api.URL)))
sydney := astronomy.Observer{Lat: -33.87, Lng: 151.21}
p, err := tracker.Position(context.Background(), sydney, time.Date(2026, 9, 20, 12, 0, 0, 0, time.UTC))
if err != nil {
	panic(err)
}
fmt.Printf("%s: altitude %.2f, azimuth %.2f, %.0f km away\n", tracker.Name(), p.Altitude, p.Azimuth, p.RangeKm)
```

```text
James Webb Space Telescope: altitude 34.58, azimuth 35.30, 1223891 km away
```
