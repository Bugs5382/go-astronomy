package satellite

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

	astronomy "github.com/Bugs5382/go-astronomy"
)

func TestGMST82(t *testing.T) {
	ref := readPyReference(t)
	for _, g := range ref.GMST {
		if got := gmst82(g.JD); math.Abs(got-g.GMST) > 1e-12 {
			t.Errorf("gmst82(%v) = %v, want %v", g.JD, got, g.GMST)
		}
	}
	// J2000.0: 67310.54841 s of sidereal time = 280.46061837 degrees.
	if got := gmst82(2451545.0) / deg2rd; math.Abs(got-280.46061837504) > 1e-9 {
		t.Errorf("gmst82(J2000) = %v deg", got)
	}
}

// TestGeodeticRoundTrip checks that the geodetic sub-point of an observer's
// own site is the site, and that a point above it reports its height.
func TestGeodeticRoundTrip(t *testing.T) {
	t.Parallel()
	for _, o := range []astronomy.Observer{{Lat: 39.74, Lng: -104.99}, {Lat: -33.87, Lng: 151.21}, {Lat: 78.2, Lng: 15.6}} {
		p := siteECEF(o)
		lat, lng, h := geodetic(p)
		if math.Abs(lat-o.Lat) > 1e-9 || math.Abs(lng-o.Lng) > 1e-9 || math.Abs(h) > 1e-6 {
			t.Errorf("%+v: %v %v %v km", o, lat, lng, h)
		}
		n := norm(p)
		up := [3]float64{p[0] / n * 400, p[1] / n * 400, p[2] / n * 400}
		_, _, h = geodetic([3]float64{p[0] + up[0], p[1] + up[1], p[2] + up[2]})
		if math.Abs(h-400) > 2 {
			t.Errorf("%+v: 400 km along the radius gave height %.2f km", o, h)
		}
	}
	if d := angleBetween([3]float64{1, 0, 0}, [3]float64{0, 1, 0}); math.Abs(d-math.Pi/2) > 1e-15 {
		t.Errorf("angleBetween = %v", d)
	}
}
