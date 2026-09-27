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
	"math"
	"testing"
	"time"

	"github.com/Bugs5382/go-astronomy/earth/moon"
	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
	"github.com/Bugs5382/go-astronomy/internal/julian"
)

// TestIlluminationMeeusExample anchors the phase angle and illuminated fraction
// against the meeus worked example for 1992 April 12.0, which computes the
// phase angle by the accurate method of chapter 48 and gets 69.0756 degrees and
// 0.6786 illuminated. JPL Horizons gives 69.0782 and 0.678547 for the same
// instant in Terrestrial Time.
func TestIlluminationMeeusExample(t *testing.T) {
	t.Parallel()
	when := time.Date(1992, 4, 12, 0, 0, 0, 0, time.UTC)
	if got, want := moon.PhaseAngle(when), 69.078; abs(got-want) > 0.05 {
		t.Errorf("PhaseAngle = %.4f, want ~%.3f", got, want)
	}
	if got, want := moon.Illumination(when), 0.678547; abs(got-want) > 0.001 {
		t.Errorf("Illumination = %.4f, want ~%.6f", got, want)
	}
}

// ttMinusUTC2026 is TT - UTC throughout 2026: 32.184 s plus 37 leap seconds.
const ttMinusUTC2026 = 69184 * time.Millisecond

// ephemerisPhase is the Moon's phase angle in degrees and illuminated fraction
// of the disc, from the JPL Horizons system (target 301, center 500@399,
// quantities "phi" and "Illu%"), tabulated in Terrestrial Time.
//
// The rows sample the synodic month from the New Moon of 2026 January to the
// New Moon of 2026 February at two and a half day steps. Three of them fall
// inside an hour of the January New Moon, which is where the phase angle is
// worst conditioned: it approaches 180 degrees, where a small change in the
// geometry is a large change in the angle.
//
// The instants are the Terrestrial Time instants Horizons reports. The library
// takes UTC, so each is handed over 69.184 s earlier (TT - UTC in 2026), which
// is the same physical instant.
var ephemerisPhase = []struct {
	when                    time.Time
	phaseAngle, illuminated float64
}{
	{time.Date(2026, 1, 18, 12, 0, 0, 0, time.UTC), 174.7589, 0.0020904},
	{time.Date(2026, 1, 18, 18, 57, 36, 0, time.UTC), 176.5630, 0.0008993},
	{time.Date(2026, 1, 18, 20, 24, 0, 0, time.UTC), 176.6326, 0.0008633},
	{time.Date(2026, 1, 21, 0, 0, 0, 0, time.UTC), 154.5371, 0.0485679},
	{time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC), 124.1524, 0.2193022},
	{time.Date(2026, 1, 26, 0, 0, 0, 0, time.UTC), 92.4513, 0.4786145},
	{time.Date(2026, 1, 28, 12, 0, 0, 0, time.UTC), 59.5299, 0.7535440},
	{time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC), 25.9234, 0.9496898},
	{time.Date(2026, 2, 2, 12, 0, 0, 0, time.UTC), 7.6440, 0.9955569},
	{time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC), 38.5898, 0.8908157},
	{time.Date(2026, 2, 7, 12, 0, 0, 0, time.UTC), 67.5186, 0.6911919},
	{time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC), 94.9273, 0.4570539},
	{time.Date(2026, 2, 12, 12, 0, 0, 0, time.UTC), 122.0666, 0.2345479},
	{time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC), 150.1421, 0.0663685},
}

// TestPhaseAngleAgainstEphemeris measures the public phase angle, and the
// illuminated fraction that follows from it, against a modern numerical
// ephemeris across a synodic month.
//
// A caller may read the phase angle for something other than illumination --
// the position angle of the illuminated limb, for instance -- where a degree
// of error shows. The worst disagreement measured over these instants, handed
// over as UTC, is 0.013 degrees in the angle and 9e-5 in the fraction, so the
// tolerances are 0.02 degrees and 1.5e-4.
func TestPhaseAngleAgainstEphemeris(t *testing.T) {
	t.Parallel()
	for _, e := range ephemerisPhase {
		when := e.when.Add(-ttMinusUTC2026)
		if got := moon.PhaseAngle(when); abs(got-e.phaseAngle) > 0.02 {
			t.Errorf("PhaseAngle(%s) = %.4f, ephemeris %.4f", when, got, e.phaseAngle)
		}
		if got := moon.Illumination(when); abs(got-e.illuminated) > 1.5e-4 {
			t.Errorf("Illumination(%s) = %.6f, ephemeris %.6f", when, got, e.illuminated)
		}
	}
}

