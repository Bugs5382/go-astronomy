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
	"github.com/Bugs5382/go-astronomy/internal/planetary"
	"github.com/Bugs5382/go-astronomy/planet"
)

const (
	// Name is the planet's lowercase name.
	Name = "uranus"
	// RadiusKm is the IAU mean equatorial radius (Archinal et al. 2018),
	// which sizes the disc.
	RadiusKm = 25559
	// NearSunElongation is the elongation, in degrees, below which Position
	// flags Uranus NearSun: a guide for a planet that needs binoculars. It is a heuristic; the elongation and
	// the magnitude are always reported as well.
	NearSunElongation = 15
)

// spec describes Uranus to the shared reduction.
var spec = &planetary.Spec{
	PlanetName:        Name,
	Series:            &table,
	Radius:            RadiusKm,
	NearSun:           NearSunElongation,
	AbsoluteMagnitude: absoluteMagnitude,
}

// Planet is Uranus as a planet.Body, for code that handles every planet alike
// (see planet/all). Its methods are the package functions.
var Planet planet.Body = spec
