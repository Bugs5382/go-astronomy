package sun_test

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

	meeusbase "github.com/soniakeys/meeus/v3/base"
	meeuscoord "github.com/soniakeys/meeus/v3/coord"
	meeusjulian "github.com/soniakeys/meeus/v3/julian"
	"github.com/soniakeys/meeus/v3/sidereal"
	"github.com/soniakeys/meeus/v3/solar"
	"github.com/soniakeys/unit"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/sun"
)

const deg2rad = math.Pi / 180

// observer test fixtures spanning hemispheres and longitudes.
var (
	brooklyn = astronomy.Observer{Lat: 40.678, Lng: -73.944, TZ: mustLoad("America/New_York")}
	quito    = astronomy.Observer{Lat: -0.1807, Lng: -78.4678, TZ: mustLoad("America/Guayaquil")}
	tromso   = astronomy.Observer{Lat: 69.6492, Lng: 18.9553, TZ: mustLoad("Europe/Oslo")}
	auckland = astronomy.Observer{Lat: -36.8485, Lng: 174.7633, TZ: mustLoad("Pacific/Auckland")}
)

func mustLoad(name string) *time.Location {
	loc, err := time.LoadLocation(name)
	if err != nil {
		panic(err)
	}
	return loc
}

// expectedAltitude computes the Sun's geometric altitude independently of the
// coordinates package, using plain spherical trigonometry on the apparent
// solar equatorial position and mean sidereal time. It is an independent
// cross-check of the horizontal transform, not a copy of it.
func expectedAltitude(t time.Time, latDeg, lngEastDeg float64) float64 {
	jd := meeusjulian.TimeToJD(t.UTC())
	ra, dec := solar.ApparentEquatorial(jd)
	gst := sidereal.Mean(jd).Hour() * 15 // degrees
	hourAngle := (gst + lngEastDeg - ra.Deg()) * deg2rad
	latR := latDeg * deg2rad
	decR := dec.Deg() * deg2rad
	sinAlt := math.Sin(latR)*math.Sin(decR) + math.Cos(latR)*math.Cos(decR)*math.Cos(hourAngle)
	return math.Asin(sinAlt) / deg2rad
}

// meeusPipeline reproduces the exact wiring sun.Position is expected to use:
// apparent solar equatorial position, mean Greenwich sidereal time, and the
// meeus horizontal transform with west-positive longitude and a 180 degree
// azimuth rotation to clockwise-from-north.
func meeusPipeline(t time.Time, latDeg, lngEastDeg float64) (alt, az, diam float64) {
	jd := meeusjulian.TimeToJD(t.UTC())
	ra, dec := solar.ApparentEquatorial(jd)
	gst := sidereal.Mean(jd).Hour() * 15
	st := unit.TimeFromHour(gst / 15)
	a, h := meeuscoord.EqToHz(
		ra,
		dec,
		unit.AngleFromDeg(latDeg),
		unit.AngleFromDeg(-lngEastDeg),
		st,
	)
	az = math.Mod(a.Deg()+180+360, 360)
	alt = h.Deg()
	r := solar.Radius(meeusbase.J2000Century(jd))
	diam = 2 * 959.63 / 3600 / r
	return
}

func TestPositionMatchesMeeusPipeline(t *testing.T) {
	t.Parallel()
	instants := []time.Time{
		time.Date(1982, 5, 3, 16, 0, 0, 0, time.UTC),
		time.Date(1992, 10, 13, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 9, 4, 12, 30, 0, 0, time.UTC),
		time.Date(2026, 12, 21, 17, 0, 0, 0, time.UTC),
	}
	observers := []astronomy.Observer{brooklyn, quito, tromso, auckland}
	for _, obs := range observers {
		for _, when := range instants {
			pos := sun.Position(obs, when)
			wantAlt, wantAz, wantDiam := meeusPipeline(when, obs.Lat, obs.Lng)
			if math.Abs(pos.Altitude-wantAlt) > 1e-9 {
				t.Errorf("%v %v altitude = %.12f, want %.12f", obs, when, pos.Altitude, wantAlt)
			}
			if math.Abs(pos.Azimuth-wantAz) > 1e-9 {
				t.Errorf("%v %v azimuth = %.12f, want %.12f", obs, when, pos.Azimuth, wantAz)
			}
			if math.Abs(float64(pos.Diameter)-wantDiam) > 1e-12 {
				t.Errorf("%v %v diameter = %.12f, want %.12f", obs, when, float64(pos.Diameter), wantDiam)
			}
			if pos.Azimuth < 0 || pos.Azimuth >= 360 {
				t.Errorf("%v %v azimuth = %v out of [0,360)", obs, when, pos.Azimuth)
			}
		}
	}
}

func TestPositionAltitudeIndependent(t *testing.T) {
	t.Parallel()
	cases := []struct {
		obs astronomy.Observer
		t   time.Time
	}{
		{brooklyn, time.Date(1982, 5, 3, 16, 0, 0, 0, time.UTC)},
		{quito, time.Date(2026, 3, 20, 17, 0, 0, 0, time.UTC)},
		{tromso, time.Date(2026, 6, 21, 10, 0, 0, 0, time.UTC)},
		{auckland, time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)},
	}
	for _, c := range cases {
		pos := sun.Position(c.obs, c.t)
		want := expectedAltitude(c.t, c.obs.Lat, c.obs.Lng)
		if math.Abs(pos.Altitude-want) > 1e-4 {
			t.Errorf("%v %v altitude = %.6f, want %.6f", c.obs, c.t, pos.Altitude, want)
		}
	}
}

