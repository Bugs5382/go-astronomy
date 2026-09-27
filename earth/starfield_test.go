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
	"sync"
	"testing"
	"time"

	apperr "github.com/Bugs5382/go-apperr"
	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
	"github.com/Bugs5382/go-astronomy/star"
)

// TestStarFieldIsStarPosition checks a StarField gives exactly StarPosition's
// answers, and that StarPosition gives the same answers when goroutines at
// different instants interleave, so the remembered frame never leaks from one
// instant into another.
func TestStarFieldIsStarPosition(t *testing.T) {
	t.Parallel()
	obs := astronomy.Observer{Lat: 51.48, Lng: -0.0015}
	stars := star.Brighter(2)
	instants := []time.Time{
		time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2027, 1, 1, 0, 0, 0, 1, time.UTC),
		time.Date(2031, 8, 12, 21, 30, 0, 0, time.UTC),
	}
	want := map[time.Time][]astronomy.Horizontal{}
	for _, when := range instants {
		f, err := earth.NewStarField(obs, when)
		if err != nil {
			t.Fatal(err)
		}
		for _, s := range stars {
			want[when] = append(want[when], f.Position(s))
		}
	}
	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for rep := 0; rep < 20; rep++ {
				when := instants[(g+rep)%len(instants)]
				for i, s := range stars {
					got, err := earth.StarPosition(s, obs, when)
					if err != nil || got != want[when][i] {
						t.Errorf("%s at %s: %+v %v, want %+v", s.ProperName, when, got, err, want[when][i])
						return
					}
				}
			}
		}(g)
	}
	wg.Wait()
	if _, err := earth.NewStarField(astronomy.Observer{Lat: 91}, instants[0]); err == nil {
		t.Error("NewStarField took latitude 91")
	} else if code, _ := apperr.Code(err); code != astronomy.CodeInvalidLatitude {
		t.Errorf("code %d", code)
	}
}

var starSink astronomy.Horizontal

// BenchmarkStarPosition is one star at one instant, as a loop over the
// catalog at one instant calls it: the instant's frame is remembered.
func BenchmarkStarPosition(b *testing.B) {
	s, _ := star.GetNamedStar("Sirius")
	obs := astronomy.Observer{Lat: 40.678, Lng: -73.944}
	when := time.Date(2027, 6, 21, 3, 0, 0, 0, time.UTC)
	for b.Loop() {
		starSink, _ = earth.StarPosition(s, obs, when)
	}
}

// BenchmarkStarPositionNewInstant is one star at a new instant every call,
// as a star's track is drawn: the frame is built every time.
func BenchmarkStarPositionNewInstant(b *testing.B) {
	s, _ := star.GetNamedStar("Sirius")
	obs := astronomy.Observer{Lat: 40.678, Lng: -73.944}
	when := time.Date(2027, 6, 21, 3, 0, 0, 0, time.UTC)
	for i := 0; b.Loop(); i++ {
		starSink, _ = earth.StarPosition(s, obs, when.Add(time.Duration(i)*time.Second))
	}
}

// BenchmarkStarFieldSky is the whole embedded catalog at one instant.
func BenchmarkStarFieldSky(b *testing.B) {
	all := star.All()
	obs := astronomy.Observer{Lat: 40.678, Lng: -73.944}
	when := time.Date(2027, 6, 21, 3, 0, 0, 0, time.UTC)
	for b.Loop() {
		f, _ := earth.NewStarField(obs, when)
		for _, s := range all {
			starSink = f.Position(s)
		}
	}
}
