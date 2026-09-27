package sky

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
	"os/exec"
	"strings"
	"testing"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/earth/moon"
	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
	"github.com/Bugs5382/go-astronomy/internal/julian"
	"github.com/Bugs5382/go-astronomy/internal/planettest"
	"github.com/Bugs5382/go-astronomy/planet/mars"
)

// TestGenericTransformOnEarth checks the general transform against the
// Earth pipeline it stands beside: placing the Sun, the Moon, and Mars from
// an Earth site through the general code, not the routed v1 packages, agrees
// with earth.SunPosition, moon.Position, and mars.Position to 2 arc seconds
// (it measures under 1). It is the same code every other body's sky goes
// through, with the Earth's own orientation (precession, nutation, apparent
// sidereal time) in place of the IAU Earth model.
func TestGenericTransformOnEarth(t *testing.T) {
	t.Parallel()
	obs := astronomy.Observer{Lat: 39.74, Lng: -104.99, TZ: time.UTC}
	site := Site{Body: Earth, Lat: obs.Lat, Lon: obs.Lng}
	marsBody, err := Planet(mars.Planet)
	if err != nil {
		t.Fatal(err)
	}
	for h := 0; h < 48; h += 3 {
		when := time.Date(2027, 3, 1, h, 0, 0, 0, time.UTC)
		jde := julian.TT(when)
		sun := earth.SunPosition(obs, when)
		mp, _ := moon.Position(obs, when)
		pr, _ := mars.Position(obs, when)
		for name, c := range map[string]struct {
			target  *Body
			alt, az float64
		}{
			"sun":  {Sun, sun.Altitude, sun.Azimuth},
			"moon": {Moon, mp.Altitude, mp.Azimuth},
			"mars": {marsBody, pr.Altitude, pr.Azimuth},
		} {
			l := site.look(c.target, jde)
			if d := sep(l.alt, l.az, c.alt, c.az); d > 2 {
				t.Errorf("%s at %s: general transform off the v1 place by %.1f arcsec", name, when.Format(time.RFC3339), d)
			}
		}
	}
}

// TestPrecessionMatchesMeeus checks the rectangular precession against the
// spherical Meeus 21.4 the ephemeris package implements.
func TestPrecessionMatchesMeeus(t *testing.T) {
	t.Parallel()
	for _, jde := range []float64{2451545, 2462502.5, 2415020, 2488069.5} {
		m := ofDateToJ2000(jde)
		eps := ephemeris.MeanObliquity(jde)
		for _, p := range [][2]float64{{0, 0}, {123.4, 5.6}, {280, -1.2}} {
			ra, dec := ephemeris.EclToEq(p[0], p[1], eps)
			year := 2000 + (jde-2451545)/365.25
			wantRA, wantDec := ephemeris.PrecessEq(ra, dec, year, 2000)
			v := m.apply(spherical(p[0], p[1], 1))
			gotRA := math.Mod(math.Atan2(v[1], v[0])/radPerDeg+360, 360)
			gotDec := math.Asin(v[2]) / radPerDeg
			if d := math.Hypot((gotRA-wantRA)*math.Cos(gotDec*radPerDeg), gotDec-wantDec) * 3600; d > 0.05 {
				t.Errorf("jde %v (%v, %v): off Meeus by %.3f arcsec", jde, p[0], p[1], d)
			}
		}
	}
}

// TestLinksNoPlanetTable checks that importing sky links no planet package:
// a planet's table comes in only with the planet a program passes to Planet.
func TestLinksNoPlanetTable(t *testing.T) {
	t.Parallel()
	gobin, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("no go command: %v", err)
	}
	out, err := exec.Command(gobin, "list", "-deps", "-f", "{{.ImportPath}}", planettest.Module+"/sky").Output() // #nosec G204 -- fixed arguments.
	if err != nil {
		t.Fatalf("go list: %v", err)
	}
	for _, d := range strings.Fields(string(out)) {
		for _, p := range append(planettest.Planets, "all") {
			if d == planettest.Module+"/planet/"+p {
				t.Errorf("sky depends on planet/%s", p)
			}
		}
		if d == "net/http" {
			t.Error("sky links net/http")
		}
	}
}

func sep(alt1, az1, alt2, az2 float64) float64 {
	c := math.Sin(alt1*radPerDeg)*math.Sin(alt2*radPerDeg) +
		math.Cos(alt1*radPerDeg)*math.Cos(alt2*radPerDeg)*math.Cos((az1-az2)*radPerDeg)
	return math.Acos(clamp(c)) / radPerDeg * 3600
}
