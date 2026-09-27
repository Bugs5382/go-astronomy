package mars

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
	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
	"github.com/Bugs5382/go-astronomy/internal/planetary"
)

// absoluteMagnitude returns Mars's V(1, alpha), the magnitude at unit distance
// from the Sun and the observer, from A. Mallama and J. L. Hilton, "Computing
// apparent planetary magnitudes for The Astronomical Almanac" (Astronomy and
// Computing 25, 10, 2018): two polynomials in the phase angle, split at 50 degrees; the small terms in the orbital longitude (a few hundredths of a magnitude) are left out.
func absoluteMagnitude(g planetary.Geometry) float64 {
	if g.Alpha <= 50 {
		return ephemeris.Horner(g.Alpha, -1.601, 2.267e-02, -1.302e-04)
	}
	return ephemeris.Horner(g.Alpha, -0.367, -0.02573, 3.445e-04)
}
