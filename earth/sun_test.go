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
	"math"
	"testing"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/internal/julian"
)

const deg2rad = math.Pi / 180

// sunPinned holds the Sun's horizontal position and apparent diameter, in
// degrees, as this library computes them. The values pin the observer/instant
// grid below so a change to the solar model, the sidereal time, or the
// horizontal transform is caught to the twelfth decimal even when the change
// is far too small for the authority checks in this file to see. The independent
// accuracy checks are TestSunPositionAgainstEphemeris and, for the solar model
// itself, the Meeus worked examples in internal/julian and internal/coordinates.
// The values were re-pinned when the Sun moved to Terrestrial Time with full
// nutation, apparent sidereal time, and the observer's parallax (issue 45),
// which moved them by up to about 30 arc seconds, and again when the solar
// theory moved to VSOP87, which moved them by up to about 10 arc seconds.
var sunPinned = []struct {
	observer string
	when     string
	alt      float64
	az       float64
	diameter float64
}{
	{"brooklyn", "1982-05-03T16:00:00Z", 62.542321469376, 151.623185947405, 0.528797984926},
	{"brooklyn", "1992-10-13T00:00:00Z", -19.811420543502, 276.878220435616, 0.534405942064},
	{"brooklyn", "2026-09-04T12:30:00Z", 22.569034617598, 100.471780730241, 0.528647877561},
	{"brooklyn", "2026-12-21T17:00:00Z", 25.866979105624, 181.543856793914, 0.541938252649},
	{"quito", "1982-05-03T16:00:00Z", 66.392800707931, 46.898716502804, 0.528797984926},
	{"quito", "1992-10-13T00:00:00Z", -14.794815212325, 261.899119997438, 0.534405942064},
	{"quito", "2026-09-04T12:30:00Z", 19.098420197739, 82.463440868277, 0.528647877561},
	{"quito", "2026-12-21T17:00:00Z", 66.559424961010, 173.044064201396, 0.541938252649},
	{"tromso", "1982-05-03T16:00:00Z", 18.265216533379, 265.979005577795, 0.528797984926},
	{"tromso", "1992-10-13T00:00:00Z", -26.463367755099, 24.924258087123, 0.534405942064},
	{"tromso", "2026-09-04T12:30:00Z", 25.057694890790, 209.487755378282, 0.528647877561},
	{"tromso", "2026-12-21T17:00:00Z", -23.422510179949, 265.491743858510, 0.541938252649},
	{"auckland", "1982-05-03T16:00:00Z", -36.738439352882, 97.884571773758, 0.528797984926},
	{"auckland", "1992-10-13T00:00:00Z", 60.887764663045, 3.688179199708, 0.534405942064},
	{"auckland", "2026-09-04T12:30:00Z", -60.124934317185, 175.000050299962, 0.528647877561},
	{"auckland", "2026-12-21T17:00:00Z", -0.570634711357, 120.298100582586, 0.541938252649},
}

// namedObservers resolves the observer fixtures used by the pinned tables.
var namedObservers = map[string]astronomy.Observer{
	"brooklyn": brooklyn,
	"quito":    quito,
	"tromso":   tromso,
	"auckland": auckland,
}

// mustParse parses an RFC 3339 instant or fails the test.
func mustParse(t *testing.T, s string) time.Time {
	t.Helper()
	when, err := time.Parse(time.RFC3339, s)
	if err != nil {
		t.Fatalf("parse %q: %v", s, err)
	}
	return when
}