// TestPositionSolarNoonAltitude verifies the physical identity that at meridian
// transit the Sun's altitude equals 90 - |lat - dec|, with the declination
// taken from meeus at the transit instant. Transit is located by scanning the
// day for peak altitude.
func TestPositionSolarNoonAltitude(t *testing.T) {
	t.Parallel()
	observers := []astronomy.Observer{brooklyn, quito, auckland}
	date := time.Date(1982, 5, 3, 0, 0, 0, 0, time.UTC)
	for _, obs := range observers {
		start := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
		var peak astronomy.Position
		var peakTime time.Time
		peak.Altitude = -1000
		for i := 0; i <= 24*60; i++ {
			when := start.Add(time.Duration(i) * time.Minute)
			p := sun.Position(obs, when)
			if p.Altitude > peak.Altitude {
				peak = p
				peakTime = when
			}
		}
		// declination at transit
		_, dec := solar.ApparentEquatorial(meeusjulian.TimeToJD(peakTime))
		want := 90 - math.Abs(obs.Lat-dec.Deg())
		if math.Abs(peak.Altitude-want) > 0.1 {
			t.Errorf("%v transit altitude = %.4f, want ~%.4f", obs, peak.Altitude, want)
		}
		// northern-hemisphere transit is due south (azimuth ~180) when dec < lat.
		if obs.Lat > 0 && dec.Deg() < obs.Lat {
			if math.Abs(peak.Azimuth-180) > 1.0 {
				t.Errorf("%v transit azimuth = %.4f, want ~180", obs, peak.Azimuth)
			}
		}
	}
}

func TestPositionAngularDiameter(t *testing.T) {
	t.Parallel()
	// Sampled across a year, the Sun's apparent diameter stays within the
	// perihelion/aphelion band, roughly 0.524 to 0.545 degrees.
	for month := 1; month <= 12; month++ {
		when := time.Date(2026, time.Month(month), 15, 12, 0, 0, 0, time.UTC)
		pos := sun.Position(brooklyn, when)
		d := float64(pos.Diameter)
		if d < 0.523 || d > 0.546 {
			t.Errorf("%v diameter = %.5f out of expected band", when, d)
		}
	}
	// Diameter is independent of the observer's location at a given instant.
	when := time.Date(2026, 7, 4, 0, 0, 0, 0, time.UTC)
	a := sun.Position(brooklyn, when).Diameter
	b := sun.Position(auckland, when).Diameter
	if math.Abs(float64(a)-float64(b)) > 1e-12 {
		t.Errorf("diameter observer-dependent: %v vs %v", a, b)
	}
}

func TestTrackSampleCount(t *testing.T) {
	t.Parallel()
	date := time.Date(1982, 5, 3, 0, 0, 0, 0, brooklyn.Location())
	for _, n := range []int{2, 10, 97, 288} {
		got := sun.Track(brooklyn, date, n)
		if len(got) != n {
			t.Errorf("Track(n=%d) returned %d samples", n, len(got))
		}
	}
}

func TestTrackDegenerateSamples(t *testing.T) {
	t.Parallel()
	date := time.Date(1982, 5, 3, 0, 0, 0, 0, brooklyn.Location())
	for _, n := range []int{-1, 0, 1} {
		if got := sun.Track(brooklyn, date, n); got != nil {
			t.Errorf("Track(n=%d) = %v, want nil", n, got)
		}
	}
}

func TestTrackMonotonicTimeAndProgress(t *testing.T) {
	t.Parallel()
	date := time.Date(1982, 5, 3, 0, 0, 0, 0, brooklyn.Location())
	const n = 48
	samples := sun.Track(brooklyn, date, n)
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

func TestTrackSpansCivilDay(t *testing.T) {
	t.Parallel()
	loc := brooklyn.Location()
	date := time.Date(1982, 5, 3, 9, 30, 0, 0, loc) // time-of-day should be ignored
	samples := sun.Track(brooklyn, date, 25)
	wantStart := time.Date(1982, 5, 3, 0, 0, 0, 0, loc)
	wantEnd := wantStart.AddDate(0, 0, 1)
	if !samples[0].Time.Equal(wantStart) {
		t.Errorf("first sample %v, want %v", samples[0].Time, wantStart)
	}
	if !samples[len(samples)-1].Time.Equal(wantEnd) {
		t.Errorf("last sample %v, want %v", samples[len(samples)-1].Time, wantEnd)
	}
}

func TestTrackMatchesPosition(t *testing.T) {
	t.Parallel()
	date := time.Date(1982, 5, 3, 0, 0, 0, 0, brooklyn.Location())
	samples := sun.Track(brooklyn, date, 20)
	for _, s := range samples {
		pos := sun.Position(brooklyn, s.Time)
		if math.Abs(pos.Altitude-s.Altitude) > 1e-12 || math.Abs(pos.Azimuth-s.Azimuth) > 1e-12 {
			t.Errorf("sample at %v = (%.6f,%.6f), Position = (%.6f,%.6f)",
				s.Time, s.Altitude, s.Azimuth, pos.Altitude, pos.Azimuth)
		}
	}
}
