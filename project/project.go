// Package project maps already-computed celestial positions, expressed in
// degrees, to canvas pixels. It is pure screen-projection geometry: it holds no
// colors and computes no ephemeris. A consumer such as the header/sky service
// supplies each object's altitude, azimuth, time-progress, and angular diameter,
// and receives pixel coordinates and a pixel disc size.
//
// # Coordinate frame
//
// The canvas origin is the top-left corner, x increasing right and y increasing
// down. The horizon is the ground-top line inside the canvas, not the bottom
// edge: altitude 0 maps to y = HorizonFraction * CanvasH (for example 0.70 of
// the height). Positive altitude moves up toward y=0; a slightly-negative
// altitude dips below the horizon into the ground band and is flagged with
// BelowHorizon, yet still yields a y so a consumer can draw a setting body.
//
// # Vertical mode (YMode)
//
//   - NormalizedByPeak (default) normalizes altitude by a supplied peak altitude
//     so the arch fills the sky band, preserving the header's arch shape:
//     y = horizonY - (altitude/peak) * skyBandHeight, where skyBandHeight is the
//     sky region above the horizon (horizonY pixels tall). An object at the peak
//     altitude reaches the top of the band.
//   - Geometric uses a true fixed degrees-per-pixel elevation:
//     y = horizonY - altitude / ElevationDegPerPixel.
//
// # Horizontal mode (XMode)
//
//   - TimeProgress maps the 0..1 sunrise-to-sunset progress across the full
//     width: x = timeProgress * CanvasW. There is no azimuth axis in this mode,
//     so DiameterPx is 0 (the consumer sizes the disc itself) and ColumnAzimuth
//     is not defined.
//   - Azimuth maps the object's compass azimuth; the exact mapping is chosen by
//     ViewMode.
//
// # Azimuth view mode (ViewMode)
//
//   - FullArc (default) spans the whole sunrise-to-sunset azimuth sweep across
//     the full width, so every object between sunrise and sunset is visible. The
//     sweep runs clockwise from SunriseAzimuth to SunsetAzimuth.
//   - Directional shows a heading-centered slice: ViewHeading sits at the canvas
//     middle and FieldOfView spans the width. An object is visible only when its
//     signed angular difference from the heading is within half the field of
//     view; otherwise it is behind the viewer or off to the side and is marked
//     not visible.
//
// # Disc sizing
//
// DiameterPx scales the angular diameter by the same degrees-per-pixel the
// horizontal axis uses (FieldOfView/CanvasW for Directional, the arc span over
// CanvasW for FullArc), so a disc is sized consistently with its horizontal
// placement in azimuth-based modes.
//
// # Direction-aware coloring support
//
// The package provides two helpers so a consumer can drive direction-aware
// coloring (for example warm hues near the Sun and darkening toward the
// anti-solar point) without this package owning any palette. ColumnAzimuth is
// the inverse of the azimuth X map: given a screen column it returns the azimuth
// being looked at. AngularSeparation returns the shortest angular difference
// between two azimuths, wrapping across the 0/360 seam. Per screen column a
// consumer computes ColumnAzimuth, then its AngularSeparation from the Sun's
// azimuth, and colors as a function of that separation and the Sun's altitude.
//
// All functions are stateless, deterministic, and safe for concurrent use.
// Invalid inputs return go-apperr coded errors; recover the code with
// apperr.Code or match the cause with errors.Is against the Err... sentinels.
package project

/*
MIT License

Copyright (c) 2026 Shane

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE USE OR PERFORMANCE OF THIS SOFTWARE.
*/

import (
	"errors"
	"math"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/internal/angles"
)

// YMode selects the vertical (altitude-to-y) projection strategy.
type YMode int

const (
	// NormalizedByPeak normalizes altitude by a supplied peak so the arch fills
	// the sky band. It is the zero value and the default.
	NormalizedByPeak YMode = iota
	// Geometric uses a fixed degrees-per-pixel elevation.
	Geometric
)

