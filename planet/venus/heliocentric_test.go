package venus_test

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

	"github.com/Bugs5382/go-astronomy/internal/planettest"
	"github.com/Bugs5382/go-astronomy/planet/venus"
)

// TestHeliocentric checks Venus's heliocentric vector against Earth's and
// Position's distance.
func TestHeliocentric(t *testing.T) {
	t.Parallel()
	planettest.CheckHeliocentric(t, venus.Planet)
	when := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	if venus.Heliocentric(when) != venus.Planet.Heliocentric(when) {
		t.Error("Heliocentric and Planet.Heliocentric disagree")
	}
}

// TestHeliocentricMeeusExample checks Venus against Meeus example 32.a (1992
// December 20, 0h TD): L = 26.11428, B = -2.62070 degrees, R = 0.724603 au.
// The book evaluates the abridged series of its Appendix III, which differs
// from this table's truncation by under an arc second.
func TestHeliocentricMeeusExample(t *testing.T) {
	t.Parallel()
	// 0h TD is TT - UTC, 59.184 s in late 1992, before 0h UTC.
	h := venus.Heliocentric(time.Date(1992, 12, 19, 23, 59, 0, 816000000, time.UTC))
	if math.Abs(h.Lon-26.11428) > 0.001 || math.Abs(h.Lat+2.62070) > 0.001 || math.Abs(h.DistanceAU-0.724603) > 2e-6 {
		t.Errorf("Venus = %+v, want L 26.11428, B -2.62070, R 0.724603", h)
	}
}
