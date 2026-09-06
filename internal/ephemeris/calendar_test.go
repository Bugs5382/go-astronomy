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

// TestGregorianToJD anchors the calendar-to-Julian-day conversion against the
// worked example and the check table of Meeus, Astronomical Algorithms,
// chapter 7 (example 7.a and the list on p. 62).
func TestGregorianToJD(t *testing.T) {
	t.Parallel()
	cases := []struct {
		y, m int
		d    float64
		want float64
	}{
		{1957, 10, 4.81, 2436116.31}, // example 7.a, Sputnik 1
		{2000, 1, 1.5, 2451545.0},
		{1999, 1, 1.0, 2451179.5},
		{1987, 1, 27.0, 2446822.5},
		{1987, 6, 19.5, 2446966.0},
		{1988, 1, 27.0, 2447187.5},
		{1988, 6, 19.5, 2447332.0},
		{1900, 1, 1.0, 2415020.5},
		{1600, 1, 1.0, 2305447.5},
		{1600, 12, 31.0, 2305812.5},
	}
	for _, c := range cases {
		if got := GregorianToJD(c.y, c.m, c.d); math.Abs(got-c.want) > 1e-6 {
			t.Errorf("GregorianToJD(%d, %d, %v) = %.6f, want %.6f", c.y, c.m, c.d, got, c.want)
		}
	}
}

// TestJDToGregorian anchors the inverse against Meeus example 7.c
// (JD 2436116.31 is 1957 October 4.81) and the same check table.
func TestJDToGregorian(t *testing.T) {
	t.Parallel()
	cases := []struct {
		jd   float64
		y, m int
		d    float64
	}{
		{2436116.31, 1957, 10, 4.81},
		{2451545.0, 2000, 1, 1.5},
		{2446822.5, 1987, 1, 27.0},
		{2415020.5, 1900, 1, 1.0},
		{2305447.5, 1600, 1, 1.0},
		{2305812.5, 1600, 12, 31.0},
	}
	for _, c := range cases {
		y, m, d := JDToGregorian(c.jd)
		if y != c.y || m != c.m || math.Abs(d-c.d) > 1e-6 {
			t.Errorf("JDToGregorian(%v) = %d-%02d-%v, want %d-%02d-%v",
				c.jd, y, m, d, c.y, c.m, c.d)
		}
	}
}

func TestTimeToJD(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   time.Time
		want float64
	}{
		{time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC), 2451545.0},
		{time.Date(1987, 4, 10, 0, 0, 0, 0, time.UTC), 2446895.5},       // Meeus 22.a
		{time.Date(1987, 4, 10, 19, 21, 0, 0, time.UTC), 2446896.30625}, // Meeus 12.b
		{time.Date(1992, 4, 12, 0, 0, 0, 0, time.UTC), 2448724.5},       // Meeus 47.a
		{time.Date(1992, 10, 13, 0, 0, 0, 0, time.UTC), 2448908.5},      // Meeus 25.a
	}
	for _, c := range cases {
		if got := TimeToJD(c.in); math.Abs(got-c.want) > 1e-9 {
			t.Errorf("TimeToJD(%v) = %.9f, want %.9f", c.in, got, c.want)
		}
	}
}

// TestTimeToJDIgnoresZone confirms a zoned instant converts by its UTC value,
// not by its wall clock.
func TestTimeToJDIgnoresZone(t *testing.T) {
	t.Parallel()
	loc := time.FixedZone("UTC-5", -5*3600)
	if got := TimeToJD(time.Date(2000, 1, 1, 7, 0, 0, 0, loc)); math.Abs(got-2451545.0) > 1e-9 {
		t.Errorf("TimeToJD of zoned time = %.9f, want 2451545.0", got)
	}
}

func TestJDToTime(t *testing.T) {
	t.Parallel()
	cases := []struct {
		jd   float64
		want time.Time
	}{
		{2451545.0, time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC)},
		{2446895.5, time.Date(1987, 4, 10, 0, 0, 0, 0, time.UTC)},
		{2448724.5, time.Date(1992, 4, 12, 0, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		got := JDToTime(c.jd)
		if d := got.Sub(c.want); d > time.Millisecond || d < -time.Millisecond {
			t.Errorf("JDToTime(%v) = %v, want %v", c.jd, got.UTC(), c.want)
		}
		if got.Location() != time.UTC {
			t.Errorf("JDToTime(%v) location = %v, want UTC", c.jd, got.Location())
		}
	}
}

// TestJDToTimeMeeusPhaseExample checks the conversion at the instant Meeus
// example 49.a produces: the New Moon of 1977 February 18 at JDE 2443192.65118,
// which is 1977 February 18 at 3h37m42s.
func TestJDToTimeMeeusPhaseExample(t *testing.T) {
	t.Parallel()
	got := JDToTime(2443192.65118)
	want := time.Date(1977, 2, 18, 3, 37, 42, 0, time.UTC)
	if d := got.Sub(want); d > time.Second || d < -time.Second {
		t.Errorf("JDToTime(2443192.65118) = %v, want %v", got.UTC(), want)
	}
}

// TestJDTimeRoundTrip walks a range of instants through both conversions.
func TestJDTimeRoundTrip(t *testing.T) {
	t.Parallel()
	base := time.Date(1970, 3, 7, 4, 15, 22, 0, time.UTC)
	for i := 0; i < 400; i++ {
		when := base.AddDate(0, 0, i*37)
		back := JDToTime(TimeToJD(when))
		if d := back.Sub(when); d > time.Millisecond || d < -time.Millisecond {
			t.Errorf("round trip %v -> %v", when, back.UTC())
		}
	}
}
