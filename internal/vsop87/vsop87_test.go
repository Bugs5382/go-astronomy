package vsop87

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

// TestSum checks the series sum on a hand-built table: a constant, a term in
// tau, and a periodic term multiplied by tau squared.
func TestSum(t *testing.T) {
	t.Parallel()
	series := [][]Term{
		{{A: 1.5}},
		{{A: 2, B: 0, C: 0}},
		{{A: 3, B: math.Pi / 2, C: 1}},
	}
	for _, tau := range []float64{0, 0.1, -0.2, 0.3} {
		want := 1.5 + 2*tau + 3*math.Cos(math.Pi/2+tau)*tau*tau
		if got := Sum(series, tau); math.Abs(got-want) > 1e-15 {
			t.Errorf("Sum(tau %v) = %v, want %v", tau, got, want)
		}
	}
	if Sum(nil, 0.5) != 0 {
		t.Error("an empty series sums to zero")
	}
}

// TestHeliocentric checks the unit conversion and the longitude reduction.
func TestHeliocentric(t *testing.T) {
	t.Parallel()
	b := &Body{
		L: [][]Term{{{A: -math.Pi / 2}}},
		B: [][]Term{{{A: math.Pi / 180}}},
		R: [][]Term{{{A: 1.25}}},
	}
	l, lat, r := b.Heliocentric(j2000)
	if math.Abs(l-270) > 1e-12 || math.Abs(lat-1) > 1e-12 || r != 1.25 {
		t.Errorf("Heliocentric = %v %v %v, want 270 1 1.25", l, lat, r)
	}
	if Millennia(j2000+millennium) != 1 {
		t.Error("one millennium after J2000 is tau 1")
	}
}