// TestSunPositionPinnedValues locks the Sun's altitude, azimuth, and apparent
// diameter to the values in sunPinned across four observers and four instants
// spanning both hemispheres, the Arctic, the equator, and the anti-meridian.
func TestSunPositionPinnedValues(t *testing.T) {
	t.Parallel()
	for _, c := range sunPinned {
		obs, ok := namedObservers[c.observer]
		if !ok {
			t.Fatalf("unknown observer %q", c.observer)
		}
		when := mustParse(t, c.when)
		pos := earth.SunPosition(obs, when)
		if math.Abs(pos.Altitude-c.alt) > 1e-9 {
			t.Errorf("%s %s altitude = %.12f, want %.12f", c.observer, c.when, pos.Altitude, c.alt)
		}
		if math.Abs(pos.Azimuth-c.az) > 1e-9 {
			t.Errorf("%s %s azimuth = %.12f, want %.12f", c.observer, c.when, pos.Azimuth, c.az)
		}
		if math.Abs(float64(pos.Diameter)-c.diameter) > 1e-12 {
			t.Errorf("%s %s diameter = %.12f, want %.12f", c.observer, c.when, float64(pos.Diameter), c.diameter)
		}
		if pos.Azimuth < 0 || pos.Azimuth >= 360 {
			t.Errorf("%s %s azimuth = %v out of [0,360)", c.observer, c.when, pos.Azimuth)
		}
	}
}

// ephemerisSun is the Sun's apparent geocentric right ascension and declination
// in degrees, and its geocentric distance in astronomical units, as published by
// the JPL Horizons system (target 10, center 500@399, apparent RA/Dec referred
// to the true equator and equinox of date). These are independent of this
// library and of the algorithms it implements.
var ephemerisSun = []struct {
	when     string
	ra       float64
	dec      float64
	distance float64
}{
	{"1982-05-03T16:00:00Z", 40.43755, 15.70779, 1.00818792359644},
	{"1992-10-13T00:00:00Z", 198.37875, -7.78407, 0.99760832548205},
	{"2026-09-04T12:30:00Z", 163.40180, 7.05982, 1.00847421895488},
	{"2026-12-21T17:00:00Z", 269.82259, -23.43732, 0.98374267926946},
	{"2026-01-15T00:00:00Z", 296.76541, -21.16059, 0},
	{"2026-03-20T17:00:00Z", 0.08484, 0.03689, 0},
	{"2026-06-21T10:00:00Z", 90.06897, 23.43792, 0},
}

// horizontalFromEquatorial converts an equatorial position to altitude and
// azimuth with plain spherical trigonometry, independent of the internal
// coordinates package. Azimuth is measured clockwise from true north.
func horizontalFromEquatorial(raDeg, decDeg, gstDeg, latDeg, lngEastDeg float64) (alt, az float64) {
	h := (gstDeg + lngEastDeg - raDeg) * deg2rad
	lat := latDeg * deg2rad
	dec := decDeg * deg2rad
	sinAlt := math.Sin(lat)*math.Sin(dec) + math.Cos(lat)*math.Cos(dec)*math.Cos(h)
	alt = math.Asin(sinAlt) / deg2rad
	// Azimuth from the south, measured westward (Meeus 13.5), rotated to
	// clockwise from north.
	azSouth := math.Atan2(math.Sin(h), math.Cos(h)*math.Sin(lat)-math.Tan(dec)*math.Cos(lat))
	az = math.Mod(azSouth/deg2rad+180+360, 360)
	return
}

// TestSunPositionAgainstEphemeris checks the Earth-vantage solar chain against
// the JPL Horizons apparent right ascension and declination, converted to the
// horizontal frame by independent spherical trigonometry. The sidereal time is
// this library's own, itself anchored to Meeus examples 12.a and 12.b in
// internal/julian.
//
// The Horizons places are geocentric and the reference conversion uses mean
// sidereal time, while SunPosition is topocentric (solar parallax, up to 8.8
// arc seconds) and uses apparent sidereal time (the nutation in right
// ascension, up to about 17 arc seconds). Those two account for the observed
// worst of 0.0064 degrees in altitude and 0.0087 in azimuth, so the tolerances
// are 0.01 and 0.02 degrees. The solar theory itself is measured to 0.5 arc
// seconds in internal/ephemeris.
func TestSunPositionAgainstEphemeris(t *testing.T) {
	t.Parallel()
	const tol = 0.01
	observers := []astronomy.Observer{brooklyn, quito, tromso, auckland}
	for _, e := range ephemerisSun {
		when := mustParse(t, e.when)
		gst := julian.GreenwichSiderealTime(when)
		for _, obs := range observers {
			pos := earth.SunPosition(obs, when)
			wantAlt, wantAz := horizontalFromEquatorial(e.ra, e.dec, gst, obs.Lat, obs.Lng)
			if math.Abs(pos.Altitude-wantAlt) > tol {
				t.Errorf("%v %s altitude = %.6f, ephemeris %.6f", obs, e.when, pos.Altitude, wantAlt)
			}
			// Azimuth is ill-conditioned near the zenith and near the pole,
			// where a tiny altitude error swings the bearing widely, so it is
			// only checked where the geometry is well behaved.
			if math.Abs(pos.Altitude) > 5 && math.Abs(pos.Altitude) < 80 {
				d := math.Abs(pos.Azimuth - wantAz)
				if d > 180 {
					d = 360 - d
				}
				if d > 0.02 {
					t.Errorf("%v %s azimuth = %.6f, ephemeris %.6f", obs, e.when, pos.Azimuth, wantAz)
				}
			}
			if e.distance > 0 {
				wantDiam := 2 * 959.63 / 3600 / e.distance
				if math.Abs(float64(pos.Diameter)-wantDiam) > 1e-4 {
					t.Errorf("%v %s diameter = %.6f, ephemeris %.6f", obs, e.when, float64(pos.Diameter), wantDiam)
				}
			}
		}
	}
}

