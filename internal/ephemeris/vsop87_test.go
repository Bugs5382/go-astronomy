package ephemeris

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
)

// horizonsSun is one row of testdata/horizons-sun-geocentric.txt.
type horizonsSun struct {
	jde           float64
	ra, dec, dist float64
}

// loadHorizonsSun reads the committed Horizons fixture. Its header records the
// queries, the DE441 source, and the fetch date.
func loadHorizonsSun(t *testing.T) []horizonsSun {
	t.Helper()
	f, err := os.Open("testdata/horizons-sun-geocentric.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	var rows []horizonsSun
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p := strings.Fields(line)
		if len(p) != 4 {
			t.Fatalf("row %q has %d fields, want 4", line, len(p))
		}
		when, err := time.Parse("2006-01-02T15:04:05", p[0])
		if err != nil {
			t.Fatal(err)
		}
		var v [3]float64
		for i := range v {
			if v[i], err = strconv.ParseFloat(p[i+1], 64); err != nil {
				t.Fatal(err)
			}
		}
		// The instants are Terrestrial Time, so the calendar instant is the
		// Julian ephemeris day directly.
		rows = append(rows, horizonsSun{TimeToJD(when), v[0], v[1], v[2]})
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(rows) < 50 {
		t.Fatalf("only %d fixture rows", len(rows))
	}
	return rows
}

// sepArcsec is the angle between two RA/Dec directions, in arc seconds.
func sepArcsec(ra1, dec1, ra2, dec2 float64) float64 {
	c := math.Sin(radians(dec1))*math.Sin(radians(dec2)) +
		math.Cos(radians(dec1))*math.Cos(radians(dec2))*math.Cos(radians(ra1-ra2))
	return degrees(math.Acos(math.Min(1, c))) * arcsecPerDeg
}

// TestSunApparentAgainstHorizons measures the Sun's apparent place from the
// truncated VSOP87 series against JPL Horizons DE441 at 79 instants from 1900
// to 2100 (issue 45). The chapter 25 theory it replaces is off by up to 28 arc
// seconds over the same instants.
func TestSunApparentAgainstHorizons(t *testing.T) {
	t.Parallel()
	var worst, worstChapter25, worstDist float64
	for _, h := range loadHorizonsSun(t) {
		ra, dec, distKm := SunApparent(h.jde)
		d := sepArcsec(ra, dec, h.ra, h.dec)
		worst = math.Max(worst, d)
		if d > sunApparentTolArcsec {
			t.Errorf("JDE %.2f: Sun off Horizons by %.2f arcsec", h.jde, d)
		}
		dist := math.Abs(distKm/KmPerAU - h.dist)
		worstDist = math.Max(worstDist, dist)
		if dist > sunDistanceTolAU {
			t.Errorf("JDE %.2f: distance off Horizons by %.2e au", h.jde, dist)
		}
		ra25, dec25 := SolarApparentEquatorial(h.jde)
		worstChapter25 = math.Max(worstChapter25, sepArcsec(ra25, dec25, h.ra, h.dec))
	}
	t.Logf("worst: VSOP87 %.2f arcsec, %.2e au; chapter 25 %.1f arcsec", worst, worstDist, worstChapter25)
}

// The observed worst over the fixture is 0.14 arc seconds and 2.6e-7 au: the
// truncation of the series (about 0.13 arc seconds at most, 1700 to 2300) and
// the abridged nutation account for it.
const (
	sunApparentTolArcsec = 0.5
	sunDistanceTolAU     = 1e-6
)
