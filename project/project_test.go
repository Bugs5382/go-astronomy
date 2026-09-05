package project_test

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
	"testing"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/project"
)

const eps = 1e-9

// stdViewport is the reference canvas shared by most tests: 1000x500 pixels with
// the horizon 70% down, so the ground-top line sits at y=350 and the sky band
// above it is 350 pixels tall.
var stdViewport = project.Viewport{CanvasW: 1000, CanvasH: 500, HorizonFraction: 0.7}

func approx(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("%s = %v, want %v", name, got, want)
	}
}

func mustProject(t *testing.T, vp project.Viewport, v project.View, obj project.Object) project.Projected {
	t.Helper()
	p, err := project.Project(vp, v, obj)
	if err != nil {
		t.Fatalf("Project: unexpected error: %v", err)
	}
	return p
}

// TestHorizonAnchoring checks that altitude 0 lands on the horizon line for both
// vertical modes, independent of the horizontal strategy.
func TestHorizonAnchoring(t *testing.T) {
	t.Parallel()
	horizonY := stdViewport.HorizonFraction * stdViewport.CanvasH
	cases := []struct {
		name string
		view project.View
	}{
		{"normalized/time", project.View{XMode: project.TimeProgress, YMode: project.NormalizedByPeak, PeakAltitudeDeg: 60}},
		{"geometric/time", project.View{XMode: project.TimeProgress, YMode: project.Geometric, ElevationDegPerPixel: 0.2}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := mustProject(t, stdViewport, c.view, project.Object{AltitudeDeg: 0, TimeProgress: 0.5})
			approx(t, "Y at altitude 0", p.Y, horizonY)
			if p.BelowHorizon {
				t.Error("altitude 0 should not be marked below horizon")
			}
		})
	}
}

// TestNormalizedByPeakFillsSkyBand checks that an object at the supplied peak
// altitude reaches the top of the sky band (y near 0) and one at half the peak
// sits at half the band height.
func TestNormalizedByPeakFillsSkyBand(t *testing.T) {
	t.Parallel()
	v := project.View{XMode: project.TimeProgress, YMode: project.NormalizedByPeak, PeakAltitudeDeg: 60}
	horizonY := stdViewport.HorizonFraction * stdViewport.CanvasH

	peak := mustProject(t, stdViewport, v, project.Object{AltitudeDeg: 60, TimeProgress: 0.5})
	approx(t, "Y at peak", peak.Y, 0)

	half := mustProject(t, stdViewport, v, project.Object{AltitudeDeg: 30, TimeProgress: 0.5})
	approx(t, "Y at half peak", half.Y, horizonY/2)
}

// TestGeometricElevation checks the fixed degrees-per-pixel vertical mapping.
func TestGeometricElevation(t *testing.T) {
	t.Parallel()
	v := project.View{XMode: project.TimeProgress, YMode: project.Geometric, ElevationDegPerPixel: 0.2}
	horizonY := stdViewport.HorizonFraction * stdViewport.CanvasH
	// 10 degrees up at 0.2 deg/px is 50 pixels above the horizon.
	p := mustProject(t, stdViewport, v, project.Object{AltitudeDeg: 10, TimeProgress: 0})
	approx(t, "Y", p.Y, horizonY-50)
}

// TestNegativeAltitudeBelowHorizon checks that a slightly-negative altitude dips
// into the ground band and is flagged, for both vertical modes.
func TestNegativeAltitudeBelowHorizon(t *testing.T) {
	t.Parallel()
	horizonY := stdViewport.HorizonFraction * stdViewport.CanvasH
	cases := []struct {
		name string
		view project.View
	}{
		{"normalized", project.View{XMode: project.TimeProgress, YMode: project.NormalizedByPeak, PeakAltitudeDeg: 60}},
		{"geometric", project.View{XMode: project.TimeProgress, YMode: project.Geometric, ElevationDegPerPixel: 0.2}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := mustProject(t, stdViewport, c.view, project.Object{AltitudeDeg: -2, TimeProgress: 0.5})
			if !p.BelowHorizon {
				t.Error("negative altitude should be marked below horizon")
			}
			if p.Y <= horizonY {
				t.Errorf("Y = %v, want below the horizon line %v", p.Y, horizonY)
			}
		})
	}
}