// ephemerisSunDecMay1982 is the Sun's apparent declination in degrees at six
// hour intervals through 1982 May 3 UTC, from JPL Horizons. The declination
// changes by only about 0.07 degrees over each interval and does so almost
// linearly, so interpolating between these anchors is accurate to well under a
// thousandth of a degree.
var ephemerisSunDecMay1982 = [5]float64{15.51116, 15.58512, 15.65881, 15.73223, 15.80537}

// ephemerisSunDec returns the Sun's apparent declination at an instant on
// 1982 May 3 UTC, linearly interpolated from the published anchors.
func ephemerisSunDec(when time.Time) float64 {
	day := time.Date(1982, 5, 3, 0, 0, 0, 0, time.UTC)
	x := when.UTC().Sub(day).Hours() / 6
	if x < 0 {
		x = 0
	}
	if x > 4 {
		x = 4
	}
	i := int(x)
	if i > 3 {
		i = 3
	}
	f := x - float64(i)
	return ephemerisSunDecMay1982[i] + f*(ephemerisSunDecMay1982[i+1]-ephemerisSunDecMay1982[i])
}

// TestSunPositionSolarNoonAltitude verifies the physical identity that at
// meridian transit the Sun's altitude equals 90 - |lat - dec|, with the
// declination taken from the JPL Horizons anchors at the transit instant.
// Transit is located by scanning the day for peak altitude.
func TestSunPositionSolarNoonAltitude(t *testing.T) {
	t.Parallel()
	observers := []astronomy.Observer{brooklyn, quito, auckland}
	start := time.Date(1982, 5, 3, 0, 0, 0, 0, time.UTC)
	for _, obs := range observers {
		var peak astronomy.Position
		var peakTime time.Time
		peak.Altitude = -1000
		for i := 0; i <= 24*60; i++ {
			when := start.Add(time.Duration(i) * time.Minute)
			p := earth.SunPosition(obs, when)
			if p.Altitude > peak.Altitude {
				peak = p
				peakTime = when
			}
		}
		dec := ephemerisSunDec(peakTime)
		want := 90 - math.Abs(obs.Lat-dec)
		if math.Abs(peak.Altitude-want) > 0.1 {
			t.Errorf("%v transit altitude = %.4f, want ~%.4f", obs, peak.Altitude, want)
		}
		// Northern-hemisphere transit is due south (azimuth ~180) when dec < lat.
		if obs.Lat > 0 && dec < obs.Lat {
			if math.Abs(peak.Azimuth-180) > 1.0 {
				t.Errorf("%v transit azimuth = %.4f, want ~180", obs, peak.Azimuth)
			}
		}
	}
}

