package tiangong_test

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
	"github.com/Bugs5382/go-astronomy/satellite/tiangong"
)

// elements reads the element set CelesTrak served on 2026-09-26.
func elements() satellite.Elements {
	f, err := os.Open("testdata/gp-48274.json")
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

// The next passes over Madrid, from an element set the caller already has.
func ExampleNew() {
	e := elements()
	tracker := tiangong.New(satellite.StaticElements(e))
	obs := astronomy.Observer{Lat: 40.42, Lng: -3.70}
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
	// element set age: 12h13m0s
	// 06:22:16 to 06:31:07, peak 12.7 at 06:26:41, visible false
	// 07:57:55 to 08:08:20, peak 54.7 at 08:03:07, visible false
	// 09:34:45 to 09:45:16, peak 70.9 at 09:40:00, visible false
	// 11:11:41 to 11:22:12, peak 78.0 at 11:16:57, visible false
	// 12:48:42 to 12:58:23, peak 20.6 at 12:53:33, visible false
	// 14:27:41 to 14:31:32, peak  1.4 at 14:29:36, visible false
}

// Where it is at one instant.
func ExampleNew_position() {
	tracker := tiangong.New(satellite.StaticElements(elements()))
	obs := astronomy.Observer{Lat: 40.42, Lng: -3.70}
	l, err := tracker.Position(context.Background(), obs, time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC))
	if err != nil {
		panic(err)
	}
	fmt.Printf("altitude %.2f, azimuth %.2f, range %.0f km\n", l.Altitude, l.Azimuth, l.RangeKm)
	fmt.Printf("over %.2f, %.2f at %.0f km, sunlit %v\n", l.Latitude, l.Longitude, l.AltitudeKm, l.Sunlit)
	// Output:
	// altitude -79.63, azimuth 100.87, range 12933 km
	// over -41.51, 149.55 at 398 km, sunlit false
}

// With the element set from CelesTrak instead: fetched once, cached for a
// day, and propagated locally. (Not run as a test, which never reaches the
// network.)
func ExampleNew_celestrak() {
	tracker := tiangong.New(celestrak.New())
	l, err := tracker.Position(context.Background(), astronomy.Observer{Lat: 40.42, Lng: -3.70}, time.Now())
	if err != nil {
		fmt.Println("no element set:", err)
		return
	}
	fmt.Printf("altitude %.1f\n", l.Altitude)
}
