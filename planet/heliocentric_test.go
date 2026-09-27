package planet_test

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
	"math"
	"testing"
	"time"

	"github.com/Bugs5382/go-astronomy/planet"
)

// TestEarthHeliocentric checks Earth sits opposite the Sun near the March
// equinox, about 1 au out.
func TestEarthHeliocentric(t *testing.T) {
	t.Parallel()
	e := planet.EarthHeliocentric(time.Date(2027, 3, 20, 0, 0, 0, 0, time.UTC))
	if math.Abs(e.Lon-180) > 1 || e.DistanceAU < 0.98 || e.DistanceAU > 1.02 || math.Abs(e.Lat) > 0.001 {
		t.Errorf("Earth %+v at the March equinox", e)
	}
}

// TestVector checks the rectangular form of a heliocentric position.
func TestVector(t *testing.T) {
	t.Parallel()
	v := planet.HeliocentricPosition{Lon: 90, Lat: 0, DistanceAU: 2}.Vector()
	if math.Abs(v[0]) > 1e-12 || math.Abs(v[1]-2) > 1e-12 || v[2] != 0 {
		t.Errorf("Vector = %v, want (0, 2, 0)", v)
	}
	v = planet.HeliocentricPosition{Lon: 0, Lat: 90, DistanceAU: 1}.Vector()
	if math.Abs(v[2]-1) > 1e-12 {
		t.Errorf("Vector = %v, want (0, 0, 1)", v)
	}
}
