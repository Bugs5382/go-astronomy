package moon_test

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
	"testing"
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth/moon"
)

// brooklyn is a mid-latitude northern observer used across the position tests.
func brooklyn() astronomy.Observer {
	return astronomy.Observer{Lat: 40.678, Lng: -73.944, TZ: time.UTC}
}

// TestPositionMeeusExample anchors the topocentric horizontal transform and the
// apparent diameter against the meeus worked example (Astronomical Algorithms,
// example 47.a: 1992 April 12, 0h UT), for which the geocentric Moon sits at a
// distance of 368409.7 km. The expected altitude, azimuth, and diameter were
// derived from the library's own coordinate chain and pinned here as a
// regression anchor.
func TestPositionMeeusExample(t *testing.T) {
	t.Parallel()
	when := time.Date(1992, 4, 12, 0, 0, 0, 0, time.UTC)
	pos, err := moon.Position(brooklyn(), when)
	if err != nil {
		t.Fatalf("Position returned error: %v", err)
	}
	if got, want := pos.Altitude, 61.69; abs(got-want) > 0.1 {
		t.Errorf("altitude = %.4f, want ~%.2f", got, want)
	}
	if got, want := pos.Azimuth, 162.78; abs(got-want) > 0.1 {
		t.Errorf("azimuth = %.4f, want ~%.2f", got, want)
	}
	if got, want := float64(pos.Diameter), 0.5406; abs(got-want) > 0.002 {
		t.Errorf("diameter = %.4f, want ~%.4f", got, want)
	}
}

// TestPositionDiameterRange checks that the Moon's apparent angular diameter
// stays within its physical bounds (roughly 0.49 to 0.57 degrees) across a
// range of instants spanning the anomalistic cycle of the Earth-Moon distance.
func TestPositionDiameterRange(t *testing.T) {
	t.Parallel()
	obs := brooklyn()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 40; i++ {
		when := base.Add(time.Duration(i) * 24 * time.Hour)
		pos, err := moon.Position(obs, when)
		if err != nil {
			t.Fatalf("Position(%s) error: %v", when, err)
		}
		if d := float64(pos.Diameter); d < 0.48 || d > 0.58 {
			t.Errorf("diameter at %s = %.4f, out of physical range", when, d)
		}
	}
}

// TestPositionAzimuthRange asserts azimuth is always normalized to [0, 360).
func TestPositionAzimuthRange(t *testing.T) {
	t.Parallel()
	obs := astronomy.Observer{Lat: -33.87, Lng: 151.21, TZ: time.UTC} // Sydney
	base := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 48; i++ {
		when := base.Add(time.Duration(i) * 30 * time.Minute)
		pos, err := moon.Position(obs, when)
		if err != nil {
			t.Fatalf("Position error: %v", err)
		}
		if pos.Azimuth < 0 || pos.Azimuth >= 360 {
			t.Errorf("azimuth at %s = %.4f, out of [0,360)", when, pos.Azimuth)
		}
	}
}

// TestApparentPositionRefractionLifts checks that refraction raises the apparent
// altitude above the geometric altitude while the Moon is above the horizon, and
// never lowers it, matching the physics of atmospheric refraction.
func TestApparentPositionRefractionLifts(t *testing.T) {
	t.Parallel()
	obs := brooklyn()
	base := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	sawAbove := false
	for i := 0; i < 96; i++ {
		when := base.Add(time.Duration(i) * 15 * time.Minute)
		geo, err := moon.Position(obs, when)
		if err != nil {
			t.Fatalf("Position error: %v", err)
		}
		app, err := moon.ApparentPosition(obs, when)
		if err != nil {
			t.Fatalf("ApparentPosition error: %v", err)
		}
		if app.Altitude < geo.Altitude-1e-9 {
			t.Errorf("apparent altitude %.4f below geometric %.4f at %s", app.Altitude, geo.Altitude, when)
		}
		if geo.Altitude > 5 {
			sawAbove = true
		}
	}
	if !sawAbove {
		t.Fatal("test window never placed the Moon above the horizon")
	}
}

// TestPositionInvalidObserver verifies both position functions reject an
// out-of-range observer with the shared coded errors.
func TestPositionInvalidObserver(t *testing.T) {
	t.Parallel()
	when := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name string
		obs  astronomy.Observer
		sent error
		code int
	}{
		{"lat", astronomy.Observer{Lat: 91, Lng: 0}, moon.ErrInvalidLatitude, astronomy.CodeInvalidLatitude},
		{"lng", astronomy.Observer{Lat: 0, Lng: 181}, moon.ErrInvalidLongitude, astronomy.CodeInvalidLongitude},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if _, err := moon.Position(tc.obs, when); !errors.Is(err, tc.sent) || !hasCode(err, tc.code) {
				t.Errorf("Position err = %v, want %v code %d", err, tc.sent, tc.code)
			}
			if _, err := moon.ApparentPosition(tc.obs, when); !errors.Is(err, tc.sent) || !hasCode(err, tc.code) {
				t.Errorf("ApparentPosition err = %v, want %v code %d", err, tc.sent, tc.code)
			}
		})
	}
}

// hasCode reports whether err carries the given go-apperr code.
func hasCode(err error, code int) bool {
	got, ok := apperr.Code(err)
	return ok && got == code
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
