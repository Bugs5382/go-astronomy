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
	"fmt"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/earth"
)

// The dip of the sea horizon for an observer at Denver's height.
func ExampleHorizonDip() {
	fmt.Printf("%.3f degrees\n", earth.HorizonDip(1609))
	fmt.Printf("%.3f degrees below sea level\n", earth.HorizonDip(-430))
	// Output:
	// 1.177 degrees
	// 0.000 degrees below sea level
}

// Height moves sunrise: the same day at Denver, at sea level and at the
// city's 1609 m.
func Example_observerElevation() {
	sea := astronomy.Observer{Lat: 39.74, Lng: -104.99, TZ: time.UTC}
	high := sea
	high.Elevation = 1609
	date := time.Date(2027, 6, 21, 0, 0, 0, 0, time.UTC)

	sunrise := func(obs astronomy.Observer) time.Time {
		day, err := earth.NewSunTimes(obs, date)
		if err != nil {
			panic(err)
		}
		for _, s := range day.Segments() {
			if s.Label == earth.LabelSunrise {
				return s.From
			}
		}
		return time.Time{}
	}
	a, b := sunrise(sea), sunrise(high)
	fmt.Println("sea level:", a.Format("15:04 MST"))
	fmt.Println("1609 m:   ", b.Format("15:04 MST"))
	fmt.Printf("earlier by %.1f minutes\n", a.Sub(b).Minutes())
	// Output:
	// sea level: 11:32 UTC
	// 1609 m:    11:24 UTC
	// earlier by 7.3 minutes
}

// Twilight stays on the astronomical horizon by default; WithTwilightDip
// moves it with the horizon the observer sees.
func ExampleSegmentation_WithTwilightDip() {
	obs := astronomy.Observer{Lat: 39.74, Lng: -104.99, TZ: time.UTC, Elevation: 1609}
	date := time.Date(2027, 3, 20, 0, 0, 0, 0, time.UTC)
	civilDawn := func(seg earth.Segmentation) time.Time {
		day, err := earth.NewSunTimesWith(obs, date, seg)
		if err != nil {
			panic(err)
		}
		for _, s := range day.Segments() {
			if s.Label == earth.LabelCivilDawn {
				return s.From
			}
		}
		return time.Time{}
	}
	usno := civilDawn(earth.DefaultSegmentation)
	dipped := civilDawn(earth.DefaultSegmentation.WithTwilightDip())
	fmt.Printf("civil dawn moves %.1f minutes earlier\n", usno.Sub(dipped).Minutes())
	// Output:
	// civil dawn moves 6.1 minutes earlier
}
