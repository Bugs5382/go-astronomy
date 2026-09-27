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
	"math"
	"testing"
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/earth/moon"
)

// TestMoonRiseSetTakeTheDip checks that height moves moonrise earlier and
// moonset later by the horizon dip (issue 52): at 1609 m the dip is 1.18
// degrees, several minutes of the Moon's motion at Denver's latitude.
func TestMoonRiseSetTakeTheDip(t *testing.T) {
	t.Parallel()
	sea := astronomy.Observer{Lat: 39.74, Lng: -104.99, TZ: time.UTC}
	high := sea
	high.Height = astronomy.Meters(1609)
	from := time.Date(2027, 6, 21, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		start := from.AddDate(0, 0, 6*i)
		// Find each event at sea level, then search from half an hour before it
		// at height, so both searches land on the same event.
		r0, ok, err := moon.NextRise(sea, start)
		if err != nil || !ok {
			t.Fatalf("NextRise: %v %v", ok, err)
		}
		r1, _, _ := moon.NextRise(high, r0.Add(-30*time.Minute))
		if d := r0.Sub(r1); d < 4*time.Minute || d > 15*time.Minute {
			t.Errorf("rise at %s: height moved it earlier by %v, want 4 to 15 minutes", r0.Format(time.RFC3339), d)
		}
		s0, ok, err := moon.NextSet(sea, start)
		if err != nil || !ok {
			t.Fatalf("NextSet: %v %v", ok, err)
		}
		s1, _, _ := moon.NextSet(high, s0.Add(-30*time.Minute))
		if d := s1.Sub(s0); d < 4*time.Minute || d > 15*time.Minute {
			t.Errorf("set at %s: height moved it later by %v, want 4 to 15 minutes", s0.Format(time.RFC3339), d)
		}
	}
}

// TestMoonRefractionThinsWithHeight checks the Moon's refraction follows the
// standard atmosphere at the observer's height (issue 68). At moonrise from
// 1609 m the geometric centre sits at the scaled horizon refraction plus the
// semidiameter plus the dip below the horizon, about 0.08 degrees higher than
// the sea-level refraction would put it; and ApparentPosition lifts the Moon
// by the scaled refraction, not the sea-level one.
func TestMoonRefractionThinsWithHeight(t *testing.T) {
	t.Parallel()
	high := astronomy.Observer{Lat: 39.74, Lng: -104.99, TZ: time.UTC, Height: astronomy.Meters(1609)}
	f := earth.StandardAtmosphere.Factor(high.Height)
	if f > 0.86 || f < 0.83 {
		t.Fatalf("factor at 1609 m = %v", f)
	}
	rise, ok, err := moon.NextRise(high, time.Date(2027, 6, 21, 0, 0, 0, 0, time.UTC))
	if err != nil || !ok {
		t.Fatalf("NextRise: %v %v", ok, err)
	}
	pos, err := moon.Position(high, rise)
	if err != nil {
		t.Fatal(err)
	}
	want := -(0.5667*f + pos.Diameter.Radius() + earth.HorizonDip(high.Height))
	if math.Abs(pos.Altitude-want) > 0.002 {
		t.Errorf("altitude at moonrise %.4f, want %.4f (sea-level refraction would give %.4f)",
			pos.Altitude, want, want-(1-f)*0.5667)
	}

	for _, when := range []time.Time{rise.Add(20 * time.Minute), rise.Add(time.Hour), rise.Add(3 * time.Hour)} {
		geo, _ := moon.Position(high, when)
		app, err := moon.ApparentPosition(high, when)
		if err != nil {
			t.Fatal(err)
		}
		lift, want := app.Altitude-geo.Altitude, earth.StandardAtmosphere.Refraction(geo.Altitude, high.Height)
		if math.Abs(lift-want) > 1e-12 || !(lift < earth.Refraction(geo.Altitude)) {
			t.Errorf("at %s: refraction %.5f, want %.5f (sea level %.5f)", when.Format(time.RFC3339), lift, want, earth.Refraction(geo.Altitude))
		}
	}
}

// TestMoonPositionHeightParallax checks that height enters the parallax but
// moves the Moon by well under an arc second at 1609 m.
func TestMoonPositionHeightParallax(t *testing.T) {
	t.Parallel()
	sea := astronomy.Observer{Lat: 39.74, Lng: -104.99}
	high := sea
	high.Height = astronomy.Meters(1609)
	when := time.Date(2027, 6, 25, 6, 0, 0, 0, time.UTC)
	a, err := moon.Position(sea, when)
	if err != nil {
		t.Fatal(err)
	}
	b, err := moon.Position(high, when)
	if err != nil {
		t.Fatal(err)
	}
	d := math.Hypot(a.Altitude-b.Altitude, (a.Azimuth-b.Azimuth)*math.Cos(a.Altitude*math.Pi/180)) * 3600
	if d == 0 || d > 1 {
		t.Errorf("height moved the Moon by %.3f arcsec, want more than 0 and under 1", d)
	}
}

// TestMoonInvalidHeight checks the Moon's functions reject a NaN or infinite
// height with the coded ErrInvalidHeight.
func TestMoonInvalidHeight(t *testing.T) {
	t.Parallel()
	when := time.Date(2027, 6, 21, 0, 0, 0, 0, time.UTC)
	for _, h := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		obs := astronomy.Observer{Lat: 39.74, Lng: -104.99, Height: astronomy.Meters(h)}
		_, err := moon.Position(obs, when)
		if code, _ := apperr.Code(err); !errors.Is(err, astronomy.ErrInvalidHeight) || code != astronomy.CodeInvalidHeight {
			t.Errorf("Position(height %v) error = %v, want ErrInvalidHeight", h, err)
		}
		if _, _, err := moon.NextRise(obs, when); !errors.Is(err, astronomy.ErrInvalidHeight) {
			t.Errorf("NextRise(height %v) error = %v, want ErrInvalidHeight", h, err)
		}
	}
}