// TestTimeProgressEndpoints checks the time-progress horizontal mapping across
// the full canvas width.
func TestTimeProgressEndpoints(t *testing.T) {
	t.Parallel()
	v := project.View{XMode: project.TimeProgress, YMode: project.NormalizedByPeak, PeakAltitudeDeg: 60}
	cases := []struct {
		tp   float64
		want float64
	}{
		{0, 0},
		{0.5, 500},
		{1, 1000},
	}
	for _, c := range cases {
		p := mustProject(t, stdViewport, v, project.Object{AltitudeDeg: 20, TimeProgress: c.tp})
		approx(t, "X", p.X, c.want)
		if !p.Visible {
			t.Error("time-progress objects should be visible")
		}
	}
}

// TestDirectionalCentering checks the heading-centered slice: the heading lands
// at the canvas middle, heading +/- half the field of view lands at the edges,
// and objects beyond the field of view are off-screen.
func TestDirectionalCentering(t *testing.T) {
	t.Parallel()
	v := project.View{
		XMode:           project.Azimuth,
		ViewMode:        project.Directional,
		YMode:           project.NormalizedByPeak,
		PeakAltitudeDeg: 60,
		ViewHeading:     180,
		FieldOfView:     90,
	}
	cases := []struct {
		name    string
		az      float64
		wantX   float64
		visible bool
	}{
		{"heading center", 180, 500, true},
		{"right edge", 225, 1000, true},
		{"left edge", 135, 0, true},
		{"beyond right", 250, 0, false},
		{"beyond left", 110, 0, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			p := mustProject(t, stdViewport, v, project.Object{AltitudeDeg: 20, AzimuthDeg: c.az})
			if p.Visible != c.visible {
				t.Errorf("Visible = %v, want %v", p.Visible, c.visible)
			}
			if c.visible {
				approx(t, "X", p.X, c.wantX)
			}
		})
	}
}

// TestDirectionalWrap checks that the signed shortest angular difference is used,
// so a heading near north handles the 0/360 seam.
func TestDirectionalWrap(t *testing.T) {
	t.Parallel()
	v := project.View{
		XMode:           project.Azimuth,
		ViewMode:        project.Directional,
		YMode:           project.NormalizedByPeak,
		PeakAltitudeDeg: 60,
		ViewHeading:     10,
		FieldOfView:     60,
	}
	// Azimuth 350 is 20 degrees west (left) of heading 10 across the seam.
	p := mustProject(t, stdViewport, v, project.Object{AltitudeDeg: 20, AzimuthDeg: 350})
	if !p.Visible {
		t.Fatal("azimuth 350 within 30 deg of heading 10 should be visible")
	}
	// X = 500 + (-20)*(1000/60) = 500 - 333.333
	approx(t, "X", p.X, 500-20*(1000.0/60.0))
}

// TestFullArcSpansSweep checks that the sunrise->sunset azimuth sweep fills the
// canvas width: sunrise at x=0, sunset at x=CanvasW, midpoint centered.
func TestFullArcSpansSweep(t *testing.T) {
	t.Parallel()
	v := project.View{
		XMode:           project.Azimuth,
		ViewMode:        project.FullArc,
		YMode:           project.NormalizedByPeak,
		PeakAltitudeDeg: 60,
		SunriseAzimuth:  80,
		SunsetAzimuth:   280, // span 200
	}
	cases := []struct {
		az    float64
		wantX float64
	}{
		{80, 0},
		{180, 500},
		{280, 1000},
	}
	for _, c := range cases {
		p := mustProject(t, stdViewport, v, project.Object{AltitudeDeg: 20, AzimuthDeg: c.az})
		approx(t, "X", p.X, c.wantX)
		if !p.Visible {
			t.Error("full-arc objects should be visible")
		}
	}
}

// TestDiameterScalesWithFieldOfView checks that the disc is sized by the same
// degrees-per-pixel as the horizontal axis, so a narrower field of view (more
// pixels per degree) yields a larger disc.
func TestDiameterScalesWithFieldOfView(t *testing.T) {
	t.Parallel()
	base := project.View{
		XMode:           project.Azimuth,
		ViewMode:        project.Directional,
		YMode:           project.NormalizedByPeak,
		PeakAltitudeDeg: 60,
		ViewHeading:     180,
	}
	obj := project.Object{AltitudeDeg: 20, AzimuthDeg: 180, AngularDiameterDeg: 0.5}

	wide := base
	wide.FieldOfView = 90
	narrow := base
	narrow.FieldOfView = 45

	pWide := mustProject(t, stdViewport, wide, obj)
	pNarrow := mustProject(t, stdViewport, narrow, obj)

	approx(t, "wide diameter", pWide.DiameterPx, 0.5*1000/90)
	approx(t, "narrow diameter", pNarrow.DiameterPx, 0.5*1000/45)
	if !(pNarrow.DiameterPx > pWide.DiameterPx) {
		t.Errorf("narrower field of view should enlarge the disc: narrow=%v wide=%v", pNarrow.DiameterPx, pWide.DiameterPx)
	}
}

