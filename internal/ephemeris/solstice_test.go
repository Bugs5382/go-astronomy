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
	"math"
	"testing"
	"time"
)

// TestJuneSolsticeMeeusExample anchors the equinox and solstice series against
// Meeus, Astronomical Algorithms, example 27.a: the series puts the June
// solstice of 1962 at JDE 2437837.39245. The refinement on VSOP87 then moves it
// by under a minute, onto the crossing itself.
func TestJuneSolsticeMeeusExample(t *testing.T) {
	t.Parallel()
	series := seasonBoundary(1962, juneAncient, juneModern)
	if math.Abs(series-2437837.39245) > 1e-5 {
		t.Errorf("series JuneSolstice(1962) = %.5f, want 2437837.39245", series)
	}
	if got := JuneSolstice(1962); math.Abs(got-series) > 1.0/1440 {
		t.Errorf("refined JuneSolstice(1962) = %.5f, more than a minute from the series", got)
	}
	lon, _, _ := SunApparentEcliptic(JuneSolstice(1962))
	if d := math.Abs(lon - 90); d > 0.01/arcsecPerDeg {
		t.Errorf("apparent longitude at the refined solstice = %.7f, want 90", lon)
	}
}

// TestSeasonBoundaryOrder checks the four boundaries of a year come out in
// calendar order and roughly a quarter of a year apart.
func TestSeasonBoundaryOrder(t *testing.T) {
	t.Parallel()
	for year := 1900; year <= 2100; year += 7 {
		b := [4]float64{
			MarchEquinox(year),
			JuneSolstice(year),
			SeptemberEquinox(year),
			DecemberSolstice(year),
		}
		for i := 1; i < 4; i++ {
			gap := b[i] - b[i-1]
			if gap < 88 || gap > 95 {
				t.Errorf("%d: boundary gap %d is %.3f days, want 88 to 95", year, i, gap)
			}
		}
		// Each boundary must land in its own month.
		wantMonth := [4]int{3, 6, 9, 12}
		for i, jde := range b {
			_, m, _ := JDToGregorian(jde)
			if m != wantMonth[i] {
				t.Errorf("%d: boundary %d fell in month %d, want %d", year, i, m, wantMonth[i])
			}
		}
	}
}

// TestSeasonBoundaryYearLength checks successive March equinoxes are one
// tropical year apart, 365.2422 days, to within a few minutes.
func TestSeasonBoundaryYearLength(t *testing.T) {
	t.Parallel()
	for year := 1950; year < 2050; year++ {
		length := MarchEquinox(year+1) - MarchEquinox(year)
		if math.Abs(length-365.2422) > 0.01 {
			t.Errorf("%d to %d is %.5f days, want ~365.2422", year, year+1, length)
		}
	}
}

// ephemerisSeasonBoundaries are the instants at which the Sun's apparent
// geocentric ecliptic longitude of date reaches 0, 90, 180, and 270 degrees,
// interpolated from the JPL Horizons system at half-hour steps. Horizons
// reports UTC; these are therefore the civil instants of the equinoxes and
// solstices, independent of this library and of the book.
var ephemerisSeasonBoundaries = []struct {
	name string
	fn   func(int) float64
	year int
	utc  time.Time
}{
	{"March equinox 2026", MarchEquinox, 2026, time.Date(2026, 3, 20, 14, 46, 0, 0, time.UTC)},
	{"June solstice 2026", JuneSolstice, 2026, time.Date(2026, 6, 21, 8, 24, 32, 0, time.UTC)},
	{"September equinox 2026", SeptemberEquinox, 2026, time.Date(2026, 9, 23, 0, 5, 14, 0, time.UTC)},
	{"December solstice 2026", DecemberSolstice, 2026, time.Date(2026, 12, 21, 20, 50, 15, 0, time.UTC)},
	{"June solstice 1962", JuneSolstice, 1962, time.Date(1962, 6, 21, 21, 24, 6, 0, time.UTC)},
}

// TestSeasonBoundariesAgainstEphemeris measures the season boundaries against a
// modern numerical ephemeris.
//
// The instants are refined on the VSOP87 Sun and converted from dynamical
// time to UTC with the leap-second table (issue 45). The Sun moves an arc
// second in about 24 s, so its 0.14 arc second error against DE441 is worth
// about 3 s here, and the fixtures are interpolated from half-hour steps. The
// observed error is 3.1 s at most, and the tolerance is 5 s.
func TestSeasonBoundariesAgainstEphemeris(t *testing.T) {
	t.Parallel()
	const tol = 5 * time.Second
	for _, c := range ephemerisSeasonBoundaries {
		got := JDEToTime(c.fn(c.year))
		if d := got.Sub(c.utc); d > tol || d < -tol {
			t.Errorf("%s = %v, ephemeris %v (%v off)", c.name, got.UTC(), c.utc, d)
		}
	}
}

// TestSeasonBoundariesOldEpoch exercises the branch that uses the coefficients
// for years before 1000, where the series is expressed in the year itself
// rather than in the offset from 2000.
func TestSeasonBoundariesOldEpoch(t *testing.T) {
	t.Parallel()
	for _, year := range []int{-500, 0, 500, 999} {
		b := [4]float64{
			MarchEquinox(year),
			JuneSolstice(year),
			SeptemberEquinox(year),
			DecemberSolstice(year),
		}
		for i := 1; i < 4; i++ {
			if gap := b[i] - b[i-1]; gap < 88 || gap > 95 {
				t.Errorf("%d: boundary gap %d is %.3f days", year, i, gap)
			}
		}
		if y, _, _ := JDToGregorian(b[0]); y != year {
			t.Errorf("MarchEquinox(%d) landed in year %d", year, y)
		}
	}
}

// TestSeasonBoundaryEpochSeam checks the two coefficient sets agree where they
// meet, at the year 1000: the difference must be far below the accuracy of the
// series itself.
func TestSeasonBoundaryEpochSeam(t *testing.T) {
	t.Parallel()
	// The year 1000 uses the modern coefficients and 999 the ancient set, so
	// the seam shows up as a discontinuity in the year length across it.
	length := MarchEquinox(1000) - MarchEquinox(999)
	if math.Abs(length-365.2423) > 0.01 {
		t.Errorf("year length across the coefficient seam is %.5f days", length)
	}
}
