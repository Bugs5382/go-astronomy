package satellite_test

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
	"errors"
	"fmt"
	"strings"
	"time"

	astronomy "github.com/Bugs5382/go-astronomy"
	"github.com/Bugs5382/go-astronomy/satellite"
)

// An ISS-like element set with an epoch of 2026-09-26. It is synthetic, used
// here and in the tests only; real element sets come from CelesTrak or
// Space-Track, fetched by the caller.
const (
	exampleLine1 = "1 25544U 98067A   26269.51782528  .00016717  00000-0  30306-3 0  9990"
	exampleLine2 = "2 25544  51.6416 247.4627 0006703 130.5360 325.0288 15.49450470 12348"
)

func ExampleParseTLE() {
	e, err := satellite.ParseTLE(exampleLine1, exampleLine2)
	if err != nil {
		panic(err)
	}
	fmt.Println(e.SatNum, e.IntlDesignator, e.Epoch().Format(time.RFC3339))
	fmt.Printf("inclination %.4f, %.8f rev/day\n", e.Inclination, e.MeanMotion)

	// A corrupted line fails its checksum.
	_, err = satellite.ParseTLE(exampleLine1[:68]+"1", exampleLine2)
	fmt.Println(errors.Is(err, satellite.ErrChecksum))
	// Output:
	// 25544 98067A 2026-09-26T12:25:40Z
	// inclination 51.6416, 15.49450470 rev/day
	// true
}

func ExampleParseOMM() {
	omm := `[{"OBJECT_NAME":"VANGUARD 1","OBJECT_ID":"1958-002B","EPOCH":"2025-02-14T14:36:48.662784",
	"MEAN_MOTION":10.85873516,"ECCENTRICITY":0.1841322,"INCLINATION":34.2493,"RA_OF_ASC_NODE":19.2327,
	"ARG_OF_PERICENTER":100.1057,"MEAN_ANOMALY":281.1229,"EPHEMERIS_TYPE":0,"CLASSIFICATION_TYPE":"U",
	"NORAD_CAT_ID":5,"ELEMENT_SET_NO":999,"REV_AT_EPOCH":39027,"BSTAR":0.00035436,
	"MEAN_MOTION_DOT":2.64e-6,"MEAN_MOTION_DDOT":0}]`
	sets, err := satellite.ParseOMM(strings.NewReader(omm))
	if err != nil {
		panic(err)
	}
	fmt.Println(len(sets), sets[0].SatNum, sets[0].IntlDesignator, sets[0].Epoch().Format(time.RFC3339Nano))
	// Output:
	// 1 5 58002B 2025-02-14T14:36:48.662784Z
}

func ExampleElements_Age() {
	e, _ := satellite.ParseTLE(exampleLine1, exampleLine2)
	now := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	age := e.Age(now)
	fmt.Printf("%.1f days old\n", age.Hours()/24)
	if age > 7*24*time.Hour {
		fmt.Println("too old for a low orbit")
	}
	// Output:
	// 3.0 days old
}

func ExampleElements_Propagate() {
	e, _ := satellite.ParseTLE(exampleLine1, exampleLine2)
	pos, vel, err := e.Propagate(e.Epoch().Add(90 * time.Minute))
	if err != nil {
		panic(err)
	}
	fmt.Printf("TEME position %.1f %.1f %.1f km\n", pos[0], pos[1], pos[2])
	fmt.Printf("TEME velocity %.3f %.3f %.3f km/s\n", vel[0], vel[1], vel[2])
	// Output:
	// TEME position 3604.8 -2242.9 5294.2 km
	// TEME velocity 3.391 6.851 0.588 km/s
}

func ExamplePosition() {
	e, _ := satellite.ParseTLE(exampleLine1, exampleLine2)
	sydney := astronomy.Observer{Lat: -33.87, Lng: 151.21}
	when := time.Date(2026, 9, 26, 4, 5, 3, 0, time.UTC) // an almost overhead pass
	l, err := satellite.Position(sydney, e, when)
	if err != nil {
		panic(err)
	}
	fmt.Printf("altitude %.2f, azimuth %.1f, range %.1f km\n", l.Altitude, l.Azimuth, l.RangeKm)
	fmt.Printf("sunlit %v, Sun altitude %.1f\n", l.Sunlit, l.SunAltitude)
	fmt.Printf("over %.2f, %.2f at %.1f km\n", l.Latitude, l.Longitude, l.AltitudeKm)
	// Output:
	// altitude 88.51, azimuth 294.8, range 433.5 km
	// sunlit true, Sun altitude 44.0
	// over -33.83, 151.11 at 433.4 km
}

func ExampleLook_Magnitude() {
	// At the standard geometry, 1000 km and half lit, the magnitude is the
	// standard magnitude; closer and fuller is brighter.
	std := satellite.Look{RangeKm: 1000, PhaseAngle: 90, Sunlit: true}
	close := satellite.Look{RangeKm: 450, PhaseAngle: 60, Sunlit: true}
	fmt.Printf("%.2f %.2f\n", std.Magnitude(satellite.ISSStandardMagnitude), close.Magnitude(satellite.ISSStandardMagnitude))
	// Output:
	// -1.80 -4.24
}

func ExamplePasses() {
	e, _ := satellite.ParseTLE(exampleLine1, exampleLine2)
	denver := astronomy.Observer{Lat: 39.74, Lng: -104.99}
	opt := satellite.DefaultPassOptions()
	opt.StdMagnitude = satellite.ISSStandardMagnitude
	from := time.Date(2026, 9, 26, 0, 0, 0, 0, time.UTC)
	passes, err := satellite.Passes(denver, e, from, from.Add(6*time.Hour), opt)
	if err != nil {
		panic(err)
	}
	for _, p := range passes {
		fmt.Printf("%s-%s peak %4.1f at %s visible %-5v",
			p.Rise.Time.Format("15:04:05"), p.Set.Time.Format("15:04:05"),
			p.Peak.Altitude, p.Peak.Time.Format("15:04:05"), p.Visible)
		if !p.ShadowEntry.IsZero() {
			fmt.Printf(" enters shadow %s", p.ShadowEntry.Format("15:04:05"))
		}
		fmt.Println()
	}
	// Output:
	// 01:18:12-01:28:44 peak 34.7 at 01:23:28 visible true  enters shadow 01:27:04
	// 02:55:02-03:05:31 peak 28.7 at 03:00:16 visible true  enters shadow 03:00:03
	// 04:33:21-04:42:27 peak 11.8 at 04:37:54 visible false
}

func ExampleDefaultPassOptions() {
	opt := satellite.DefaultPassOptions()
	fmt.Println(opt.MinAltitude, opt.DarkSunAltitude, opt.Step)
	// Output:
	// 0 -6 10s
}
