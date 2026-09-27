package all_test

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
	"github.com/Bugs5382/go-astronomy/planet/all"
)

// Every planet for an observer in London at one instant.
func ExamplePlanets() {
	london := astronomy.Observer{Lat: 51.5074, Lng: -0.1278, TZ: time.UTC}
	when := time.Date(2027, 3, 1, 21, 0, 0, 0, time.UTC)
	for _, p := range all.Planets() {
		r, err := p.Position(london, when)
		if err != nil {
			panic(err)
		}
		fmt.Printf("%-8s alt %6.1f az %6.1f mag %5.1f elong %5.1f near Sun %v\n",
			p.Name(), r.Altitude, r.Azimuth, r.Magnitude, r.Elongation, r.NearSun)
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

// Picking a planet by name, for example from a request parameter.
func ExampleByName() {
	p, ok := all.ByName("jupiter")
	fmt.Println(ok, p.Name(), p.RadiusKm(), p.NearSunElongation())
	_, ok = all.ByName("pluto")
	fmt.Println(ok)
	// Output:
	// true jupiter 71492 9
	// false
}
