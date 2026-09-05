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

	"github.com/Bugs5382/go-astronomy/sun"
)

// TestApparentSemidiameterAt1AU pins the semidiameter at unit distance to the
// documented constant, converted to degrees.
func TestApparentSemidiameterAt1AU(t *testing.T) {
	t.Parallel()
	got := sun.ApparentSemidiameter(1)
	want := sun.SemidiameterArcsecAt1AU / 3600
	if math.Abs(got-want) > 1e-12 {
		t.Errorf("ApparentSemidiameter(1) = %.12f deg, want %.12f", got, want)
	}
	// About sixteen arc minutes, the familiar mean solar semidiameter.
	if math.Abs(got*60-16) > 0.05 {
		t.Errorf("ApparentSemidiameter(1) = %.5f arcmin, want ~16", got*60)
	}
}

// TestApparentDiameterIsTwiceSemidiameter checks the diameter/semidiameter
// relationship and that AngularDiameter.Radius recovers the semidiameter.
func TestApparentDiameterIsTwiceSemidiameter(t *testing.T) {
	t.Parallel()
	for _, d := range []float64{0.9833, 1.0, 1.0167, 1.5, 30} {
		diam := sun.ApparentDiameter(d)
		semi := sun.ApparentSemidiameter(d)
		if math.Abs(float64(diam)-2*semi) > 1e-12 {
			t.Errorf("ApparentDiameter(%.4f) = %.12f, want %.12f", d, float64(diam), 2*semi)
		}
		if math.Abs(diam.Radius()-semi) > 1e-12 {
			t.Errorf("ApparentDiameter(%.4f).Radius() = %.12f, want %.12f", d, diam.Radius(), semi)
		}
	}
}

// TestApparentDiameterScalesInversely confirms apparent size falls off as 1/d:
// doubling the distance halves the diameter.
func TestApparentDiameterScalesInversely(t *testing.T) {
	t.Parallel()
	base := float64(sun.ApparentDiameter(1))
	for _, k := range []float64{2, 4, 10} {
		got := float64(sun.ApparentDiameter(k))
		if math.Abs(got-base/k) > 1e-12 {
			t.Errorf("ApparentDiameter(%g) = %.12f, want %.12f", k, got, base/k)
		}
	}
}

// TestApparentDiameterEarthBand checks that across Earth's orbital distance range
// (roughly perihelion 0.9833 AU to aphelion 1.0167 AU) the Sun's apparent
// diameter stays in the familiar band of about 0.524 to 0.545 degrees.
func TestApparentDiameterEarthBand(t *testing.T) {
	t.Parallel()
	for _, d := range []float64{0.9833, 1.0, 1.0167} {
		got := float64(sun.ApparentDiameter(d))
		if got < 0.523 || got > 0.546 {
			t.Errorf("ApparentDiameter(%.4f) = %.5f out of expected band", d, got)
		}
	}
}
