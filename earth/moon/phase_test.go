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
	"testing"
	"time"

	"github.com/Bugs5382/go-astronomy/earth/moon"
)

// TestIlluminationMeeusExample anchors the illuminated fraction against the
// meeus worked example for 1992 April 12, 0h UT (phase angle 68.88 degrees,
// illuminated fraction 0.6801).
func TestIlluminationMeeusExample(t *testing.T) {
	t.Parallel()
	when := time.Date(1992, 4, 12, 0, 0, 0, 0, time.UTC)
	if got, want := moon.PhaseAngle(when), 68.88; abs(got-want) > 0.05 {
		t.Errorf("PhaseAngle = %.4f, want ~%.2f", got, want)
	}
	if got, want := moon.Illumination(when), 0.6801; abs(got-want) > 0.001 {
		t.Errorf("Illumination = %.4f, want ~%.4f", got, want)
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
// genuine new moon: minimal illumination and phase angle near 180 degrees.
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
	if p := moon.PhaseAngle(got); p < 178 {
		t.Errorf("phase angle at new moon = %.4f, want ~180", p)
	}
}

// TestNextFullIsFullMoon checks NextFull returns a future instant within one
// synodic month whose illumination is essentially full and whose phase angle is
// near zero.
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
	if p := moon.PhaseAngle(got); p > 2 {
		t.Errorf("phase angle at full moon = %.4f, want ~0", p)
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