// XMode selects the horizontal (x) projection strategy.
type XMode int

const (
	// TimeProgress maps the 0..1 sunrise-to-sunset progress across the width. It
	// is the zero value and the default.
	TimeProgress XMode = iota
	// Azimuth maps the object's compass azimuth per ViewMode.
	Azimuth
)

// ViewMode selects how azimuth is mapped when XMode is Azimuth.
type ViewMode int

const (
	// FullArc spans the whole sunrise-to-sunset azimuth sweep across the width.
	// It is the zero value and the default.
	FullArc ViewMode = iota
	// Directional shows a heading-centered slice of width FieldOfView.
	Directional
)

// Viewport is the pixel canvas and the placement of the horizon within it.
type Viewport struct {
	// CanvasW and CanvasH are the canvas width and height in pixels; both must be
	// positive.
	CanvasW float64
	CanvasH float64
	// HorizonFraction is the fraction of the height at which the horizon
	// (altitude 0) sits, measured from the top. It must be in [0, 1]; a value
	// like 0.70 leaves a ground band below the horizon.
	HorizonFraction float64
}

// View is the projection configuration: the horizontal and vertical strategies
// plus the parameters each strategy needs.
type View struct {
	// XMode selects the horizontal strategy (default TimeProgress).
	XMode XMode
	// YMode selects the vertical strategy (default NormalizedByPeak).
	YMode YMode
	// ViewMode selects the azimuth mapping when XMode is Azimuth (default
	// FullArc).
	ViewMode ViewMode

	// PeakAltitudeDeg is the altitude that reaches the top of the sky band under
	// NormalizedByPeak; it must be positive in that mode.
	PeakAltitudeDeg float64
	// ElevationDegPerPixel is the fixed vertical scale under Geometric; it must be
	// positive in that mode.
	ElevationDegPerPixel float64

	// ViewHeading is the compass azimuth placed at the canvas middle under
	// Directional.
	ViewHeading float64
	// FieldOfView is the angular width of the Directional slice in degrees; it
	// must be positive in that mode.
	FieldOfView float64

	// SunriseAzimuth and SunsetAzimuth bound the FullArc sweep. The sweep runs
	// clockwise from SunriseAzimuth to SunsetAzimuth and its span must be
	// positive.
	SunriseAzimuth float64
	SunsetAzimuth  float64
}

// Object is a single celestial body's already-computed position and size.
type Object struct {
	// AltitudeDeg is the altitude above the horizon in degrees; negative is below.
	AltitudeDeg float64
	// AzimuthDeg is the compass azimuth in degrees, clockwise from true north.
	AzimuthDeg float64
	// TimeProgress is the 0..1 progress from sunrise to sunset, used by the
	// TimeProgress horizontal mode.
	TimeProgress float64
	// AngularDiameterDeg is the apparent disc diameter in degrees.
	AngularDiameterDeg float64
}

// Projected is the pixel-space result for one object.
type Projected struct {
	// X and Y are the disc center in canvas pixels.
	X float64
	Y float64
	// DiameterPx is the disc diameter in pixels in azimuth-based modes; it is 0
	// in TimeProgress mode, which has no angular horizontal scale.
	DiameterPx float64
	// Visible reports whether the object falls within the view. It is always true
	// in TimeProgress and FullArc modes and depends on the field of view in
	// Directional mode.
	Visible bool
	// BelowHorizon reports whether the object's altitude is below the horizon.
	BelowHorizon bool
}

