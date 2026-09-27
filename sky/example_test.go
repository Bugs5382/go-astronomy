package sky_test

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

	"github.com/Bugs5382/go-astronomy/planet/mars"
	"github.com/Bugs5382/go-astronomy/sky"
)

// Earth from the Moon: from the middle of the near side it hangs high in the
// sky, goes through phases, and neither rises nor sets.
func Example_earthFromTheMoon() {
	site := sky.Site{Body: sky.Moon, Lat: 0, Lon: 0}
	when := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)

	earth, err := sky.Position(site, sky.Earth, when)
	if err != nil {
		panic(err)
	}
	fmt.Printf("altitude %.1f, azimuth %.1f, %.1f degrees across, %.0f%% lit\n",
		earth.Altitude, earth.Azimuth, float64(earth.Diameter), earth.Illuminated*100)

	rise, _ := sky.NextRise(site, sky.Earth, when)
	fmt.Println("next earthrise:", rise.State)
	// Output:
	// altitude 80.6, azimuth 50.0, 1.9 degrees across, 62% lit
	// next earthrise: always_above
}

// Sunrise and sunset at Jezero crater on Mars, and the local mean solar time
// there, on the Mars clock of the Mars24 algorithm.
func Example_sunriseOnMars() {
	m, err := sky.Planet(mars.Planet)
	if err != nil {
		panic(err)
	}
	jezero := sky.Site{Body: m, Lat: 18.44, Lon: 77.45}
	from := time.Date(2027, 1, 1, 12, 0, 0, 0, time.UTC)

	rise, _ := sky.NextRise(jezero, sky.Sun, from)
	set, _ := sky.NextSet(jezero, sky.Sun, rise.Time)
	fmt.Println("sunrise:", rise.Time.Format(time.RFC3339))
	fmt.Println("sunset: ", set.Time.Format(time.RFC3339))
	fmt.Printf("local mean solar time at sunrise: %.2f Mars hours\n", mars.LocalMeanSolarTime(rise.Time, jezero.Lon))
	fmt.Println("a sol lasts", m.SolarDay().Round(time.Second))
	// Output:
	// sunrise: 2027-01-01T18:38:12Z
	// sunset:  2027-01-02T07:47:56Z
	// local mean solar time at sunrise: 5.75 Mars hours
	// a sol lasts 24h39m35s
}

// The lunar day has no twilight: it divides at sunrise and sunset only, into
// days and nights about two weeks long.
func ExampleSegments() {
	site := sky.Site{Body: sky.Moon}
	from := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	segs, err := sky.Segments(site, from, from.AddDate(0, 0, 45))
	if err != nil {
		panic(err)
	}
	for _, s := range segs {
		fmt.Printf("%-5s %s to %s (%.1f days)\n", s.Label, s.From.Format("Jan 02 15:04"), s.To.Format("Jan 02 15:04"), s.Seconds/86400)
	}
	// Output:
	// night Jan 01 00:00 to Jan 15 04:36 (14.2 days)
	// day   Jan 15 04:36 to Jan 30 01:32 (14.9 days)
	// night Jan 30 01:32 to Feb 13 19:02 (14.7 days)
	// day   Feb 13 19:02 to Feb 15 00:00 (1.2 days)
}