// TestDiameterFullArc checks disc sizing under the full-arc azimuth span.
func TestDiameterFullArc(t *testing.T) {
	t.Parallel()
	v := project.View{
		XMode:           project.Azimuth,
		ViewMode:        project.FullArc,
		YMode:           project.NormalizedByPeak,
		PeakAltitudeDeg: 60,
		SunriseAzimuth:  80,
		SunsetAzimuth:   280, // span 200
	}
	p := mustProject(t, stdViewport, v, project.Object{AltitudeDeg: 20, AzimuthDeg: 180, AngularDiameterDeg: 0.5})
	approx(t, "diameter", p.DiameterPx, 0.5*1000/200)
}

// TestColumnAzimuthInvertsDirectional checks that ColumnAzimuth is the inverse of
// the directional X map: a column's azimuth reprojects to the same column.
func TestColumnAzimuthInvertsDirectional(t *testing.T) {
	t.Parallel()
	v := project.View{
		XMode:           project.Azimuth,
		ViewMode:        project.Directional,
		YMode:           project.NormalizedByPeak,
		PeakAltitudeDeg: 60,
		ViewHeading:     180,
		FieldOfView:     90,
	}
	for x := 0.0; x <= 1000; x += 125 {
		az, err := project.ColumnAzimuth(stdViewport, v, x)
		if err != nil {
			t.Fatalf("ColumnAzimuth(%v): %v", x, err)
		}
		p := mustProject(t, stdViewport, v, project.Object{AltitudeDeg: 20, AzimuthDeg: az})
		approx(t, "reprojected X", p.X, x)
	}
}

// TestColumnAzimuthInvertsFullArc checks the inverse under the full-arc sweep and
// that the endpoints recover the sunrise and sunset azimuths.
func TestColumnAzimuthInvertsFullArc(t *testing.T) {
	t.Parallel()
	v := project.View{
		XMode:          project.Azimuth,
		ViewMode:       project.FullArc,
		SunriseAzimuth: 80,
		SunsetAzimuth:  280,
	}
	start, err := project.ColumnAzimuth(stdViewport, v, 0)
	if err != nil {
		t.Fatal(err)
	}
	approx(t, "sunrise azimuth", start, 80)
	end, err := project.ColumnAzimuth(stdViewport, v, 1000)
	if err != nil {
		t.Fatal(err)
	}
	approx(t, "sunset azimuth", end, 280)
	mid, err := project.ColumnAzimuth(stdViewport, v, 500)
	if err != nil {
		t.Fatal(err)
	}
	approx(t, "midpoint azimuth", mid, 180)
}

// TestColumnAzimuthRequiresAzimuthMode checks that ColumnAzimuth rejects the
// time-progress horizontal mode, which has no azimuth axis.
func TestColumnAzimuthRequiresAzimuthMode(t *testing.T) {
	t.Parallel()
	v := project.View{XMode: project.TimeProgress, YMode: project.NormalizedByPeak, PeakAltitudeDeg: 60}
	_, err := project.ColumnAzimuth(stdViewport, v, 500)
	wantCoded(t, "ColumnAzimuth time-progress", err, project.ErrXModeNotAzimuth, astronomy.CodeXModeNotAzimuth)
}

// TestAngularSeparation checks the shortest angular separation, including wraps
// across the 0/360 seam, in the range [0, 180].
func TestAngularSeparation(t *testing.T) {
	t.Parallel()
	cases := []struct {
		a, b, want float64
	}{
		{0, 0, 0},
		{10, 350, 20},
		{350, 10, 20},
		{0, 180, 180},
		{0, 270, 90},
		{90, 90, 0},
		{359, 1, 2},
		{45, 315, 90},
	}
	for _, c := range cases {
		if got := project.AngularSeparation(c.a, c.b); math.Abs(got-c.want) > eps {
			t.Errorf("AngularSeparation(%v, %v) = %v, want %v", c.a, c.b, got, c.want)
		}
	}
}

// wantCoded asserts that err both matches the sentinel cause via errors.Is and
// carries the expected stable go-apperr code.
func wantCoded(t *testing.T, context string, err error, cause error, code int) {
	t.Helper()
	if !errors.Is(err, cause) {
		t.Errorf("%s: err = %v, want errors.Is %v", context, err, cause)
	}
	if got, ok := apperr.Code(err); !ok || got != code {
		t.Errorf("%s: code = %d ok=%v, want %d", context, got, ok, code)
	}
}