func TestSunPositionAngularDiameter(t *testing.T) {
	t.Parallel()
	// Sampled across a year, the Sun's apparent diameter stays within the
	// perihelion/aphelion band, roughly 0.524 to 0.545 degrees.
	for month := 1; month <= 12; month++ {
		when := time.Date(2026, time.Month(month), 15, 12, 0, 0, 0, time.UTC)
		pos := earth.SunPosition(brooklyn, when)
		d := float64(pos.Diameter)
		if d < 0.523 || d > 0.546 {
			t.Errorf("%v diameter = %.5f out of expected band", when, d)
		}
	}
	// Diameter is independent of the observer's location at a given instant.
	when := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	a := earth.SunPosition(brooklyn, when).Diameter
	b := earth.SunPosition(auckland, when).Diameter
	if math.Abs(float64(a)-float64(b)) > 1e-12 {
		t.Errorf("diameter observer-dependent: %v vs %v", a, b)
	}
}

func TestSunTrackSampleCount(t *testing.T) {
	t.Parallel()
	date := time.Date(1982, 5, 3, 0, 0, 0, 0, brooklyn.Location())
	for _, n := range []int{2, 10, 97, 288} {
		got := earth.SunTrack(brooklyn, date, n)
		if len(got) != n {
			t.Errorf("SunTrack(n=%d) returned %d samples", n, len(got))
		}
	}
}

func TestSunTrackDegenerateSamples(t *testing.T) {
	t.Parallel()
	date := time.Date(1982, 5, 3, 0, 0, 0, 0, brooklyn.Location())
	for _, n := range []int{-1, 0, 1} {
		if got := earth.SunTrack(brooklyn, date, n); got != nil {
			t.Errorf("SunTrack(n=%d) = %v, want nil", n, got)
		}
	}
}

func TestSunTrackMonotonicTimeAndProgress(t *testing.T) {
	t.Parallel()
	date := time.Date(1982, 5, 3, 0, 0, 0, 0, brooklyn.Location())
	const n = 48
	samples := earth.SunTrack(brooklyn, date, n)
	if samples[0].TimeProgress != 0 {
		t.Errorf("first TimeProgress = %v, want 0", samples[0].TimeProgress)
	}
	if math.Abs(samples[n-1].TimeProgress-1) > 1e-12 {
		t.Errorf("last TimeProgress = %v, want 1", samples[n-1].TimeProgress)
	}
	for i := 1; i < n; i++ {
		if !samples[i].Time.After(samples[i-1].Time) {
			t.Errorf("time not monotonic at %d: %v then %v", i, samples[i-1].Time, samples[i].Time)
		}
		if samples[i].TimeProgress <= samples[i-1].TimeProgress {
			t.Errorf("progress not monotonic at %d: %v then %v", i, samples[i-1].TimeProgress, samples[i].TimeProgress)
		}
		if samples[i].Azimuth < 0 || samples[i].Azimuth >= 360 {
			t.Errorf("azimuth %v out of [0,360) at %d", samples[i].Azimuth, i)
		}
	}
}

func TestSunTrackSpansCivilDay(t *testing.T) {
	t.Parallel()
	loc := brooklyn.Location()
	date := time.Date(1982, 5, 3, 9, 30, 0, 0, loc) // time-of-day should be ignored
	samples := earth.SunTrack(brooklyn, date, 25)
	wantStart := time.Date(1982, 5, 3, 0, 0, 0, 0, loc)
	wantEnd := wantStart.AddDate(0, 0, 1)
	if !samples[0].Time.Equal(wantStart) {
		t.Errorf("first sample %v, want %v", samples[0].Time, wantStart)
	}
	if !samples[len(samples)-1].Time.Equal(wantEnd) {
		t.Errorf("last sample %v, want %v", samples[len(samples)-1].Time, wantEnd)
	}
}

func TestSunTrackMatchesPosition(t *testing.T) {
	t.Parallel()
	date := time.Date(1982, 5, 3, 0, 0, 0, 0, brooklyn.Location())
	samples := earth.SunTrack(brooklyn, date, 20)
	for _, s := range samples {
		pos := earth.SunPosition(brooklyn, s.Time)
		if math.Abs(pos.Altitude-s.Altitude) > 1e-12 || math.Abs(pos.Azimuth-s.Azimuth) > 1e-12 {
			t.Errorf("sample at %v = (%.6f,%.6f), SunPosition = (%.6f,%.6f)",
				s.Time, s.Altitude, s.Azimuth, pos.Altitude, pos.Azimuth)
		}
	}
}
