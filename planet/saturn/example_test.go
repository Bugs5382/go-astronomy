package saturn_test

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
	"github.com/Bugs5382/go-astronomy/planet/saturn"
)

// Where Saturn is from Greenwich, how big and bright it looks, how much of it is
// lit, and how long its light took.
func ExamplePosition() {
	greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}
	when := time.Date(2027, 3, 1, 21, 0, 0, 0, time.UTC)
	r, err := saturn.Position(greenwich, when)
	if err != nil {
		panic(err)
	}
	fmt.Printf("altitude %.2f, azimuth %.2f degrees\n", r.Altitude, r.Azimuth)
	fmt.Printf("RA %.4f, Dec %.4f degrees (true equator and equinox of date)\n", r.RA, r.Dec)
	fmt.Printf("distance %.4f au, light-time %v\n", r.DistanceAU, r.LightTime.Round(time.Second))
	fmt.Printf("diameter %.2f arcsec, magnitude %.2f\n", float64(r.Diameter)*3600, r.Magnitude)
	fmt.Printf("phase angle %.2f, %.1f%% lit, elongation %.1f, near Sun %v\n", r.PhaseAngle, 100*r.Illuminated, r.Elongation, r.NearSun)
	// Output:
	// altitude -4.64, azimuth 280.92 degrees
	// RA 13.0027, Dec 3.1127 degrees (true equator and equinox of date)
	// distance 10.2124 au, light-time 1h24m56s
	// diameter 16.27 arcsec, magnitude 0.80
	// phase angle 3.22, 99.9% lit, elongation 32.2, near Sun false
}

// Saturn seen from the Sun, which belongs to no observer.
func ExampleHeliocentric() {
	h := saturn.Heliocentric(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC))
	fmt.Printf("L %.4f B %.4f R %.6f au\n", h.Lon, h.Lat, h.DistanceAU)
	// Output:
	// L 14.3336 B -2.4514 R 9.406765 au
}

// Saturn's rise, transit, and set at Greenwich from 1 September 2027.
func ExampleNextRise() {
	greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}
	rise, ok, err := saturn.NextRise(greenwich, time.Date(2027, 9, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || !ok {
		panic(err)
	}
	transit, _, _ := saturn.NextTransit(greenwich, rise)
	set, _, _ := saturn.NextSet(greenwich, transit)
	fmt.Println("rise   ", rise.Format(time.DateTime))
	fmt.Println("transit", transit.Format(time.DateTime))
	fmt.Println("set    ", set.Format(time.DateTime))
	// Output:
	// rise    2027-09-01 20:17:59
	// transit 2027-09-02 03:01:30
	// set     2027-09-02 09:44:57
}

// Saturn's culmination, and its altitude there.
func ExampleNextTransit() {
	greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}
	t, ok, err := saturn.NextTransit(greenwich, time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || !ok {
		panic(err)
	}
	r, _ := saturn.Position(greenwich, t)
	fmt.Printf("%s at altitude %.1f\n", t.Format(time.TimeOnly), r.Altitude)
	// Output:
	// 14:15:14 at altitude 41.6
}