// TestIlluminationDerivesFromPhaseAngle checks the two cannot disagree: the
// illuminated fraction is exactly (1 + cos i) / 2 of the phase angle the same
// call returns, not a second, independently computed quantity.
func TestIlluminationDerivesFromPhaseAngle(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 30*24; i++ {
		when := base.Add(time.Duration(i) * time.Hour)
		angle := moon.PhaseAngle(when)
		want := (1 + math.Cos(angle*math.Pi/180)) / 2
		if got := moon.Illumination(when); math.Abs(got-want) > 1e-12 {
			t.Fatalf("Illumination(%s) = %.15f, but (1+cos %.15f)/2 = %.15f",
				when, got, angle, want)
		}
	}
}

// TestPhaseRangesOverCycle walks a full synodic month and checks the invariants
// that hold for every instant: age within the synodic period, illumination in
// [0,1], and phase angle in [0,180].
func TestPhaseRangesOverCycle(t *testing.T) {
	t.Parallel()
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 60; i++ {
		when := base.Add(time.Duration(i) * 12 * time.Hour)
		if a := moon.Age(when); a < 0 || a > moon.SynodicMonth+0.01 {
			t.Errorf("Age(%s) = %.4f, out of [0, %.4f]", when, a, moon.SynodicMonth)
		}
		if k := moon.Illumination(when); k < 0 || k > 1 {
			t.Errorf("Illumination(%s) = %.4f, out of [0,1]", when, k)
		}
		if p := moon.PhaseAngle(when); p < 0 || p > 180 {
			t.Errorf("PhaseAngle(%s) = %.4f, out of [0,180]", when, p)
		}
	}
}

// TestNextNewIsNewMoon anchors NextNew against the meeus example (New Moon of
// 1977 February 18, JDE 2443192.65118) and checks the returned instant is a
// genuine new moon: minimal illumination, and a phase angle matching what JPL
// Horizons reports for that instant in Terrestrial Time, 175.7494 degrees.
//
// The angle falls short of 180 because the Moon is 4.2 degrees off the ecliptic
// there: New Moon is the instant the two apparent longitudes agree, so the
// remaining elongation is the Moon's latitude.
func TestNextNewIsNewMoon(t *testing.T) {
	t.Parallel()
	from := time.Date(1977, 2, 10, 0, 0, 0, 0, time.UTC)
	got := moon.NextNew(from)
	want := time.Date(1977, 2, 18, 3, 37, 42, 0, time.UTC)
	if d := got.Sub(want); d < -2*time.Minute || d > 2*time.Minute {
		t.Errorf("NextNew = %s, want ~%s", got.UTC(), want)
	}
	if !got.After(from) {
		t.Errorf("NextNew %s is not after %s", got, from)
	}
	if k := moon.Illumination(got); k > 0.01 {
		t.Errorf("illumination at new moon = %.4f, want ~0", k)
	}
	if p, want := moon.PhaseAngle(got), 175.7494; abs(p-want) > 0.05 {
		t.Errorf("phase angle at new moon = %.4f, ephemeris %.4f", p, want)
	}
}

