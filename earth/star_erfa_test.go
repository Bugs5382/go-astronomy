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
	"bufio"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
	"github.com/Bugs5382/go-astronomy/internal/julian"
)

// erfaStar is one row of star/testdata/erfa-stars.txt: a star at an instant,
// with the places the IAU SOFA library (through ERFA) gives for it.
type erfaStar struct {
	name                   string
	when                   time.Time
	meanRA, meanDec        float64 // J2000 equator, carried to the epoch
	apparentRA, apparentDe float64 // true equator and equinox of date
	az, alt                float64 // from 40.678 N, 73.944 W, no refraction
}

func loadERFAStars(t *testing.T) []erfaStar {
	t.Helper()
	f, err := os.Open("../star/testdata/erfa-stars.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var out []erfaStar
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}
		p := strings.Split(line, "|")
		when, err := time.Parse(time.RFC3339, p[1])
		if err != nil {
			t.Fatal(err)
		}
		var v [6]float64
		for i := range v {
			if v[i], err = strconv.ParseFloat(p[2+i], 64); err != nil {
				t.Fatal(err)
			}
		}
		out = append(out, erfaStar{p[0], when, v[0], v[1], v[2], v[3], v[4], v[5]})
	}
	if len(out) < 100 {
		t.Fatalf("only %d rows", len(out))
	}
	return out
}

// arcsecBetween returns the angle between two directions, in arc seconds.
func arcsecBetween(lon1, lat1, lon2, lat2 float64) float64 {
	const r = math.Pi / 180
	c := math.Sin(lat1*r)*math.Sin(lat2*r) + math.Cos(lat1*r)*math.Cos(lat2*r)*math.Cos((lon1-lon2)*r)
	return math.Acos(math.Max(-1, math.Min(1, c))) / r * 3600
}

// TestStarPositionAgainstSOFA checks the star chain against the IAU SOFA
// library (ERFA 2.0.1.5): twenty bright, near, and fast-moving stars, from
// Polaris to Barnard's Star and Alpha Centauri, at six instants from 1950 to
// 2100.
//
// The mean place carried along the space motion agrees to 0.005 arc second.
// The apparent place of date agrees to 0.1 arc second around the present and
// drifts by about a quarter of an arc second a century away from 2000: SOFA
// uses the IAU 2006 precession and 2000A nutation with the frame bias, and
// this library the IAU 1976 precession and 1980 nutation of Meeus, whose
// precession rate is known to be about 0.3 arc second a century off. The
// altitude and azimuth take up to another 0.35 arc second, the diurnal
// aberration of the observer's rotation, which SOFA applies and this library
// does not.
func TestStarPositionAgainstSOFA(t *testing.T) {
	t.Parallel()
	obs := astronomy.Observer{Lat: 40.678, Lng: -73.944}
	for _, r := range loadERFAStars(t) {
		s := mustStar(t, r.name)
		centuries := math.Abs(r.when.Sub(time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)).Hours()) / (24 * 36525)
		bound := 0.1 + 0.25*centuries

		ra, dec := s.PositionAt(r.when)
		if d := arcsecBetween(ra, dec, r.meanRA, r.meanDec); d > 0.005 {
			t.Errorf("%s %s: mean place off SOFA by %.4f arcsec", r.name, r.when.Format(time.DateOnly), d)
		}
		jde := julian.TT(r.when)
		dpsi, deps := ephemeris.Nutation(jde)
		ra, dec = ephemeris.ApparentFromJ2000(ra, dec, jde, dpsi, deps, s.Distance)
		if d := arcsecBetween(ra, dec, r.apparentRA, r.apparentDe); d > bound {
			t.Errorf("%s %s: apparent place off SOFA by %.3f arcsec, want %.3f", r.name, r.when.Format(time.DateOnly), d, bound)
		}

		hz, err := earth.StarPosition(s, obs, r.when)
		if err != nil {
			t.Fatal(err)
		}
		if d := arcsecBetween(hz.Azimuth, hz.Altitude, r.az, r.alt); d > bound+0.35 {
			t.Errorf("%s %s: alt/az off SOFA by %.3f arcsec, want %.3f", r.name, r.when.Format(time.DateOnly), d, bound+0.35)
		}
	}
}
