package horizons

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
	"fmt"
	"math"
	"time"
)

// points is the order of the Lagrange interpolation.
const points = 8

// at interpolates the table at t with an eight-point Lagrange polynomial
// through the samples around it. The right ascension is unwrapped across
// 0/360 degrees before it is interpolated.
func (t *table) at(when time.Time) (sample, error) {
	n := len(t.Samples)
	x := float64(when.Sub(t.Start)) / float64(t.Step)
	if x < float64(points/2-1) || x > float64(n-points/2) {
		return sample{}, fmt.Errorf("%w: %s is outside %s to %s", ErrOutsideCoverage, when.Format(time.RFC3339),
			t.Start.Add(time.Duration(points/2-1)*t.Step).Format(time.RFC3339),
			t.Start.Add(time.Duration(n-points/2)*t.Step).Format(time.RFC3339))
	}
	i0 := int(math.Floor(x)) - (points/2 - 1)
	i0 = max(0, min(i0, n-points))
	var ra, dec, km [points]float64
	ref := t.Samples[i0].RA
	for k := range points {
		s := t.Samples[i0+k]
		ra[k] = ref + math.Remainder(s.RA-ref, 360)
		dec[k], km[k] = s.Dec, s.Km
	}
	u := x - float64(i0)
	r := lagrange(ra[:], u)
	r = math.Mod(r, 360)
	if r < 0 {
		r += 360
	}
	return sample{RA: r, Dec: lagrange(dec[:], u), Km: lagrange(km[:], u)}, nil
}

// lagrange evaluates the polynomial through (k, y[k]) for k = 0..len(y)-1 at
// u.
func lagrange(y []float64, u float64) float64 {
	var sum float64
	for i := range y {
		w := 1.0
		for j := range y {
			if j != i {
				w *= (u - float64(j)) / float64(i-j)
			}
		}
		sum += w * y[i]
	}
	return sum
}
