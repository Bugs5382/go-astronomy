package planet_test

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
	"math"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/planet"
)

// Every planet for an observer in London at one instant.
func Example() {
	london := astronomy.Observer{Lat: 51.5074, Lng: -0.1278, TZ: time.UTC}
	when := time.Date(2027, 3, 1, 21, 0, 0, 0, time.UTC)
	for _, b := range planet.Bodies {
		r, err := planet.Position(london, b, when)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%-8s alt %6.1f az %6.1f mag %5.1f elong %5.1f near Sun %v\n",
			b, r.Altitude, r.Azimuth, r.Magnitude, r.Elongation, r.NearSun)
	}
	// Output:
	// mercury  alt  -44.6 az  319.2 mag   1.1 elong  20.1 near Sun false
	// venus    alt  -57.1 az  345.2 mag  -4.1 elong  40.3 near Sun false
	// mars     alt   44.8 az  127.0 mag  -1.1 elong 165.1 near Sun false
	// jupiter  alt   48.0 az  137.4 mag  -2.5 elong 158.6 near Sun false
	// saturn   alt   -4.6 az  280.8 mag   0.8 elong  32.2 near Sun false
	// uranus   alt   37.7 az  254.7 mag   5.7 elong  80.8 near Sun false
	// neptune  alt  -12.7 az  286.6 mag   7.8 elong  22.2 near Sun false
}

// Mars from Greenwich: where it is, how big and bright it looks, how much of
// it is lit, and how long its light took.
func ExamplePosition() {
	greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}
	when := time.Date(2027, 2, 19, 22, 0, 0, 0, time.UTC) // near opposition
	r, err := planet.Position(greenwich, planet.Mars, when)
	if err != nil {
		panic(err)
	}
	fmt.Printf("altitude %.2f, azimuth %.2f degrees\n", r.Altitude, r.Azimuth)
	fmt.Printf("RA %.4f, Dec %.4f degrees (true equator and equinox of date)\n", r.RA, r.Dec)
	fmt.Printf("distance %.4f au, light-time %v\n", r.DistanceAU, r.LightTime.Round(time.Second))
	fmt.Printf("diameter %.2f arcsec, magnitude %.2f\n", float64(r.Diameter)*3600, r.Magnitude)
	fmt.Printf("phase angle %.2f, %.1f%% lit, elongation %.1f\n", r.PhaseAngle, 100*r.Illuminated, r.Elongation)
	// Output:
	// altitude 44.53, azimuth 129.60 degrees
	// RA 154.3613, Dec 15.4028 degrees (true equator and equinox of date)
	// distance 0.6779 au, light-time 5m38s
	// diameter 13.82 arcsec, magnitude -1.28
	// phase angle 2.66, 99.9% lit, elongation 175.5
}

// Heliocentric positions belong to no observer; Earth is included.
func ExampleHeliocentric() {
	when := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, b := range []planet.Body{planet.Earth, planet.Jupiter} {
		h, err := planet.Heliocentric(b, when)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%-7s L %.4f B %.4f R %.6f au\n", b, h.Lon, h.Lat, h.DistanceAU)
	}
	// Output:
	// earth   L 100.3232 B 0.0001 R 0.983343 au
	// jupiter L 138.8042 B 0.8031 R 5.335777 au
}

// The geometric view of Mars from Earth is the difference of two heliocentric
// vectors at the same instant (no light-time or aberration).
func ExampleHeliocentricPosition_Vector() {
	when := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	e, _ := planet.Heliocentric(planet.Earth, when)
	m, _ := planet.Heliocentric(planet.Mars, when)
	ev, mv := e.Vector(), m.Vector()
	d := [3]float64{mv[0] - ev[0], mv[1] - ev[1], mv[2] - ev[2]}
	dist := math.Sqrt(d[0]*d[0] + d[1]*d[1] + d[2]*d[2])
	lon := math.Mod(math.Atan2(d[1], d[0])*180/math.Pi+360, 360)
	fmt.Printf("Mars from Earth: %.4f au, ecliptic longitude %.2f degrees\n", dist, lon)
	// Output:
	// Mars from Earth: 0.9139 au, ecliptic longitude 159.87 degrees
}

// Jupiter's rise, transit, and set at Greenwich on 1 September 2027.
func ExampleNextRise() {
	greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}
	from := time.Date(2027, 9, 1, 0, 0, 0, 0, time.UTC)
	rise, ok, err := planet.NextRise(greenwich, planet.Jupiter, from)
	if err != nil || !ok {
		panic(err)
	}
	transit, _, _ := planet.NextTransit(greenwich, planet.Jupiter, rise)
	set, _, _ := planet.NextSet(greenwich, planet.Jupiter, transit)
	fmt.Println("rise   ", rise.Format(time.TimeOnly))
	fmt.Println("transit", transit.Format(time.TimeOnly))
	fmt.Println("set    ", set.Format(time.TimeOnly))
	// Output:
	// rise    05:06:53
	// transit 11:58:08
	// set     18:49:07
}

// A planet that never sets: Jupiter from high in the Arctic.
func ExampleNextSet() {
	svalbard := astronomy.Observer{Lat: 78.22, Lng: 15.65, TZ: time.UTC}
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	set, ok, err := planet.NextSet(svalbard, planet.Jupiter, from)
	if err != nil {
		panic(err)
	}
	fmt.Println(ok, set.Format(time.DateTime))
	// Output:
	// false 0001-01-01 00:00:00
}

// Saturn's upper culmination, whether or not it is above the horizon.
func ExampleNextTransit() {
	greenwich := astronomy.Observer{Lat: 51.4769, Lng: -0.0005, TZ: time.UTC}
	t, ok, err := planet.NextTransit(greenwich, planet.Saturn, time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || !ok {
		panic(err)
	}
	r, _ := planet.Position(greenwich, planet.Saturn, t)
	fmt.Printf("%s at altitude %.1f\n", t.Format(time.TimeOnly), r.Altitude)
	// Output:
	// 14:15:14 at altitude 41.6
}

// The per-planet elongation limits behind Result.NearSun.
func ExampleNearSunElongation() {
	for _, b := range planet.Bodies {
		fmt.Printf("%s %.1f\n", b, planet.NearSunElongation(b))
	}
	// Output:
	// mercury 10.0
	// venus 5.0
	// mars 11.5
	// jupiter 9.0
	// saturn 11.0
	// uranus 15.0
	// neptune 15.0
}

// Bodies print as lowercase names, which suits labels and map keys.
func ExampleBody_String() {
	fmt.Println(planet.Venus, planet.Earth, planet.Body(0))
	// Output:
	// venus earth unknown
}
