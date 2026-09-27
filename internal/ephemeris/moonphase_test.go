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
//
// The rows sample one synodic month, the New Moon of 2026 January to the New
// Moon of 2026 February, at two and a half day steps, with three instants
// inside an hour of the New Moon itself, where the phase angle is worst
// conditioned. The instant of Meeus example 48.a, the New Moon of Meeus
// example 49.a, and one instant from a later month are carried as well.
var ephemerisMoonPhase = []struct {
	jde, phaseAngle, illuminated float64
}{
	{2443192.65118, 175.7494, 0.0013753},
	{2448724.5, 69.0782, 0.6785466},
	{2461059.0, 174.7589, 0.0020904},
	{2461059.29, 176.5630, 0.0008993},
	{2461059.35, 176.6326, 0.0008633},
	{2461059.5, 176.2131, 0.0010917},
	{2461061.5, 154.5371, 0.0485679},
	{2461064.0, 124.1524, 0.2193022},
	{2461066.5, 92.4513, 0.4786145},
	{2461069.0, 59.5299, 0.7535440},
	{2461071.5, 25.9234, 0.9496898},
	{2461074.0, 7.6440, 0.9955569},
	{2461076.5, 38.5898, 0.8908157},
	{2461079.0, 67.5186, 0.6911919},
	{2461081.5, 94.9273, 0.4570539},
	{2461084.0, 122.0666, 0.2345479},
	{2461086.5, 150.1421, 0.0663685},
	{2461088.16667, 169.7425, 0.0079913},
	{2461212.5, 100.8320, 0.4060354},
}

// lowAccuracyPhaseAngle computes the Sun-Moon-Earth phase angle, in degrees in
// [0, 360), by the closed-form low-accuracy formula of Meeus (48.4), which
// works from the fundamental arguments alone and uses neither the Moon's nor
// the Sun's position.
//
// It is kept here, out of the package's own code, only to measure what the
// accurate method buys: the library computes the phase angle by (48.2) and
// (48.3) instead. The arguments D, M and M' are the fundamental arguments of
// chapter 47, so moonArguments supplies them rather than a second copy of the
// polynomials; only the leading term needs D back in degrees.
func lowAccuracyPhaseAngle(jde float64) float64 {
	dr, mr, mpr, _ := moonArguments(J2000Century(jde))
	angle := 180 - pmod(degrees(dr), 360) +
		-6.289*math.Sin(mpr) +
		2.100*math.Sin(mr) +
		-1.274*math.Sin(2*dr-mpr) +
		-0.658*math.Sin(2*dr) +
		-0.214*math.Sin(2*mpr) +
		-0.110*math.Sin(dr)
	return pmod(angle, 360)
}

// foldPhaseAngle reduces an angle in [0, 360) to the [0, 180] a phase angle
// physically occupies. Only the low-accuracy formula needs it; (48.3) resolves
// the quadrant itself.
func foldPhaseAngle(deg float64) float64 {
	if deg > 180 {
		return 360 - deg
	}
	return deg
}

// TestMoonPhaseAngleAgainstEphemeris checks the phase angle, and the
// illuminated fraction that follows from it, against a modern numerical
// ephemeris across a synodic month.
//
// The check is a joint statement about three ported series at once -- the
// lunar position, the solar longitude, and the solar distance -- because the
// accurate method combines all three. Agreement to a few hundredths of a
// degree is a much tighter statement about the port than any one of them gives
// alone.
func TestMoonPhaseAngleAgainstEphemeris(t *testing.T) {
	t.Parallel()
	// The worst disagreement measured over these instants is 0.0131 degrees
	// in the angle and 0.00009 in the fraction.
	const angleTol, fractionTol = 0.02, 2e-4
	for _, e := range ephemerisMoonPhase {
		got := MoonPhaseAngle(e.jde)
		if math.Abs(got-e.phaseAngle) > angleTol {
			t.Errorf("JDE %v: phase angle = %.4f, ephemeris %.4f", e.jde, got, e.phaseAngle)
		}
		k := IlluminatedFraction(got)
		if math.Abs(k-e.illuminated) > fractionTol {
			t.Errorf("JDE %v: illuminated fraction = %.6f, ephemeris %.6f", e.jde, k, e.illuminated)
		}
	}
}

// TestMoonPhaseAngleBeatsLowAccuracyFormula records why the library computes
// the phase angle the expensive way. Both methods are measured against the same
// ephemeris table: the closed form is out by degrees, the accurate method by
// hundredths of a degree.
//
// The closed form is worst near New Moon, where the phase angle approaches 180
// degrees and is a badly conditioned way to describe the geometry. The
// illuminated fraction is insensitive there, because the cosine is flat at the
// ends of its range, which is why the closed form was serviceable for as long
// as illumination was all anyone read.
func TestMoonPhaseAngleBeatsLowAccuracyFormula(t *testing.T) {
	t.Parallel()
	var worstLow, worstAccurate, worstLowFraction float64
	for _, e := range ephemerisMoonPhase {
		low := foldPhaseAngle(lowAccuracyPhaseAngle(e.jde))
		worstLow = math.Max(worstLow, math.Abs(low-e.phaseAngle))
		worstAccurate = math.Max(worstAccurate, math.Abs(MoonPhaseAngle(e.jde)-e.phaseAngle))
		worstLowFraction = math.Max(worstLowFraction,
			math.Abs(IlluminatedFraction(low)-e.illuminated))
	}
	// The closed form reaches 4.14 degrees over this table, at the New Moon of
	// 1977 February.
	if worstLow < 3 {
		t.Errorf("closed form was out by only %.4f degrees; the two methods should differ by more", worstLow)
	}
	if worstAccurate > 0.02 {
		t.Errorf("accurate method was out by %.4f degrees", worstAccurate)
	}
	if worstAccurate > worstLow/100 {
		t.Errorf("accurate method %.4f is not two orders better than the closed form %.4f",
			worstAccurate, worstLow)
	}
	// The same error is worth only thousandths of the illuminated fraction.
	if worstLowFraction > 0.0025 {
		t.Errorf("closed-form illuminated fraction was out by %.6f, more than expected", worstLowFraction)
	}
}

