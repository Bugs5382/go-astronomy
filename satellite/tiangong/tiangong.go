// Package tiangong follows the Tiangong space station (the Tianhe core module carries the station's catalogue entry): its position in an observer's sky and its
// passes, propagated with SGP4 from an element set.
//
// The package fixes the NORAD catalogue number and name and never fetches
// anything itself. The caller passes an explicit element source: its own TLE
// or OMM (satellite.StaticElements), or the CelesTrak fetcher
// (satellite/celestrak), which caches the set and hits the network at most
// every few hours. Every position and pass is then computed locally.
package tiangong

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

	"github.com/Bugs5382/go-astronomy/satellite"
)

const (
	// CatalogNumber is the NORAD catalogue number.
	CatalogNumber = 48274
	// Name is the name CelesTrak gives the object.
	Name = "CSS (TIANHE)"
)

// StandardMagnitude is NaN: no published standard magnitude for the assembled station is used here, so Look.Magnitude and pass magnitudes are left out (NaN) rather than guessed.
var StandardMagnitude = math.NaN()

// New returns a tracker for the Tiangong space station (the Tianhe core module carries the station's catalogue entry) that takes its element sets from src,
// for example satellite.StaticElements(elements) or a *celestrak.Client.
// Its Position(ctx, obs, t) and Passes(ctx, obs, from, to) propagate
// locally; a stale set from the source is used and reported with an error
// wrapping satellite.ErrStaleElements.
func New(src satellite.ElementSource) *satellite.Tracker {
	return satellite.NewTracker(CatalogNumber, Name, StandardMagnitude, src)
}
