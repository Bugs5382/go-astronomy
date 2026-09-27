package earth_test

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
	"testing"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
)

var (
	benchObs  = astronomy.Observer{Lat: 40.678, Lng: -73.944, TZ: time.UTC}
	benchTime = time.Date(2027, 6, 21, 15, 4, 5, 0, time.UTC)
	benchSink astronomy.Position
)

// BenchmarkSunPosition is one Sun position: the unit every sun sampling path
// repeats.
func BenchmarkSunPosition(b *testing.B) {
	for i := 0; b.Loop(); i++ {
		benchSink = earth.SunPosition(benchObs, benchTime.Add(time.Duration(i)*time.Second))
	}
}

// BenchmarkSunTrack is a 97-sample arc across a civil day.
func BenchmarkSunTrack(b *testing.B) {
	for b.Loop() {
		_ = earth.SunTrack(benchObs, benchTime, 97)
	}
}

// BenchmarkSunTrackFine is a one-minute track across a civil day, 1441
// samples, as an animated sky draws it.
func BenchmarkSunTrackFine(b *testing.B) {
	for b.Loop() {
		_ = earth.SunTrack(benchObs, benchTime, 1441)
	}
}

// BenchmarkNewSunTimes resolves one civil day of bands.
func BenchmarkNewSunTimes(b *testing.B) {
	for b.Loop() {
		if _, err := earth.NewSunTimes(benchObs, benchTime); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSegmentAt answers which band holds an instant, as a live clock
// does on every tick.
func BenchmarkSegmentAt(b *testing.B) {
	for b.Loop() {
		if _, _, err := earth.SegmentAt(benchObs, benchTime); err != nil {
			b.Fatal(err)
		}
	}
}
