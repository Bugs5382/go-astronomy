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
	"errors"
	"math"
	"testing"
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
)

// TestStandardAtmosphereSeaLevelIsExact checks the default refraction factor
// is exactly 1 at sea level, so every sea-level answer is unchanged by the
// height scaling, and that the sea-level horizon is still HorizonAltitude.
func TestStandardAtmosphereSeaLevelIsExact(t *testing.T) {
	t.Parallel()
	if f := earth.StandardAtmosphere.Factor(astronomy.SeaLevel); f != 1 {
		t.Errorf("Factor(sea level) = %v, want exactly 1", f)
	}
	for _, alt := range []float64{-1, 0, 5, 30, 89} {
		if got, want := earth.StandardAtmosphere.Refraction(alt, astronomy.SeaLevel), earth.Refraction(alt); got != want {
			t.Errorf("Refraction(%v, sea level) = %v, want %v", alt, got, want)
		}
	}
	if got := earth.HorizonAltitudeAt(astronomy.SeaLevel); got != earth.HorizonAltitude {
		t.Errorf("HorizonAltitudeAt(sea level) = %v, want %v", got, earth.HorizonAltitude)
	}
}

// TestStandardAtmosphereFactor checks the ISA density ratio against published
// standard atmosphere values, as P/P0 * T0/T from the ICAO and US 1976 tables:
// 5000 ft (1524 m) is 843.1 hPa and 5.1 C; 35000 ft (10668 m) is 238.4 hPa and
// -54.3 C; 15 km is 121.1 hPa and -56.5 C; 20 km is 55.3 hPa and -56.5 C.
func TestStandardAtmosphereFactor(t *testing.T) {
	t.Parallel()
	ratio := func(hPa, c float64) float64 { return hPa / 1013.25 * 288.15 / (273.15 + c) }
	cases := []struct {
		name string
		h    astronomy.Height
		want float64
	}{
		{"5000 ft", astronomy.Meters(1524), ratio(843.1, 5.1)},
		{"Denver", astronomy.Meters(1609), ratio(834.3, 4.5)},
		{"La Paz", astronomy.Meters(3640), ratio(645.9, -8.7)},
		{"35000 ft", astronomy.Meters(10668), ratio(238.4, -54.3)},
		{"15 km", astronomy.Meters(15000), ratio(121.1, -56.5)},
		{"20 km", astronomy.Meters(20000), ratio(55.29, -56.5)},
	}
	for _, c := range cases {
		got := earth.StandardAtmosphere.Factor(c.h)
		if math.Abs(got-c.want) > 0.002 {
			t.Errorf("%s: Factor = %.4f, want %.4f", c.name, got, c.want)
		}
	}
}

// TestStandardAtmosphereReduction checks the refraction a body on the horizon
// receives at 1524 m and 10668 m: about 14% and 69% less than at sea level.
func TestStandardAtmosphereReduction(t *testing.T) {
	t.Parallel()
	sea := earth.Refraction(0)
	for _, c := range []struct {
		h         float64
		reduction float64
	}{{1524, 0.138}, {10668, 0.690}} {
		got := earth.StandardAtmosphere.Refraction(0, astronomy.Meters(c.h))
		if r := 1 - got/sea; math.Abs(r-c.reduction) > 0.003 {
			t.Errorf("%v m: horizon refraction %.2f arcmin, reduced by %.3f, want %.3f", c.h, got*60, r, c.reduction)
		}
	}
}

// TestStandardAtmosphereShape checks the factor falls smoothly from below sea
// level to 50 km: strictly decreasing, positive, and continuous across the
// layer boundaries of the standard atmosphere.
func TestStandardAtmosphereShape(t *testing.T) {
	t.Parallel()
	prev := earth.StandardAtmosphere.Factor(astronomy.Meters(-500))
	if prev <= 1 {
		t.Errorf("Factor(-500 m) = %v, want above 1 (denser air below sea level)", prev)
	}
	for h := -490.0; h <= 50000; h += 10 {
		f := earth.StandardAtmosphere.Factor(astronomy.Meters(h))
		if !(f > 0) || f >= prev {
			t.Fatalf("Factor(%v m) = %v after %v, want positive and falling", h, f, prev)
		}
		if prev-f > 0.0015 {
			t.Fatalf("Factor jumps by %v at %v m", prev-f, h)
		}
		prev = f
	}
	for _, h := range []float64{1e5, 1e6} {
		if f := earth.StandardAtmosphere.Factor(astronomy.Meters(h)); !(f >= 0) || f > 1e-5 {
			t.Errorf("Factor(%v m) = %v, want a tiny non-negative number", h, f)
		}
	}
	if f := earth.StandardAtmosphere.Factor(astronomy.Meters(-1e6)); math.IsNaN(f) || math.IsInf(f, 0) || f < 1 {
		t.Errorf("Factor(-1e6 m) = %v, want a finite factor above 1", f)
	}
	if f := earth.StandardAtmosphere.Factor(astronomy.Meters(math.NaN())); f != 1 {
		t.Errorf("Factor(NaN) = %v, want 1 (no correction)", f)
	}
}

