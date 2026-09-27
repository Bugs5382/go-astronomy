// Package roman places the Nancy Grace Roman Space Telescope in an observer's
// sky, from JPL Horizons.
// Horizons lists it as -211. It launched on 2026-08-30 and is on its way to
// the Sun-Earth L2 point. Horizons carries only its published predicted
// trajectory, which ran to 2026-10-19 when this package was written; an
// instant past the end gives an error wrapping horizons.ErrOutsideCoverage.
//
// Two-line element sets and SGP4 do not apply out at L2: SGP4 models a
// satellite in Earth orbit, perturbed by the Earth's shape and drag, while an
// L2 telescope circles a point beyond the Moon under the Sun's and Earth's
// gravity. The ephemeris comes from Horizons instead, through an explicit
// horizons.Client that fetches a 30-day table once and interpolates locally.
//
// There is Position but no Passes. From over a million km the telescope moves
// across the stars by about a degree a day, near the point opposite the Sun,
// so it rises and sets once a day with the sky like a faint star; there are no
// minutes-long passes to predict, and it is far too faint to see without a
// large telescope.
package roman

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

import "github.com/Bugs5382/go-astronomy/satellite/horizons"

const (
	// Target is the Horizons command for the telescope.
	Target = "-211"
	// Name is the telescope's name.
	Name = "Nancy Grace Roman Space Telescope"
)

// New returns a tracker for the Nancy Grace Roman Space Telescope that fetches
// through c. Its
// Position(ctx, obs, t) gives the topocentric altitude, azimuth, RA and Dec of
// date, and range.
func New(c *horizons.Client) *horizons.Tracker {
	return horizons.NewTracker(Target, Name, c)
}
