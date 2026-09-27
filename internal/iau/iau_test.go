package iau

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
)

// TestSolarDays checks the solar day each model implies from its rotation
// and orbit against the published values.
func TestSolarDays(t *testing.T) {
	t.Parallel()
	cases := []struct {
		m    *Model
		want float64
		tol  float64
	}{
		{Earth, 1.0, 1e-4},
		{Moon, 29.5306, 1e-3},
		{Mars, 1.0274912517, 1e-5},
		{Venus, 116.75, 0.01},
		{Mercury, 175.94, 0.05},
		{Jupiter, 0.41358, 1e-4},
	}
	for _, c := range cases {
		if got := c.m.SolarDays(); math.Abs(got-c.want) > c.tol {
			t.Errorf("%s: solar day %.6f days, want %.6f", c.m.Name, got, c.want)
		}
	}
}

// TestJ2000Orientation checks the models at J2000.0, where the periodic
// terms of the 2015 Mars model must reproduce the 2009 constants they
// replaced (317.681, 52.887, 176.630) to a few thousandths of a degree.
func TestJ2000Orientation(t *testing.T) {
	t.Parallel()
	o := Mars.At(j2000)
	for name, c := range map[string][2]float64{
		"alpha0": {o.Alpha0, 317.68143},
		"delta0": {o.Delta0, 52.88650},
		"W":      {o.W, 176.630},
	} {
		if math.Abs(c[0]-c[1]) > 0.005 {
			t.Errorf("Mars %s at J2000 = %.5f, want %.5f", name, c[0], c[1])
		}
	}
	if w := Venus.At(j2000 + 1).W; math.Abs(w-(160.20-1.4813688)) > 1e-9 {
		t.Errorf("Venus W a day after J2000 = %v", w)
	}
	for _, m := range []*Model{Sun, Mercury, Venus, Earth, Moon, Mars, Jupiter, Saturn, Uranus, Neptune} {
		o := m.At(2461000.5)
		if o.W < 0 || o.W >= 360 || math.Abs(o.Delta0) > 90 || m.PolarKm > m.EquatorialKm {
			t.Errorf("%s: %+v", m.Name, o)
		}
		if ByName(m.Name) != m {
			t.Errorf("ByName(%q)", m.Name)
		}
	}
	if ByName("pluto") != nil {
		t.Error("ByName(pluto) is not nil")
	}
}
