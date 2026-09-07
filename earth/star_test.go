package earth_test

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

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/internal/coordinates"
	"github.com/Bugs5382/go-astronomy/internal/julian"
	"github.com/Bugs5382/go-astronomy/star"
)

func mustStar(t *testing.T, name string) star.Star {
	t.Helper()
	s, ok := star.GetNamedStar(name)
	if !ok {
		t.Fatalf("star %q not in catalog", name)
	}
	return s
}

// starPinned holds each star's altitude and azimuth in degrees as this library
// computes them, over a grid of four named stars, four observers, and three
// instants. The values pin the Earth-vantage star chain so any change to
// precession, sidereal time, or the horizontal transform is caught to the
// twelfth decimal.
//
// The chain's accuracy is established elsewhere against the book rather than
// against itself: precession by Meeus example 21.b and the horizontal transform
// by example 13.b, both in internal/coordinates, and the sidereal time by
// examples 12.a and 12.b in internal/julian.
var starPinned = []struct {
	star     string
	observer string
	when     string
	alt      float64
	az       float64
}{
	{"Sirius", "brooklyn", "2026-01-15T03:00:00Z", 30.698114003175, 162.192176778997},
	{"Sirius", "brooklyn", "2026-07-04T22:30:00Z", -7.091306567564, 253.983161674363},
	{"Sirius", "brooklyn", "1982-05-03T06:00:00Z", -45.064019617046, 288.966912458304},
	{"Sirius", "quito", "2026-01-15T03:00:00Z", 63.906351316179, 130.436007736220},
	{"Sirius", "quito", "2026-07-04T22:30:00Z", 9.248841331046, 253.058101057527},
	{"Sirius", "quito", "1982-05-03T06:00:00Z", -39.106175554452, 248.110253547595},
	{"Sirius", "tromso", "2026-01-15T03:00:00Z", -11.244041906107, 252.019487023630},
	{"Sirius", "tromso", "2026-07-04T22:30:00Z", -37.078796919277, 357.378237449514},
	{"Sirius", "tromso", "1982-05-03T06:00:00Z", -29.294679955982, 55.577457433082},
	{"Sirius", "auckland", "2026-01-15T03:00:00Z", -16.909298593303, 127.166193610509},
	{"Sirius", "auckland", "2026-07-04T22:30:00Z", 59.238388917606, 56.280714870585},
	{"Sirius", "auckland", "1982-05-03T06:00:00Z", 60.438028770582, 306.407147375089},
	{"Vega", "brooklyn", "2026-01-15T03:00:00Z", -9.520195444743, 349.125155128596},
	{"Vega", "brooklyn", "2026-07-04T22:30:00Z", 22.210151138683, 57.201936855248},
	{"Vega", "brooklyn", "1982-05-03T06:00:00Z", 57.789142972751, 79.354121031945},
	{"Vega", "quito", "2026-01-15T03:00:00Z", -47.872925172478, 338.561454997771},
	{"Vega", "quito", "2026-07-04T22:30:00Z", -5.936857031172, 50.967390910371},
	{"Vega", "quito", "1982-05-03T06:00:00Z", 32.166619395846, 42.123625432105},
	{"Vega", "tromso", "2026-01-15T03:00:00Z", 32.430090555708, 65.031515063835},
	{"Vega", "tromso", "2026-07-04T22:30:00Z", 59.158440464058, 179.912048942488},
	{"Vega", "tromso", "1982-05-03T06:00:00Z", 49.365096409300, 247.864275082007},
	{"Vega", "auckland", "2026-01-15T03:00:00Z", -0.986136309057, 320.387542822882},
	{"Vega", "auckland", "2026-07-04T22:30:00Z", -70.802866586192, 283.260095987266},
	{"Vega", "auckland", "1982-05-03T06:00:00Z", -69.053217682249, 76.640832057748},
	{"Betelgeuse", "brooklyn", "2026-01-15T03:00:00Z", 56.584918462732, 173.677309538668},
	{"Betelgeuse", "brooklyn", "2026-07-04T22:30:00Z", -0.694903171831, 280.398149982677},
	{"Betelgeuse", "brooklyn", "1982-05-03T06:00:00Z", -33.779078768510, 321.202140068030},
	{"Betelgeuse", "quito", "2026-01-15T03:00:00Z", 78.966217730252, 46.369229742804},
	{"Betelgeuse", "quito", "2026-07-04T22:30:00Z", -2.824177626869, 277.409818579850},
	{"Betelgeuse", "quito", "1982-05-03T06:00:00Z", -53.186471964354, 282.173163723684},
	{"Betelgeuse", "tromso", "2026-01-15T03:00:00Z", 7.155732141047, 272.021479225887},
	{"Betelgeuse", "tromso", "2026-07-04T22:30:00Z", -12.617890316775, 10.415327953401},
	{"Betelgeuse", "tromso", "1982-05-03T06:00:00Z", -2.591071468933, 60.463928619268},
	{"Betelgeuse", "auckland", "2026-01-15T03:00:00Z", -24.216100650903, 99.226997891368},
	{"Betelgeuse", "auckland", "2026-07-04T22:30:00Z", 43.853450179749, 19.353030040759},
	{"Betelgeuse", "auckland", "1982-05-03T06:00:00Z", 33.796505205810, 314.061206088896},
	{"Polaris", "brooklyn", "2026-01-15T03:00:00Z", 41.163851848081, 359.472891632627},
	{"Polaris", "brooklyn", "2026-07-04T22:30:00Z", 40.197928953289, 359.471220482354},
	{"Polaris", "brooklyn", "1982-05-03T06:00:00Z", 39.928482401642, 0.422392298012},
	{"Polaris", "quito", "2026-01-15T03:00:00Z", 0.336127294678, 359.642824264762},
	{"Polaris", "quito", "2026-07-04T22:30:00Z", -0.626202196523, 359.559572824116},
	{"Polaris", "quito", "1982-05-03T06:00:00Z", -0.952654501631, 0.263881532596},
	{"Polaris", "tromso", "2026-01-15T03:00:00Z", 69.223235031159, 358.685239986938},
	{"Polaris", "tromso", "2026-07-04T22:30:00Z", 69.264301050259, 1.408575246107},
	{"Polaris", "tromso", "1982-05-03T06:00:00Z", 69.997765802604, 2.138668372960},
	{"Polaris", "auckland", "2026-01-15T03:00:00Z", -36.653296381021, 0.745271924290},
	{"Polaris", "auckland", "2026-07-04T22:30:00Z", -36.297706805107, 359.628388599746},
	{"Polaris", "auckland", "1982-05-03T06:00:00Z", -36.874064225402, 358.980853427804},
}