// TestProjectInvalidInputs checks the coded errors for every invalid input path.
func TestProjectInvalidInputs(t *testing.T) {
	t.Parallel()
	good := project.View{XMode: project.TimeProgress, YMode: project.NormalizedByPeak, PeakAltitudeDeg: 60}
	obj := project.Object{AltitudeDeg: 20, TimeProgress: 0.5}

	cases := []struct {
		name  string
		vp    project.Viewport
		view  project.View
		cause error
		code  int
	}{
		{
			"zero width",
			project.Viewport{CanvasW: 0, CanvasH: 500, HorizonFraction: 0.7},
			good, project.ErrInvalidCanvas, astronomy.CodeInvalidCanvas,
		},
		{
			"negative height",
			project.Viewport{CanvasW: 1000, CanvasH: -1, HorizonFraction: 0.7},
			good, project.ErrInvalidCanvas, astronomy.CodeInvalidCanvas,
		},
		{
			"horizon fraction too large",
			project.Viewport{CanvasW: 1000, CanvasH: 500, HorizonFraction: 1.5},
			good, project.ErrInvalidHorizonFraction, astronomy.CodeInvalidHorizonFraction,
		},
		{
			"horizon fraction negative",
			project.Viewport{CanvasW: 1000, CanvasH: 500, HorizonFraction: -0.1},
			good, project.ErrInvalidHorizonFraction, astronomy.CodeInvalidHorizonFraction,
		},
		{
			"non-positive peak",
			stdViewport,
			project.View{XMode: project.TimeProgress, YMode: project.NormalizedByPeak, PeakAltitudeDeg: 0},
			project.ErrInvalidPeakAltitude, astronomy.CodeInvalidPeakAltitude,
		},
		{
			"non-positive elevation scale",
			stdViewport,
			project.View{XMode: project.TimeProgress, YMode: project.Geometric, ElevationDegPerPixel: 0},
			project.ErrInvalidElevationScale, astronomy.CodeInvalidElevationScale,
		},
		{
			"non-positive field of view",
			stdViewport,
			project.View{XMode: project.Azimuth, ViewMode: project.Directional, YMode: project.NormalizedByPeak, PeakAltitudeDeg: 60, FieldOfView: 0},
			project.ErrInvalidFieldOfView, astronomy.CodeInvalidFieldOfView,
		},
		{
			"degenerate full-arc span",
			stdViewport,
			project.View{XMode: project.Azimuth, ViewMode: project.FullArc, YMode: project.NormalizedByPeak, PeakAltitudeDeg: 60, SunriseAzimuth: 90, SunsetAzimuth: 90},
			project.ErrInvalidFieldOfView, astronomy.CodeInvalidFieldOfView,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := project.Project(c.vp, c.view, obj)
			wantCoded(t, c.name, err, c.cause, c.code)
		})
	}
}

// TestColumnAzimuthInvalidInputs checks that ColumnAzimuth validates the viewport
// and azimuth-mode parameters the same way Project does.
func TestColumnAzimuthInvalidInputs(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		vp    project.Viewport
		view  project.View
		cause error
		code  int
	}{
		{
			"bad canvas",
			project.Viewport{CanvasW: 0, CanvasH: 500, HorizonFraction: 0.7},
			project.View{XMode: project.Azimuth, ViewMode: project.Directional, FieldOfView: 90, ViewHeading: 180},
			project.ErrInvalidCanvas, astronomy.CodeInvalidCanvas,
		},
		{
			"bad field of view",
			stdViewport,
			project.View{XMode: project.Azimuth, ViewMode: project.Directional, FieldOfView: 0, ViewHeading: 180},
			project.ErrInvalidFieldOfView, astronomy.CodeInvalidFieldOfView,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			_, err := project.ColumnAzimuth(c.vp, c.view, 100)
			wantCoded(t, c.name, err, c.cause, c.code)
		})
	}
}

// TestDefaultsAreZeroValues documents that the zero values of the mode enums are
// the rabbit-hole defaults: normalized-by-peak, time-progress, full-arc.
func TestDefaultsAreZeroValues(t *testing.T) {
	t.Parallel()
	if project.NormalizedByPeak != 0 {
		t.Error("NormalizedByPeak should be the zero-value YMode default")
	}
	if project.TimeProgress != 0 {
		t.Error("TimeProgress should be the zero-value XMode default")
	}
	if project.FullArc != 0 {
		t.Error("FullArc should be the zero-value ViewMode default")
	}
}
