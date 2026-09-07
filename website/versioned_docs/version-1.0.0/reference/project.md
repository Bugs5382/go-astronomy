---
id: project
title: project
sidebar_position: 6
---

# 🖥️ Package `project`

Import path: `github.com/Bugs5382/go-astronomy/project`

Maps already-computed celestial positions, expressed in degrees, to canvas pixels. It is pure screen-projection geometry: it holds no colors and computes no ephemeris. A consumer supplies each object's altitude, azimuth, time-progress, and angular diameter, and receives pixel coordinates and a pixel disc size. All functions are stateless, deterministic, and safe for concurrent use.

## 🧭 Coordinate frame

The canvas origin is the top-left corner, x increasing right and y increasing down. The horizon is a line **inside** the canvas, not the bottom edge: altitude 0° maps to `y = HorizonFraction × CanvasH` (for example 0.70 of the height). Positive altitude moves up toward `y = 0`; a slightly-negative altitude dips below the horizon into the ground band and is flagged with `BelowHorizon`, yet still yields a `y` so a consumer can draw a setting body.

## ⚙️ Modes

```go
type YMode int
const (
	NormalizedByPeak YMode = iota // default: normalize altitude by a peak
	Geometric                     // fixed degrees-per-pixel elevation
)

type XMode int
const (
	TimeProgress XMode = iota // default: 0..1 sunrise-to-sunset across the width
	Azimuth                   // map the object's compass azimuth
)

type ViewMode int
const (
	FullArc ViewMode = iota // default: whole sunrise-to-sunset azimuth sweep
	Directional             // heading-centered slice of width FieldOfView
)
```

- **`YMode`** — `NormalizedByPeak` normalizes altitude by a supplied peak so the arch fills the sky band: `y = horizonY - (altitude/peak) × skyBandHeight`. `Geometric` uses a true fixed scale: `y = horizonY - altitude / ElevationDegPerPixel`.
- **`XMode`** — `TimeProgress` maps the 0..1 progress across the full width; there is no azimuth axis, so `DiameterPx` is 0 and `ColumnAzimuth` is not defined. `Azimuth` maps the compass azimuth per `ViewMode`.
- **`ViewMode`** (used when `XMode` is `Azimuth`) — `FullArc` spans the whole sunrise-to-sunset sweep across the width. `Directional` centers `ViewHeading` at the canvas middle and spans `FieldOfView`, marking an object outside the field not visible.

## 🪟 Inputs

```go
type Viewport struct {
	CanvasW         float64 // pixels, must be positive
	CanvasH         float64 // pixels, must be positive
	HorizonFraction float64 // horizon position from the top, [0, 1]
}

type View struct {
	XMode    XMode
	YMode    YMode
	ViewMode ViewMode

	PeakAltitudeDeg      float64 // top of the sky band under NormalizedByPeak; must be positive there
	ElevationDegPerPixel float64 // vertical scale under Geometric; must be positive there

	ViewHeading float64 // azimuth at the canvas middle under Directional
	FieldOfView float64 // angular width of the Directional slice, degrees; must be positive there

	SunriseAzimuth float64 // FullArc sweep bounds; sweep runs clockwise
	SunsetAzimuth  float64 // from SunriseAzimuth to SunsetAzimuth, span must be positive
}

type Object struct {
	AltitudeDeg        float64 // negative is below the horizon
	AzimuthDeg         float64 // compass azimuth, clockwise from true north
	TimeProgress       float64 // 0..1, used by the TimeProgress X mode
	AngularDiameterDeg float64 // apparent disc diameter, degrees
}
```

## 📤 Output

```go
type Projected struct {
	X            float64 // disc center, canvas pixels
	Y            float64
	DiameterPx   float64 // disc diameter in pixels; 0 in TimeProgress mode
	Visible      bool    // false only outside a Directional field of view
	BelowHorizon bool    // altitude is below the horizon
}
```

In azimuth-based modes `DiameterPx` scales the angular diameter by the same degrees-per-pixel the horizontal axis uses, so the disc is sized consistently with its placement.

## 🧮 Functions

```go
func Project(vp Viewport, v View, obj Object) (Projected, error)
func ColumnAzimuth(vp Viewport, v View, x float64) (float64, error)
func AngularSeparation(azA, azB float64) float64
```

- `Project` maps one object to pixels, validating the viewport and the view parameters the chosen modes require.
- `ColumnAzimuth` inverts the azimuth X-map: given a screen column it returns the azimuth being looked at. It returns a coded error when `XMode` is not `Azimuth` (there is no azimuth axis to invert).
- `AngularSeparation` returns the shortest angular difference between two azimuths, wrapping across the 0/360° seam.

Together the last two support direction-aware coloring without this package owning a palette: per column, compute `ColumnAzimuth`, then its `AngularSeparation` from the Sun's azimuth, and color as a function of that separation and the Sun's altitude.

## 🚑 Errors

Invalid inputs return `go-apperr` coded errors. Match the cause with `errors.Is` against the sentinels — `ErrInvalidCanvas`, `ErrInvalidHorizonFraction`, `ErrInvalidPeakAltitude`, `ErrInvalidFieldOfView`, `ErrInvalidElevationScale`, `ErrXModeNotAzimuth` — or recover the stable code with `apperr.Code`. See [Errors](./errors.md).

Full godoc: [pkg.go.dev/github.com/Bugs5382/go-astronomy/project](https://pkg.go.dev/github.com/Bugs5382/go-astronomy/project).
