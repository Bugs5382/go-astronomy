package moon_test

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
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/earth/moon"
)

// Tolerances for the Sun and Moon against JPL Horizons DE441 (issues 45 and
// 37). The Moon runs on the ELP 2000-82B series and is observed within 1.5 arc
// seconds here (the chapter 47 series it replaced was off by up to 6.2), and
// the Sun on VSOP87 within 1.4. The closest approach is observed within 3.2 s
// (it was 9.3 s with chapter 47).
const (
	moonToleranceArcsec = 3
	sunToleranceArcsec  = 5
	closestApproachTol  = 5 * time.Second
)

// eclipseRow is one ten-second sample of a committed Horizons eclipse fixture.
type eclipseRow struct {
	when                         time.Time
	sunRA, sunDec, sunAz, sunEl  float64
	moonRA, moonDec, moonAz, mEl float64
}

// eclipseFixture is the observer and the rows of one fixture file.
type eclipseFixture struct {
	name string
	obs  astronomy.Observer
	rows []eclipseRow
}

// loadEclipseFixtures reads every testdata/horizons-eclipse-*.txt file. Each
// header records the Horizons queries, the DE441 source, and the fetch date.
func loadEclipseFixtures(t *testing.T) []eclipseFixture {
	t.Helper()
	files, err := filepath.Glob("testdata/horizons-eclipse-*.txt")
	if err != nil || len(files) == 0 {
		t.Fatalf("no eclipse fixtures (%v)", err)
	}
	var out []eclipseFixture
	for _, name := range files {
		out = append(out, loadEclipseFixture(t, name))
	}
	return out
}

func loadEclipseFixture(t *testing.T, name string) eclipseFixture {
	t.Helper()
	f, err := os.Open(name)
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	defer func() { _ = f.Close() }()
	fx := eclipseFixture{name: filepath.Base(name)}
	num := func(s string) float64 {
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		return v
	}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		p := strings.Fields(line)
		if p[0] == "observer" {
			fx.obs = astronomy.Observer{Lat: num(p[1]), Lng: num(p[2]), TZ: time.UTC}
			continue
		}
		if len(p) != 9 {
			t.Fatalf("%s: row %q has %d fields, want 9", name, line, len(p))
		}
		when, err := time.Parse(time.RFC3339, p[0])
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		fx.rows = append(fx.rows, eclipseRow{
			when:  when,
			sunRA: num(p[1]), sunDec: num(p[2]), sunAz: num(p[3]), sunEl: num(p[4]),
			moonRA: num(p[5]), moonDec: num(p[6]), moonAz: num(p[7]), mEl: num(p[8]),
		})
	}
	if err := sc.Err(); err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	if len(fx.rows) < 3 {
		t.Fatalf("%s: too few rows", name)
	}
	return fx
}

// separation returns the angle, in degrees, between two directions given as
// (longitude-like, latitude-like) pairs in degrees: RA/Dec or azimuth/altitude.
func separation(lon1, lat1, lon2, lat2 float64) float64 {
	const r = math.Pi / 180
	a := math.Sin(lat1*r)*math.Sin(lat2*r) + math.Cos(lat1*r)*math.Cos(lat2*r)*math.Cos((lon1-lon2)*r)
	return math.Acos(math.Max(-1, math.Min(1, a))) / r
}

// horizonsClosestApproach returns the instant of least Sun-Moon separation in
// the fixture, refined between samples by a parabola through the smallest
// sample and its neighbours.
func horizonsClosestApproach(fx eclipseFixture) time.Time {
	sep := make([]float64, len(fx.rows))
	best := 0
	for i, r := range fx.rows {
		sep[i] = separation(r.sunRA, r.sunDec, r.moonRA, r.moonDec)
		if sep[i] < sep[best] {
			best = i
		}
	}
	if best == 0 || best == len(sep)-1 {
		return fx.rows[best].when
	}
	a, b, c := sep[best-1], sep[best], sep[best+1]
	step := fx.rows[best+1].when.Sub(fx.rows[best].when)
	offset := 0.5 * (a - c) / (a - 2*b + c)
	return fx.rows[best].when.Add(time.Duration(offset * float64(step)))
}

// libraryClosestApproach returns the instant of least Sun-Moon separation the
// library computes over the fixture's window, sampled every second.
func libraryClosestApproach(t *testing.T, fx eclipseFixture) time.Time {
	t.Helper()
	from, to := fx.rows[0].when, fx.rows[len(fx.rows)-1].when
	var best time.Time
	bestSep := math.Inf(1)
	for when := from; !when.After(to); when = when.Add(time.Second) {
		sun := earth.SunPosition(fx.obs, when)
		m, err := moon.Position(fx.obs, when)
		if err != nil {
			t.Fatalf("Position: %v", err)
		}
		if s := separation(sun.Azimuth, sun.Altitude, m.Azimuth, m.Altitude); s < bestSep {
			best, bestSep = when, s
		}
	}
	return best
}

// TestEclipseClosestApproach checks the instant of greatest eclipse, the least
// topocentric Sun-Moon separation, against JPL Horizons DE441 for four solar
// eclipses (issue 45). v1.0.0 ran 58 to 154 seconds late here.
func TestEclipseClosestApproach(t *testing.T) {
	t.Parallel()
	for _, fx := range loadEclipseFixtures(t) {
		want := horizonsClosestApproach(fx)
		got := libraryClosestApproach(t, fx)
		if d := got.Sub(want); d > closestApproachTol || d < -closestApproachTol {
			t.Errorf("%s: closest approach %s, Horizons %s (off %v)", fx.name,
				got.Format("15:04:05"), want.Format("15:04:05.0"), d)
		} else {
			t.Logf("%s: closest approach off Horizons by %v", fx.name, d)
		}
	}
}

// TestPositionsAgainstHorizons checks the Sun's and the Moon's airless
// topocentric altitude and azimuth against JPL Horizons DE441 at the middle of
// each eclipse window, where both bodies are well above the horizon.
func TestPositionsAgainstHorizons(t *testing.T) {
	t.Parallel()
	for _, fx := range loadEclipseFixtures(t) {
		r := fx.rows[len(fx.rows)/2]
		sun := earth.SunPosition(fx.obs, r.when)
		if d := separation(sun.Azimuth, sun.Altitude, r.sunAz, r.sunEl) * 3600; d > sunToleranceArcsec {
			t.Errorf("%s %s: Sun off Horizons by %.1f arcsec", fx.name, r.when.Format("15:04:05"), d)
		} else {
			t.Logf("%s: Sun off Horizons by %.1f arcsec", fx.name, d)
		}
		m, err := moon.Position(fx.obs, r.when)
		if err != nil {
			t.Fatalf("Position: %v", err)
		}
		if d := separation(m.Azimuth, m.Altitude, r.moonAz, r.mEl) * 3600; d > moonToleranceArcsec {
			t.Errorf("%s %s: Moon off Horizons by %.1f arcsec", fx.name, r.when.Format("15:04:05"), d)
		} else {
			t.Logf("%s: Moon off Horizons by %.1f arcsec", fx.name, d)
		}
	}
}
