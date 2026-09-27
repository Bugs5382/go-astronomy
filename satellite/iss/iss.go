// Package iss follows the International Space Station: its position in an observer's sky and its
// passes, propagated with SGP4 from an element set.
//
// The package fixes the NORAD catalogue number and name and never fetches
// anything itself. The caller passes an explicit element source: its own TLE
// or OMM (satellite.StaticElements), or the CelesTrak fetcher
// (satellite/celestrak), which caches the set and hits the network at most
// every few hours. Every position and pass is then computed locally.
package iss

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
	"github.com/Bugs5382/go-astronomy/satellite"
)

const (
	// CatalogNumber is the NORAD catalogue number.
	CatalogNumber = 25544
	// Name is the name CelesTrak gives the object.
	Name = "ISS (ZARYA)"
)

// StandardMagnitude is -1.8, the International Space Station's standard magnitude (McCants), so passes carry magnitudes.
var StandardMagnitude = satellite.ISSStandardMagnitude

// New returns a tracker for the International Space Station that takes its element sets from src,
// for example satellite.StaticElements(elements) or a *celestrak.Client.
// Its Position(ctx, obs, t) and Passes(ctx, obs, from, to) propagate
// locally, and each result's ElementEpoch says how old the set behind it is.
func New(src satellite.ElementSource) *satellite.Tracker {
	return satellite.NewTracker(CatalogNumber, Name, StandardMagnitude, src)
}
