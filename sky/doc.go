// Package sky places an observer on any body in the solar system and
// resolves any other body from there: Earth from Mars, the Sun from the
// Moon, Jupiter from Venus. It is the general form of what the earth,
// earth/moon, and planet packages do for an observer on Earth, and it sits
// beside them: nothing in the v1 surface changes.
//
// A Site is a latitude, an east longitude, and an optional height on a Body.
// The bodies are the package values Sun, Earth, and Moon, and any planet
// package's Planet through Planet(mars.Planet), which keeps the rule that a
// program links only the planetary tables it imports. Each Body carries what
// its sky depends on:
//
//   - its rotation: the IAU WGCCRE 2015 pole and prime meridian (the Earth
//     uses its full precession, nutation, and sidereal time instead);
//   - its shape: the IAU reference ellipsoid the site stands on;
//   - its refraction: 34 arc minutes at the horizon on Earth, and zero on the
//     Moon, which has no air, and on the planets, where it is not modelled;
//   - its twilight: the Earth's twilight bands, and none elsewhere, so an
//     airless day divides only at the Sun's horizon crossings.
//
// Position differences the target's heliocentric position (VSOP87, and the
// Meeus Moon) with the site's, corrects for light-time and for the aberration
// of the site's motion, and turns the direction into the site's local horizon.
// It reports the apparent direction, the disc's diameter, the distance and
// light-time, the phase angle and lit fraction, and the elongation from the
// Sun. NextRise, NextSet, and NextTransit scale their search to the body's day
// (a sol on Mars, a synodic month on the Moon) and report a target that never
// crosses the horizon plainly, as AlwaysAbove or AlwaysBelow: from most of the
// lunar near side, Earth neither rises nor sets. Segments divides a window of
// the site's day into bands.
//
// A Site on Earth answers through the v1 packages, so its positions and
// events are exactly theirs. Off Earth, against JPL Horizons (DE441), the
// direction agrees to under half an arc second from Mars and Venus, and to
// about 11 arc seconds from the Moon, where Horizons orients the site in the
// mean Earth frame of the DE441 libration and the IAU model approximates it.
//
// Every instant is a time.Time and comes back in UTC: a time zone is an Earth
// institution. Mars keeps its own clock; see mars.SolDate and
// mars.LocalMeanSolarTime. The package is stateless and safe for concurrent
// use, and it makes no network calls and writes no logs.
package sky

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
