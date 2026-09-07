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

// ephemerisMoonPhase is the Moon's phase angle in degrees and illuminated
// fraction of the disk, from the JPL Horizons system (target 301, center
// 500@399, quantities "phi" and "Illu%") evaluated in Terrestrial Time.
var ephemerisMoonPhase = []struct {
	jde, phaseAngle, illuminated float64
}{
	{2448724.5, 69.0782, 0.6785466},
	{2461059.5, 176.2131, 0.0010917},
	{2461212.5, 100.8320, 0.4060354},
}

// accuratePhaseAngle computes the Sun-Moon-Earth phase angle, in degrees, by
// the accurate method of Meeus (48.2) and (48.3), from the ported lunar and
// solar series. It is written out here rather than exported because this
// library does not need it; its purpose is to measure the low-accuracy formula
// and, more usefully, to check the whole ported chain -- lunar position, solar
// longitude, and solar distance together -- against an outside ephemeris.
func accuratePhaseAngle(jde float64) float64 {
	moonLon, moonLat, moonDistKm := MoonPosition(jde)
	tc := J2000Century(jde)
	sunLon := SolarApparentLongitude(tc)
	// The astronomical unit, IAU 2012 definition, in kilometres.
	const kmPerAU = 149597870.700
	sunDistKm := SolarRadius(tc) * kmPerAU
	// Cosine of the geocentric elongation of the Moon from the Sun (48.2).
	cosElong := math.Cos(radians(moonLat)) * math.Cos(radians(moonLon-sunLon))
	sinElong := math.Sin(math.Acos(cosElong))
	// (48.3)
	return degrees(math.Atan2(sunDistKm*sinElong, moonDistKm-sunDistKm*cosElong))
}

// TestAccuratePhaseAngleAgainstEphemeris checks the ported lunar position, solar
// longitude, and solar distance series jointly, by combining them into the
// accurate phase angle and comparing that against a modern numerical
// ephemeris. Agreement to a hundredth of a degree is a much tighter statement
// about the port than any one of the three series gives on its own.
func TestAccuratePhaseAngleAgainstEphemeris(t *testing.T) {
	t.Parallel()
	for _, e := range ephemerisMoonPhase {
		got := accuratePhaseAngle(e.jde)
		if math.Abs(got-e.phaseAngle) > 0.01 {
			t.Errorf("JDE %v: accurate phase angle = %.4f, ephemeris %.4f", e.jde, got, e.phaseAngle)
		}
		k := IlluminatedFraction(got)
		if math.Abs(k-e.illuminated) > 1e-4 {
			t.Errorf("JDE %v: illuminated fraction = %.6f, ephemeris %.6f", e.jde, k, e.illuminated)
		}
	}
}

// TestMoonPhaseAngleAgainstEphemeris measures the low-accuracy phase-angle
// formula against a modern numerical ephemeris. Meeus example 48.a computes the
// same instant by the accurate method and gets 69.0756 degrees, which the
// ephemeris confirms.
//
// Formula (48.4) trades accuracy for a closed form in the fundamental arguments
// alone: it uses neither the Moon's nor the Sun's position. Its error in the
// angle reaches about 3.4 degrees, and it is worst near New Moon, where the
// phase angle approaches 180 degrees and is a badly conditioned way to describe
// the geometry. The quantity a caller actually renders, the illuminated
// fraction, is insensitive there: the same 3.4 degree error is worth 0.0024 of
// illumination, because the cosine is flat at the ends of its range.
//
// This is a property of the algorithm, not of the port. Whether to move the
// public phase angle onto the accurate method is a separate question from
// replacing the dependency, and the accurate method is checked above.
func TestMoonPhaseAngleAgainstEphemeris(t *testing.T) {
	t.Parallel()
	for _, e := range ephemerisMoonPhase {
		got := MoonPhaseAngle(e.jde)
		if got > 180 {
			got = 360 - got
		}
		if math.Abs(got-e.phaseAngle) > 4 {
			t.Errorf("JDE %v: phase angle = %.4f, ephemeris %.4f", e.jde, got, e.phaseAngle)
		}
		k := IlluminatedFraction(MoonPhaseAngle(e.jde))
		if math.Abs(k-e.illuminated) > 0.005 {
			t.Errorf("JDE %v: illuminated fraction = %.6f, ephemeris %.6f", e.jde, k, e.illuminated)
		}
	}
}

