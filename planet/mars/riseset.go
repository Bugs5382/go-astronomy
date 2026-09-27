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
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
)

// NextRise returns the first instant strictly after t at which Mars's centre
// rises above planet.HorizonAltitude for the observer, and true. It returns
// the zero time and false when no rise happens within 30 days, which can
// happen at high latitudes, and a coded error for an out-of-range observer.
func NextRise(obs astronomy.Observer, t time.Time) (time.Time, bool, error) {
	return spec.NextRise(obs, t)
}

// NextSet returns the first instant strictly after t at which Mars's centre
// sets below planet.HorizonAltitude, and true, as NextRise.
func NextSet(obs astronomy.Observer, t time.Time) (time.Time, bool, error) {
	return spec.NextSet(obs, t)
}

// NextTransit returns the first instant strictly after t at which Mars crosses
// the observer's meridian at its upper culmination, and true, whether or not
// it is above the horizon then.
func NextTransit(obs astronomy.Observer, t time.Time) (time.Time, bool, error) {
	return spec.NextTransit(obs, t)
}
