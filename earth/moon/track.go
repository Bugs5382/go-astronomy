package moon

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

// Sample is one point on the Moon's arc: the instant, its geometric topocentric
// altitude and azimuth in degrees, TimeProgress (the fraction of the sampled
// span elapsed at this point, from 0 at the first sample to 1 at the last), and
// the illuminated fraction and named phase at that instant.
type Sample struct {
	Time         time.Time
	Altitude     float64
	Azimuth      float64
	TimeProgress float64
	Illumination float64
	Phase        Phase
}

// Track samples the Moon's arc from from to to, inclusive of both endpoints,
// with exactly samples points evenly spaced in time. Each sample carries the
// geometric topocentric altitude and azimuth along with the illuminated fraction
// and named phase, which are cheap to add. Fewer than two samples cannot define a
// progress span, so Track returns nil in that case. It returns a go-apperr coded
// error when the observer's latitude or longitude is out of range.
func Track(obs astronomy.Observer, from, to time.Time, samples int) ([]Sample, error) {
	if err := validateObserver(obs); err != nil {
		return nil, err
	}
	if samples < 2 {
		return nil, nil
	}
	span := to.Sub(from)
	out := make([]Sample, samples)
	last := samples - 1
	for i := range out {
		progress := float64(i) / float64(last)
		var when time.Time
		if i == last {
			when = to // exact endpoint, free of rounding drift
		} else {
			when = from.Add(time.Duration(progress * float64(span)))
		}
		pos := position(obs, when)
		out[i] = Sample{
			Time:         when,
			Altitude:     pos.Altitude,
			Azimuth:      pos.Azimuth,
			TimeProgress: progress,
			Illumination: Illumination(when),
			Phase:        PhaseAt(when),
		}
	}
	return out, nil
}
