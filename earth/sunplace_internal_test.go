package earth

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

	astronomy "github.com/Bugs5382/go-astronomy"
)

// TestInterpolatedSunMatchesDirect checks the hourly interpolation that
// resolving a day uses against the direct Sun position, over five years
// either side of 2026 (so every phase of the nutation's short terms and both
// ends of the orbit), at instants off the hour, and across the RA wrap at the
// March equinox: the place differs by under 1e-9 degrees (it measures
// 5e-10, two millionths of an arc second), and a whole hour gives the direct
// place exactly.
func TestInterpolatedSunMatchesDirect(t *testing.T) {
	t.Parallel()
	obs := astronomy.Observer{Lat: 39.74, Lng: -104.99, Height: astronomy.Meters(1609)}
	places := newSunPlaces()
	var worstPlace, worstAlt float64
	check := func(when time.Time) {
		direct, interp := sunPlaceAt(when), places.at(when)
		dra := math.Mod(direct.ra-interp.ra+540, 360) - 180
		worstPlace = math.Max(worstPlace, math.Max(math.Abs(dra), math.Abs(direct.dec-interp.dec)))
		a := SunPosition(obs, when)
		b := sunHorizontal(obs, when, interp)
		worstAlt = math.Max(worstAlt, math.Max(math.Abs(a.Altitude-b.Altitude), math.Abs(a.Azimuth-b.Azimuth)))
	}
	start := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	for d := 0; d < 3650; d += 7 {
		day := start.AddDate(0, 0, d)
		for m := 0; m < 24*60; m += 97 {
			check(day.Add(time.Duration(m)*time.Minute + 13*time.Second + 250*time.Millisecond))
		}
	}
	equinox := time.Date(2026, 3, 20, 14, 46, 0, 0, time.UTC) // RA passes 360 to 0
	for m := -180; m <= 180; m++ {
		check(equinox.Add(time.Duration(m) * time.Minute))
	}
	if worstPlace > 1e-9 || worstAlt > 1e-9 {
		t.Errorf("interpolation off the direct Sun by %.2e degrees in place, %.2e in alt/az", worstPlace, worstAlt)
	}
	hour := time.Date(2026, 7, 4, 13, 0, 0, 0, time.UTC)
	if places.at(hour) != sunPlaceAt(hour) {
		t.Error("a whole hour is not the direct place")
	}
	// Before 1972, TT - UTC comes from a Delta-T polynomial evaluated per
	// calendar day, so it steps by a few milliseconds at midnight and the
	// interpolation across it is a little looser.
	before1970 := time.Date(1969, 12, 31, 23, 30, 0, 0, time.UTC)
	if d := math.Abs(places.at(before1970).dec - sunPlaceAt(before1970).dec); d > 5e-9 {
		t.Errorf("before 1970 off by %.2e degrees", d)
	}
}

// TestInterpolatedDayMatchesDirect checks the day built on the interpolated
// Sun against the day built on direct Sun positions: the same bands with the
// same labels, every boundary within 0.1 s (the crossing search itself stops
// at 0.2 s), across a year at four latitudes.
func TestInterpolatedDayMatchesDirect(t *testing.T) {
	t.Parallel()
	for _, obs := range []astronomy.Observer{
		{Lat: 40.678, Lng: -73.944, TZ: time.UTC},
		{Lat: 39.74, Lng: -104.99, TZ: time.UTC, Height: astronomy.Meters(1609)},
		{Lat: 69.65, Lng: 18.96, TZ: time.UTC},
		{Lat: -33.9, Lng: 151.2, TZ: time.UTC},
	} {
		for d := 0; d < 365; d += 29 {
			date := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, d)
			seg := DefaultSegmentation.atHeight(obs.Height)
			fast := build(obs, date, seg, sunAltitudes(obs))
			direct := build(obs, date, seg, func(t time.Time) float64 { return SunPosition(obs, t).Altitude })
			if len(fast.segments) != len(direct.segments) || fast.polar != direct.polar {
				t.Fatalf("%v %s: %d bands (%v), direct %d (%v)", obs.Lat, date.Format(time.DateOnly),
					len(fast.segments), fast.polar, len(direct.segments), direct.polar)
			}
			for i := range fast.segments {
				f, g := fast.segments[i], direct.segments[i]
				if f.Label != g.Label || f.From.Sub(g.From).Abs() > 100*time.Millisecond || f.To.Sub(g.To).Abs() > 100*time.Millisecond {
					t.Errorf("%v %s band %d: %+v, direct %+v", obs.Lat, date.Format(time.DateOnly), i, f, g)
				}
			}
			if d := fast.noon.Sub(direct.noon).Abs(); d > 100*time.Millisecond {
				t.Errorf("%v %s: solar noon off by %v", obs.Lat, date.Format(time.DateOnly), d)
			}
		}
	}
}

// TestFineSunTrackMatchesDirect checks a track with more samples than hours,
// which interpolates, against SunPosition at the same instants, and that a
// coarse track is still SunPosition exactly.
func TestFineSunTrackMatchesDirect(t *testing.T) {
	t.Parallel()
	obs := astronomy.Observer{Lat: 51.48, Lng: 0, TZ: time.UTC}
	date := time.Date(2027, 3, 20, 0, 0, 0, 0, time.UTC)
	for _, s := range SunTrack(obs, date, 1441) {
		p := SunPosition(obs, s.Time)
		if math.Abs(p.Altitude-s.Altitude) > 1e-9 || math.Abs(p.Azimuth-s.Azimuth) > 1e-9 {
			t.Fatalf("%s: track %v/%v, direct %v/%v", s.Time, s.Altitude, s.Azimuth, p.Altitude, p.Azimuth)
		}
	}
	for _, s := range SunTrack(obs, date, trackInterpolateAbove) {
		if p := SunPosition(obs, s.Time); p.Altitude != s.Altitude || p.Azimuth != s.Azimuth {
			t.Fatalf("coarse track at %s is not SunPosition", s.Time)
		}
	}
}
