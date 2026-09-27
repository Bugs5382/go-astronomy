// Package horizons fetches spacecraft ephemerides from the JPL Horizons API
// (https://ssd.jpl.nasa.gov/api/horizons.api), for objects SGP4 and two-line
// element sets cannot describe, such as the telescopes at the Sun-Earth L2
// point (satellite/jwst, satellite/roman). It is an explicit fetcher: nothing
// is fetched unless a caller builds a Client and passes it in.
//
// The network is hit rarely. One request fetches the geocentric apparent
// place over a whole window (30 days at a one-hour step by default), which is
// cached and interpolated locally with an eight-point Lagrange polynomial; a
// new request is made only when an instant falls outside the cached window.
// The window is observer-independent, and the topocentric correction and the
// altitude and azimuth are computed locally, so every observer shares one
// fetch. The cache is pluggable (satellite.Cache).
//
// Interpolating the hourly table of the James Webb Space Telescope agrees
// with a ten-minute Horizons table to 4.4 milliarcseconds and 0.7 m, and the
// local topocentric correction agrees with Horizons' own topocentric places
// to 0.2 arc seconds; see the package tests.
package horizons

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