// TestMoonPhaseAngleTracksAccurateMethod bounds the low-accuracy formula against
// the accurate one over a full synodic month, which is the claim the comment
// above rests on: at most a few degrees in the angle, and a few thousandths in
// the illuminated fraction.
func TestMoonPhaseAngleTracksAccurateMethod(t *testing.T) {
	t.Parallel()
	var worstAngle, worstFraction float64
	for i := 0; i < 30*48; i++ {
		jde := 2461059.0 + float64(i)/48
		low := MoonPhaseAngle(jde)
		folded := low
		if folded > 180 {
			folded = 360 - folded
		}
		accurate := accuratePhaseAngle(jde)
		worstAngle = math.Max(worstAngle, math.Abs(folded-accurate))
		worstFraction = math.Max(worstFraction,
			math.Abs(IlluminatedFraction(low)-IlluminatedFraction(accurate)))
	}
	if worstAngle > 4 {
		t.Errorf("worst phase-angle disagreement over a month was %.4f degrees", worstAngle)
	}
	if worstFraction > 0.003 {
		t.Errorf("worst illuminated-fraction disagreement over a month was %.6f", worstFraction)
	}
	// The disagreement is real, so guard against the two methods silently
	// becoming the same code.
	if worstAngle < 0.1 {
		t.Errorf("the two methods agree to %.6f degrees, which they should not", worstAngle)
	}
}

// TestMoonPhaseAngleRange checks the phase angle is reduced to one revolution
// and sweeps the full cycle over a synodic month.
func TestMoonPhaseAngleRange(t *testing.T) {
	t.Parallel()
	var sawNew, sawFull bool
	for i := 0; i < 30*24; i++ {
		jde := 2461059.0 + float64(i)/24
		a := MoonPhaseAngle(jde)
		if a < 0 || a >= 360 {
			t.Fatalf("phase angle %v out of [0,360) at JDE %v", a, jde)
		}
		folded := a
		if folded > 180 {
			folded = 360 - folded
		}
		if folded > 178 {
			sawNew = true
		}
		if folded < 2 {
			sawFull = true
		}
	}
	if !sawNew || !sawFull {
		t.Errorf("phase angle did not sweep the cycle: new=%v full=%v", sawNew, sawFull)
	}
}

// TestIlluminatedFraction checks the closed form (Meeus 48.1): the disk is fully
// lit at a phase angle of zero, dark at 180 degrees, and half lit at 90.
func TestIlluminatedFraction(t *testing.T) {
	t.Parallel()
	cases := []struct{ angle, want float64 }{
		{0, 1},
		{90, 0.5},
		{180, 0},
		{270, 0.5},
		{360, 1},
	}
	for _, c := range cases {
		if got := IlluminatedFraction(c.angle); math.Abs(got-c.want) > 1e-12 {
			t.Errorf("IlluminatedFraction(%v) = %v, want %v", c.angle, got, c.want)
		}
	}
	// The fraction never leaves [0, 1].
	for a := -720.0; a <= 720; a += 0.37 {
		if k := IlluminatedFraction(a); k < 0 || k > 1 {
			t.Errorf("IlluminatedFraction(%v) = %v, out of [0,1]", a, k)
		}
	}
}

// TestLunationNumber checks the lunation index snaps to the whole lunation
// nearest a decimal year, offset by the quarter being sought (Meeus 49.2).
func TestLunationNumber(t *testing.T) {
	t.Parallel()
	// Meeus example 49.a seeks the New Moon of 1977 February, for which the
	// book gives k = -283.
	if got := lunationNumber(1977.13, 0); got != -283 {
		t.Errorf("lunationNumber(1977.13, 0) = %v, want -283", got)
	}
	// A New Moon index is a whole number; a Full Moon index ends in one half.
	if got := lunationNumber(2026.05, 0.5); got != math.Trunc(got)+0.5 {
		t.Errorf("full moon index %v is not a half integer", got)
	}
	// The epoch itself is lunation zero.
	if got := lunationNumber(2000.0, 0); got != 0 {
		t.Errorf("lunationNumber(2000, 0) = %v, want 0", got)
	}
}

