package satellite_test

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
	"github.com/Bugs5382/go-astronomy/satellite"
)

// TestPassesAgainstSkyfield checks rise, peak, set, peak altitude, and shadow
// entry and exit against Skyfield for the same element set, two observers,
// and two days (testdata/passes-skyfield.txt).
func TestPassesAgainstSkyfield(t *testing.T) {
	t.Parallel()
	e := iss(t)
	from := time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)
	to := from.Add(48 * time.Hour)
	got := map[string][]satellite.Pass{}
	rows := fields(t, "testdata/passes-skyfield.txt")
	parse := func(s string) time.Time {
		if s == "-" {
			return time.Time{}
		}
		v, err := time.Parse(time.RFC3339Nano, s)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	// Observed worst: 0.19 s on rise, peak, and set, 0.05 s on shadow
	// crossings, and 0.007 degrees of peak altitude.
	const tol = time.Second
	var worstT, worstShadow time.Duration
	var worstAlt float64
	defer func() {
		t.Logf("worst rise/peak/set %v, shadow %v, peak altitude %.4f degrees", worstT, worstShadow, worstAlt)
	}()
	near := func(a, b time.Time) bool {
		if a.IsZero() || b.IsZero() {
			return a.IsZero() && b.IsZero()
		}
		return a.Sub(b).Abs() <= tol
	}
	for _, r := range rows {
		site := r[0]
		if got[site] == nil {
			obs := astronomy.Observer{Lat: num(t, r[1]), Lng: num(t, r[2]), TZ: time.UTC}
			p, err := satellite.Passes(obs, e, from, to, satellite.DefaultPassOptions())
			if err != nil {
				t.Fatal(err)
			}
			got[site] = p
		}
		rise := parse(r[3])
		var match *satellite.Pass
		for i := range got[site] {
			if got[site][i].Rise.Time.Sub(rise).Abs() < time.Minute {
				match = &got[site][i]
			}
		}
		if match == nil {
			t.Errorf("%s: no pass rising near %s", site, r[3])
			continue
		}
		if !near(match.Rise.Time, rise) || !near(match.Peak.Time, parse(r[4])) || !near(match.Set.Time, parse(r[5])) {
			t.Errorf("%s %s: rise/peak/set %s %s %s, Skyfield %s %s %s", site, r[3],
				match.Rise.Time.Format(time.TimeOnly), match.Peak.Time.Format(time.TimeOnly), match.Set.Time.Format(time.TimeOnly), r[3], r[4], r[5])
		}
		worstT = max(worstT, match.Rise.Time.Sub(rise).Abs(), match.Peak.Time.Sub(parse(r[4])).Abs(), match.Set.Time.Sub(parse(r[5])).Abs())
		if !match.ShadowEntry.IsZero() && r[7] != "-" {
			worstShadow = max(worstShadow, match.ShadowEntry.Sub(parse(r[7])).Abs())
		}
		if !match.ShadowExit.IsZero() && r[8] != "-" {
			worstShadow = max(worstShadow, match.ShadowExit.Sub(parse(r[8])).Abs())
		}
		worstAlt = math.Max(worstAlt, math.Abs(match.Peak.Altitude-num(t, r[6])))
		if d := math.Abs(match.Peak.Altitude - num(t, r[6])); d > 0.02 {
			t.Errorf("%s %s: peak altitude %.4f, Skyfield %s", site, r[3], match.Peak.Altitude, r[6])
		}
		if !near(match.ShadowEntry, parse(r[7])) || !near(match.ShadowExit, parse(r[8])) {
			t.Errorf("%s %s: shadow entry/exit %s %s, Skyfield %s %s", site, r[3], match.ShadowEntry, match.ShadowExit, r[7], r[8])
		}
	}
	for site, p := range got {
		n := 0
		for _, r := range rows {
			if r[0] == site {
				n++
			}
		}
		if len(p) != n {
			t.Errorf("%s: %d passes, Skyfield %d", site, len(p), n)
		}
	}
}

// TestVisiblePass checks the visibility flag on the Denver evening pass of
// 2026-09-26 01:18 UTC (19:18 MDT): the Sun is below -6 degrees at Denver and
// the station is sunlit until it enters the shadow at 01:27:05.
func TestVisiblePass(t *testing.T) {
	t.Parallel()
	obs := astronomy.Observer{Lat: 39.74, Lng: -104.99}
	opt := satellite.DefaultPassOptions()
	opt.StdMagnitude = satellite.ISSStandardMagnitude
	from := time.Date(2026, 9, 26, 1, 0, 0, 0, time.UTC)
	passes, err := satellite.Passes(obs, iss(t), from, from.Add(time.Hour), opt)
	if err != nil || len(passes) != 1 {
		t.Fatalf("passes = %v, %v", passes, err)
	}
	p := passes[0]
	if !p.Visible || p.VisibleTo.After(p.ShadowEntry.Add(time.Second)) || p.VisibleFrom.Before(p.Rise.Time) {
		t.Errorf("visible %v from %s to %s, shadow entry %s", p.Visible, p.VisibleFrom, p.VisibleTo, p.ShadowEntry)
	}
	if m := p.Peak.Magnitude; math.IsNaN(m) || m > 0 || m < -4 {
		t.Errorf("peak magnitude %.2f, want a bright pass between -4 and 0", m)
	}
	if !math.IsInf(p.Set.Magnitude, 1) {
		t.Errorf("magnitude after shadow entry %.2f, want +Inf", p.Set.Magnitude)
	}
}
