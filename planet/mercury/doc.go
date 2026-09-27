// Package mercury places Mercury in an observer's sky: topocentric apparent position,
// apparent diameter, phase, magnitude, elongation from the Sun, rise, set, and
// transit, and Mercury's heliocentric position. The results are the shared types
// of package planet, and Planet implements planet.Body.
//
// The package holds only Mercury's own VSOP87 table (vsop87.go, generated from
// the IMCCE file by internal/cmd/genvsop), so a program that imports it links
// no other planet's table. The reduction is shared with the other planet
// packages and follows Meeus, Astronomical Algorithms, chapter 33.
package mercury

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

//go:generate go run ../../internal/cmd/genvsop -body mercury -var table -pkg mercury -out vsop87.go
