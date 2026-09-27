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
	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
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
// the space motion, precession, nutation, aberration, sidereal time, or the
// horizontal transform is caught to the twelfth decimal.
//
// The values moved by up to 43 arc seconds when the chain gained proper
// motion, nutation, annual aberration, and parallax (issue 37); the new
// chain's accuracy is established against the IAU SOFA library in
// TestStarPositionAgainstSOFA and against Meeus example 23.a in
// internal/ephemeris, not against itself.
var starPinned = []struct {
	star     string
	observer string
	when     string
	alt      float64
	az       float64
}{
	{"Sirius", "brooklyn", "2026-01-15T03:00:00Z", 30.689988297730, 162.193014075609},
	{"Sirius", "brooklyn", "2026-07-04T22:30:00Z", -7.103158648039, 253.984684451110},
	{"Sirius", "brooklyn", "1982-05-03T06:00:00Z", -45.061963922789, 288.969721762789},
	{"Sirius", "quito", "2026-01-15T03:00:00Z", 63.900451777849, 130.448815397483},
	{"Sirius", "quito", "2026-07-04T22:30:00Z", 9.238551282755, 253.051949087248},
	{"Sirius", "quito", "1982-05-03T06:00:00Z", -39.106665093698, 248.113881106581},
	{"Sirius", "tromso", "2026-01-15T03:00:00Z", -11.251165494507, 252.015434266336},
	{"Sirius", "tromso", "2026-07-04T22:30:00Z", -37.085497184718, 357.390637620832},
	{"Sirius", "tromso", "1982-05-03T06:00:00Z", -29.292029426190, 55.576234254323},
	{"Sirius", "auckland", "2026-01-15T03:00:00Z", -16.904132331288, 127.172791890495},
	{"Sirius", "auckland", "2026-07-04T22:30:00Z", 59.250039698099, 56.275535737147},
	{"Sirius", "auckland", "1982-05-03T06:00:00Z", 60.436070953363, 306.411365071916},
	{"Vega", "brooklyn", "2026-01-15T03:00:00Z", -9.522701713164, 349.129586705073},
	{"Vega", "brooklyn", "2026-07-04T22:30:00Z", 22.205466456412, 57.197407088517},
	{"Vega", "brooklyn", "1982-05-03T06:00:00Z", 57.784393927959, 79.361383333492},
	{"Vega", "quito", "2026-01-15T03:00:00Z", -47.876623131948, 338.566556265420},
	{"Vega", "quito", "2026-07-04T22:30:00Z", -5.943022906934, 50.966153120096},
	{"Vega", "quito", "1982-05-03T06:00:00Z", 32.166289593835, 42.130852980491},
	{"Vega", "tromso", "2026-01-15T03:00:00Z", 32.430495515287, 65.037464798565},
	{"Vega", "tromso", "2026-07-04T22:30:00Z", 59.159172268566, 179.899867671213},
	{"Vega", "tromso", "1982-05-03T06:00:00Z", 49.361862180626, 247.856284578229},
	{"Vega", "auckland", "2026-01-15T03:00:00Z", -0.987994899805, 320.382859338238},
	{"Vega", "auckland", "2026-07-04T22:30:00Z", -70.796647753298, 283.262915592297},
	{"Vega", "auckland", "1982-05-03T06:00:00Z", -69.056342539818, 76.655575126672},
	{"Betelgeuse", "brooklyn", "2026-01-15T03:00:00Z", 56.586151534329, 173.667088226371},
	{"Betelgeuse", "brooklyn", "2026-07-04T22:30:00Z", -0.697063830909, 280.403453434505},
	{"Betelgeuse", "brooklyn", "1982-05-03T06:00:00Z", -33.782827429396, 321.205480866860},
	{"Betelgeuse", "quito", "2026-01-15T03:00:00Z", 78.961035301221, 46.382389290813},
	{"Betelgeuse", "quito", "2026-07-04T22:30:00Z", -2.829288884302, 277.412403574593},
	{"Betelgeuse", "quito", "1982-05-03T06:00:00Z", -53.190974895539, 282.171128898935},
	{"Betelgeuse", "tromso", "2026-01-15T03:00:00Z", 7.159256571544, 272.016884680471},
	{"Betelgeuse", "tromso", "2026-07-04T22:30:00Z", -12.615005536596, 10.420396958258},
	{"Betelgeuse", "tromso", "1982-05-03T06:00:00Z", -2.591646056262, 60.468562910894},
	{"Betelgeuse", "auckland", "2026-01-15T03:00:00Z", -24.221515090096, 99.229159873669},
	{"Betelgeuse", "auckland", "2026-07-04T22:30:00Z", 43.852347092206, 19.345237889252},
	{"Betelgeuse", "auckland", "1982-05-03T06:00:00Z", 33.795646823251, 314.055688283061},
	{"Polaris", "brooklyn", "2026-01-15T03:00:00Z", 41.159247195163, 359.480550253897},
	{"Polaris", "brooklyn", "2026-07-04T22:30:00Z", 40.193656494843, 359.472788245249},
	{"Polaris", "brooklyn", "1982-05-03T06:00:00Z", 39.928110524098, 0.431136215664},
	{"Polaris", "quito", "2026-01-15T03:00:00Z", 0.331050138703, 359.648178819394},
	{"Polaris", "quito", "2026-07-04T22:30:00Z", -0.630560886020, 359.560403412495},
	{"Polaris", "quito", "1982-05-03T06:00:00Z", -0.953521704187, 0.270541788762},
	{"Polaris", "tromso", "2026-01-15T03:00:00Z", 69.229292044168, 358.697122040079},
	{"Polaris", "tromso", "2026-07-04T22:30:00Z", 69.265590174792, 1.420567206006},
	{"Polaris", "tromso", "1982-05-03T06:00:00Z", 70.004477246000, 2.139356011919},
	{"Polaris", "auckland", "2026-01-15T03:00:00Z", -36.657007762248, 0.737322247984},
	{"Polaris", "auckland", "2026-07-04T22:30:00Z", -36.297227553880, 359.622915393702},
	{"Polaris", "auckland", "1982-05-03T06:00:00Z", -36.880160289282, 358.977331557795},
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
// apparent equatorial place. It catches a wiring mistake in earth (wrong
// sidereal time, wrong longitude sign, wrong epoch) without reusing the
// transform under test.
func TestStarPositionAltitudeIndependent(t *testing.T) {
	t.Parallel()
	when := time.Date(2026, 2, 10, 2, 0, 0, 0, time.UTC)
	gst := julian.ApparentSiderealTime(when)
	jde := julian.TT(when)
	dpsi, deps := ephemeris.Nutation(jde)
	for _, name := range []string{"Sirius", "Vega"} {
		s := mustStar(t, name)
		ra, dec := s.PositionAt(when)
		ra, dec = ephemeris.ApparentFromJ2000(ra, dec, jde, dpsi, deps, s.Distance)
		eq := coordinates.Equatorial{RA: ra, Dec: dec}
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
