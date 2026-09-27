package satellite

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

// TestSDP4DeepSpace checks the deep-space satellites of the verification set
// on their own: every entry the initialisation routes to SDP4 (method 'd',
// periods of 225 minutes or more), against tcppver.out. TestVerificationSet
// covers the whole set; this isolates the routines in sdp4.go.
func TestSDP4DeepSpace(t *testing.T) {
	tles := readVerTLE(t)
	blocks := readTcppver(t)
	deep, compared := 0, 0
	var maxPos float64
	for i, tle := range tles {
		e, err := parseTLE(tle.line1, tle.line2, false)
		if err != nil {
			t.Fatal(err)
		}
		if e.rec.method != 'd' {
			continue
		}
		deep++
		want, hasErr := verErrors[i]
		for _, l := range blocks[i].lines {
			if hasErr && want.tsince == 0 {
				break
			}
			r, _, err := e.PropagateMinutes(l.tsince)
			if err != nil {
				continue
			}
			for k := range 3 {
				maxPos = math.Max(maxPos, math.Abs(r[k]-l.r[k]))
			}
			compared++
		}
	}
	t.Logf("%d deep-space satellites, %d states, max position error %.3g km", deep, compared, maxPos)
	if deep < 10 || compared < 200 {
		t.Fatalf("only %d deep-space satellites and %d states", deep, compared)
	}
	if maxPos > 1e-6 {
		t.Errorf("max position error %g km", maxPos)
	}
}
