package mars

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
	"time"

	"github.com/Bugs5382/go-astronomy/internal/julian"
)

// Sol is the mean solar day of Mars, 88775.244147 s (24h 39m 35.244s): the
// 1.0274912517 Earth days of the Mars24 algorithm, to the microsecond.
const Sol = 88775244147 * time.Microsecond

// solDays is the sol in Earth days, and msdEpoch the Julian ephemeris day at
// which the Mars Sol Date is zero (Allison and McEwen 2000, as revised for
// Mars24).
const (
	solDays  = 1.0274912517
	msdEpoch = 2405522.0028779
)

// SolDate returns the Mars Sol Date at instant t: the count of sols since
// 1873 December 29, as used by Mars24 and the mission clocks (Allison and
// McEwen 2000). It is 44795.9998 at 2000 January 6, 00:00 UTC.
func SolDate(t time.Time) float64 {
	return (julian.TT(t) - msdEpoch) / solDays
}

// LocalMeanSolarTime returns the local mean solar time on Mars at instant t
// for a site at the given east longitude, in Mars hours in [0, 24): the
// Coordinated Mars Time of the prime meridian (the fraction of the Mars Sol
// Date, in 24ths of a sol) plus the longitude at 15 degrees per Mars hour. A
// Mars hour is a 24th of a Sol, 1h 1m 39.8s.
func LocalMeanSolarTime(t time.Time, lonEastDeg float64) float64 {
	h := math.Mod(24*SolDate(t)+lonEastDeg/15, 24)
	if h < 0 {
		h += 24
	}
	return h
}