// TestNextFullIsFullMoon checks NextFull returns a future instant within one
// synodic month whose illumination is essentially full and whose phase angle is
// near zero.
//
// The bound on the angle is the Moon's greatest ecliptic latitude, about 5.3
// degrees. Full Moon is the instant the two apparent longitudes are 180 degrees
// apart, so whatever latitude the Moon has then is left over as the phase
// angle. JPL Horizons gives 2.0732 degrees for the instant this returns.
func TestNextFullIsFullMoon(t *testing.T) {
	t.Parallel()
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	got := moon.NextFull(from)
	if !got.After(from) {
		t.Fatalf("NextFull %s is not after %s", got, from)
	}
	if d := got.Sub(from).Hours() / 24; d > moon.SynodicMonth+0.5 {
		t.Errorf("NextFull is %.2f days out, beyond a synodic month", d)
	}
	if k := moon.Illumination(got); k < 0.99 {
		t.Errorf("illumination at full moon = %.4f, want ~1", k)
	}
	if p := moon.PhaseAngle(got); p > 5.4 {
		t.Errorf("phase angle at full moon = %.4f, want at most the lunar latitude", p)
	}
}

// TestNextNewFullWithinSynodicMonth verifies that from any starting instant the
// next new and full moons both fall within one synodic month.
func TestNextNewFullWithinSynodicMonth(t *testing.T) {
	t.Parallel()
	base := time.Date(2025, 1, 3, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 30; i++ {
		from := base.Add(time.Duration(i) * 25 * time.Hour)
		for _, next := range []time.Time{moon.NextNew(from), moon.NextFull(from)} {
			if !next.After(from) {
				t.Errorf("next phase %s not after %s", next, from)
			}
			if d := next.Sub(from).Hours() / 24; d > moon.SynodicMonth+0.5 {
				t.Errorf("next phase %.2f days from %s, beyond synodic month", d, from)
			}
		}
	}
}

// TestPhaseNames maps a full cycle onto the eight named phases and asserts the
// ordering of the named points: New near age 0, First Quarter near a quarter
// period, Full near mid-cycle, Last Quarter near three quarters.
func TestPhaseNames(t *testing.T) {
	t.Parallel()
	// Start just after a known new moon so age increases predictably.
	newMoon := moon.NextNew(time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	sm := moon.SynodicMonth
	cases := []struct {
		offsetDays float64
		want       moon.Phase
	}{
		{0.2, moon.New},
		{sm * 0.125, moon.WaxingCrescent},
		{sm * 0.25, moon.FirstQuarter},
		{sm * 0.375, moon.WaxingGibbous},
		{sm * 0.5, moon.Full},
		{sm * 0.625, moon.WaningGibbous},
		{sm * 0.75, moon.LastQuarter},
		{sm * 0.875, moon.WaningCrescent},
	}
	for _, tc := range cases {
		when := newMoon.Add(time.Duration(tc.offsetDays * 24 * float64(time.Hour)))
		if got := moon.PhaseAt(when); got != tc.want {
			t.Errorf("Phase at +%.2fd = %v, want %v", tc.offsetDays, got, tc.want)
		}
	}
}

// TestPhaseString checks each named phase has a stable, non-empty label and that
// all eight are distinct.
func TestPhaseString(t *testing.T) {
	t.Parallel()
	phases := []moon.Phase{
		moon.New, moon.WaxingCrescent, moon.FirstQuarter, moon.WaxingGibbous,
		moon.Full, moon.WaningGibbous, moon.LastQuarter, moon.WaningCrescent,
	}
	seen := map[string]bool{}
	for _, p := range phases {
		s := p.String()
		if s == "" {
			t.Errorf("phase %d has empty string", int(p))
		}
		if seen[s] {
			t.Errorf("duplicate phase label %q", s)
		}
		seen[s] = true
	}
	if got := moon.Phase(99).String(); got == "" {
		t.Error("out-of-range phase should still have a label")
	}
}

// TestPhaseAtAgreesWithPhaseEvents is the invariant that names the phase from
// the sky rather than from a clock: at the instant this package's own series
// places a New Moon, the phase must be New, and likewise for Full.
//
// Slicing the cycle by age cannot guarantee this. SynodicMonth is a mean and
// individual cycles run several hours either side of it, so an age-based
// boundary drifts against the event it is supposed to straddle.
func TestPhaseAtAgreesWithPhaseEvents(t *testing.T) {
	t.Parallel()
	from := time.Date(1977, 1, 1, 0, 0, 0, 0, time.UTC)
	for lunation := 0; lunation < 60; lunation++ {
		newMoon := moon.NextNew(from)
		if got := moon.PhaseAt(newMoon); got != moon.New {
			t.Errorf("PhaseAt(New Moon %s) = %v, want new", newMoon.Format(time.RFC3339), got)
		}
		full := moon.NextFull(from)
		if got := moon.PhaseAt(full); got != moon.Full {
			t.Errorf("PhaseAt(Full Moon %s) = %v, want full", full.Format(time.RFC3339), got)
		}
		from = newMoon.Add(36 * time.Hour)
	}
}

// TestPhaseAtAdvancesInOrder walks a synodic month and checks the phases arrive
// in their cycle order, each one following the last, wrapping once from waning
// crescent back to new.
func TestPhaseAtAdvancesInOrder(t *testing.T) {
	t.Parallel()
	start := moon.NextNew(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))
	previous := moon.PhaseAt(start)
	if previous != moon.New {
		t.Fatalf("cycle did not start at new moon, got %v", previous)
	}
	wraps := 0
	for hour := 1; hour <= 24*30; hour++ {
		at := start.Add(time.Duration(hour) * time.Hour)
		got := moon.PhaseAt(at)
		switch {
		case got == previous:
		case got == previous+1:
			previous = got
		case previous == moon.WaningCrescent && got == moon.New:
			wraps++
			previous = got
		default:
			t.Fatalf("phase jumped from %v to %v at %s", previous, got, at.Format(time.RFC3339))
		}
	}
	if wraps != 1 {
		t.Errorf("cycle wrapped %d times over 30 days, want 1", wraps)
	}
}

// elongationCrossing returns the instant, within a lunation of the New Moon
// nearest year, at which the Moon's elongation reaches target degrees. It
// bisects, which is enough for a test and keeps no root-finder in the library.
func elongationCrossing(year, target float64) time.Time {
	lo, hi := ephemeris.NewMoon(year)-1, ephemeris.NewMoon(year)+30
	for i := 0; i < 60; i++ {
		mid := (lo + hi) / 2
		e := ephemeris.MoonElongation(mid)
		// Near New Moon the value wraps, so read the far side as negative.
		if target == 0 && e > 180 {
			e -= 360
		}
		if e < target {
			lo = mid
		} else {
			hi = mid
		}
	}
	return julian.Time(hi)
}

// TestPhaseAtSectorCentresMatchIllumination checks the named phases sit where
// the geometry puts them, using a quantity computed by a different route.
//
// PhaseAt reads the elongation; Illumination comes from the phase angle of Meeus
// (48.3), which combines the two distances and the Moon's latitude. So agreement
// between them is a real cross-check rather than a restatement: at the centre of
// the New sector the disc must be dark, at the quarters half lit, and at Full
// lit. The quarters land a touch over half because the Moon is a finite distance
// away, not because the sector is misplaced.
func TestPhaseAtSectorCentresMatchIllumination(t *testing.T) {
	t.Parallel()
	cases := []struct {
		elongation float64
		want       moon.Phase
		illum      float64
		tolerance  float64
	}{
		{elongation: 0, want: moon.New, illum: 0, tolerance: 0.001},
		{elongation: 90, want: moon.FirstQuarter, illum: 0.5, tolerance: 0.005},
		{elongation: 180, want: moon.Full, illum: 1, tolerance: 0.001},
		{elongation: 270, want: moon.LastQuarter, illum: 0.5, tolerance: 0.005},
	}
	for _, tc := range cases {
		for year := 2020.0; year < 2027; year += 1.9 {
			at := elongationCrossing(year, tc.elongation)
			if got := moon.PhaseAt(at); got != tc.want {
				t.Errorf("at elongation %.0f (%s): PhaseAt = %v, want %v",
					tc.elongation, at.Format(time.RFC3339), got, tc.want)
			}
			if got := moon.Illumination(at); math.Abs(got-tc.illum) > tc.tolerance {
				t.Errorf("at elongation %.0f (%s): illumination = %.4f, want %.1f +/- %.3f",
					tc.elongation, at.Format(time.RFC3339), got, tc.illum, tc.tolerance)
			}
		}
	}
}
