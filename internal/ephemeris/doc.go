// Package ephemeris implements the astronomical series this library needs,
// ported directly rather than pulled in as a dependency.
//
// # Provenance
//
// The algorithms are those of Jean Meeus, Astronomical Algorithms, 2nd edition
// (Willmann-Bell, 1998). Every file names the chapters and numbered formulae it
// implements, and every series table is reproduced from the book.
//
// The Go form of these algorithms derives from soniakeys/meeus
// (https://github.com/soniakeys/meeus, MIT), which this package replaces. That
// project is an accurate and near-complete rendering of the book, but it has
// been quiet upstream for years, and this library uses a small fraction of it.
// Owning the code removes an unmaintained dependency and lets corrections be
// applied here instead of worked around. Two defects in the ported source are
// fixed rather than carried over: the untyped integer-constant divisions that
// silently zeroed the higher-order terms of the lunar fundamental arguments,
// and the value and units of argument A1 in the lunar phase series. Both are
// noted at the code that fixes them.
//
// # Conventions
//
// Angles are radians internally and degrees at the boundary, which is where
// this package differs most from its source: it uses plain float64 throughout
// rather than a units type, and states the unit in every signature.
//
// Times are Julian ephemeris days (JDE). This library does not model Delta-T,
// so callers pass a Julian day derived from UTC and accept the resulting
// sub-arcminute error; see the accuracy note in AGENTS.md.
//
// The package is stateless, allocation-free on the hot paths, and safe for
// concurrent use.
package ephemeris

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
