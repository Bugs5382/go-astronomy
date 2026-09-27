// Package planet holds the types every planet package shares: the result of
// placing a planet for an observer, the heliocentric position, the Body
// interface each planet satisfies, and the errors they return. It holds no
// planetary tables.
//
// Each planet is its own package, planet/mercury through planet/neptune, with
// its own generated VSOP87 table and the same small API: Position,
// Heliocentric, NextRise, NextSet, and NextTransit, plus a Planet value that
// implements Body. A program that imports planet/mars links only the Mars
// table (and the Earth table the Sun already needs). planet/all imports every
// planet for a caller who wants to iterate over them.
//
// Positions come from the VSOP87 theory (Bretagnon and Francou 1988), version
// D, reduced as in Meeus, Astronomical Algorithms, chapter 33: light-time and
// aberration, the FK5 correction, nutation, and the rigorous topocentric
// correction. Magnitudes follow Mallama and Hilton (2018).
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
