package moon_test

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
	"github.com/Bugs5382/go-astronomy/earth/moon"
)

// The Moon from Brooklyn: where it is, how big it looks, and when it next
// rises and sets.
func ExamplePosition() {
	brooklyn := astronomy.Observer{Lat: 40.678, Lng: -73.944, TZ: time.UTC}
	when := time.Date(2027, 6, 21, 3, 0, 0, 0, time.UTC)

	pos, err := moon.Position(brooklyn, when)
	if err != nil {
		panic(err)
	}
	fmt.Printf("altitude %.4f, azimuth %.4f, %.4f degrees across, %.0f%% lit\n",
		pos.Altitude, pos.Azimuth, float64(pos.Diameter), moon.Illumination(when)*100)

	rise, _, _ := moon.NextRise(brooklyn, when)
	set, _, _ := moon.NextSet(brooklyn, rise)
	fmt.Println("moonrise:", rise.Format("2006-01-02 15:04:05 MST"))
	fmt.Println("moonset: ", set.Format("2006-01-02 15:04:05 MST"))
	// Output:
	// altitude 7.5488, azimuth 130.8081, 0.4924 degrees across, 96% lit
	// moonrise: 2027-06-22 02:35:22 UTC
	// moonset:  2027-06-22 12:34:07 UTC
}
