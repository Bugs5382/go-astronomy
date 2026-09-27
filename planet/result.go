package planet

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
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
)

// Result is a planet as an observer sees it.
type Result struct {
	// Position is the topocentric direction to the centre of the disc
	// (geometric altitude, without refraction, and azimuth, in degrees),
	// paired with the apparent angular diameter, sized from the same
	// observer-to-planet distance.
	astronomy.Position
	// RA and Dec are the topocentric apparent right ascension, in [0, 360),
	// and declination, in degrees, referred to the true equator and equinox
	// of date.
	RA, Dec float64
	// DistanceAU is the distance from the observer, in astronomical units.
	DistanceAU float64
	// LightTime is how long ago the light now arriving left the planet.
	LightTime time.Duration
	// Magnitude is the apparent visual magnitude (Mallama and Hilton 2018).
	Magnitude float64
	// PhaseAngle is the Sun-planet-observer angle, in degrees: 0 when the disc
	// is fully lit.
	PhaseAngle float64
	// Illuminated is the fraction of the disc that is lit, in [0, 1].
	Illuminated float64
	// Elongation is the angle between the planet and the Sun as seen from the
	// Earth, in degrees, [0, 180].
	Elongation float64
	// NearSun reports that the elongation is below the planet's
	// NearSunElongation, so the planet is probably lost in the Sun's glare
	// whatever its altitude.
	NearSun bool
}
