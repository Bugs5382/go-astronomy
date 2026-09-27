package satellite_test

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

	"github.com/Bugs5382/go-astronomy/satellite"
)

// TestMagnitudeStandard checks the standard magnitude is reproduced at 1000
// km and a 90 degree phase angle, and that a fuller phase is brighter.
func TestMagnitudeStandard(t *testing.T) {
	t.Parallel()
	l := satellite.Look{RangeKm: 1000, PhaseAngle: 90, Sunlit: true}
	if m := l.Magnitude(-1.8); math.Abs(m+1.8) > 1e-9 {
		t.Errorf("magnitude %.4f at the standard geometry, want -1.8", m)
	}
	l.PhaseAngle = 30
	if l.Magnitude(-1.8) >= -1.8 {
		t.Error("a fuller phase should be brighter")
	}
	l.Sunlit = false
	if !math.IsInf(l.Magnitude(-1.8), 1) {
		t.Error("a satellite in shadow should have no magnitude")
	}
}