// TestMoonPhaseAngleTracksLowAccuracyFormula bounds the two methods against each
// other continuously, at hourly steps over a synodic month, rather than at the
// tabulated instants alone. The closed form stays within a few degrees in the
// angle and a few thousandths in the illuminated fraction.
func TestMoonPhaseAngleTracksLowAccuracyFormula(t *testing.T) {
	t.Parallel()
	var worstAngle, worstFraction float64
	for i := 0; i < 30*24; i++ {
		jde := 2461059.0 + float64(i)/24
		low := lowAccuracyPhaseAngle(jde)
		accurate := MoonPhaseAngle(jde)
		worstAngle = math.Max(worstAngle, math.Abs(foldPhaseAngle(low)-accurate))
		worstFraction = math.Max(worstFraction,
			math.Abs(IlluminatedFraction(low)-IlluminatedFraction(accurate)))
	}
	// Measured: 3.36 degrees and 0.0023 over this month.
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

// TestMoonPhaseAngleRange checks the phase angle stays inside the [0, 180] it
// physically occupies and sweeps the cycle over a synodic month.
//
// It does not reach either limit. At New Moon and at Full Moon the geocentric
// elongation of the Moon from the Sun is its ecliptic latitude, up to about
// 5.3 degrees, so the angle stops that far short of 180 and of 0.
func TestMoonPhaseAngleRange(t *testing.T) {
	t.Parallel()
	var sawNew, sawFull bool
	for i := 0; i < 30*24; i++ {
		jde := 2461059.0 + float64(i)/24
		a := MoonPhaseAngle(jde)
		if a < 0 || a > 180 {
			t.Fatalf("phase angle %v out of [0,180] at JDE %v", a, jde)
		}
		if a > 174 {
			sawNew = true
		}
		if a < 3 {
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
// The series returns dynamical time, which JDEToTime converts to UTC with the
// leap-second table (issue 45). What remains is the series itself, which Meeus
// gives as good to a few seconds over the modern range; the fixtures are also
// interpolated at twenty-minute steps. The observed error is under 8 s, and
// the tolerance is 15 s.
func TestLunarPhasesAgainstEphemeris(t *testing.T) {
	t.Parallel()
	const tol = 15 * time.Second
	for _, c := range ephemerisLunarPhases {
		got := JDEToTime(c.fn(c.year))
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

// TestMoonElongationAtPhaseEvents checks the elongation against the instants the
// phase series places New and Full Moon at. Elongation is the difference in
// apparent ecliptic longitude, so it is near 0 (or 360) at New Moon and near 180
// at Full Moon -- unlike the phase angle, which stops short of both limits
// because it carries the Moon's ecliptic latitude.
func TestMoonElongationAtPhaseEvents(t *testing.T) {
	t.Parallel()
	for year := 1977.0; year < 2027; year += 3.7 {
		newMoon := NewMoon(year)
		e := MoonElongation(newMoon)
		// Near 0 from either side.
		if d := math.Min(e, 360-e); d > 0.5 {
			t.Errorf("elongation at New Moon (JDE %.5f) = %.4f, want within 0.5 of 0", newMoon, e)
		}
		full := FullMoon(year)
		if e := MoonElongation(full); math.Abs(e-180) > 0.5 {
			t.Errorf("elongation at Full Moon (JDE %.5f) = %.4f, want within 0.5 of 180", full, e)
		}
	}
}

// TestMoonElongationRange checks the elongation stays inside [0, 360) and, unlike
// the phase angle, does reach both ends of the cycle.
func TestMoonElongationRange(t *testing.T) {
	t.Parallel()
	base := NewMoon(2026.0)
	var low, high float64 = 360, 0
	for step := 0.0; step < 30; step += 0.05 {
		e := MoonElongation(base + step)
		if e < 0 || e >= 360 {
			t.Fatalf("elongation %.4f out of [0, 360) at JDE %.4f", e, base+step)
		}
		low = math.Min(low, e)
		high = math.Max(high, e)
	}
	if low > 1 {
		t.Errorf("elongation never came within 1 degree of 0 (min %.4f)", low)
	}
	if high < 359 {
		t.Errorf("elongation never came within 1 degree of 360 (max %.4f)", high)
	}
}

// TestMoonElongationIncreasesThroughCycle checks the elongation advances through
// the cycle rather than folding, which is what lets it carry the waxing or waning
// sense that the phase angle cannot.
func TestMoonElongationIncreasesThroughCycle(t *testing.T) {
	t.Parallel()
	base := NewMoon(2026.0) + 1
	previous := MoonElongation(base)
	for step := 0.25; step < 28; step += 0.25 {
		e := MoonElongation(base + step)
		if e < previous {
			t.Fatalf("elongation went backwards at JDE %.4f: %.4f then %.4f", base+step, previous, e)
		}
		previous = e
	}
}
