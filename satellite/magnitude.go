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

// Apparent magnitude from a standard magnitude and the phase angle.

import "math"

// Magnitude returns the satellite's apparent visual magnitude for a standard
// magnitude stdMag, the magnitude at 1000 km with half the disc lit (a 90
// degree phase angle), as published by McCants and Heavens-Above (about -1.8
// for the ISS; see ISSStandardMagnitude). The satellite is modelled as a
// diffusely reflecting sphere. It returns +Inf when the satellite is not
// sunlit.
func (l Look) Magnitude(stdMag float64) float64 {
	if !l.Sunlit {
		return math.Inf(1)
	}
	b := l.PhaseAngle * math.Pi / 180
	f := math.Sin(b) + (math.Pi-b)*math.Cos(b)
	if f <= 0 {
		return math.Inf(1)
	}
	return stdMag + 5*math.Log10(l.RangeKm/1000) - 2.5*math.Log10(f)
}

// ISSStandardMagnitude is the International Space Station's standard
// magnitude, for Look.Magnitude and Pass magnitudes. A satellite's standard
// magnitude is not part of its element set; the caller supplies it.
const ISSStandardMagnitude = -1.8
