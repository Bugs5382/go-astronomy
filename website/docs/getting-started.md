---
id: getting-started
title: Getting started
sidebar_position: 2
---

# 🚀 Getting started

## 📋 Requirements

- Go **`>= 1.27`**

## 📥 Install

```sh
go get github.com/Bugs5382/go-astronomy
```

Import the root package for the shared types plus the packages you need. The root package holds `Observer` and the position types; the `earth` package holds the Earth vantage on the Sun and the twilight bands.

## 🌅 A first example

Every call takes an `Observer` and a `time.Time`. Build the observer once and pass the instant on every call — nothing is captured at construction.

```go
package main

import (
	"fmt"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/earth/moon"
)

func main() {
	tz, err := time.LoadLocation("America/New_York")
	if err != nil {
		panic(err)
	}
	obs := astronomy.Observer{Lat: 40.678, Lng: -73.944, TZ: tz} // Brooklyn, NY
	t := time.Date(1982, 5, 3, 12, 0, 0, 0, tz)

	// Earth vantage: geometric alt/az of the Sun's disc center, plus its size.
	pos := earth.SunPosition(obs, t)
	fmt.Printf("sun alt=%.2f az=%.2f diameter=%.4f\n",
		pos.Altitude, pos.Azimuth, float64(pos.Diameter))

	// One civil day of twilight bands, resolved in obs.TZ.
	day, err := earth.NewSunTimes(obs, t)
	if err != nil {
		panic(err)
	}
	if noon, ok := day.SolarNoon(); ok {
		fmt.Println("solar noon:", noon.Format(time.Kitchen))
	}
	if state, isPolar := day.Polar(); isPolar {
		fmt.Println("polar state:", state) // midnight_sun or polar_night
	}
	for _, seg := range day.Segments() {
		fmt.Printf("  %-18s %s -> %s (%.0f min)\n",
			seg.Label,
			seg.From.Format(time.Kitchen),
			seg.To.Format(time.Kitchen),
			seg.Seconds/60)
	}

	// The Moon's phase and illuminated fraction at the same instant.
	fmt.Printf("moon: %s, %.0f%% lit\n",
		moon.PhaseAt(t), moon.Illumination(t)*100)
}
```

`SunPosition` never returns an error: an altitude below the horizon is a valid geometric answer. The functions that resolve a civil day (`NewSunTimes`) or the Moon's topocentric position validate the observer and return a coded error when the latitude or longitude is out of range — see [Errors](./reference/errors.md).


An `Observer` is a latitude and longitude in degrees (positive north and east), a time zone (nil means UTC), and an optional `Elevation` in metres above sea level (`astronomy.MinElevation` to `astronomy.MaxElevation`, −1000 to 9000). Leave `Elevation` at zero for sea-level answers. A height makes the Sun and Moon rise earlier and set later by the horizon dip; see [`earth`](./reference/earth.md) and [`elevation`](./reference/elevation.md).

## 🖥️ Projecting to a canvas

The library emits degrees. When you want pixels, hand the already-computed altitude/azimuth to the `project` package. It owns no ephemeris and no colors — only screen geometry.

```go
package main

import (
	"fmt"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/project"
)

func placeSun(obs astronomy.Observer, day *earth.SunTimes) {
	pos := earth.SunPosition(obs, someInstant)

	vp := project.Viewport{CanvasW: 1200, CanvasH: 400, HorizonFraction: 0.70}
	view := project.View{
		XMode:           project.TimeProgress,
		YMode:           project.NormalizedByPeak,
		PeakAltitudeDeg: 70,
	}
	obj := project.Object{
		AltitudeDeg:        pos.Altitude,
		AzimuthDeg:         pos.Azimuth,
		TimeProgress:       0.5, // sunrise-to-sunset fraction
		AngularDiameterDeg: float64(pos.Diameter),
	}

	out, err := project.Project(vp, view, obj)
	if err != nil {
		panic(err)
	}
	fmt.Printf("draw disc at (%.0f, %.0f), below horizon: %v\n",
		out.X, out.Y, out.BelowHorizon)
}
```

See [Concepts](./concepts.md) for the projection contract and [`project`](./reference/project.md) for every mode.

## 🛠️ Working in the repo

The repo uses a [`Taskfile`](https://taskfile.dev):

```sh
task build   # go build ./...
task test    # go test ./...
task lint    # gofmt check, golangci-lint, yamllint
task docs    # build this documentation site
```
