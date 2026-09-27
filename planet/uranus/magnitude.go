package uranus

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

	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
	"github.com/Bugs5382/go-astronomy/internal/planetary"
)

// absoluteMagnitude returns Uranus's V(1, alpha), the magnitude at unit distance
// from the Sun and the observer, from A. Mallama and J. L. Hilton, "Computing
// apparent planetary magnitudes for The Astronomical Almanac" (Astronomy and
// Computing 25, 10, 2018): a quadratic in the phase angle, plus the sub-Earth latitude term.
func absoluteMagnitude(g planetary.Geometry) float64 {
	// The sub-Earth latitude term uses the planetocentric latitude, which
	// differs from the planetographic one by under a degree here.
	phi := math.Abs(subEarthLatitude(g))
	return ephemeris.Horner(g.Alpha, -7.110, 6.587e-03, 1.045e-04) - 8.4e-04*phi
}
