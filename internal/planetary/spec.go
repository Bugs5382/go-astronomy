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
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/internal/julian"
	"github.com/Bugs5382/go-astronomy/internal/vsop87"
	"github.com/Bugs5382/go-astronomy/planet"
)

// Spec is what the engine needs to know about one planet.
type Spec struct {
	// PlanetName is the lowercase name.
	PlanetName string
	// Series is the planet's VSOP87D table.
	Series *vsop87.Body
	// Radius is the IAU mean equatorial radius in km, which sizes the disc.
	Radius float64
	// NearSun is the elongation, in degrees, below which Position flags the
	// planet NearSun.
	NearSun float64
	// AbsoluteMagnitude returns V(1, alpha), the magnitude at unit distance
	// from the Sun and the observer, for the geometry of one observation.
	AbsoluteMagnitude func(g Geometry) float64
}

var _ planet.Body = (*Spec)(nil)

// Name returns the planet's name.
func (s *Spec) Name() string { return s.PlanetName }

// RadiusKm returns the planet's radius in km.
func (s *Spec) RadiusKm() float64 { return s.Radius }

// NearSunElongation returns the near-Sun limit in degrees.
func (s *Spec) NearSunElongation() float64 { return s.NearSun }

// Heliocentric returns the planet's heliocentric position at t.
func (s *Spec) Heliocentric(t time.Time) planet.HeliocentricPosition {
	l, b, r := s.Series.Heliocentric(julian.TT(t))
	return planet.HeliocentricPosition{Lon: l, Lat: b, DistanceAU: r}
}

// Position places the planet for the observer at t.
func (s *Spec) Position(obs astronomy.Observer, t time.Time) (planet.Result, error) {
	if err := validateObserver(obs); err != nil {
		return planet.Result{}, err
	}
	return s.position(obs, t), nil
}