// Sentinel causes wrapped by the package. Each is returned inside a go-apperr
// coded error, so errors.Is keeps matching these values while apperr.Code
// recovers the stable numeric code from the astronomy package's registry.
var (
	// ErrInvalidCanvas is the cause when the canvas width or height is not
	// positive; the coded error carries astronomy.CodeInvalidCanvas.
	ErrInvalidCanvas = errors.New("project: canvas width and height must be positive")
	// ErrInvalidHorizonFraction is the cause when the horizon fraction is outside
	// [0, 1]; the coded error carries astronomy.CodeInvalidHorizonFraction.
	ErrInvalidHorizonFraction = errors.New("project: horizon fraction must be within [0, 1]")
	// ErrInvalidPeakAltitude is the cause when a NormalizedByPeak projection has a
	// non-positive peak altitude; the coded error carries
	// astronomy.CodeInvalidPeakAltitude.
	ErrInvalidPeakAltitude = errors.New("project: normalized peak altitude must be positive")
	// ErrInvalidFieldOfView is the cause when a Directional field of view or a
	// FullArc azimuth span is not positive; the coded error carries
	// astronomy.CodeInvalidFieldOfView.
	ErrInvalidFieldOfView = errors.New("project: azimuth field of view or arc span must be positive")
	// ErrInvalidElevationScale is the cause when a Geometric projection has a
	// non-positive elevation scale; the coded error carries
	// astronomy.CodeInvalidElevationScale.
	ErrInvalidElevationScale = errors.New("project: geometric elevation degrees-per-pixel must be positive")
	// ErrXModeNotAzimuth is the cause when ColumnAzimuth is called on a projection
	// whose horizontal mode is not azimuth-based; the coded error carries
	// astronomy.CodeXModeNotAzimuth.
	ErrXModeNotAzimuth = errors.New("project: column-to-azimuth query requires the Azimuth horizontal mode")
)

// validateViewport checks that the canvas spans pixels and the horizon line
// falls within it.
func validateViewport(vp Viewport) error {
	if vp.CanvasW <= 0 || vp.CanvasH <= 0 {
		return apperr.Coded(astronomy.CodeInvalidCanvas, ErrInvalidCanvas)
	}
	if vp.HorizonFraction < 0 || vp.HorizonFraction > 1 {
		return apperr.Coded(astronomy.CodeInvalidHorizonFraction, ErrInvalidHorizonFraction)
	}
	return nil
}

// horizonPixel returns the y of the horizon (altitude 0) line.
func horizonPixel(vp Viewport) float64 { return vp.HorizonFraction * vp.CanvasH }

// projectY maps altitude to a y pixel per the vertical mode. It assumes the view
// has already been validated.
func projectY(vp Viewport, v View, altitudeDeg float64) float64 {
	horizonY := horizonPixel(vp)
	if v.YMode == Geometric {
		return horizonY - altitudeDeg/v.ElevationDegPerPixel
	}
	// NormalizedByPeak: the sky band above the horizon is horizonY pixels tall.
	return horizonY - (altitudeDeg/v.PeakAltitudeDeg)*horizonY
}

// signedDiff returns the signed shortest angular difference a-b in degrees,
// within (-180, 180], so it handles the 0/360 seam.
func signedDiff(a, b float64) float64 {
	d := angles.Normalize(a - b)
	if d > 180 {
		d -= 360
	}
	return d
}

// arcSpan returns the clockwise azimuth span of the FullArc sweep.
func arcSpan(v View) float64 { return angles.Normalize(v.SunsetAzimuth - v.SunriseAzimuth) }

// azimuthDegPerPixel returns the horizontal degrees-per-pixel for an azimuth
// view, and validates its defining parameter. It assumes XMode is Azimuth.
func azimuthDegPerPixel(vp Viewport, v View) (float64, error) {
	switch v.ViewMode {
	case Directional:
		if v.FieldOfView <= 0 {
			return 0, apperr.Coded(astronomy.CodeInvalidFieldOfView, ErrInvalidFieldOfView)
		}
		return v.FieldOfView / vp.CanvasW, nil
	default: // FullArc
		span := arcSpan(v)
		if span <= 0 {
			return 0, apperr.Coded(astronomy.CodeInvalidFieldOfView, ErrInvalidFieldOfView)
		}
		return span / vp.CanvasW, nil
	}
}

