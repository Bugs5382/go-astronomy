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

// Body is one planet. Every planet package exports a Planet value that
// implements it, and planet/all lists them all. The methods mirror the
// package-level functions of the planet's own package.
type Body interface {
	// Name is the planet's lowercase name, such as "mars".
	Name() string
	// RadiusKm is the IAU mean equatorial radius, which sizes the disc.
	RadiusKm() float64
	// NearSunElongation is the elongation, in degrees, below which Position
	// flags the planet NearSun.
	NearSunElongation() float64
	// Position places the planet for an observer at an instant.
	Position(obs astronomy.Observer, t time.Time) (Result, error)
	// Heliocentric is the planet's position seen from the Sun's centre.
	Heliocentric(t time.Time) HeliocentricPosition
	// NextRise, NextSet, and NextTransit find the next event after t, and
	// report false when none happens within 30 days.
	NextRise(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
	NextSet(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
	NextTransit(obs astronomy.Observer, t time.Time) (time.Time, bool, error)
}