// TestMeanLunarPhaseMeeusExample anchors the mean phase instant against Meeus
// example 49.a, which gives JDE 2443192.94102 for the mean New Moon of
// 1977 February.
func TestMeanLunarPhaseMeeusExample(t *testing.T) {
	t.Parallel()
	k := lunationNumber(1977.13, 0)
	if got := meanLunarPhase(k); math.Abs(got-2443192.94102) > 1e-5 {
		t.Errorf("meanLunarPhase = %.5f, want 2443192.94102", got)
	}
}

// TestNewMoonMeeusExample anchors the true phase instant against Meeus example
// 49.a: the New Moon of 1977 February 18 falls at JDE 2443192.65118.
func TestNewMoonMeeusExample(t *testing.T) {
	t.Parallel()
	if got := NewMoon(1977.13); math.Abs(got-2443192.65118) > 1e-5 {
		t.Errorf("NewMoon(1977.13) = %.5f, want 2443192.65118", got)
	}
}

// TestFullMoonCorrectionsDifferFromNew guards against the two coefficient sets
// being wired to the same table: New and Full use series that differ in the
// third decimal of their leading terms, so the half-lunation offset between
// them is not exactly half a synodic month.
func TestFullMoonCorrectionsDifferFromNew(t *testing.T) {
	t.Parallel()
	newMoon := NewMoon(2026.05)
	fullMoon := FullMoon(2026.05)
	gap := fullMoon - newMoon
	if gap < 13 || gap > 16 {
		t.Fatalf("full moon is %.4f days after new moon, want about half a month", gap)
	}
	// Half a mean synodic month is 14.765294 days. The true interval differs
	// by hours, which is the whole point of the correction series.
	if math.Abs(gap-14.765294) < 0.01 {
		t.Errorf("interval %.6f days is suspiciously close to the mean, corrections may not be applied", gap)
	}
}

// ephemerisLunarPhases are the instants at which the apparent geocentric
// ecliptic longitudes of the Moon and the Sun differ by 0 and by 180 degrees,
// interpolated from the JPL Horizons system at twenty-minute steps. Horizons
// reports UTC.
var ephemerisLunarPhases = []struct {
	name string
	fn   func(float64) float64
	year float64
	utc  time.Time
}{
	{"new moon January 2026", NewMoon, 2026.05, time.Date(2026, 1, 18, 19, 51, 59, 0, time.UTC)},
	{"full moon July 2026", FullMoon, 2026.55, time.Date(2026, 7, 29, 14, 35, 43, 0, time.UTC)},
}

// TestLunarPhasesAgainstEphemeris measures the phase series against a modern
// numerical ephemeris.
//
// The series returns dynamical time and this library does not model Delta-T, so
// the instant it hands back is about seventy seconds later than the civil
// instant in the present era. Meeus gives the accuracy of the chapter 49 series
// itself as a few seconds over the modern range. The three minute tolerance
// covers both, and both are documented limits rather than defects.
func TestLunarPhasesAgainstEphemeris(t *testing.T) {
	t.Parallel()
	const tol = 3 * time.Minute
	for _, c := range ephemerisLunarPhases {
		got := JDToTime(c.fn(c.year))
		if d := got.Sub(c.utc); d > tol || d < -tol {
			t.Errorf("%s = %v, ephemeris %v (%v off)", c.name, got.UTC(), c.utc, d)
		}
	}
}

// TestLunarPhaseSpacing checks successive New Moons are one synodic month
// apart, which exercises the lunation stepping the callers rely on.
func TestLunarPhaseSpacing(t *testing.T) {
	t.Parallel()
	const lunation = 1 / 12.3685 // one lunation as a fraction of a Julian year
	prev := NewMoon(2020.0)
	for i := 1; i < 60; i++ {
		next := NewMoon(2020.0 + float64(i)*lunation)
		gap := next - prev
		// The synodic month varies between about 29.27 and 29.83 days.
		if gap < 29.2 || gap > 29.9 {
			t.Errorf("lunation %d spans %.5f days, want 29.2 to 29.9", i, gap)
		}
		prev = next
	}
}