func TestStarPositionPinnedValues(t *testing.T) {
	t.Parallel()
	for _, c := range starPinned {
		s := mustStar(t, c.star)
		obs, ok := namedObservers[c.observer]
		if !ok {
			t.Fatalf("unknown observer %q", c.observer)
		}
		when := mustParse(t, c.when)
		hz, err := earth.StarPosition(s, obs, when)
		if err != nil {
			t.Fatalf("StarPosition(%s): %v", c.star, err)
		}
		if math.Abs(hz.Altitude-c.alt) > 1e-9 {
			t.Errorf("%s %s %s alt = %.12f, want %.12f", c.star, c.observer, c.when, hz.Altitude, c.alt)
		}
		if math.Abs(hz.Azimuth-c.az) > 1e-9 {
			t.Errorf("%s %s %s az = %.12f, want %.12f", c.star, c.observer, c.when, hz.Azimuth, c.az)
		}
		if hz.Azimuth < 0 || hz.Azimuth >= 360 {
			t.Errorf("%s %s %s azimuth %v out of [0,360)", c.star, c.observer, c.when, hz.Azimuth)
		}
	}
}

// TestStarPositionAltitudeIndependent cross-checks the horizontal transform with
// plain spherical trigonometry written out in the test, applied to the same
// precessed equatorial position. It catches a wiring mistake in earth (wrong
// sidereal time, wrong longitude sign, wrong epoch) without reusing the
// transform under test.
func TestStarPositionAltitudeIndependent(t *testing.T) {
	t.Parallel()
	when := time.Date(2026, 2, 10, 2, 0, 0, 0, time.UTC)
	gst := julian.GreenwichSiderealTime(when)
	epochOfDate := 2000.0 + (julian.Date(when)-2451545.0)/365.25
	for _, name := range []string{"Sirius", "Vega"} {
		s := mustStar(t, name)
		eq := coordinates.PrecessEquatorial(
			coordinates.Equatorial{RA: s.RA, Dec: s.Dec}, 2000.0, epochOfDate)
		for _, obs := range []astronomy.Observer{brooklyn, quito, auckland} {
			hz, err := earth.StarPosition(s, obs, when)
			if err != nil {
				t.Fatal(err)
			}
			wantAlt, wantAz := horizontalFromEquatorial(eq.RA, eq.Dec, gst, obs.Lat, obs.Lng)
			if math.Abs(hz.Altitude-wantAlt) > 1e-9 {
				t.Errorf("%s @ %v altitude = %.9f, want %.9f", name, obs, hz.Altitude, wantAlt)
			}
			d := math.Abs(hz.Azimuth - wantAz)
			if d > 180 {
				d = 360 - d
			}
			if d > 1e-9 {
				t.Errorf("%s @ %v azimuth = %.9f, want %.9f", name, obs, hz.Azimuth, wantAz)
			}
		}
	}
}

