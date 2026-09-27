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

// TestTTMinusUTC anchors the time-scale offset. From 1972 it is exact: 32.184 s
// plus TAI-UTC from the IERS Bulletin C leap-second table. Before 1972 it is
// the Espenak and Meeus (2006) Delta-T polynomial, which gives -2.79 s at the
// start of 1900 and 29.07 s at the start of 1950.
func TestTTMinusUTC(t *testing.T) {
	t.Parallel()
	cases := []struct {
		when time.Time
		want float64
		tol  float64
	}{
		{time.Date(2017, 1, 1, 0, 0, 0, 0, time.UTC), 69.184, 1e-9},
		{time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC), 69.184, 1e-9},
		{time.Date(2016, 12, 31, 23, 59, 59, 0, time.UTC), 68.184, 1e-9},
		{time.Date(1999, 1, 1, 0, 0, 0, 0, time.UTC), 64.184, 1e-9},
		{time.Date(1998, 12, 31, 23, 59, 59, 0, time.UTC), 63.184, 1e-9},
		{time.Date(1972, 7, 1, 0, 0, 0, 0, time.UTC), 43.184, 1e-9},
		{time.Date(1972, 1, 1, 0, 0, 0, 0, time.UTC), 42.184, 1e-9},
		{time.Date(1950, 1, 1, 0, 0, 0, 0, time.UTC), 29.07, 1},
		{time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC), -2.79, 1},
		{time.Date(1820, 1, 1, 0, 0, 0, 0, time.UTC), 12, 2},
	}
	for _, c := range cases {
		if got := TTMinusUTC(c.when); math.Abs(got-c.want) > c.tol {
			t.Errorf("TTMinusUTC(%s) = %.3f s, want %.3f", c.when.Format(time.DateTime), got, c.want)
		}
	}
}

// TestTTMinusUTCContinuity checks that the polynomial hands over to the
// leap-second table without a jump larger than the table's own one-second
// steps, and that the polynomial pieces meet each other.
func TestTTMinusUTCContinuity(t *testing.T) {
	t.Parallel()
	before := TTMinusUTC(time.Date(1971, 12, 31, 23, 59, 59, 0, time.UTC))
	after := TTMinusUTC(time.Date(1972, 1, 1, 0, 0, 0, 0, time.UTC))
	if math.Abs(after-before) > 1 {
		t.Errorf("jump at 1972: %.3f to %.3f s", before, after)
	}
	for _, y := range []int{1600, 1700, 1800, 1860, 1900, 1920, 1941, 1961} {
		a := TTMinusUTC(time.Date(y-1, 12, 31, 0, 0, 0, 0, time.UTC))
		b := TTMinusUTC(time.Date(y, 1, 2, 0, 0, 0, 0, time.UTC))
		if math.Abs(a-b) > 1 {
			t.Errorf("jump at %d: %.3f to %.3f s", y, a, b)
		}
	}
}

// TestTTRoundTrip checks that converting to a Julian ephemeris day and back is
// the identity to well under a millisecond, across a leap second.
func TestTTRoundTrip(t *testing.T) {
	t.Parallel()
	for _, when := range []time.Time{
		time.Date(2024, 4, 8, 18, 42, 39, 0, time.UTC),
		time.Date(2016, 12, 31, 23, 59, 30, 0, time.UTC),
		time.Date(1950, 6, 1, 12, 0, 0, 0, time.UTC),
	} {
		jde := TimeToJDE(when)
		if d := jde - TimeToJD(when); math.Abs(d*86400-TTMinusUTC(when)) > 1e-3 {
			t.Errorf("JDE - JD at %s = %.6f s, want %.6f", when, d*86400, TTMinusUTC(when))
		}
		if back := JDEToTime(jde); back.Sub(when).Abs() > time.Millisecond {
			t.Errorf("round trip of %s gave %s", when, back)
		}
	}
}
