package star_test

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

	"github.com/Bugs5382/go-astronomy/star"
)

// TestPositionAtAgainstSOFA checks the space motion against the IAU SOFA
// library (ERFA pmsafe, testdata/erfa-stars.txt): twenty stars, including
// Barnard's Star, Kapteyn's Star, and Groombridge 1830, from 1950 to 2100,
// agree to 0.005 arc second.
func TestPositionAtAgainstSOFA(t *testing.T) {
	t.Parallel()
	f, err := os.Open("testdata/erfa-stars.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	rows := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "#") {
			continue
		}
		p := strings.Split(line, "|")
		s, ok := star.GetNamedStar(p[0])
		if !ok {
			t.Fatalf("no %q", p[0])
		}
		when, _ := time.Parse(time.RFC3339, p[1])
		wantRA, _ := strconv.ParseFloat(p[2], 64)
		wantDec, _ := strconv.ParseFloat(p[3], 64)
		ra, dec := s.PositionAt(when)
		if d := sep(ra, dec, wantRA, wantDec); d > 0.005 {
			t.Errorf("%s %s: %.4f arcsec off SOFA", p[0], p[1], d)
		}
		rows++
	}
	if rows < 100 {
		t.Fatalf("only %d rows", rows)
	}
}

// TestPositionAtMotion checks the catalog's motion columns and the motion
// itself: Barnard's Star, the fastest, moves 10.4 arc seconds a year (its
// Hipparcos declination motion, which HYG caps at 9999.99, is restored); a
// star with no known motion stays put; and J2000.0 is the catalog place.
func TestPositionAtMotion(t *testing.T) {
	t.Parallel()
	barnard, ok := star.GetNamedStar("Barnard's Star")
	if !ok {
		t.Fatal("no Barnard's Star")
	}
	if barnard.PMDec != 10326.93 || barnard.PMRA != -797.84 || barnard.RadialVelocity != -111 {
		t.Errorf("Barnard's Star motion %+v", barnard)
	}
	epoch := time.Date(2000, 1, 1, 11, 58, 55, 816000000, time.UTC) // J2000.0, 12:00 TT
	ra, dec := barnard.PositionAt(epoch)
	if d := sep(ra, dec, barnard.RA, barnard.Dec); d > 1e-6 {
		t.Errorf("at J2000.0 Barnard's Star is %.2e arcsec off its catalog place", d)
	}
	ra, dec = barnard.PositionAt(epoch.AddDate(1, 0, 0))
	if d := sep(ra, dec, barnard.RA, barnard.Dec); math.Abs(d-10.36) > 0.05 {
		t.Errorf("Barnard's Star moved %.2f arcsec in a year, want 10.36", d)
	}

	still := star.Star{RA: 12.3, Dec: -45.6}
	if ra, dec := still.PositionAt(epoch.AddDate(80, 0, 0)); ra != 12.3 || dec != -45.6 {
		t.Errorf("a star with no motion moved to %v, %v", ra, dec)
	}
	moving := 0
	for _, s := range star.All() {
		if s.PMRA != 0 || s.PMDec != 0 {
			moving++
		}
	}
	if moving < star.Count()*9/10 {
		t.Errorf("only %d of %d stars carry a proper motion", moving, star.Count())
	}
}

func sep(ra1, dec1, ra2, dec2 float64) float64 {
	const r = math.Pi / 180
	c := math.Sin(dec1*r)*math.Sin(dec2*r) + math.Cos(dec1*r)*math.Cos(dec2*r)*math.Cos((ra1-ra2)*r)
	return math.Acos(math.Max(-1, math.Min(1, c))) / r * 3600
}
