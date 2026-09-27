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
	"bufio"
	"errors"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/planet"
)

// greenwich is the observer of the Horizons fixtures.
var greenwich = astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}

var byName = map[string]planet.Body{
	"mercury": planet.Mercury, "venus": planet.Venus, "mars": planet.Mars, "jupiter": planet.Jupiter,
	"saturn": planet.Saturn, "uranus": planet.Uranus, "neptune": planet.Neptune,
}

// horizonsRow is one row of testdata/horizons-planets.txt.
type horizonsRow struct {
	body                   planet.Body
	when                   time.Time
	ra, dec, az, el        float64
	mag, illu, diam, delta float64
	elongation, phaseAngle float64
}

func loadPlanets(t *testing.T) []horizonsRow {
	t.Helper()
	f, err := os.Open("testdata/horizons-planets.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var out []horizonsRow
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p := strings.Fields(line)
		if len(p) != 12 {
			t.Fatalf("row %q has %d fields, want 12", line, len(p))
		}
		when, err := time.Parse(time.RFC3339, p[1])
		if err != nil {
			t.Fatal(err)
		}
		var v [10]float64
		for i := range v {
			if v[i], err = strconv.ParseFloat(p[i+2], 64); err != nil {
				t.Fatal(err)
			}
		}
		out = append(out, horizonsRow{byName[p[0]], when, v[0], v[1], v[2], v[3], v[4], v[5], v[6], v[7], v[8], v[9]})
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(out) < 90 {
		t.Fatalf("only %d fixture rows", len(out))
	}
	return out
}

// sepArcsec is the angle between two (longitude-like, latitude-like)
// directions in degrees, in arc seconds.
func sepArcsec(lon1, lat1, lon2, lat2 float64) float64 {
	const r = math.Pi / 180
	c := math.Sin(lat1*r)*math.Sin(lat2*r) + math.Cos(lat1*r)*math.Cos(lat2*r)*math.Cos((lon1-lon2)*r)
	return math.Acos(math.Min(1, c)) / r * 3600
}

// positionTolArcsec is the per-body tolerance on the topocentric apparent
// place against Horizons DE441. The observed worst over the fixture is 0.3
// arc seconds for Mercury, Venus, and Mars, 0.6 for Jupiter and Saturn, and
// 1.5 and 1.7 for Uranus and Neptune, where VSOP87 itself (fitted to an older
// ephemeris) departs from DE441 by that much; the truncation costs under 0.6
// arc seconds anywhere.
var positionTolArcsec = map[planet.Body]float64{
	planet.Mercury: 1, planet.Venus: 1, planet.Mars: 1,
	planet.Jupiter: 1.5, planet.Saturn: 1.5,
	planet.Uranus: 3, planet.Neptune: 3,
}

// horizontalTolArcsec bounds altitude and azimuth, which also carry the
// sidereal time on UTC rather than UT1 (up to 0.9 s of rotation, 13 arc
// seconds at the equator, and about 2 to 3 here).
const horizontalTolArcsec = 5

// TestPositionAgainstHorizons checks every planet's topocentric apparent place,
// distance, diameter, phase, elongation, and magnitude against JPL Horizons
// DE441 at thirteen epochs from 2020 to 2030 and at two conjunctions.
func TestPositionAgainstHorizons(t *testing.T) {
	t.Parallel()
	type worst struct{ pos, hz, mag, illu, diam, delta, elong, phase float64 }
	w := map[planet.Body]*worst{}
	for _, h := range loadPlanets(t) {
		r, err := planet.Position(greenwich, h.body, h.when)
		if err != nil {
			t.Fatal(err)
		}
		if w[h.body] == nil {
			w[h.body] = &worst{}
		}
		ww := w[h.body]
		ww.pos = math.Max(ww.pos, sepArcsec(r.RA, r.Dec, h.ra, h.dec))
		ww.hz = math.Max(ww.hz, sepArcsec(-r.Azimuth, r.Altitude, -h.az, h.el))
		ww.mag = math.Max(ww.mag, math.Abs(r.Magnitude-h.mag))
		ww.illu = math.Max(ww.illu, math.Abs(r.Illuminated*100-h.illu))
		ww.diam = math.Max(ww.diam, math.Abs(float64(r.Diameter)*3600-h.diam))
		ww.delta = math.Max(ww.delta, math.Abs(r.DistanceAU-h.delta))
		ww.elong = math.Max(ww.elong, math.Abs(r.Elongation-h.elongation))
		ww.phase = math.Max(ww.phase, math.Abs(r.PhaseAngle-h.phaseAngle))
	}
	for _, b := range planet.Bodies {
		ww := w[b]
		tol := positionTolArcsec[b]
		switch {
		case ww.pos > tol:
			t.Errorf("%s: apparent place off Horizons by %.2f arcsec, tolerance %.1f", b, ww.pos, tol)
		case ww.hz > horizontalTolArcsec:
			t.Errorf("%s: altitude and azimuth off Horizons by %.2f arcsec", b, ww.hz)
		case ww.mag > 0.1:
			t.Errorf("%s: magnitude off Horizons by %.3f", b, ww.mag)
		case ww.illu > 0.05:
			t.Errorf("%s: illuminated percentage off Horizons by %.3f", b, ww.illu)
		case ww.diam > 0.01:
			t.Errorf("%s: diameter off Horizons by %.3f arcsec", b, ww.diam)
		case ww.delta > 2e-4:
			t.Errorf("%s: distance off Horizons by %.2e au", b, ww.delta)
		case ww.elong > 0.02 || ww.phase > 0.02:
			t.Errorf("%s: elongation or phase angle off Horizons by %.4f, %.4f degrees", b, ww.elong, ww.phase)
		}
		t.Logf("%-8s pos %.2f\" hz %.2f\" mag %.3f illu %.3f%% diam %.3f\" delta %.2e au elong %.4f phase %.4f",
			b, ww.pos, ww.hz, ww.mag, ww.illu, ww.diam, ww.delta, ww.elong, ww.phase)
	}
}

// TestHeliocentricMeeusExample checks Venus against Meeus example 32.a (1992
// December 20, 0h TD): L = 26.11428, B = -2.62070 degrees, R = 0.724603 au. The
// book evaluates the abridged series of its Appendix III, which differs from
// this table's truncation by under an arc second.
func TestHeliocentricMeeusExample(t *testing.T) {
	t.Parallel()
	// 0h TD is TT - UTC, 59.184 s in late 1992, before 0h UTC.
	when := time.Date(1992, 12, 19, 23, 59, 0, 816000000, time.UTC)
	h, err := planet.Heliocentric(planet.Venus, when)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(h.Lon-26.11428) > 0.001 || math.Abs(h.Lat+2.62070) > 0.001 || math.Abs(h.DistanceAU-0.724603) > 2e-6 {
		t.Errorf("Venus = %+v, want L 26.11428, B -2.62070, R 0.724603", h)
	}
}

// TestHeliocentricEarthOppositeSun checks that Earth's heliocentric vector
// points away from the Sun's geocentric direction, and that differencing two
// heliocentric vectors gives the geometric view of one planet from another.
func TestHeliocentricEarthOppositeSun(t *testing.T) {
	t.Parallel()
	when := time.Date(2027, 3, 20, 0, 0, 0, 0, time.UTC)
	e, err := planet.Heliocentric(planet.Earth, when)
	if err != nil {
		t.Fatal(err)
	}
	if e.DistanceAU < 0.98 || e.DistanceAU > 1.02 {
		t.Errorf("Earth distance %.4f au", e.DistanceAU)
	}
	// Near the March equinox the Sun is at longitude 0, so the Earth is at 180.
	if d := math.Abs(e.Lon - 180); d > 1 {
		t.Errorf("Earth longitude %.3f at the March equinox, want about 180", e.Lon)
	}
	m, _ := planet.Heliocentric(planet.Mars, when)
	ev, mv := e.Vector(), m.Vector()
	dx, dy, dz := mv[0]-ev[0], mv[1]-ev[1], mv[2]-ev[2]
	d := math.Sqrt(dx*dx + dy*dy + dz*dz)
	r, err := planet.Position(greenwich, planet.Mars, when)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(d-r.DistanceAU) > 1e-3 {
		t.Errorf("Mars-Earth vector length %.5f au, Position distance %.5f", d, r.DistanceAU)
	}
}

// TestRiseTransitSetAgainstHorizons checks rise, transit, and set for four
// planets at Greenwich against JPL Horizons. Horizons stamps each event with
// the one-minute step at or after it, so the library's instant falls up to a
// minute before the stamp.
func TestRiseTransitSetAgainstHorizons(t *testing.T) {
	t.Parallel()
	f, err := os.Open("testdata/horizons-planets-rts.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	checked := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p := strings.Fields(line)
		want, err := time.Parse(time.RFC3339, p[2])
		if err != nil {
			t.Fatal(err)
		}
		b := byName[p[0]]
		from := want.Add(-3 * time.Hour)
		var got time.Time
		var ok bool
		switch p[1] {
		case "r":
			got, ok, err = planet.NextRise(greenwich, b, from)
		case "s":
			got, ok, err = planet.NextSet(greenwich, b, from)
		case "t":
			got, ok, err = planet.NextTransit(greenwich, b, from)
		}
		if err != nil || !ok {
			t.Fatalf("%s %s: %v %v", p[0], p[1], ok, err)
		}
		if d := got.Sub(want); d < -65*time.Second || d > 5*time.Second {
			t.Errorf("%s %s: %s, Horizons %s (off %v)", p[0], p[1], got.Format(time.RFC3339), p[2], d.Round(time.Second))
		}
		t.Logf("%s %s off Horizons by %v", p[0], p[1], got.Sub(want).Round(time.Second))
		checked++
	}
	if checked < 20 {
		t.Fatalf("only %d events checked", checked)
	}
}

// TestInvalidInput checks the coded errors for a bad observer or body.
func TestInvalidInput(t *testing.T) {
	t.Parallel()
	when := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	if _, err := planet.Position(astronomy.Observer{Lat: 91}, planet.Mars, when); !errors.Is(err, planet.ErrInvalidLatitude) {
		t.Errorf("latitude 91: %v", err)
	}
	if _, err := planet.Position(astronomy.Observer{Lng: 181}, planet.Mars, when); !errors.Is(err, planet.ErrInvalidLongitude) {
		t.Errorf("longitude 181: %v", err)
	}
	for _, b := range []planet.Body{planet.Earth, 0, 99} {
		_, err := planet.Position(greenwich, b, when)
		code, _ := apperr.Code(err)
		if !errors.Is(err, planet.ErrInvalidBody) || code != astronomy.CodeInvalidBody {
			t.Errorf("body %d: %v", int(b), err)
		}
		if _, _, err := planet.NextRise(greenwich, b, when); !errors.Is(err, planet.ErrInvalidBody) {
			t.Errorf("NextRise body %d: %v", int(b), err)
		}
	}
	if _, err := planet.Heliocentric(99, when); !errors.Is(err, planet.ErrInvalidBody) {
		t.Errorf("Heliocentric(99): %v", err)
	}
	if planet.Body(99).String() != "unknown" || planet.Mars.String() != "mars" {
		t.Errorf("String: %q %q", planet.Body(99), planet.Mars)
	}
}

// TestNearSun checks the flag follows the elongation and the per-body limit.
func TestNearSun(t *testing.T) {
	t.Parallel()
	for h := 0; h < 400*24; h += 97 {
		when := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC).Add(time.Duration(h) * time.Hour)
		for _, b := range planet.Bodies {
			r, err := planet.Position(greenwich, b, when)
			if err != nil {
				t.Fatal(err)
			}
			if r.NearSun != (r.Elongation < planet.NearSunElongation(b)) {
				t.Fatalf("%s at %s: NearSun %v with elongation %.2f", b, when, r.NearSun, r.Elongation)
			}
		}
	}
}
