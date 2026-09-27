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
	"testing"
	"time"

	"github.com/Bugs5382/go-astronomy/sky"
)

// TestRiseSetAgainstHorizons checks rise, transit, and set off Earth against
// the JPL Horizons true visual horizon (upper limb, airless). Horizons stamps
// each event with the one-minute step at or after it, so the library's
// instant should fall up to a minute before the stamp, widened by the time
// the target takes to move through the position error: from Mars, where the
// direction agrees to 0.1 arc second, a few seconds; from the Moon, whose
// IAU rotation model sits about 10 arc seconds from the frame Horizons uses,
// up to 30 s for the Sun, which climbs half a degree an hour, and a few
// minutes for Earth, which the libration moves by only a few degrees a day.
func TestRiseSetAgainstHorizons(t *testing.T) {
	t.Parallel()
	cases := []struct {
		file   string
		site   sky.Site
		target *sky.Body
		slack  time.Duration
	}{
		{"sun-rts-jezero.txt", jezero(t), sky.Sun, 5 * time.Second},
		{"sun-rts-moon-0-0.txt", sky.Site{Body: sky.Moon}, sky.Sun, 30 * time.Second},
		{"earth-rts-moon-90-0.txt", sky.Site{Body: sky.Moon, Lon: 90}, sky.Earth, 5 * time.Minute},
	}
	for _, c := range cases {
		for _, r := range load(t, c.file) {
			from := r.when.Add(-sky.Window(c.site.Body) / 8)
			var ev sky.Event
			var err error
			switch r.fields[0] {
			case "r":
				ev, err = sky.NextRise(c.site, c.target, from)
			case "s":
				ev, err = sky.NextSet(c.site, c.target, from)
			case "t":
				ev, err = sky.NextTransit(c.site, c.target, from)
			default:
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			d := ev.Time.Sub(r.when)
			if !ev.Found() || d < -time.Minute-c.slack || d > c.slack {
				t.Errorf("%s %s at %s: %s %s, off %v (want -1m-%v to %v)", c.file, r.fields[0], r.when.Format(time.RFC3339),
					ev.State, ev.Time.Format(time.RFC3339), d.Round(time.Second), c.slack, c.slack)
			}
		}
	}
}
