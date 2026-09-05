// Package sun provides universal Sun physics: quantities that depend only on the
// Sun itself, not on any observer, atmosphere, or vantage body. Its centerpiece
// is the Sun's apparent angular size as a function of distance, derived from the
// semidiameter at unit distance.
//
// Apparent solar quantities that depend on where the Sun is seen from, such as
// horizontal altitude and azimuth, solar noon, and twilight, are per-observer-body
// concerns and do not live here. They belong to a body package that supplies its
// own orbit, rotation, and distance to the Sun; the Earth vantage lives in the
// earth package. Keeping sun universal is what lets a future per-planet package
// reuse this same physics with its own distance and rotation.
//
// Angles are in degrees. The package is stateless and concurrency-safe.
package sun

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

import astronomy "github.com/Bugs5382/go-astronomy"

// SemidiameterArcsecAt1AU is the Sun's apparent angular semidiameter, in arc
// seconds, seen from a distance of one astronomical unit (Meeus, Astronomical
// Algorithms, chapter 55). It is a physical property of the Sun and is
// independent of any observer or vantage body. The apparent semidiameter at any
// distance scales inversely with that distance in AU.
const SemidiameterArcsecAt1AU = 959.63

// ApparentSemidiameter returns the Sun's apparent angular semidiameter, in
// degrees, seen from distanceAU astronomical units. It scales inversely with
// distance: at one AU it is SemidiameterArcsecAt1AU (converted to degrees). This
// is universal Sun physics, so any vantage body obtains the Sun's apparent size
// by supplying its own distance to the Sun.
func ApparentSemidiameter(distanceAU float64) float64 {
	return SemidiameterArcsecAt1AU / distanceAU / 3600
}

// ApparentDiameter returns the Sun's full apparent angular diameter, as an
// AngularDiameter in degrees, seen from distanceAU astronomical units. It is
// twice ApparentSemidiameter and, like it, depends only on the distance to the
// Sun, not on the observer.
func ApparentDiameter(distanceAU float64) astronomy.AngularDiameter {
	return astronomy.AngularDiameter(2 * ApparentSemidiameter(distanceAU))
}
