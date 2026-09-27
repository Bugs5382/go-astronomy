package saturn_test

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
	"testing"
	"time"

	"github.com/Bugs5382/go-astronomy/internal/planettest"
	"github.com/Bugs5382/go-astronomy/planet/saturn"
)

// TestRiseSetAgainstHorizons checks rise, transit, and set against the
// Horizons fixture where it has Saturn events.
func TestRiseSetAgainstHorizons(t *testing.T) {
	t.Parallel()
	planettest.CheckRiseSetHorizons(t, saturn.Planet)
}

// TestRiseSetConsistency checks rise, transit, and set agree with Position.
func TestRiseSetConsistency(t *testing.T) {
	t.Parallel()
	planettest.CheckRiseSetConsistency(t, saturn.Planet)
}

// TestFunctionsMatchPlanetRiseSet checks the package functions and Planet
// agree.
func TestFunctionsMatchPlanetRiseSet(t *testing.T) {
	t.Parallel()
	from := time.Date(2027, 5, 1, 0, 0, 0, 0, time.UTC)
	a, _, _ := saturn.NextRise(planettest.Greenwich, from)
	b, _, _ := saturn.Planet.NextRise(planettest.Greenwich, from)
	c, _, _ := saturn.NextSet(planettest.Greenwich, from)
	d, _, _ := saturn.Planet.NextSet(planettest.Greenwich, from)
	e, _, _ := saturn.NextTransit(planettest.Greenwich, from)
	f, _, _ := saturn.Planet.NextTransit(planettest.Greenwich, from)
	if !a.Equal(b) || !c.Equal(d) || !e.Equal(f) {
		t.Errorf("functions %s %s %s, Planet %s %s %s", a, c, e, b, d, f)
	}
}
