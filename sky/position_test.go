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
	"math"
	"testing"

	"github.com/Bugs5382/go-astronomy/planet"
	"github.com/Bugs5382/go-astronomy/planet/jupiter"
	"github.com/Bugs5382/go-astronomy/planet/mars"
	"github.com/Bugs5382/go-astronomy/planet/venus"
	"github.com/Bugs5382/go-astronomy/sky"
)

// mustPlanet is sky.Planet for a planet the test knows is supported.
func mustPlanet(t *testing.T, p planet.Body) *sky.Body {
	t.Helper()
	b, err := sky.Planet(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// jezero is Jezero crater on Mars, 77.45 E (282.55 W), 18.44 N.
func jezero(t *testing.T) sky.Site {
	return sky.Site{Body: mustPlanet(t, mars.Planet), Lat: 18.44, Lon: 77.45}
}

// TestPositionAgainstHorizons checks the direction, the distance, and the lit
// fraction of a target from a site off Earth against JPL Horizons (DE441,
// airless, apparent): Earth, the Sun, and Mars from the Moon, the Sun, Earth,
// and Jupiter from Jezero on Mars, and the Sun from Venus.
//
// From Mars and Venus the direction agrees to 0.1 to 0.4 arc second. From the
// Moon it agrees to about 11: Horizons orients a lunar site in the mean Earth
// frame of the DE441 libration, and the IAU rotation model used here
// approximates that frame to about 10 arc seconds. The Sun and Mars from the
// Moon show the same 9 arc seconds, so it is the frame, not the targets.
func TestPositionAgainstHorizons(t *testing.T) {
	t.Parallel()
	moon0 := sky.Site{Body: sky.Moon, Lat: 0, Lon: 0}
	venus0 := sky.Site{Body: mustPlanet(t, venus.Planet)}
	cases := []struct {
		file    string
		site    sky.Site
		target  *sky.Body
		arcsec  float64 // direction
		illum   float64 // lit fraction
		distRel float64 // relative distance
	}{
		{"earth-from-moon-0-0.txt", moon0, sky.Earth, 15, 0.0005, 5e-5},
		{"earth-from-moon-180-0.txt", sky.Site{Body: sky.Moon, Lon: 180}, sky.Earth, 15, 0.0005, 5e-5},
		{"sun-from-moon-0-0.txt", moon0, sky.Sun, 15, 0, 1e-6},
		{"mars-from-moon-0-0.txt", moon0, mustPlanet(t, mars.Planet), 15, 0.0005, 1e-6},
		{"sun-from-jezero.txt", jezero(t), sky.Sun, 1, 0, 1e-6},
		{"earth-from-jezero.txt", jezero(t), sky.Earth, 1, 0.0005, 1e-6},
		{"jupiter-from-jezero.txt", jezero(t), mustPlanet(t, jupiter.Planet), 1, 0.0005, 1e-6},
		{"sun-from-venus-0-0.txt", venus0, sky.Sun, 1, 0, 1e-6},
	}
	for _, c := range cases {
		var worst, worstIllum, worstDist float64
		for _, r := range load(t, c.file) {
			got, err := sky.Position(c.site, c.target, r.when)
			if err != nil {
				t.Fatal(err)
			}
			sep := separation(got.Altitude, got.Azimuth, r.num(t, 1), r.num(t, 0))
			worst = math.Max(worst, sep)
			worstIllum = math.Max(worstIllum, math.Abs(got.Illuminated-r.num(t, 2)/100))
			worstDist = math.Max(worstDist, math.Abs(got.DistanceAU/r.num(t, 3)-1))
		}
		if worst > c.arcsec || worstIllum > c.illum || worstDist > c.distRel {
			t.Errorf("%s: direction off by up to %.2f arcsec (want %.0f), lit fraction %.5f (want %.5f), distance %.2e (want %.0e)",
				c.file, worst, c.arcsec, worstIllum, c.illum, worstDist, c.distRel)
		}
	}
}