// TestMeasuredAtmosphere checks the absolute factor for measured air,
// P/1010 * 283/(273+T): exactly 1 at the formula's reference conditions,
// whatever the height, and the documented values elsewhere.
func TestMeasuredAtmosphere(t *testing.T) {
	t.Parallel()
	ref := earth.MeasuredAtmosphere(1010, 10)
	for _, h := range []float64{0, 1609, 10668} {
		if f := ref.Factor(astronomy.Meters(h)); f != 1 {
			t.Errorf("reference air at %v m: Factor = %v, want exactly 1", h, f)
		}
	}
	cases := []struct{ p, c, want float64 }{
		{1013.25, 15, 1013.25 / 1010 * 283 / 288},
		{840, 25, 840.0 / 1010 * 283 / 298},
		{1030, -30, 1030.0 / 1010 * 283 / 243},
	}
	for _, c := range cases {
		a := earth.MeasuredAtmosphere(c.p, c.c)
		if err := a.Err(); err != nil {
			t.Fatalf("%v hPa %v C: %v", c.p, c.c, err)
		}
		if got := a.Factor(astronomy.Meters(1609)); math.Abs(got-c.want) > 1e-12 {
			t.Errorf("%v hPa %v C: Factor = %v, want %v", c.p, c.c, got, c.want)
		}
		if got, want := a.Refraction(0, astronomy.SeaLevel), earth.Refraction(0)*c.want; math.Abs(got-want) > 1e-12 {
			t.Errorf("%v hPa %v C: Refraction(0) = %v, want %v", c.p, c.c, got, want)
		}
	}
}

// TestInvalidAtmosphere checks that measured air that cannot exist is
// rejected with the coded ErrInvalidAtmosphere by the functions that take a
// segmentation, and that its factor falls back to no correction.
func TestInvalidAtmosphere(t *testing.T) {
	t.Parallel()
	obs := astronomy.Observer{Lat: 39.74, Lng: -104.99, Height: astronomy.Meters(1609)}
	day := time.Date(2027, 6, 21, 12, 0, 0, 0, time.UTC)
	bad := []earth.Atmosphere{
		earth.MeasuredAtmosphere(0, 10),
		earth.MeasuredAtmosphere(-5, 10),
		earth.MeasuredAtmosphere(math.NaN(), 10),
		earth.MeasuredAtmosphere(1010, -273),
		earth.MeasuredAtmosphere(1010, math.Inf(1)),
	}
	for _, a := range bad {
		err := a.Err()
		if code, _ := apperr.Code(err); !errors.Is(err, earth.ErrInvalidAtmosphere) || code != astronomy.CodeInvalidAtmosphere {
			t.Errorf("%+v: Err() = %v, want ErrInvalidAtmosphere with code %d", a, err, astronomy.CodeInvalidAtmosphere)
		}
		if f := a.Factor(obs.Height); f != 1 {
			t.Errorf("%+v: Factor = %v, want 1", a, f)
		}
		seg := earth.DefaultSegmentation.WithAtmosphere(a)
		if _, err := earth.NewSunTimesWith(obs, day, seg); !errors.Is(err, earth.ErrInvalidAtmosphere) {
			t.Errorf("NewSunTimesWith(%+v) error = %v, want ErrInvalidAtmosphere", a, err)
		}
		if _, _, err := earth.SegmentAtWith(obs, day, seg); !errors.Is(err, earth.ErrInvalidAtmosphere) {
			t.Errorf("SegmentAtWith(%+v) error = %v, want ErrInvalidAtmosphere", a, err)
		}
	}
	if err := earth.StandardAtmosphere.Err(); err != nil {
		t.Errorf("StandardAtmosphere.Err() = %v", err)
	}
}