// TestStarPositionPolarisAltitude verifies the classic identity that Polaris
// sits at an altitude close to the observer's latitude in the northern
// hemisphere, at any time of night. Polaris is about 0.7 degrees from the
// celestial pole, so the altitude tracks latitude to within roughly that much.
func TestStarPositionPolarisAltitude(t *testing.T) {
	t.Parallel()
	polaris := mustStar(t, "Polaris")
	observers := []astronomy.Observer{brooklyn, tromso}
	hours := []int{0, 6, 12, 18}
	for _, obs := range observers {
		for _, h := range hours {
			when := time.Date(2026, 9, 5, h, 0, 0, 0, time.UTC)
			hz, err := earth.StarPosition(polaris, obs, when)
			if err != nil {
				t.Fatal(err)
			}
			if math.Abs(hz.Altitude-obs.Lat) > 1.0 {
				t.Errorf("Polaris altitude %.3f at lat %.3f (%dh), want within 1 degree",
					hz.Altitude, obs.Lat, h)
			}
		}
	}
}

func TestStarPositionInvalidObserver(t *testing.T) {
	t.Parallel()
	sirius := mustStar(t, "Sirius")
	when := time.Date(2026, 5, 3, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		obs  astronomy.Observer
		code int
	}{
		{astronomy.Observer{Lat: 120, Lng: 0}, astronomy.CodeInvalidLatitude},
		{astronomy.Observer{Lat: -91, Lng: 0}, astronomy.CodeInvalidLatitude},
		{astronomy.Observer{Lat: 0, Lng: 200}, astronomy.CodeInvalidLongitude},
		{astronomy.Observer{Lat: 0, Lng: -181}, astronomy.CodeInvalidLongitude},
	}
	for _, c := range cases {
		_, err := earth.StarPosition(sirius, c.obs, when)
		if err == nil {
			t.Errorf("StarPosition(%v) expected error", c.obs)
			continue
		}
		if got, ok := apperr.Code(err); !ok || got != c.code {
			t.Errorf("StarPosition(%v) code = (%d,%v), want %d", c.obs, got, ok, c.code)
		}
	}
}

// TestStarPositionAboveHorizon confirms the AboveHorizon helper agrees with the
// sign of the altitude across a full day for a star and observer.
func TestStarPositionAboveHorizon(t *testing.T) {
	t.Parallel()
	sirius := mustStar(t, "Sirius")
	start := time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)
	var up, down int
	for i := 0; i < 24; i++ {
		when := start.Add(time.Duration(i) * time.Hour)
		hz, err := earth.StarPosition(sirius, brooklyn, when)
		if err != nil {
			t.Fatal(err)
		}
		if hz.AboveHorizon() != (hz.Altitude > 0) {
			t.Errorf("AboveHorizon %v disagrees with altitude %.3f", hz.AboveHorizon(), hz.Altitude)
		}
		if hz.AboveHorizon() {
			up++
		} else {
			down++
		}
	}
	// Sirius rises and sets from Brooklyn, so both states occur within a day.
	if up == 0 || down == 0 {
		t.Errorf("Sirius never changed horizon state: up=%d down=%d", up, down)
	}
}
