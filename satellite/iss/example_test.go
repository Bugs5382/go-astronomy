package iss_test

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
	"github.com/Bugs5382/go-astronomy/satellite/iss"
)

// elements reads the element set CelesTrak served on 2026-09-26.
func elements() satellite.Elements {
	f, err := os.Open("testdata/gp-25544.json")
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

// The next passes over Denver, from an element set the caller already has.
func ExampleNew() {
	e := elements()
	tracker := iss.New(satellite.StaticElements(e))
	obs := astronomy.Observer{Lat: 39.74, Lng: -104.99}
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
	// element set age: 3h34m0s
	// 23:53:27 to 00:02:52, peak 13.1 at 23:58:10, visible false
	// 01:30:21 to 01:41:05, peak 38.6 at 01:35:44, visible true
	// 03:07:20 to 03:17:28, peak 23.2 at 03:12:25, visible true
	// 18:13:46 to 18:23:40, peak 18.7 at 18:18:42, visible false
	// 19:49:52 to 20:00:45, peak 47.6 at 19:55:18, visible false
	// 21:27:59 to 21:37:34, peak 14.2 at 21:32:47, visible false
	// 23:06:10 to 23:15:20, peak 11.7 at 23:10:46, visible false
}

// Where it is at one instant.
func ExampleNew_position() {
	tracker := iss.New(satellite.StaticElements(elements()))
	obs := astronomy.Observer{Lat: 39.74, Lng: -104.99}
	l, err := tracker.Position(context.Background(), obs, time.Date(2026, 9, 27, 12, 0, 0, 0, time.UTC))
	if err != nil {
		panic(err)
	}
	fmt.Printf("altitude %.2f, azimuth %.2f, range %.0f km\n", l.Altitude, l.Azimuth, l.RangeKm)
	fmt.Printf("over %.2f, %.2f at %.0f km, sunlit %v\n", l.Latitude, l.Longitude, l.AltitudeKm, l.Sunlit)
	// Output:
	// altitude -35.80, azimuth 83.15, range 8155 km
	// over 13.80, -21.41 at 425 km, sunlit true
}

// With the element set from CelesTrak instead: fetched once, cached for a
// day, and propagated locally. (Not run as a test, which never reaches the
// network.)
func ExampleNew_celestrak() {
	tracker := iss.New(celestrak.New())
	l, err := tracker.Position(context.Background(), astronomy.Observer{Lat: 39.74, Lng: -104.99}, time.Now())
	if err != nil {
		fmt.Println("no element set:", err)
		return
	}
	fmt.Printf("altitude %.1f\n", l.Altitude)
}