// TestHorizonAltitudeAt checks the Sun's rise and set altitude at height: the
// sea-level -0.833 degrees, raised by the refraction the thinner air no longer
// gives and lowered by the dip.
func TestHorizonAltitudeAt(t *testing.T) {
	t.Parallel()
	for _, m := range []float64{-430, 1524, 1609, 3640, 10668} {
		h := astronomy.Meters(m)
		f := earth.StandardAtmosphere.Factor(h)
		want := earth.HorizonAltitude + (1-f)*earth.HorizonRefraction - earth.HorizonDip(h)
		if got := earth.HorizonAltitudeAt(h); math.Abs(got-want) > 1e-12 {
			t.Errorf("HorizonAltitudeAt(%v m) = %.5f, want %.5f", m, got, want)
		}
	}
	if d := earth.HorizonRefraction - 34.0/60; d != 0 {
		t.Errorf("HorizonRefraction off 34 arcmin by %v", d)
	}
}

// sunEdges returns the start of the sunrise band and the end of the sunset
// band on the civil day of date.
func sunEdges(t *testing.T, obs astronomy.Observer, date time.Time, seg earth.Segmentation) (rise, set time.Time) {
	t.Helper()
	day, err := earth.NewSunTimesWith(obs, date, seg)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range day.Segments() {
		switch s.Label {
		case earth.LabelSunrise:
			if rise.IsZero() {
				rise = s.From
			}
		case earth.LabelSunset:
			set = s.To
		}
	}
	if rise.IsZero() || set.IsZero() {
		t.Fatalf("no sunrise or sunset at %+v", obs)
	}
	return rise, set
}

// TestRefractionHeightMovesRiseAndSet checks the direction and size of the
// refraction scaling at Denver (1609 m) and La Paz (3640 m). Against the
// sea-level refraction (the reference air, factor 1), the thinner standard air
// lifts the Sun less, so sunrise comes later and sunset earlier, by the time
// the Sun takes to climb (1-f) * 34 arc minutes. The dip still dominates, so
// both stay well ahead of the sea-level observer. At sea level the default and
// the reference air give identical days.
func TestRefractionHeightMovesRiseAndSet(t *testing.T) {
	t.Parallel()
	seaLevelAir := earth.DefaultSegmentation.WithAtmosphere(earth.MeasuredAtmosphere(1010, 10))
	cases := []struct {
		name     string
		obs      astronomy.Observer
		min, max time.Duration
	}{
		{"Denver", astronomy.Observer{Lat: 39.74, Lng: -104.99, TZ: time.UTC, Height: astronomy.Meters(1609)},
			15 * time.Second, 45 * time.Second},
		{"La Paz", astronomy.Observer{Lat: -16.50, Lng: -68.15, TZ: time.UTC, Height: astronomy.Meters(3640)},
			30 * time.Second, 70 * time.Second},
	}
	for _, date := range []time.Time{
		time.Date(2027, 3, 20, 12, 0, 0, 0, time.UTC),
		time.Date(2027, 6, 21, 12, 0, 0, 0, time.UTC),
		time.Date(2027, 12, 21, 12, 0, 0, 0, time.UTC),
	} {
		for _, c := range cases {
			rise, set := sunEdges(t, c.obs, date, earth.DefaultSegmentation)
			rise1, set1 := sunEdges(t, c.obs, date, seaLevelAir)
			sea := c.obs
			sea.Height = astronomy.SeaLevel
			rise0, set0 := sunEdges(t, sea, date, earth.DefaultSegmentation)

			if d := rise.Sub(rise1); d < c.min || d > c.max {
				t.Errorf("%s %s: sunrise later by %v with the thinner air, want %v to %v", c.name, date.Format(time.DateOnly), d, c.min, c.max)
			}
			if d := set1.Sub(set); d < c.min || d > c.max {
				t.Errorf("%s %s: sunset earlier by %v with the thinner air, want %v to %v", c.name, date.Format(time.DateOnly), d, c.min, c.max)
			}
			if !rise.Before(rise0) || !set.After(set0) {
				t.Errorf("%s: rise %s set %s, want before %s and after %s (the dip dominates)", c.name, rise, set, rise0, set0)
			}
			r0, s0 := sunEdges(t, sea, date, seaLevelAir)
			if !r0.Equal(rise0) || !s0.Equal(set0) {
				t.Errorf("%s at sea level: the reference air moved the day (%s %s against %s %s)", c.name, r0, s0, rise0, set0)
			}
		}
	}
}

// TestFlightAltitudeHorizon checks the horizon at 35000 ft: the dip of about
// 3 degrees, with less than a third of the sea-level refraction.
func TestFlightAltitudeHorizon(t *testing.T) {
	t.Parallel()
	// -0.833 + (1 - 0.31) * 0.5667 - 1.76 * sqrt(10668) / 60
	if got, want := earth.HorizonAltitudeAt(astronomy.Feet(35000)), -3.472; math.Abs(got-want) > 0.005 {
		t.Errorf("HorizonAltitudeAt(35000 ft) = %.4f, want %.3f", got, want)
	}
}
