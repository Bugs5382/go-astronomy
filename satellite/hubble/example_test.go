package hubble_test

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
	"context"
	"fmt"
	"os"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/satellite"
	"github.com/Bugs5382/go-astronomy/satellite/celestrak"
	"github.com/Bugs5382/go-astronomy/satellite/hubble"
)

// elements reads the element set CelesTrak served on 2026-09-26.
func elements() satellite.Elements {
	f, err := os.Open("testdata/gp-20580.json")
	if err != nil {
		panic(err)
	}
	defer func() { _ = f.Close() }()
	sets, err := satellite.ParseOMM(f)
	if err != nil {
		panic(err)
	}
	return sets[0]
}

// The next passes over Nairobi, from an element set the caller already has.
func ExampleNew() {
	e := elements()
	tracker := hubble.New(satellite.StaticElements(e))
	obs := astronomy.Observer{Lat: -1.29, Lng: 36.82}
	from := time.Date(2026, 9, 27, 0, 0, 0, 0, time.UTC)
	passes, err := tracker.Passes(context.Background(), obs, from, from.Add(24*time.Hour))
	if err != nil {
		panic(err)
	}
	fmt.Println("element set age:", e.Age(from).Round(time.Minute))
	for _, p := range passes {
		fmt.Printf("%s to %s, peak %4.1f at %s, visible %v\n",
			p.Rise.Time.Format("15:04:05"), p.Set.Time.Format("15:04:05"),
			p.Peak.Altitude, p.Peak.Time.Format("15:04:05"), p.Visible)
	}
	// Output:
	// element set age: 18h32m0s
	// 01:34:25 to 01:42:52, peak  7.3 at 01:38:39, visible true
	// 03:12:38 to 03:24:15, peak 43.4 at 03:18:27, visible false
	// 04:52:31 to 05:03:30, peak 22.8 at 04:58:01, visible false
	// 06:35:15 to 06:40:38, peak  2.3 at 06:37:56, visible false
	// 13:19:11 to 13:25:52, peak  3.9 at 13:22:31, visible false
	// 14:56:44 to 15:08:00, peak 29.2 at 15:02:22, visible false
	// 16:36:14 to 16:47:38, peak 32.8 at 16:41:55, visible true
	// 18:18:01 to 18:25:36, peak  5.4 at 18:21:48, visible false
}

// Where it is at one instant.
func ExampleNew_position() {
	tracker := hubble.New(satellite.StaticElements(elements()))
	obs := astronomy.Observer{Lat: -1.29, Lng: 36.82}
	l, err := tracker.Position(context.Background(), obs, time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC))
	if err != nil {
		panic(err)
	}
	fmt.Printf("altitude %.2f, azimuth %.2f, range %.0f km\n", l.Altitude, l.Azimuth, l.RangeKm)
	fmt.Printf("over %.2f, %.2f at %.0f km, sunlit %v\n", l.Latitude, l.Longitude, l.AltitudeKm, l.Sunlit)
	// Output:
	// altitude -30.76, azimuth 95.28, range 7366 km
	// over -5.40, 104.43 at 469 km, sunlit true
}

// With the element set from CelesTrak instead: fetched once, cached for a
// day, and propagated locally. (Not run as a test, which never reaches the
// network.)
func ExampleNew_celestrak() {
	tracker := hubble.New(celestrak.New())
	l, err := tracker.Position(context.Background(), astronomy.Observer{Lat: -1.29, Lng: 36.82}, time.Now())
	if err != nil {
		fmt.Println("no element set:", err)
		return
	}
	fmt.Printf("altitude %.1f\n", l.Altitude)
}
