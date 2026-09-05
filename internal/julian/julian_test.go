package julian

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

func TestDate(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   time.Time
		want float64
		tol  float64
	}{
		{
			// J2000.0 standard epoch.
			name: "J2000",
			in:   time.Date(2000, 1, 1, 12, 0, 0, 0, time.UTC),
			want: 2451545.0,
			tol:  1e-9,
		},
		{
			// Meeus, Astronomical Algorithms, example 22.a reference instant.
			name: "1987 April 10 0h",
			in:   time.Date(1987, 4, 10, 0, 0, 0, 0, time.UTC),
			want: 2446895.5,
			tol:  1e-9,
		},
	}
	for _, c := range cases {
		if got := Date(c.in); math.Abs(got-c.want) > c.tol {
			t.Errorf("%s: Date = %.9f, want %.9f", c.name, got, c.want)
		}
	}
}

func TestDateUsesUTC(t *testing.T) {
	t.Parallel()
	loc := time.FixedZone("UTC-5", -5*3600)
	local := time.Date(2000, 1, 1, 7, 0, 0, 0, loc) // == 12:00 UTC
	if got := Date(local); math.Abs(got-2451545.0) > 1e-9 {
		t.Errorf("Date of zoned time = %.9f, want 2451545.0", got)
	}
}

func TestGreenwichSiderealTime(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   time.Time
		want float64 // degrees
	}{
		{
			// Meeus example 12.a: theta0 = 13h10m46.3668s.
			name: "1987 April 10 0h",
			in:   time.Date(1987, 4, 10, 0, 0, 0, 0, time.UTC),
			want: (13 + 10.0/60 + 46.3668/3600) * 15,
		},
		{
			// Meeus example 12.b: theta0 = 8h34m57.0896s.
			name: "1987 April 10 19:21",
			in:   time.Date(1987, 4, 10, 19, 21, 0, 0, time.UTC),
			want: (8 + 34.0/60 + 57.0896/3600) * 15,
		},
	}
	for _, c := range cases {
		got := GreenwichSiderealTime(c.in)
		if got < 0 || got >= 360 {
			t.Errorf("%s: GST = %v out of [0,360)", c.name, got)
		}
		if math.Abs(got-c.want) > 1e-3 {
			t.Errorf("%s: GST = %.6f deg, want %.6f deg", c.name, got, c.want)
		}
	}
}

func TestLocalSiderealTime(t *testing.T) {
	t.Parallel()
	when := time.Date(1987, 4, 10, 19, 21, 0, 0, time.UTC)
	gst := GreenwichSiderealTime(when)

	// At the prime meridian LST equals GST.
	if got := LocalSiderealTime(when, 0); math.Abs(got-gst) > 1e-9 {
		t.Errorf("LST at lon 0 = %v, want GST %v", got, gst)
	}

	// East-positive longitude advances local sidereal time.
	cases := []float64{-77.065556, -73.944, 0, 45, 151.2}
	for _, lon := range cases {
		got := LocalSiderealTime(when, lon)
		want := gst + lon
		for want < 0 {
			want += 360
		}
		for want >= 360 {
			want -= 360
		}
		if math.Abs(got-want) > 1e-9 {
			t.Errorf("LST(lon=%v) = %v, want %v", lon, got, want)
		}
		if got < 0 || got >= 360 {
			t.Errorf("LST(lon=%v) = %v out of [0,360)", lon, got)
		}
	}
}

func TestMeanObliquity(t *testing.T) {
	t.Parallel()
	// Meeus example 22.a: eps0 = 23d26'27.407" for 1987 April 10 0h.
	when := time.Date(1987, 4, 10, 0, 0, 0, 0, time.UTC)
	want := 23 + 26.0/60 + 27.407/3600
	got := MeanObliquity(when)
	if math.Abs(got-want) > 1e-4 {
		t.Errorf("MeanObliquity = %.7f deg, want %.7f deg", got, want)
	}

	// Sanity: obliquity is near 23.44 deg across the modern era.
	for _, y := range []int{1900, 1950, 2000, 2050, 2100} {
		e := MeanObliquity(time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC))
		if e < 23.0 || e > 24.0 {
			t.Errorf("MeanObliquity(%d) = %v, out of plausible range", y, e)
		}
	}
}
