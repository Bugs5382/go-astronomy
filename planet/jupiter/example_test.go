package jupiter_test

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
	"github.com/Bugs5382/go-astronomy/planet/jupiter"
)

// Where Jupiter is from Greenwich, how big and bright it looks, how much of it is
// lit, and how long its light took.
func ExamplePosition() {
	greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}
	when := time.Date(2027, 3, 1, 21, 0, 0, 0, time.UTC)
	r, err := jupiter.Position(greenwich, when)
	if err != nil {
		panic(err)
	}
	fmt.Printf("altitude %.2f, azimuth %.2f degrees\n", r.Altitude, r.Azimuth)
	fmt.Printf("RA %.4f, Dec %.4f degrees (true equator and equinox of date)\n", r.RA, r.Dec)
	fmt.Printf("distance %.4f au, light-time %v\n", r.DistanceAU, r.LightTime.Round(time.Second))
	fmt.Printf("diameter %.2f arcsec, magnitude %.2f\n", float64(r.Diameter)*3600, r.Magnitude)
	fmt.Printf("phase angle %.2f, %.1f%% lit, elongation %.1f, near Sun %v\n", r.PhaseAngle, 100*r.Illuminated, r.Elongation, r.NearSun)
	// Output:
	// altitude 48.06, azimuth 137.57 degrees
	// RA 142.4123, Dec 15.9411 degrees (true equator and equinox of date)
	// distance 4.4186 au, light-time 36m45s
	// diameter 44.62 arcsec, magnitude -2.52
	// phase angle 3.88, 99.9% lit, elongation 158.6, near Sun false
}

// Jupiter seen from the Sun, which belongs to no observer.
func ExampleHeliocentric() {
	h := jupiter.Heliocentric(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC))
	fmt.Printf("L %.4f B %.4f R %.6f au\n", h.Lon, h.Lat, h.DistanceAU)
	// Output:
	// L 138.8042 B 0.8031 R 5.335777 au
}

// Jupiter's rise, transit, and set at Greenwich from 1 September 2027.
func ExampleNextRise() {
	greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}
	rise, ok, err := jupiter.NextRise(greenwich, time.Date(2027, 9, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || !ok {
		panic(err)
	}
	transit, _, _ := jupiter.NextTransit(greenwich, rise)
	set, _, _ := jupiter.NextSet(greenwich, transit)
	fmt.Println("rise   ", rise.Format(time.DateTime))
	fmt.Println("transit", transit.Format(time.DateTime))
	fmt.Println("set    ", set.Format(time.DateTime))
	// Output:
	// rise    2027-09-01 05:06:53
	// transit 2027-09-01 11:58:08
	// set     2027-09-01 18:49:07
}

// A planet that never sets: Jupiter, high in Taurus, from Svalbard in January
// 2025.
func ExampleNextSet() {
	svalbard := astronomy.Observer{Lat: 78.22, Lng: 15.65, TZ: time.UTC}
	set, ok, err := jupiter.NextSet(svalbard, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		panic(err)
	}
	fmt.Println(ok, set.IsZero())
	// Output:
	// false true
}
