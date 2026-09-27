package satellite_test

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
	"github.com/Bugs5382/go-astronomy/satellite"
)

// TestPositionBasics checks a Look against the pass geometry and the input
// validation.
func TestPositionBasics(t *testing.T) {
	t.Parallel()
	e := iss(t)
	obs := astronomy.Observer{Lat: -33.87, Lng: 151.21}
	peak := time.Date(2026, 9, 26, 4, 5, 3, 457000000, time.UTC) // Skyfield: 88.60 degrees
	l, err := satellite.Position(obs, e, peak)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(l.Altitude-88.5994) > 0.05 || l.RangeKm < 350 || l.RangeKm > 450 {
		t.Errorf("look %+v", l)
	}
	if l.AltitudeKm < 350 || l.AltitudeKm > 450 || math.Abs(l.Latitude-obs.Lat) > 1 {
		t.Errorf("sub-satellite point %.2f %.2f at %.1f km", l.Latitude, l.Longitude, l.AltitudeKm)
	}
	if math.Abs(l.RangeRateKmS) > 0.2 {
		t.Errorf("range rate %.3f km/s at the peak, want about zero", l.RangeRateKmS)
	}
	if got := e.Age(e.Epoch().Add(36 * time.Hour)); got != 36*time.Hour {
		t.Errorf("Age = %v", got)
	}
	if _, err := satellite.Position(astronomy.Observer{Lat: 91}, e, peak); !errors.Is(err, satellite.ErrInvalidLatitude) {
		t.Errorf("latitude 91: %v", err)
	}
	for _, w := range [][2]time.Time{{peak, peak}, {peak, peak.Add(-time.Hour)}, {peak, peak.Add(32 * 24 * time.Hour)}} {
		_, err := satellite.Passes(obs, e, w[0], w[1], satellite.DefaultPassOptions())
		code, _ := apperr.Code(err)
		if !errors.Is(err, satellite.ErrInvalidPassWindow) || code != astronomy.CodeInvalidPassWindow {
			t.Errorf("window %v: %v", w, err)
		}
	}
}
