package planetary

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
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/internal/angles"
	"github.com/Bugs5382/go-astronomy/internal/coordinates"
	"github.com/Bugs5382/go-astronomy/internal/ephemeris"
	"github.com/Bugs5382/go-astronomy/internal/julian"
	"github.com/Bugs5382/go-astronomy/planet"
)

// position is Position without the observer check.
func (s *Spec) position(obs astronomy.Observer, t time.Time) planet.Result {
	jde := julian.TT(t)
	g := s.geometryAt(jde)

	ra, dec := ephemeris.EclToEq(g.Lon, g.Lat, ephemeris.TrueObliquity(jde))
	gst := julian.ApparentSiderealTime(t)
	ra, dec, topoKm := ephemeris.Topocentric(ra, dec, g.Delta*ephemeris.KmPerAU, obs.Lat, 0, gst+obs.Lng)
	hz := coordinates.EquatorialToHorizontal(coordinates.Equatorial{RA: ra, Dec: dec}, obs.Lat, obs.Lng, gst)

	// Elongation from the triangle Sun, Earth, planet (Meeus 48.2 in its
	// distance form).
	cosE := (g.EarthR*g.EarthR + g.Delta*g.Delta - g.R*g.R) / (2 * g.EarthR * g.Delta)
	elong := angles.RadToDeg(math.Acos(Clamp(cosE)))
	diam := 2 * angles.RadToDeg(math.Asin(s.Radius/topoKm))

	return planet.Result{
		Position: astronomy.Position{
			Horizontal: astronomy.Horizontal{Altitude: hz.Altitude, Azimuth: hz.Azimuth},
			Diameter:   astronomy.AngularDiameter(diam),
		},
		RA:          ra,
		Dec:         dec,
		DistanceAU:  topoKm / ephemeris.KmPerAU,
		LightTime:   time.Duration(g.Tau * 86400 * float64(time.Second)),
		Magnitude:   s.magnitude(g),
		PhaseAngle:  g.Alpha,
		Illuminated: (1 + math.Cos(angles.DegToRad(g.Alpha))) / 2,
		Elongation:  elong,
		NearSun:     elong < s.NearSun,
	}
}
