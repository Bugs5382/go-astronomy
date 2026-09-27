// Package satellite places Earth satellites, the International Space Station
// among them, in an observer's sky from an element set the caller supplies.
//
// An Earth satellite is not like the Sun or a planet: its position comes from
// a measured orbital element set that goes stale in days, so the element set
// is an input. The package parses two-line element sets (ParseTLE, including
// Alpha-5 catalogue numbers) and CCSDS Orbit Mean-Elements Messages in JSON
// and XML (ParseOMM), and never reaches the network: obtaining the elements is
// the caller's business, and Elements.Age says how old they are.
//
// Propagation is SGP4/SDP4, the model the element sets are fitted against,
// ported from the public-domain reference code of Vallado, Crawford, Hujsak
// and Kelso (2006) and checked against its verification output to under a
// millimetre. Position gives the topocentric altitude, azimuth, range, and
// range rate, whether the satellite is sunlit, the Sun's altitude for the
// observer, and the phase angle; Look.Magnitude turns a standard magnitude
// into an apparent one. Passes reports passes as intervals with rise, peak,
// and set, whether each is visible (the satellite sunlit while the observer
// is in darkness), and where it enters or leaves the Earth's shadow.
//
// The package is stateless and concurrency-safe: an Elements value can be
// shared between goroutines.
package satellite

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
