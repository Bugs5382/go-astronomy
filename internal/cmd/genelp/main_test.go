package main

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

// The heads of three ELP files, in their own fixed-width layouts.
const (
	elp1Head  = " MAIN PROBLEM. LONGITUDE(SINE)\n  0  0  0  2     -411.60287      168.48   -18433.81     -121.62        0.40       -0.18        0.00\n"
	elp4Head  = " EARTH FIGURE PERTURBATIONS. LONGITUDE\n  0  0  0  0  1 270.00000   0.00003     0.075\n"
	elp10Head = " PLANETARY PERTURBATIONS. TABLE 1. LONGITUDE\n  0  0  0  0  0  0  0  0  0  1 -2 359.98254   0.00007     0.074\n"
)

// TestParseMainProblem checks a main-problem line takes ELP82B_2's
// corrections of the constants and folds its Delaunay multipliers into the
// argument polynomial: 2F for the largest latitude-coupled longitude term.
func TestParseMainProblem(t *testing.T) {
	t.Parallel()
	a := newArgs()
	ts, err := parseFile(1, []byte(elp1Head), a)
	if err != nil || len(ts) != 1 {
		t.Fatalf("%v %v", ts, err)
	}
	got := ts[0]
	tgv := 168.48 + a.dtasm*(-0.18)
	want := -411.60287 + tgv*(a.delnp-am*a.delnu) + (-18433.81)*a.delg + (-121.62)*a.dele + 0.40*a.delep
	if math.Abs(got.a-want) > 1e-12 || got.coord != 0 || got.tpow != 0 || len(got.phase) != 5 {
		t.Errorf("term %+v, want amplitude %v", got, want)
	}
	for k := range 5 {
		if math.Abs(got.phase[k]-2*a.del[4][k]) > 1e-15 {
			t.Errorf("phase %d = %v, want 2F = %v", k, got.phase[k], 2*a.del[4][k])
		}
	}
}

// TestParsePerturbations checks the two perturbation layouts: an Earth
// figure term with its zeta multiplier and phase, and a planetary term. The
// last column of each is the period in years, which the theory does not use.
func TestParsePerturbations(t *testing.T) {
	t.Parallel()
	a := newArgs()
	ts, err := parseFile(4, []byte(elp4Head), a)
	if err != nil || len(ts) != 1 {
		t.Fatalf("%v %v", ts, err)
	}
	if got := ts[0]; got.a != 0.00003 || math.Abs(got.phase[0]-(270*deg+a.del[4][0])) > 1e-12 || math.Abs(got.phase[1]-a.del[4][1]) > 1e-12 {
		t.Errorf("ELP4 term %+v", got)
	}
	ts, err = parseFile(10, []byte(elp10Head), a)
	if err != nil || len(ts) != 1 {
		t.Fatalf("%v %v", ts, err)
	}
	want0 := 359.98254*deg + a.del[3][0] - 2*a.del[4][0]
	if got := ts[0]; got.a != 0.00007 || math.Abs(got.phase[0]-want0) > 1e-12 {
		t.Errorf("ELP10 term %+v, want phase %v", got, want0)
	}
	if !keep(term{coord: 0, a: 0.011}, 5e-8) || keep(term{coord: 0, a: 0.010}, 5e-8) {
		t.Error("truncation at 5e-8 rad (0.0103 arc second) is wrong")
	}
}
