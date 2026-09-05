package angles

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

const tol = 1e-12

func TestDegToRad(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		deg  float64
		want float64
	}{
		{"zero", 0, 0},
		{"right angle", 90, math.Pi / 2},
		{"straight angle", 180, math.Pi},
		{"full turn", 360, 2 * math.Pi},
		{"negative", -90, -math.Pi / 2},
	}
	for _, c := range cases {
		if got := DegToRad(c.deg); math.Abs(got-c.want) > tol {
			t.Errorf("DegToRad(%v) = %v, want %v", c.deg, got, c.want)
		}
	}
}

func TestRadToDeg(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		rad  float64
		want float64
	}{
		{"zero", 0, 0},
		{"right angle", math.Pi / 2, 90},
		{"straight angle", math.Pi, 180},
		{"full turn", 2 * math.Pi, 360},
		{"negative", -math.Pi / 2, -90},
	}
	for _, c := range cases {
		if got := RadToDeg(c.rad); math.Abs(got-c.want) > tol {
			t.Errorf("RadToDeg(%v) = %v, want %v", c.rad, got, c.want)
		}
	}
}

func TestDegRadRoundTrip(t *testing.T) {
	t.Parallel()
	for deg := -720.0; deg <= 720.0; deg += 13.7 {
		if got := RadToDeg(DegToRad(deg)); math.Abs(got-deg) > 1e-9 {
			t.Errorf("round trip for %v = %v", deg, got)
		}
	}
}

func TestNormalize(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   float64
		want float64
	}{
		{"already in range", 12.5, 12.5},
		{"zero", 0, 0},
		{"exactly 360", 360, 0},
		{"just under 360", 359.999, 359.999},
		{"over 360", 370, 10},
		{"large positive", 360*3 + 45, 45},
		{"negative", -10, 350},
		{"negative multiple", -360*2 - 30, 330},
		{"negative 360", -360, 0},
		// A tiny negative input is lost when added to 360, producing exactly
		// 360.0; the result must still fold back into range.
		{"tiny negative", -1e-16, 0},
	}
	for _, c := range cases {
		got := Normalize(c.in)
		if math.Abs(got-c.want) > 1e-9 {
			t.Errorf("Normalize(%v) = %v, want %v", c.in, got, c.want)
		}
		if got < 0 || got >= 360 {
			t.Errorf("Normalize(%v) = %v, out of [0,360)", c.in, got)
		}
	}
}

func TestClampDeclination(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		in   float64
		want float64
	}{
		{"in range", 23.5, 23.5},
		{"zero", 0, 0},
		{"upper edge", 90, 90},
		{"lower edge", -90, -90},
		{"above pole", 95.2, 90},
		{"below pole", -120, -90},
	}
	for _, c := range cases {
		if got := ClampDeclination(c.in); got != c.want {
			t.Errorf("ClampDeclination(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}