// validateY checks the parameter the active vertical mode needs.
func validateY(v View) error {
	switch v.YMode {
	case Geometric:
		if v.ElevationDegPerPixel <= 0 {
			return apperr.Coded(astronomy.CodeInvalidElevationScale, ErrInvalidElevationScale)
		}
	default: // NormalizedByPeak
		if v.PeakAltitudeDeg <= 0 {
			return apperr.Coded(astronomy.CodeInvalidPeakAltitude, ErrInvalidPeakAltitude)
		}
	}
	return nil
}

// Project maps one object to pixel space under the given viewport and view. It
// returns a go-apperr coded error for any invalid input: a non-positive canvas
// (CodeInvalidCanvas), an out-of-range horizon fraction
// (CodeInvalidHorizonFraction), a non-positive normalized peak
// (CodeInvalidPeakAltitude), a non-positive geometric elevation scale
// (CodeInvalidElevationScale), or a non-positive azimuth field of view or arc
// span (CodeInvalidFieldOfView).
func Project(vp Viewport, v View, obj Object) (Projected, error) {
	if err := validateViewport(vp); err != nil {
		return Projected{}, err
	}
	if err := validateY(v); err != nil {
		return Projected{}, err
	}

	out := Projected{
		Y:            projectY(vp, v, obj.AltitudeDeg),
		BelowHorizon: obj.AltitudeDeg < 0,
	}

	if v.XMode == TimeProgress {
		out.X = obj.TimeProgress * vp.CanvasW
		out.Visible = true
		// No angular horizontal scale exists, so the disc is not sized here.
		return out, nil
	}

	degPerPixel, err := azimuthDegPerPixel(vp, v)
	if err != nil {
		return Projected{}, err
	}
	out.DiameterPx = obj.AngularDiameterDeg / degPerPixel

	if v.ViewMode == Directional {
		diff := signedDiff(obj.AzimuthDeg, v.ViewHeading)
		out.X = vp.CanvasW/2 + diff/degPerPixel
		out.Visible = math.Abs(diff) <= v.FieldOfView/2
		return out, nil
	}

	// FullArc: the whole sweep is on screen, so everything is visible.
	from := angles.Normalize(obj.AzimuthDeg - v.SunriseAzimuth)
	out.X = from / degPerPixel
	out.Visible = true
	return out, nil
}

// ColumnAzimuth is the inverse of the azimuth X map: given a screen column x it
// returns the compass azimuth, in [0, 360), that the column looks at. It applies
// only to the Azimuth horizontal mode; called on TimeProgress it returns a coded
// error carrying astronomy.CodeXModeNotAzimuth. It also validates the viewport
// and the active azimuth parameter the same way Project does.
func ColumnAzimuth(vp Viewport, v View, x float64) (float64, error) {
	if err := validateViewport(vp); err != nil {
		return 0, err
	}
	if v.XMode != Azimuth {
		return 0, apperr.Coded(astronomy.CodeXModeNotAzimuth, ErrXModeNotAzimuth)
	}
	degPerPixel, err := azimuthDegPerPixel(vp, v)
	if err != nil {
		return 0, err
	}
	if v.ViewMode == Directional {
		return angles.Normalize(v.ViewHeading + (x-vp.CanvasW/2)*degPerPixel), nil
	}
	// FullArc: x=0 is the sunrise azimuth, x=CanvasW the sunset azimuth.
	return angles.Normalize(v.SunriseAzimuth + x*degPerPixel), nil
}

// AngularSeparation returns the shortest angular separation between two azimuths
// in degrees, in [0, 180]. It wraps across the 0/360 seam, so the separation of
// 350 and 10 is 20. A consumer uses it with ColumnAzimuth to find each screen
// column's angular distance from the Sun for direction-aware coloring.
func AngularSeparation(azA, azB float64) float64 {
	return math.Abs(signedDiff(azA, azB))
}
